from flask import Blueprint, request, jsonify
from werkzeug.exceptions import Unauthorized, BadRequest

from database import db
from models import LeaderboardEntry
from config import Config
from schemas import LeaderboardSubmit

leaderboard_bp = Blueprint("leaderboard", __name__, url_prefix="/api/leaderboard")

@leaderboard_bp.route("", methods=["GET"])
def get_leaderboard():
    entries = LeaderboardEntry.query.order_by(LeaderboardEntry.time_seconds.asc()).limit(100).all()
    return jsonify([entry.to_dict() for entry in entries]), 200

@leaderboard_bp.route("", methods=["POST"])
def submit_leaderboard():
    api_key = request.headers.get("X-API-Key")
    if api_key != Config.API_SECRET_KEY:
        raise Unauthorized("Invalid API key")

    data = request.get_json()
    if not data:
        raise BadRequest("Invalid JSON data")

    try:
        parsed_data = LeaderboardSubmit(**data)
    except Exception as e:
        raise BadRequest(str(e))

    entry = LeaderboardEntry.query.filter_by(player_id=parsed_data.player_id).first()
    
    if entry:
        if parsed_data.time_seconds < entry.time_seconds:
            entry.time_seconds = parsed_data.time_seconds
            entry.username = parsed_data.username
            entry.algorithm = parsed_data.algorithm
            db.session.commit()
            return jsonify(entry.to_dict()), 200
        else:
            return jsonify(entry.to_dict()), 200
    else:
        entry = LeaderboardEntry(
            player_id=parsed_data.player_id,
            username=parsed_data.username,
            time_seconds=parsed_data.time_seconds,
            algorithm=parsed_data.algorithm
        )
        db.session.add(entry)
        db.session.commit()

        return jsonify(entry.to_dict()), 201

@leaderboard_bp.route("/<entry_id>", methods=["GET"])
def get_leaderboard_entry(entry_id):
    entry = LeaderboardEntry.query.get(entry_id)
    if not entry:
        raise BadRequest("Entry not found")
    return jsonify(entry.to_dict()), 200
