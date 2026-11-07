from sqlalchemy import func

import frame_codec
import replay_files
from benchmark_models import (
    RUN_KIND_SWEEP,
    RUN_KIND_SWEEP_TRIAL,
    RUN_KIND_TRIAL,
    Benchmark,
    BenchmarkRun,
)
from database import db


def _benchmark_of(evaluation):
    if not evaluation.benchmark_id:
        return None
    return db.session.get(Benchmark, evaluation.benchmark_id)


def _meta_of(evaluation):
    benchmark = _benchmark_of(evaluation)
    return benchmark.meta_dict() if benchmark is not None else {}


def _runs_query(evaluation, kind):
    return (
        BenchmarkRun.query.filter_by(benchmark_id=evaluation.benchmark_id, kind=kind)
        .order_by(BenchmarkRun.position.asc())
    )


def _runs(evaluation, kind):
    if not evaluation.benchmark_id:
        return []
    return _runs_query(evaluation, kind).all()


def _first_run(evaluation, kind, **filters):
    if not evaluation.benchmark_id:
        return None
    return _runs_query(evaluation, kind).filter_by(**filters).first()


def _sweep_children(evaluation, sweep_run):
    return (
        BenchmarkRun.query.filter_by(
            benchmark_id=evaluation.benchmark_id,
            kind=RUN_KIND_SWEEP_TRIAL,
            parent_position=sweep_run.position,
        )
        .order_by(BenchmarkRun.position.asc())
        .all()
    )


def run_frames(run):
    if run.frames_key is None:
        return []
    return frame_codec.decode(replay_files.read(run.frames_key), run.frames_format)


def replay_index(evaluation):
    meta = _meta_of(evaluation)
    return {
        "fps": meta.get("fps", 12),
        "defenders": meta.get("defenders", 0),
        "view": meta.get("view", 300),
        "fov": meta.get("fov", 70),
        "planet": meta.get("planet"),
        "arena": meta.get("arena"),
        "runs": [
            {
                "trial": run.details_dict().get("trial"),
                "outcome": run.details_dict().get("outcome"),
                "recorded": run.has_frames(),
            }
            for run in _runs(evaluation, RUN_KIND_TRIAL)
        ],
    }


def replay_for(evaluation, trial):
    run = _first_run(evaluation, RUN_KIND_TRIAL, trial=trial)
    if run is None:
        return None
    meta = _meta_of(evaluation)
    details = run.details_dict()
    return {
        "trial": details.get("trial"),
        "outcome": details.get("outcome"),
        "detection_time": details.get("detection_time", -1),
        "capture_time": details.get("capture_time", -1),
        "goal_time": details.get("goal_time", -1),
        "fps": meta.get("fps", 12),
        "defenders": meta.get("defenders", 0),
        "view": meta.get("view", 300),
        "fov": meta.get("fov", 70),
        "speed": meta.get("speed"),
        "hull": meta.get("hull"),
        "planet": meta.get("planet"),
        "arena": meta.get("arena"),
        "stats": details.get("stats", {}),
        "genome_switches": details.get("genome_switches", []),
        "genome_colors": evaluation.genome_colors(),
        "frames": run_frames(run),
    }


def _sweep_child_counts(evaluation):
    if not evaluation.benchmark_id:
        return {}
    counts = (
        db.session.query(BenchmarkRun.parent_position, func.count(BenchmarkRun.id))
        .filter_by(benchmark_id=evaluation.benchmark_id, kind=RUN_KIND_SWEEP_TRIAL)
        .group_by(BenchmarkRun.parent_position)
        .all()
    )
    return dict(counts)


def sweep_index(evaluation):
    child_counts = _sweep_child_counts(evaluation)
    rows = []
    for run in _runs(evaluation, RUN_KIND_SWEEP):
        details = run.details_dict()
        rows.append({
            "n": details.get("n"),
            "outcome": details.get("outcome"),
            "detection_time": details.get("detection_time", -1),
            "capture_time": details.get("capture_time", -1),
            "goal_time": details.get("goal_time", -1),
            "detection_rate": details.get("detection_rate"),
            "capture_rate": details.get("capture_rate"),
            "trial_count": child_counts.get(run.position, 0),
            "recorded": run.has_frames(),
        })
    return rows


def sweep_trial_index(evaluation, n):
    sweep_run = _first_run(evaluation, RUN_KIND_SWEEP, sweep_n=n)
    if sweep_run is None:
        return []
    rows = []
    for trial_run in _sweep_children(evaluation, sweep_run):
        details = trial_run.details_dict()
        rows.append({
            "trial": details.get("trial"),
            "outcome": details.get("outcome"),
            "detection_time": details.get("detection_time", -1),
            "capture_time": details.get("capture_time", -1),
            "goal_time": details.get("goal_time", -1),
            "recorded": trial_run.has_frames(),
        })
    return rows


def sweep_trial_replay_for(evaluation, n, trial):
    sweep_run = _first_run(evaluation, RUN_KIND_SWEEP, sweep_n=n)
    if sweep_run is None:
        return None
    trial_run = next(
        (child for child in _sweep_children(evaluation, sweep_run) if child.trial == trial),
        None,
    )
    if trial_run is None:
        return None
    meta = _meta_of(evaluation)
    sweep_details = sweep_run.details_dict()
    details = trial_run.details_dict()
    return {
        "n": sweep_details.get("n"),
        "trial": details.get("trial"),
        "outcome": details.get("outcome"),
        "detection_time": details.get("detection_time", -1),
        "capture_time": details.get("capture_time", -1),
        "goal_time": details.get("goal_time", -1),
        "fps": meta.get("fps", 12),
        "defenders": sweep_details.get("defenders", 0),
        "view": meta.get("view", 300),
        "fov": meta.get("fov", 70),
        "hull": meta.get("hull"),
        "planet": meta.get("planet"),
        "arena": meta.get("arena"),
        "genome_switches": details.get("genome_switches", []),
        "genome_colors": evaluation.genome_colors(),
        "frames": run_frames(trial_run),
    }


def sweep_replay_for(evaluation, n):
    sweep_run = _first_run(evaluation, RUN_KIND_SWEEP, sweep_n=n)
    if sweep_run is None:
        return None
    meta = _meta_of(evaluation)
    details = sweep_run.details_dict()
    return {
        "n": details.get("n"),
        "outcome": details.get("outcome"),
        "detection_time": details.get("detection_time", -1),
        "capture_time": details.get("capture_time", -1),
        "goal_time": details.get("goal_time", -1),
        "fps": meta.get("fps", 12),
        "defenders": details.get("defenders", 0),
        "view": meta.get("view", 300),
        "fov": meta.get("fov", 70),
        "planet": meta.get("planet"),
        "arena": meta.get("arena"),
        "genome_switches": details.get("genome_switches", []),
        "genome_colors": evaluation.genome_colors(),
        "frames": run_frames(sweep_run),
    }
