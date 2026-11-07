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

INLINE_UPLOAD_ID = "inline"
REPLAY_LIST_KEYS = ("runs", "sweep_runs", "pending_run")
INLINE_FRAME_KEYS = ("frames_packed", "frames")
EXTRA_PACKED_KEY = "inline_frames_packed_key"
EXTRA_PLAIN_KEY = "inline_frames_plain_key"
RATE_COLUMNS = ("success_rate", "attacker_rate", "defender_rate", "detection_rate", "capture_rate")


class UnsupportedReplayShape(ValueError):
    pass


def _number(value):
    if isinstance(value, bool):
        return None
    try:
        return float(value)
    except (TypeError, ValueError):
        return None


def _whole_number(value):
    if isinstance(value, bool):
        return None
    if isinstance(value, int):
        return value
    if isinstance(value, float) and value.is_integer():
        return int(value)
    return None


def _outcome(value):
    return value[:20] if isinstance(value, str) else None


def replays_shape(replays):
    if replays is None:
        return "null"
    if isinstance(replays, list):
        return "list"
    return "object"


def replay_meta(replays):
    if not isinstance(replays, dict):
        return {}
    return {key: value for key, value in replays.items() if key not in REPLAY_LIST_KEYS}


def replay_lists(replays):
    if isinstance(replays, list):
        if replays:
            raise UnsupportedReplayShape("replays is a non-empty list")
        return [], []
    if not isinstance(replays, dict):
        return [], []
    runs = replays.get("runs") or []
    sweep_runs = replays.get("sweep_runs") or []
    if not isinstance(runs, list) or not isinstance(sweep_runs, list):
        raise UnsupportedReplayShape("runs or sweep_runs is not a list")
    for run in runs + sweep_runs:
        if not isinstance(run, dict):
            raise UnsupportedReplayShape("a run is not an object")
        trial_runs = run.get("trial_runs") or []
        if not isinstance(trial_runs, list) or not all(isinstance(child, dict) for child in trial_runs):
            raise UnsupportedReplayShape("trial_runs is not a list of objects")
    return runs, sweep_runs


def has_benchmark_data(results, replays):
    if results:
        return True
    if isinstance(replays, list):
        return bool(replays)
    if isinstance(replays, dict):
        return any(key != "pending_run" for key in replays)
    return False


def run_details(run):
    return {key: value for key, value in run.items() if key not in INLINE_FRAME_KEYS and key != "trial_runs"}


def _write_inline(evaluation_id, file_name, data):
    return replay_files.write_file(evaluation_id, INLINE_UPLOAD_ID, file_name, data)


def _frames_columns(evaluation_id, run, file_name, details):
    frames_file = run.get("frames_file")
    packed = run.get("frames_packed")
    plain = run.get("frames")
    packed_key = None
    plain_key = None
    if packed:
        packed_key = _write_inline(evaluation_id, file_name + "-packed", frame_codec.unpack_base64(packed))
    if plain:
        plain_key = _write_inline(evaluation_id, file_name + "-plain", frame_codec.encode_plain(plain))

    if frames_file:
        main_key, main_format = frames_file, frame_codec.FORMAT_DELTA
    elif packed_key:
        main_key, main_format = packed_key, frame_codec.FORMAT_DELTA
    elif plain_key:
        main_key, main_format = plain_key, frame_codec.FORMAT_PLAIN
    else:
        return None, None, None

    if packed_key and packed_key != main_key:
        details[EXTRA_PACKED_KEY] = packed_key
    if plain_key and plain_key != main_key:
        details[EXTRA_PLAIN_KEY] = plain_key
    return main_key, main_format, replay_files.file_size(main_key)


def _add_run(evaluation_id, benchmark_id, kind, position, run, file_name, parent_position=None, sweep_n=None):
    details = run_details(run)
    frames_key, frames_format, frames_bytes = _frames_columns(evaluation_id, run, file_name, details)
    db.session.add(BenchmarkRun(
        benchmark_id=benchmark_id,
        kind=kind,
        position=position,
        parent_position=parent_position,
        sweep_n=sweep_n if sweep_n is not None else _whole_number(run.get("n")),
        trial=_whole_number(run.get("trial")),
        outcome=_outcome(run.get("outcome")),
        details=details,
        frames_key=frames_key,
        frames_format=frames_format,
        frames_bytes=frames_bytes,
    ))


def build_benchmark(evaluation_id, level_id, results, replays, content_hash=None, source_evaluation_id=None):
    runs, sweep_runs = replay_lists(replays)
    rates = results if isinstance(results, dict) else {}
    benchmark = Benchmark(
        content_hash=content_hash,
        level_id=level_id,
        summary=results,
        replay_meta=replay_meta(replays),
        replays_shape=replays_shape(replays),
        source_evaluation_id=source_evaluation_id,
        **{column: _number(rates.get(column)) for column in RATE_COLUMNS},
    )
    db.session.add(benchmark)
    db.session.flush()

    for position, run in enumerate(runs):
        _add_run(evaluation_id, benchmark.id, RUN_KIND_TRIAL, position, run, "trial-%d" % position)
    for position, sweep_run in enumerate(sweep_runs):
        sweep_n = _whole_number(sweep_run.get("n"))
        _add_run(evaluation_id, benchmark.id, RUN_KIND_SWEEP, position, sweep_run, "sweep-%d" % position)
        for child_position, child in enumerate(sweep_run.get("trial_runs") or []):
            _add_run(
                evaluation_id,
                benchmark.id,
                RUN_KIND_SWEEP_TRIAL,
                child_position,
                child,
                "sweep-%d-trial-%d" % (position, child_position),
                parent_position=position,
                sweep_n=sweep_n,
            )
    db.session.flush()
    return benchmark


def find_by_hash(content_hash):
    if not content_hash:
        return None
    return Benchmark.query.filter_by(content_hash=content_hash).first()


def release(benchmark_id):
    from models import PlayerEvaluation

    if not benchmark_id:
        return
    still_used = PlayerEvaluation.query.filter_by(benchmark_id=benchmark_id).first() is not None
    if still_used:
        return
    BenchmarkRun.query.filter_by(benchmark_id=benchmark_id).delete(synchronize_session=False)
    Benchmark.query.filter_by(id=benchmark_id).delete(synchronize_session=False)
