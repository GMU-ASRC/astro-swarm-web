package blocks

import "strconv"

const (
	Done    = true
	Running = false
)

type Block struct {
	Type     string
	Params   map[string]any
	Children []Block
}

type Script struct {
	X      float64
	Y      float64
	Blocks []Block
}

func ParamFloat(params map[string]any, key string, fallback float64) float64 {
	value, present := params[key]
	if !present {
		return fallback
	}
	return toFloat(value, fallback)
}

func ParamString(params map[string]any, key string, fallback string) string {
	value, present := params[key]
	if !present {
		return fallback
	}
	text, ok := value.(string)
	if !ok {
		return fallback
	}
	return text
}

func toFloat(value any, fallback float64) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case float32:
		return float64(typed)
	case int:
		return float64(typed)
	case int64:
		return float64(typed)
	case bool:
		if typed {
			return 1.0
		}
		return 0.0
	case string:
		parsed, err := strconv.ParseFloat(typed, 64)
		if err != nil {
			return fallback
		}
		return parsed
	case nil:
		return 0.0
	}
	return fallback
}

func toSlice(value any) []any {
	list, ok := value.([]any)
	if !ok {
		return nil
	}
	return list
}

func toMap(value any) map[string]any {
	mapping, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	return mapping
}

func stringOf(value any) string {
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return text
}

func floatOf(value any, fallback float64) float64 {
	if value == nil {
		return fallback
	}
	return toFloat(value, fallback)
}

func copyParams(value any) map[string]any {
	source := toMap(value)
	copied := make(map[string]any, len(source))
	for key, item := range source {
		copied[key] = item
	}
	return copied
}
