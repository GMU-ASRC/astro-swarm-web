from datetime import datetime, timedelta, timezone

from flask import Blueprint, jsonify, request
from werkzeug.exceptions import BadRequest, NotFound, Unauthorized

import merge
from auth import require_admin
from app_settings import (
    get_enemy_start,
    get_seed,
    get_sweep_max,
    get_sweep_trials,
    is_wave_level,
)
from config import Config
from database import db
from models import JOB_SILENT_SECONDS, PlayerEvaluation, Worker

workers_bp = Blueprint("workers", __name__, url_prefix="/api")


def _require_api_key():
    if request.headers.get("X-API-Key") != Config.API_SECRET_KEY:
        raise Unauthorized("Invalid API key")


def _now():
    return datetime.now(timezone.utc)


def queue_evaluation(evaluation):
    # An evaluation is one unit of work: a single worker runs the whole thing and
    # parallelises the matches internally.
    evaluation.status = "queued"
    evaluation.progress = 0.0
    evaluation.stage = None
    evaluation.error = None
    evaluation.worker_id = None
    evaluation.done_units = 0
    evaluation.total_units = _total_units(evaluation)
    evaluation.last_update = _now()


def _total_units(evaluation):
    # A piloted entry carries its own recording, so it is one render job rather
    # than a benchmark.
    if _pending_run(evaluation) is not None:
        return 1
    trials = int(evaluation.trials or 0)
    sweep = get_sweep_max() * get_sweep_trials()
    if is_wave_level(evaluation.level_id):
        return max(1, trials * 2 + sweep)
    return max(1, trials + sweep)


def _reap_stale():
    # Requeue jobs that have gone silent, so the queue does not stall on a dead
    # node or on a worker that finished but could never deliver its result. A live
    # job reports progress the whole time it runs, uploads included, so a stale
    # last_update means the job is gone even when its worker still pings.
    cutoff = _now() - timedelta(seconds=JOB_SILENT_SECONDS)
    stale = (
        PlayerEvaluation.query.filter_by(status="running")
        .filter(PlayerEvaluation.last_update < cutoff)
        .all()
    )
    changed = False
    for evaluation in stale:
        evaluation.status = "queued"
        evaluation.worker_id = None
        evaluation.done_units = 0
        evaluation.progress = 0.0
        evaluation.last_update = _now()
        changed = True
    for worker in Worker.query.all():
        if not worker.is_online() and worker.reported_status != "offline":
            worker.current_job_id = None
            worker.reported_status = "offline"
            changed = True
    if changed:
        db.session.commit()


def _pending_run(evaluation):
    replays = evaluation.replays or {}
    if isinstance(replays, dict):
        return replays.get("pending_run")
    return None


def _keep_pending_run(evaluation, replays):
    # The recorded run is the only copy of a piloted flight, so it has to survive
    # every result write. Dropping it would leave the entry impossible to render
    # again.
    run = _pending_run(evaluation)
    if run is not None:
        replays = dict(replays)
        replays["pending_run"] = run
    return replays


def _job_payload(evaluation):
    enemy_x, enemy_y = get_enemy_start()
    return {
        "job_id": evaluation.id,
        "evaluation_id": evaluation.id,
        "algorithm": evaluation.algorithm or [],
        "placements": evaluation.placements or [],
        "run": _pending_run(evaluation),
        "trials": evaluation.trials,
        "total_units": evaluation.total_units or 1,
        "config": {
            "seed": get_seed(),
            "sweep_max": get_sweep_max(),
            "sweep_trials": get_sweep_trials(),
            "match_seconds": Config.EVAL_MATCH_CAP_SECONDS,
            "goal_tail_seconds": Config.EVAL_GOAL_TAIL_SECONDS,
            "enemy_x": enemy_x,
            "enemy_y": enemy_y,
            "sweep_spawn": Config.EVAL_SWEEP_SPAWN,
            "level_id": evaluation.level_id or "farp",
            "collisions": bool(evaluation.collisions),
        },
    }


def _annotate_rates(results, evaluation):
    if not isinstance(results, dict):
        return
    rate = results.get("success_rate")
    if rate is None:
        return
    rate = float(rate)
    if evaluation.is_attack_level():
        results["attacker_rate"] = round(rate, 1)
        results["defender_rate"] = round(100.0 - rate, 1)
    else:
        results["defender_rate"] = round(rate, 1)
        results["attacker_rate"] = round(100.0 - rate, 1)


def _store_result(evaluation, payload):
    results, replays = merge.summarize(payload)
    _annotate_rates(results, evaluation)
    evaluation.results = results
    evaluation.replays = _keep_pending_run(evaluation, replays)
    evaluation.status = "done"
    evaluation.progress = 1.0
    evaluation.done_units = evaluation.total_units or 1
    evaluation.error = None
    evaluation.completed_at = _now()
    evaluation.worker_id = None
    evaluation.stage = None
    evaluation.last_update = _now()


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
    worker.last_seen = _now()
    worker.reported_status = "idle"
    worker.current_job_id = None
    db.session.commit()
    return jsonify({"enabled": worker.enabled})


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
    return jsonify({"enabled": worker.enabled, "known": True})


@workers_bp.post("/worker/claim")
def claim_jobs():
    _require_api_key()
    data = request.get_json(silent=True) or {}
    worker_id = str(data.get("worker_id", "")).strip()
    if not worker_id:
        raise BadRequest("worker_id is required")

    worker = db.session.get(Worker, worker_id)
    if worker is None:
        return jsonify({"jobs": [], "enabled": False, "known": False})

    worker.last_seen = _now()
    if not worker.enabled:
        db.session.commit()
        return jsonify({"jobs": [], "enabled": False})

    _reap_stale()

    # One worker runs one whole evaluation at a time.
    claimed = (
        PlayerEvaluation.query.filter_by(status="queued")
        .order_by(PlayerEvaluation.created_at.asc())
        .limit(1)
        .with_for_update(skip_locked=True)
        .all()
    )

    jobs = []
    for evaluation in claimed:
        evaluation.status = "running"
        evaluation.progress = 0.0
        evaluation.worker_id = worker_id
        evaluation.done_units = 0
        evaluation.last_update = _now()
        if not evaluation.total_units:
            evaluation.total_units = _total_units(evaluation)
        jobs.append(_job_payload(evaluation))

    if jobs:
        worker.reported_status = "busy"
        worker.current_job_id = jobs[0]["evaluation_id"]
    else:
        worker.reported_status = "idle"
    db.session.commit()
    return jsonify({"jobs": jobs, "enabled": True})


@workers_bp.post("/worker/jobs/<job_id>/progress")
def job_progress(job_id):
    _require_api_key()
    data = request.get_json(silent=True) or {}
    worker_id = str(data.get("worker_id", "")).strip()
    evaluation = db.session.get(PlayerEvaluation, job_id)
    if evaluation is None:
        return jsonify({"cancel": True})

    worker = db.session.get(Worker, worker_id) if worker_id else None
    if worker is not None:
        worker.last_seen = _now()

    owns = evaluation.status == "running" and evaluation.worker_id == worker_id
    if owns:
        total = max(1, evaluation.total_units or 1)
        try:
            evaluation.done_units = max(0, min(int(data.get("done", 0)), total))
        except (TypeError, ValueError):
            pass
        evaluation.progress = round(min(0.99, evaluation.done_units / total), 3)
        evaluation.last_update = _now()
        stage = data.get("stage")
        if stage:
            evaluation.stage = str(stage)[:200]
    db.session.commit()
    return jsonify({"cancel": not owns})


@workers_bp.post("/worker/jobs/<job_id>/result")
def job_result(job_id):
    _require_api_key()
    data = request.get_json(silent=True) or {}
    worker_id = str(data.get("worker_id", "")).strip()

    worker = db.session.get(Worker, worker_id) if worker_id else None
    if worker is not None:
        worker.last_seen = _now()

    evaluation = (
        PlayerEvaluation.query.filter_by(id=job_id)
        .with_for_update()
        .first()
    )
    if evaluation is None:
        raise NotFound("Job not found")

    accepted = evaluation.status == "running" and evaluation.worker_id == worker_id
    if accepted:
        _store_result(evaluation, data.get("result") or {})
    db.session.commit()
    # A result for a job this worker no longer owns is dropped. Say so, so the
    # worker logs it instead of treating the job as finished.
    return jsonify({"ok": True, "accepted": accepted})


@workers_bp.post("/worker/jobs/<job_id>/fail")
def job_fail(job_id):
    _require_api_key()
    data = request.get_json(silent=True) or {}
    worker_id = str(data.get("worker_id", "")).strip()

    worker = db.session.get(Worker, worker_id) if worker_id else None
    if worker is not None:
        worker.last_seen = _now()

    evaluation = (
        PlayerEvaluation.query.filter_by(id=job_id)
        .with_for_update()
        .first()
    )
    if evaluation is None:
        raise NotFound("Job not found")

    if evaluation.status == "running" and evaluation.worker_id == worker_id:
        evaluation.status = "failed"
        evaluation.error = str(data.get("error", "worker error"))[:400]
        evaluation.completed_at = _now()
        evaluation.worker_id = None
        evaluation.stage = None
        evaluation.last_update = _now()
    db.session.commit()
    return jsonify({"ok": True})


@workers_bp.get("/workers")
def list_workers():
    require_admin()
    _reap_stale()
    workers = Worker.query.order_by(Worker.created_at.asc()).all()
    return jsonify([worker.to_dict() for worker in workers])


def _requeue_worker_jobs(worker_id):
    running = PlayerEvaluation.query.filter_by(worker_id=worker_id, status="running").all()
    for evaluation in running:
        evaluation.status = "queued"
        evaluation.worker_id = None
        evaluation.done_units = 0
        evaluation.progress = 0.0
        evaluation.last_update = _now()


def _worker_jobs(worker_id):
    running = (
        PlayerEvaluation.query
        .filter_by(worker_id=worker_id, status="running")
        .order_by(PlayerEvaluation.last_update.desc())
        .all()
    )
    return [
        {
            "evaluation_id": evaluation.id,
            "status": evaluation.status,
            "done_units": evaluation.done_units or 0,
            "total_units": evaluation.total_units or 1,
            "username": evaluation.username,
            "level_id": evaluation.level_id or "farp",
            "stage": evaluation.stage,
            "last_update": evaluation.last_update.isoformat() if evaluation.last_update else None,
        }
        for evaluation in running
    ]


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
    _requeue_worker_jobs(worker_id)
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
    _requeue_worker_jobs(worker_id)
    db.session.delete(worker)
    db.session.commit()
    return jsonify({"ok": True})
