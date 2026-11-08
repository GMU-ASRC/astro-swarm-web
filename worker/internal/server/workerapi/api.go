package workerapi

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

	"astroswarm/worker/internal/server/benchmarks"
	"astroswarm/worker/internal/server/deps"
	"astroswarm/worker/internal/server/httpx"
	"astroswarm/worker/internal/server/pytime"
	"astroswarm/worker/internal/server/values"
)

const reapEvery = 30 * time.Second

type API struct {
	deps   *deps.Deps
	reaper *Reaper
}

func New(dependencies *deps.Deps) *API {
	return &API{deps: dependencies, reaper: &Reaper{deps: dependencies}}
}

func (api *API) Reaper() *Reaper {
	return api.reaper
}

func (api *API) Register(router *httpx.Router) {
	router.Handle("POST /api/worker/register", api.register)
	router.Handle("POST /api/worker/heartbeat", api.heartbeat)
	router.Handle("POST /api/worker/claim", api.claim)
	router.Handle("POST /api/worker/jobs/{job_id}/progress", api.progress)
	router.Handle("POST /api/worker/jobs/{job_id}/replays/{upload_id}/{key}", api.replayChunk)
	router.Handle("POST /api/worker/jobs/{job_id}/result", api.result)
	router.Handle("POST /api/worker/jobs/{job_id}/fail", api.fail)
	router.Handle("GET /api/workers", api.listWorkers)
	router.Handle("GET /api/workers/{worker_id}", api.getWorker)
	router.Handle("POST /api/workers/{worker_id}/settings", api.updateSettings)
	router.Handle("POST /api/workers/{worker_id}/connect", api.connect)
	router.Handle("POST /api/workers/{worker_id}/disconnect", api.disconnect)
	router.Handle("DELETE /api/workers/{worker_id}", api.deleteWorker)
}

func (api *API) jobConfig(ctx context.Context, levelID string, collisions bool) (map[string]any, error) {
	return benchmarks.JobConfig(ctx, api.deps.Settings, api.deps.Config, levelID, collisions)
}

type Reaper struct {
	deps     *deps.Deps
	mutex    sync.Mutex
	lastReap time.Time
}

func (reaper *Reaper) reapIfDue(ctx context.Context) error {
	reaper.mutex.Lock()
	if !reaper.lastReap.IsZero() && time.Since(reaper.lastReap) < reapEvery {
		reaper.mutex.Unlock()
		return nil
	}
	reaper.lastReap = time.Now()
	reaper.mutex.Unlock()
	return reaper.Reap(ctx)
}

func (reaper *Reaper) Reap(ctx context.Context) error {
	now := pytime.Now()
	pool := reaper.deps.Store.Pool
	_, err := pool.Exec(ctx, `UPDATE player_evaluations SET status = 'queued', worker_id = NULL, done_units = 0, progress = 0,
		last_update = $1 WHERE status = 'running' AND last_update < $2`, now, now.Add(-jobSilentSeconds*time.Second))
	if err != nil {
		return err
	}
	_, err = pool.Exec(ctx, `UPDATE workers SET current_job_id = NULL, reported_status = 'offline'
		WHERE (last_seen IS NULL OR last_seen < $1) AND reported_status IS DISTINCT FROM 'offline'`,
		now.Add(-workerOnlineSeconds*time.Second))
	return err
}

func (reaper *Reaper) Run(ctx context.Context) {
	ticker := time.NewTicker(reapEvery)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := reaper.reapIfDue(ctx); err != nil {
				reaper.deps.Logger.Printf("[WARNING] Could not requeue stale jobs: %v", err)
			}
		}
	}
}

func (api *API) listWorkers(w http.ResponseWriter, r *http.Request) error {
	if _, err := api.deps.Auth.RequireAdmin(r); err != nil {
		return err
	}
	ctx := r.Context()
	if err := api.reaper.reapIfDue(ctx); err != nil {
		return err
	}
	workers, err := listWorkers(ctx, api.deps.Store.Pool)
	if err != nil {
		return err
	}
	counts, err := runningCounts(ctx, api.deps.Store.Pool)
	if err != nil {
		return err
	}
	summaries := make([]any, 0, len(workers))
	for _, worker := range workers {
		summaries = append(summaries, worker.summary(counts[worker.ID]))
	}
	httpx.WriteJSON(w, http.StatusOK, summaries)
	return nil
}

func (api *API) getWorker(w http.ResponseWriter, r *http.Request) error {
	if _, err := api.deps.Auth.RequireAdmin(r); err != nil {
		return err
	}
	ctx := r.Context()
	if err := api.reaper.reapIfDue(ctx); err != nil {
		return err
	}
	worker, err := api.requireWorker(r)
	if err != nil {
		return err
	}
	jobs, err := workerJobs(ctx, api.deps.Store.Pool, worker.ID)
	if err != nil {
		return err
	}
	payload := worker.summary(len(jobs))
	payload["jobs"] = jobs
	httpx.WriteJSON(w, http.StatusOK, payload)
	return nil
}

func (api *API) requireWorker(r *http.Request) (*Worker, error) {
	worker, err := loadWorker(r.Context(), api.deps.Store.Pool, r.PathValue("worker_id"))
	if err != nil {
		return nil, err
	}
	if worker == nil {
		return nil, httpx.NotFound("Worker not found")
	}
	return worker, nil
}

func (api *API) respondWithWorker(w http.ResponseWriter, r *http.Request, workerID string) error {
	worker, err := loadWorker(r.Context(), api.deps.Store.Pool, workerID)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, worker.ToMap())
	return nil
}

func (api *API) updateSettings(w http.ResponseWriter, r *http.Request) error {
	if _, err := api.deps.Auth.RequireAdmin(r); err != nil {
		return err
	}
	worker, err := api.requireWorker(r)
	if err != nil {
		return err
	}
	data := httpx.ReadJSONObjectOrEmpty(r)
	if nameValue, present := data["name"]; present {
		name := values.Truncate(strings.TrimSpace(values.ToText(nameValue)), 80)
		if name != "" {
			if _, err := api.deps.Store.Pool.Exec(r.Context(), "UPDATE workers SET name = $2 WHERE id = $1", worker.ID, name); err != nil {
				return err
			}
		}
	}
	return api.respondWithWorker(w, r, worker.ID)
}

func (api *API) connect(w http.ResponseWriter, r *http.Request) error {
	if _, err := api.deps.Auth.RequireAdmin(r); err != nil {
		return err
	}
	worker, err := api.requireWorker(r)
	if err != nil {
		return err
	}
	if _, err := api.deps.Store.Pool.Exec(r.Context(), "UPDATE workers SET enabled = true WHERE id = $1", worker.ID); err != nil {
		return err
	}
	return api.respondWithWorker(w, r, worker.ID)
}

func (api *API) disconnect(w http.ResponseWriter, r *http.Request) error {
	if _, err := api.deps.Auth.RequireAdmin(r); err != nil {
		return err
	}
	worker, err := api.requireWorker(r)
	if err != nil {
		return err
	}
	ctx := r.Context()
	if err := requeueWorkerJobs(ctx, api.deps.Store.Pool, worker.ID); err != nil {
		return err
	}
	_, err = api.deps.Store.Pool.Exec(ctx, `UPDATE workers SET enabled = false, current_job_id = NULL,
		reported_status = 'disconnected' WHERE id = $1`, worker.ID)
	if err != nil {
		return err
	}
	return api.respondWithWorker(w, r, worker.ID)
}

func (api *API) deleteWorker(w http.ResponseWriter, r *http.Request) error {
	if _, err := api.deps.Auth.RequireAdmin(r); err != nil {
		return err
	}
	worker, err := api.requireWorker(r)
	if err != nil {
		return err
	}
	ctx := r.Context()
	if err := requeueWorkerJobs(ctx, api.deps.Store.Pool, worker.ID); err != nil {
		return err
	}
	if _, err := api.deps.Store.Pool.Exec(ctx, "DELETE FROM workers WHERE id = $1", worker.ID); err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"ok": true})
	return nil
}
