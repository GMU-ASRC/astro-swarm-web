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
    sweep = [_sweep_point(run) for run in sweep_runs]

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

    assault = _assault_summary(runs)
    if assault:
        results.update(assault)

    reported = result.get("results")
    if isinstance(reported, dict):
        for key, value in reported.items():
            if value is not None:
                results[key] = value

    return results, replays


# Levels 3 to 5 send a stream of evaders at one line, so the headline is the
# share destroyed rather than the share of trials won, and the run also has to
# account for the defenders it spent.
def _assault_summary(runs):
    assaults = [run for run in runs if isinstance(run.get("stats"), dict) and "resolved" in run["stats"]]
    if not assaults:
        return {}

    destroyed_per_trial = []
    resolved_per_trial = []
    breaches_per_trial = []
    lost_per_trial = []
    held = 0

    for run in assaults:
        stats = run["stats"]
        destroyed_per_trial.append(int(stats.get("destroyed", 0)))
        resolved_per_trial.append(int(stats.get("resolved", 0)))
        breaches_per_trial.append(int(stats.get("breaches", 0)))
        lost_per_trial.append(int(stats.get("lost", 0)))
        if run.get("outcome") == "win":
            held += 1

    trials = len(assaults)
    destroyed = sum(destroyed_per_trial)
    resolved = sum(resolved_per_trial)
    capture_rate = round(100.0 * destroyed / resolved, 1) if resolved else 0.0

    return {
        "success_rate": capture_rate,
        "evader_destroyed_rate": capture_rate,
        "evaders_destroyed": destroyed,
        "evaders_resolved": resolved,
        "breaches": sum(breaches_per_trial),
        "defenders_lost": sum(lost_per_trial),
        "risk": round(100.0 - capture_rate, 1),
        "trials_held_rate": round(100.0 * held / trials, 1),
        "trial_destroyed": destroyed_per_trial,
        "trial_resolved": resolved_per_trial,
        "trial_breaches": breaches_per_trial,
        "trial_lost": lost_per_trial,
    }


def _sweep_point(run):
    rate = run.get("capture_rate")
    if rate is None:
        rate = 100.0 if run.get("outcome") == "win" else 0.0
    rate = round(float(rate), 1)
    return {
        "n": run.get("n"),
        "success_rate": rate,
        "capture_rate": rate,
        "detection_rate": run.get("detection_rate"),
        "risk": round(100.0 - rate, 1),
    }


def _rate(times):
    if not times:
        return 0.0
    hits = sum(1 for value in times if value is not None and value >= 0.0)
    return round(100.0 * hits / len(times), 1)
