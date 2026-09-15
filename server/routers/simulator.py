from flask import Blueprint, jsonify, request
from werkzeug.exceptions import BadRequest, NotFound

from auth import require_admin
from database import db
from simulator_models import SimulatorEntry, pack_replay
from simulator_schema import SimulatorEntrySubmit

MAX_LIST_LIMIT = 200

simulator_bp = Blueprint("simulator", __name__, url_prefix="/api/simulator")


def _entry_or_404(entry_id):
    entry = db.session.get(SimulatorEntry, entry_id)
    if entry is None:
        raise NotFound("Simulator entry not found")
    return entry


@simulator_bp.get("/entries")
def list_entries():
    limit = min(MAX_LIST_LIMIT, max(1, request.args.get("limit", 100, type=int)))
    query = SimulatorEntry.query
    player_id = request.args.get("player_id")
    if player_id:
        query = query.filter_by(player_id=player_id)
    entries = query.order_by(SimulatorEntry.created_at.desc()).limit(limit).all()
    return jsonify([entry.to_list_dict() for entry in entries]), 200


@simulator_bp.get("/entries/<entry_id>")
def get_entry(entry_id: str):
    return jsonify(_entry_or_404(entry_id).to_dict()), 200


@simulator_bp.get("/entries/<entry_id>/replay")
def get_entry_replay(entry_id: str):
    return jsonify(_entry_or_404(entry_id).replay_dict()), 200


@simulator_bp.post("/entries")
def submit_entry():
    require_admin()

    data = request.get_json(silent=True)
    if not isinstance(data, dict):
        raise BadRequest("Invalid JSON data")

    try:
        parsed = SimulatorEntrySubmit(**data)
    except (TypeError, ValueError) as exc:
        raise BadRequest(str(exc))

    entry = SimulatorEntry(
        player_id=parsed.player_id,
        username=parsed.username,
        title=parsed.title,
        description=parsed.description,
        game_version=parsed.game_version,
        arena_width=parsed.arena_width,
        arena_height=parsed.arena_height,
        species=parsed.species,
        behaviors=parsed.behaviors,
        arena_program=parsed.arena_program,
        variables=parsed.variables,
        obstacles=parsed.obstacles,
        spawn_zones=parsed.spawn_zones,
        placement_count=parsed.placement_count,
        peak_robot_count=parsed.peak_robot_count,
        frame_count=parsed.frame_count,
        record_interval=parsed.record_interval,
        replay_packed=pack_replay(parsed.replay),
    )
    db.session.add(entry)
    db.session.commit()

    return jsonify(entry.to_dict()), 201


@simulator_bp.delete("/entries/<entry_id>")
def delete_entry(entry_id: str):
    require_admin()
    entry = _entry_or_404(entry_id)
    db.session.delete(entry)
    db.session.commit()
    return jsonify({"deleted": entry_id}), 200
