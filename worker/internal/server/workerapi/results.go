package workerapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"

	"astroswarm/worker/internal/server/benchmarks"
	"astroswarm/worker/internal/server/entries"
	"astroswarm/worker/internal/server/httpx"
	"astroswarm/worker/internal/server/pytime"
	"astroswarm/worker/internal/server/values"
)

func (api *API) progress(w http.ResponseWriter, r *http.Request) error {
	if err := api.deps.Auth.RequireAPIKey(r); err != nil {
		return err
	}
	data := httpx.ReadJSONObjectOrEmpty(r)
	workerID := strings.TrimSpace(textField(data, "worker_id", "", 1000))
	ctx := r.Context()
	entry, err := entries.Get(ctx, api.deps.Store.Pool, r.PathValue("job_id"))
	if err != nil {
		return err
	}
	if entry == nil {
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"cancel": true})
		return nil
	}
	if workerID != "" {
		if _, err := api.deps.Store.Pool.Exec(ctx, "UPDATE workers SET last_seen = $2 WHERE id = $1", workerID, pytime.Now()); err != nil {
			return err
		}
	}
	owns := entry.StatusText() == "running" && entry.WorkerID != nil && *entry.WorkerID == workerID
	if owns {
		total := 1
		if entry.TotalUnits != nil && *entry.TotalUnits > 1 {
			total = *entry.TotalUnits
		}
		doneUnits := 0
		if entry.DoneUnits != nil {
			doneUnits = *entry.DoneUnits
		}
		if reported, err := values.ToInt(valueOrDefault(data, "done", int64(0))); err == nil {
			doneUnits = int(max(0, min(reported, int64(total))))
		}
		progress := values.RoundTo(min(0.99, float64(doneUnits)/float64(total)), 3)
		var stage any
		if values.Truthy(data["stage"]) {
			stage = values.Truncate(values.ToText(data["stage"]), 200)
		}
		_, err = api.deps.Store.Pool.Exec(ctx, `UPDATE player_evaluations SET done_units = $2, progress = $3, last_update = $4,
			stage = COALESCE($5, stage) WHERE id = $1`, entry.ID, doneUnits, progress, pytime.Now(), stage)
		if err != nil {
			return err
		}
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"cancel": !owns})
	return nil
}

func (api *API) replayChunk(w http.ResponseWriter, r *http.Request) error {
	if err := api.deps.Auth.RequireAPIKey(r); err != nil {
		return err
	}
	workerID := strings.TrimSpace(r.Header.Get("X-Worker-ID"))
	offsetText := r.URL.Query().Get("offset")
	if offsetText == "" {
		offsetText = "0"
	}
	offset, err := strconv.ParseInt(offsetText, 10, 64)
	if err != nil {
		return httpx.BadRequest("offset must be an integer")
	}
	if offset < 0 {
		return httpx.BadRequest("offset must not be negative")
	}
	ctx := r.Context()
	jobID := r.PathValue("job_id")
	entry, err := entries.Get(ctx, api.deps.Store.Pool, jobID)
	if err != nil {
		return err
	}
	if entry == nil {
		return httpx.NotFound("Job not found")
	}
	if entry.StatusText() != "running" || entry.WorkerID == nil || *entry.WorkerID != workerID {
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"accepted": false, "size": 0, "path": ""})
		return nil
	}
	body, err := httpx.ReadBody(r, benchmarks.MaxChunkBytes)
	var httpError *httpx.Error
	if errors.As(err, &httpError) && httpError.Status == http.StatusRequestEntityTooLarge {
		return httpx.BadRequest(benchmarks.ErrChunkTooLarge.Error())
	}
	if err != nil {
		return err
	}
	path, size, err := api.deps.Files.WriteChunk(jobID, r.PathValue("upload_id"), r.PathValue("key"), offset, body)
	if err != nil {
		if errors.Is(err, benchmarks.ErrInvalidName) || errors.Is(err, benchmarks.ErrChunkTooLarge) || errors.Is(err, benchmarks.ErrReplayTooLarge) {
			return httpx.BadRequest(err.Error())
		}
		return err
	}
	if _, err := api.deps.Store.Pool.Exec(ctx, "UPDATE player_evaluations SET last_update = $2 WHERE id = $1", jobID, pytime.Now()); err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"accepted": true, "size": size, "path": path})
	return nil
}

func (api *API) missingReplayFile(result map[string]any) string {
	check := func(run map[string]any) string {
		if framesFile, _ := run["frames_file"].(string); framesFile != "" && !api.deps.Files.Exists(framesFile) {
			return framesFile
		}
		return ""
	}
	for _, run := range objectList(result["runs"]) {
		if missing := check(run); missing != "" {
			return missing
		}
	}
	for _, sweepRun := range objectList(result["sweep_runs"]) {
		if missing := check(sweepRun); missing != "" {
			return missing
		}
		for _, child := range objectList(sweepRun["trial_runs"]) {
			if missing := check(child); missing != "" {
				return missing
			}
		}
	}
	return ""
}

func objectList(value any) []map[string]any {
	list, _ := value.([]any)
	objects := []map[string]any{}
	for _, item := range list {
		if object, isObject := item.(map[string]any); isObject {
			objects = append(objects, object)
		}
	}
	return objects
}

func (api *API) result(w http.ResponseWriter, r *http.Request) error {
	if err := api.deps.Auth.RequireAPIKey(r); err != nil {
		return err
	}
	body, err := httpx.ReadBody(r, api.deps.Config.MaxWorkerResultBytes)
	if err != nil {
		return err
	}
	data := map[string]any{}
	if len(body) > 0 {
		if !json.Valid(body) {
			return httpx.BadRequest("Invalid JSON body")
		}
		if decoded, isObject := httpx.DecodeObject(body); isObject {
			data = decoded
		}
	}
	workerID := strings.TrimSpace(textField(data, "worker_id", "", 1000))
	ctx := r.Context()
	if workerID != "" {
		if _, err := api.deps.Store.Pool.Exec(ctx, "UPDATE workers SET last_seen = $2 WHERE id = $1", workerID, pytime.Now()); err != nil {
			return err
		}
	}
	result, _ := data["result"].(map[string]any)
	if result == nil {
		result = map[string]any{}
	}
	jobID := r.PathValue("job_id")
	accepted := false
	err = api.deps.Store.InTransaction(ctx, func(transaction pgx.Tx) error {
		entry, err := entries.GetForUpdate(ctx, transaction, jobID)
		if err != nil {
			return err
		}
		if entry == nil {
			return httpx.NotFound("Job not found")
		}
		accepted = entry.StatusText() == "running" && entry.WorkerID != nil && *entry.WorkerID == workerID
		if !accepted {
			return nil
		}
		if missing := api.missingReplayFile(result); missing != "" {
			return httpx.BadRequest("Replay file " + missing + " was never uploaded")
		}
		return api.storeResult(ctx, transaction, entry, result)
	})
	if err != nil {
		return err
	}
	if accepted {
		uploadID := ""
		if values.Truthy(data["upload_id"]) {
			uploadID = values.ToText(data["upload_id"])
		}
		if err := benchmarks.RemoveUnusedUploads(ctx, api.deps.Store.Pool, api.deps.Files, jobID, uploadID); err != nil {
			api.deps.Logger.Printf("[WARNING] Could not clean up uploads for %s: %v", jobID, err)
		}
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"ok": true, "accepted": accepted})
	return nil
}

func (api *API) storeResult(ctx context.Context, transaction pgx.Tx, entry *entries.Entry, payload map[string]any) error {
	results, replays := benchmarks.Summarize(payload)
	benchmarks.AnnotateRates(results, entry.IsAttackLevel())
	contentHash, err := entry.FinishedJobHash(ctx, api.deps.Settings, api.deps.Config)
	if err != nil {
		return err
	}
	benchmarkID := ""
	found := false
	if contentHash != nil {
		if benchmarkID, found, err = benchmarks.FindByHash(ctx, transaction, *contentHash); err != nil {
			return err
		}
	}
	if !found {
		if benchmarkID, err = api.deps.Builder.Build(ctx, transaction, entry.ID, entry.Level(), results, replays, contentHash); err != nil {
			return err
		}
	}
	now := pytime.Now()
	totalUnits := 1
	if entry.TotalUnits != nil && *entry.TotalUnits != 0 {
		totalUnits = *entry.TotalUnits
	}
	_, err = transaction.Exec(ctx, `UPDATE player_evaluations SET benchmark_id = $2, status = 'done', progress = 1.0,
		done_units = $3, error = NULL, completed_at = $4, worker_id = NULL, stage = NULL, last_update = $4 WHERE id = $1`,
		entry.ID, benchmarkID, totalUnits, now)
	if err != nil {
		return err
	}
	if entry.BenchmarkID != nil && *entry.BenchmarkID != benchmarkID {
		return benchmarks.Release(ctx, transaction, entry.BenchmarkID)
	}
	return nil
}

func (api *API) fail(w http.ResponseWriter, r *http.Request) error {
	if err := api.deps.Auth.RequireAPIKey(r); err != nil {
		return err
	}
	data := httpx.ReadJSONObjectOrEmpty(r)
	workerID := strings.TrimSpace(textField(data, "worker_id", "", 1000))
	ctx := r.Context()
	if workerID != "" {
		if _, err := api.deps.Store.Pool.Exec(ctx, "UPDATE workers SET last_seen = $2 WHERE id = $1", workerID, pytime.Now()); err != nil {
			return err
		}
	}
	err := api.deps.Store.InTransaction(ctx, func(transaction pgx.Tx) error {
		entry, err := entries.GetForUpdate(ctx, transaction, r.PathValue("job_id"))
		if err != nil {
			return err
		}
		if entry == nil {
			return httpx.NotFound("Job not found")
		}
		if entry.StatusText() != "running" || entry.WorkerID == nil || *entry.WorkerID != workerID {
			return nil
		}
		now := pytime.Now()
		_, err = transaction.Exec(ctx, `UPDATE player_evaluations SET status = 'failed', error = $2, completed_at = $3,
			worker_id = NULL, stage = NULL, last_update = $3 WHERE id = $1`,
			entry.ID, values.Truncate(values.ToText(valueOrDefault(data, "error", "worker error")), 400), now)
		return err
	})
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"ok": true})
	return nil
}
