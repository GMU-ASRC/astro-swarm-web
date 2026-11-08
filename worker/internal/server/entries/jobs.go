package entries

import (
	"context"

	"astroswarm/worker/internal/server/benchmarks"
	"astroswarm/worker/internal/server/config"
	"astroswarm/worker/internal/server/pytime"
	"astroswarm/worker/internal/server/settings"
	"astroswarm/worker/internal/server/store"
	"astroswarm/worker/internal/server/values"
)

func PendingRun(ctx context.Context, db store.Querier, entryID string) ([]byte, error) {
	var raw []byte
	err := db.QueryRow(ctx, "SELECT pending_run FROM player_evaluations WHERE id = $1", entryID).Scan(&raw)
	if store.IsNoRows(err) {
		return nil, nil
	}
	if len(raw) == 0 || string(raw) == "null" {
		return nil, err
	}
	return raw, err
}

func TotalUnits(ctx context.Context, db store.Querier, appSettings *settings.Settings, entry *Entry) (int, error) {
	pendingRun, err := PendingRun(ctx, db, entry.ID)
	if err != nil {
		return 0, err
	}
	if pendingRun != nil {
		return 1, nil
	}
	trials := intOr(entry.Trials, 0)
	if settings.IsAssaultLevel(entry.Level()) {
		return max(1, trials+settings.AssaultSweepMaxFor(entry.Level())*settings.AssaultSweepTrials), nil
	}
	sweepMax, err := appSettings.SweepMax(ctx)
	if err != nil {
		return 0, err
	}
	sweepTrials, err := appSettings.SweepTrials(ctx)
	if err != nil {
		return 0, err
	}
	return max(1, trials+sweepMax*sweepTrials), nil
}

func Queue(ctx context.Context, db store.Querier, appSettings *settings.Settings, entry *Entry) error {
	totalUnits, err := TotalUnits(ctx, db, appSettings, entry)
	if err != nil {
		return err
	}
	_, err = db.Exec(ctx, `UPDATE player_evaluations SET status = 'queued', progress = 0, stage = NULL, error = NULL,
		worker_id = NULL, done_units = 0, total_units = $2, last_update = $3 WHERE id = $1`,
		entry.ID, totalUnits, pytime.Now())
	return err
}

func (entry *Entry) HashInput() benchmarks.HashInput {
	var trials any
	if entry.Trials != nil {
		trials = *entry.Trials
	}
	return benchmarks.HashInput{
		LevelID:        entry.Level(),
		Algorithm:      entry.AlgorithmValue(),
		Placements:     entry.PlacementsValue(),
		GenomeSchedule: entry.GenomeScheduleList(),
		Trials:         trials,
		Collisions:     entry.Collisions != nil && *entry.Collisions,
	}
}

func (entry *Entry) FinishedJobHash(ctx context.Context, appSettings *settings.Settings, serverConfig config.Config) (*string, error) {
	if settings.IsPilotLevel(entry.Level()) {
		return nil, nil
	}
	jobConfig := entry.JobConfigValue()
	if !values.Truthy(jobConfig) {
		freshConfig, err := benchmarks.JobConfig(ctx, appSettings, serverConfig, entry.Level(), entry.Collisions != nil && *entry.Collisions)
		if err != nil {
			return nil, err
		}
		jobConfig = freshConfig
	}
	contentHash, err := benchmarks.ContentHash(entry.HashInput(), jobConfig, serverConfig.SimulatorVersion)
	if err != nil {
		return nil, err
	}
	return &contentHash, nil
}
