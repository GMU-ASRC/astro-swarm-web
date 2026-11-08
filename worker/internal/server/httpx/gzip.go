package httpx

import (
	"bytes"
	"compress/gzip"
	"net/http"
	"strings"
)

const (
	gzipMinimumBytes = 1024
	gzipLevel        = 5
)

var compressibleTypes = []string{
	"application/json",
	"text/html",
	"text/css",
	"text/javascript",
	"application/javascript",
	"image/svg+xml",
}

type gzipWriter struct {
	http.ResponseWriter
	status     int
	buffer     bytes.Buffer
	decided    bool
	compressor *gzip.Writer
}

func Gzip(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") || r.Method == http.MethodHead {
			next.ServeHTTP(w, r)
			return
		}
		writer := &gzipWriter{ResponseWriter: w}
		next.ServeHTTP(writer, r)
		writer.finish()
	})
}

func (writer *gzipWriter) WriteHeader(status int) {
	if writer.status == 0 {
		writer.status = status
	}
}

func (writer *gzipWriter) Write(data []byte) (int, error) {
	if writer.status == 0 {
		writer.status = http.StatusOK
	}
	if writer.decided {
		if writer.compressor != nil {
			return writer.compressor.Write(data)
		}
		return writer.ResponseWriter.Write(data)
	}
	writer.buffer.Write(data)
	if writer.buffer.Len() >= gzipMinimumBytes {
		writer.decide(true)
	}
	return len(data), nil
}

func (writer *gzipWriter) Flush() {
	if !writer.decided {
		writer.decide(writer.buffer.Len() >= gzipMinimumBytes)
	}
	if writer.compressor != nil {
		writer.compressor.Flush()
	}
	if flusher, canFlush := writer.ResponseWriter.(http.Flusher); canFlush {
		flusher.Flush()
	}
}

func (writer *gzipWriter) Unwrap() http.ResponseWriter {
	return writer.ResponseWriter
}

func (writer *gzipWriter) decide(largeEnough bool) {
	writer.decided = true
	header := writer.ResponseWriter.Header()
	if largeEnough && writer.compressible(header) {
		header.Set("Content-Encoding", "gzip")
		header.Add("Vary", "Accept-Encoding")
		header.Del("Content-Length")
		writer.compressor, _ = gzip.NewWriterLevel(writer.ResponseWriter, gzipLevel)
	}
	status := writer.status
	if status == 0 {
		status = http.StatusOK
	}
	writer.ResponseWriter.WriteHeader(status)
	if writer.buffer.Len() == 0 {
		return
	}
	if writer.compressor != nil {
		writer.compressor.Write(writer.buffer.Bytes())
	} else {
		writer.ResponseWriter.Write(writer.buffer.Bytes())
	}
	writer.buffer.Reset()
}

func (writer *gzipWriter) compressible(header http.Header) bool {
	if header.Get("Content-Encoding") != "" || writer.status < 200 || writer.status == http.StatusNoContent || writer.status == http.StatusNotModified || writer.status == http.StatusPartialContent {
		return false
	}
	contentType := strings.ToLower(header.Get("Content-Type"))
	for _, compressibleType := range compressibleTypes {
		if strings.HasPrefix(contentType, compressibleType) {
			return true
		}
	}
	return false
}

func (writer *gzipWriter) finish() {
	if !writer.decided {
		if writer.status == 0 && writer.buffer.Len() == 0 {
			return
		}
		writer.decide(writer.buffer.Len() >= gzipMinimumBytes)
	}
	if writer.compressor != nil {
		writer.compressor.Close()
	}
}
