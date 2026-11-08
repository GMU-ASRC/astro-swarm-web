package workerapi

import (
	"bytes"
	"context"
	"encoding/json"
	"time"

	"astroswarm/worker/internal/server/pytime"
	"astroswarm/worker/internal/server/store"
	"astroswarm/worker/internal/server/values"
)

const (
	workerOnlineSeconds = 30
	jobSilentSeconds    = 300
	maxClaimSlots       = 64
)

type Worker struct {
	ID             string
	Name           *string
	Hostname       *string
	Enabled        *bool
	ReportedStatus *string
	CurrentJobID   *string
	SystemStats    []byte
	LastSeen       *time.Time
	CreatedAt      *time.Time
}

const workerColumns = "id, name, hostname, enabled, reported_status, current_job_id, system_stats, last_seen, created_at"

func scanWorker(row interface{ Scan(...any) error }) (*Worker, error) {
	worker := &Worker{}
	err := row.Scan(&worker.ID, &worker.Name, &worker.Hostname, &worker.Enabled, &worker.ReportedStatus,
		&worker.CurrentJobID, &worker.SystemStats, &worker.LastSeen, &worker.CreatedAt)
	return worker, err
}

func loadWorker(ctx context.Context, db store.Querier, workerID string) (*Worker, error) {
	worker, err := scanWorker(db.QueryRow(ctx, "SELECT "+workerColumns+" FROM workers WHERE id = $1", workerID))
	if store.IsNoRows(err) {
		return nil, nil
	}
	return worker, err
}

func listWorkers(ctx context.Context, db store.Querier) ([]*Worker, error) {
	rows, err := db.Query(ctx, "SELECT "+workerColumns+" FROM workers ORDER BY created_at ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	workers := []*Worker{}
	for rows.Next() {
		worker, err := scanWorker(rows)
		if err != nil {
			return nil, err
		}
		workers = append(workers, worker)
	}
	return workers, rows.Err()
}

func (worker *Worker) enabled() bool {
	return worker.Enabled != nil && *worker.Enabled
}

func (worker *Worker) isOnline() bool {
	if worker.LastSeen == nil {
		return false
	}
	return time.Since(*worker.LastSeen).Seconds() <= workerOnlineSeconds
}

func (worker *Worker) status() string {
	if !worker.isOnline() {
		return "offline"
	}
	if !worker.enabled() {
		return "disconnected"
	}
	if worker.CurrentJobID != nil && *worker.CurrentJobID != "" {
		return "busy"
	}
	return "idle"
}

func (worker *Worker) stats() map[string]any {
	if len(worker.SystemStats) == 0 {
		return map[string]any{}
	}
	var stats map[string]any
	decoder := json.NewDecoder(bytesReader(worker.SystemStats))
	decoder.UseNumber()
	if decoder.Decode(&stats) != nil || stats == nil {
		return map[string]any{}
	}
	return stats
}

func (worker *Worker) ToMap() map[string]any {
	return map[string]any{
		"id":             worker.ID,
		"name":           worker.Name,
		"hostname":       worker.Hostname,
		"enabled":        worker.Enabled,
		"status":         worker.status(),
		"online":         worker.isOnline(),
		"current_job_id": worker.CurrentJobID,
		"system_stats":   worker.stats(),
		"last_seen":      pytime.IsoformatOrNil(worker.LastSeen),
		"created_at":     pytime.IsoformatOrNil(worker.CreatedAt),
	}
}

func (worker *Worker) summary(runningJobs int) map[string]any {
	payload := worker.ToMap()
	payload["running_jobs"] = runningJobs
	jobSlots := int64(1)
	if value := worker.stats()["job_slots"]; values.Truthy(value) {
		if converted, err := values.ToInt(value); err == nil {
			jobSlots = converted
		}
	}
	payload["job_slots"] = jobSlots
	return payload
}

func runningCounts(ctx context.Context, db store.Querier) (map[string]int, error) {
	rows, err := db.Query(ctx, `SELECT worker_id, count(id) FROM player_evaluations
		WHERE status = 'running' AND worker_id IS NOT NULL GROUP BY worker_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	counts := map[string]int{}
	for rows.Next() {
		var workerID string
		var count int
		if err := rows.Scan(&workerID, &count); err != nil {
			return nil, err
		}
		counts[workerID] = count
	}
	return counts, rows.Err()
}

func runningCount(ctx context.Context, db store.Querier, workerID string) (int, error) {
	var count int
	err := db.QueryRow(ctx, "SELECT count(*) FROM player_evaluations WHERE worker_id = $1 AND status = 'running'", workerID).Scan(&count)
	return count, err
}

func requeueWorkerJobs(ctx context.Context, db store.Querier, workerID string) error {
	_, err := db.Exec(ctx, `UPDATE player_evaluations SET status = 'queued', worker_id = NULL, done_units = 0, progress = 0,
		last_update = $2 WHERE worker_id = $1 AND status = 'running'`, workerID, pytime.Now())
	return err
}

func textField(data map[string]any, key string, fallback string, limit int) string {
	value, found := data[key]
	if !found {
		return values.Truncate(fallback, limit)
	}
	return values.Truncate(values.ToText(value), limit)
}

func bytesReader(data []byte) *bytes.Reader {
	return bytes.NewReader(data)
}

func workerJobs(ctx context.Context, db store.Querier, workerID string) ([]any, error) {
	rows, err := db.Query(ctx, `SELECT id, status, done_units, total_units, username, level_id, stage, last_update
		FROM player_evaluations WHERE worker_id = $1 AND status = 'running' ORDER BY last_update DESC`, workerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	jobs := []any{}
	for rows.Next() {
		var entryID, username string
		var status, levelID, stage *string
		var doneUnits, totalUnits *int
		var lastUpdate *time.Time
		if err := rows.Scan(&entryID, &status, &doneUnits, &totalUnits, &username, &levelID, &stage, &lastUpdate); err != nil {
			return nil, err
		}
		done, total, level := 0, 1, "farp"
		if doneUnits != nil {
			done = *doneUnits
		}
		if totalUnits != nil && *totalUnits != 0 {
			total = *totalUnits
		}
		if levelID != nil && *levelID != "" {
			level = *levelID
		}
		jobs = append(jobs, map[string]any{
			"evaluation_id": entryID,
			"status":        status,
			"done_units":    done,
			"total_units":   total,
			"username":      username,
			"level_id":      level,
			"stage":         stage,
			"last_update":   pytime.IsoformatOrNil(lastUpdate),
		})
	}
	return jobs, rows.Err()
}
