from datetime import datetime, timezone

from flask import Blueprint, jsonify, request
from werkzeug.exceptions import BadRequest

from auth import require_admin
from database import db
from models import TrainingScore
from schemas import TrainingScoreSubmit

MAX_LEADERBOARD_LIMIT = 200
DEFAULT_LEADERBOARD_LIMIT = 100

training_bp = Blueprint("training", __name__, url_prefix="/api/training")


@training_bp.get("/leaderboard")
def get_leaderboard():
    try:
        limit = max(1, min(MAX_LEADERBOARD_LIMIT, int(request.args.get("limit", DEFAULT_LEADERBOARD_LIMIT))))
    except (TypeError, ValueError):
        limit = DEFAULT_LEADERBOARD_LIMIT
    scores = (
        TrainingScore.query.order_by(TrainingScore.circliness.desc(), TrainingScore.updated_at.asc())
        .limit(limit)
        .all()
    )
    return jsonify([score.to_dict(rank=index + 1) for index, score in enumerate(scores)]), 200


@training_bp.post("/scores")
def submit_score():
    require_admin()

    data = request.get_json(silent=True)
    if not data:
        raise BadRequest("Invalid JSON data")

    try:
        parsed = TrainingScoreSubmit(**data)
    except (TypeError, ValueError) as exc:
        raise BadRequest(str(exc))

    score = TrainingScore.query.filter_by(player_id=parsed.player_id).first()
    if score is None:
        score = TrainingScore(player_id=parsed.player_id, circliness=parsed.circliness)
        db.session.add(score)
        status = 201
    elif parsed.circliness <= score.circliness:
        score.username = parsed.username
        db.session.commit()
        return jsonify(score.to_dict()), 200
    else:
        status = 200

    score.username = parsed.username
    score.game_version = parsed.game_version
    score.circliness = parsed.circliness
    score.defenders = len(parsed.placements)
    score.algorithm = parsed.algorithm
    score.placements = parsed.placements
    score.updated_at = datetime.now(timezone.utc)
    db.session.commit()
    return jsonify(score.to_dict()), status
