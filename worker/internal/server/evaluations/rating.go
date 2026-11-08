package evaluations

import (
	"sort"
	"time"

	"astroswarm/worker/internal/server/entries"
	"astroswarm/worker/internal/server/values"
)

const (
	priorEntries = 2.0
	fallbackRate = 50.0
)

type levelStats struct {
	successSum  float64
	count       int
	xp          int
	successRate *float64
}

type player struct {
	playerID       string
	username       string
	totalXP        int
	entries        int
	successSum     float64
	successCount   int
	bestSuccess    *float64
	overallSuccess *float64
	levels         map[int]*levelStats
	last           *time.Time
	rating         *float64
	ratingLevels   []map[string]any
	levelsPlayed   int
	levelsTotal    int
}

type playerTable struct {
	order   []*player
	byID    map[string]*player
	average map[int]float64
}

func aggregatePlayers(listed []*entries.Entry) *playerTable {
	table := &playerTable{byID: map[string]*player{}}
	for _, entry := range listed {
		current := table.byID[entry.PlayerID]
		if current == nil {
			current = &player{playerID: entry.PlayerID, username: entry.Username, levels: map[int]*levelStats{}, last: entry.CreatedAt}
			table.byID[entry.PlayerID] = current
			table.order = append(table.order, current)
		}
		if entry.CreatedAt != nil && (current.last == nil || !entry.CreatedAt.Before(*current.last)) {
			current.username = entry.Username
			current.last = entry.CreatedAt
		}
		xp := 0
		if entry.XPAwarded != nil {
			xp = *entry.XPAwarded
		}
		current.totalXP += xp
		current.entries++
		levelNumber := entry.LevelNumber()
		level := current.levels[levelNumber]
		if level == nil {
			level = &levelStats{}
			current.levels[levelNumber] = level
		}
		level.xp += xp
		if entry.SuccessRate != nil && entry.StatusText() == "done" {
			rate := *entry.SuccessRate
			current.successSum += rate
			current.successCount++
			if current.bestSuccess == nil || rate > *current.bestSuccess {
				best := rate
				current.bestSuccess = &best
			}
			level.successSum += rate
			level.count++
		}
	}
	for _, current := range table.order {
		if current.successCount > 0 {
			overall := values.RoundTo(current.successSum/float64(current.successCount), 1)
			current.overallSuccess = &overall
		}
		if current.bestSuccess != nil {
			rounded := values.RoundTo(*current.bestSuccess, 1)
			current.bestSuccess = &rounded
		}
		for _, level := range current.levels {
			if level.count > 0 {
				rate := values.RoundTo(level.successSum/float64(level.count), 1)
				level.successRate = &rate
			}
		}
	}
	table.applyRating()
	return table
}

func (table *playerTable) applyRating() {
	totals := map[int][2]float64{}
	for _, current := range table.order {
		for levelNumber, level := range current.levels {
			if level.count == 0 {
				continue
			}
			bucket := totals[levelNumber]
			bucket[0] += level.successSum / float64(level.count)
			bucket[1]++
			totals[levelNumber] = bucket
		}
	}
	table.average = map[int]float64{}
	for levelNumber, bucket := range totals {
		if bucket[1] > 0 {
			table.average[levelNumber] = values.RoundTo(bucket[0]/bucket[1], 2)
		}
	}
	levelNumbers := make([]int, 0, len(table.average))
	for levelNumber := range table.average {
		levelNumbers = append(levelNumbers, levelNumber)
	}
	sort.Ints(levelNumbers)
	for _, current := range table.order {
		breakdown := []map[string]any{}
		weightedTotal := 0.0
		for _, levelNumber := range levelNumbers {
			levelAverage, found := table.average[levelNumber]
			if !found {
				levelAverage = fallbackRate
			}
			level := current.levels[levelNumber]
			entryCount := 0
			if level != nil {
				entryCount = level.count
			}
			var rawRate any
			weightedRate := levelAverage
			if entryCount > 0 {
				raw := level.successSum / float64(entryCount)
				rawRate = values.RoundTo(raw, 1)
				weightedRate = (float64(entryCount)*raw + priorEntries*levelAverage) / (float64(entryCount) + priorEntries)
			}
			roundedWeighted := values.RoundTo(weightedRate, 1)
			weightedTotal += roundedWeighted
			breakdown = append(breakdown, map[string]any{
				"level_number": levelNumber, "entries": entryCount, "raw_rate": rawRate,
				"weighted_rate": roundedWeighted, "level_average": levelAverage, "played": entryCount > 0,
			})
			if entryCount > 0 {
				current.levelsPlayed++
			}
		}
		current.ratingLevels = breakdown
		current.levelsTotal = len(levelNumbers)
		if len(breakdown) > 0 && current.successCount > 0 {
			rating := values.RoundTo(weightedTotal/float64(len(breakdown)), 1)
			current.rating = &rating
		}
	}
}

func (table *playerTable) sortedBySuccess() []*player {
	sorted := append([]*player{}, table.order...)
	sort.SliceStable(sorted, func(left, right int) bool {
		return ratingLess(sorted[right], sorted[left])
	})
	return sorted
}

func ratingLess(first *player, second *player) bool {
	firstRated, secondRated := first.rating != nil, second.rating != nil
	if firstRated != secondRated {
		return !firstRated
	}
	firstRating, secondRating := 0.0, 0.0
	if first.rating != nil {
		firstRating = *first.rating
	}
	if second.rating != nil {
		secondRating = *second.rating
	}
	if firstRating != secondRating {
		return firstRating < secondRating
	}
	if first.levelsPlayed != second.levelsPlayed {
		return first.levelsPlayed < second.levelsPlayed
	}
	return first.entries < second.entries
}

func weightedLevelRate(current *player, levelNumber int) float64 {
	for _, entry := range current.ratingLevels {
		if entry["level_number"] == levelNumber {
			return entry["weighted_rate"].(float64)
		}
	}
	return 0.0
}

func optionalNumber(value *float64) any {
	if value == nil {
		return nil
	}
	return *value
}
