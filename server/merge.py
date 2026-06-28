META_KEYS = ("fps", "defenders", "view", "fov", "planet", "arena")


def merge_shards(parts):
    runs = []
    sweep_runs = []
    meta = {}
    for part in parts:
        if not part:
            continue
        if not meta and part.get("meta"):
            meta = {key: part["meta"].get(key) for key in META_KEYS}
        runs.extend(part.get("runs", []))
        sweep_runs.extend(part.get("sweep_runs", []))

    runs.sort(key=lambda run: run.get("trial", 0))
    sweep_runs.sort(key=lambda run: run.get("n", 0))

    outcomes = [run.get("outcome") for run in runs]
    detection_times = [run.get("detection_time", -1.0) for run in runs]
    capture_times = [run.get("capture_time", -1.0) for run in runs]
    wins = sum(1 for outcome in outcomes if outcome == "win")
    total = max(1, len(outcomes))
    success_rate = round(100.0 * wins / total, 1)
    sweep = [
        {"n": run.get("n"), "success_rate": run.get("detection_rate", 100.0 if run.get("outcome") == "win" else 0.0)}
        for run in sweep_runs
    ]

    replays = dict(meta)
    replays["runs"] = runs
    replays["sweep_runs"] = sweep_runs

    results = {
        "trials": len(outcomes),
        "success_rate": success_rate,
        "outcomes": outcomes,
        "detection_times": detection_times,
        "capture_times": capture_times,
        "sweep": sweep,
    }
    return results, replays
