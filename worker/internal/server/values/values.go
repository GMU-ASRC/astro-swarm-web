package values

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode/utf8"
)

var ErrNotConvertible = errors.New("value cannot be converted")

func Integer(value any) (int64, bool) {
	switch typed := value.(type) {
	case json.Number:
		text := typed.String()
		if strings.ContainsAny(text, ".eE") {
			return 0, false
		}
		parsed, err := strconv.ParseInt(text, 10, 64)
		return parsed, err == nil
	case int:
		return int64(typed), true
	case int64:
		return typed, true
	}
	return 0, false
}

func Number(value any) (float64, bool) {
	switch typed := value.(type) {
	case json.Number:
		parsed, err := typed.Float64()
		return parsed, err == nil
	case float64:
		return typed, true
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	}
	return 0, false
}

func String(value any) (string, bool) {
	text, isText := value.(string)
	return text, isText
}

func List(value any) ([]any, bool) {
	list, isList := value.([]any)
	return list, isList
}

func Object(value any) (map[string]any, bool) {
	object, isObject := value.(map[string]any)
	return object, isObject
}

func Length(text string) int {
	return utf8.RuneCountInString(text)
}

func Truncate(text string, limit int) string {
	if utf8.RuneCountInString(text) <= limit {
		return text
	}
	return string([]rune(text)[:limit])
}

func ToInt(value any) (int64, error) {
	switch typed := value.(type) {
	case bool:
		if typed {
			return 1, nil
		}
		return 0, nil
	case string:
		parsed, err := strconv.ParseInt(strings.TrimSpace(typed), 10, 64)
		if err != nil {
			return 0, ErrNotConvertible
		}
		return parsed, nil
	}
	if whole, isWhole := Integer(value); isWhole {
		return whole, nil
	}
	if number, isNumber := Number(value); isNumber {
		if math.IsNaN(number) || math.IsInf(number, 0) {
			return 0, ErrNotConvertible
		}
		return int64(number), nil
	}
	return 0, ErrNotConvertible
}

func ToFloat(value any) (float64, error) {
	switch typed := value.(type) {
	case bool:
		if typed {
			return 1, nil
		}
		return 0, nil
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		if err != nil {
			return 0, ErrNotConvertible
		}
		return parsed, nil
	}
	if number, isNumber := Number(value); isNumber {
		return number, nil
	}
	return 0, ErrNotConvertible
}

func ToText(value any) string {
	switch typed := value.(type) {
	case nil:
		return "None"
	case string:
		return typed
	case bool:
		if typed {
			return "True"
		}
		return "False"
	case json.Number:
		return typed.String()
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return string(encoded)
}

func Truthy(value any) bool {
	switch typed := value.(type) {
	case nil:
		return false
	case bool:
		return typed
	case string:
		return typed != ""
	case json.Number:
		number, err := typed.Float64()
		return err != nil || number != 0
	case float64:
		return typed != 0
	case []any:
		return len(typed) > 0
	case map[string]any:
		return len(typed) > 0
	}
	return true
}

func RoundTo(value float64, digits int) float64 {
	scale := math.Pow(10, float64(digits))
	return math.RoundToEven(value*scale) / scale
}

func FloatRepr(value float64) string {
	if math.IsInf(value, 1) {
		return "Infinity"
	}
	if math.IsInf(value, -1) {
		return "-Infinity"
	}
	if math.IsNaN(value) {
		return "NaN"
	}
	scientific := strconv.FormatFloat(value, 'e', -1, 64)
	mantissa, exponentText, _ := strings.Cut(scientific, "e")
	exponent, _ := strconv.Atoi(exponentText)
	if exponent >= -4 && exponent < 16 {
		fixed := strconv.FormatFloat(value, 'f', -1, 64)
		if !strings.Contains(fixed, ".") {
			fixed += ".0"
		}
		return fixed
	}
	sign := "+"
	if exponent < 0 {
		sign = "-"
		exponent = -exponent
	}
	return fmt.Sprintf("%se%s%02d", mantissa, sign, exponent)
}
