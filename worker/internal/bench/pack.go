package bench

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/json"
)

func PackFrames(frames [][]int) string {
	if len(frames) == 0 {
		return ""
	}

	deltas := make([][]int, 0, len(frames))
	deltas = append(deltas, frames[0])
	for index := 1; index < len(frames); index++ {
		previous := frames[index-1]
		current := frames[index]
		size := len(current)
		if len(previous) < size {
			size = len(previous)
		}
		delta := make([]int, size)
		for position := 0; position < size; position++ {
			delta[position] = current[position] - previous[position]
		}
		deltas = append(deltas, delta)
	}

	encoded, err := json.Marshal(deltas)
	if err != nil {
		return ""
	}

	var compressed bytes.Buffer
	writer, err := zlib.NewWriterLevel(&compressed, zlib.BestCompression)
	if err != nil {
		return ""
	}
	if _, err := writer.Write(encoded); err != nil {
		return ""
	}
	if err := writer.Close(); err != nil {
		return ""
	}
	return base64.StdEncoding.EncodeToString(compressed.Bytes())
}
