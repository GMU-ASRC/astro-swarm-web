package entrycard

import (
	"fmt"
	"strings"

	"astroswarm/worker/internal/server/settings"
	"astroswarm/worker/internal/server/values"
)

const missingTime = "—"

var pilotHeadlines = map[string]string{"win": "Planet reached", "lose": "Caught", "timeout": "Out of time"}

type TrendPoint struct {
	Trial int
	Rate  float64
}

type Fact struct {
	Label string
	Value string
}

type Card struct {
	LevelName  string
	Headline   string
	Caption    string
	Positive   bool
	TrendTitle string
	Trend      []TrendPoint
	Facts      []Fact
}

func (card Card) Description() string {
	return card.Headline + ", " + card.Caption + "."
}

func CanonicalLevelID(levelID string) string {
	if levelID == "" || levelID == "farp" {
		return "farp1"
	}
	return levelID
}

func LevelName(levelID string) string {
	for _, level := range settings.AllLevels() {
		if level.ID == levelID {
			return level.Name
		}
	}
	return strings.ToUpper(levelID)
}

func Build(levelID string, status string, results map[string]any) Card {
	levelID = CanonicalLevelID(levelID)
	levelName := LevelName(levelID)
	outcomes, _ := values.List(results["outcomes"])
	switch {
	case len(outcomes) == 0:
		return pendingCard(levelName, status)
	case settings.IsSupplyLevel(levelID):
		return supplyCard(levelName, results, outcomes)
	case settings.IsSwarmLevel(levelID):
		return swarmCard(levelName, results, outcomes)
	case settings.IsPilotLevel(levelID):
		return pilotCard(levelName, results, outcomes)
	case settings.IsAssaultLevel(levelID):
		return assaultCard(levelName, results, outcomes)
	}
	return defenseCard(levelName, results, outcomes)
}

func valueOr(object map[string]any, key string, fallback any) any {
	if value, found := object[key]; found {
		return value
	}
	return fallback
}

func numbers(value any) []float64 {
	list, _ := values.List(value)
	output := make([]float64, len(list))
	for index, item := range list {
		output[index], _ = values.Number(item)
	}
	return output
}

func successText(results map[string]any) string {
	return values.ToText(valueOr(results, "success_rate", 0)) + "%"
}

func pendingCard(levelName string, status string) Card {
	if status == "queued" || status == "running" {
		return Card{LevelName: levelName, Headline: "Running", Caption: "results appear when the benchmark finishes"}
	}
	return Card{LevelName: levelName, Headline: "No results", Caption: "this entry has no finished results"}
}

func defenseCard(levelName string, results map[string]any, outcomes []any) Card {
	captures := make([]float64, len(outcomes))
	totals := make([]float64, len(outcomes))
	for index, outcome := range outcomes {
		if outcome == "win" {
			captures[index] = 1
		}
		totals[index] = 1
	}
	return Card{
		LevelName: levelName, Headline: successText(results), Positive: true,
		Caption:    fmt.Sprintf("capture success rate over %d trials", len(outcomes)),
		TrendTitle: "Cumulative capture success rate", Trend: RunningRate(captures, totals),
	}
}

func assaultCard(levelName string, results map[string]any, outcomes []any) Card {
	destroyed := numbers(results["trial_destroyed"])
	resolved := numbers(results["trial_resolved"])
	resolvedSum := 0.0
	for _, count := range resolved {
		resolvedSum += count
	}
	evaders := valueOr(results, "evaders_resolved", int64(resolvedSum))
	return Card{
		LevelName: levelName, Headline: successText(results), Positive: true,
		Caption:    fmt.Sprintf("capture success rate over %s evaders in %d trials", values.ToText(evaders), len(outcomes)),
		TrendTitle: "Cumulative capture success rate", Trend: RunningRate(destroyed, resolved),
	}
}

func pilotCard(levelName string, results map[string]any, outcomes []any) Card {
	outcome, _ := outcomes[0].(string)
	headline, known := pilotHeadlines[outcome]
	if !known {
		headline = pilotHeadlines["timeout"]
	}
	return Card{
		LevelName: levelName, Headline: headline, Caption: "piloted evasion run", Positive: outcome == "win",
		Facts: []Fact{
			{"Detected", FormatTime(first(results["detection_times"]))},
			{"Captured", FormatTime(first(results["capture_times"]))},
			{"Reached planet", FormatTime(first(results["goal_times"]))},
		},
	}
}

func swarmCard(levelName string, results map[string]any, outcomes []any) Card {
	stats, _ := values.Object(results["stats"])
	passed := outcomes[0] == "win"
	delivered, deliveredIsNumber := values.Number(stats["delivered"])
	mergeOnly := passed && deliveredIsNumber && delivered == 0 && values.Truthy(stats["merge_passed"])
	headline := "Out of time"
	if mergeOnly {
		headline = "Swarms merged"
	} else if passed {
		headline = "Swarm delivered"
	}
	return Card{
		LevelName: levelName, Headline: headline, Caption: "piloted swarm run", Positive: passed,
		Facts: []Fact{
			{"Merged", FormatTime(stats["merge_time"])},
			{"Delivered", FormatTime(stats["deliver_time"])},
			{"Escaped", FormatTime(stats["escape_time"])},
		},
	}
}

func supplyCard(levelName string, results map[string]any, outcomes []any) Card {
	stats, _ := values.Object(results["stats"])
	held := outcomes[0] == "win"
	planets := 2
	if count, err := values.ToInt(valueOr(stats, "planets", 2)); err == nil {
		planets = int(count)
	}
	facts := []Fact{{"Evaders stopped", successText(results)}}
	letters := "ABC"[:max(0, min(planets, 3))]
	for _, letter := range letters {
		key := "breached_" + strings.ToLower(string(letter))
		facts = append(facts, Fact{"Through on planet " + string(letter), values.ToText(valueOr(stats, key, 0))})
	}
	headline := "Line broken"
	if held && planets == 2 {
		headline = "Both planets held"
	} else if held {
		headline = "All planets held"
	}
	across := "all three planets"
	if planets == 2 {
		across = "the pair"
	}
	return Card{
		LevelName: levelName, Headline: headline, Positive: held, Facts: facts,
		Caption: fmt.Sprintf("%s of %s evaders stopped across %s",
			values.ToText(valueOr(results, "evaders_destroyed", 0)), values.ToText(valueOr(results, "evaders_resolved", 0)), across),
	}
}

func RunningRate(hits []float64, totals []float64) []TrendPoint {
	points := []TrendPoint{}
	hitSum, totalSum := 0.0, 0.0
	for index := 0; index < len(hits) && index < len(totals); index++ {
		hitSum += hits[index]
		totalSum += totals[index]
		if totalSum > 0 {
			points = append(points, TrendPoint{Trial: index + 1, Rate: 100.0 * hitSum / totalSum})
		}
	}
	return points
}

func FormatTime(value any) string {
	seconds, isNumber := values.Number(value)
	if !isNumber || seconds < 0 {
		return missingTime
	}
	return fmt.Sprintf("%.2fs", seconds)
}

func first(value any) any {
	list, _ := values.List(value)
	if len(list) == 0 {
		return nil
	}
	return list[0]
}
