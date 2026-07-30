from flask import Blueprint, jsonify, request
from werkzeug.exceptions import BadRequest, NotFound

from auth import require_admin
from database import db
from models import SurviveMatch
from schemas import SurviveMatchSubmit

MAX_LIST_LIMIT = 200

survive_bp = Blueprint("survive", __name__, url_prefix="/api/survive")


@survive_bp.get("/matches")
def list_matches():
    limit = min(MAX_LIST_LIMIT, max(1, int(request.args.get("limit", 100))))
    query = SurviveMatch.query
    player_id = request.args.get("player_id")
    if player_id:
        query = query.filter_by(player_id=player_id)
    matches = query.order_by(SurviveMatch.created_at.desc()).limit(limit).all()
    return jsonify([match.to_list_dict() for match in matches]), 200


@survive_bp.get("/matches/<match_id>")
def get_match(match_id: str):
    match = db.session.get(SurviveMatch, match_id)
    if match is None:
        raise NotFound("Match not found")
    return jsonify(match.to_dict()), 200


@survive_bp.post("/matches")
def submit_match():
    require_admin()

    data = request.get_json(silent=True)
    if not data:
        raise BadRequest("Invalid JSON data")

    try:
        parsed = SurviveMatchSubmit(**data)
    except TypeError as exc:
        raise BadRequest(str(exc))
    except ValueError as exc:
        raise BadRequest(str(exc))

    match = SurviveMatch(
        player_id=parsed.player_id,
        username=parsed.username,
        game_version=parsed.game_version,
        outcome=parsed.outcome,
        duration=parsed.duration,
        players=parsed.players,
        apm_bucket_seconds=parsed.apm_bucket_seconds,
        apm_times=parsed.apm_times,
        apm_player1=parsed.apm_player1,
        apm_player2=parsed.apm_player2,
        evaders_spawned=parsed.evaders_spawned,
        wild_remaining=parsed.wild_remaining,
        first_target=parsed.first_target,
    )
    db.session.add(match)
    db.session.commit()

    return jsonify(match.to_dict()), 201


@survive_bp.delete("/matches/<match_id>")
def delete_match(match_id: str):
    require_admin()
    match = db.session.get(SurviveMatch, match_id)
    if match is None:
        raise NotFound("Match not found")
    db.session.delete(match)
    db.session.commit()
    return jsonify({"deleted": match_id}), 200
