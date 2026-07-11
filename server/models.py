import base64
import json
import threading
import uuid
import zlib
from collections import OrderedDict
from datetime import datetime, timezone

from werkzeug.security import check_password_hash, generate_password_hash

from database import db

ADMIN_SESSION_TTL_SECONDS = 7 * 24 * 3600


_REPLAYS_CACHE = OrderedDict()
_REPLAYS_CACHE_MAX = 6
_CACHE_LOCK = threading.Lock()
_KEY_LOCKS = {}


def _get_key_lock(key):
    with _CACHE_LOCK:
        lock = _KEY_LOCKS.get(key)
        if lock is None:
            lock = threading.Lock()
            _KEY_LOCKS[key] = lock
        return lock


def _unpack_frames(packed):
    if not packed:
        return []
    raw = zlib.decompress(base64.b64decode(packed))
    delta_encoded = json.loads(raw)
    if not delta_encoded:
        return []
    frames = [delta_encoded[0]]
    for i in range(1, len(delta_encoded)):
        prev = frames[i - 1]
        delta = delta_encoded[i]
        frame = [prev[j] + delta[j] for j in range(len(delta))]
        frames.append(frame)
    return frames


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
    stage = db.Column(db.String(200), nullable=True)
    worker_id = db.Column(db.String(64), nullable=True)
    n_max = db.Column(db.Integer, default=40)
    trials = db.Column(db.Integer, default=100)
    results = db.Column(db.JSON, default=dict)
    replays = db.Column(db.JSON, default=dict)
    error = db.Column(db.String(400), nullable=True)

    game_version = db.Column(db.String(20), default="v0.0.4")
    defender_count = db.Column(db.Integer, default=0)
    xp_awarded = db.Column(db.Integer, nullable=True)
    collisions = db.Column(db.Boolean, default=False)

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

    def level_number(self):
        lid = (self.level_id or "farp").lower()
        digits = "".join(ch for ch in lid if ch.isdigit())
        if digits:
            return int(digits)
        return 1

    def is_attack_level(self):
        return self.level_number() >= 3

    def to_dict(self):
        results = self._results_dict()
        return {
            "id": self.id,
            "player_id": self.player_id,
            "username": self.username,
            "level_id": self.level_id or "farp",
            "level_number": self.level_number(),
            "is_attack": self.is_attack_level(),
            "game_version": self.game_version or "v0.0.4",
            "defender_count": self.defender_count or 0,
            "algorithm": self.algorithm or [],
            "placements": self.placements or [],
            "status": self.status,
            "progress": self.progress or 0.0,
            "stage": self.stage,
            "trials": self.trials,
            "results": results,
            "attacker_rate": results.get("attacker_rate"),
            "defender_rate": results.get("defender_rate"),
            "xp_awarded": self.xp_awarded,
            "error": self.error,
            "created_at": self.created_at.isoformat(),
            "completed_at": self.completed_at.isoformat() if self.completed_at else None,
        }

    def to_list_dict(self):
        results = self._results_dict()
        return {
            "id": self.id,
            "player_id": self.player_id,
            "username": self.username,
            "level_id": self.level_id or "farp",
            "level_number": self.level_number(),
            "is_attack": self.is_attack_level(),
            "game_version": self.game_version or "v0.0.4",
            "defender_count": self.defender_count or 0,
            "status": self.status,
            "progress": self.progress or 0.0,
            "stage": self.stage,
            "trials": self.trials,
            "success_rate": results.get("success_rate"),
            "attacker_rate": results.get("attacker_rate"),
            "defender_rate": results.get("defender_rate"),
            "xp_awarded": self.xp_awarded,
            "created_at": self.created_at.isoformat(),
            "completed_at": self.completed_at.isoformat() if self.completed_at else None,
        }

    def _replays_dict(self):
        replays = self.replays or {}
        if isinstance(replays, list):
            return {}
        return replays

    def _cache_version(self):
        stamp = self.completed_at or self.created_at
        return "%s:%s:%s" % (stamp.isoformat() if stamp else "", self.status, self.progress or 0.0)

    def _cached_replays(self):
        key = (self.id, self._cache_version())
        with _CACHE_LOCK:
            cached = _REPLAYS_CACHE.get(key)
            if cached is not None:
                _REPLAYS_CACHE.move_to_end(key)
                return cached
        with _get_key_lock(key):
            with _CACHE_LOCK:
                cached = _REPLAYS_CACHE.get(key)
                if cached is not None:
                    return cached
            replays = self._replays_dict()
            with _CACHE_LOCK:
                _REPLAYS_CACHE[key] = replays
                _REPLAYS_CACHE.move_to_end(key)
                while len(_REPLAYS_CACHE) > _REPLAYS_CACHE_MAX:
                    old_key, _ = _REPLAYS_CACHE.popitem(last=False)
                    _KEY_LOCKS.pop(old_key, None)
            return replays

    def replay_index(self):
        replays = self._cached_replays()
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
        replays = self._cached_replays()
        for run in replays.get("runs", []):
            if run.get("trial") == trial:
                packed = run.get("frames_packed")
                frames = _unpack_frames(packed) if packed else run.get("frames", [])
                return {
                    "trial": run.get("trial"),
                    "outcome": run.get("outcome"),
                    "detection_time": run.get("detection_time", -1),
                    "capture_time": run.get("capture_time", -1),
                    "fps": replays.get("fps", 12),
                    "defenders": replays.get("defenders", 0),
                    "view": replays.get("view", 300),
                    "fov": replays.get("fov", 70),
                    "planet": replays.get("planet"),
                    "arena": replays.get("arena"),
                    "frames": frames,
                }
        return None

    def sweep_index(self):
        replays = self._cached_replays()
        return [
            {
                "n": run.get("n"),
                "outcome": run.get("outcome"),
                "detection_time": run.get("detection_time", -1),
                "capture_time": run.get("capture_time", -1),
                "detection_rate": run.get("detection_rate"),
                "capture_rate": run.get("capture_rate"),
            }
            for run in replays.get("sweep_runs", [])
        ]

    def sweep_replay_for(self, n: int):
        replays = self._cached_replays()
        for run in replays.get("sweep_runs", []):
            if run.get("n") == n:
                packed = run.get("frames_packed")
                frames = _unpack_frames(packed) if packed else run.get("frames", [])
                return {
                    "n": run.get("n"),
                    "outcome": run.get("outcome"),
                    "detection_time": run.get("detection_time", -1),
                    "capture_time": run.get("capture_time", -1),
                    "fps": replays.get("fps", 12),
                    "defenders": run.get("defenders", 0),
                    "view": replays.get("view", 300),
                    "fov": replays.get("fov", 70),
                    "planet": replays.get("planet"),
                    "arena": replays.get("arena"),
                    "frames": frames,
                }
        return None


class EvaluationShard(db.Model):
    __tablename__ = "evaluation_shards"

    id = db.Column(db.String, primary_key=True, default=lambda: str(uuid.uuid4()))
    evaluation_id = db.Column(db.String, index=True, nullable=False)
    shard_index = db.Column(db.Integer, default=0)

    trial_start = db.Column(db.Integer, default=0)
    trial_count = db.Column(db.Integer, default=0)
    n_start = db.Column(db.Integer, default=1)
    n_count = db.Column(db.Integer, default=0)
    total_units = db.Column(db.Integer, default=1)

    status = db.Column(db.String(12), default="queued", index=True)
    done_units = db.Column(db.Integer, default=0)
    worker_id = db.Column(db.String(64), nullable=True)
    result = db.Column(db.JSON, nullable=True)
    error = db.Column(db.String(400), nullable=True)

    created_at = db.Column(
        db.DateTime(timezone=True),
        default=lambda: datetime.now(timezone.utc),
    )
    last_update = db.Column(
        db.DateTime(timezone=True),
        default=lambda: datetime.now(timezone.utc),
    )


class AppSetting(db.Model):
    __tablename__ = "app_settings"

    key = db.Column(db.String(64), primary_key=True)
    value = db.Column(db.String(255))


WORKER_ONLINE_SECONDS = 30
SHARD_STALE_SECONDS = 45


class Worker(db.Model):
    __tablename__ = "workers"

    id = db.Column(db.String(64), primary_key=True)
    name = db.Column(db.String(80), default="worker")
    hostname = db.Column(db.String(120), default="")
    enabled = db.Column(db.Boolean, default=True)
    max_jobs = db.Column(db.Integer, default=1)
    reported_status = db.Column(db.String(20), default="idle")
    current_job_id = db.Column(db.String(64), nullable=True)
    system_stats = db.Column(db.JSON, nullable=True)
    last_seen = db.Column(
        db.DateTime(timezone=True),
        default=lambda: datetime.now(timezone.utc),
    )
    created_at = db.Column(
        db.DateTime(timezone=True),
        default=lambda: datetime.now(timezone.utc),
    )

    def is_online(self):
        if self.last_seen is None:
            return False
        last = self.last_seen
        if last.tzinfo is None:
            last = last.replace(tzinfo=timezone.utc)
        return (datetime.now(timezone.utc) - last).total_seconds() <= WORKER_ONLINE_SECONDS

    def status(self):
        if not self.is_online():
            return "offline"
        if not self.enabled:
            return "disconnected"
        if self.current_job_id:
            return "busy"
        return "idle"

    def to_dict(self):
        return {
            "id": self.id,
            "name": self.name,
            "hostname": self.hostname,
            "enabled": self.enabled,
            "max_jobs": self.max_jobs,
            "status": self.status(),
            "online": self.is_online(),
            "current_job_id": self.current_job_id,
            "system_stats": self.system_stats or {},
            "last_seen": self.last_seen.isoformat() if self.last_seen else None,
            "created_at": self.created_at.isoformat() if self.created_at else None,
        }


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


class AdminUser(db.Model):
    __tablename__ = "admin_users"

    id = db.Column(db.String, primary_key=True, default=lambda: str(uuid.uuid4()))
    username = db.Column(db.String(64), unique=True, nullable=False, index=True)
    password_hash = db.Column(db.String(255), nullable=False)
    created_at = db.Column(
        db.DateTime(timezone=True),
        default=lambda: datetime.now(timezone.utc),
    )
    last_login = db.Column(db.DateTime(timezone=True), nullable=True)

    def set_password(self, password):
        self.password_hash = generate_password_hash(password)

    def check_password(self, password):
        return check_password_hash(self.password_hash, password)

    def to_dict(self):
        return {
            "id": self.id,
            "username": self.username,
            "created_at": self.created_at.isoformat() if self.created_at else None,
            "last_login": self.last_login.isoformat() if self.last_login else None,
        }


class AdminSession(db.Model):
    __tablename__ = "admin_sessions"

    token = db.Column(db.String(64), primary_key=True)
    user_id = db.Column(db.String, nullable=False, index=True)
    created_at = db.Column(
        db.DateTime(timezone=True),
        default=lambda: datetime.now(timezone.utc),
    )
    expires_at = db.Column(db.DateTime(timezone=True), nullable=True)
    last_seen = db.Column(
        db.DateTime(timezone=True),
        default=lambda: datetime.now(timezone.utc),
    )
