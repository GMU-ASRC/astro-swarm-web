import uuid
from datetime import datetime, timezone

from database import db


class SimConfig(db.Model):
    __tablename__ = "sim_configs"

    id = db.Column(db.String, primary_key=True, default=lambda: str(uuid.uuid4()))
    title = db.Column(db.String(80), nullable=False)
    description = db.Column(db.String(400), default="")
    author = db.Column(db.String(60), default="anonymous")

    original_filename = db.Column(db.String(255), nullable=False)
    stored_filename = db.Column(db.String(255), nullable=False)
    file_type = db.Column(db.String(10), nullable=False)
    file_size = db.Column(db.Integer, nullable=False)

    species = db.Column(db.JSON, default=list)
    robot_count = db.Column(db.Integer, default=0)
    arena_width = db.Column(db.Float, default=1280.0)
    arena_height = db.Column(db.Float, default=720.0)

    download_count = db.Column(db.Integer, default=0)
    created_at = db.Column(
        db.DateTime(timezone=True),
        default=lambda: datetime.now(timezone.utc),
    )

    def to_dict(self):
        return {
            "id": self.id,
            "title": self.title,
            "description": self.description,
            "author": self.author,
            "original_filename": self.original_filename,
            "file_type": self.file_type,
            "file_size": self.file_size,
            "species": self.species or [],
            "robot_count": self.robot_count,
            "arena_width": self.arena_width,
            "arena_height": self.arena_height,
            "download_count": self.download_count,
            "created_at": self.created_at.isoformat(),
        }

    def to_list_dict(self):
        return {
            "id": self.id,
            "title": self.title,
            "description": self.description,
            "author": self.author,
            "file_type": self.file_type,
            "species": self.species or [],
            "robot_count": self.robot_count,
            "download_count": self.download_count,
            "created_at": self.created_at.isoformat(),
        }


class SimRun(db.Model):
    __tablename__ = "sim_runs"

    id = db.Column(db.String, primary_key=True, default=lambda: str(uuid.uuid4()))
    title = db.Column(db.String(80), nullable=False)
    description = db.Column(db.String(400), default="")
    author = db.Column(db.String(60), default="anonymous")

    original_filename = db.Column(db.String(255), nullable=False)
    stored_filename = db.Column(db.String(255), nullable=False)
    file_size = db.Column(db.Integer, nullable=False)

    species = db.Column(db.JSON, default=list)
    robot_count = db.Column(db.Integer, default=0)
    frame_count = db.Column(db.Integer, default=0)
    arena_width = db.Column(db.Float, default=1280.0)
    arena_height = db.Column(db.Float, default=720.0)

    download_count = db.Column(db.Integer, default=0)
    created_at = db.Column(
        db.DateTime(timezone=True),
        default=lambda: datetime.now(timezone.utc),
    )

    @property
    def duration_seconds(self):
        return round(self.frame_count * 0.1, 1)

    def to_dict(self):
        return {
            "id": self.id,
            "title": self.title,
            "description": self.description,
            "author": self.author,
            "original_filename": self.original_filename,
            "file_size": self.file_size,
            "species": self.species or [],
            "robot_count": self.robot_count,
            "frame_count": self.frame_count,
            "duration_seconds": self.duration_seconds,
            "arena_width": self.arena_width,
            "arena_height": self.arena_height,
            "download_count": self.download_count,
            "created_at": self.created_at.isoformat(),
        }

    def to_list_dict(self):
        return {
            "id": self.id,
            "title": self.title,
            "description": self.description,
            "author": self.author,
            "file_size": self.file_size,
            "species": self.species or [],
            "robot_count": self.robot_count,
            "duration_seconds": self.duration_seconds,
            "download_count": self.download_count,
            "created_at": self.created_at.isoformat(),
        }
