import base64
import json
import uuid
import zlib
from datetime import datetime, timezone

from sqlalchemy.orm import deferred

from database import db


def pack_replay(replay):
    encoded = json.dumps(replay, separators=(",", ":")).encode("utf-8")
    return base64.b64encode(zlib.compress(encoded, 6)).decode("ascii")


def unpack_replay(packed):
    if not packed:
        return {"interval": 0.1, "robots": [], "frames": []}
    return json.loads(zlib.decompress(base64.b64decode(packed)))


class SimulatorEntry(db.Model):
    __tablename__ = "simulator_entries"

    id = db.Column(db.String, primary_key=True, default=lambda: str(uuid.uuid4()))
    player_id = db.Column(db.String(36), nullable=False, index=True)
    username = db.Column(db.String(30), nullable=False)
    title = db.Column(db.String(80), nullable=False)
    description = db.Column(db.String(400), default="")
    game_version = db.Column(db.String(20), default="")

    arena_width = db.Column(db.Float, default=1280.0)
    arena_height = db.Column(db.Float, default=720.0)
    species = db.Column(db.JSON, default=list)
    behaviors = db.Column(db.JSON, default=dict)
    arena_program = db.Column(db.JSON, default=list)
    variables = db.Column(db.JSON, default=list)
    obstacles = db.Column(db.JSON, default=list)
    spawn_zones = db.Column(db.JSON, default=list)

    placement_count = db.Column(db.Integer, default=0)
    peak_robot_count = db.Column(db.Integer, default=0)
    frame_count = db.Column(db.Integer, default=0)
    record_interval = db.Column(db.Float, default=0.1)
    replay_packed = deferred(db.Column(db.Text, nullable=True))

    created_at = db.Column(
        db.DateTime(timezone=True),
        default=lambda: datetime.now(timezone.utc),
        index=True,
    )

    @property
    def duration_seconds(self):
        return round((self.frame_count or 0) * (self.record_interval or 0.1), 1)

    def species_summary(self):
        return [
            {"id": entry.get("id"), "name": entry.get("name"), "color": entry.get("color")}
            for entry in (self.species or [])
        ]

    def to_list_dict(self):
        return {
            "id": self.id,
            "player_id": self.player_id,
            "username": self.username,
            "title": self.title,
            "description": self.description or "",
            "game_version": self.game_version or "",
            "species": self.species_summary(),
            "placement_count": self.placement_count or 0,
            "peak_robot_count": self.peak_robot_count or 0,
            "spawn_zone_count": len(self.spawn_zones or []),
            "duration_seconds": self.duration_seconds,
            "created_at": self.created_at.isoformat(),
        }

    def to_dict(self):
        details = self.to_list_dict()
        details.update({
            "arena_width": self.arena_width,
            "arena_height": self.arena_height,
            "species": self.species or [],
            "behaviors": self.behaviors or {},
            "arena_program": self.arena_program or [],
            "variables": self.variables or [],
            "obstacles": self.obstacles or [],
            "spawn_zones": self.spawn_zones or [],
            "frame_count": self.frame_count or 0,
            "record_interval": self.record_interval or 0.1,
        })
        return details

    def replay_dict(self):
        return unpack_replay(self.replay_packed)
