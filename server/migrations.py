import logging

from sqlalchemy import text

from database import db

logger = logging.getLogger(__name__)

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
