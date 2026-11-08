package workerapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"

	"astroswarm/worker/internal/server/entries"
	"astroswarm/worker/internal/server/httpx"
	"astroswarm/worker/internal/server/pytime"
	"astroswarm/worker/internal/server/store"
	"astroswarm/worker/internal/server/values"
)

func requiredWorkerID(data map[string]any) (string, error) {
	workerID := strings.TrimSpace(textField(data, "worker_id", "", 1000))
	if workerID == "" {
		return "", httpx.BadRequest("worker_id is required")
	}
	return workerID, nil
}

func (api *API) register(w http.ResponseWriter, r *http.Request) error {
	if err := api.deps.Auth.RequireAPIKey(r); err != nil {
		return err
	}
	data := httpx.ReadJSONObjectOrEmpty(r)
	workerID, err := requiredWorkerID(data)
	if err != nil {
		return err
	}
	ctx := r.Context()
	worker, err := loadWorker(ctx, api.deps.Store.Pool, workerID)
	if err != nil {
		return err
	}
	now := pytime.Now()
	if worker == nil {
		_, err = api.deps.Store.Pool.Exec(ctx, `INSERT INTO workers (id, name, hostname, enabled, reported_status, current_job_id, last_seen, created_at)
			VALUES ($1, $2, $3, true, 'idle', NULL, $4, $4)`,
			workerID, textField(data, "name", "worker", 80), textField(data, "hostname", "", 120), now)
		if err != nil {
			return err
		}
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"enabled": true})
		return nil
	}
	hostname := ""
	if worker.Hostname != nil {
		hostname = *worker.Hostname
	}
	_, err = api.deps.Store.Pool.Exec(ctx, `UPDATE workers SET hostname = $2, last_seen = $3, reported_status = 'idle',
		current_job_id = NULL WHERE id = $1`, workerID, textField(data, "hostname", hostname, 120), now)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"enabled": worker.Enabled})
	return nil
}

func (api *API) heartbeat(w http.ResponseWriter, r *http.Request) error {
	if err := api.deps.Auth.RequireAPIKey(r); err != nil {
		return err
	}
	data := httpx.ReadJSONObjectOrEmpty(r)
	workerID, err := requiredWorkerID(data)
	if err != nil {
		return err
	}
	ctx := r.Context()
	worker, err := loadWorker(ctx, api.deps.Store.Pool, workerID)
	if err != nil {
		return err
	}
	if worker == nil {
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"enabled": false, "known": false})
		return nil
	}
	var currentJob any
	if currentJobs, isList := data["current_jobs"].([]any); isList && len(currentJobs) > 0 {
		currentJob = values.Truncate(values.ToText(currentJobs[0]), 64)
	} else if values.Truthy(data["current_job"]) {
		currentJob = values.ToText(data["current_job"])
	}
	var stats any
	if statsObject, isObject := data["system_stats"].(map[string]any); isObject {
		if stats, err = store.JSONParameter(statsObject); err != nil {
			return err
		}
	}
	_, err = api.deps.Store.Pool.Exec(ctx, `UPDATE workers SET last_seen = $2, reported_status = $3, current_job_id = $4,
		system_stats = COALESCE($5::json, system_stats) WHERE id = $1`,
		workerID, pytime.Now(), textField(data, "status", "idle", 20), currentJob, stats)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"enabled": worker.Enabled, "known": true})
	return nil
}

func requestedSlots(data map[string]any) int {
	slots, err := values.ToInt(valueOrDefault(data, "slots", int64(1)))
	if err != nil {
		slots = 1
	}
	return int(max(1, min(slots, maxClaimSlots)))
}

func valueOrDefault(data map[string]any, key string, fallback any) any {
	if value, found := data[key]; found {
		return value
	}
	return fallback
}

func (api *API) claim(w http.ResponseWriter, r *http.Request) error {
	if err := api.deps.Auth.RequireAPIKey(r); err != nil {
		return err
	}
	data := httpx.ReadJSONObjectOrEmpty(r)
	workerID, err := requiredWorkerID(data)
	if err != nil {
		return err
	}
	ctx := r.Context()
	worker, err := loadWorker(ctx, api.deps.Store.Pool, workerID)
	if err != nil {
		return err
	}
	if worker == nil {
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"jobs": []any{}, "enabled": false, "known": false})
		return nil
	}
	if _, err := api.deps.Store.Pool.Exec(ctx, "UPDATE workers SET last_seen = $2 WHERE id = $1", workerID, pytime.Now()); err != nil {
		return err
	}
	if !worker.enabled() {
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"jobs": []any{}, "enabled": false})
		return nil
	}
	if err := api.reaper.reapIfDue(ctx); err != nil {
		return err
	}
	jobs := []any{}
	err = api.deps.Store.InTransaction(ctx, func(transaction pgx.Tx) error {
		claimed, err := entries.List(ctx, transaction,
			entries.SelectEntries("player_evaluations.status = 'queued'")+
				" ORDER BY player_evaluations.created_at ASC LIMIT $1 FOR UPDATE OF player_evaluations SKIP LOCKED",
			requestedSlots(data))
		if err != nil {
			return err
		}
		for _, entry := range claimed {
			job, err := api.startJob(ctx, transaction, entry, workerID)
			if err != nil {
				return err
			}
			jobs = append(jobs, job)
		}
		return api.updateClaimingWorker(ctx, transaction, workerID, worker, jobs)
	})
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"jobs": jobs, "enabled": true})
	return nil
}

func (api *API) startJob(ctx context.Context, transaction pgx.Tx, entry *entries.Entry, workerID string) (map[string]any, error) {
	totalUnits := 0
	if entry.TotalUnits != nil {
		totalUnits = *entry.TotalUnits
	}
	if totalUnits == 0 {
		computed, err := entries.TotalUnits(ctx, transaction, api.deps.Settings, entry)
		if err != nil {
			return nil, err
		}
		totalUnits = computed
	}
	collisions := entry.Collisions != nil && *entry.Collisions
	jobConfig, err := api.jobConfig(ctx, entry.Level(), collisions)
	if err != nil {
		return nil, err
	}
	jobConfigJSON, err := store.JSONParameter(jobConfig)
	if err != nil {
		return nil, err
	}
	_, err = transaction.Exec(ctx, `UPDATE player_evaluations SET status = 'running', progress = 0, worker_id = $2,
		done_units = 0, last_update = $3, total_units = $4, job_config = $5::jsonb WHERE id = $1`,
		entry.ID, workerID, pytime.Now(), totalUnits, jobConfigJSON)
	if err != nil {
		return nil, err
	}
	pendingRun, err := entries.PendingRun(ctx, transaction, entry.ID)
	if err != nil {
		return nil, err
	}
	var run any
	if pendingRun != nil {
		run = json.RawMessage(pendingRun)
	}
	return map[string]any{
		"job_id":          entry.ID,
		"evaluation_id":   entry.ID,
		"algorithm":       entry.AlgorithmValue(),
		"genome_schedule": entry.GenomeScheduleValue(),
		"placements":      entry.PlacementsValue(),
		"run":             run,
		"trials":          entry.Trials,
		"total_units":     max(totalUnits, 1),
		"config":          jobConfig,
	}, nil
}

func (api *API) updateClaimingWorker(ctx context.Context, transaction pgx.Tx, workerID string, worker *Worker, jobs []any) error {
	running, err := runningCount(ctx, transaction, workerID)
	if err != nil {
		return err
	}
	if running == 0 {
		_, err = transaction.Exec(ctx, "UPDATE workers SET reported_status = 'idle', current_job_id = NULL WHERE id = $1", workerID)
		return err
	}
	currentJob := worker.CurrentJobID
	if len(jobs) > 0 && (currentJob == nil || *currentJob == "") {
		firstJobID := jobs[0].(map[string]any)["evaluation_id"].(string)
		currentJob = &firstJobID
	}
	_, err = transaction.Exec(ctx, "UPDATE workers SET reported_status = 'busy', current_job_id = $2 WHERE id = $1", workerID, currentJob)
	return err
}
