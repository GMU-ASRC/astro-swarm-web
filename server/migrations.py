import logging

from sqlalchemy import text

from database import db

logger = logging.getLogger(__name__)

LEVEL_SHIFT_KEY = "level_ids_shifted_v2"
PILOT_SHIFT_KEY = "level_ids_shifted_v3"

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
    "ALTER TABLE workers DROP COLUMN IF EXISTS max_jobs",
    # One worker runs one whole evaluation, so the job state moved onto the
    # evaluation itself and the shard table is gone.
    "ALTER TABLE player_evaluations ADD COLUMN IF NOT EXISTS done_units integer DEFAULT 0",
    "ALTER TABLE player_evaluations ADD COLUMN IF NOT EXISTS total_units integer DEFAULT 1",
    "ALTER TABLE player_evaluations ADD COLUMN IF NOT EXISTS last_update timestamptz DEFAULT now()",
    "DROP TABLE IF EXISTS evaluation_shards",
    "UPDATE player_evaluations SET level_id = 'farp6' WHERE level_id = 'farp4'"
    " AND NOT EXISTS (SELECT 1 FROM app_settings WHERE key = '%s')" % LEVEL_SHIFT_KEY,
    "UPDATE player_evaluations SET level_id = 'farp5' WHERE level_id = 'farp3'"
    " AND NOT EXISTS (SELECT 1 FROM app_settings WHERE key = '%s')" % LEVEL_SHIFT_KEY,
    "UPDATE player_evaluations SET level_id = 'farp1' WHERE level_id = 'farp' OR level_id IS NULL",
    # Levels 3 and 4 became continuous-wave levels and a new siege level took
    # slot 5, so the two piloted levels moved up one. This only runs once the
    # earlier shift has been recorded, so a database that is behind on both
    # never gets shifted twice in one pass.
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
    "ALTER TABLE player_evaluations ALTER COLUMN level_id SET DEFAULT 'farp1'",
    "ALTER TABLE player_evaluations SET ("
    " autovacuum_vacuum_scale_factor = 0.02,"
    " autovacuum_vacuum_threshold = 5,"
    " autovacuum_analyze_scale_factor = 0.05,"
    " toast.autovacuum_vacuum_scale_factor = 0.02,"
    " toast.autovacuum_vacuum_threshold = 5)",
]


def apply(log=None):
    log = log or logger
    ok = 0
    skipped = 0
    for statement in STATEMENTS:
        try:
            db.session.execute(text(statement))
            db.session.commit()
            ok += 1
        except Exception as exc:
            db.session.rollback()
            skipped += 1
            log.warning("Migration skipped (%s): %s", statement, exc)
    return ok, skipped


def main():
    from flask import Flask

    from config import Config

    logging.basicConfig(level=logging.INFO, format="%(levelname)s %(message)s")

    app = Flask(__name__)
    app.config.from_object(Config)
    db.init_app(app)

    with app.app_context():
        ok, skipped = apply()
        db.engine.dispose()

    print(f"{ok} statement(s) ran, {skipped} skipped, out of {len(STATEMENTS)}")


if __name__ == "__main__":
    main()
