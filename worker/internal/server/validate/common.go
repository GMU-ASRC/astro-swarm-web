package validate

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	"astroswarm/worker/internal/server/pytime"
	"astroswarm/worker/internal/server/values"
)

const (
	maxClientTimestampAge    = 365 * 24 * time.Hour
	maxClientTimestampLength = 40
	genomeNameMaxLength      = 24
	MaxGenomeStages          = 9
)

var (
	genomeSwitchLevels = []string{"farp4", "farp5", "farp7", "farp8"}
	genomeColorPattern = regexp.MustCompile(`^[0-9a-fA-F]{6}$`)
)

type Fields struct {
	typeName string
	data     map[string]any
}

func NewFields(typeName string, data map[string]any, required []string, optional []string) (Fields, error) {
	allowed := append(append([]string{}, required...), optional...)
	for key := range data {
		if !slices.Contains(allowed, key) {
			return Fields{}, fmt.Errorf("%s.__init__() got an unexpected keyword argument '%s'", typeName, key)
		}
	}
	missing := []string{}
	for _, key := range required {
		if _, found := data[key]; !found {
			missing = append(missing, "'"+key+"'")
		}
	}
	if len(missing) == 1 {
		return Fields{}, fmt.Errorf("%s.__init__() missing 1 required positional argument: %s", typeName, missing[0])
	}
	if len(missing) > 1 {
		listed := strings.Join(missing[:len(missing)-1], ", ") + " and " + missing[len(missing)-1]
		return Fields{}, fmt.Errorf("%s.__init__() missing %d required positional arguments: %s", typeName, len(missing), listed)
	}
	return Fields{typeName: typeName, data: data}, nil
}

func (fields Fields) Value(key string, fallback any) any {
	if value, found := fields.data[key]; found {
		return value
	}
	return fallback
}

func ParseCreatedAt(value any) (time.Time, error) {
	now := pytime.Now()
	if value == nil || value == "" {
		return now, nil
	}
	text, isText := value.(string)
	if !isText || len(text) > maxClientTimestampLength {
		return time.Time{}, errors.New("created_at must be an ISO 8601 date string")
	}
	stamp, parsed := pytime.ParseISO(text)
	if !parsed {
		return time.Time{}, errors.New("created_at must be an ISO 8601 date string")
	}
	earliest := now.Add(-maxClientTimestampAge)
	if stamp.After(now) {
		stamp = now
	}
	if stamp.Before(earliest) {
		stamp = earliest
	}
	return stamp, nil
}

func requireText(value any, low int, high int, message string) (string, error) {
	text, isText := value.(string)
	if !isText {
		return "", errors.New(message)
	}
	length := values.Length(text)
	if length < low || length > high {
		return "", errors.New(message)
	}
	return text, nil
}

func requirePlayerID(value any) (string, error) {
	text, isText := value.(string)
	if !isText || values.Length(text) != 36 {
		return "", errors.New("player_id must be exactly 36 characters")
	}
	return text, nil
}

func requireUsername(value any) (string, error) {
	return requireText(value, 1, 30, "username must be between 1 and 30 characters")
}

func requireGameVersion(value any) (string, error) {
	return requireText(value, 0, 20, "game_version must be a string up to 20 characters")
}

func requireList(value any, message string) ([]any, error) {
	list, isList := value.([]any)
	if !isList {
		return nil, errors.New(message)
	}
	return list, nil
}

func pythonInt(value any) (int64, error) {
	converted, err := values.ToInt(value)
	if err != nil {
		return 0, fmt.Errorf("int() argument must be a string, a bytes-like object or a real number, not '%s'", pythonTypeName(value))
	}
	return converted, nil
}

func pythonFloat(value any) (float64, error) {
	converted, err := values.ToFloat(value)
	if err != nil {
		return 0, fmt.Errorf("float() argument must be a string or a real number, not '%s'", pythonTypeName(value))
	}
	return converted, nil
}

func pythonTypeName(value any) string {
	switch value.(type) {
	case nil:
		return "NoneType"
	case []any:
		return "list"
	case map[string]any:
		return "dict"
	case string:
		return "str"
	case bool:
		return "bool"
	}
	return "float"
}

func ValidateGenomeSchedule(schedule any, levelID string) ([]any, error) {
	stages, isList := schedule.([]any)
	if !isList {
		return nil, errors.New("genome_schedule must be a list")
	}
	if len(stages) > 1 && !slices.Contains(genomeSwitchLevels, levelID) {
		return nil, errors.New("genome switches are only allowed on levels 4 and up")
	}
	if len(stages) > MaxGenomeStages {
		return nil, fmt.Errorf("genome_schedule must not exceed %d stages", MaxGenomeStages)
	}
	var previous *int64
	for _, stageValue := range stages {
		stage, isObject := stageValue.(map[string]any)
		if !isObject {
			return nil, errors.New("each genome stage must be an object")
		}
		if _, err := requireText(stage["name"], 1, genomeNameMaxLength, fmt.Sprintf("genome stage name must be between 1 and %d characters", genomeNameMaxLength)); err != nil {
			return nil, err
		}
		if _, isList := stage["algorithm"].([]any); !isList {
			return nil, errors.New("genome stage algorithm must be a list")
		}
		if color, present := stage["color"]; present && color != nil {
			text, isText := color.(string)
			if !isText || !genomeColorPattern.MatchString(text) {
				return nil, errors.New("genome stage color must be a six digit hex color")
			}
		}
		atDefenders, isWhole := values.Integer(stage["at_defenders"])
		if !isWhole || atDefenders < 0 {
			return nil, errors.New("genome stage at_defenders must be a whole number")
		}
		if previous != nil && atDefenders >= *previous {
			return nil, errors.New("each genome switch must happen at fewer defenders than the last")
		}
		previous = &atDefenders
	}
	return stages, nil
}
