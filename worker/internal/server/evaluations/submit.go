package evaluations

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"astroswarm/worker/internal/server/benchmarks"
	"astroswarm/worker/internal/server/entries"
	"astroswarm/worker/internal/server/httpx"
	"astroswarm/worker/internal/server/pytime"
	"astroswarm/worker/internal/server/settings"
	"astroswarm/worker/internal/server/store"
	"astroswarm/worker/internal/server/validate"
)

const outdatedVersionStatus = 426

func (api *API) requireCurrentVersion(ctx context.Context, version string) error {
	current, err := api.deps.Settings.VersionIsCurrent(ctx, version)
	if err != nil || current {
		return err
	}
	required, err := api.deps.Settings.RequiredGameVersion(ctx)
	if err != nil {
		return err
	}
	return httpx.NewError(outdatedVersionStatus, "This build is out of date. Update to "+required+" to submit entries.")
}

func dedupKey(algorithm any, placements any, schedule []any) (string, error) {
	content := map[string]any{"a": emptyListIfFalsy(algorithm), "p": emptyListIfFalsy(placements)}
	if len(schedule) > 1 {
		content["g"] = schedule
	}
	return benchmarks.PythonJSON(content)
}

func emptyListIfFalsy(value any) any {
	if value == nil {
		return []any{}
	}
	if list, isList := value.([]any); isList && len(list) == 0 {
		return []any{}
	}
	return value
}

func (api *API) hasPlayerDuplicate(ctx context.Context, submission validate.EvaluationSubmission) (bool, error) {
	key, err := dedupKey(submission.Algorithm, submission.Placements, submission.GenomeSchedule)
	if err != nil {
		return false, err
	}
	candidates, err := entries.List(ctx, api.deps.Store.Pool, entries.SelectEntries(
		"player_evaluations.player_id = $1 AND player_evaluations.level_id = ANY($2) AND player_evaluations.status IS DISTINCT FROM 'cancelled'"),
		submission.PlayerID, settings.LevelIDsFor(submission.LevelID))
	if err != nil {
		return false, err
	}
	for _, candidate := range candidates {
		candidateKey, err := dedupKey(candidate.AlgorithmValue(), candidate.PlacementsValue(), candidate.GenomeScheduleList())
		if err != nil {
			return false, err
		}
		if candidateKey == key {
			return true, nil
		}
	}
	return false, nil
}

type newEntry struct {
	playerID       string
	username       string
	levelID        string
	algorithm      any
	placements     []any
	genomeSchedule []any
	trials         int
	gameVersion    string
	collisions     bool
	createdAt      any
	pendingRun     any
}

func insertEntry(ctx context.Context, db store.Querier, entry newEntry) (string, error) {
	entryID := uuid.NewString()
	jsonValues := []any{}
	for _, value := range []any{entry.algorithm, entry.placements, entry.genomeSchedule} {
		encoded, err := store.JSONParameter(value)
		if err != nil {
			return "", err
		}
		jsonValues = append(jsonValues, encoded)
	}
	pendingRun, err := store.JSONParameter(entry.pendingRun)
	if err != nil {
		return "", err
	}
	now := pytime.Now()
	_, err = db.Exec(ctx, `INSERT INTO player_evaluations (id, player_id, username, level_id, algorithm, placements, genome_schedule,
		status, progress, n_max, trials, pending_run, storage_version, done_units, total_units, last_update,
		game_version, defender_count, collisions, created_at)
		VALUES ($1, $2, $3, $4, $5::json, $6::json, $7::json, 'queued', 0, 40, $8, $9::jsonb, $10, 0, 1, $11,
		$12, $13, $14, $15)`,
		entryID, entry.playerID, entry.username, entry.levelID, jsonValues[0], jsonValues[1], jsonValues[2], entry.trials,
		pendingRun, entries.CurrentStorageVersion, now, entry.gameVersion, len(entry.placements),
		entry.collisions, entry.createdAt)
	return entryID, err
}

func (api *API) submit(w http.ResponseWriter, r *http.Request) error {
	if _, err := api.deps.Auth.RequireAdmin(r); err != nil {
		return err
	}
	data, found := httpx.ReadJSONObject(r)
	if !found {
		return httpx.BadRequest("Invalid JSON data")
	}
	submission, err := validate.ParseEvaluationSubmission(data)
	if err != nil {
		return httpx.BadRequest(err.Error())
	}
	ctx := r.Context()
	if err := api.requireCurrentVersion(ctx, submission.GameVersion); err != nil {
		return err
	}
	if !settings.IsBenchmarkLevel(submission.LevelID) {
		return httpx.BadRequest("Level is not benchmarked")
	}
	if enabled, err := api.deps.Settings.LevelEnabled(ctx, submission.LevelID); err != nil || !enabled {
		if err != nil {
			return err
		}
		return httpx.BadRequest("Level is disabled")
	}
	duplicate, err := api.hasPlayerDuplicate(ctx, submission)
	if err != nil {
		return err
	}
	if duplicate {
		return httpx.Conflict("You have already submitted this algorithm and placement for this level")
	}

	var entryID string
	err = api.deps.Store.InTransaction(ctx, func(transaction pgx.Tx) error {
		entryID, err = insertEntry(ctx, transaction, newEntry{
			playerID: submission.PlayerID, username: submission.Username, levelID: submission.LevelID,
			algorithm: submission.Algorithm, placements: submission.Placements, genomeSchedule: submission.GenomeSchedule,
			trials: submission.Trials, gameVersion: submission.GameVersion, collisions: submission.Collisions,
			createdAt: submission.CreatedAt,
		})
		if err != nil {
			return err
		}
		entry, err := entries.Get(ctx, transaction, entryID)
		if err != nil {
			return err
		}
		return api.linkOrQueue(ctx, transaction, entry)
	})
	if err != nil {
		return err
	}
	return api.respondWithEntry(w, r, entryID, http.StatusAccepted)
}

func (api *API) linkOrQueue(ctx context.Context, transaction pgx.Tx, entry *entries.Entry) error {
	if !settings.IsPilotLevel(entry.Level()) {
		jobConfig, err := benchmarks.JobConfig(ctx, api.deps.Settings, api.deps.Config, entry.Level(), entry.Collisions != nil && *entry.Collisions)
		if err != nil {
			return err
		}
		contentHash, err := benchmarks.ContentHash(entry.HashInput(), jobConfig, api.deps.Config.SimulatorVersion)
		if err != nil {
			return err
		}
		benchmarkID, found, err := benchmarks.FindByHash(ctx, transaction, contentHash)
		if err != nil {
			return err
		}
		if found {
			_, err = transaction.Exec(ctx, `UPDATE player_evaluations SET benchmark_id = $2, progress = 1.0, status = 'done',
				completed_at = $3 WHERE id = $1`, entry.ID, benchmarkID, pytime.Now())
			return err
		}
	}
	return entries.Queue(ctx, transaction, api.deps.Settings, entry)
}

func (api *API) respondWithEntry(w http.ResponseWriter, r *http.Request, entryID string, status int) error {
	ctx := r.Context()
	entry, err := entries.Get(ctx, api.deps.Store.Pool, entryID)
	if err != nil {
		return err
	}
	if entry == nil {
		return httpx.BadRequest("Evaluation not found")
	}
	summary, err := entries.Summary(ctx, api.deps.Store.Pool, entry)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, status, entry.ToMap(summary))
	return nil
}

func (api *API) submitRun(w http.ResponseWriter, r *http.Request) error {
	if _, err := api.deps.Auth.RequireAdmin(r); err != nil {
		return err
	}
	data, found := httpx.ReadJSONObject(r)
	if !found {
		return httpx.BadRequest("Invalid JSON data")
	}
	submission, err := validate.ParseRunSubmission(data)
	if err != nil {
		return httpx.BadRequest(err.Error())
	}
	ctx := r.Context()
	if err := api.requireCurrentVersion(ctx, submission.GameVersion); err != nil {
		return err
	}
	if !settings.IsPilotLevel(submission.LevelID) {
		return httpx.BadRequest("Level does not accept piloted runs")
	}
	if enabled, err := api.deps.Settings.LevelEnabled(ctx, submission.LevelID); err != nil || !enabled {
		if err != nil {
			return err
		}
		return httpx.BadRequest("Level is disabled")
	}
	var entryID string
	err = api.deps.Store.InTransaction(ctx, func(transaction pgx.Tx) error {
		entryID, err = insertEntry(ctx, transaction, newEntry{
			playerID: submission.PlayerID, username: submission.Username, levelID: submission.LevelID,
			algorithm: submission.Algorithm, placements: submission.Placements, genomeSchedule: submission.GenomeSchedule,
			trials: 1, gameVersion: submission.GameVersion, createdAt: submission.CreatedAt, pendingRun: submission.Run,
		})
		if err != nil {
			return err
		}
		entry, err := entries.Get(ctx, transaction, entryID)
		if err != nil {
			return err
		}
		return entries.Queue(ctx, transaction, api.deps.Settings, entry)
	})
	if err != nil {
		return err
	}
	return api.respondWithEntry(w, r, entryID, http.StatusAccepted)
}
