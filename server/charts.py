from io import BytesIO

import matplotlib

matplotlib.use("Agg")

import matplotlib.pyplot as plt


PERCENT_CEILING = 100
PERCENT_TICKS = [0, 25, 50, 75, 100]
AXIS_PADDING = 0.04


def _caption(username, level_id, eval_id, date_label):
    return f"{username}  ·  level: {level_id}  ·  id: {eval_id[:8]}  ·  {date_label}"


def _percent_axis(ax):
    ax.set_ylim(0, PERCENT_CEILING)
    ax.set_yticks(PERCENT_TICKS)


def _padded_xlim(ax, xs):
    if not xs:
        return
    low = min(xs)
    high = max(xs)
    padding = (high - low) * AXIS_PADDING or 1.0
    ax.set_xlim(low - padding, high + padding)


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


def render_thumbnail_png(username, level_id, detection_rate, trials, outcomes):
    fig = plt.figure(figsize=(12.0, 6.3), dpi=100)
    fig.patch.set_facecolor("#0a0e1a")

    fig.text(0.06, 0.84, "ASTROSWARM", fontsize=26, color="#7c9eff", weight="bold")
    fig.text(0.06, 0.66, str(username), fontsize=52, color="#ffffff", weight="bold")
    fig.text(0.06, 0.55, f"{str(level_id).upper()} defender benchmark", fontsize=20, color="#8ba3c9")
    fig.text(0.06, 0.30, f"{detection_rate}%", fontsize=72, color="#4ade80", weight="bold")
    fig.text(0.06, 0.20, f"detection rate over {trials} trials", fontsize=18, color="#8ba3c9")

    if outcomes:
        ax = fig.add_axes([0.55, 0.2, 0.39, 0.58])
        ax.set_facecolor("#0a0e1a")
        wins = 0
        xs = []
        ys = []
        for index, outcome in enumerate(outcomes):
            if outcome == "win":
                wins += 1
            xs.append(index + 1)
            ys.append(100.0 * wins / (index + 1))
        ax.plot(xs, ys, color="#7c9eff", linewidth=3)
        _percent_axis(ax)
        _padded_xlim(ax, xs)
        ax.set_title("Cumulative detection rate", color="#8ba3c9", fontsize=14)
        ax.tick_params(colors="#8ba3c9")
        for spine in ax.spines.values():
            spine.set_color("#22304d")
        ax.grid(True, color="#16203a")

    return _save(fig)


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
    _percent_axis(ax)
    _padded_xlim(ax, xs)
    ax.grid(True, color="#e5e7eb")
    fig.text(0.5, 0.005, _caption(username, level_id, eval_id, date_label), ha="center", fontsize=8, color="#6b7280")
    fig.tight_layout(rect=(0, 0.04, 1, 1))
    return _save(fig)


def _sweep_rate(point, rate_key, time_key):
    value = point.get(rate_key)
    if value is not None:
        return value
    time_value = point.get(time_key, -1)
    return 100.0 if time_value is not None and time_value >= 0 else 0.0


def _render_sweep_rate_png(rows, rate_key, time_key, color, title, ylabel, meta):
    points = sorted(rows, key=lambda point: point.get("n", 0))
    xs = [point.get("n", 0) for point in points]
    ys = [_sweep_rate(point, rate_key, time_key) for point in points]

    fig, ax = plt.subplots(figsize=(6.4, 3.8))
    ax.plot(xs, ys, color=color, linewidth=2)
    ax.set_title(title)
    ax.set_xlabel("Defenders in ring (n)")
    ax.set_ylabel(ylabel)
    _percent_axis(ax)
    _padded_xlim(ax, xs)
    ax.grid(True, color="#e5e7eb")
    fig.text(0.5, 0.005, _caption(*meta), ha="center", fontsize=8, color="#6b7280")
    fig.tight_layout(rect=(0, 0.04, 1, 1))
    return _save(fig)


def render_detection_rate_png(rows, username, level_id, eval_id, date_label):
    return _render_sweep_rate_png(
        rows, "detection_rate", "detection_time", "#2563eb",
        "Detection Success Rate vs Number of Defenders", "Detection success rate (%)",
        (username, level_id, eval_id, date_label),
    )


def render_capture_rate_png(rows, username, level_id, eval_id, date_label):
    return _render_sweep_rate_png(
        rows, "capture_rate", "capture_time", "#dc2626",
        "Capture Success Rate vs Number of Defenders", "Capture success rate (%)",
        (username, level_id, eval_id, date_label),
    )


def _sweep_risk(point):
    risk = point.get("risk")
    if risk is not None:
        return risk
    return 100.0 - _sweep_rate(point, "capture_rate", "capture_time")


def render_risk_png(rows, username, level_id, eval_id, date_label):
    points = sorted(rows, key=lambda point: point.get("n", 0))
    xs = [point.get("n", 0) for point in points]
    risks = [_sweep_risk(point) for point in points]
    captures = [_sweep_rate(point, "capture_rate", "capture_time") for point in points]

    fig, ax = plt.subplots(figsize=(6.4, 3.8))
    ax.plot(xs, risks, color="#dc2626", linewidth=2, label="Risk")
    ax.plot(xs, captures, color="#16a34a", linewidth=2, label="Capture success rate")
    ax.set_title("Risk vs Number of Defenders")
    ax.set_xlabel("Defenders in ring (n)")
    ax.set_ylabel("Risk = 1 - capture success rate (%)")
    _percent_axis(ax)
    _padded_xlim(ax, xs)
    ax.grid(True, color="#e5e7eb")
    ax.legend(loc="upper right", fontsize=8)
    fig.text(0.5, 0.005, _caption(username, level_id, eval_id, date_label), ha="center", fontsize=8, color="#6b7280")
    fig.tight_layout(rect=(0, 0.04, 1, 1))
    return _save(fig)


# The attrition curve reads right to left: a full line sits at the high end of
# the axis, and every trade moves the algorithm one rung down it.
def render_attrition_png(points, username, level_id, eval_id, date_label):
    rows = sorted(points, key=lambda point: point.get("defenders", 0))
    xs = [point.get("defenders", 0) for point in rows]
    ys = [point.get("risk", 0.0) for point in rows]

    fig, ax = plt.subplots(figsize=(6.4, 3.8))
    ax.plot(xs, ys, color="#dc2626", linewidth=2, marker="o", markersize=4)
    ax.set_title("Risk as the Defender Line Thins")
    ax.set_xlabel("Defenders still standing when the evader launched")
    ax.set_ylabel("Risk = 1 - capture success rate (%)")
    _percent_axis(ax)
    _padded_xlim(ax, xs)
    ax.grid(True, color="#e5e7eb")
    fig.text(0.5, 0.005, _caption(username, level_id, eval_id, date_label), ha="center", fontsize=8, color="#6b7280")
    fig.tight_layout(rect=(0, 0.04, 1, 1))
    return _save(fig)


SERIES_COLORS = ["#2563eb", "#16a34a", "#d97706", "#9333ea", "#0891b2", "#db2777"]
MAX_ATTRITION_SERIES = 6


# Ring sizes are consecutive, so drawing every one of them is unreadable. Take an
# even spread across the sweep and always keep the largest ring, which is the one
# the sweep stopped at.
def _spread(series, limit=MAX_ATTRITION_SERIES):
    if len(series) <= limit:
        return series
    step = (len(series) - 1) / (limit - 1)
    return [series[round(index * step)] for index in range(limit)]


# One curve per ring size: a line that started at n and traded itself down, so
# two algorithms can be compared on what their risk does as the line thins.
def render_sweep_attrition_png(series, username, level_id, eval_id, date_label):
    chosen = _spread(sorted(series, key=lambda entry: entry.get("n", 0)))

    fig, ax = plt.subplots(figsize=(6.4, 3.8))
    everything = []
    for index, entry in enumerate(chosen):
        points = sorted(entry.get("points", []), key=lambda point: point.get("defenders", 0))
        xs = [point.get("defenders", 0) for point in points]
        ys = [point.get("risk", 0.0) for point in points]
        everything.extend(xs)
        ax.plot(xs, ys, color=SERIES_COLORS[index % len(SERIES_COLORS)], linewidth=2, label=f"n = {entry.get('n')}")
    ax.set_title("Risk as the Line Thins, by Ring Size")
    ax.set_xlabel("Defenders still standing when the evader launched")
    ax.set_ylabel("Risk = 1 - capture success rate (%)")
    _percent_axis(ax)
    _padded_xlim(ax, everything)
    ax.grid(True, color="#e5e7eb")
    if chosen:
        ax.legend(loc="upper right", fontsize=8)
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
    tallest = max(detections + captures + [0.0])
    ax.set_ylim(0, tallest * 1.18 if tallest > 0 else 1.0)
    ax.margins(x=AXIS_PADDING)
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
    _percent_axis(ax)
    ax.grid(True, axis="y", color="#e5e7eb")
    for index, value in enumerate(values):
        ax.text(index, value + 1.5, f"{value:.0f}%", ha="center", fontsize=9, color="#374151")
    fig.text(0.5, 0.005, _caption(username, level_id, eval_id, date_label), ha="center", fontsize=8, color="#6b7280")
    fig.tight_layout(rect=(0, 0.04, 1, 1))
    return _save(fig)
