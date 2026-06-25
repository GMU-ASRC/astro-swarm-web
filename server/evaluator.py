import json
import logging
import os
import subprocess
import tempfile
import threading
import time
from datetime import datetime, timezone

from app_settings import get_max_jobs
from config import Config
from database import db
from models import PlayerEvaluation

logger = logging.getLogger(__name__)


def run_evaluation_async(app, evaluation_id):
    thread = threading.Thread(target=_run, args=(app, evaluation_id), daemon=True)
    thread.start()


def _run(app, evaluation_id):
    with app.app_context():
        evaluation = db.session.get(PlayerEvaluation, evaluation_id)
        if evaluation is None:
            return

        evaluation.status = "running"
        evaluation.progress = 0.0
        db.session.commit()

        def on_progress(fraction):
            evaluation.progress = round(min(0.99, fraction), 3)
            db.session.commit()

        logger.info("evaluation %s: starting (defenders=%s trials=%s)", evaluation_id, len(evaluation.placements or []), evaluation.trials)
        try:
            data = _run_godot(evaluation.algorithm, evaluation.placements or [], evaluation.trials, on_progress)
            evaluation.results = data.get("results", {})
            evaluation.replays = data.get("replays", {})
            evaluation.status = "done"
            evaluation.progress = 1.0
            evaluation.error = None
            run_count = len((evaluation.replays or {}).get("runs", []))
            logger.info("evaluation %s: done (success_rate=%s, %d replays)", evaluation_id, (evaluation.results or {}).get("success_rate"), run_count)
        except Exception as exc:
            evaluation.results = {}
            evaluation.status = "failed"
            evaluation.error = str(exc)[:400]
            logger.error("evaluation %s: failed: %s", evaluation_id, exc)

        evaluation.completed_at = datetime.now(timezone.utc)
        db.session.commit()


def _chunks(total, jobs):
    ranges = []
    base = total // jobs
    extra = total % jobs
    start = 0
    for index in range(jobs):
        count = base + (1 if index < extra else 0)
        ranges.append((start, count))
        start += count
    return ranges


def _run_godot(algorithm, placements, trials, on_progress):
    godot_bin = os.environ.get("GODOT_SERVER_BIN")
    if not godot_bin:
        raise RuntimeError("GODOT_SERVER_BIN is not configured")
    if not os.path.isfile(godot_bin):
        raise RuntimeError(f"GODOT_SERVER_BIN not found at {godot_bin}")
    if not os.access(godot_bin, os.X_OK):
        raise RuntimeError(f"GODOT_SERVER_BIN not executable: {godot_bin}")

    timeout = int(os.environ.get("EVAL_TIMEOUT_SECONDS", "1800"))
    sweep_max = Config.EVAL_SWEEP_MAX
    sweep_trials = Config.EVAL_SWEEP_TRIALS
    total_work = max(1, trials + sweep_max * sweep_trials)
    jobs = max(1, min(get_max_jobs(), total_work))

    placement_ranges = _chunks(trials, jobs)
    sweep_ranges = _chunks(sweep_max, jobs)

    with tempfile.TemporaryDirectory() as tmp:
        algorithm_path = os.path.join(tmp, "algorithm.json")
        placements_path = os.path.join(tmp, "placements.json")

        with open(algorithm_path, "w") as f:
            json.dump({"algorithm": algorithm}, f)
        with open(placements_path, "w") as f:
            json.dump({"placements": placements}, f)

        fixed_fps = os.environ.get("EVAL_FIXED_FPS", "60")
        main_pack = os.environ.get("GODOT_PCK")

        shards = []
        for index in range(jobs):
            trial_start, trial_count = placement_ranges[index]
            sweep_start, sweep_count = sweep_ranges[index]
            if trial_count == 0 and sweep_count == 0:
                continue
            output_path = os.path.join(tmp, f"result_{index}.json")
            cmd = [godot_bin, "--headless", "--fixed-fps", fixed_fps]
            if main_pack:
                cmd += ["--main-pack", main_pack]
            cmd += [
                "--",
                "--bench",
                f"--algorithm={algorithm_path}",
                f"--placements={placements_path}",
                f"--out={output_path}",
                f"--trials={trials}",
                f"--seed={Config.EVAL_SEED}",
                f"--n-max={sweep_max}",
                f"--sweep-trials={sweep_trials}",
                f"--spawn-points={Config.EVAL_SPAWN_POINTS}",
                f"--match-seconds={Config.EVAL_MATCH_CAP_SECONDS}",
                f"--trial-start={trial_start}",
                f"--trial-count={trial_count}",
                f"--n-start={sweep_start + 1}",
                f"--n-count={sweep_count}",
            ]
            shards.append({"cmd": cmd, "output": output_path, "done": 0})

        logger.info("evaluation split into %d parallel shard(s)", len(shards))

        progress_lock = threading.Lock()

        def run_shard(shard):
            proc = subprocess.Popen(shard["cmd"], stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
            shard["proc"] = proc
            shard["tail"] = []
            for line in proc.stdout:
                line = line.strip()
                if not line:
                    continue
                shard["tail"].append(line)
                if len(shard["tail"]) > 10:
                    shard["tail"].pop(0)
                if line.startswith("PROGRESS"):
                    count = _progress_done(line)
                    if count is not None:
                        with progress_lock:
                            shard["done"] = count
            proc.wait()

        threads = [threading.Thread(target=run_shard, args=(shard,), daemon=True) for shard in shards]
        for thread in threads:
            thread.start()

        def kill_all():
            for shard in shards:
                proc = shard.get("proc")
                if proc is not None:
                    proc.kill()

        watchdog = threading.Timer(timeout, kill_all)
        watchdog.start()
        try:
            while any(thread.is_alive() for thread in threads):
                with progress_lock:
                    completed = sum(shard["done"] for shard in shards)
                on_progress(completed / total_work)
                time.sleep(1.0)
            for thread in threads:
                thread.join()
        finally:
            watchdog.cancel()

        parts = []
        for shard in shards:
            if not os.path.isfile(shard["output"]):
                proc = shard.get("proc")
                code = proc.returncode if proc is not None else "?"
                tail = " | ".join(shard.get("tail", [])[-5:])
                raise RuntimeError(f"benchmark shard produced no output (exit {code}): {tail}")
            with open(shard["output"]) as f:
                parts.append(json.load(f))

        return _merge(parts)


def _merge(parts):
    runs = []
    sweep_runs = []
    meta = {}
    for part in parts:
        replays = part.get("replays", {})
        if not meta and replays:
            meta = {key: replays.get(key) for key in ("fps", "defenders", "view", "fov", "planet", "arena")}
        runs.extend(replays.get("runs", []))
        sweep_runs.extend(replays.get("sweep_runs", []))

    runs.sort(key=lambda run: run.get("trial", 0))
    sweep_runs.sort(key=lambda run: run.get("n", 0))

    outcomes = [run.get("outcome") for run in runs]
    detection_times = [run.get("detection_time", -1.0) for run in runs]
    capture_times = [run.get("capture_time", -1.0) for run in runs]
    wins = sum(1 for outcome in outcomes if outcome == "win")
    total = max(1, len(outcomes))
    success_rate = round(100.0 * wins / total, 1)
    sweep = [
        {"n": run.get("n"), "success_rate": 100.0 if run.get("outcome") == "win" else 0.0}
        for run in sweep_runs
    ]

    replays = dict(meta)
    replays["runs"] = runs
    replays["sweep_runs"] = sweep_runs

    return {
        "trials": len(outcomes),
        "results": {
            "trials": len(outcomes),
            "success_rate": success_rate,
            "outcomes": outcomes,
            "detection_times": detection_times,
            "capture_times": capture_times,
            "sweep": sweep,
        },
        "replays": replays,
    }


def _progress_done(line):
    try:
        done, _total = line.split()[1].split("/")
        return int(done)
    except (IndexError, ValueError):
        return None
