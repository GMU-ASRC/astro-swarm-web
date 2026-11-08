package apicompare

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
)

type Result struct {
	Route       Route
	Differences differences
	FlaskTime   time.Duration
	GoTime      time.Duration
}

func (result Result) Same() bool {
	return result.Differences.total == 0
}

func Compare(client *Client, flaskBase string, goBase string, route Route) Result {
	flask := client.Get(flaskBase+route.Path, route.Admin)
	golang := client.Get(goBase+route.Path, route.Admin)
	result := Result{Route: route, FlaskTime: flask.Duration, GoTime: golang.Duration}
	found := &result.Differences
	if flask.Err != nil || golang.Err != nil {
		found.add("request", "flask error %v, go error %v", flask.Err, golang.Err)
		return result
	}
	if flask.Status != golang.Status {
		found.add("status", "flask %d, go %d", flask.Status, golang.Status)
	}
	if flask.ContentType != golang.ContentType {
		found.add("content type", "flask %q, go %q", flask.ContentType, golang.ContentType)
	}
	switch {
	case flask.ContentType == "application/json" && golang.ContentType == "application/json":
		compareJSONBodies("body", flask.Body, golang.Body, found)
	case route.Kind == KindZip && flask.Status == 200 && golang.Status == 200:
		compareZips(flask.Body, golang.Body, found)
	case route.Kind == KindHTML && flask.Status == 200 && golang.Status == 200:
		if normalizeHTML(flask.Body) != normalizeHTML(golang.Body) {
			found.add("html", "page text differs (flask %d bytes, go %d bytes)", len(flask.Body), len(golang.Body))
		}
	}
	return result
}

func compareJSONBodies(path string, flaskBody []byte, goBody []byte, found *differences) {
	flaskValue, flaskErr := decodeJSON(flaskBody)
	goValue, goErr := decodeJSON(goBody)
	if flaskErr != nil || goErr != nil {
		found.add(path, "invalid JSON (flask %v, go %v)", flaskErr, goErr)
		return
	}
	compareValues(path, flaskValue, goValue, found)
}

func normalizeHTML(body []byte) string {
	return strings.ReplaceAll(string(body), "&#x27;", "&#39;")
}

func zipFiles(body []byte) (map[string][]byte, error) {
	reader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return nil, err
	}
	files := map[string][]byte{}
	for _, file := range reader.File {
		opened, err := file.Open()
		if err != nil {
			return nil, err
		}
		content, err := io.ReadAll(opened)
		opened.Close()
		if err != nil {
			return nil, err
		}
		files[file.Name] = content
	}
	return files, nil
}

func compareZips(flaskBody []byte, goBody []byte, found *differences) {
	flaskFiles, flaskErr := zipFiles(flaskBody)
	goFiles, goErr := zipFiles(goBody)
	if flaskErr != nil || goErr != nil {
		found.add("zip", "unreadable (flask %v, go %v)", flaskErr, goErr)
		return
	}
	names := map[string]bool{}
	for name := range flaskFiles {
		names[name] = true
	}
	for name := range goFiles {
		names[name] = true
	}
	sorted := make([]string, 0, len(names))
	for name := range names {
		sorted = append(sorted, name)
	}
	sort.Strings(sorted)
	for _, name := range sorted {
		flaskContent, inFlask := flaskFiles[name]
		goContent, inGo := goFiles[name]
		switch {
		case !inGo:
			found.add("zip:"+name, "missing in go")
		case !inFlask:
			found.add("zip:"+name, "extra in go")
		case strings.HasSuffix(name, ".json"):
			compareJSONBodies("zip:"+name, flaskContent, goContent, found)
		case !bytes.Equal(flaskContent, goContent):
			found.add("zip:"+name, "contents differ")
		}
	}
}

func Report(results []Result, verbose bool) (string, bool) {
	var builder strings.Builder
	different, unstable := 0, 0
	var flaskTotal, goTotal time.Duration
	for _, result := range results {
		flaskTotal += result.FlaskTime
		goTotal += result.GoTime
		if result.Same() {
			if verbose {
				fmt.Fprintf(&builder, "SAME  %-70s flask %8s  go %8s\n", result.Route.Path, rounded(result.FlaskTime), rounded(result.GoTime))
			}
			continue
		}
		label := "DIFF"
		if result.Route.Unstable != "" {
			label = "NOTE"
			unstable++
		} else {
			different++
		}
		fmt.Fprintf(&builder, "%s  %s  (%d difference(s))\n", label, result.Route.Path, result.Differences.total)
		if result.Route.Unstable != "" {
			fmt.Fprintf(&builder, "      expected to vary: %s\n", result.Route.Unstable)
		}
		for _, line := range result.Differences.lines {
			fmt.Fprintf(&builder, "      %s\n", line)
		}
	}
	fmt.Fprintf(&builder, "\n%d route(s) compared: %d same, %d different, %d expected to vary\n",
		len(results), len(results)-different-unstable, different, unstable)
	fmt.Fprintf(&builder, "Total request time: flask %s, go %s\n", rounded(flaskTotal), rounded(goTotal))
	return builder.String(), different == 0
}

func rounded(duration time.Duration) time.Duration {
	return duration.Round(time.Millisecond)
}
