from dataclasses import dataclass, field

from app_settings import (
    LEVELS,
    PILOT_LEVELS,
    is_assault_level,
    is_pilot_level,
    is_supply_level,
    is_swarm_level,
)

LEVEL_NAMES = {level["id"]: level["name"] for level in LEVELS + PILOT_LEVELS}

PILOT_HEADLINES = {"win": "Planet reached", "lose": "Caught", "timeout": "Out of time"}
MISSING_TIME = "—"


@dataclass
class EntryCard:
    level_name: str
    headline: str
    caption: str
    positive: bool
    trend_title: str = ""
    trend: list = field(default_factory=list)
    facts: list = field(default_factory=list)

    def description(self):
        return f"{self.headline}, {self.caption}."


def canonical_level_id(level_id):
    if not level_id or level_id == "farp":
        return "farp1"
    return level_id


def build_entry_card(evaluation):
    level_id = canonical_level_id(evaluation.level_id)
    level_name = LEVEL_NAMES.get(level_id, level_id.upper())
    results = evaluation.results if isinstance(evaluation.results, dict) else {}
    outcomes = results.get("outcomes") or []

    if not outcomes:
        return _pending_card(level_name, evaluation.status)
    if is_supply_level(level_id):
        return _supply_card(level_name, results, outcomes)
    if is_swarm_level(level_id):
        return _swarm_card(level_name, results, outcomes)
    if is_pilot_level(level_id):
        return _pilot_card(level_name, results, outcomes)
    if is_assault_level(level_id):
        return _assault_card(level_name, results, outcomes)
    return _defense_card(level_name, results, outcomes)


def _pending_card(level_name, status):
    if status in ("queued", "running"):
        return EntryCard(level_name, "Running", "results appear when the benchmark finishes", False)
    return EntryCard(level_name, "No results", "this entry has no finished results", False)


def _defense_card(level_name, results, outcomes):
    captures = [1 if outcome == "win" else 0 for outcome in outcomes]
    return EntryCard(
        level_name=level_name,
        headline=f"{results.get('success_rate', 0)}%",
        caption=f"capture success rate over {len(outcomes)} trials",
        positive=True,
        trend_title="Cumulative capture success rate",
        trend=running_rate(captures, [1] * len(outcomes)),
    )


def _assault_card(level_name, results, outcomes):
    destroyed = results.get("trial_destroyed") or []
    resolved = results.get("trial_resolved") or []
    evaders = results.get("evaders_resolved", sum(resolved))
    return EntryCard(
        level_name=level_name,
        headline=f"{results.get('success_rate', 0)}%",
        caption=f"capture success rate over {evaders} evaders in {len(outcomes)} trials",
        positive=True,
        trend_title="Cumulative capture success rate",
        trend=running_rate(destroyed, resolved),
    )


def _pilot_card(level_name, results, outcomes):
    outcome = outcomes[0]
    return EntryCard(
        level_name=level_name,
        headline=PILOT_HEADLINES.get(outcome, PILOT_HEADLINES["timeout"]),
        caption="piloted evasion run",
        positive=outcome == "win",
        facts=[
            ("Detected", format_time(_first(results.get("detection_times")))),
            ("Captured", format_time(_first(results.get("capture_times")))),
            ("Reached planet", format_time(_first(results.get("goal_times")))),
        ],
    )


def _swarm_card(level_name, results, outcomes):
    stats = results.get("stats") or {}
    passed = outcomes[0] == "win"
    merge_only = passed and stats.get("delivered") == 0 and bool(stats.get("merge_passed"))
    headline = "Swarms merged" if merge_only else "Swarm delivered" if passed else "Out of time"
    return EntryCard(
        level_name=level_name,
        headline=headline,
        caption="piloted swarm run",
        positive=passed,
        facts=[
            ("Merged", format_time(stats.get("merge_time"))),
            ("Delivered", format_time(stats.get("deliver_time"))),
            ("Escaped", format_time(stats.get("escape_time"))),
        ],
    )


def _supply_card(level_name, results, outcomes):
    stats = results.get("stats") or {}
    held = outcomes[0] == "win"
    destroyed = results.get("evaders_destroyed", 0)
    resolved = results.get("evaders_resolved", 0)
    planets = int(stats.get("planets", 2))
    facts = [("Evaders stopped", f"{results.get('success_rate', 0)}%")]
    for letter in "ABC"[:planets]:
        facts.append((f"Through on planet {letter}", str(stats.get("breached_" + letter.lower(), 0))))
    return EntryCard(
        level_name=level_name,
        headline=("Both planets held" if planets == 2 else "All planets held") if held else "Line broken",
        caption=f"{destroyed} of {resolved} evaders stopped across {'the pair' if planets == 2 else 'all three planets'}",
        positive=held,
        facts=facts,
    )


def running_rate(hits, totals):
    points = []
    hit_sum = 0
    total_sum = 0
    for trial, (hit, total) in enumerate(zip(hits, totals), start=1):
        hit_sum += hit
        total_sum += total
        if total_sum > 0:
            points.append((trial, 100.0 * hit_sum / total_sum))
    return points


def format_time(seconds):
    if seconds is None or seconds < 0:
        return MISSING_TIME
    return f"{seconds:.2f}s"


def _first(values):
    return values[0] if values else None
