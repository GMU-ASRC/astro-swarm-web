PRIOR_ENTRIES = 2.0
FALLBACK_RATE = 50.0


def level_averages(players):
    totals = {}
    for player in players.values():
        for level_number, level in player["levels"].items():
            if not level["count"]:
                continue
            bucket = totals.setdefault(level_number, [0.0, 0])
            bucket[0] += level["success_sum"] / level["count"]
            bucket[1] += 1
    return {
        level_number: round(total / player_count, 2)
        for level_number, (total, player_count) in totals.items()
        if player_count
    }


def rate_player(player, averages):
    breakdown = []
    for level_number in sorted(averages):
        level_average = averages.get(level_number, FALLBACK_RATE)
        level = player["levels"].get(level_number)
        entries = level["count"] if level else 0
        if entries:
            raw_rate = level["success_sum"] / entries
            weighted_rate = (
                entries * raw_rate + PRIOR_ENTRIES * level_average
            ) / (entries + PRIOR_ENTRIES)
        else:
            raw_rate = None
            weighted_rate = level_average
        breakdown.append({
            "level_number": level_number,
            "entries": entries,
            "raw_rate": round(raw_rate, 1) if raw_rate is not None else None,
            "weighted_rate": round(weighted_rate, 1),
            "level_average": level_average,
            "played": entries > 0,
        })
    if not breakdown or not player["success_count"]:
        return None, breakdown
    rating = sum(entry["weighted_rate"] for entry in breakdown) / len(breakdown)
    return round(rating, 1), breakdown


def apply(players):
    averages = level_averages(players)
    total_levels = len(averages)
    for player in players.values():
        rating, breakdown = rate_player(player, averages)
        player["rating"] = rating
        player["rating_levels"] = breakdown
        player["levels_played"] = sum(1 for entry in breakdown if entry["played"])
        player["levels_total"] = total_levels
    return averages


def sort_key(player):
    return (
        player["rating"] is not None,
        player["rating"] or 0.0,
        player["levels_played"],
        player["entries"],
    )
