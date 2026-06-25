from io import BytesIO

import matplotlib

matplotlib.use("Agg")

import matplotlib.pyplot as plt


def _caption(username, level_id, eval_id, date_label):
    return f"{username}  ·  level: {level_id}  ·  id: {eval_id[:8]}  ·  {date_label}"


def _outcome_counts(outcomes):
    counts = {"win": 0, "lose": 0, "timeout": 0}
    for outcome in outcomes:
        key = outcome if outcome in counts else "timeout"
        counts[key] += 1
    return counts


def _save(fig):
    buffer = BytesIO()
    fig.savefig(buffer, format="png", dpi=110, facecolor=fig.get_facecolor())
    plt.close(fig)
    buffer.seek(0)
    return buffer.getvalue()


def render_line_png(outcomes, username, level_id, eval_id, date_label):
    wins = 0
    xs = []
    ys = []
    for index, outcome in enumerate(outcomes):
        if outcome == "win":
            wins += 1
        xs.append(index + 1)
        ys.append(100.0 * wins / (index + 1))

    fig, ax = plt.subplots(figsize=(6.4, 3.8))
    ax.plot(xs, ys, color="#1f77b4", linewidth=2)
    ax.set_title("Cumulative Detection Rate")
    ax.set_xlabel("Trial")
    ax.set_ylabel("Detection Rate (%)")
    ax.set_ylim(0, 100)
    ax.grid(True, color="#e5e7eb")
    fig.text(0.5, 0.005, _caption(username, level_id, eval_id, date_label), ha="center", fontsize=8, color="#6b7280")
    fig.tight_layout(rect=(0, 0.04, 1, 1))
    return _save(fig)


def render_sweep_png(sweep, username, level_id, eval_id, date_label):
    points = sorted(sweep, key=lambda point: point.get("n", 0))
    xs = [point.get("n", 0) for point in points]
    ys = [point.get("success_rate", 0.0) for point in points]

    fig, ax = plt.subplots(figsize=(6.4, 3.8))
    ax.plot(xs, ys, color="#1f77b4", linewidth=2)
    ax.set_title("Detection Rate vs Number of Defenders")
    ax.set_xlabel("Defenders in ring (n)")
    ax.set_ylabel("Detection Rate (%)")
    ax.set_ylim(0, 100)
    if xs:
        ax.set_xlim(min(xs), max(xs))
    ax.grid(True, color="#e5e7eb")
    fig.text(0.5, 0.005, _caption(username, level_id, eval_id, date_label), ha="center", fontsize=8, color="#6b7280")
    fig.tight_layout(rect=(0, 0.04, 1, 1))
    return _save(fig)


def render_times_png(detection_times, capture_times, username, level_id, eval_id, date_label):
    count = max(len(detection_times), len(capture_times))
    trials = list(range(1, count + 1))

    def clamp(values):
        out = []
        for index in range(count):
            value = values[index] if index < len(values) else -1.0
            out.append(value if value is not None and value >= 0 else 0.0)
        return out

    detections = clamp(detection_times)
    captures = clamp(capture_times)
    width = 0.42
    left = [trial - width / 2 for trial in trials]
    right = [trial + width / 2 for trial in trials]

    fig, ax = plt.subplots(figsize=(7.2, 3.8))
    ax.bar(left, detections, width=width, color="#4ade80", label="Detection time")
    ax.bar(right, captures, width=width, color="#f87171", label="Capture time")
    ax.set_title("Detection and Capture Times per Trial")
    ax.set_xlabel("Trial")
    ax.set_ylabel("Time (s)")
    ax.grid(True, axis="y", color="#e5e7eb")
    ax.legend(loc="upper right", fontsize=8)
    fig.text(0.5, 0.005, _caption(username, level_id, eval_id, date_label), ha="center", fontsize=8, color="#6b7280")
    fig.tight_layout(rect=(0, 0.04, 1, 1))
    return _save(fig)


def render_bar_png(outcomes, username, level_id, eval_id, date_label):
    counts = _outcome_counts(outcomes)
    total = max(1, len(outcomes))
    labels = ["Intercept", "Planet hit", "Timeout"]
    values = [
        100.0 * counts["win"] / total,
        100.0 * counts["lose"] / total,
        100.0 * counts["timeout"] / total,
    ]
    colors = ["#4ade80", "#f87171", "#fbbf24"]

    fig, ax = plt.subplots(figsize=(6.4, 3.8))
    ax.bar(labels, values, color=colors)
    ax.set_title("Outcome Breakdown")
    ax.set_ylabel("% of trials")
    ax.set_ylim(0, 100)
    ax.grid(True, axis="y", color="#e5e7eb")
    for index, value in enumerate(values):
        ax.text(index, value + 1.5, f"{value:.0f}%", ha="center", fontsize=9, color="#374151")
    fig.text(0.5, 0.005, _caption(username, level_id, eval_id, date_label), ha="center", fontsize=8, color="#6b7280")
    fig.tight_layout(rect=(0, 0.04, 1, 1))
    return _save(fig)
