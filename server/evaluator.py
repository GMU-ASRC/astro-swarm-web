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
        db.session.commit()

        try:
            results = _run_godot(evaluation.algorithm, evaluation.n_max, evaluation.trials)
            evaluation.results = results
            evaluation.status = "done"
            evaluation.error = None
        except Exception as exc:
            evaluation.results = []
            evaluation.status = "failed"
            evaluation.error = str(exc)[:400]

        evaluation.completed_at = datetime.now(timezone.utc)
        db.session.commit()


def _run_godot(algorithm, n_max, trials):
    godot_bin = os.environ.get("GODOT_SERVER_BIN")
    if not godot_bin:
        raise RuntimeError("GODOT_SERVER_BIN is not configured")

    timeout = int(os.environ.get("EVAL_TIMEOUT_SECONDS", "1800"))

    with tempfile.TemporaryDirectory() as tmp:
        algorithm_path = os.path.join(tmp, "algorithm.json")
        output_path = os.path.join(tmp, "result.json")

        with open(algorithm_path, "w") as f:
            json.dump({"algorithm": algorithm}, f)

        cmd = [godot_bin, "--headless"]
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

        proc = subprocess.run(cmd, capture_output=True, text=True, timeout=timeout)

        if os.path.isfile(output_path):
            with open(output_path) as f:
                return json.load(f).get("results", [])

        for line in reversed(proc.stdout.splitlines()):
            line = line.strip()
            if line.startswith("{") and "results" in line:
                return json.loads(line).get("results", [])

        raise RuntimeError(f"benchmark produced no output (exit {proc.returncode}): {proc.stderr[-400:]}")
