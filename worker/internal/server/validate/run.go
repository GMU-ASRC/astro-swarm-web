package validate

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"astroswarm/worker/internal/server/values"
)

const (
	MaxRunSeconds  = 180
	MaxRunFPS      = 60
	maxStatEntries = 24
	maxStatKeySize = 40
)

var validRunOutcomes = []string{"win", "lose", "timeout"}

type RunLimits struct {
	MaxShips        int     `json:"max_ships"`
	MaxSeconds      int     `json:"max_seconds"`
	FrameSlots      *int    `json:"frame_slots"`
	DefenderOptions []int64 `json:"defender_options,omitempty"`
	Planets         int     `json:"planets,omitempty"`
}

var defaultRunLimits = RunLimits{MaxShips: 6, MaxSeconds: MaxRunSeconds}

var runLimits = map[string]RunLimits{
	"farp6": {MaxShips: 24, MaxSeconds: 300},
	"farp7": {MaxShips: 40, MaxSeconds: 300, DefenderOptions: []int64{10, 25}, Planets: 2},
	"farp8": {MaxShips: 60, MaxSeconds: 400, DefenderOptions: []int64{10, 25, 50}, Planets: 3},
}

func RunLimitsFor(levelID string) RunLimits {
	if limits, found := runLimits[levelID]; found {
		return limits
	}
	return defaultRunLimits
}

type RunSubmission struct {
	PlayerID       string
	Username       string
	LevelID        string
	Run            map[string]any
	Algorithm      []any
	Placements     []any
	GameVersion    string
	GenomeSchedule []any
	CreatedAt      time.Time
}

func ParseRunSubmission(data map[string]any) (RunSubmission, error) {
	fields, err := NewFields("RunSubmit", data,
		[]string{"player_id", "username", "level_id", "run"},
		[]string{"algorithm", "placements", "game_version", "genome_schedule", "created_at"})
	if err != nil {
		return RunSubmission{}, err
	}
	submission := RunSubmission{}
	if submission.CreatedAt, err = ParseCreatedAt(fields.Value("created_at", "")); err != nil {
		return submission, err
	}
	levelValue := fields.Value("level_id", nil)
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
	limits := RunLimitsFor(submission.LevelID)
	if submission.Placements, err = requireList(fields.Value("placements", []any{}), "placements must be a list"); err != nil {
		return submission, err
	}
	if len(submission.Placements) > limits.MaxShips {
		return submission, fmt.Errorf("placements must not exceed %d ships", limits.MaxShips)
	}
	run, isObject := fields.Value("run", nil).(map[string]any)
	if !isObject {
		return submission, errNotObject
	}
	if err := validateRun(run, limits, len(submission.Placements)); err != nil {
		return submission, err
	}
	submission.Run = run
	return submission, nil
}

func validateRun(run map[string]any, limits RunLimits, placementCount int) error {
	maxFrames := limits.MaxSeconds*MaxRunFPS + 60
	outcome, _ := run["outcome"].(string)
	if !containsText(validRunOutcomes, outcome) {
		return errors.New("run.outcome must be one of ('win', 'lose', 'timeout')")
	}
	frames, isList := run["frames"].([]any)
	if !isList || len(frames) == 0 {
		return errors.New("run.frames must be a non-empty list")
	}
	if len(frames) > maxFrames {
		return fmt.Errorf("run.frames must not exceed %d frames", maxFrames)
	}
	width, err := frameWidth(run, limits, placementCount)
	if err != nil {
		return err
	}
	for _, frame := range frames {
		frameValues, isList := frame.([]any)
		if !isList || len(frameValues) != width {
			return fmt.Errorf("each run frame must hold %d values", width)
		}
	}
	fps, err := pythonInt(valueOr(run, "fps", int64(30)))
	if err != nil {
		return err
	}
	if fps < 1 || fps > MaxRunFPS {
		return fmt.Errorf("run.fps must be between 1 and %d", MaxRunFPS)
	}
	run["fps"] = fps
	goalTime, err := pythonFloat(valueOr(run, "goal_time", -1.0))
	if err != nil {
		return err
	}
	if goalTime > float64(limits.MaxSeconds) {
		return fmt.Errorf("run.goal_time must not exceed %d seconds", limits.MaxSeconds)
	}
	if float64(len(frames))/float64(fps) > float64(limits.MaxSeconds+5) {
		return fmt.Errorf("run must not exceed %d seconds", limits.MaxSeconds)
	}
	if err := validateGenomeSwitches(valueOr(run, "genome_switches", []any{}), len(frames)); err != nil {
		return err
	}
	return cleanStats(run)
}

func frameWidth(run map[string]any, limits RunLimits, placementCount int) (int, error) {
	if len(limits.DefenderOptions) == 0 {
		return placementCount*3 + 3, nil
	}
	defenders, isNumber := values.Number(run["defenders"])
	matchesOption := false
	for _, option := range limits.DefenderOptions {
		if isNumber && defenders == float64(option) {
			matchesOption = true
		}
	}
	if !matchesOption || float64(placementCount) > defenders {
		choices := make([]string, len(limits.DefenderOptions))
		for index, option := range limits.DefenderOptions {
			choices[index] = fmt.Sprint(option)
		}
		return 0, fmt.Errorf("run.defenders must be one of %s and match the placements", strings.Join(choices, ", "))
	}
	defenderCount := int(defenders)
	slots := defenderCount + defenderCount/limits.Planets + 1
	return slots * 3, nil
}

func validateGenomeSwitches(value any, frameCount int) error {
	switches, isList := value.([]any)
	if !isList || len(switches) >= MaxGenomeStages {
		return fmt.Errorf("run.genome_switches must be a list of at most %d frames", MaxGenomeStages-1)
	}
	for _, switchValue := range switches {
		frameIndex, isWhole := values.Integer(switchValue)
		if !isWhole || frameIndex < 0 || frameIndex > int64(frameCount) {
			return errors.New("run.genome_switches must hold frame indexes within the run")
		}
	}
	return nil
}

func cleanStats(run map[string]any) error {
	statsValue, present := run["stats"]
	if !present || statsValue == nil {
		return nil
	}
	stats, isObject := statsValue.(map[string]any)
	if !isObject {
		return errors.New("run.stats must be an object")
	}
	keys := make([]string, 0, len(stats))
	for key := range stats {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	if len(keys) > maxStatEntries {
		keys = keys[:maxStatEntries]
	}
	cleaned := map[string]any{}
	for _, key := range keys {
		value := stats[key]
		if flag, isBool := value.(bool); isBool {
			cleaned[values.Truncate(key, maxStatKeySize)] = boolToFloat(flag)
			continue
		}
		if number, isNumber := values.Number(value); isNumber {
			cleaned[values.Truncate(key, maxStatKeySize)] = number
		}
	}
	run["stats"] = cleaned
	return nil
}

func boolToFloat(flag bool) float64 {
	if flag {
		return 1
	}
	return 0
}

func valueOr(object map[string]any, key string, fallback any) any {
	if value, found := object[key]; found {
		return value
	}
	return fallback
}

func containsText(options []string, text string) bool {
	for _, option := range options {
		if option == text {
			return true
		}
	}
	return false
}
