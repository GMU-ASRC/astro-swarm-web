import logging
import os
import socket
import threading
import time
import uuid

import requests

try:
    import psutil
except ImportError:
    psutil = None

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
    "timeout_seconds": int(os.environ.get("EVAL_TIMEOUT_SECONDS", "3600")),
}

HEADERS = {"X-API-Key": API_SECRET_KEY}

_state_lock = threading.Lock()
_active = {}
_current_max_jobs = max(1, int(os.environ.get("WORKER_MAX_JOBS", "4")))


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


def _safe_post(path, payload, timeout=30):
    try:
        return _post(path, payload, timeout=timeout)
    except requests.RequestException as exc:
        logger.warning("post %s error: %s", path, exc)
        return None


def _set_max_jobs(value):
    global _current_max_jobs
    if value:
        with _state_lock:
            _current_max_jobs = max(1, int(value))


def register():
    while True:
        try:
            resp = _post("/api/worker/register", {
                "worker_id": WORKER_ID,
                "name": WORKER_NAME,
                "hostname": socket.gethostname(),
                "max_jobs": _current_max_jobs,
            }, timeout=30)
            if resp.ok:
                _set_max_jobs(resp.json().get("max_jobs"))
                logger.info("registered as %s (%s), max_jobs=%d", WORKER_NAME, WORKER_ID, _current_max_jobs)
                return
            logger.warning("register failed: %s %s", resp.status_code, resp.text[:200])
        except requests.RequestException as exc:
            logger.warning("register error: %s", exc)
        time.sleep(POLL_SECONDS)


def claim(slots):
    resp = _post("/api/worker/claim", {"worker_id": WORKER_ID, "slots": slots}, timeout=30)
    if not resp.ok:
        logger.warning("claim failed: %s %s", resp.status_code, resp.text[:200])
        return None
    return resp.json()


def collect_system_stats():
    if psutil is None:
        return {}
    try:
        memory = psutil.virtual_memory()
        disk = psutil.disk_usage("/")
        stats = {
            "cpu_percent": round(psutil.cpu_percent(interval=None), 1),
            "cpu_count": psutil.cpu_count(logical=True),
            "memory_percent": round(memory.percent, 1),
            "memory_used_mb": round(memory.used / (1024 * 1024)),
            "memory_total_mb": round(memory.total / (1024 * 1024)),
            "disk_percent": round(disk.percent, 1),
            "disk_used_gb": round(disk.used / (1024 ** 3), 1),
            "disk_total_gb": round(disk.total / (1024 ** 3), 1),
        }
        if hasattr(os, "getloadavg"):
            stats["load_avg_1m"] = round(os.getloadavg()[0], 2)
        return stats
    except Exception as exc:
        logger.warning("could not collect system stats: %s", exc)
        return {}


def heartbeat():
    with _state_lock:
        active = list(_active.values())
    status = "busy" if active else "idle"
    current_job = active[0] if active else None
    _safe_post("/api/worker/heartbeat", {
        "worker_id": WORKER_ID,
        "status": status,
        "current_job": current_job,
        "system_stats": collect_system_stats(),
    })


def godot_ready():
    binary = SETTINGS.get("godot_bin")
    return bool(binary) and os.path.isfile(binary)


def run_shard(shard):
    shard_id = shard.get("shard_id")
    eval_id = shard.get("evaluation_id")

    def report_progress(done_units, stage=None):
        payload = {"worker_id": WORKER_ID, "done": done_units}
        if stage:
            payload["stage"] = stage
        resp = _safe_post(f"/api/worker/shards/{shard_id}/progress", payload)
        if resp is not None and resp.ok:
            return bool(resp.json().get("cancel"))
        return False

    try:
        result = runner.run_shard(shard, SETTINGS, report_progress, logger)
    except runner.CancelledError:
        logger.info("shard %s (job %s): cancelled", shard_id, eval_id)
        return
    except Exception as exc:
        logger.error("shard %s (job %s): failed: %s", shard_id, eval_id, exc)
        _safe_post(f"/api/worker/shards/{shard_id}/fail", {"worker_id": WORKER_ID, "error": str(exc)[:400]})
        return

    resp = _safe_post(f"/api/worker/shards/{shard_id}/result", {
        "worker_id": WORKER_ID,
        "result": result,
    }, timeout=300)
    if resp is not None and resp.ok:
        logger.info("shard %s (job %s): done", shard_id, eval_id)
    elif resp is not None:
        logger.error("shard %s: result post failed: %s %s", shard_id, resp.status_code, resp.text[:200])


def _spawn(shard):
    shard_id = shard.get("shard_id")

    def worker_thread():
        try:
            run_shard(shard)
        finally:
            with _state_lock:
                _active.pop(shard_id, None)

    thread = threading.Thread(target=worker_thread, daemon=True)
    with _state_lock:
        _active[shard_id] = shard.get("evaluation_id")
    thread.start()


def main():
    logger.info("worker starting; server=%s", SERVER_URL)
    register()
    while True:
        try:
            if not godot_ready():
                heartbeat()
                try:
                    binary, pck = godot_release.ensure_server_build()
                    SETTINGS["godot_bin"] = binary
                    SETTINGS["godot_pck"] = pck
                except Exception as exc:
                    logger.warning("could not prepare server build: %s", exc)
                    time.sleep(POLL_SECONDS)
                    continue

            with _state_lock:
                free = _current_max_jobs - len(_active)
            if free > 0:
                response = claim(free)
                if response is None:
                    time.sleep(POLL_SECONDS)
                    continue
                if response.get("known") is False:
                    logger.info("server does not know this worker; re-registering")
                    register()
                    continue
                _set_max_jobs(response.get("max_jobs"))
                if not response.get("enabled", True):
                    heartbeat()
                    time.sleep(POLL_SECONDS)
                    continue
                for shard in response.get("shards", []):
                    _spawn(shard)

            heartbeat()
            time.sleep(POLL_SECONDS)
        except requests.RequestException as exc:
            logger.warning("loop error: %s", exc)
            time.sleep(POLL_SECONDS)


if __name__ == "__main__":
    main()
