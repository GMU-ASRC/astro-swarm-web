META_KEYS = ("fps", "defenders", "view", "fov", "speed", "planet", "arena")


def summarize(result):
    # One worker runs a whole evaluation, so a job comes back as a single result.
    result = result if isinstance(result, dict) else {}
    meta_source = result.get("meta") or {}
    meta = {key: meta_source.get(key) for key in META_KEYS}

    runs = list(result.get("runs") or [])
    sweep_runs = list(result.get("sweep_runs") or [])
    runs.sort(key=lambda run: run.get("trial", 0))
    sweep_runs.sort(key=lambda run: run.get("n", 0))

    outcomes = [run.get("outcome") for run in runs]
    detection_times = [run.get("detection_time", -1.0) for run in runs]
    capture_times = [run.get("capture_time", -1.0) for run in runs]
    goal_times = [run.get("goal_time", -1.0) for run in runs]
    wins = sum(1 for outcome in outcomes if outcome == "win")
    total = max(1, len(outcomes))
    success_rate = round(100.0 * wins / total, 1)
    sweep = [
        {"n": run.get("n"), "success_rate": run.get("capture_rate", 100.0 if run.get("outcome") == "win" else 0.0)}
        for run in sweep_runs
    ]

    replays = dict(meta)
    replays["runs"] = runs
    replays["sweep_runs"] = sweep_runs

    results = {
        "trials": len(outcomes),
        "success_rate": success_rate,
        "detection_rate": _rate(detection_times),
        "capture_rate": _rate(capture_times),
        "outcomes": outcomes,
        "detection_times": detection_times,
        "capture_times": capture_times,
        "goal_times": goal_times,
        "sweep": sweep,
    }
    stats = next((run.get("stats") for run in runs if run.get("stats")), None)
    if stats:
        results["stats"] = stats
    return results, replays


def _rate(times):
    if not times:
        return 0.0
    hits = sum(1 for value in times if value is not None and value >= 0.0)
    return round(100.0 * hits / len(times), 1)
