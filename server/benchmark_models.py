import uuid
from datetime import datetime, timezone

from sqlalchemy.dialects.postgresql import JSONB

from database import db

RUN_KIND_TRIAL = "trial"
RUN_KIND_SWEEP = "sweep"
RUN_KIND_SWEEP_TRIAL = "sweep_trial"


class Benchmark(db.Model):
    __tablename__ = "benchmarks"

    id = db.Column(db.String(36), primary_key=True, default=lambda: str(uuid.uuid4()))
    content_hash = db.Column(db.String(64), nullable=True)
    level_id = db.Column(db.String(40), nullable=True)

    success_rate = db.Column(db.Float, nullable=True)
    attacker_rate = db.Column(db.Float, nullable=True)
    defender_rate = db.Column(db.Float, nullable=True)
    detection_rate = db.Column(db.Float, nullable=True)
    capture_rate = db.Column(db.Float, nullable=True)

    summary = db.Column(JSONB, nullable=True)
    replay_meta = db.Column(JSONB, nullable=True)
    replays_shape = db.Column(db.String(10), nullable=False, default="object")
    source_evaluation_id = db.Column(db.String(36), nullable=True)

    created_at = db.Column(
        db.DateTime(timezone=True),
        default=lambda: datetime.now(timezone.utc),
    )

    def summary_dict(self):
        return self.summary if isinstance(self.summary, dict) else {}

    def meta_dict(self):
        return self.replay_meta if isinstance(self.replay_meta, dict) else {}


class BenchmarkRun(db.Model):
    __tablename__ = "benchmark_runs"

    id = db.Column(db.BigInteger, primary_key=True, autoincrement=True)
    benchmark_id = db.Column(
        db.String(36),
        db.ForeignKey("benchmarks.id", ondelete="CASCADE"),
        nullable=False,
    )
    kind = db.Column(db.String(12), nullable=False)
    position = db.Column(db.Integer, nullable=False)
    parent_position = db.Column(db.Integer, nullable=True)
    sweep_n = db.Column(db.Integer, nullable=True)
    trial = db.Column(db.Integer, nullable=True)
    outcome = db.Column(db.String(20), nullable=True)

    details = db.Column(JSONB, nullable=False, default=dict)
    frames_key = db.Column(db.Text, nullable=True)
    frames_format = db.Column(db.String(10), nullable=True)
    frames_bytes = db.Column(db.BigInteger, nullable=True)

    def details_dict(self):
        return self.details if isinstance(self.details, dict) else {}

    def has_frames(self):
        return self.frames_key is not None
