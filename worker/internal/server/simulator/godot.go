package simulator

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"strings"
)

const (
	godotNil        = 0
	godotBool       = 1
	godotInt        = 2
	godotFloat      = 3
	godotString     = 4
	godotVector2    = 5
	godotColor      = 20
	godotDictionary = 27
	godotArray      = 28
	godotEncode64   = 1 << 16

	defaultArenaWidth  = 1280.0
	defaultArenaHeight = 720.0
)

var errGodotTruncated = errors.New("unexpected end of Godot variant data")

type vector2 [2]float64

type godotReader struct {
	data   []byte
	offset int
}

func (reader *godotReader) take(count int) ([]byte, error) {
	end := reader.offset + count
	if count < 0 || end > len(reader.data) {
		return nil, errGodotTruncated
	}
	chunk := reader.data[reader.offset:end]
	reader.offset = end
	return chunk, nil
}

func (reader *godotReader) u32() (uint32, error) {
	chunk, err := reader.take(4)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint32(chunk), nil
}

func (reader *godotReader) u64() (uint64, error) {
	chunk, err := reader.take(8)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint64(chunk), nil
}

func (reader *godotReader) real(wide bool) (float64, error) {
	if wide {
		bits, err := reader.u64()
		return math.Float64frombits(bits), err
	}
	bits, err := reader.u32()
	return float64(math.Float32frombits(bits)), err
}

func (reader *godotReader) text() (string, error) {
	length, err := reader.u32()
	if err != nil {
		return "", err
	}
	raw, err := reader.take(int(length))
	if err != nil {
		return "", err
	}
	if padding := (4 - int(length)%4) % 4; padding > 0 {
		if _, err := reader.take(padding); err != nil {
			return "", err
		}
	}
	return strings.ToValidUTF8(string(raw), "�"), nil
}

func colorChannel(value float64) int {
	return max(0, min(255, int(math.RoundToEven(value*255))))
}

func (reader *godotReader) variant() (any, error) {
	header, err := reader.u32()
	if err != nil {
		return nil, err
	}
	wide := header&godotEncode64 != 0
	switch header & 0xFFFF {
	case godotNil:
		return nil, nil
	case godotBool:
		value, err := reader.u32()
		return value != 0, err
	case godotInt:
		if wide {
			value, err := reader.u64()
			return int64(value), err
		}
		value, err := reader.u32()
		return int64(int32(value)), err
	case godotFloat:
		return reader.real(wide)
	case godotString:
		return reader.text()
	case godotVector2:
		x, err := reader.real(wide)
		if err != nil {
			return nil, err
		}
		y, err := reader.real(wide)
		return vector2{x, y}, err
	case godotColor:
		channels := [4]float64{}
		for index := range channels {
			if channels[index], err = reader.real(false); err != nil {
				return nil, err
			}
		}
		return fmt.Sprintf("#%02x%02x%02x", colorChannel(channels[0]), colorChannel(channels[1]), colorChannel(channels[2])), nil
	case godotDictionary:
		return reader.dictionary()
	case godotArray:
		return reader.array()
	}
	return nil, fmt.Errorf("unsupported Godot variant type: %d", header&0xFFFF)
}

func (reader *godotReader) dictionary() (map[any]any, error) {
	count, err := reader.u32()
	if err != nil {
		return nil, err
	}
	result := map[any]any{}
	for index := uint32(0); index < count&0x7FFFFFFF; index++ {
		key, err := reader.variant()
		if err != nil {
			return nil, err
		}
		value, err := reader.variant()
		if err != nil {
			return nil, err
		}
		if !isHashable(key) {
			return nil, errors.New("unhashable Godot dictionary key")
		}
		result[key] = value
	}
	return result, nil
}

func isHashable(key any) bool {
	switch key.(type) {
	case []any, map[any]any:
		return false
	}
	return true
}

func (reader *godotReader) array() ([]any, error) {
	count, err := reader.u32()
	if err != nil {
		return nil, err
	}
	result := []any{}
	for index := uint32(0); index < count&0x7FFFFFFF; index++ {
		value, err := reader.variant()
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

type runMetadata struct {
	species     []any
	robotCount  int
	frameCount  int
	arenaWidth  float64
	arenaHeight float64
}

func defaultRunMetadata() runMetadata {
	return runMetadata{species: []any{}, arenaWidth: defaultArenaWidth, arenaHeight: defaultArenaHeight}
}

func extractRunMetadata(data []byte) (runMetadata, error) {
	if len(data) < 8 {
		return runMetadata{}, errors.New("file too small to be a Godot store_var payload")
	}
	top, err := (&godotReader{data: data, offset: 4}).variant()
	if err != nil {
		return runMetadata{}, err
	}
	topObject, isObject := top.(map[any]any)
	if !isObject {
		return runMetadata{}, errors.New("expected a Dictionary at the top level")
	}
	var setupValue, framesValue any = topObject, []any{}
	_, hasSetup := topObject["setup"]
	_, hasFrames := topObject["frames"]
	if hasSetup && hasFrames {
		setupValue, framesValue = topObject["setup"], topObject["frames"]
	}
	setup, isObject := setupValue.(map[any]any)
	if !isObject {
		setup = map[any]any{}
	}
	metadata := defaultRunMetadata()
	if metadata.arenaWidth, metadata.arenaHeight, err = arenaDimensions(setup); err != nil {
		return runMetadata{}, err
	}
	metadata.species = speciesFromSetup(setup)
	if placements, isList := lookup(setup, "placements", []any{}).([]any); isList {
		metadata.robotCount = len(placements)
	}
	if frames, isList := framesValue.([]any); isList {
		metadata.frameCount = len(frames)
	}
	return metadata, nil
}

func lookup(object map[any]any, key string, fallback any) any {
	if value, found := object[key]; found {
		return value
	}
	return fallback
}

func godotText(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case nil:
		return "None"
	case bool:
		if typed {
			return "True"
		}
		return "False"
	}
	return fmt.Sprint(value)
}

func speciesFromSetup(setup map[any]any) []any {
	robotTypes, isList := lookup(setup, "robot_types", []any{}).([]any)
	if !isList {
		return []any{}
	}
	species := []any{}
	for _, item := range robotTypes {
		entry, isObject := item.(map[any]any)
		if !isObject {
			continue
		}
		color, isText := entry["color"].(string)
		if !isText {
			color = "#ffffff"
		}
		species = append(species, map[string]any{
			"id": godotText(lookup(entry, "id", "")), "name": godotText(lookup(entry, "name", "")), "color": color,
		})
	}
	return species
}

func arenaDimensions(setup map[any]any) (float64, float64, error) {
	settings, isObject := lookup(setup, "settings", map[any]any{}).(map[any]any)
	if !isObject {
		return defaultArenaWidth, defaultArenaHeight, nil
	}
	width, err := godotFloatValue(lookup(settings, "arena_width", defaultArenaWidth))
	if err != nil {
		return 0, 0, err
	}
	height, err := godotFloatValue(lookup(settings, "arena_height", defaultArenaHeight))
	return width, height, err
}

func godotFloatValue(value any) (float64, error) {
	switch typed := value.(type) {
	case float64:
		return typed, nil
	case int64:
		return float64(typed), nil
	case bool:
		if typed {
			return 1, nil
		}
		return 0, nil
	}
	return 0, errors.New("arena size is not a number")
}
