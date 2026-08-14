WAVE_PHASES = 2

META_KEYS = ("fps", "defenders", "view", "fov", "speed", "hull", "planet", "arena")


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

    wave = _wave_summary(runs)
    if wave:
        results.update(wave)

    reported = result.get("results")
    if isinstance(reported, dict):
        for key, value in reported.items():
            if value is not None:
                results[key] = value

    return results, replays


def _wave_summary(runs):
    waves = [run for run in runs if isinstance(run.get("stats"), dict) and "evaders" in run["stats"]]
    if not waves:
        return {}

    destroyed_per_trial = []
    evaders_per_trial = []
    detected_first = []
    detected_second = []
    held_first = 0
    held_second = 0
    held_both = 0

    for run in waves:
        stats = run["stats"]
        sent = int(stats.get("evaders", 0)) * WAVE_PHASES
        destroyed_per_trial.append(int(stats.get("destroyed", 0)))
        evaders_per_trial.append(sent)
        detected_first.append(int(stats.get("detected_first", 0)))
        detected_second.append(int(stats.get("detected_second", 0)))
        held_first += int(stats.get("held_first", 0))
        held_second += int(stats.get("held_second", 0))
        if run.get("outcome") == "win":
            held_both += 1

    trials = len(waves)
    destroyed = sum(destroyed_per_trial)
    sent = sum(evaders_per_trial)
    kill_rate = round(100.0 * destroyed / sent, 1) if sent else 0.0

    summary = {
        "success_rate": kill_rate,
        "evader_destroyed_rate": kill_rate,
        "evaders_destroyed": destroyed,
        "evaders_total": sent,
        "trials_held_rate": round(100.0 * held_both / trials, 1),
        "sequential_rate": round(100.0 * held_first / trials, 1),
        "simultaneous_rate": round(100.0 * held_second / trials, 1),
        "trial_destroyed": destroyed_per_trial,
        "trial_evaders": evaders_per_trial,
    }
    if any(detected_first) or any(detected_second):
        summary["trial_detected_first"] = detected_first
        summary["trial_detected_second"] = detected_second
        summary["sequential_detection_rate"] = round(100.0 * sum(detected_first) / trials, 1)
        summary["simultaneous_detection_rate"] = round(100.0 * sum(detected_second) / trials, 1)
    return summary


def _rate(times):
    if not times:
        return 0.0
    hits = sum(1 for value in times if value is not None and value >= 0.0)
    return round(100.0 * hits / len(times), 1)
