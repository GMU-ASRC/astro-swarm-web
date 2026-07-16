import io
import json
import random
import re
import zipfile
from datetime import datetime, timezone

from flask import Blueprint, Response, jsonify, request, send_file
from sqlalchemy.orm import defer
from werkzeug.exceptions import BadRequest, Conflict, NotFound, Unauthorized

from auth import require_admin
import charts
import merge
from app_settings import (
    PILOT_LEVELS,
    SWEEP_MATCH_OFFSET,
    SWEEP_SEED_OFFSET,
    SWEEP_SEED_STRIDE,
    get_enemy_start,
    get_level_enabled,
    get_levels,
    get_seed,
    get_sweep_max,
    get_sweep_trial_seeds,
    get_sweep_trials,
    regenerate_seed,
    set_seed,
    is_benchmark_level,
    is_pilot_level,
    level_ids_for,
    set_enemy_start,
    set_level_enabled,
    set_sweep_params,
)
from config import Config
from database import db
from models import EvaluationShard, PlayerEvaluation
from routers.workers import create_shards
from schemas import MAX_RUN_FPS, MAX_RUN_SECONDS, EvaluationSubmit, RunSubmit

MAX_XP_PER_LEVEL_UNIT = 100
PILOT_LEVEL_MAX_XP = 1000

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


def _dedup_key(algorithm, placements):
    return json.dumps(
        {"a": algorithm or [], "p": placements or []},
        sort_keys=True,
        separators=(",", ":"),
    )


def _find_player_duplicate(player_id, level_id, algorithm, placements):
    key = _dedup_key(algorithm, placements)
    candidates = (
        PlayerEvaluation.query.options(defer(PlayerEvaluation.replays))
        .filter(
            PlayerEvaluation.player_id == player_id,
            PlayerEvaluation.level_id.in_(level_ids_for(level_id)),
            PlayerEvaluation.status != "cancelled",
        )
        .all()
    )
    for candidate in candidates:
        if _dedup_key(candidate.algorithm, candidate.placements) == key:
            return candidate
    return None


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
    query = PlayerEvaluation.query.options(
        defer(PlayerEvaluation.replays),
        defer(PlayerEvaluation.algorithm),
        defer(PlayerEvaluation.placements),
    )

    level_id = (request.args.get("level_id") or "").strip()
    if level_id:
        query = query.filter(PlayerEvaluation.level_id.in_(level_ids_for(level_id)))

    if request.args.get("exclude_cancelled") in ("1", "true", "yes"):
        query = query.filter(PlayerEvaluation.status != "cancelled")

    player_id = (request.args.get("player_id") or "").strip()
    if player_id:
        query = query.filter(PlayerEvaluation.player_id == player_id)

    evaluations = query.order_by(PlayerEvaluation.created_at.desc()).limit(500).all()
    return jsonify([item.to_list_dict() for item in evaluations])


@evaluations_bp.get("/best")
def best_evaluation():
    level_id = (request.args.get("level_id") or "farp2").strip()
    candidates = PlayerEvaluation.query.filter(
        PlayerEvaluation.level_id.in_(level_ids_for(level_id)),
        PlayerEvaluation.status == "done",
    ).all()
    scored = []
    for item in candidates:
        results = item.results if isinstance(item.results, dict) else {}
        rate = results.get("success_rate")
        if rate is None:
            continue
        scored.append((item, float(rate)))
    if not scored:
        raise NotFound("No submitted algorithm for this level yet")

    best_rate = max(rate for _, rate in scored)
    leaders = [item for item, rate in scored if rate == best_rate]
    winner = random.choice(leaders)
    return jsonify({
        "id": winner.id,
        "level_id": winner.level_id,
        "username": winner.username,
        "player_id": winner.player_id,
        "success_rate": round(best_rate, 1),
        "algorithm": winner.algorithm or [],
        "placements": winner.placements or [],
    })


def _aggregate_players():
    evaluations = (
        PlayerEvaluation.query.options(
            defer(PlayerEvaluation.replays),
            defer(PlayerEvaluation.algorithm),
            defer(PlayerEvaluation.placements),
        )
        .filter(PlayerEvaluation.status != "cancelled")
        .all()
    )
    players = {}
    for e in evaluations:
        p = players.get(e.player_id)
        if p is None:
            p = {
                "player_id": e.player_id,
                "username": e.username,
                "total_xp": 0,
                "entries": 0,
                "success_sum": 0.0,
                "success_count": 0,
                "levels": {},
                "last": e.created_at,
            }
            players[e.player_id] = p
        if e.created_at is not None and (p["last"] is None or e.created_at >= p["last"]):
            p["username"] = e.username
            p["last"] = e.created_at
        p["total_xp"] += e.xp_awarded or 0
        p["entries"] += 1
        results = e.results if isinstance(e.results, dict) else {}
        rate = results.get("success_rate")
        level_num = e.level_number()
        lv = p["levels"].get(level_num)
        if lv is None:
            lv = {"success_sum": 0.0, "count": 0, "xp": 0}
            p["levels"][level_num] = lv
        lv["xp"] += e.xp_awarded or 0
        if rate is not None and e.status == "done":
            p["success_sum"] += float(rate)
            p["success_count"] += 1
            lv["success_sum"] += float(rate)
            lv["count"] += 1
    for p in players.values():
        p["overall_success"] = (
            round(p["success_sum"] / p["success_count"], 1) if p["success_count"] else None
        )
        for lv in p["levels"].values():
            lv["success_rate"] = round(lv["success_sum"] / lv["count"], 1) if lv["count"] else None
    return players


def _sorted_by_xp(players):
    return sorted(
        players.values(),
        key=lambda p: (p["total_xp"], p["overall_success"] or 0.0),
        reverse=True,
    )


@evaluations_bp.get("/players")
def players_leaderboard():
    players = _aggregate_players()
    rows = _sorted_by_xp(players)
    return jsonify([
        {
            "player_id": p["player_id"],
            "username": p["username"],
            "total_xp": p["total_xp"],
            "entries": p["entries"],
            "overall_success": p["overall_success"],
            "rank": index + 1,
        }
        for index, p in enumerate(rows)
    ])


@evaluations_bp.get("/players/<player_id>")
def player_profile(player_id: str):
    players = _aggregate_players()
    me = players.get(player_id)
    if me is None:
        raise NotFound("Player not found")

    ranked = _sorted_by_xp(players)
    overall_rank = next((i + 1 for i, p in enumerate(ranked) if p["player_id"] == player_id), None)

    levels_out = []
    for level_num in sorted(me["levels"].keys()):
        contenders = [
            p for p in players.values()
            if level_num in p["levels"] and p["levels"][level_num]["success_rate"] is not None
        ]
        contenders.sort(key=lambda p: p["levels"][level_num]["success_rate"], reverse=True)
        rank = next((i + 1 for i, p in enumerate(contenders) if p["player_id"] == player_id), None)
        lv = me["levels"][level_num]
        levels_out.append({
            "level_number": level_num,
            "success_rate": lv["success_rate"],
            "xp": lv["xp"],
            "rank": rank,
            "players": len(contenders),
        })

    entries = (
        PlayerEvaluation.query.options(
            defer(PlayerEvaluation.replays),
            defer(PlayerEvaluation.algorithm),
            defer(PlayerEvaluation.placements),
        )
        .filter(
            PlayerEvaluation.player_id == player_id,
            PlayerEvaluation.status != "cancelled",
        )
        .order_by(PlayerEvaluation.created_at.desc())
        .all()
    )

    return jsonify({
        "player_id": player_id,
        "username": me["username"],
        "total_xp": me["total_xp"],
        "overall_success": me["overall_success"],
        "overall_rank": overall_rank,
        "total_players": len(ranked),
        "entries": me["entries"],
        "levels": levels_out,
        "recent_entries": [e.to_list_dict() for e in entries],
    })


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

    if not is_benchmark_level(parsed.level_id):
        raise BadRequest("Level is not benchmarked")

    if not get_level_enabled(parsed.level_id):
        raise BadRequest("Level is disabled")

    duplicate = _find_player_duplicate(
        parsed.player_id, parsed.level_id, parsed.algorithm, parsed.placements
    )
    if duplicate is not None:
        raise Conflict("You have already submitted this algorithm and placement for this level")

    evaluation = PlayerEvaluation(
        player_id=parsed.player_id,
        username=parsed.username,
        level_id=parsed.level_id,
        algorithm=parsed.algorithm,
        placements=parsed.placements,
        trials=parsed.trials,
        status="queued",
        game_version=parsed.game_version,
        defender_count=len(parsed.placements or []),
        collisions=parsed.collisions,
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


@evaluations_bp.post("/run")
def submit_run():
    require_admin()

    data = request.get_json(silent=True)
    if not data:
        raise BadRequest("Invalid JSON data")

    try:
        parsed = RunSubmit(**data)
    except Exception as exc:
        raise BadRequest(str(exc))

    if not is_pilot_level(parsed.level_id):
        raise BadRequest("Level does not accept piloted runs")

    if not get_level_enabled(parsed.level_id):
        raise BadRequest("Level is disabled")

    evaluation = PlayerEvaluation(
        player_id=parsed.player_id,
        username=parsed.username,
        level_id=parsed.level_id,
        algorithm=parsed.algorithm,
        placements=parsed.placements,
        trials=1,
        status="queued",
        game_version=parsed.game_version,
        defender_count=len(parsed.placements or []),
        replays={"pending_run": parsed.run},
    )
    db.session.add(evaluation)
    db.session.flush()

    db.session.add(EvaluationShard(
        evaluation_id=evaluation.id,
        shard_index=0,
        trial_start=0,
        trial_count=1,
        n_start=1,
        n_count=0,
        total_units=1,
        status="queued",
    ))
    db.session.commit()

    return jsonify(evaluation.to_dict()), 202


def _derived_seeds(seed):
    return [
        {"name": "Static enemy spawn locations", "formula": "seed", "value": seed},
        {"name": "Placement match RNG (per trial)", "formula": "seed + trial_index", "value": f"{seed} + trial_index"},
        {"name": "Sweep ring placement and orientations (per n, trial)", "formula": "sweep_seed[trial] + n", "value": f"({seed} + {SWEEP_SEED_OFFSET} + trial * {SWEEP_SEED_STRIDE}) + n"},
        {"name": "Sweep match RNG (per n, trial)", "formula": "sweep_seed[trial] + 500000 + n", "value": f"({seed} + {SWEEP_SEED_OFFSET} + trial * {SWEEP_SEED_STRIDE}) + {SWEEP_MATCH_OFFSET} + n"},
    ]


@evaluations_bp.get("/settings")
def settings():
    seed = get_seed()
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
        "pilot_level_ids": [level["id"] for level in PILOT_LEVELS],
        "pilot_time_limit_seconds": MAX_RUN_SECONDS,
        "pilot_max_fps": MAX_RUN_FPS,
        "pilot_max_xp": PILOT_LEVEL_MAX_XP,
        "goal_tail_seconds": Config.EVAL_GOAL_TAIL_SECONDS,
        "derived_seeds": _derived_seeds(seed),
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
    if data.get("regenerate_seeds"):
        regenerate_seed()
    elif "seed" in data:
        try:
            set_seed(data["seed"])
        except (TypeError, ValueError):
            raise BadRequest("seed must be an integer")
    if "level_id" in data and "enabled" in data:
        set_level_enabled(str(data["level_id"]), bool(data["enabled"]))
    enemy_x, enemy_y = get_enemy_start()
    return jsonify({
        "seed": get_seed(),
        "enemy_start_x": enemy_x,
        "enemy_start_y": enemy_y,
        "sweep_max": get_sweep_max(),
        "sweep_trials": get_sweep_trials(),
        "sweep_trial_seeds": get_sweep_trial_seeds(),
        "derived_seeds": _derived_seeds(get_seed()),
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


def _success_fraction(evaluation):
    results = evaluation.results if isinstance(evaluation.results, dict) else {}
    rate = results.get("success_rate")
    if rate is None:
        return 0.0
    return max(0.0, min(1.0, float(rate) / 100.0))


@evaluations_bp.post("/<eval_id>/claim-xp")
def claim_xp(eval_id: str):
    require_admin()
    evaluation = db.session.get(PlayerEvaluation, eval_id)
    if evaluation is None:
        raise BadRequest("Evaluation not found")
    if evaluation.status != "done":
        raise BadRequest("Evaluation is not finished yet")

    if is_pilot_level(evaluation.level_id):
        max_xp = PILOT_LEVEL_MAX_XP
    else:
        max_xp = MAX_XP_PER_LEVEL_UNIT * evaluation.level_number()

    if evaluation.xp_awarded is not None:
        return jsonify({
            "xp": evaluation.xp_awarded,
            "already_claimed": True,
            "max_xp": max_xp,
            "success_rate": round(_success_fraction(evaluation) * 100.0, 1),
        })

    others = (
        PlayerEvaluation.query.filter(
            PlayerEvaluation.player_id == evaluation.player_id,
            PlayerEvaluation.status != "cancelled",
            PlayerEvaluation.id != evaluation.id,
        )
        .all()
    )
    level_num = evaluation.level_number()
    prior = [
        e for e in others
        if e.level_number() == level_num and e.xp_awarded is not None
    ]

    rate = _success_fraction(evaluation)
    if not prior:
        xp = round(rate * max_xp)
    else:
        earned = sum(e.xp_awarded or 0 for e in prior)
        best = max((_success_fraction(e) for e in prior), default=0.0)
        if rate > best:
            xp = round((max_xp - earned) * 1.5 * (rate - best))
            xp = max(0, min(xp, max_xp - earned))
        else:
            xp = 0

    evaluation.xp_awarded = xp
    db.session.commit()
    return jsonify({
        "xp": xp,
        "already_claimed": False,
        "max_xp": max_xp,
        "success_rate": round(rate * 100.0, 1),
    })


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


@evaluations_bp.get("/<eval_id>/sweep-replay/<int:n>/trials")
def list_sweep_trial_replays(eval_id: str, n: int):
    evaluation = _get_eval_light(eval_id)
    if evaluation is None:
        raise BadRequest("Evaluation not found")
    return jsonify(evaluation.sweep_trial_index(n))


@evaluations_bp.get("/<eval_id>/sweep-replay/<int:n>/trial/<int:trial>")
def get_sweep_trial_replay(eval_id: str, n: int, trial: int):
    evaluation = _get_eval_light(eval_id)
    if evaluation is None:
        raise BadRequest("Evaluation not found")
    replay = evaluation.sweep_trial_replay_for(n, trial)
    if replay is None:
        raise BadRequest("Replay not found")
    return jsonify(replay)
