package entry

import (
	"fmt"
	"strings"
)

type Settings struct {
	Seed            int64   `json:"seed"`
	PlacementTrials int     `json:"placement_trials"`
	SweepMax        int     `json:"sweep_max"`
	SweepTrials     int     `json:"sweep_trials"`
	MatchCapSeconds float64 `json:"match_cap_seconds"`
	GoalTailSeconds float64 `json:"goal_tail_seconds"`
	EnemyStartX     float64 `json:"enemy_start_x"`
	EnemyStartY     float64 `json:"enemy_start_y"`
	SweepSpawn      string  `json:"sweep_spawn"`
	ArenaWidth      float64 `json:"arena_width"`
	ArenaHeight     float64 `json:"arena_height"`

	// Levels 3 to 5 grade a stream of evaders rather than one approach, so the
	// server gives them their own trial and sweep budget.
	AssaultTrials      int `json:"assault_trials"`
	AssaultSweepMax    int `json:"assault_sweep_max"`
	AssaultSweepTrials int `json:"assault_sweep_trials"`
}

type SweepIndexEntry struct {
	N             int      `json:"n"`
	DetectionRate *float64 `json:"detection_rate"`
	CaptureRate   *float64 `json:"capture_rate"`
	TrialCount    int      `json:"trial_count"`
}

func FetchSweepIndex(serverURL string, id string) ([]SweepIndexEntry, error) {
	endpoint := fmt.Sprintf("%s/api/evaluations/%s/sweep-replays", strings.TrimRight(serverURL, "/"), id)
	index := []SweepIndexEntry{}
	if err := getJSON(endpoint, &index); err != nil {
		return nil, err
	}
	return index, nil
}

func FetchSettings(serverURL string) (*Settings, error) {
	endpoint := fmt.Sprintf("%s/api/evaluations/settings", strings.TrimRight(serverURL, "/"))
	settings := &Settings{}
	if err := getJSON(endpoint, settings); err != nil {
		return nil, err
	}
	return settings, nil
}
