package validate

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"astroswarm/worker/internal/server/values"
)

const (
	MaxSurviveSeconds = 180
	maxAPMSamples     = 240
)

var (
	validSurviveOutcomes = []string{"player1", "player2", "tie"}
	survivePlayerFields  = []string{"evaders_through", "defenders_at_end", "herded_total", "freezes_used", "actions", "apm_peak"}
)

type TrainingSubmission struct {
	PlayerID    string
	Username    string
	Circliness  float64
	Algorithm   []any
	Placements  []any
	GameVersion string
	CreatedAt   time.Time
}

func ParseTrainingSubmission(data map[string]any) (TrainingSubmission, error) {
	fields, err := NewFields("TrainingScoreSubmit", data,
		[]string{"player_id", "username", "circliness"},
		[]string{"algorithm", "placements", "game_version", "created_at"})
	if err != nil {
		return TrainingSubmission{}, err
	}
	submission := TrainingSubmission{}
	if submission.CreatedAt, err = ParseCreatedAt(fields.Value("created_at", "")); err != nil {
		return submission, err
	}
	if submission.PlayerID, err = requirePlayerID(fields.Value("player_id", nil)); err != nil {
		return submission, err
	}
	if submission.Username, err = requireUsername(fields.Value("username", nil)); err != nil {
		return submission, err
	}
	circliness, isNumber := values.Number(fields.Value("circliness", nil))
	if !isNumber {
		return submission, errors.New("circliness must be a number")
	}
	if circliness < 0 || circliness > 1 {
		return submission, errors.New("circliness must be between 0 and 1")
	}
	submission.Circliness = circliness
	if submission.GameVersion, err = requireGameVersion(fields.Value("game_version", "v0.0.9")); err != nil {
		return submission, err
	}
	if submission.Algorithm, err = requireList(fields.Value("algorithm", []any{}), "algorithm must be a list"); err != nil {
		return submission, err
	}
	if submission.Placements, err = requireList(fields.Value("placements", []any{}), "placements must be a list"); err != nil {
		return submission, err
	}
	if len(submission.Placements) < 2 || len(submission.Placements) > TrainingPlacementLimit {
		return submission, fmt.Errorf("placements must hold between 2 and %d defenders", TrainingPlacementLimit)
	}
	return submission, nil
}

type LeaderboardSubmission struct {
	PlayerID    string
	Username    string
	TimeSeconds float64
	Algorithm   any
}

func ParseLeaderboardSubmission(data map[string]any) (LeaderboardSubmission, error) {
	fields, err := NewFields("LeaderboardSubmit", data,
		[]string{"player_id", "username", "time_seconds"},
		[]string{"algorithm"})
	if err != nil {
		return LeaderboardSubmission{}, err
	}
	submission := LeaderboardSubmission{Algorithm: fields.Value("algorithm", []any{})}
	if submission.PlayerID, err = requirePlayerID(fields.Value("player_id", nil)); err != nil {
		return submission, err
	}
	if submission.Username, err = requireUsername(fields.Value("username", nil)); err != nil {
		return submission, err
	}
	timeValue := fields.Value("time_seconds", nil)
	seconds, isNumber := values.Number(timeValue)
	if flag, isBool := timeValue.(bool); isBool {
		seconds, isNumber = boolToFloat(flag), true
	}
	if !isNumber || seconds < 2.0 || seconds > 90.0 {
		return submission, errors.New("time_seconds must be between 2.0 and 90.0")
	}
	submission.TimeSeconds = seconds
	return submission, nil
}

type SurviveSubmission struct {
	PlayerID         string
	Username         string
	Outcome          string
	Players          []map[string]any
	Duration         int64
	GameVersion      string
	APMBucketSeconds int64
	APMTimes         []int64
	APMPlayer1       []int64
	APMPlayer2       []int64
	EvadersSpawned   int64
	WildRemaining    int64
	FirstTarget      int64
	CreatedAt        time.Time
}

func ParseSurviveSubmission(data map[string]any) (SurviveSubmission, error) {
	fields, err := NewFields("SurviveMatchSubmit", data,
		[]string{"player_id", "username", "outcome"},
		[]string{"players", "duration", "game_version", "apm_bucket_seconds", "apm_times", "apm_player1", "apm_player2", "evaders_spawned", "wild_remaining", "first_target", "created_at"})
	if err != nil {
		return SurviveSubmission{}, err
	}
	submission := SurviveSubmission{}
	if submission.CreatedAt, err = ParseCreatedAt(fields.Value("created_at", "")); err != nil {
		return submission, err
	}
	if submission.PlayerID, err = requirePlayerID(fields.Value("player_id", nil)); err != nil {
		return submission, err
	}
	if submission.Username, err = requireUsername(fields.Value("username", nil)); err != nil {
		return submission, err
	}
	if submission.GameVersion, err = requireGameVersion(fields.Value("game_version", "v0.0.5")); err != nil {
		return submission, err
	}
	outcome, _ := fields.Value("outcome", nil).(string)
	if !containsText(validSurviveOutcomes, outcome) {
		return submission, errors.New("outcome must be one of ('player1', 'player2', 'tie')")
	}
	submission.Outcome = outcome
	if submission.Duration, err = pythonInt(fields.Value("duration", int64(0))); err != nil {
		return submission, err
	}
	if submission.Duration < 0 || submission.Duration > MaxSurviveSeconds {
		return submission, fmt.Errorf("duration must be between 0 and %d seconds", MaxSurviveSeconds)
	}
	players, isList := fields.Value("players", []any{}).([]any)
	if !isList || len(players) != 2 {
		return submission, errors.New("players must hold exactly 2 entries")
	}
	for slot, entry := range players {
		cleaned, err := cleanSurvivePlayer(entry, slot+1)
		if err != nil {
			return submission, err
		}
		submission.Players = append(submission.Players, cleaned)
	}
	if submission.APMBucketSeconds, err = pythonInt(fields.Value("apm_bucket_seconds", int64(5))); err != nil {
		return submission, err
	}
	if submission.APMBucketSeconds < 1 || submission.APMBucketSeconds > 60 {
		return submission, errors.New("apm_bucket_seconds must be between 1 and 60")
	}
	series := map[string]*[]int64{"apm_times": &submission.APMTimes, "apm_player1": &submission.APMPlayer1, "apm_player2": &submission.APMPlayer2}
	for _, label := range []string{"apm_times", "apm_player1", "apm_player2"} {
		if *series[label], err = cleanSeries(fields.Value(label, []any{}), label); err != nil {
			return submission, err
		}
	}
	if len(submission.APMPlayer1) != len(submission.APMTimes) || len(submission.APMPlayer2) != len(submission.APMTimes) {
		return submission, errors.New("apm series must be the same length as apm_times")
	}
	for label, target := range map[string]*int64{"evaders_spawned": &submission.EvadersSpawned, "wild_remaining": &submission.WildRemaining} {
		value, err := pythonInt(fields.Value(label, int64(0)))
		if err != nil {
			return submission, err
		}
		*target = max(0, value)
	}
	firstTarget, err := pythonInt(fields.Value("first_target", int64(1)))
	if err != nil {
		return submission, err
	}
	submission.FirstTarget = 1
	if firstTarget == 2 {
		submission.FirstTarget = 2
	}
	return submission, nil
}

func cleanSeries(value any, label string) ([]int64, error) {
	list, isList := value.([]any)
	if !isList {
		return nil, fmt.Errorf("%s must be a list", label)
	}
	if len(list) > maxAPMSamples {
		return nil, fmt.Errorf("%s must not exceed %d samples", label, maxAPMSamples)
	}
	cleaned := make([]int64, 0, len(list))
	for _, item := range list {
		converted, err := pythonInt(item)
		if err != nil {
			return nil, err
		}
		cleaned = append(cleaned, max(0, converted))
	}
	return cleaned, nil
}

func cleanSurvivePlayer(entry any, slot int) (map[string]any, error) {
	player, isObject := entry.(map[string]any)
	if !isObject {
		return nil, errors.New("each player entry must be an object")
	}
	name := strings.TrimSpace(values.ToText(valueOr(player, "name", "")))
	if name == "" {
		name = fmt.Sprintf("Player %d", slot)
	}
	cleaned := map[string]any{"slot": slot, "name": values.Truncate(name, 30)}
	for _, key := range survivePlayerFields {
		converted, err := pythonInt(valueOr(player, key, int64(0)))
		if err != nil {
			return nil, err
		}
		cleaned[key] = max(0, converted)
	}
	average, err := pythonFloat(valueOr(player, "apm_average", 0.0))
	if err != nil {
		return nil, err
	}
	cleaned["apm_average"] = max(0.0, values.RoundTo(average, 1))
	return cleaned, nil
}
