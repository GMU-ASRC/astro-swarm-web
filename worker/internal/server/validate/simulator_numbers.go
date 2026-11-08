package validate

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"

	"astroswarm/worker/internal/server/values"
)

func isIntegerText(number json.Number) bool {
	return !strings.ContainsAny(number.String(), ".eE")
}

func simulatorNumber(value any, label string) (json.Number, error) {
	switch typed := value.(type) {
	case json.Number:
		parsed, err := typed.Float64()
		if err != nil || math.IsInf(parsed, 0) || math.IsNaN(parsed) {
			return "", fmt.Errorf("%s must be finite", label)
		}
		return typed, nil
	case int:
		return json.Number(strconv.Itoa(typed)), nil
	case int64:
		return json.Number(strconv.FormatInt(typed, 10)), nil
	case float64:
		if math.IsInf(typed, 0) || math.IsNaN(typed) {
			return "", fmt.Errorf("%s must be finite", label)
		}
		return json.Number(values.FloatRepr(typed)), nil
	}
	return "", fmt.Errorf("%s must be a number", label)
}

func numberFloat(number json.Number) float64 {
	parsed, _ := number.Float64()
	return parsed
}

func roundNumber(number json.Number, digits int) json.Number {
	if isIntegerText(number) {
		return number
	}
	rounded, _ := strconv.ParseFloat(strconv.FormatFloat(numberFloat(number), 'f', digits, 64), 64)
	return json.Number(values.FloatRepr(rounded))
}

func roundedNumber(value any, label string, digits int) (json.Number, error) {
	number, err := simulatorNumber(value, label)
	if err != nil {
		return "", err
	}
	return roundNumber(number, digits), nil
}

func integerNumber(value any, label string) (int64, error) {
	number, err := simulatorNumber(value, label)
	if err != nil {
		return 0, err
	}
	if isIntegerText(number) {
		if parsed, err := number.Int64(); err == nil {
			return parsed, nil
		}
	}
	return int64(math.Trunc(numberFloat(number))), nil
}

func simulatorText(value any, label string, limit int) (string, error) {
	switch value.(type) {
	case nil:
		return "", nil
	case string, json.Number, int, int64, float64:
	default:
		return "", fmt.Errorf("%s must be text", label)
	}
	text := strings.TrimSpace(values.ToText(value))
	if values.Length(text) > limit {
		return "", fmt.Errorf("%s must be at most %d characters", label, limit)
	}
	return text, nil
}

func simulatorPoint(value any) ([]any, error) {
	list, isList := value.([]any)
	if !isList || len(list) != 2 {
		return nil, fmt.Errorf("points must be [x, y]")
	}
	x, err := roundedNumber(list[0], "x", 1)
	if err != nil {
		return nil, err
	}
	y, err := roundedNumber(list[1], "y", 1)
	if err != nil {
		return nil, err
	}
	return []any{x, y}, nil
}

func objectValue(object map[string]any, key string, fallback any) any {
	if value, found := object[key]; found {
		return value
	}
	return fallback
}
