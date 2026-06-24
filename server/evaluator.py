import json
import os
import subprocess
import tempfile
import threading
from datetime import datetime, timezone

from database import db
from models import PlayerEvaluation


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

        try:
            results = _run_godot(evaluation.algorithm, evaluation.n_max, evaluation.trials, on_progress)
            evaluation.results = results
            evaluation.status = "done"
            evaluation.progress = 1.0
            evaluation.error = None
        except Exception as exc:
            evaluation.results = []
            evaluation.status = "failed"
            evaluation.error = str(exc)[:400]

        evaluation.completed_at = datetime.now(timezone.utc)
        db.session.commit()


def _run_godot(algorithm, n_max, trials, on_progress):
    godot_bin = os.environ.get("GODOT_SERVER_BIN")
    if not godot_bin:
        raise RuntimeError("GODOT_SERVER_BIN is not configured")

    timeout = int(os.environ.get("EVAL_TIMEOUT_SECONDS", "1800"))

    with tempfile.TemporaryDirectory() as tmp:
        algorithm_path = os.path.join(tmp, "algorithm.json")
        output_path = os.path.join(tmp, "result.json")

        with open(algorithm_path, "w") as f:
            json.dump({"algorithm": algorithm}, f)

        fixed_fps = os.environ.get("EVAL_FIXED_FPS", "60")
        cmd = [godot_bin, "--headless", "--fixed-fps", fixed_fps]
        main_pack = os.environ.get("GODOT_PCK")
        if main_pack:
            cmd += ["--main-pack", main_pack]
        cmd += [
            "--",
            "--bench",
            f"--algorithm={algorithm_path}",
            f"--out={output_path}",
            f"--nmax={n_max}",
            f"--trials={trials}",
        ]

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
                return json.load(f).get("results", [])
        if result_line:
            return json.loads(result_line).get("results", [])

        raise RuntimeError(f"benchmark produced no output (exit {proc.returncode}): {' | '.join(recent[-5:])}")


def _parse_progress(line, on_progress):
    # Expected: "PROGRESS <done>/<total> ..."
    try:
        done, total = line.split()[1].split("/")
        on_progress(int(done) / int(total))
    except (IndexError, ValueError, ZeroDivisionError):
        pass
