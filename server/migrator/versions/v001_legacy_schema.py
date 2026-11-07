from database import db
from migrator.migration import Migration

LEVEL_SHIFT_KEY = "level_ids_shifted_v2"
PILOT_SHIFT_KEY = "level_ids_shifted_v3"
PILOT_HIDDEN_KEY = "level_ids_shifted_v4"
PILOT_HIDDEN_RENAMES = (("farp6", "farp9"), ("farp7", "farp6"), ("farp8", "farp7"))

LEGACY_TABLES = (
    "sim_configs",
    "sim_runs",
    "player_evaluations",
    "app_settings",
    "workers",
    "survive_matches",
    "training_scores",
    "leaderboard_entries",
    "admin_users",
    "admin_sessions",
    "simulator_entries",
)

STATEMENTS = [
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
    "UPDATE player_evaluations SET level_id = 'farp6' WHERE level_id = 'farp4'"
    " AND NOT EXISTS (SELECT 1 FROM app_settings WHERE key = '%s')" % LEVEL_SHIFT_KEY,
    "UPDATE player_evaluations SET level_id = 'farp5' WHERE level_id = 'farp3'"
    " AND NOT EXISTS (SELECT 1 FROM app_settings WHERE key = '%s')" % LEVEL_SHIFT_KEY,
    "UPDATE player_evaluations SET level_id = 'farp1' WHERE level_id = 'farp' OR level_id IS NULL",
    "UPDATE player_evaluations SET level_id = 'farp7' WHERE level_id = 'farp6'"
    " AND EXISTS (SELECT 1 FROM app_settings WHERE key = '%s')"
    " AND NOT EXISTS (SELECT 1 FROM app_settings WHERE key = '%s')" % (LEVEL_SHIFT_KEY, PILOT_SHIFT_KEY),
    "UPDATE player_evaluations SET level_id = 'farp6' WHERE level_id = 'farp5'"
    " AND EXISTS (SELECT 1 FROM app_settings WHERE key = '%s')"
    " AND NOT EXISTS (SELECT 1 FROM app_settings WHERE key = '%s')" % (LEVEL_SHIFT_KEY, PILOT_SHIFT_KEY),
    "INSERT INTO app_settings (key, value)"
    " SELECT '%s', '1' WHERE EXISTS (SELECT 1 FROM app_settings WHERE key = '%s')"
    " ON CONFLICT (key) DO NOTHING" % (PILOT_SHIFT_KEY, LEVEL_SHIFT_KEY),
    "INSERT INTO app_settings (key, value) VALUES ('%s', '1')"
    " ON CONFLICT (key) DO NOTHING" % LEVEL_SHIFT_KEY,
    *[
        "UPDATE player_evaluations SET level_id = '%s' WHERE level_id = '%s'"
        " AND NOT EXISTS (SELECT 1 FROM app_settings WHERE key = '%s')" % (new_id, old_id, PILOT_HIDDEN_KEY)
        for old_id, new_id in PILOT_HIDDEN_RENAMES
    ],
    *[
        "UPDATE app_settings SET key = 'level_enabled_%s' WHERE key = 'level_enabled_%s'"
        " AND NOT EXISTS (SELECT 1 FROM app_settings WHERE key = '%s')" % (new_id, old_id, PILOT_HIDDEN_KEY)
        for old_id, new_id in PILOT_HIDDEN_RENAMES
    ],
    "INSERT INTO app_settings (key, value) VALUES ('%s', '1')"
    " ON CONFLICT (key) DO NOTHING" % PILOT_HIDDEN_KEY,
    "ALTER TABLE player_evaluations ALTER COLUMN level_id SET DEFAULT 'farp1'",
    "ALTER TABLE player_evaluations SET ("
    " autovacuum_vacuum_scale_factor = 0.02,"
    " autovacuum_vacuum_threshold = 5,"
    " autovacuum_analyze_scale_factor = 0.05,"
    " toast.autovacuum_vacuum_scale_factor = 0.02,"
    " toast.autovacuum_vacuum_threshold = 5)",
]


class LegacySchema(Migration):
    version = 1
    name = "legacy_schema"
    changes_data = False

    def describe(self, context):
        return ["create missing tables: %s" % ", ".join(LEGACY_TABLES), "apply %d legacy statements" % len(STATEMENTS)]

    def run(self, context):
        import models
        import simulator_models

        legacy_tables = [db.metadata.tables[name] for name in LEGACY_TABLES]
        db.metadata.create_all(bind=context.connection, tables=legacy_tables)
        context.connection.commit()
        skipped = 0
        for statement in STATEMENTS:
            try:
                context.execute(statement)
            except Exception as error:
                context.connection.rollback()
                skipped += 1
                context.log("skipped statement (%s): %s", statement, error)
        context.log("applied %d statement(s), skipped %d", len(STATEMENTS) - skipped, skipped)

    def verify(self, context):
        problems = []
        for name in LEGACY_TABLES:
            if not context.scalar("SELECT to_regclass(:name) IS NOT NULL", {"name": "public." + name}):
                problems.append("table %s is missing" % name)
        return problems
