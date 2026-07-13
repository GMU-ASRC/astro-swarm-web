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
        if not isinstance(self.placements, list):
            raise ValueError("placements must be a list")
        if len(self.placements) > 6:
            raise ValueError("placements must not exceed 6 defenders")
        if not isinstance(self.run, dict):
            raise ValueError("run must be an object")

        outcome = self.run.get("outcome")
        if outcome not in VALID_RUN_OUTCOMES:
            raise ValueError("run.outcome must be one of %s" % (VALID_RUN_OUTCOMES,))

        frames = self.run.get("frames")
        if not isinstance(frames, list) or not frames:
            raise ValueError("run.frames must be a non-empty list")
        if len(frames) > MAX_RUN_FRAMES:
            raise ValueError("run.frames must not exceed %d frames" % MAX_RUN_FRAMES)

        width = len(self.placements) * 3 + 3
        for frame in frames:
            if not isinstance(frame, list) or len(frame) != width:
                raise ValueError("each run frame must hold %d values" % width)

        fps = int(self.run.get("fps", 30))
        if not (1 <= fps <= MAX_RUN_FPS):
            raise ValueError("run.fps must be between 1 and %d" % MAX_RUN_FPS)
        self.run["fps"] = fps

        goal_time = float(self.run.get("goal_time", -1.0))
        if goal_time > MAX_RUN_SECONDS:
            raise ValueError("run.goal_time must not exceed %d seconds" % MAX_RUN_SECONDS)
        if len(frames) / fps > MAX_RUN_SECONDS + 5:
            raise ValueError("run must not exceed %d seconds" % MAX_RUN_SECONDS)
