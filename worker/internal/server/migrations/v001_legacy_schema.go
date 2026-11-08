package migrations

import (
	"context"
	"fmt"
)

const (
	levelShiftKey  = "level_ids_shifted_v2"
	pilotShiftKey  = "level_ids_shifted_v3"
	pilotHiddenKey = "level_ids_shifted_v4"
)

var pilotHiddenRenames = [][2]string{{"farp6", "farp9"}, {"farp7", "farp6"}, {"farp8", "farp7"}}

func legacyStatements() []string {
	statements := []string{
		"ALTER TABLE player_evaluations ADD COLUMN IF NOT EXISTS progress double precision DEFAULT 0",
		"ALTER TABLE player_evaluations ADD COLUMN IF NOT EXISTS replays json DEFAULT '[]'::json",
		"ALTER TABLE player_evaluations ADD COLUMN IF NOT EXISTS placements json DEFAULT '[]'::json",
		"ALTER TABLE player_evaluations ADD COLUMN IF NOT EXISTS level_id varchar(40) DEFAULT 'farp'",
		"ALTER TABLE player_evaluations ADD COLUMN IF NOT EXISTS worker_id varchar(64)",
		"ALTER TABLE sim_runs ADD COLUMN IF NOT EXISTS thumbnail_filename varchar(255)",
		"ALTER TABLE sim_runs ADD COLUMN IF NOT EXISTS video_filename varchar(255)",
		"ALTER TABLE sim_runs ADD COLUMN IF NOT EXISTS frame_count integer DEFAULT 0",
		"ALTER TABLE workers ADD COLUMN IF NOT EXISTS system_stats json",
		"ALTER TABLE player_evaluations ADD COLUMN IF NOT EXISTS stage varchar(200)",
		"ALTER TABLE player_evaluations ADD COLUMN IF NOT EXISTS game_version varchar(20) DEFAULT 'v0.0.4'",
		"ALTER TABLE player_evaluations ADD COLUMN IF NOT EXISTS defender_count integer DEFAULT 0",
		"ALTER TABLE player_evaluations ADD COLUMN IF NOT EXISTS xp_awarded integer",
		"ALTER TABLE player_evaluations ADD COLUMN IF NOT EXISTS collisions boolean DEFAULT false",
		"ALTER TABLE player_evaluations ADD COLUMN IF NOT EXISTS genome_schedule json DEFAULT '[]'::json",
		"ALTER TABLE player_evaluations ADD COLUMN IF NOT EXISTS coins_awarded integer",
		"ALTER TABLE workers DROP COLUMN IF EXISTS max_jobs",
		"ALTER TABLE player_evaluations ADD COLUMN IF NOT EXISTS done_units integer DEFAULT 0",
		"ALTER TABLE player_evaluations ADD COLUMN IF NOT EXISTS total_units integer DEFAULT 1",
		"ALTER TABLE player_evaluations ADD COLUMN IF NOT EXISTS last_update timestamptz DEFAULT now()",
		"DROP TABLE IF EXISTS evaluation_shards",
		fmt.Sprintf("UPDATE player_evaluations SET level_id = 'farp6' WHERE level_id = 'farp4' AND NOT EXISTS (SELECT 1 FROM app_settings WHERE key = '%s')", levelShiftKey),
		fmt.Sprintf("UPDATE player_evaluations SET level_id = 'farp5' WHERE level_id = 'farp3' AND NOT EXISTS (SELECT 1 FROM app_settings WHERE key = '%s')", levelShiftKey),
		"UPDATE player_evaluations SET level_id = 'farp1' WHERE level_id = 'farp' OR level_id IS NULL",
		fmt.Sprintf("UPDATE player_evaluations SET level_id = 'farp7' WHERE level_id = 'farp6' AND EXISTS (SELECT 1 FROM app_settings WHERE key = '%s') AND NOT EXISTS (SELECT 1 FROM app_settings WHERE key = '%s')", levelShiftKey, pilotShiftKey),
		fmt.Sprintf("UPDATE player_evaluations SET level_id = 'farp6' WHERE level_id = 'farp5' AND EXISTS (SELECT 1 FROM app_settings WHERE key = '%s') AND NOT EXISTS (SELECT 1 FROM app_settings WHERE key = '%s')", levelShiftKey, pilotShiftKey),
		fmt.Sprintf("INSERT INTO app_settings (key, value) SELECT '%s', '1' WHERE EXISTS (SELECT 1 FROM app_settings WHERE key = '%s') ON CONFLICT (key) DO NOTHING", pilotShiftKey, levelShiftKey),
		fmt.Sprintf("INSERT INTO app_settings (key, value) VALUES ('%s', '1') ON CONFLICT (key) DO NOTHING", levelShiftKey),
	}
	for _, rename := range pilotHiddenRenames {
		statements = append(statements, fmt.Sprintf(
			"UPDATE player_evaluations SET level_id = '%s' WHERE level_id = '%s' AND NOT EXISTS (SELECT 1 FROM app_settings WHERE key = '%s')",
			rename[1], rename[0], pilotHiddenKey))
	}
	for _, rename := range pilotHiddenRenames {
		statements = append(statements, fmt.Sprintf(
			"UPDATE app_settings SET key = 'level_enabled_%s' WHERE key = 'level_enabled_%s' AND NOT EXISTS (SELECT 1 FROM app_settings WHERE key = '%s')",
			rename[1], rename[0], pilotHiddenKey))
	}
	return append(statements,
		fmt.Sprintf("INSERT INTO app_settings (key, value) VALUES ('%s', '1') ON CONFLICT (key) DO NOTHING", pilotHiddenKey),
		"ALTER TABLE player_evaluations ALTER COLUMN level_id SET DEFAULT 'farp1'",
		"ALTER TABLE player_evaluations SET (autovacuum_vacuum_scale_factor = 0.02, autovacuum_vacuum_threshold = 5, autovacuum_analyze_scale_factor = 0.05, toast.autovacuum_vacuum_scale_factor = 0.02, toast.autovacuum_vacuum_threshold = 5)",
	)
}

type legacySchema struct{}

func (legacySchema) Version() int      { return 1 }
func (legacySchema) Name() string      { return "legacy_schema" }
func (legacySchema) ChangesData() bool { return false }

func (legacySchema) Describe(ctx context.Context, run *Context) []string {
	return []string{
		fmt.Sprintf("create missing tables: %v", legacyRequiredTables),
		fmt.Sprintf("apply %d legacy statements", len(legacyStatements())),
	}
}

func (legacySchema) Run(ctx context.Context, run *Context) error {
	for _, statement := range legacyTables {
		if err := run.Execute(ctx, statement); err != nil {
			return err
		}
	}
	statements := legacyStatements()
	skipped := 0
	for _, statement := range statements {
		if err := run.Execute(ctx, statement); err != nil {
			skipped++
			run.Log("skipped statement (%s): %v", statement, err)
		}
	}
	run.Log("applied %d statement(s), skipped %d", len(statements)-skipped, skipped)
	return nil
}

func (legacySchema) Verify(ctx context.Context, run *Context) []string {
	problems := []string{}
	for _, table := range legacyRequiredTables {
		exists, err := tableExists(ctx, run, table)
		if err != nil {
			return append(problems, err.Error())
		}
		if !exists {
			problems = append(problems, "table "+table+" is missing")
		}
	}
	return problems
}
