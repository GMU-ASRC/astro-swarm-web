import base64
import json
import os
import subprocess
import tempfile
import threading
import time
import zlib


class CancelledError(Exception):
    pass


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


def _progress_done(line):
    try:
        done, _total = line.split()[1].split("/")
        return int(done)
    except (IndexError, ValueError):
        return None


def run_benchmark(job, settings, report_progress, logger):
    algorithm = job.get("algorithm", [])
    placements = job.get("placements", [])
    trials = int(job.get("trials", 100))
    config = job.get("config", {})

    godot_bin = settings["godot_bin"]
    if not godot_bin:
        raise RuntimeError("GODOT_SERVER_BIN is not configured")
    if not os.path.isfile(godot_bin):
        raise RuntimeError(f"GODOT_SERVER_BIN not found at {godot_bin}")
    if not os.access(godot_bin, os.X_OK):
        raise RuntimeError(f"GODOT_SERVER_BIN not executable: {godot_bin}")

    main_pack = settings.get("godot_pck")
    fixed_fps = str(settings.get("fixed_fps", 60))
    timeout = int(settings.get("timeout_seconds", 1800))
    max_jobs = max(1, int(settings.get("max_jobs", 1)))

    seed = int(config.get("seed", 987654321))
    sweep_max = int(config.get("sweep_max", 100))
    sweep_trials = int(config.get("sweep_trials", 1))
    match_seconds = int(config.get("match_seconds", 240))
    enemy_x = float(config.get("enemy_x", 1920))
    enemy_y = float(config.get("enemy_y", 40))

    total_work = max(1, trials + sweep_max * sweep_trials)
    jobs = max(1, min(max_jobs, total_work))
    placement_ranges = _chunks(trials, jobs)
    sweep_ranges = _chunks(sweep_max, jobs)

    with tempfile.TemporaryDirectory() as tmp:
        algorithm_path = os.path.join(tmp, "algorithm.json")
        placements_path = os.path.join(tmp, "placements.json")
        with open(algorithm_path, "w") as f:
            json.dump({"algorithm": algorithm}, f)
        with open(placements_path, "w") as f:
            json.dump({"placements": placements}, f)

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
                f"--seed={seed}",
                f"--n-max={sweep_max}",
                f"--sweep-trials={sweep_trials}",
                f"--match-seconds={match_seconds}",
                f"--trial-start={trial_start}",
                f"--trial-count={trial_count}",
                f"--n-start={sweep_start + 1}",
                f"--n-count={sweep_count}",
                f"--enemy-x={enemy_x}",
                f"--enemy-y={enemy_y}",
            ]
            shards.append({"cmd": cmd, "output": output_path, "done": 0, "tail": []})

        logger.info("job %s: split into %d shard(s) (max_jobs=%d)", job.get("id"), len(shards), max_jobs)

        progress_lock = threading.Lock()
        procs_lock = threading.Lock()
        procs = []

        def run_shard(shard):
            proc = subprocess.Popen(shard["cmd"], stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
            shard["proc"] = proc
            with procs_lock:
                procs.append(proc)
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
            with procs_lock:
                for proc in procs:
                    try:
                        proc.kill()
                    except Exception:
                        pass

        cancelled = False
        watchdog = threading.Timer(timeout, kill_all)
        watchdog.start()
        try:
            while any(thread.is_alive() for thread in threads):
                with progress_lock:
                    completed = sum(shard["done"] for shard in shards)
                if report_progress(completed / total_work):
                    cancelled = True
                    kill_all()
                    break
                time.sleep(1.0)
            for thread in threads:
                thread.join()
        finally:
            watchdog.cancel()

        if cancelled:
            raise CancelledError()

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

    for run in runs:
        raw_frames = run.pop("frames", [])
        run["frames_packed"] = _pack_frames(raw_frames)
    for run in sweep_runs:
        raw_frames = run.pop("frames", [])
        run["frames_packed"] = _pack_frames(raw_frames)

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


def _pack_frames(frames):
    if not frames:
        return ""
    delta_encoded = [frames[0]]
    for i in range(1, len(frames)):
        prev = frames[i - 1]
        curr = frames[i]
        delta = [curr[j] - prev[j] for j in range(min(len(curr), len(prev)))]
        delta_encoded.append(delta)
    raw = json.dumps(delta_encoded, separators=(",", ":"))
    return base64.b64encode(zlib.compress(raw.encode(), 9)).decode("ascii")
