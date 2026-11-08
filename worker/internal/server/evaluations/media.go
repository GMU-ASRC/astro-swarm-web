package evaluations

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"astroswarm/worker/internal/server/benchmarks"
	"astroswarm/worker/internal/server/entries"
	"astroswarm/worker/internal/server/httpx"
)

func pathNumber(r *http.Request, name string) (int64, error) {
	text := r.PathValue(name)
	if text == "" || strings.Trim(text, "0123456789") != "" {
		return 0, httpx.NotFound("")
	}
	number, err := strconv.ParseInt(text, 10, 64)
	if err != nil {
		return 0, httpx.NotFound("")
	}
	return number, nil
}

func owner(entry *entries.Entry) benchmarks.Owner {
	return benchmarks.Owner{BenchmarkID: entry.BenchmarkID, GenomeColors: entry.GenomeColors()}
}

func replayOwner(r *http.Request, entry *entries.Entry) benchmarks.Owner {
	current := owner(entry)
	current.FramesLink = r.URL.Query().Get("frames") == "binary"
	return current
}

func (api *API) replayFrames(w http.ResponseWriter, r *http.Request) error {
	runID, err := pathNumber(r, "run_id")
	if err != nil {
		return err
	}
	data, err := api.deps.Replays().RunFrames(r.Context(), runID)
	if err != nil {
		return err
	}
	if data == nil {
		return httpx.NotFound("Replay not found")
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(data)
	return err
}

func writeReplay(w http.ResponseWriter, replay map[string]any, err error) error {
	if err != nil {
		return err
	}
	if replay == nil {
		return httpx.BadRequest("Replay not found")
	}
	httpx.WriteJSON(w, http.StatusOK, replay)
	return nil
}

func (api *API) replays(w http.ResponseWriter, r *http.Request) error {
	entry, err := api.requireEntry(r)
	if err != nil {
		return err
	}
	index, err := api.deps.Replays().ReplayIndex(r.Context(), owner(entry))
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, index)
	return nil
}

func (api *API) replay(w http.ResponseWriter, r *http.Request) error {
	trial, err := pathNumber(r, "trial")
	if err != nil {
		return err
	}
	entry, err := api.requireEntry(r)
	if err != nil {
		return err
	}
	replay, err := api.deps.Replays().ReplayFor(r.Context(), replayOwner(r, entry), trial)
	return writeReplay(w, replay, err)
}

func (api *API) sweepReplays(w http.ResponseWriter, r *http.Request) error {
	entry, err := api.requireEntry(r)
	if err != nil {
		return err
	}
	index, err := api.deps.Replays().SweepIndex(r.Context(), owner(entry))
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, index)
	return nil
}

func (api *API) sweepReplay(w http.ResponseWriter, r *http.Request) error {
	n, err := pathNumber(r, "n")
	if err != nil {
		return err
	}
	entry, err := api.requireEntry(r)
	if err != nil {
		return err
	}
	replay, err := api.deps.Replays().SweepReplayFor(r.Context(), replayOwner(r, entry), n)
	return writeReplay(w, replay, err)
}

func (api *API) sweepTrialReplays(w http.ResponseWriter, r *http.Request) error {
	n, err := pathNumber(r, "n")
	if err != nil {
		return err
	}
	entry, err := api.requireEntry(r)
	if err != nil {
		return err
	}
	index, err := api.deps.Replays().SweepTrialIndex(r.Context(), owner(entry), n)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, index)
	return nil
}

func (api *API) sweepTrialReplay(w http.ResponseWriter, r *http.Request) error {
	n, err := pathNumber(r, "n")
	if err != nil {
		return err
	}
	trial, err := pathNumber(r, "trial")
	if err != nil {
		return err
	}
	entry, err := api.requireEntry(r)
	if err != nil {
		return err
	}
	replay, err := api.deps.Replays().SweepTrialReplayFor(r.Context(), replayOwner(r, entry), n, trial)
	return writeReplay(w, replay, err)
}

var unsafeFilenameCharacters = regexp.MustCompile(`[^A-Za-z0-9_-]+`)

func safeFilename(value string) string {
	cleaned := strings.Trim(unsafeFilenameCharacters.ReplaceAllString(value, "_"), "_")
	if cleaned == "" {
		return "entry"
	}
	return cleaned
}

func indentedJSON(value any) ([]byte, error) {
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		return nil, err
	}
	return bytes.TrimRight(buffer.Bytes(), "\n"), nil
}

func addZipFile(archive *zip.Writer, name string, value any) error {
	content, err := indentedJSON(value)
	if err != nil {
		return err
	}
	file, err := archive.Create(name)
	if err != nil {
		return err
	}
	_, err = file.Write(content)
	return err
}

func (api *API) export(w http.ResponseWriter, r *http.Request) error {
	entry, err := api.requireEntry(r)
	if err != nil {
		return err
	}
	ctx := r.Context()
	summary, err := entries.Summary(ctx, api.deps.Store.Pool, entry)
	if err != nil {
		return err
	}
	reader := api.deps.Replays()
	index, err := reader.ReplayIndex(ctx, owner(entry))
	if err != nil {
		return err
	}
	var buffer bytes.Buffer
	archive := zip.NewWriter(&buffer)
	if err := addZipFile(archive, "evaluation.json", entry.ToMap(summary)); err != nil {
		return err
	}
	runs, _ := index["runs"].([]any)
	for _, run := range runs {
		runObject, _ := run.(map[string]any)
		trial, isNumber := entries.WholeNumber(runObject["trial"])
		if !isNumber {
			continue
		}
		replay, err := reader.ReplayFor(ctx, owner(entry), trial)
		if err != nil {
			return err
		}
		if replay != nil {
			if err := addZipFile(archive, fmt.Sprintf("runs/trial_%d.json", trial), replay); err != nil {
				return err
			}
		}
	}
	if err := archive.Close(); err != nil {
		return err
	}
	downloadName := fmt.Sprintf("evaluation_%s_%s.zip", safeFilename(entry.Username), entry.ID)
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename="+downloadName)
	w.Header().Set("Content-Length", strconv.Itoa(buffer.Len()))
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(buffer.Bytes())
	return err
}
