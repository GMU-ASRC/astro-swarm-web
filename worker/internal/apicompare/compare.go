package apicompare

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
)

const maxDifferencesPerRoute = 12

func decodeJSON(body []byte) (any, error) {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	var value any
	err := decoder.Decode(&value)
	return value, err
}

type differences struct {
	lines []string
	total int
}

func (found *differences) add(path string, format string, arguments ...any) {
	found.total++
	if len(found.lines) < maxDifferencesPerRoute {
		found.lines = append(found.lines, path+": "+fmt.Sprintf(format, arguments...))
	}
}

func numbersEqual(left, right json.Number) bool {
	if left == right {
		return true
	}
	leftValue, leftErr := left.Float64()
	rightValue, rightErr := right.Float64()
	if leftErr != nil || rightErr != nil {
		return false
	}
	if leftValue == rightValue {
		return true
	}
	scale := math.Max(math.Abs(leftValue), math.Abs(rightValue))
	return math.Abs(leftValue-rightValue) <= 1e-9*scale
}

func short(value any) string {
	encoded, _ := json.Marshal(value)
	text := string(encoded)
	if len(text) > 80 {
		return text[:77] + "..."
	}
	return text
}

func compareValues(path string, flask any, golang any, found *differences) {
	switch flaskValue := flask.(type) {
	case map[string]any:
		goValue, isObject := golang.(map[string]any)
		if !isObject {
			found.add(path, "flask has an object, go has %s", short(golang))
			return
		}
		keys := map[string]bool{}
		for key := range flaskValue {
			keys[key] = true
		}
		for key := range goValue {
			keys[key] = true
		}
		sorted := make([]string, 0, len(keys))
		for key := range keys {
			sorted = append(sorted, key)
		}
		sort.Strings(sorted)
		for _, key := range sorted {
			flaskChild, inFlask := flaskValue[key]
			goChild, inGo := goValue[key]
			switch {
			case !inGo:
				found.add(path+"."+key, "missing in go (flask %s)", short(flaskChild))
			case !inFlask:
				found.add(path+"."+key, "extra in go (%s)", short(goChild))
			default:
				compareValues(path+"."+key, flaskChild, goChild, found)
			}
		}
	case []any:
		goValue, isList := golang.([]any)
		if !isList {
			found.add(path, "flask has a list, go has %s", short(golang))
			return
		}
		if len(flaskValue) != len(goValue) {
			found.add(path, "list length flask %d, go %d", len(flaskValue), len(goValue))
		}
		for index := 0; index < min(len(flaskValue), len(goValue)); index++ {
			compareValues(fmt.Sprintf("%s[%d]", path, index), flaskValue[index], goValue[index], found)
		}
	case json.Number:
		goValue, isNumber := golang.(json.Number)
		if !isNumber || !numbersEqual(flaskValue, goValue) {
			found.add(path, "flask %s, go %s", flaskValue, short(golang))
		}
	default:
		if short(flask) != short(golang) {
			found.add(path, "flask %s, go %s", short(flask), short(golang))
		}
	}
}

func baseContentType(value string) string {
	kind, _, _ := strings.Cut(value, ";")
	return strings.TrimSpace(strings.ToLower(kind))
}
