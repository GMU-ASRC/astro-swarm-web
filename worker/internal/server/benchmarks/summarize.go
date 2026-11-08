package benchmarks

import (
	"sort"

	"astroswarm/worker/internal/server/values"
)

var metaKeys = []string{"fps", "defenders", "view", "fov", "speed", "hull", "planet", "arena"}

func Summarize(result map[string]any) (map[string]any, map[string]any) {
	if result == nil {
		result = map[string]any{}
	}
	metaSource, _ := result["meta"].(map[string]any)
	replays := map[string]any{}
	for _, key := range metaKeys {
		replays[key] = metaSource[key]
	}

	runs := objectList(result["runs"])
	sweepRuns := objectList(result["sweep_runs"])
	sort.SliceStable(runs, func(left, right int) bool {
		return numberOr(runs[left]["trial"], 0) < numberOr(runs[right]["trial"], 0)
	})
	sort.SliceStable(sweepRuns, func(left, right int) bool {
		return numberOr(sweepRuns[left]["n"], 0) < numberOr(sweepRuns[right]["n"], 0)
	})

	outcomes := make([]any, len(runs))
	detectionTimes := make([]any, len(runs))
	captureTimes := make([]any, len(runs))
	goalTimes := make([]any, len(runs))
	wins := 0
	for index, run := range runs {
		outcomes[index] = run["outcome"]
		detectionTimes[index] = valueOr(run, "detection_time", -1.0)
		captureTimes[index] = valueOr(run, "capture_time", -1.0)
		goalTimes[index] = valueOr(run, "goal_time", -1.0)
		if run["outcome"] == "win" {
			wins++
		}
	}
	sweep := make([]any, len(sweepRuns))
	for index, sweepRun := range sweepRuns {
		sweep[index] = sweepPoint(sweepRun)
	}
	replays["runs"] = mapsToAny(runs)
	replays["sweep_runs"] = mapsToAny(sweepRuns)

	results := map[string]any{
		"trials":          len(outcomes),
		"success_rate":    values.RoundTo(100.0*float64(wins)/float64(max(1, len(outcomes))), 1),
		"detection_rate":  hitRate(detectionTimes),
		"capture_rate":    hitRate(captureTimes),
		"outcomes":        outcomes,
		"detection_times": detectionTimes,
		"capture_times":   captureTimes,
		"goal_times":      goalTimes,
		"sweep":           sweep,
	}
	for _, run := range runs {
		if stats, isObject := run["stats"].(map[string]any); isObject && len(stats) > 0 {
			results["stats"] = stats
			break
		}
	}
	for key, value := range assaultSummary(runs) {
		results[key] = value
	}
	for key, value := range supplySummary(runs) {
		results[key] = value
	}
	if reported, isObject := result["results"].(map[string]any); isObject {
		for key, value := range reported {
			if value != nil {
				results[key] = value
			}
		}
	}
	return results, replays
}

func assaultSummary(runs []map[string]any) map[string]any {
	var destroyedPerTrial, resolvedPerTrial, breachesPerTrial, lostPerTrial []any
	destroyed, resolved, breaches, lost, held, trials := int64(0), int64(0), int64(0), int64(0), 0, 0
	for _, run := range runs {
		stats, isObject := run["stats"].(map[string]any)
		if !isObject {
			continue
		}
		if _, hasResolved := stats["resolved"]; !hasResolved {
			continue
		}
		trials++
		trialDestroyed, trialResolved := intOf(stats, "destroyed"), intOf(stats, "resolved")
		trialBreaches, trialLost := intOf(stats, "breaches"), intOf(stats, "lost")
		destroyedPerTrial = append(destroyedPerTrial, trialDestroyed)
		resolvedPerTrial = append(resolvedPerTrial, trialResolved)
		breachesPerTrial = append(breachesPerTrial, trialBreaches)
		lostPerTrial = append(lostPerTrial, trialLost)
		destroyed += trialDestroyed
		resolved += trialResolved
		breaches += trialBreaches
		lost += trialLost
		if run["outcome"] == "win" {
			held++
		}
	}
	if trials == 0 {
		return nil
	}
	captureRate := 0.0
	if resolved != 0 {
		captureRate = values.RoundTo(100.0*float64(destroyed)/float64(resolved), 1)
	}
	return map[string]any{
		"success_rate":          captureRate,
		"evader_destroyed_rate": captureRate,
		"evaders_destroyed":     destroyed,
		"evaders_resolved":      resolved,
		"breaches":              breaches,
		"defenders_lost":        lost,
		"risk":                  values.RoundTo(100.0-captureRate, 1),
		"trials_held_rate":      values.RoundTo(100.0*float64(held)/float64(trials), 1),
		"trial_destroyed":       destroyedPerTrial,
		"trial_resolved":        resolvedPerTrial,
		"trial_breaches":        breachesPerTrial,
		"trial_lost":            lostPerTrial,
	}
}

func supplySummary(runs []map[string]any) map[string]any {
	var stats map[string]any
	for _, run := range runs {
		if candidate, isObject := run["stats"].(map[string]any); isObject {
			if _, hasHoldRate := candidate["hold_rate"]; hasHoldRate {
				stats = candidate
				break
			}
		}
	}
	if stats == nil {
		return nil
	}
	letters := "abc"[:max(0, min(3, int(intOrDefault(stats, "planets", 2))))]
	destroyed, breaches, lost := int64(0), int64(0), int64(0)
	for _, letter := range letters {
		destroyed += intOf(stats, "destroyed_"+string(letter))
		breaches += intOf(stats, "breached_"+string(letter))
		lost += intOf(stats, "lost_"+string(letter))
	}
	holdRate, _ := values.ToFloat(valueOr(stats, "hold_rate", 0.0))
	summary := map[string]any{
		"success_rate":          values.RoundTo(holdRate, 1),
		"evader_destroyed_rate": values.RoundTo(holdRate, 1),
		"evaders_destroyed":     destroyed,
		"evaders_resolved":      destroyed + breaches,
		"breaches":              breaches,
		"defenders_a":           intOf(stats, "defenders_a"),
		"defenders_b":           intOf(stats, "defenders_b"),
		"defenders_delivered":   intOf(stats, "delivered"),
		"defenders_total":       intOrDefault(stats, "defenders_total", 10),
		"defenders_lost":        lost,
		"crossings":             intOf(stats, "trips"),
		"planets":               len(letters),
	}
	for _, letter := range letters[min(2, len(letters)):] {
		summary["defenders_"+string(letter)] = intOf(stats, "defenders_"+string(letter))
	}
	return summary
}

func sweepPoint(run map[string]any) map[string]any {
	rate := 0.0
	if captureRate, present := run["capture_rate"]; present && captureRate != nil {
		rate, _ = values.ToFloat(captureRate)
	} else if run["outcome"] == "win" {
		rate = 100.0
	}
	rate = values.RoundTo(rate, 1)
	return map[string]any{
		"n":              run["n"],
		"success_rate":   rate,
		"capture_rate":   rate,
		"detection_rate": run["detection_rate"],
		"risk":           values.RoundTo(100.0-rate, 1),
	}
}

func hitRate(times []any) float64 {
	if len(times) == 0 {
		return 0.0
	}
	hits := 0
	for _, value := range times {
		if number, isNumber := values.Number(value); isNumber && number >= 0.0 {
			hits++
		}
	}
	return values.RoundTo(100.0*float64(hits)/float64(len(times)), 1)
}

func AnnotateRates(results map[string]any, attackLevel bool) {
	rateValue, present := results["success_rate"]
	if !present || rateValue == nil {
		return
	}
	rate, err := values.ToFloat(rateValue)
	if err != nil {
		return
	}
	if attackLevel {
		results["attacker_rate"] = values.RoundTo(rate, 1)
		results["defender_rate"] = values.RoundTo(100.0-rate, 1)
	} else {
		results["defender_rate"] = values.RoundTo(rate, 1)
		results["attacker_rate"] = values.RoundTo(100.0-rate, 1)
	}
}

func objectList(value any) []map[string]any {
	list, _ := value.([]any)
	objects := make([]map[string]any, 0, len(list))
	for _, item := range list {
		if object, isObject := item.(map[string]any); isObject {
			objects = append(objects, object)
		}
	}
	return objects
}

func mapsToAny(objects []map[string]any) []any {
	list := make([]any, len(objects))
	for index, object := range objects {
		list[index] = object
	}
	return list
}

func valueOr(object map[string]any, key string, fallback any) any {
	if value, found := object[key]; found {
		return value
	}
	return fallback
}

func numberOr(value any, fallback float64) float64 {
	if number, isNumber := values.Number(value); isNumber {
		return number
	}
	return fallback
}

func intOf(object map[string]any, key string) int64 {
	return intOrDefault(object, key, 0)
}

func intOrDefault(object map[string]any, key string, fallback int64) int64 {
	value, found := object[key]
	if !found {
		return fallback
	}
	converted, err := values.ToInt(value)
	if err != nil {
		return fallback
	}
	return converted
}
