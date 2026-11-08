package benchmarks

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"unicode/utf16"

	"astroswarm/worker/internal/server/values"
)

func PythonJSON(value any) (string, error) {
	var builder strings.Builder
	if err := writePythonJSON(&builder, value); err != nil {
		return "", err
	}
	return builder.String(), nil
}

func writePythonJSON(builder *strings.Builder, value any) error {
	switch typed := value.(type) {
	case nil:
		builder.WriteString("null")
	case bool:
		if typed {
			builder.WriteString("true")
		} else {
			builder.WriteString("false")
		}
	case string:
		writePythonString(builder, typed)
	case int:
		builder.WriteString(strconv.Itoa(typed))
	case int64:
		builder.WriteString(strconv.FormatInt(typed, 10))
	case float64:
		builder.WriteString(values.FloatRepr(typed))
	case json.Number:
		text := typed.String()
		if isIntegerLiteral(typed) {
			if parsed, err := strconv.ParseInt(text, 10, 64); err == nil {
				builder.WriteString(strconv.FormatInt(parsed, 10))
				return nil
			}
			builder.WriteString(strings.TrimPrefix(text, "+"))
			return nil
		}
		number, err := typed.Float64()
		if err != nil {
			return err
		}
		builder.WriteString(values.FloatRepr(number))
	case []any:
		builder.WriteString("[")
		for index, item := range typed {
			if index > 0 {
				builder.WriteString(",")
			}
			if err := writePythonJSON(builder, item); err != nil {
				return err
			}
		}
		builder.WriteString("]")
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		builder.WriteString("{")
		for index, key := range keys {
			if index > 0 {
				builder.WriteString(",")
			}
			writePythonString(builder, key)
			builder.WriteString(":")
			if err := writePythonJSON(builder, typed[key]); err != nil {
				return err
			}
		}
		builder.WriteString("}")
	default:
		encoded, err := json.Marshal(value)
		if err != nil {
			return err
		}
		var generic any
		if err := decodeJSON(encoded, &generic); err != nil {
			return err
		}
		return writePythonJSON(builder, generic)
	}
	return nil
}

func writePythonString(builder *strings.Builder, text string) {
	builder.WriteByte('"')
	for _, character := range text {
		switch {
		case character == '"':
			builder.WriteString(`\"`)
		case character == '\\':
			builder.WriteString(`\\`)
		case character == '\n':
			builder.WriteString(`\n`)
		case character == '\r':
			builder.WriteString(`\r`)
		case character == '\t':
			builder.WriteString(`\t`)
		case character == '\b':
			builder.WriteString(`\b`)
		case character == '\f':
			builder.WriteString(`\f`)
		case character < 0x20:
			fmt.Fprintf(builder, `\u%04x`, character)
		case character < 0x7f:
			builder.WriteRune(character)
		case character == 0x7f:
			builder.WriteRune(character)
		case character > 0xffff:
			high, low := utf16.EncodeRune(character)
			fmt.Fprintf(builder, `\u%04x\u%04x`, high, low)
		default:
			fmt.Fprintf(builder, `\u%04x`, character)
		}
	}
	builder.WriteByte('"')
}
