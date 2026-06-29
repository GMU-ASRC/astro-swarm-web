import io
import json
import re
import zipfile
from datetime import datetime, timezone

from flask import Blueprint, Response, jsonify, request, send_file
from sqlalchemy.orm import defer
from werkzeug.exceptions import BadRequest, NotFound, Unauthorized

from auth import require_admin
import charts
from app_settings import (
    get_enemy_start,
    get_level_enabled,
    get_levels,
    get_sweep_max,
    get_sweep_trial_seeds,
    get_sweep_trials,
    set_enemy_start,
    set_level_enabled,
    set_sweep_params,
)
from config import Config
from database import db
from models import EvaluationShard, PlayerEvaluation
from routers.workers import create_shards
from schemas import EvaluationSubmit

evaluations_bp = Blueprint("evaluations", __name__, url_prefix="/api/evaluations")


def _get_eval_light(eval_id: str):
    return (
        PlayerEvaluation.query.options(defer(PlayerEvaluation.replays))
        .filter_by(id=eval_id)
        .first()
    )


def _submission_key(algorithm, placements, trials):
    return json.dumps(
        {"a": algorithm or [], "p": placements or [], "t": trials},
        sort_keys=True,
        separators=(",", ":"),
    )


def _find_matching_done(level_id, algorithm, placements, trials):
    key = _submission_key(algorithm, placements, trials)
    sweep_max = get_sweep_max()
    candidates = (
        PlayerEvaluation.query.options(defer(PlayerEvaluation.replays))
        .filter_by(status="done", level_id=level_id)
        .all()
    )
    for candidate in candidates:
        if _submission_key(candidate.algorithm, candidate.placements, candidate.trials) != key:
            continue
        # Only reuse a result that was produced under the current ring-sweep size,
        # so changing n/n2 in settings forces a fresh run instead of stale reuse.
        results = candidate.results if isinstance(candidate.results, dict) else {}
        if len(results.get("sweep", [])) == sweep_max:
            return candidate
    return None


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
    require_admin()

    data = request.get_json(silent=True)
    if not data:
        raise BadRequest("Invalid JSON data")

    try:
        parsed = EvaluationSubmit(**data)
    except Exception as exc:
        raise BadRequest(str(exc))

    if not get_level_enabled(parsed.level_id):
        raise BadRequest("Level is disabled")

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

    existing = _find_matching_done(parsed.level_id, parsed.algorithm, parsed.placements, parsed.trials)
    if existing is not None:
        evaluation.results = existing.results
        evaluation.replays = existing.replays
        evaluation.progress = 1.0
        evaluation.status = "done"
        evaluation.completed_at = datetime.now(timezone.utc)
        db.session.commit()
        return jsonify(evaluation.to_dict()), 202

    db.session.flush()
    create_shards(evaluation)
    db.session.commit()

    return jsonify(evaluation.to_dict()), 202


@evaluations_bp.get("/settings")
def settings():
    seed = Config.EVAL_SEED
    enemy_x, enemy_y = get_enemy_start()
    return jsonify({
        "seed": seed,
        "placement_trials": 100,
        "sweep_max": get_sweep_max(),
        "sweep_trials": get_sweep_trials(),
        "match_cap_seconds": Config.EVAL_MATCH_CAP_SECONDS,
        "enemy_start_x": enemy_x,
        "enemy_start_y": enemy_y,
        "arena_width": Config.EVAL_ARENA_WIDTH,
        "arena_height": Config.EVAL_ARENA_HEIGHT,
        "planet_x": Config.EVAL_ARENA_WIDTH / 2,
        "planet_y": Config.EVAL_ARENA_HEIGHT / 2,
        "sweep_trial_seeds": get_sweep_trial_seeds(),
        "levels": get_levels(),
        "derived_seeds": [
            {"name": "Static enemy spawn locations", "formula": "seed", "value": seed},
            {"name": "Placement match RNG (per trial)", "formula": "seed + trial_index", "value": f"{seed} + trial_index"},
            {"name": "Sweep ring orientations (per n, trial)", "formula": "sweep_seed[trial] + n", "value": f"({seed} + 100000 + trial * 1000000) + n"},
            {"name": "Sweep match RNG (per n, trial)", "formula": "sweep_seed[trial] + 500000 + n", "value": f"({seed} + 100000 + trial * 1000000) + 500000 + n"},
        ],
    })


@evaluations_bp.put("/settings")
def update_settings():
    require_admin()
    data = request.get_json(silent=True) or {}
    if "enemy_start_x" in data or "enemy_start_y" in data:
        try:
            set_enemy_start(float(data["enemy_start_x"]), float(data["enemy_start_y"]))
        except (TypeError, ValueError, KeyError):
            raise BadRequest("enemy_start_x and enemy_start_y must both be numbers")
    if "sweep_max" in data or "sweep_trials" in data:
        try:
            set_sweep_params(
                sweep_max=data.get("sweep_max"),
                sweep_trials=data.get("sweep_trials"),
            )
        except (TypeError, ValueError):
            raise BadRequest("sweep_max and sweep_trials must be integers")
    if "level_id" in data and "enabled" in data:
        set_level_enabled(str(data["level_id"]), bool(data["enabled"]))
    enemy_x, enemy_y = get_enemy_start()
    return jsonify({
        "enemy_start_x": enemy_x,
        "enemy_start_y": enemy_y,
        "sweep_max": get_sweep_max(),
        "sweep_trials": get_sweep_trials(),
        "sweep_trial_seeds": get_sweep_trial_seeds(),
        "levels": get_levels(),
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
    evaluation = _get_eval_light(eval_id)
    if evaluation is None:
        raise BadRequest("Evaluation not found")
    return jsonify(evaluation.to_dict())


@evaluations_bp.delete("/<eval_id>")
def delete_evaluation(eval_id: str):
    require_admin()
    evaluation = db.session.get(PlayerEvaluation, eval_id)
    if evaluation is None:
        raise BadRequest("Evaluation not found")
    db.session.delete(evaluation)
    db.session.commit()
    return "", 204


@evaluations_bp.post("/<eval_id>/cancel")
def cancel_evaluation_route(eval_id: str):
    require_admin()
    evaluation = db.session.get(PlayerEvaluation, eval_id)
    if evaluation is None:
        raise BadRequest("Evaluation not found")
    if evaluation.status not in ("queued", "running"):
        raise BadRequest("Evaluation is not running")

    evaluation.status = "cancelled"
    evaluation.error = "cancelled"
    evaluation.worker_id = None
    evaluation.stage = None
    evaluation.completed_at = datetime.now(timezone.utc)
    EvaluationShard.query.filter_by(evaluation_id=eval_id).filter(
        EvaluationShard.status.in_(("queued", "running"))
    ).update({"status": "cancelled"}, synchronize_session=False)
    db.session.commit()
    return jsonify(evaluation.to_dict()), 202


@evaluations_bp.post("/<eval_id>/resimulate")
def resimulate_evaluation(eval_id: str):
    require_admin()
    evaluation = db.session.get(PlayerEvaluation, eval_id)
    if evaluation is None:
        raise BadRequest("Evaluation not found")
    if evaluation.status in ("queued", "running"):
        raise BadRequest("Evaluation is already running")

    evaluation.status = "queued"
    evaluation.progress = 0.0
    evaluation.error = None
    evaluation.worker_id = None
    create_shards(evaluation)
    db.session.commit()

    return jsonify(evaluation.to_dict()), 202


@evaluations_bp.get("/<eval_id>/chart/<kind>.png")
def chart(eval_id: str, kind: str):
    evaluation = _get_eval_light(eval_id)
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
        png = charts.render_detection_rate_png(evaluation.sweep_index(), *meta)
    elif kind == "capture":
        png = charts.render_capture_rate_png(evaluation.sweep_index(), *meta)
    elif kind == "times":
        png = charts.render_times_png(results.get("detection_times", []), results.get("capture_times", []), *meta)
    else:
        raise NotFound("Unknown chart")
    return Response(png, mimetype="image/png", headers={"Content-Disposition": f'attachment; filename="{kind}_{evaluation.id}.png"'})


@evaluations_bp.get("/<eval_id>/thumbnail.png")
def thumbnail(eval_id: str):
    evaluation = _get_eval_light(eval_id)
    if evaluation is None:
        raise NotFound("Evaluation not found")
    results = evaluation.results if isinstance(evaluation.results, dict) else {}
    outcomes = results.get("outcomes", [])
    rate = results.get("success_rate", 0)
    png = charts.render_thumbnail_png(evaluation.username, evaluation.level_id or "farp", rate, evaluation.trials, outcomes)
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
    evaluation = _get_eval_light(eval_id)
    if evaluation is None:
        raise BadRequest("Evaluation not found")
    return jsonify(evaluation.replay_index())


@evaluations_bp.get("/<eval_id>/replay/<int:trial>")
def get_replay(eval_id: str, trial: int):
    evaluation = _get_eval_light(eval_id)
    if evaluation is None:
        raise BadRequest("Evaluation not found")
    replay = evaluation.replay_for(trial)
    if replay is None:
        raise BadRequest("Replay not found")
    return jsonify(replay)


@evaluations_bp.get("/<eval_id>/sweep-replays")
def list_sweep_replays(eval_id: str):
    evaluation = _get_eval_light(eval_id)
    if evaluation is None:
        raise BadRequest("Evaluation not found")
    return jsonify(evaluation.sweep_index())


@evaluations_bp.get("/<eval_id>/sweep-replay/<int:n>")
def get_sweep_replay(eval_id: str, n: int):
    evaluation = _get_eval_light(eval_id)
    if evaluation is None:
        raise BadRequest("Evaluation not found")
    replay = evaluation.sweep_replay_for(n)
    if replay is None:
        raise BadRequest("Replay not found")
    return jsonify(replay)
