package benchmarks

import (
	"bytes"
	"compress/flate"
	"compress/zlib"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
)

const (
	FormatDelta = "delta"
	FormatPlain = "plain"
)

func inflate(data []byte) ([]byte, error) {
	if reader, err := zlib.NewReader(bytes.NewReader(data)); err == nil {
		defer reader.Close()
		if inflated, err := io.ReadAll(reader); err == nil {
			return inflated, nil
		}
	}
	reader := flate.NewReader(bytes.NewReader(data))
	defer reader.Close()
	return io.ReadAll(reader)
}

func decodeJSON(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	return decoder.Decode(target)
}

func DecodeFrames(compressed []byte, framesFormat string) ([]any, error) {
	if len(compressed) == 0 {
		return []any{}, nil
	}
	raw, err := inflate(compressed)
	if err != nil {
		return nil, err
	}
	if framesFormat == FormatPlain {
		var frames []any
		if err := decodeJSON(raw, &frames); err != nil {
			return nil, err
		}
		if frames == nil {
			frames = []any{}
		}
		return frames, nil
	}
	return decodeDelta(raw)
}

func decodeDelta(raw []byte) ([]any, error) {
	var deltas [][]json.Number
	if err := decodeJSON(raw, &deltas); err != nil {
		return nil, err
	}
	if len(deltas) == 0 {
		return []any{}, nil
	}
	frames := make([]any, len(deltas))
	previous := make([]float64, len(deltas[0]))
	integers := true
	for index, number := range deltas[0] {
		value, err := number.Float64()
		if err != nil {
			return nil, err
		}
		previous[index] = value
		integers = integers && isIntegerLiteral(number)
	}
	frames[0] = frameValues(previous, integers)
	for frameIndex := 1; frameIndex < len(deltas); frameIndex++ {
		delta := deltas[frameIndex]
		if len(delta) > len(previous) {
			return nil, errors.New("frame delta is longer than the previous frame")
		}
		current := make([]float64, len(delta))
		for position, number := range delta {
			value, err := number.Float64()
			if err != nil {
				return nil, err
			}
			current[position] = previous[position] + value
			integers = integers && isIntegerLiteral(number)
		}
		frames[frameIndex] = frameValues(current, integers)
		previous = current
	}
	return frames, nil
}

func frameValues(numbers []float64, integers bool) []any {
	frame := make([]any, len(numbers))
	for index, number := range numbers {
		if integers {
			frame[index] = int64(number)
		} else {
			frame[index] = number
		}
	}
	return frame
}

func isIntegerLiteral(number json.Number) bool {
	for _, character := range number.String() {
		if character == '.' || character == 'e' || character == 'E' {
			return false
		}
	}
	return true
}

func encodePlain(frames []any) ([]byte, error) {
	encoded, err := json.Marshal(frames)
	if err != nil {
		return nil, err
	}
	var compressed bytes.Buffer
	writer := zlib.NewWriter(&compressed)
	if _, err := writer.Write(encoded); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return compressed.Bytes(), nil
}

func UnpackBase64(packed string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(packed)
}
