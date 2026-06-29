from datetime import datetime, timedelta, timezone

from flask import Blueprint, jsonify, request
from werkzeug.exceptions import BadRequest, NotFound, Unauthorized

import merge
from auth import require_admin
from app_settings import get_enemy_start, get_sweep_max, get_sweep_trials
from config import Config
from database import db
from models import SHARD_STALE_SECONDS, EvaluationShard, PlayerEvaluation, Worker

workers_bp = Blueprint("workers", __name__, url_prefix="/api")


def _require_api_key():
    if request.headers.get("X-API-Key") != Config.API_SECRET_KEY:
        raise Unauthorized("Invalid API key")


def _now():
    return datetime.now(timezone.utc)


def _chunks(total, parts):
    ranges = []
    base = total // parts
    extra = total % parts
    start = 0
    for index in range(parts):
        count = base + (1 if index < extra else 0)
        ranges.append((start, count))
        start += count
    return ranges


def _weighted_sweep_chunks(total, parts):
    # Split the ring sweep n=1..total into contiguous ranges of roughly equal
    # cost. Per-match work grows with n (more defenders), so an equal-width
    # split makes high-n shards far heavier than low-n ones; here we balance by
    # sum-of-n instead, isolating the heaviest n-values into smaller shards.
    if parts <= 0:
        return []
    if total <= 0:
        return [(0, 0) for _ in range(parts)]
    total_weight = total * (total + 1) / 2.0
    ranges = []
    assigned = 0
    for part in range(parts):
        remaining_parts = parts - part
        remaining_n = total - assigned
        if remaining_n <= remaining_parts:
            count = 1 if remaining_n > 0 else 0
        else:
            target = total_weight * (part + 1) / parts
            max_take = remaining_n - (remaining_parts - 1)
            count = 0
            cum = assigned * (assigned + 1) / 2.0
            while count < max_take:
                next_n = assigned + count + 1
                if cum + next_n > target and count >= 1:
                    break
                cum += next_n
                count += 1
            count = max(1, count)
        ranges.append((assigned, count))
        assigned += count
    if assigned < total:
        start, count = ranges[-1]
        ranges[-1] = (start, count + (total - assigned))
    return ranges


def create_shards(evaluation):
    # Split an evaluation into small work units (a trial-range plus a ring-sweep
    # n-range) so any number of workers can each claim and run a portion of it.
    EvaluationShard.query.filter_by(evaluation_id=evaluation.id).delete(synchronize_session=False)

    trials = int(evaluation.trials or 0)
    sweep_max = get_sweep_max()
    sweep_trials = get_sweep_trials()
    total_work = max(1, trials + sweep_max * sweep_trials)
    # Use at least one shard per ring-sweep n-value when possible, so a single
    # heavy high-n value is never bundled with others into one over-long shard.
    parts = max(1, min(max(Config.EVAL_SHARD_COUNT, sweep_max), total_work))

    trial_ranges = _chunks(trials, parts)
    sweep_ranges = _weighted_sweep_chunks(sweep_max, parts)

    created = 0
    for index in range(parts):
        trial_start, trial_count = trial_ranges[index]
        sweep_start, sweep_count = sweep_ranges[index]
        if trial_count == 0 and sweep_count == 0:
            continue
        units = max(1, trial_count + sweep_count * sweep_trials)
        db.session.add(EvaluationShard(
            evaluation_id=evaluation.id,
            shard_index=created,
            trial_start=trial_start,
            trial_count=trial_count,
            n_start=sweep_start + 1,
            n_count=sweep_count,
            total_units=units,
            status="queued",
        ))
        created += 1


def _reap_stale(worker_ids=None):
    # Requeue running shards whose worker has gone offline (or whose last update
    # is too old), so the queue does not stall on a dead node. Completed shards
    # are never touched, so a finished job is never re-run.
    online = {w.id for w in Worker.query.all() if w.is_online()}
    cutoff = _now() - timedelta(seconds=SHARD_STALE_SECONDS)
    stale = (
        EvaluationShard.query.filter_by(status="running")
        .filter(EvaluationShard.last_update < cutoff)
        .all()
    )
    changed = False
    for shard in stale:
        if shard.worker_id in online:
            continue
        shard.status = "queued"
        shard.worker_id = None
        shard.done_units = 0
        shard.last_update = _now()
        changed = True
    for worker in Worker.query.all():
        if not worker.is_online() and worker.reported_status != "offline":
            worker.current_job_id = None
            worker.reported_status = "offline"
            changed = True
    if changed:
        db.session.commit()


def _shard_payload(evaluation, shard):
    enemy_x, enemy_y = get_enemy_start()
    return {
        "shard_id": shard.id,
        "evaluation_id": evaluation.id,
        "algorithm": evaluation.algorithm or [],
        "placements": evaluation.placements or [],
        "trials": evaluation.trials,
        "trial_start": shard.trial_start,
        "trial_count": shard.trial_count,
        "n_start": shard.n_start,
        "n_count": shard.n_count,
        "total_units": shard.total_units,
        "config": {
            "seed": Config.EVAL_SEED,
            "sweep_max": get_sweep_max(),
            "sweep_trials": get_sweep_trials(),
            "match_seconds": Config.EVAL_MATCH_CAP_SECONDS,
            "enemy_x": enemy_x,
            "enemy_y": enemy_y,
        },
    }


def _update_progress(evaluation_id):
    shards = EvaluationShard.query.filter_by(evaluation_id=evaluation_id).all()
    if not shards:
        return
    total = sum(max(1, s.total_units) for s in shards)
    done = sum(min(s.done_units, s.total_units) for s in shards)
    evaluation = db.session.get(PlayerEvaluation, evaluation_id)
    if evaluation is not None and evaluation.status == "running":
        evaluation.progress = round(min(0.99, done / max(1, total)), 3)


def _finalize_if_complete(evaluation_id):
    # Called while holding a FOR UPDATE lock on the evaluation row so concurrent
    # shard results from different workers cannot finalize the same job twice.
    shards = EvaluationShard.query.filter_by(evaluation_id=evaluation_id).all()
    if not shards:
        return
    if any(s.status in ("queued", "running") for s in shards):
        return

    evaluation = db.session.get(PlayerEvaluation, evaluation_id)
    if evaluation is None or evaluation.status not in ("running", "queued"):
        return

    failed = next((s for s in shards if s.status == "failed"), None)
    if failed is not None:
        evaluation.status = "failed"
        evaluation.error = failed.error or "worker error"
        evaluation.completed_at = _now()
        evaluation.worker_id = None
        evaluation.stage = None
        EvaluationShard.query.filter_by(evaluation_id=evaluation_id).delete(synchronize_session=False)
        return

    parts = [s.result for s in sorted(shards, key=lambda s: s.shard_index)]
    results, replays = merge.merge_shards(parts)
    evaluation.results = results
    evaluation.replays = replays
    evaluation.status = "done"
    evaluation.progress = 1.0
    evaluation.error = None
    evaluation.completed_at = _now()
    evaluation.worker_id = None
    evaluation.stage = None
    EvaluationShard.query.filter_by(evaluation_id=evaluation_id).delete(synchronize_session=False)


@workers_bp.post("/worker/register")
def register_worker():
    _require_api_key()
    data = request.get_json(silent=True) or {}
    worker_id = str(data.get("worker_id", "")).strip()
    if not worker_id:
        raise BadRequest("worker_id is required")

    worker = db.session.get(Worker, worker_id)
    is_new = worker is None
    if is_new:
        worker = Worker(id=worker_id)
        db.session.add(worker)
    worker.hostname = str(data.get("hostname", worker.hostname or ""))[:120]
    if is_new:
        worker.name = str(data.get("name", "worker"))[:80]
        worker.max_jobs = max(1, int(data.get("max_jobs", 1)))
    worker.last_seen = _now()
    worker.reported_status = "idle"
    worker.current_job_id = None
    db.session.commit()
    return jsonify({"enabled": worker.enabled, "max_jobs": worker.max_jobs})


@workers_bp.post("/worker/heartbeat")
def worker_heartbeat():
    _require_api_key()
    data = request.get_json(silent=True) or {}
    worker_id = str(data.get("worker_id", "")).strip()
    if not worker_id:
        raise BadRequest("worker_id is required")

    worker = db.session.get(Worker, worker_id)
    if worker is None:
        return jsonify({"enabled": False, "known": False})
    worker.last_seen = _now()
    worker.reported_status = str(data.get("status", "idle"))[:20]
    worker.current_job_id = data.get("current_job") or None
    stats = data.get("system_stats")
    if isinstance(stats, dict):
        worker.system_stats = stats
    db.session.commit()
    return jsonify({"enabled": worker.enabled, "known": True, "max_jobs": worker.max_jobs})


@workers_bp.post("/worker/claim")
def claim_shards():
    _require_api_key()
    data = request.get_json(silent=True) or {}
    worker_id = str(data.get("worker_id", "")).strip()
    if not worker_id:
        raise BadRequest("worker_id is required")

    worker = db.session.get(Worker, worker_id)
    if worker is None:
        return jsonify({"shards": [], "enabled": False, "known": False})

    worker.last_seen = _now()
    if not worker.enabled:
        db.session.commit()
        return jsonify({"shards": [], "enabled": False})

    _reap_stale()

    try:
        slots = max(1, min(64, int(data.get("slots", worker.max_jobs))))
    except (TypeError, ValueError):
        slots = worker.max_jobs

    claimed = (
        EvaluationShard.query.filter_by(status="queued")
        .order_by(EvaluationShard.created_at.asc(), EvaluationShard.shard_index.asc())
        .limit(slots)
        .with_for_update(skip_locked=True)
        .all()
    )

    payloads = []
    busy_eval = None
    for shard in claimed:
        evaluation = db.session.get(PlayerEvaluation, shard.evaluation_id)
        if evaluation is None or evaluation.status not in ("queued", "running"):
            shard.status = "cancelled"
            continue
        if evaluation.status == "queued":
            evaluation.status = "running"
            evaluation.progress = 0.0
        shard.status = "running"
        shard.worker_id = worker_id
        shard.done_units = 0
        shard.last_update = _now()
        busy_eval = evaluation.id
        payloads.append(_shard_payload(evaluation, shard))

    if payloads:
        worker.reported_status = "busy"
        worker.current_job_id = busy_eval
    else:
        worker.reported_status = "idle"
    db.session.commit()
    return jsonify({"shards": payloads, "enabled": True, "max_jobs": worker.max_jobs})


@workers_bp.post("/worker/shards/<shard_id>/progress")
def shard_progress(shard_id):
    _require_api_key()
    data = request.get_json(silent=True) or {}
    worker_id = str(data.get("worker_id", "")).strip()
    shard = db.session.get(EvaluationShard, shard_id)
    if shard is None:
        return jsonify({"cancel": True})

    worker = db.session.get(Worker, worker_id) if worker_id else None
    if worker is not None:
        worker.last_seen = _now()

    owns = shard.status == "running" and shard.worker_id == worker_id
    if owns:
        try:
            shard.done_units = max(0, min(int(data.get("done", 0)), shard.total_units))
        except (TypeError, ValueError):
            pass
        shard.last_update = _now()
        stage = data.get("stage")
        if stage:
            evaluation = db.session.get(PlayerEvaluation, shard.evaluation_id)
            if evaluation is not None:
                evaluation.stage = str(stage)[:200]
        _update_progress(shard.evaluation_id)
    db.session.commit()
    return jsonify({"cancel": not owns})


@workers_bp.post("/worker/shards/<shard_id>/result")
def shard_result(shard_id):
    _require_api_key()
    data = request.get_json(silent=True) or {}
    worker_id = str(data.get("worker_id", "")).strip()
    shard = db.session.get(EvaluationShard, shard_id)
    if shard is None:
        raise NotFound("Shard not found")

    worker = db.session.get(Worker, worker_id) if worker_id else None
    if worker is not None:
        worker.last_seen = _now()

    if shard.status == "running" and shard.worker_id == worker_id:
        evaluation = (
            PlayerEvaluation.query.filter_by(id=shard.evaluation_id)
            .with_for_update()
            .first()
        )
        shard.result = data.get("result") or {}
        shard.status = "done"
        shard.done_units = shard.total_units
        shard.last_update = _now()
        if evaluation is not None:
            _finalize_if_complete(shard.evaluation_id)
    db.session.commit()
    return jsonify({"ok": True})


@workers_bp.post("/worker/shards/<shard_id>/fail")
def shard_fail(shard_id):
    _require_api_key()
    data = request.get_json(silent=True) or {}
    worker_id = str(data.get("worker_id", "")).strip()
    shard = db.session.get(EvaluationShard, shard_id)
    if shard is None:
        raise NotFound("Shard not found")

    worker = db.session.get(Worker, worker_id) if worker_id else None
    if worker is not None:
        worker.last_seen = _now()

    if shard.status == "running" and shard.worker_id == worker_id:
        evaluation = (
            PlayerEvaluation.query.filter_by(id=shard.evaluation_id)
            .with_for_update()
            .first()
        )
        shard.status = "failed"
        shard.error = str(data.get("error", "worker error"))[:400]
        shard.last_update = _now()
        if evaluation is not None:
            # Stop the remaining shards of a failed job from running on.
            EvaluationShard.query.filter_by(
                evaluation_id=shard.evaluation_id, status="queued"
            ).update({"status": "cancelled"}, synchronize_session=False)
            _finalize_if_complete(shard.evaluation_id)
    db.session.commit()
    return jsonify({"ok": True})


@workers_bp.get("/workers")
def list_workers():
    require_admin()
    _reap_stale()
    workers = Worker.query.order_by(Worker.created_at.asc()).all()
    return jsonify([worker.to_dict() for worker in workers])


def _requeue_worker_shards(worker_id):
    shards = EvaluationShard.query.filter_by(worker_id=worker_id, status="running").all()
    for shard in shards:
        shard.status = "queued"
        shard.worker_id = None
        shard.done_units = 0
        shard.last_update = _now()


def _worker_jobs(worker_id):
    shards = (
        EvaluationShard.query
        .filter_by(worker_id=worker_id, status="running")
        .order_by(EvaluationShard.last_update.desc())
        .all()
    )
    evaluation_ids = {shard.evaluation_id for shard in shards}
    evaluations = {
        ev.id: ev
        for ev in PlayerEvaluation.query.filter(PlayerEvaluation.id.in_(evaluation_ids)).all()
    } if evaluation_ids else {}
    jobs = []
    for shard in shards:
        evaluation = evaluations.get(shard.evaluation_id)
        jobs.append({
            "shard_id": shard.id,
            "evaluation_id": shard.evaluation_id,
            "shard_index": shard.shard_index,
            "status": shard.status,
            "done_units": shard.done_units,
            "total_units": shard.total_units,
            "username": evaluation.username if evaluation else None,
            "level_id": (evaluation.level_id or "farp") if evaluation else None,
            "last_update": shard.last_update.isoformat() if shard.last_update else None,
        })
    return jobs


@workers_bp.get("/workers/<worker_id>")
def get_worker(worker_id):
    require_admin()
    _reap_stale()
    worker = db.session.get(Worker, worker_id)
    if worker is None:
        raise NotFound("Worker not found")
    payload = worker.to_dict()
    payload["jobs"] = _worker_jobs(worker_id)
    return jsonify(payload)


@workers_bp.post("/workers/<worker_id>/settings")
def update_worker_settings(worker_id):
    require_admin()
    worker = db.session.get(Worker, worker_id)
    if worker is None:
        raise NotFound("Worker not found")
    data = request.get_json(silent=True) or {}
    if "name" in data:
        name = str(data["name"]).strip()[:80]
        if name:
            worker.name = name
    if "max_jobs" in data:
        try:
            value = int(data["max_jobs"])
        except (TypeError, ValueError):
            raise BadRequest("max_jobs must be an integer")
        worker.max_jobs = max(1, min(64, value))
    db.session.commit()
    return jsonify(worker.to_dict())


@workers_bp.post("/workers/<worker_id>/connect")
def connect_worker(worker_id):
    require_admin()
    worker = db.session.get(Worker, worker_id)
    if worker is None:
        raise NotFound("Worker not found")
    worker.enabled = True
    db.session.commit()
    return jsonify(worker.to_dict())


@workers_bp.post("/workers/<worker_id>/disconnect")
def disconnect_worker(worker_id):
    require_admin()
    worker = db.session.get(Worker, worker_id)
    if worker is None:
        raise NotFound("Worker not found")
    worker.enabled = False
    _requeue_worker_shards(worker_id)
    worker.current_job_id = None
    worker.reported_status = "disconnected"
    db.session.commit()
    return jsonify(worker.to_dict())


@workers_bp.delete("/workers/<worker_id>")
def delete_worker(worker_id):
    require_admin()
    worker = db.session.get(Worker, worker_id)
    if worker is None:
        raise NotFound("Worker not found")
    _requeue_worker_shards(worker_id)
    db.session.delete(worker)
    db.session.commit()
    return "", 204
