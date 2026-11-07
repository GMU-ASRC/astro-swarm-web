import json

from sqlalchemy import text

import benchmark_store
from database import db
from models import CURRENT_STORAGE_VERSION, LEGACY_STORAGE_VERSION

SELECT_LEGACY_ROW = text(
    "SELECT level_id, results::text AS results_text, replays::text AS replays_text, storage_version"
    " FROM player_evaluations WHERE id = :id FOR UPDATE"
)

MARK_COPIED = text(
    "UPDATE player_evaluations"
    " SET benchmark_id = :benchmark_id,"
    " pending_run = CAST(:pending_run AS jsonb),"
    " storage_version = :storage_version"
    " WHERE id = :id"
)


def _parse(json_text):
    if json_text is None:
        return None
    return json.loads(json_text)


def pending_row_ids(limit):
    rows = db.session.execute(
        text("SELECT id FROM player_evaluations WHERE storage_version = :version ORDER BY id LIMIT :limit"),
        {"version": LEGACY_STORAGE_VERSION, "limit": limit},
    ).scalars().all()
    db.session.commit()
    return rows


def pending_row_count():
    count = db.session.execute(
        text("SELECT count(*) FROM player_evaluations WHERE storage_version = :version"),
        {"version": LEGACY_STORAGE_VERSION},
    ).scalar()
    db.session.commit()
    return count


def copy_row(evaluation_id):
    try:
        row = db.session.execute(SELECT_LEGACY_ROW, {"id": evaluation_id}).mappings().first()
        if row is None or row["storage_version"] != LEGACY_STORAGE_VERSION:
            db.session.rollback()
            return False
        results = _parse(row["results_text"])
        replays = _parse(row["replays_text"])
        pending_run = replays.get("pending_run") if isinstance(replays, dict) else None

        benchmark_id = None
        if benchmark_store.has_benchmark_data(results, replays):
            benchmark = benchmark_store.build_benchmark(
                evaluation_id,
                row["level_id"],
                results,
                replays,
                source_evaluation_id=evaluation_id,
            )
            benchmark_id = benchmark.id

        db.session.execute(MARK_COPIED, {
            "id": evaluation_id,
            "benchmark_id": benchmark_id,
            "pending_run": json.dumps(pending_run) if pending_run is not None else None,
            "storage_version": CURRENT_STORAGE_VERSION,
        })
        db.session.commit()
        return True
    except Exception:
        db.session.rollback()
        raise


def copy_all_pending(log=None, batch_size=20):
    copied = 0
    while True:
        evaluation_ids = pending_row_ids(batch_size)
        if not evaluation_ids:
            return copied
        for evaluation_id in evaluation_ids:
            if copy_row(evaluation_id):
                copied += 1
                if log is not None:
                    log("copied %s (%d so far)", evaluation_id, copied)
