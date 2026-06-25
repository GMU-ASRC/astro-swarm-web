import json
import logging
import os
import subprocess
import tempfile
import threading
from datetime import datetime, timezone

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


def _run_godot(algorithm, placements, trials, on_progress):
    godot_bin = os.environ.get("GODOT_SERVER_BIN")
    if not godot_bin:
        raise RuntimeError("GODOT_SERVER_BIN is not configured")
    if not os.path.isfile(godot_bin):
        raise RuntimeError(f"GODOT_SERVER_BIN not found at {godot_bin}")
    if not os.access(godot_bin, os.X_OK):
        raise RuntimeError(f"GODOT_SERVER_BIN not executable: {godot_bin}")

    timeout = int(os.environ.get("EVAL_TIMEOUT_SECONDS", "1800"))

    with tempfile.TemporaryDirectory() as tmp:
        algorithm_path = os.path.join(tmp, "algorithm.json")
        placements_path = os.path.join(tmp, "placements.json")
        output_path = os.path.join(tmp, "result.json")

        with open(algorithm_path, "w") as f:
            json.dump({"algorithm": algorithm}, f)
        with open(placements_path, "w") as f:
            json.dump({"placements": placements}, f)

        fixed_fps = os.environ.get("EVAL_FIXED_FPS", "60")
        cmd = [godot_bin, "--headless", "--fixed-fps", fixed_fps]
        main_pack = os.environ.get("GODOT_PCK")
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
            f"--n-max={Config.EVAL_SWEEP_MAX}",
            f"--sweep-trials={Config.EVAL_SWEEP_TRIALS}",
            f"--spawn-points={Config.EVAL_SPAWN_POINTS}",
            f"--match-seconds={Config.EVAL_MATCH_CAP_SECONDS}",
        ]

        logger.info("running: %s", " ".join(cmd))
        proc = subprocess.Popen(cmd, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
        watchdog = threading.Timer(timeout, proc.kill)
        watchdog.start()

        result_line = None
        recent = []
        try:
            for line in proc.stdout:
                line = line.strip()
                if not line:
                    continue
                recent.append(line)
                if len(recent) > 20:
                    recent.pop(0)
                if line.startswith("PROGRESS"):
                    _parse_progress(line, on_progress)
                elif line.startswith("{") and "results" in line:
                    result_line = line
            proc.wait()
        finally:
            watchdog.cancel()

        if os.path.isfile(output_path):
            with open(output_path) as f:
                return json.load(f)
        if result_line:
            return json.loads(result_line)

        raise RuntimeError(f"benchmark produced no output (exit {proc.returncode}): {' | '.join(recent[-5:])}")


def _parse_progress(line, on_progress):
    # Expected: "PROGRESS <done>/<total> ..."
    try:
        done, total = line.split()[1].split("/")
        on_progress(int(done) / int(total))
    except (IndexError, ValueError, ZeroDivisionError):
        pass
