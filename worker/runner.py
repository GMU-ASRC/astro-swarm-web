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


def _progress_done(line):
    try:
        done, _total = line.split()[1].split("/")
        return int(done)
    except (IndexError, ValueError):
        return None


def run_shard(shard, settings, report_progress, logger):
    algorithm = shard.get("algorithm", [])
    placements = shard.get("placements", [])
    trials = int(shard.get("trials", 100))
    config = shard.get("config", {})
    trial_start = int(shard.get("trial_start", 0))
    trial_count = int(shard.get("trial_count", 0))
    n_start = int(shard.get("n_start", 1))
    n_count = int(shard.get("n_count", 0))

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

    seed = int(config.get("seed", 987654321))
    sweep_max = int(config.get("sweep_max", 100))
    sweep_trials = int(config.get("sweep_trials", 1))
    match_seconds = int(config.get("match_seconds", 240))
    enemy_x = float(config.get("enemy_x", 1920))
    enemy_y = float(config.get("enemy_y", 40))

    with tempfile.TemporaryDirectory() as tmp:
        algorithm_path = os.path.join(tmp, "algorithm.json")
        placements_path = os.path.join(tmp, "placements.json")
        output_path = os.path.join(tmp, "result.json")
        with open(algorithm_path, "w") as f:
            json.dump({"algorithm": algorithm}, f)
        with open(placements_path, "w") as f:
            json.dump({"placements": placements}, f)

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
            f"--n-start={n_start}",
            f"--n-count={n_count}",
            f"--enemy-x={enemy_x}",
            f"--enemy-y={enemy_y}",
        ]

        proc = subprocess.Popen(cmd, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
        watchdog = threading.Timer(timeout, proc.kill)
        watchdog.start()
        tail = []
        last_report = 0.0
        cancelled = False
        try:
            for line in proc.stdout:
                line = line.strip()
                if not line:
                    continue
                tail.append(line)
                if len(tail) > 10:
                    tail.pop(0)
                if line.startswith("PROGRESS"):
                    count = _progress_done(line)
                    now = time.time()
                    if count is not None and now - last_report >= 1.0:
                        last_report = now
                        if report_progress(count):
                            cancelled = True
                            proc.kill()
                            break
            proc.wait()
        finally:
            watchdog.cancel()

        if cancelled:
            raise CancelledError()

        if not os.path.isfile(output_path):
            joined = " | ".join(tail[-5:])
            raise RuntimeError(f"benchmark shard produced no output (exit {proc.returncode}): {joined}")

        with open(output_path) as f:
            payload = json.load(f)

    return _pack_result(payload)


def _pack_result(payload):
    replays = payload.get("replays", {})
    meta = {key: replays.get(key) for key in ("fps", "defenders", "view", "fov", "planet", "arena")}

    runs = replays.get("runs", [])
    sweep_runs = replays.get("sweep_runs", [])
    for run in runs:
        raw_frames = run.pop("frames", [])
        run["frames_packed"] = _pack_frames(raw_frames)
    for run in sweep_runs:
        raw_frames = run.pop("frames", [])
        run["frames_packed"] = _pack_frames(raw_frames)

    return {"runs": runs, "sweep_runs": sweep_runs, "meta": meta}


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
