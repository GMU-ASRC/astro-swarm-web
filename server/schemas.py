from datetime import datetime
from dataclasses import dataclass, field

@dataclass
class SpeciesInfo:
    id: str
    name: str
    color: str

@dataclass
class SimConfigCreate:
    title: str
    description: str = ""
    author: str = "anonymous"

    def __post_init__(self):
        if len(self.title) > 80:
            raise ValueError("title too long")
        if len(self.description) > 400:
            raise ValueError("description too long")
        if len(self.author) > 60:
            raise ValueError("author too long")

@dataclass
class LeaderboardSubmit:
    player_id: str
    username: str
    time_seconds: float
    algorithm: list = field(default_factory=list)

    def __post_init__(self):
        if not isinstance(self.player_id, str) or len(self.player_id) != 36:
            raise ValueError("player_id must be exactly 36 characters")
        if not isinstance(self.username, str) or not (1 <= len(self.username) <= 30):
            raise ValueError("username must be between 1 and 30 characters")
        if not isinstance(self.time_seconds, (int, float)) or not (2.0 <= self.time_seconds <= 90.0):
            raise ValueError("time_seconds must be between 2.0 and 90.0")
        self.time_seconds = float(self.time_seconds)

@dataclass
class EvaluationSubmit:
    player_id: str
    username: str
    level_id: str = "farp"
    algorithm: list = field(default_factory=list)
    placements: list = field(default_factory=list)
    trials: int = 100
    game_version: str = "v0.0.4"
    collisions: bool = False

    def __post_init__(self):
        self.collisions = bool(self.collisions)
        if not isinstance(self.player_id, str) or len(self.player_id) != 36:
            raise ValueError("player_id must be exactly 36 characters")
        if not isinstance(self.username, str) or not (1 <= len(self.username) <= 30):
            raise ValueError("username must be between 1 and 30 characters")
        if not isinstance(self.level_id, str) or not (1 <= len(self.level_id) <= 40):
            raise ValueError("level_id must be between 1 and 40 characters")
        if not isinstance(self.game_version, str) or len(self.game_version) > 20:
            raise ValueError("game_version must be a string up to 20 characters")
        if not isinstance(self.algorithm, list):
            raise ValueError("algorithm must be a list")
        if not isinstance(self.placements, list):
            raise ValueError("placements must be a list")
        if len(self.placements) > 6:
            raise ValueError("placements must not exceed 6 defenders")
        self.trials = max(100, int(self.trials))
        if self.trials > 500:
            raise ValueError("trials must not exceed 500")


MAX_RUN_SECONDS = 180
MAX_RUN_FPS = 60
MAX_RUN_FRAMES = MAX_RUN_SECONDS * MAX_RUN_FPS + 60
VALID_RUN_OUTCOMES = ("win", "lose", "timeout")

DEFAULT_RUN_LIMITS = {"max_ships": 6, "max_seconds": MAX_RUN_SECONDS}

# Level 4 flies a leader against two milling swarms, so it records far more
# ships than the defence levels and runs for longer.
RUN_LIMITS = {
    "farp4": {"max_ships": 24, "max_seconds": 300},
}


def run_limits_for(level_id):
    return RUN_LIMITS.get(level_id, DEFAULT_RUN_LIMITS)


@dataclass
class RunSubmit:
    player_id: str
    username: str
    level_id: str
    run: dict
    algorithm: list = field(default_factory=list)
    placements: list = field(default_factory=list)
    game_version: str = "v0.0.4"

    def __post_init__(self):
        if not isinstance(self.player_id, str) or len(self.player_id) != 36:
            raise ValueError("player_id must be exactly 36 characters")
        if not isinstance(self.username, str) or not (1 <= len(self.username) <= 30):
            raise ValueError("username must be between 1 and 30 characters")
        if not isinstance(self.level_id, str) or not (1 <= len(self.level_id) <= 40):
            raise ValueError("level_id must be between 1 and 40 characters")
        if not isinstance(self.game_version, str) or len(self.game_version) > 20:
            raise ValueError("game_version must be a string up to 20 characters")
        if not isinstance(self.algorithm, list):
            raise ValueError("algorithm must be a list")
        limits = run_limits_for(self.level_id)
        max_ships = limits["max_ships"]
        max_seconds = limits["max_seconds"]
        max_frames = max_seconds * MAX_RUN_FPS + 60

        if not isinstance(self.placements, list):
            raise ValueError("placements must be a list")
        if len(self.placements) > max_ships:
            raise ValueError("placements must not exceed %d ships" % max_ships)
        if not isinstance(self.run, dict):
            raise ValueError("run must be an object")

        outcome = self.run.get("outcome")
        if outcome not in VALID_RUN_OUTCOMES:
            raise ValueError("run.outcome must be one of %s" % (VALID_RUN_OUTCOMES,))

        frames = self.run.get("frames")
        if not isinstance(frames, list) or not frames:
            raise ValueError("run.frames must be a non-empty list")
        if len(frames) > max_frames:
            raise ValueError("run.frames must not exceed %d frames" % max_frames)

        width = len(self.placements) * 3 + 3
        for frame in frames:
            if not isinstance(frame, list) or len(frame) != width:
                raise ValueError("each run frame must hold %d values" % width)

        fps = int(self.run.get("fps", 30))
        if not (1 <= fps <= MAX_RUN_FPS):
            raise ValueError("run.fps must be between 1 and %d" % MAX_RUN_FPS)
        self.run["fps"] = fps

        goal_time = float(self.run.get("goal_time", -1.0))
        if goal_time > max_seconds:
            raise ValueError("run.goal_time must not exceed %d seconds" % max_seconds)
        if len(frames) / fps > max_seconds + 5:
            raise ValueError("run must not exceed %d seconds" % max_seconds)

        stats = self.run.get("stats")
        if stats is not None:
            if not isinstance(stats, dict):
                raise ValueError("run.stats must be an object")
            self.run["stats"] = {
                str(key)[:40]: float(value)
                for key, value in list(stats.items())[:24]
                if isinstance(value, (int, float))
            }


MAX_SURVIVE_SECONDS = 180
MAX_APM_SAMPLES = 240
VALID_SURVIVE_OUTCOMES = ("player1", "player2", "tie")
SURVIVE_PLAYER_FIELDS = (
    "evaders_through",
    "defenders_at_end",
    "herded_total",
    "freezes_used",
    "actions",
    "apm_peak",
)


@dataclass
class SurviveMatchSubmit:
    player_id: str
    username: str
    outcome: str
    players: list = field(default_factory=list)
    duration: int = 0
    game_version: str = "v0.0.5"
    apm_bucket_seconds: int = 5
    apm_times: list = field(default_factory=list)
    apm_player1: list = field(default_factory=list)
    apm_player2: list = field(default_factory=list)
    evaders_spawned: int = 0
    wild_remaining: int = 0
    first_target: int = 1

    def __post_init__(self):
        if not isinstance(self.player_id, str) or len(self.player_id) != 36:
            raise ValueError("player_id must be exactly 36 characters")
        if not isinstance(self.username, str) or not (1 <= len(self.username) <= 30):
            raise ValueError("username must be between 1 and 30 characters")
        if not isinstance(self.game_version, str) or len(self.game_version) > 20:
            raise ValueError("game_version must be a string up to 20 characters")
        if self.outcome not in VALID_SURVIVE_OUTCOMES:
            raise ValueError("outcome must be one of %s" % (VALID_SURVIVE_OUTCOMES,))

        self.duration = int(self.duration)
        if not (0 <= self.duration <= MAX_SURVIVE_SECONDS):
            raise ValueError("duration must be between 0 and %d seconds" % MAX_SURVIVE_SECONDS)

        if not isinstance(self.players, list) or len(self.players) != 2:
            raise ValueError("players must hold exactly 2 entries")
        self.players = [self._clean_player(entry, slot + 1) for slot, entry in enumerate(self.players)]

        self.apm_bucket_seconds = int(self.apm_bucket_seconds)
        if not (1 <= self.apm_bucket_seconds <= 60):
            raise ValueError("apm_bucket_seconds must be between 1 and 60")

        self.apm_times = self._clean_series(self.apm_times, "apm_times")
        self.apm_player1 = self._clean_series(self.apm_player1, "apm_player1")
        self.apm_player2 = self._clean_series(self.apm_player2, "apm_player2")
        if len(self.apm_player1) != len(self.apm_times) or len(self.apm_player2) != len(self.apm_times):
            raise ValueError("apm series must be the same length as apm_times")

        self.evaders_spawned = max(0, int(self.evaders_spawned))
        self.wild_remaining = max(0, int(self.wild_remaining))
        self.first_target = 2 if int(self.first_target) == 2 else 1

    def _clean_series(self, values, label):
        if not isinstance(values, list):
            raise ValueError("%s must be a list" % label)
        if len(values) > MAX_APM_SAMPLES:
            raise ValueError("%s must not exceed %d samples" % (label, MAX_APM_SAMPLES))
        return [max(0, int(value)) for value in values]

    def _clean_player(self, entry, slot):
        if not isinstance(entry, dict):
            raise ValueError("each player entry must be an object")
        name = str(entry.get("name", "")).strip() or "Player %d" % slot
        cleaned = {"slot": slot, "name": name[:30]}
        for key in SURVIVE_PLAYER_FIELDS:
            cleaned[key] = max(0, int(entry.get(key, 0)))
        cleaned["apm_average"] = max(0.0, round(float(entry.get("apm_average", 0.0)), 1))
        return cleaned
