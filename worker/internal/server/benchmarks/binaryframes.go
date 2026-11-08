package benchmarks

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
)

const (
	FormatBinary      = "asr1"
	binaryMagic       = "ASR1"
	binaryIntegerKind = 0
	binaryFloatKind   = 1
	maxBinaryFrames   = 10_000_000
	maxBinaryValues   = 100_000_000
)

var (
	ErrNotFlatFrames = errors.New("frames are not flat lists of numbers")
	errBinaryCorrupt = errors.New("binary replay frames are corrupt")
)

type flatFrames struct {
	integers bool
	ints     [][]int64
	floats   [][]float64
}

func flatten(frames []any) (flatFrames, error) {
	flat := flatFrames{integers: true}
	numbers := make([][]json.Number, 0, len(frames))
	floatFrames := make([][]float64, 0, len(frames))
	for _, frame := range frames {
		values, isList := frame.([]any)
		if !isList {
			return flat, ErrNotFlatFrames
		}
		row := make([]float64, len(values))
		literal := make([]json.Number, len(values))
		for index, value := range values {
			switch typed := value.(type) {
			case int64:
				row[index], literal[index] = float64(typed), json.Number(fmt.Sprint(typed))
			case int:
				row[index], literal[index] = float64(typed), json.Number(fmt.Sprint(typed))
			case float64:
				row[index] = typed
				flat.integers = false
			case json.Number:
				parsed, err := typed.Float64()
				if err != nil {
					return flat, ErrNotFlatFrames
				}
				row[index], literal[index] = parsed, typed
				if !isIntegerLiteral(typed) {
					flat.integers = false
				}
			default:
				return flat, ErrNotFlatFrames
			}
		}
		floatFrames = append(floatFrames, row)
		numbers = append(numbers, literal)
	}
	if !flat.integers {
		flat.floats = floatFrames
		return flat, nil
	}
	flat.ints = make([][]int64, len(numbers))
	for frameIndex, literal := range numbers {
		flat.ints[frameIndex] = make([]int64, len(literal))
		for index, number := range literal {
			parsed, err := number.Int64()
			if err != nil {
				return flat, ErrNotFlatFrames
			}
			flat.ints[frameIndex][index] = parsed
		}
	}
	return flat, nil
}

func EncodeBinaryFrames(frames []any) ([]byte, error) {
	flat, err := flatten(frames)
	if err != nil {
		return nil, err
	}
	var payload bytes.Buffer
	payload.WriteString(binaryMagic)
	scratch := make([]byte, binary.MaxVarintLen64)
	writeUvarint := func(value uint64) {
		payload.Write(scratch[:binary.PutUvarint(scratch, value)])
	}
	writeVarint := func(value int64) {
		payload.Write(scratch[:binary.PutVarint(scratch, value)])
	}
	if flat.integers {
		payload.WriteByte(binaryIntegerKind)
		writeUvarint(uint64(len(flat.ints)))
		var previous []int64
		for _, frame := range flat.ints {
			writeUvarint(uint64(len(frame)))
			for index, value := range frame {
				if index < len(previous) {
					writeVarint(value - previous[index])
				} else {
					writeVarint(value)
				}
			}
			previous = frame
		}
	} else {
		payload.WriteByte(binaryFloatKind)
		writeUvarint(uint64(len(flat.floats)))
		bits := make([]byte, 8)
		for _, frame := range flat.floats {
			writeUvarint(uint64(len(frame)))
			for _, value := range frame {
				binary.LittleEndian.PutUint64(bits, math.Float64bits(value))
				payload.Write(bits)
			}
		}
	}
	var compressed bytes.Buffer
	writer := zlib.NewWriter(&compressed)
	if _, err := writer.Write(payload.Bytes()); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return compressed.Bytes(), nil
}

func DecodeBinaryFrames(data []byte) ([]any, error) {
	reader, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	payload, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	if len(payload) < len(binaryMagic)+1 || string(payload[:len(binaryMagic)]) != binaryMagic {
		return nil, errBinaryCorrupt
	}
	kind := payload[len(binaryMagic)]
	cursor := bytes.NewReader(payload[len(binaryMagic)+1:])
	frameCount, err := binary.ReadUvarint(cursor)
	if err != nil || frameCount > maxBinaryFrames {
		return nil, errBinaryCorrupt
	}
	frames := make([]any, 0, frameCount)
	var previous []int64
	totalValues := uint64(0)
	for frameIndex := uint64(0); frameIndex < frameCount; frameIndex++ {
		length, err := binary.ReadUvarint(cursor)
		totalValues += length
		if err != nil || totalValues > maxBinaryValues {
			return nil, errBinaryCorrupt
		}
		frame := make([]any, length)
		switch kind {
		case binaryIntegerKind:
			current := make([]int64, length)
			for index := range current {
				value, err := binary.ReadVarint(cursor)
				if err != nil {
					return nil, errBinaryCorrupt
				}
				if index < len(previous) {
					value += previous[index]
				}
				current[index], frame[index] = value, value
			}
			previous = current
		case binaryFloatKind:
			bits := make([]byte, 8)
			for index := range frame {
				if _, err := io.ReadFull(cursor, bits); err != nil {
					return nil, errBinaryCorrupt
				}
				frame[index] = math.Float64frombits(binary.LittleEndian.Uint64(bits))
			}
		default:
			return nil, errBinaryCorrupt
		}
		frames = append(frames, frame)
	}
	if cursor.Len() != 0 {
		return nil, errBinaryCorrupt
	}
	return frames, nil
}

func numericValue(value any) (float64, bool, bool) {
	switch typed := value.(type) {
	case int64:
		return float64(typed), true, true
	case int:
		return float64(typed), true, true
	case float64:
		return typed, false, true
	case json.Number:
		parsed, err := typed.Float64()
		return parsed, isIntegerLiteral(typed), err == nil
	}
	return 0, false, false
}

func FramesMatch(original []any, converted []any) error {
	if len(original) != len(converted) {
		return fmt.Errorf("frame count %d became %d", len(original), len(converted))
	}
	for frameIndex := range original {
		left, leftIsList := original[frameIndex].([]any)
		right, rightIsList := converted[frameIndex].([]any)
		if !leftIsList || !rightIsList || len(left) != len(right) {
			return fmt.Errorf("frame %d changed length", frameIndex)
		}
		for index := range left {
			leftValue, _, leftOK := numericValue(left[index])
			rightValue, _, rightOK := numericValue(right[index])
			if !leftOK || !rightOK || math.Float64bits(leftValue) != math.Float64bits(rightValue) {
				return fmt.Errorf("frame %d value %d changed from %v to %v", frameIndex, index, left[index], right[index])
			}
		}
	}
	return nil
}
