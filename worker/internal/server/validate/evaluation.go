package validate

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"astroswarm/worker/internal/server/values"
)

const (
	defaultPlacementLimit  = 6
	TrainingPlacementLimit = 20
	minimumTrials          = 100
	maximumTrials          = 500
)

var (
	placementLimits         = map[string]int{"farp1": 20}
	scatterDefenderCounts   = []int64{5, 25, 50, 100, 250, 500}
	attritionDefenderCounts = []int64{5, 10, 15, 20}
	defenderCountOptions    = map[string][]int64{
		"farp2": scatterDefenderCounts,
		"farp3": scatterDefenderCounts,
		"farp4": attritionDefenderCounts,
		"farp5": attritionDefenderCounts,
	}
)

type EvaluationSubmission struct {
	PlayerID       string
	Username       string
	LevelID        string
	Algorithm      []any
	Placements     []any
	GenomeSchedule []any
	Trials         int
	GameVersion    string
	Collisions     bool
	CreatedAt      time.Time
}

func PlacementLimitFor(levelID string) int {
	if limit, found := placementLimits[levelID]; found {
		return limit
	}
	return defaultPlacementLimit
}

func ParseEvaluationSubmission(data map[string]any) (EvaluationSubmission, error) {
	fields, err := NewFields("EvaluationSubmit", data,
		[]string{"player_id", "username"},
		[]string{"level_id", "algorithm", "placements", "trials", "game_version", "collisions", "genome_schedule", "created_at"})
	if err != nil {
		return EvaluationSubmission{}, err
	}
	submission := EvaluationSubmission{}
	if submission.CreatedAt, err = ParseCreatedAt(fields.Value("created_at", "")); err != nil {
		return submission, err
	}
	submission.Collisions = values.Truthy(fields.Value("collisions", false))
	levelValue := fields.Value("level_id", "farp")
	levelText, _ := levelValue.(string)
	if submission.GenomeSchedule, err = ValidateGenomeSchedule(fields.Value("genome_schedule", []any{}), levelText); err != nil {
		return submission, err
	}
	if submission.PlayerID, err = requirePlayerID(fields.Value("player_id", nil)); err != nil {
		return submission, err
	}
	if submission.Username, err = requireUsername(fields.Value("username", nil)); err != nil {
		return submission, err
	}
	if submission.LevelID, err = requireText(levelValue, 1, 40, "level_id must be between 1 and 40 characters"); err != nil {
		return submission, err
	}
	if submission.GameVersion, err = requireGameVersion(fields.Value("game_version", "v0.0.4")); err != nil {
		return submission, err
	}
	if submission.Algorithm, err = requireList(fields.Value("algorithm", []any{}), "algorithm must be a list"); err != nil {
		return submission, err
	}
	if submission.Placements, err = requireList(fields.Value("placements", []any{}), "placements must be a list"); err != nil {
		return submission, err
	}
	if err := checkDefenderCount(submission.LevelID, len(submission.Placements)); err != nil {
		return submission, err
	}
	trials, err := pythonInt(fields.Value("trials", minimumTrials))
	if err != nil {
		return submission, err
	}
	trials = max(minimumTrials, trials)
	if trials > maximumTrials {
		return submission, fmt.Errorf("trials must not exceed %d", maximumTrials)
	}
	submission.Trials = int(trials)
	return submission, nil
}

func checkDefenderCount(levelID string, placementCount int) error {
	if options, found := defenderCountOptions[levelID]; found {
		if !slices.Contains(options, int64(placementCount)) {
			choices := make([]string, len(options))
			for index, option := range options {
				choices[index] = fmt.Sprint(option)
			}
			return fmt.Errorf("placements must hold one of %s defenders", strings.Join(choices, ", "))
		}
		return nil
	}
	if limit := PlacementLimitFor(levelID); placementCount > limit {
		return fmt.Errorf("placements must not exceed %d defenders", limit)
	}
	return nil
}

var errNotObject = errors.New("run must be an object")
