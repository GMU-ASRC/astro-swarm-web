package migrations

import (
	"context"
	"fmt"
)

var evaluationIndexes = [][2]string{
	{"ix_player_evaluations_level_status", "ON player_evaluations (level_id, status)"},
	{"ix_player_evaluations_player_created", "ON player_evaluations (player_id, created_at DESC)"},
	{"ix_player_evaluations_queued", "ON player_evaluations (created_at) WHERE status = 'queued'"},
	{"ix_player_evaluations_running", "ON player_evaluations (last_update) WHERE status = 'running'"},
}

type evaluationIndexesMigration struct{}

func (evaluationIndexesMigration) Version() int      { return 2 }
func (evaluationIndexesMigration) Name() string      { return "evaluation_indexes" }
func (evaluationIndexesMigration) ChangesData() bool { return false }

func (evaluationIndexesMigration) Describe(ctx context.Context, run *Context) []string {
	lines := []string{}
	for _, index := range evaluationIndexes {
		lines = append(lines, "create index "+index[0]+" "+index[1])
	}
	return lines
}

func (evaluationIndexesMigration) Run(ctx context.Context, run *Context) error {
	for _, index := range evaluationIndexes {
		var invalid bool
		err := run.Scalar(ctx, `SELECT EXISTS (SELECT 1 FROM pg_index JOIN pg_class index_class ON index_class.oid = pg_index.indexrelid
			WHERE index_class.relname = $1 AND NOT pg_index.indisvalid)`, &invalid, index[0])
		if err != nil {
			return err
		}
		if invalid {
			run.Log("dropping invalid index %s from an earlier failed build", index[0])
			if err := run.ExecuteOutsideTransaction(ctx, "DROP INDEX CONCURRENTLY IF EXISTS "+index[0]); err != nil {
				return err
			}
		}
		run.Log("building index %s", index[0])
		if err := run.ExecuteOutsideTransaction(ctx, "CREATE INDEX CONCURRENTLY IF NOT EXISTS "+index[0]+" "+index[1]); err != nil {
			return err
		}
	}
	return nil
}

func (evaluationIndexesMigration) Verify(ctx context.Context, run *Context) []string {
	problems := []string{}
	for _, index := range evaluationIndexes {
		var valid bool
		err := run.Scalar(ctx, `SELECT EXISTS (SELECT 1 FROM pg_index JOIN pg_class index_class ON index_class.oid = pg_index.indexrelid
			WHERE index_class.relname = $1 AND pg_index.indisvalid)`, &valid, index[0])
		if err != nil {
			return append(problems, err.Error())
		}
		if !valid {
			problems = append(problems, "index "+index[0]+" is missing or invalid")
		}
	}
	return problems
}

var benchmarkTableStatements = []string{
	`CREATE TABLE IF NOT EXISTS benchmarks (
		id varchar(36) PRIMARY KEY,
		content_hash varchar(64),
		level_id varchar(40),
		success_rate double precision,
		attacker_rate double precision,
		defender_rate double precision,
		detection_rate double precision,
		capture_rate double precision,
		summary jsonb,
		replay_meta jsonb,
		replays_shape varchar(10) NOT NULL DEFAULT 'object',
		source_evaluation_id varchar(36),
		created_at timestamptz DEFAULT now()
	)`,
	"CREATE UNIQUE INDEX IF NOT EXISTS ux_benchmarks_content_hash ON benchmarks (content_hash) WHERE content_hash IS NOT NULL",
	"CREATE INDEX IF NOT EXISTS ix_benchmarks_level_rate ON benchmarks (level_id, success_rate DESC)",
	"CREATE INDEX IF NOT EXISTS ix_benchmarks_source ON benchmarks (source_evaluation_id)",
	`CREATE TABLE IF NOT EXISTS benchmark_runs (
		id bigserial PRIMARY KEY,
		benchmark_id varchar(36) NOT NULL REFERENCES benchmarks (id) ON DELETE CASCADE,
		kind varchar(12) NOT NULL,
		position integer NOT NULL,
		parent_position integer,
		sweep_n integer,
		trial integer,
		outcome varchar(20),
		details jsonb NOT NULL DEFAULT '{}'::jsonb,
		frames_key text,
		frames_format varchar(10),
		frames_bytes bigint
	)`,
	"CREATE INDEX IF NOT EXISTS ix_benchmark_runs_position ON benchmark_runs (benchmark_id, kind, position)",
	"CREATE INDEX IF NOT EXISTS ix_benchmark_runs_lookup ON benchmark_runs (benchmark_id, kind, sweep_n, trial)",
	"CREATE INDEX IF NOT EXISTS ix_benchmark_runs_parent ON benchmark_runs (benchmark_id, kind, parent_position)",
	"CREATE INDEX IF NOT EXISTS ix_benchmark_runs_frames_key ON benchmark_runs (frames_key text_pattern_ops)",
	"ALTER TABLE player_evaluations ADD COLUMN IF NOT EXISTS benchmark_id varchar(36)",
	"ALTER TABLE player_evaluations ADD COLUMN IF NOT EXISTS pending_run jsonb",
	"ALTER TABLE player_evaluations ADD COLUMN IF NOT EXISTS job_config jsonb",
	"ALTER TABLE player_evaluations ADD COLUMN IF NOT EXISTS storage_version smallint DEFAULT 1",
	"CREATE INDEX IF NOT EXISTS ix_player_evaluations_benchmark ON player_evaluations (benchmark_id)",
	"CREATE INDEX IF NOT EXISTS ix_player_evaluations_legacy ON player_evaluations (id) WHERE storage_version = 1",
}

var benchmarkRequiredColumns = []string{"benchmark_id", "pending_run", "job_config", "storage_version"}

type benchmarkTablesMigration struct{}

func (benchmarkTablesMigration) Version() int      { return 3 }
func (benchmarkTablesMigration) Name() string      { return "benchmark_tables" }
func (benchmarkTablesMigration) ChangesData() bool { return false }

func (benchmarkTablesMigration) Describe(ctx context.Context, run *Context) []string {
	return []string{
		"create tables benchmarks and benchmark_runs with their indexes",
		fmt.Sprintf("add columns %v to player_evaluations", benchmarkRequiredColumns),
	}
}

func (benchmarkTablesMigration) Run(ctx context.Context, run *Context) error {
	for _, statement := range benchmarkTableStatements {
		if err := run.Execute(ctx, statement); err != nil {
			return err
		}
	}
	return nil
}

func (benchmarkTablesMigration) Verify(ctx context.Context, run *Context) []string {
	problems := []string{}
	for _, table := range []string{"benchmarks", "benchmark_runs"} {
		if exists, err := tableExists(ctx, run, table); err != nil || !exists {
			problems = append(problems, "table "+table+" is missing")
		}
	}
	for _, column := range benchmarkRequiredColumns {
		if exists, err := columnExists(ctx, run, "player_evaluations", column); err != nil || !exists {
			problems = append(problems, "player_evaluations."+column+" is missing")
		}
	}
	return problems
}

type copyToBenchmarksMigration struct{}

func (copyToBenchmarksMigration) Version() int      { return 4 }
func (copyToBenchmarksMigration) Name() string      { return "copy_to_benchmarks" }
func (copyToBenchmarksMigration) ChangesData() bool { return true }

func (copyToBenchmarksMigration) Describe(ctx context.Context, run *Context) []string {
	return []string{"check that no entry still uses the old results and replays storage"}
}

func (migration copyToBenchmarksMigration) Run(ctx context.Context, run *Context) error {
	return nil
}

func (copyToBenchmarksMigration) Verify(ctx context.Context, run *Context) []string {
	var remaining int
	if err := run.Scalar(ctx, "SELECT count(*) FROM player_evaluations WHERE storage_version = 1", &remaining); err != nil {
		return []string{err.Error()}
	}
	if remaining > 0 {
		return []string{fmt.Sprintf("%d entry row(s) still use the old storage; copy them with the Python migrate.py before switching to the Go server", remaining)}
	}
	return nil
}

func All() []Migration {
	return []Migration{
		legacySchema{},
		evaluationIndexesMigration{},
		benchmarkTablesMigration{},
		copyToBenchmarksMigration{},
		archiveLegacyColumnsMigration{},
		binaryColumnsMigration{},
		convertReplaysMigration{},
		retireLegacyReplaysMigration{},
	}
}
