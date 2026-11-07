from migrator.migration import Migration

STATEMENTS = [
    """
    CREATE TABLE IF NOT EXISTS benchmarks (
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
    )
    """,
    "CREATE UNIQUE INDEX IF NOT EXISTS ux_benchmarks_content_hash ON benchmarks (content_hash) WHERE content_hash IS NOT NULL",
    "CREATE INDEX IF NOT EXISTS ix_benchmarks_level_rate ON benchmarks (level_id, success_rate DESC)",
    "CREATE INDEX IF NOT EXISTS ix_benchmarks_source ON benchmarks (source_evaluation_id)",
    """
    CREATE TABLE IF NOT EXISTS benchmark_runs (
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
    )
    """,
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
]

REQUIRED_COLUMNS = ("benchmark_id", "pending_run", "job_config", "storage_version")


class BenchmarkTables(Migration):
    version = 3
    name = "benchmark_tables"
    changes_data = False

    def describe(self, context):
        return [
            "create tables benchmarks and benchmark_runs with their indexes",
            "add columns %s to player_evaluations" % ", ".join(REQUIRED_COLUMNS),
        ]

    def run(self, context):
        for statement in STATEMENTS:
            context.execute(statement)

    def verify(self, context):
        problems = []
        for table in ("benchmarks", "benchmark_runs"):
            if not context.scalar("SELECT to_regclass(:name) IS NOT NULL", {"name": "public." + table}):
                problems.append("table %s is missing" % table)
        for column in REQUIRED_COLUMNS:
            exists = context.scalar(
                "SELECT EXISTS (SELECT 1 FROM information_schema.columns"
                " WHERE table_name = 'player_evaluations' AND column_name = :column)",
                {"column": column},
            )
            if not exists:
                problems.append("player_evaluations.%s is missing" % column)
        return problems
