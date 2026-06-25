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
    thumbnail_filename = db.Column(db.String(255), nullable=True)
    video_filename = db.Column(db.String(255), nullable=True)
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
            "thumbnail_filename": self.thumbnail_filename,
            "video_filename": self.video_filename,
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
            "thumbnail_filename": self.thumbnail_filename,
            "video_filename": self.video_filename,
            "file_size": self.file_size,
            "species": self.species or [],
            "robot_count": self.robot_count,
            "duration_seconds": self.duration_seconds,
            "download_count": self.download_count,
            "created_at": self.created_at.isoformat(),
        }


class PlayerEvaluation(db.Model):
    __tablename__ = "player_evaluations"

    id = db.Column(db.String, primary_key=True, default=lambda: str(uuid.uuid4()))
    player_id = db.Column(db.String(36), nullable=False, index=True)
    username = db.Column(db.String(30), nullable=False)
    level_id = db.Column(db.String(40), default="farp", index=True)
    algorithm = db.Column(db.JSON, default=list)
    placements = db.Column(db.JSON, default=list)

    status = db.Column(db.String(12), default="queued")
    progress = db.Column(db.Float, default=0.0)
    n_max = db.Column(db.Integer, default=40)
    trials = db.Column(db.Integer, default=100)
    results = db.Column(db.JSON, default=dict)
    replays = db.Column(db.JSON, default=dict)
    error = db.Column(db.String(400), nullable=True)

    created_at = db.Column(
        db.DateTime(timezone=True),
        default=lambda: datetime.now(timezone.utc),
    )
    completed_at = db.Column(db.DateTime(timezone=True), nullable=True)

    def _results_dict(self):
        results = self.results or {}
        if isinstance(results, list):
            return {}
        return results

    def to_dict(self):
        return {
            "id": self.id,
            "player_id": self.player_id,
            "username": self.username,
            "level_id": self.level_id or "farp",
            "algorithm": self.algorithm or [],
            "placements": self.placements or [],
            "status": self.status,
            "progress": self.progress or 0.0,
            "trials": self.trials,
            "results": self._results_dict(),
            "error": self.error,
            "created_at": self.created_at.isoformat(),
            "completed_at": self.completed_at.isoformat() if self.completed_at else None,
        }

    def to_list_dict(self):
        return {
            "id": self.id,
            "player_id": self.player_id,
            "username": self.username,
            "level_id": self.level_id or "farp",
            "status": self.status,
            "progress": self.progress or 0.0,
            "trials": self.trials,
            "success_rate": self._results_dict().get("success_rate"),
            "created_at": self.created_at.isoformat(),
            "completed_at": self.completed_at.isoformat() if self.completed_at else None,
        }

    def _replays_dict(self):
        replays = self.replays or {}
        if isinstance(replays, list):
            return {}
        return replays

    def replay_index(self):
        replays = self._replays_dict()
        return {
            "fps": replays.get("fps", 12),
            "defenders": replays.get("defenders", 0),
            "view": replays.get("view", 300),
            "fov": replays.get("fov", 70),
            "planet": replays.get("planet"),
            "arena": replays.get("arena"),
            "runs": [
                {"trial": run.get("trial"), "outcome": run.get("outcome")}
                for run in replays.get("runs", [])
            ],
        }

    def replay_for(self, trial: int):
        replays = self._replays_dict()
        for run in replays.get("runs", []):
            if run.get("trial") == trial:
                return {
                    "trial": run.get("trial"),
                    "outcome": run.get("outcome"),
                    "fps": replays.get("fps", 12),
                    "defenders": replays.get("defenders", 0),
                    "view": replays.get("view", 300),
                    "fov": replays.get("fov", 70),
                    "planet": replays.get("planet"),
                    "arena": replays.get("arena"),
                    "frames": run.get("frames", []),
                }
        return None

    def sweep_index(self):
        replays = self._replays_dict()
        return [
            {"n": run.get("n"), "outcome": run.get("outcome")}
            for run in replays.get("sweep_runs", [])
        ]

    def sweep_replay_for(self, n: int):
        replays = self._replays_dict()
        for run in replays.get("sweep_runs", []):
            if run.get("n") == n:
                return {
                    "n": run.get("n"),
                    "outcome": run.get("outcome"),
                    "fps": replays.get("fps", 12),
                    "defenders": run.get("defenders", 0),
                    "view": replays.get("view", 300),
                    "fov": replays.get("fov", 70),
                    "planet": replays.get("planet"),
                    "arena": replays.get("arena"),
                    "frames": run.get("frames", []),
                }
        return None


class LeaderboardEntry(db.Model):
    __tablename__ = "leaderboard_entries"

    id = db.Column(db.String, primary_key=True, default=lambda: str(uuid.uuid4()))
    player_id = db.Column(db.String(36), unique=True, nullable=False)
    username = db.Column(db.String(30), nullable=False)
    time_seconds = db.Column(db.Float, nullable=False)
    algorithm = db.Column(db.JSON, default=list)
    created_at = db.Column(
        db.DateTime(timezone=True),
        default=lambda: datetime.now(timezone.utc),
    )

    def to_dict(self):
        return {
            "id": self.id,
            "username": self.username,
            "time_seconds": self.time_seconds,
            "algorithm": self.algorithm or [],
            "created_at": self.created_at.isoformat(),
        }
