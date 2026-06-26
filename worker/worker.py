import logging
import os
import socket
import time
import uuid

import requests

import godot_release
import runner

logging.basicConfig(level=logging.INFO, format="%(asctime)s [%(levelname)s] %(message)s")
logger = logging.getLogger("worker")

SERVER_URL = os.environ.get("SERVER_URL", "http://server:5050").rstrip("/")
API_SECRET_KEY = os.environ.get("API_SECRET_KEY", "dev_secret_key")
POLL_SECONDS = float(os.environ.get("WORKER_POLL_SECONDS", "3"))
ID_FILE = os.environ.get("WORKER_ID_FILE", "/app/worker_id")

SETTINGS = {
    "godot_bin": None,
    "godot_pck": None,
    "fixed_fps": os.environ.get("EVAL_FIXED_FPS", "60"),
    "timeout_seconds": int(os.environ.get("EVAL_TIMEOUT_SECONDS", "1800")),
    "max_jobs": max(1, int(os.environ.get("WORKER_MAX_JOBS", "4"))),
}

HEADERS = {"X-API-Key": API_SECRET_KEY}


def _load_worker_id():
    try:
        with open(ID_FILE) as f:
            value = f.read().strip()
            if value:
                return value
    except OSError:
        pass
    value = str(uuid.uuid4())
    try:
        with open(ID_FILE, "w") as f:
            f.write(value)
    except OSError as exc:
        logger.warning("could not persist worker id (%s); using ephemeral id", exc)
    return value


WORKER_ID = _load_worker_id()
WORKER_NAME = os.environ.get("WORKER_NAME", socket.gethostname())


def _post(path, payload, timeout=120):
    return requests.post(f"{SERVER_URL}{path}", json=payload, headers=HEADERS, timeout=timeout)


def register():
    while True:
        try:
            resp = _post("/api/worker/register", {
                "worker_id": WORKER_ID,
                "name": WORKER_NAME,
                "hostname": socket.gethostname(),
                "max_jobs": SETTINGS["max_jobs"],
            }, timeout=30)
            if resp.ok:
                logger.info("registered as %s (%s), max_jobs=%d", WORKER_NAME, WORKER_ID, SETTINGS["max_jobs"])
                return
            logger.warning("register failed: %s %s", resp.status_code, resp.text[:200])
        except requests.RequestException as exc:
            logger.warning("register error: %s", exc)
        time.sleep(POLL_SECONDS)


def claim():
    resp = _post("/api/worker/claim", {"worker_id": WORKER_ID}, timeout=30)
    if not resp.ok:
        logger.warning("claim failed: %s %s", resp.status_code, resp.text[:200])
        return None
    return resp.json()


def heartbeat(status, current_job=None):
    try:
        _post("/api/worker/heartbeat", {
            "worker_id": WORKER_ID,
            "status": status,
            "current_job": current_job,
        }, timeout=30)
    except requests.RequestException:
        pass


def godot_ready():
    binary = SETTINGS.get("godot_bin")
    return bool(binary) and os.path.isfile(binary)


def run_job(job, max_jobs):
    job_id = job.get("id")
    settings = dict(SETTINGS)
    settings["max_jobs"] = max(1, int(max_jobs or SETTINGS["max_jobs"]))
    logger.info("job %s: claimed (trials=%s, defenders=%s, max_jobs=%d)", job_id, job.get("trials"), len(job.get("placements", [])), settings["max_jobs"])

    def report_progress(fraction):
        try:
            resp = _post(f"/api/worker/jobs/{job_id}/progress", {
                "worker_id": WORKER_ID,
                "progress": fraction,
            }, timeout=30)
            if resp.ok:
                return bool(resp.json().get("cancel"))
        except requests.RequestException as exc:
            logger.warning("job %s: progress post error: %s", job_id, exc)
        return False

    try:
        result = runner.run_benchmark(job, settings, report_progress, logger)
    except runner.CancelledError:
        logger.info("job %s: cancelled", job_id)
        _safe_post(f"/api/worker/jobs/{job_id}/fail", {"worker_id": WORKER_ID, "error": "cancelled"})
        return
    except Exception as exc:
        logger.error("job %s: failed: %s", job_id, exc)
        _safe_post(f"/api/worker/jobs/{job_id}/fail", {"worker_id": WORKER_ID, "error": str(exc)[:400]})
        return

    try:
        resp = _post(f"/api/worker/jobs/{job_id}/result", {
            "worker_id": WORKER_ID,
            "results": result["results"],
            "replays": result["replays"],
        }, timeout=300)
        if resp.ok:
            rate = result["results"].get("success_rate")
            logger.info("job %s: done (success_rate=%s)", job_id, rate)
        else:
            logger.error("job %s: result post failed: %s %s", job_id, resp.status_code, resp.text[:200])
    except requests.RequestException as exc:
        logger.error("job %s: result post error: %s", job_id, exc)


def _safe_post(path, payload):
    try:
        _post(path, payload, timeout=30)
    except requests.RequestException as exc:
        logger.warning("post %s error: %s", path, exc)


def main():
    logger.info("worker starting; server=%s", SERVER_URL)
    register()
    while True:
        try:
            if not godot_ready():
                heartbeat("preparing")
                try:
                    binary, pck = godot_release.ensure_server_build()
                    SETTINGS["godot_bin"] = binary
                    SETTINGS["godot_pck"] = pck
                except Exception as exc:
                    logger.warning("could not prepare server build: %s", exc)
                    time.sleep(POLL_SECONDS)
                    continue
            response = claim()
            if response is None:
                time.sleep(POLL_SECONDS)
                continue
            if response.get("known") is False:
                logger.info("server does not know this worker; re-registering")
                register()
                continue
            if not response.get("enabled", True):
                time.sleep(POLL_SECONDS)
                continue
            job = response.get("job")
            if job:
                run_job(job, response.get("max_jobs"))
            else:
                time.sleep(POLL_SECONDS)
        except requests.RequestException as exc:
            logger.warning("loop error: %s", exc)
            time.sleep(POLL_SECONDS)


if __name__ == "__main__":
    main()
