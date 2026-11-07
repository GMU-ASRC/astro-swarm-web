import json

from sqlalchemy import text

import benchmark_store
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
from models import CURRENT_STORAGE_VERSION

SELECT_ROW = text(
    "SELECT id, level_id, results::text AS results_text, replays::text AS replays_text,"
    " benchmark_id, pending_run::text AS pending_run_text, storage_version"
    " FROM player_evaluations WHERE id = :id"
)

INLINE_PREFIX_FORMAT = "%s/" + benchmark_store.INLINE_UPLOAD_ID + "/"


def _parse(json_text):
    if json_text is None:
        return None
    return json.loads(json_text)


def _old_frames(run):
    packed = run.get("frames_packed")
    if packed:
        return frame_codec.decode_delta(frame_codec.unpack_base64(packed))
    return run.get("frames") or []


def _new_frames(frames_key, frames_format):
    data = replay_files.read(frames_key)
    if data is None:
        return None
    return frame_codec.decode(data, frames_format)


def _check_run(problems, label, evaluation_id, old_run, new_run):
    if new_run.details_dict() != _expected_details(new_run, old_run):
        problems.append("%s: run details differ" % label)

    frames_file = old_run.get("frames_file")
    has_old_frames = bool(frames_file or old_run.get("frames_packed") or old_run.get("frames"))
    if has_old_frames != new_run.has_frames():
        problems.append("%s: recorded flag differs" % label)
        return

    if frames_file:
        if new_run.frames_key != frames_file or new_run.frames_format != frame_codec.FORMAT_DELTA:
            problems.append("%s: frames file key differs" % label)
    elif has_old_frames:
        if not new_run.frames_key.startswith(INLINE_PREFIX_FORMAT % evaluation_id):
            problems.append("%s: inline frames were not written to the inline folder" % label)
        elif _new_frames(new_run.frames_key, new_run.frames_format) != _old_frames(old_run):
            problems.append("%s: inline frames differ after decoding" % label)

    details = new_run.details_dict()
    if old_run.get("frames_packed") and frames_file:
        stored = _new_frames(details.get(benchmark_store.EXTRA_PACKED_KEY), frame_codec.FORMAT_DELTA)
        if stored != frame_codec.decode_delta(frame_codec.unpack_base64(old_run["frames_packed"])):
            problems.append("%s: extra packed frames differ" % label)
    if old_run.get("frames") and (frames_file or old_run.get("frames_packed")):
        stored = _new_frames(details.get(benchmark_store.EXTRA_PLAIN_KEY), frame_codec.FORMAT_PLAIN)
        if stored != old_run["frames"]:
            problems.append("%s: extra plain frames differ" % label)


def _expected_details(new_run, old_run):
    expected = benchmark_store.run_details(old_run)
    details = new_run.details_dict()
    for extra_key in (benchmark_store.EXTRA_PACKED_KEY, benchmark_store.EXTRA_PLAIN_KEY):
        if extra_key in details:
            expected[extra_key] = details[extra_key]
    return expected


def _runs_of(benchmark_id, kind, parent_position=None):
    query = BenchmarkRun.query.filter_by(benchmark_id=benchmark_id, kind=kind)
    if parent_position is not None:
        query = query.filter_by(parent_position=parent_position)
    return query.order_by(BenchmarkRun.position.asc()).all()


def _check_run_lists(problems, row_id, benchmark_id, old_runs, old_sweep_runs):
    new_runs = _runs_of(benchmark_id, RUN_KIND_TRIAL)
    if len(new_runs) != len(old_runs):
        problems.append("%s: %d trial runs, expected %d" % (row_id, len(new_runs), len(old_runs)))
    for position, (old_run, new_run) in enumerate(zip(old_runs, new_runs)):
        _check_run(problems, "%s trial %d" % (row_id, position), row_id, old_run, new_run)

    new_sweeps = _runs_of(benchmark_id, RUN_KIND_SWEEP)
    if len(new_sweeps) != len(old_sweep_runs):
        problems.append("%s: %d sweep runs, expected %d" % (row_id, len(new_sweeps), len(old_sweep_runs)))
    for position, (old_sweep, new_sweep) in enumerate(zip(old_sweep_runs, new_sweeps)):
        _check_run(problems, "%s sweep %d" % (row_id, position), row_id, old_sweep, new_sweep)
        old_children = old_sweep.get("trial_runs") or []
        new_children = _runs_of(benchmark_id, RUN_KIND_SWEEP_TRIAL, parent_position=position)
        if len(new_children) != len(old_children):
            problems.append("%s sweep %d: %d trial runs, expected %d" % (row_id, position, len(new_children), len(old_children)))
        for child_position, (old_child, new_child) in enumerate(zip(old_children, new_children)):
            label = "%s sweep %d trial %d" % (row_id, position, child_position)
            _check_run(problems, label, row_id, old_child, new_child)


def verify_row(evaluation_id):
    problems = []
    row = db.session.execute(SELECT_ROW, {"id": evaluation_id}).mappings().first()
    if row is None:
        return ["%s: row disappeared" % evaluation_id]
    if row["storage_version"] != CURRENT_STORAGE_VERSION:
        return ["%s: storage_version is %s" % (evaluation_id, row["storage_version"])]

    results = _parse(row["results_text"])
    replays = _parse(row["replays_text"])
    old_pending_run = replays.get("pending_run") if isinstance(replays, dict) else None
    if old_pending_run is not None and _parse(row["pending_run_text"]) != old_pending_run:
        problems.append("%s: pending_run differs" % evaluation_id)

    if not benchmark_store.has_benchmark_data(results, replays):
        return problems

    benchmark = Benchmark.query.filter_by(source_evaluation_id=evaluation_id, content_hash=None).first()
    if benchmark is None:
        return problems + ["%s: no benchmark was created" % evaluation_id]
    if row["benchmark_id"] != benchmark.id:
        problems.append("%s: benchmark_id does not point at its copied benchmark" % evaluation_id)
    if benchmark.summary != results:
        problems.append("%s: results differ" % evaluation_id)
    if benchmark.meta_dict() != benchmark_store.replay_meta(replays):
        problems.append("%s: replay meta differs" % evaluation_id)
    if benchmark.replays_shape != benchmark_store.replays_shape(replays):
        problems.append("%s: replays shape differs" % evaluation_id)

    old_runs, old_sweep_runs = benchmark_store.replay_lists(replays)
    _check_run_lists(problems, evaluation_id, benchmark.id, old_runs, old_sweep_runs)
    return problems


def all_row_ids():
    ids = db.session.execute(text("SELECT id FROM player_evaluations ORDER BY id")).scalars().all()
    db.session.commit()
    return ids


def verify_all(log=None):
    problems = []
    for index, evaluation_id in enumerate(all_row_ids(), start=1):
        problems.extend(verify_row(evaluation_id))
        db.session.expunge_all()
        db.session.rollback()
        if log is not None and index % 50 == 0:
            log("verified %d rows", index)
    return problems
