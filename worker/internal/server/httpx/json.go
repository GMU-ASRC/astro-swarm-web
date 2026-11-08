package httpx

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

const maxJSONBodyBytes = 64 * 1024 * 1024

func WriteJSON(w http.ResponseWriter, status int, value any) {
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"could not encode response","status":500}` + "\n"))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(buffer.Bytes())
}

func ReadBody(r *http.Request, limit int64) ([]byte, error) {
	if limit <= 0 {
		limit = maxJSONBodyBytes
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, limit+1))
	if err != nil {
		return nil, BadRequest("Could not read the request body")
	}
	if int64(len(body)) > limit {
		return nil, PayloadTooLarge()
	}
	if strings.EqualFold(r.Header.Get("Content-Encoding"), "gzip") {
		reader, err := gzip.NewReader(bytes.NewReader(body))
		if err != nil {
			return nil, BadRequest("Invalid gzip body")
		}
		defer reader.Close()
		body, err = io.ReadAll(reader)
		if err != nil {
			return nil, BadRequest("Invalid gzip body")
		}
	}
	return body, nil
}

func ReadJSONObject(r *http.Request) (map[string]any, bool) {
	body, err := ReadBody(r, 0)
	if err != nil || len(bytes.TrimSpace(body)) == 0 {
		return nil, false
	}
	return DecodeObject(body)
}

func DecodeObject(body []byte) (map[string]any, bool) {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	var object map[string]any
	if err := decoder.Decode(&object); err != nil || object == nil {
		return nil, false
	}
	return object, true
}

func ReadJSONObjectOrEmpty(r *http.Request) map[string]any {
	object, found := ReadJSONObject(r)
	if !found {
		return map[string]any{}
	}
	return object
}
