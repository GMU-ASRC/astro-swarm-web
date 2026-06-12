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
