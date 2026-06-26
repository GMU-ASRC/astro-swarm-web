from datetime import datetime, timezone

from flask import Blueprint, jsonify, request
from werkzeug.exceptions import BadRequest, NotFound, Unauthorized

from app_settings import get_enemy_start
from config import Config
from database import db
from models import PlayerEvaluation, Worker

workers_bp = Blueprint("workers", __name__, url_prefix="/api")


def _require_api_key():
    if request.headers.get("X-API-Key") != Config.API_SECRET_KEY:
        raise Unauthorized("Invalid API key")


def _now():
    return datetime.now(timezone.utc)


def _reap_stale_workers():
    # Requeue running jobs whose worker has gone offline, and clear the worker's
    # current job so the queue does not stall on a dead node.
    stale = Worker.query.all()
    changed = False
    for worker in stale:
        if worker.is_online():
            continue
        if worker.current_job_id:
            evaluation = db.session.get(PlayerEvaluation, worker.current_job_id)
            if evaluation is not None and evaluation.status == "running":
                evaluation.status = "queued"
                evaluation.progress = 0.0
                evaluation.worker_id = None
            worker.current_job_id = None
            worker.reported_status = "offline"
            changed = True
    if changed:
        db.session.commit()


def _job_payload(evaluation):
    enemy_x, enemy_y = get_enemy_start()
    return {
        "id": evaluation.id,
        "algorithm": evaluation.algorithm or [],
        "placements": evaluation.placements or [],
        "trials": evaluation.trials,
        "config": {
            "seed": Config.EVAL_SEED,
            "sweep_max": Config.EVAL_SWEEP_MAX,
            "sweep_trials": Config.EVAL_SWEEP_TRIALS,
            "match_seconds": Config.EVAL_MATCH_CAP_SECONDS,
            "enemy_x": enemy_x,
            "enemy_y": enemy_y,
        },
    }


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
    worker.name = str(data.get("name", worker.name or "worker"))[:80]
    worker.hostname = str(data.get("hostname", worker.hostname or ""))[:120]
    # max_jobs is owned by the admin panel; only seed it from the worker's own
    # default when the worker first registers, so later overrides survive restarts.
    if is_new:
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
    db.session.commit()
    return jsonify({"enabled": worker.enabled, "known": True, "max_jobs": worker.max_jobs})


@workers_bp.post("/worker/claim")
def claim_job():
    _require_api_key()
    data = request.get_json(silent=True) or {}
    worker_id = str(data.get("worker_id", "")).strip()
    if not worker_id:
        raise BadRequest("worker_id is required")

    worker = db.session.get(Worker, worker_id)
    if worker is None:
        return jsonify({"job": None, "enabled": False, "known": False})

    worker.last_seen = _now()
    if not worker.enabled:
        db.session.commit()
        return jsonify({"job": None, "enabled": False})

    _reap_stale_workers()

    evaluation = (
        PlayerEvaluation.query.filter_by(status="queued")
        .order_by(PlayerEvaluation.created_at.asc())
        .with_for_update(skip_locked=True)
        .first()
    )
    if evaluation is None:
        worker.reported_status = "idle"
        db.session.commit()
        return jsonify({"job": None, "enabled": True})

    evaluation.status = "running"
    evaluation.progress = 0.0
    evaluation.worker_id = worker_id
    evaluation.error = None
    worker.current_job_id = evaluation.id
    worker.reported_status = "busy"
    db.session.commit()
    return jsonify({"job": _job_payload(evaluation), "enabled": True, "max_jobs": worker.max_jobs})


@workers_bp.post("/worker/jobs/<eval_id>/progress")
def worker_progress(eval_id):
    _require_api_key()
    data = request.get_json(silent=True) or {}
    worker_id = str(data.get("worker_id", "")).strip()
    evaluation = db.session.get(PlayerEvaluation, eval_id)
    if evaluation is None:
        raise NotFound("Evaluation not found")

    worker = db.session.get(Worker, worker_id) if worker_id else None
    if worker is not None:
        worker.last_seen = _now()

    # Tell the worker to abort if the job was cancelled, requeued, or handed to a
    # different worker (so it stops wasting compute and cannot clobber the job).
    owns_job = evaluation.status == "running" and evaluation.worker_id == worker_id
    if owns_job:
        try:
            evaluation.progress = round(min(0.99, float(data.get("progress", 0.0))), 3)
        except (TypeError, ValueError):
            pass
    db.session.commit()
    return jsonify({"cancel": not owns_job})


@workers_bp.post("/worker/jobs/<eval_id>/result")
def worker_result(eval_id):
    _require_api_key()
    data = request.get_json(silent=True) or {}
    worker_id = str(data.get("worker_id", "")).strip()
    evaluation = db.session.get(PlayerEvaluation, eval_id)
    if evaluation is None:
        raise NotFound("Evaluation not found")

    # Only accept the result if this worker still owns the running job; otherwise
    # it was cancelled, requeued, or reassigned and we leave the job alone.
    if evaluation.status == "running" and evaluation.worker_id == worker_id:
        evaluation.results = data.get("results", {})
        evaluation.replays = data.get("replays", {})
        evaluation.status = "done"
        evaluation.progress = 1.0
        evaluation.error = None
        evaluation.completed_at = _now()
        evaluation.worker_id = None

    worker = db.session.get(Worker, worker_id) if worker_id else None
    if worker is not None:
        worker.last_seen = _now()
        if worker.current_job_id == eval_id:
            worker.current_job_id = None
        worker.reported_status = "idle"
    db.session.commit()
    return jsonify({"ok": True})


@workers_bp.post("/worker/jobs/<eval_id>/fail")
def worker_fail(eval_id):
    _require_api_key()
    data = request.get_json(silent=True) or {}
    worker_id = str(data.get("worker_id", "")).strip()
    evaluation = db.session.get(PlayerEvaluation, eval_id)
    if evaluation is None:
        raise NotFound("Evaluation not found")

    # Only fail the job if this worker still owns the running job; if it was
    # cancelled, requeued, or reassigned we leave it for the new owner.
    if evaluation.status == "running" and evaluation.worker_id == worker_id:
        evaluation.status = "failed"
        evaluation.error = str(data.get("error", "worker error"))[:400]
        evaluation.completed_at = _now()
        evaluation.worker_id = None

    worker = db.session.get(Worker, worker_id) if worker_id else None
    if worker is not None:
        worker.last_seen = _now()
        if worker.current_job_id == eval_id:
            worker.current_job_id = None
        worker.reported_status = "idle"
    db.session.commit()
    return jsonify({"ok": True})


@workers_bp.get("/workers")
def list_workers():
    _require_api_key()
    _reap_stale_workers()
    workers = Worker.query.order_by(Worker.created_at.asc()).all()
    return jsonify([worker.to_dict() for worker in workers])


def _requeue_worker_job(worker):
    if not worker.current_job_id:
        return
    evaluation = db.session.get(PlayerEvaluation, worker.current_job_id)
    if evaluation is not None and evaluation.status == "running":
        evaluation.status = "queued"
        evaluation.progress = 0.0
        evaluation.worker_id = None
    worker.current_job_id = None


@workers_bp.post("/workers/<worker_id>/max-jobs")
def set_worker_max_jobs(worker_id):
    _require_api_key()
    worker = db.session.get(Worker, worker_id)
    if worker is None:
        raise NotFound("Worker not found")
    data = request.get_json(silent=True) or {}
    try:
        value = int(data.get("max_jobs"))
    except (TypeError, ValueError):
        raise BadRequest("max_jobs must be an integer")
    worker.max_jobs = max(1, min(64, value))
    db.session.commit()
    return jsonify(worker.to_dict())


@workers_bp.post("/workers/<worker_id>/connect")
def connect_worker(worker_id):
    _require_api_key()
    worker = db.session.get(Worker, worker_id)
    if worker is None:
        raise NotFound("Worker not found")
    worker.enabled = True
    db.session.commit()
    return jsonify(worker.to_dict())


@workers_bp.post("/workers/<worker_id>/disconnect")
def disconnect_worker(worker_id):
    _require_api_key()
    worker = db.session.get(Worker, worker_id)
    if worker is None:
        raise NotFound("Worker not found")
    worker.enabled = False
    _requeue_worker_job(worker)
    worker.reported_status = "disconnected"
    db.session.commit()
    return jsonify(worker.to_dict())


@workers_bp.delete("/workers/<worker_id>")
def delete_worker(worker_id):
    _require_api_key()
    worker = db.session.get(Worker, worker_id)
    if worker is None:
        raise NotFound("Worker not found")
    _requeue_worker_job(worker)
    db.session.delete(worker)
    db.session.commit()
    return "", 204
