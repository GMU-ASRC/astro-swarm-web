import io
import json
import re
import zipfile
from datetime import datetime, timezone

from flask import Blueprint, Response, current_app, jsonify, request, send_file
from sqlalchemy.orm import defer
from werkzeug.exceptions import BadRequest, NotFound, Unauthorized

import charts
from app_settings import JOBS_HARD_CAP, get_enemy_start, get_max_jobs, set_enemy_start, set_max_jobs
from config import Config
from database import db
from evaluator import cancel_evaluation, run_evaluation_async
from models import PlayerEvaluation
from schemas import EvaluationSubmit

evaluations_bp = Blueprint("evaluations", __name__, url_prefix="/api/evaluations")


@evaluations_bp.get("")
def list_evaluations():
    evaluations = (
        PlayerEvaluation.query.options(
            defer(PlayerEvaluation.replays),
            defer(PlayerEvaluation.algorithm),
            defer(PlayerEvaluation.placements),
        )
        .order_by(PlayerEvaluation.created_at.desc())
        .limit(200)
        .all()
    )
    return jsonify([item.to_list_dict() for item in evaluations])


@evaluations_bp.post("")
def submit_evaluation():
    if request.headers.get("X-API-Key") != Config.API_SECRET_KEY:
        raise Unauthorized("Invalid API key")

    data = request.get_json(silent=True)
    if not data:
        raise BadRequest("Invalid JSON data")

    try:
        parsed = EvaluationSubmit(**data)
    except Exception as exc:
        raise BadRequest(str(exc))

    evaluation = PlayerEvaluation(
        player_id=parsed.player_id,
        username=parsed.username,
        level_id=parsed.level_id,
        algorithm=parsed.algorithm,
        placements=parsed.placements,
        trials=parsed.trials,
        status="queued",
    )
    db.session.add(evaluation)
    db.session.commit()

    run_evaluation_async(current_app._get_current_object(), evaluation.id)

    return jsonify(evaluation.to_dict()), 202


@evaluations_bp.get("/settings")
def settings():
    seed = Config.EVAL_SEED
    enemy_x, enemy_y = get_enemy_start()
    return jsonify({
        "seed": seed,
        "placement_trials": 100,
        "sweep_max": Config.EVAL_SWEEP_MAX,
        "sweep_trials": Config.EVAL_SWEEP_TRIALS,
        "spawn_points": Config.EVAL_SPAWN_POINTS,
        "match_cap_seconds": Config.EVAL_MATCH_CAP_SECONDS,
        "max_jobs": get_max_jobs(),
        "max_jobs_cap": JOBS_HARD_CAP,
        "enemy_start_x": enemy_x,
        "enemy_start_y": enemy_y,
        "arena_width": Config.EVAL_ARENA_WIDTH,
        "arena_height": Config.EVAL_ARENA_HEIGHT,
        "planet_x": Config.EVAL_ARENA_WIDTH / 2,
        "planet_y": Config.EVAL_ARENA_HEIGHT / 2,
        "derived_seeds": [
            {"name": "Static enemy spawn locations", "formula": "seed", "value": seed},
            {"name": "Placement match RNG (per trial)", "formula": "seed + trial_index", "value": f"{seed} + trial_index"},
            {"name": "Sweep ring orientations (per n)", "formula": "seed + 100000 + n", "value": f"{seed + 100000} + n"},
            {"name": "Sweep match RNG (per n)", "formula": "seed + 1000000 + n", "value": f"{seed + 1000000} + n"},
        ],
    })


@evaluations_bp.put("/settings")
def update_settings():
    if request.headers.get("X-API-Key") != Config.API_SECRET_KEY:
        raise Unauthorized("Invalid API key")
    data = request.get_json(silent=True) or {}
    if "max_jobs" in data:
        try:
            set_max_jobs(int(data["max_jobs"]))
        except (TypeError, ValueError):
            raise BadRequest("max_jobs must be an integer")
    if "enemy_start_x" in data or "enemy_start_y" in data:
        try:
            set_enemy_start(float(data["enemy_start_x"]), float(data["enemy_start_y"]))
        except (TypeError, ValueError, KeyError):
            raise BadRequest("enemy_start_x and enemy_start_y must both be numbers")
    enemy_x, enemy_y = get_enemy_start()
    return jsonify({
        "max_jobs": get_max_jobs(),
        "max_jobs_cap": JOBS_HARD_CAP,
        "enemy_start_x": enemy_x,
        "enemy_start_y": enemy_y,
    })


@evaluations_bp.get("/baseline")
def baseline():
    evaluations = (
        PlayerEvaluation.query.options(
            defer(PlayerEvaluation.replays),
            defer(PlayerEvaluation.algorithm),
            defer(PlayerEvaluation.placements),
        )
        .filter_by(status="done")
        .all()
    )
    rates = []
    for evaluation in evaluations:
        rate = (evaluation.results or {}).get("success_rate") if isinstance(evaluation.results, dict) else None
        if rate is not None:
            rates.append(rate)
    average = round(sum(rates) / len(rates), 1) if rates else None
    return jsonify({"success_rate": average, "samples": len(rates)})


@evaluations_bp.get("/<eval_id>")
def get_evaluation(eval_id: str):
    evaluation = db.session.get(PlayerEvaluation, eval_id)
    if evaluation is None:
        raise BadRequest("Evaluation not found")
    return jsonify(evaluation.to_dict())


@evaluations_bp.delete("/<eval_id>")
def delete_evaluation(eval_id: str):
    if request.headers.get("X-API-Key") != Config.API_SECRET_KEY:
        raise Unauthorized("Invalid API key")
    evaluation = db.session.get(PlayerEvaluation, eval_id)
    if evaluation is None:
        raise BadRequest("Evaluation not found")
    db.session.delete(evaluation)
    db.session.commit()
    return "", 204


@evaluations_bp.post("/<eval_id>/cancel")
def cancel_evaluation_route(eval_id: str):
    if request.headers.get("X-API-Key") != Config.API_SECRET_KEY:
        raise Unauthorized("Invalid API key")
    evaluation = db.session.get(PlayerEvaluation, eval_id)
    if evaluation is None:
        raise BadRequest("Evaluation not found")
    if evaluation.status not in ("queued", "running"):
        raise BadRequest("Evaluation is not running")

    cancel_evaluation(eval_id)
    evaluation.status = "cancelled"
    evaluation.error = "cancelled"
    evaluation.completed_at = datetime.now(timezone.utc)
    db.session.commit()
    return jsonify(evaluation.to_dict()), 202


@evaluations_bp.post("/<eval_id>/resimulate")
def resimulate_evaluation(eval_id: str):
    if request.headers.get("X-API-Key") != Config.API_SECRET_KEY:
        raise Unauthorized("Invalid API key")
    evaluation = db.session.get(PlayerEvaluation, eval_id)
    if evaluation is None:
        raise BadRequest("Evaluation not found")
    if evaluation.status in ("queued", "running"):
        raise BadRequest("Evaluation is already running")

    evaluation.status = "queued"
    evaluation.progress = 0.0
    evaluation.error = None
    db.session.commit()

    run_evaluation_async(current_app._get_current_object(), evaluation.id)
    return jsonify(evaluation.to_dict()), 202


@evaluations_bp.get("/<eval_id>/chart/<kind>.png")
def chart(eval_id: str, kind: str):
    evaluation = db.session.get(PlayerEvaluation, eval_id)
    if evaluation is None:
        raise NotFound("Evaluation not found")
    results = evaluation.results if isinstance(evaluation.results, dict) else {}
    outcomes = results.get("outcomes", [])
    date_label = (evaluation.completed_at or evaluation.created_at).date().isoformat()
    meta = (evaluation.username, evaluation.level_id or "farp", evaluation.id, date_label)
    if kind == "line":
        png = charts.render_line_png(outcomes, *meta)
    elif kind == "bar":
        png = charts.render_bar_png(outcomes, *meta)
    elif kind == "sweep":
        png = charts.render_sweep_png(results.get("sweep", []), *meta)
    elif kind == "times":
        png = charts.render_times_png(results.get("detection_times", []), results.get("capture_times", []), *meta)
    else:
        raise NotFound("Unknown chart")
    return Response(png, mimetype="image/png")


def safe_filename(value: str) -> str:
    cleaned = re.sub(r"[^A-Za-z0-9_-]+", "_", value or "").strip("_")
    return cleaned or "entry"


@evaluations_bp.get("/<eval_id>/export")
def export_evaluation(eval_id: str):
    evaluation = db.session.get(PlayerEvaluation, eval_id)
    if evaluation is None:
        raise BadRequest("Evaluation not found")

    buffer = io.BytesIO()
    with zipfile.ZipFile(buffer, "w", compression=zipfile.ZIP_DEFLATED) as archive:
        archive.writestr("evaluation.json", json.dumps(evaluation.to_dict(), indent=2))
        for run in evaluation.replay_index().get("runs", []):
            trial = run.get("trial")
            replay = evaluation.replay_for(trial)
            if replay is not None:
                archive.writestr(f"runs/trial_{trial}.json", json.dumps(replay, indent=2))
    buffer.seek(0)

    download_name = f"evaluation_{safe_filename(evaluation.username)}_{evaluation.id}.zip"
    return send_file(
        buffer,
        mimetype="application/zip",
        as_attachment=True,
        download_name=download_name,
    )


@evaluations_bp.get("/<eval_id>/replays")
def list_replays(eval_id: str):
    evaluation = db.session.get(PlayerEvaluation, eval_id)
    if evaluation is None:
        raise BadRequest("Evaluation not found")
    return jsonify(evaluation.replay_index())


@evaluations_bp.get("/<eval_id>/replay/<int:trial>")
def get_replay(eval_id: str, trial: int):
    evaluation = db.session.get(PlayerEvaluation, eval_id)
    if evaluation is None:
        raise BadRequest("Evaluation not found")
    replay = evaluation.replay_for(trial)
    if replay is None:
        raise BadRequest("Replay not found")
    return jsonify(replay)


@evaluations_bp.get("/<eval_id>/sweep-replays")
def list_sweep_replays(eval_id: str):
    evaluation = db.session.get(PlayerEvaluation, eval_id)
    if evaluation is None:
        raise BadRequest("Evaluation not found")
    return jsonify(evaluation.sweep_index())


@evaluations_bp.get("/<eval_id>/sweep-replay/<int:n>")
def get_sweep_replay(eval_id: str, n: int):
    evaluation = db.session.get(PlayerEvaluation, eval_id)
    if evaluation is None:
        raise BadRequest("Evaluation not found")
    replay = evaluation.sweep_replay_for(n)
    if replay is None:
        raise BadRequest("Replay not found")
    return jsonify(replay)
