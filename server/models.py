import uuid
from datetime import datetime, timezone

from sqlalchemy.dialects.postgresql import JSONB
from sqlalchemy.orm import deferred
from werkzeug.security import check_password_hash, generate_password_hash

import benchmark_models
import benchmark_replays
from database import db

ADMIN_SESSION_TTL_SECONDS = 7 * 24 * 3600
LEGACY_STORAGE_VERSION = 1
CURRENT_STORAGE_VERSION = 2


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
    level_id = db.Column(db.String(40), default="farp1", index=True)
    algorithm = db.Column(db.JSON, default=list)
    placements = db.Column(db.JSON, default=list)
    genome_schedule = db.Column(db.JSON, default=list)

    status = db.Column(db.String(12), default="queued")
    progress = db.Column(db.Float, default=0.0)
    stage = db.Column(db.String(200), nullable=True)
    worker_id = db.Column(db.String(64), nullable=True)
    n_max = db.Column(db.Integer, default=40)
    trials = db.Column(db.Integer, default=100)
    results = deferred(db.Column(db.JSON, default=dict))
    replays = deferred(db.Column(db.JSON, default=dict))
    error = db.Column(db.String(400), nullable=True)

    benchmark_id = db.Column(db.String(36), nullable=True)
    pending_run = deferred(db.Column(JSONB, nullable=True))
    job_config = db.Column(JSONB, nullable=True)
    storage_version = db.Column(db.SmallInteger, default=CURRENT_STORAGE_VERSION)

    done_units = db.Column(db.Integer, default=0)
    total_units = db.Column(db.Integer, default=1)
    last_update = db.Column(
        db.DateTime(timezone=True),
        default=lambda: datetime.now(timezone.utc),
    )

    game_version = db.Column(db.String(20), default="v0.0.4")
    defender_count = db.Column(db.Integer, default=0)
    xp_awarded = db.Column(db.Integer, nullable=True)
    coins_awarded = db.Column(db.Integer, nullable=True)
    collisions = db.Column(db.Boolean, default=False)

    created_at = db.Column(
        db.DateTime(timezone=True),
        default=lambda: datetime.now(timezone.utc),
    )
    completed_at = db.Column(db.DateTime(timezone=True), nullable=True)

    benchmark = db.relationship(
        "Benchmark",
        primaryjoin="foreign(PlayerEvaluation.benchmark_id) == Benchmark.id",
        lazy="select",
        viewonly=True,
    )

    def result_summary(self):
        if self.benchmark is None:
            return {}
        return self.benchmark.summary_dict()

    def level_number(self):
        lid = (self.level_id or "farp").lower()
        digits = "".join(ch for ch in lid if ch.isdigit())
        if digits:
            return int(digits)
        return 1

    def is_attack_level(self):
        return self.level_number() in (6, 9)

    def to_dict(self):
        results = self.result_summary()
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
            "genome_schedule": self.genome_schedule or [],
            "placements": self.placements or [],
            "status": self.status,
            "progress": self.progress or 0.0,
            "stage": self.stage,
            "trials": self.trials,
            "results": results,
            "attacker_rate": results.get("attacker_rate"),
            "defender_rate": results.get("defender_rate"),
            "xp_awarded": self.xp_awarded,
            "coins_awarded": self.coins_awarded,
            "error": self.error,
            "created_at": self.created_at.isoformat(),
            "completed_at": self.completed_at.isoformat() if self.completed_at else None,
        }

    def _list_rates(self):
        light_rates = getattr(self, "light_rates", None)
        if light_rates is not None:
            return light_rates
        return self.result_summary()

    def to_list_dict(self):
        results = self._list_rates()
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
            "coins_awarded": self.coins_awarded,
            "created_at": self.created_at.isoformat(),
            "completed_at": self.completed_at.isoformat() if self.completed_at else None,
        }

    def genome_colors(self):
        colors = []
        for stage in self.genome_schedule or []:
            color = stage.get("color") if isinstance(stage, dict) else None
            colors.append("#" + color if isinstance(color, str) else None)
        return colors

    def replay_index(self):
        return benchmark_replays.replay_index(self)

    def replay_for(self, trial: int):
        return benchmark_replays.replay_for(self, trial)

    def sweep_index(self):
        return benchmark_replays.sweep_index(self)

    def sweep_trial_index(self, n: int):
        return benchmark_replays.sweep_trial_index(self, n)

    def sweep_trial_replay_for(self, n: int, trial: int):
        return benchmark_replays.sweep_trial_replay_for(self, n, trial)

    def sweep_replay_for(self, n: int):
        return benchmark_replays.sweep_replay_for(self, n)


class AppSetting(db.Model):
    __tablename__ = "app_settings"

    key = db.Column(db.String(64), primary_key=True)
    value = db.Column(db.String(255))


WORKER_ONLINE_SECONDS = 30
# A running job reports progress every couple of seconds, right through the
# upload of its result. This much silence means it is never coming back.
JOB_SILENT_SECONDS = 300


class Worker(db.Model):
    __tablename__ = "workers"

    id = db.Column(db.String(64), primary_key=True)
    name = db.Column(db.String(80), default="worker")
    hostname = db.Column(db.String(120), default="")
    enabled = db.Column(db.Boolean, default=True)
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
            "status": self.status(),
            "online": self.is_online(),
            "current_job_id": self.current_job_id,
            "system_stats": self.system_stats or {},
            "last_seen": self.last_seen.isoformat() if self.last_seen else None,
            "created_at": self.created_at.isoformat() if self.created_at else None,
        }


class SurviveMatch(db.Model):
    __tablename__ = "survive_matches"

    id = db.Column(db.String, primary_key=True, default=lambda: str(uuid.uuid4()))
    player_id = db.Column(db.String(36), nullable=False, index=True)
    username = db.Column(db.String(30), nullable=False)
    game_version = db.Column(db.String(20), default="v0.0.5")

    outcome = db.Column(db.String(10), default="tie")
    duration = db.Column(db.Integer, default=0)
    players = db.Column(db.JSON, default=list)

    apm_bucket_seconds = db.Column(db.Integer, default=5)
    apm_times = db.Column(db.JSON, default=list)
    apm_player1 = db.Column(db.JSON, default=list)
    apm_player2 = db.Column(db.JSON, default=list)

    evaders_spawned = db.Column(db.Integer, default=0)
    wild_remaining = db.Column(db.Integer, default=0)
    first_target = db.Column(db.Integer, default=1)

    created_at = db.Column(
        db.DateTime(timezone=True),
        default=lambda: datetime.now(timezone.utc),
    )

    def _players(self):
        players = self.players
        if not isinstance(players, list):
            return []
        return players

    def _player(self, slot):
        for entry in self._players():
            if int(entry.get("slot", 0)) == slot:
                return entry
        return {}

    def player_name(self, slot):
        return self._player(slot).get("name") or "Player %d" % slot

    def winner_name(self):
        if self.outcome == "player1":
            return self.player_name(1)
        if self.outcome == "player2":
            return self.player_name(2)
        return None

    def to_list_dict(self):
        return {
            "id": self.id,
            "username": self.username,
            "game_version": self.game_version or "v0.0.5",
            "outcome": self.outcome or "tie",
            "winner": self.winner_name(),
            "duration": self.duration or 0,
            "player1": self.player_name(1),
            "player2": self.player_name(2),
            "evaders_player1": self._player(1).get("evaders_through", 0),
            "evaders_player2": self._player(2).get("evaders_through", 0),
            "apm_player1": self._player(1).get("apm_average", 0),
            "apm_player2": self._player(2).get("apm_average", 0),
            "created_at": self.created_at.isoformat(),
        }

    def to_dict(self):
        data = self.to_list_dict()
        data.update({
            "player_id": self.player_id,
            "players": self._players(),
            "apm_bucket_seconds": self.apm_bucket_seconds or 5,
            "apm_times": self.apm_times or [],
            "apm_series_player1": self.apm_player1 or [],
            "apm_series_player2": self.apm_player2 or [],
            "evaders_spawned": self.evaders_spawned or 0,
            "wild_remaining": self.wild_remaining or 0,
            "first_target": self.first_target or 1,
        })
        return data


class TrainingScore(db.Model):
    __tablename__ = "training_scores"

    id = db.Column(db.String, primary_key=True, default=lambda: str(uuid.uuid4()))
    player_id = db.Column(db.String(36), unique=True, nullable=False)
    username = db.Column(db.String(30), nullable=False)
    game_version = db.Column(db.String(20), default="v0.0.9")
    circliness = db.Column(db.Float, nullable=False)
    defenders = db.Column(db.Integer, default=0)
    algorithm = db.Column(db.JSON, default=list)
    placements = db.Column(db.JSON, default=list)
    updated_at = db.Column(
        db.DateTime(timezone=True),
        default=lambda: datetime.now(timezone.utc),
    )

    def to_dict(self, rank=None):
        return {
            "id": self.id,
            "rank": rank,
            "player_id": self.player_id,
            "username": self.username,
            "circliness": round(self.circliness, 4),
            "defenders": self.defenders or 0,
            "game_version": self.game_version,
            "updated_at": self.updated_at.isoformat() if self.updated_at else None,
        }

    def to_detail_dict(self):
        detail = self.to_dict()
        detail["algorithm"] = self.algorithm or []
        detail["placements"] = self.placements or []
        return detail


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
