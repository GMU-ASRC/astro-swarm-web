import gzip
import json
from datetime import datetime, timedelta, timezone

from flask import Blueprint, jsonify, request
from sqlalchemy import func
from werkzeug.exceptions import BadRequest, NotFound, Unauthorized

import merge
import replay_cleanup
import replay_files
from auth import require_admin
from app_settings import (
    ASSAULT_SWEEP_TRIALS,
    assault_sweep_max,
    get_enemy_start,
    get_seed,
    get_sweep_max,
    get_sweep_trials,
    is_assault_level,
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


MAX_CLAIM_SLOTS = 64


def _json_body():
    raw = request.get_data()
    if request.headers.get("Content-Encoding", "").lower() == "gzip":
        try:
            raw = gzip.decompress(raw)
        except OSError:
            raise BadRequest("Invalid gzip body")
    if not raw:
        return {}
    try:
        data = json.loads(raw)
    except ValueError:
        raise BadRequest("Invalid JSON body")
    return data if isinstance(data, dict) else {}


def _requested_slots(data):
    try:
        slots = int(data.get("slots", 1))
    except (TypeError, ValueError):
        slots = 1
    return max(1, min(slots, MAX_CLAIM_SLOTS))


def _running_count(worker_id):
    return PlayerEvaluation.query.filter_by(worker_id=worker_id, status="running").count()


def _running_counts():
    rows = (
        db.session.query(PlayerEvaluation.worker_id, func.count(PlayerEvaluation.id))
        .filter(PlayerEvaluation.status == "running")
        .group_by(PlayerEvaluation.worker_id)
        .all()
    )
    return {worker_id: count for worker_id, count in rows if worker_id}


def _worker_summary(worker, running_jobs):
    payload = worker.to_dict()
    stats = worker.system_stats or {}
    payload["running_jobs"] = running_jobs
    payload["job_slots"] = int(stats.get("job_slots") or 1)
    return payload


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
    if is_assault_level(evaluation.level_id):
        return max(1, trials + assault_sweep_max(evaluation.level_id) * ASSAULT_SWEEP_TRIALS)
    return max(1, trials + get_sweep_max() * get_sweep_trials())


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


def _sweep_params(evaluation):
    # An assault match runs a whole stream of evaders, so the level 1 and 2 sweep
    # budget would put an assault job past the job timeout on a modest worker.
    if is_assault_level(evaluation.level_id):
        return assault_sweep_max(evaluation.level_id), ASSAULT_SWEEP_TRIALS
    return get_sweep_max(), get_sweep_trials()


def _job_payload(evaluation):
    enemy_x, enemy_y = get_enemy_start()
    sweep_max, sweep_trials = _sweep_params(evaluation)
    return {
        "job_id": evaluation.id,
        "evaluation_id": evaluation.id,
        "algorithm": evaluation.algorithm or [],
        "genome_schedule": evaluation.genome_schedule or [],
        "placements": evaluation.placements or [],
        "run": _pending_run(evaluation),
        "trials": evaluation.trials,
        "total_units": evaluation.total_units or 1,
        "config": {
            "seed": get_seed(),
            "sweep_max": sweep_max,
            "sweep_trials": sweep_trials,
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
    current_jobs = data.get("current_jobs")
    if isinstance(current_jobs, list) and current_jobs:
        worker.current_job_id = str(current_jobs[0])[:64]
    else:
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

    claimed = (
        PlayerEvaluation.query.filter_by(status="queued")
        .order_by(PlayerEvaluation.created_at.asc())
        .limit(_requested_slots(data))
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

    db.session.flush()
    if _running_count(worker_id) > 0:
        worker.reported_status = "busy"
        if jobs and not worker.current_job_id:
            worker.current_job_id = jobs[0]["evaluation_id"]
    else:
        worker.reported_status = "idle"
        worker.current_job_id = None
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


@workers_bp.post("/worker/jobs/<job_id>/replays/<upload_id>/<key>")
def job_replay_chunk(job_id, upload_id, key):
    _require_api_key()
    worker_id = request.headers.get("X-Worker-ID", "").strip()
    try:
        offset = int(request.args.get("offset", "0"))
    except ValueError:
        raise BadRequest("offset must be an integer")
    if offset < 0:
        raise BadRequest("offset must not be negative")

    evaluation = db.session.get(PlayerEvaluation, job_id)
    if evaluation is None:
        raise NotFound("Job not found")
    if evaluation.status != "running" or evaluation.worker_id != worker_id:
        return jsonify({"accepted": False, "size": 0, "path": ""})

    try:
        path, size = replay_files.write_chunk(job_id, upload_id, key, offset, request.get_data())
    except replay_files.ReplayFileError as error:
        raise BadRequest(str(error))
    evaluation.last_update = _now()
    db.session.commit()
    return jsonify({"accepted": True, "size": size, "path": path})


def _missing_replay_files(result):
    for run in result.get("runs") or []:
        if run.get("frames_file") and not replay_files.exists(run["frames_file"]):
            return run["frames_file"]
    for sweep in result.get("sweep_runs") or []:
        if sweep.get("frames_file") and not replay_files.exists(sweep["frames_file"]):
            return sweep["frames_file"]
        for trial in sweep.get("trial_runs") or []:
            if trial.get("frames_file") and not replay_files.exists(trial["frames_file"]):
                return trial["frames_file"]
    return None


@workers_bp.post("/worker/jobs/<job_id>/result")
def job_result(job_id):
    _require_api_key()
    data = _json_body()
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
    result = data.get("result") or {}
    if accepted:
        missing = _missing_replay_files(result)
        if missing:
            db.session.rollback()
            raise BadRequest("Replay file %s was never uploaded" % missing)
        _store_result(evaluation, result)
    db.session.commit()
    if accepted:
        upload_id = str(data.get("upload_id") or "")
        replay_cleanup.remove_uploads(replay_cleanup.unused_uploads(job_id, keep_upload_id=upload_id))
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
    counts = _running_counts()
    return jsonify([_worker_summary(worker, counts.get(worker.id, 0)) for worker in workers])


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
    jobs = _worker_jobs(worker_id)
    payload = _worker_summary(worker, len(jobs))
    payload["jobs"] = jobs
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
