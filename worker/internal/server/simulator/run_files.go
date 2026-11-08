package simulator

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"astroswarm/worker/internal/server/httpx"
	"astroswarm/worker/internal/server/pytime"
	"astroswarm/worker/internal/server/store"
)

const (
	runsFolder      = "runs"
	multipartMemory = 32 << 20
	maxTitleLength  = 80
	maxDescription  = 400
	maxAuthorLength = 60
)

var unsafeFilenameCharacters = regexp.MustCompile(`[^A-Za-z0-9_-]+`)

func safeFilename(value string) string {
	cleaned := strings.Trim(unsafeFilenameCharacters.ReplaceAllString(value, "_"), "_")
	if cleaned == "" {
		return "run"
	}
	return cleaned
}

func plainError(w http.ResponseWriter, status int, message string) error {
	httpx.WriteJSON(w, status, map[string]any{"error": message})
	return nil
}

func (api *API) runPath(storedFilename string) string {
	return filepath.Join(api.deps.Config.UploadDir, runsFolder, storedFilename)
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func firstRunes(text string, limit int) string {
	runes := []rune(text)
	if len(runes) > limit {
		return string(runes[:limit])
	}
	return text
}

type uploadedFile struct {
	name    string
	content []byte
}

func readUpload(r *http.Request, field string) (uploadedFile, error) {
	file, header, err := r.FormFile(field)
	if err != nil {
		return uploadedFile{}, err
	}
	defer file.Close()
	content, err := io.ReadAll(file)
	return uploadedFile{name: header.Filename, content: content}, err
}

func nameOr(name string, fallback string) string {
	if name == "" {
		return fallback
	}
	return name
}

func writeRunArchive(path string, files []uploadedFile, fallbacks []string) (int64, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return 0, err
	}
	output, err := os.Create(path)
	if err != nil {
		return 0, err
	}
	archive := zip.NewWriter(output)
	for index, file := range files {
		entry, err := archive.Create(nameOr(file.name, fallbacks[index]))
		if err == nil {
			_, err = entry.Write(file.content)
		}
		if err != nil {
			output.Close()
			return 0, err
		}
	}
	if err := archive.Close(); err != nil {
		output.Close()
		return 0, err
	}
	if err := output.Close(); err != nil {
		return 0, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

func (api *API) uploadRun(w http.ResponseWriter, r *http.Request) error {
	limit := api.deps.Config.MaxUploadBytes
	r.Body = http.MaxBytesReader(w, r.Body, limit+multipartMemory)
	if err := r.ParseMultipartForm(multipartMemory); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			return plainError(w, http.StatusRequestEntityTooLarge, "Total file size exceeds maximum size limit.")
		}
		return plainError(w, http.StatusBadRequest, "Missing run_file, config_file, or video_file.")
	}
	runFile, runErr := readUpload(r, "run_file")
	configFile, configErr := readUpload(r, "config_file")
	videoFile, videoErr := readUpload(r, "video_file")
	if runErr != nil || configErr != nil || videoErr != nil {
		return plainError(w, http.StatusBadRequest, "Missing run_file, config_file, or video_file.")
	}
	title := strings.TrimSpace(r.FormValue("title"))
	description := strings.TrimSpace(r.FormValue("description"))
	author := "anonymous"
	if values, present := r.MultipartForm.Value["author"]; present && len(values) > 0 {
		author = strings.TrimSpace(values[0])
	}
	if author == "" {
		author = "anonymous"
	}
	if title == "" {
		return plainError(w, http.StatusBadRequest, "Title is required.")
	}
	if runFile.name == "" || strings.ToLower(filepath.Ext(runFile.name)) != ".run" {
		return plainError(w, http.StatusBadRequest, "run_file must be a .run file.")
	}
	if int64(len(runFile.content)+len(configFile.content)+len(videoFile.content)) > limit {
		return plainError(w, http.StatusRequestEntityTooLarge, "Total file size exceeds maximum size limit.")
	}
	runID := uuid.NewString()
	storedFilename, videoFilename := runID+".zip", runID+".mp4"
	fileSize, err := writeRunArchive(api.runPath(storedFilename), []uploadedFile{runFile, configFile, videoFile},
		[]string{"run.run", "config.json", "video.mp4"})
	if err != nil {
		return err
	}
	if err := os.WriteFile(api.runPath(videoFilename), videoFile.content, 0o644); err != nil {
		return err
	}
	var thumbnailFilename *string
	if name := runID + ".jpg"; api.extractThumbnail(r.Context(), api.runPath(videoFilename), api.runPath(name)) {
		thumbnailFilename = &name
	}
	metadata, err := extractRunMetadata(runFile.content)
	if err != nil {
		metadata = defaultRunMetadata()
	}
	species, err := store.JSONParameter(metadata.species)
	if err != nil {
		return err
	}
	_, err = api.deps.Store.Pool.Exec(r.Context(), "INSERT INTO sim_runs ("+runColumns+`) VALUES ($1, $2, $3, $4, $5, $6, $7, $8,
		$9, $10::json, $11, $12, $13, $14, 0, $15)`,
		runID, firstRunes(title, maxTitleLength), firstRunes(description, maxDescription), firstRunes(author, maxAuthorLength),
		title+".zip", storedFilename, thumbnailFilename, videoFilename, fileSize, species, metadata.robotCount,
		metadata.frameCount, metadata.arenaWidth, metadata.arenaHeight, pytime.Now())
	if err != nil {
		return err
	}
	run, err := api.findRun(r, runID)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusCreated, run.toMap())
	return nil
}

func readArchiveEntry(path string, matches func(name string) bool) ([]byte, bool, error) {
	archive, err := zip.OpenReader(path)
	if err != nil {
		return nil, false, err
	}
	defer archive.Close()
	for _, file := range archive.File {
		if !matches(file.Name) {
			continue
		}
		reader, err := file.Open()
		if err != nil {
			return nil, false, err
		}
		defer reader.Close()
		content, err := io.ReadAll(reader)
		return content, true, err
	}
	return nil, false, nil
}

func isConfigName(name string) bool {
	return strings.HasSuffix(name, ".cfg") || strings.HasSuffix(name, ".json")
}

func isVideoName(name string) bool {
	return strings.HasSuffix(name, ".mp4") || strings.HasSuffix(name, ".webm")
}

func (api *API) exportRun(w http.ResponseWriter, r *http.Request) error {
	run, err := api.requireRun(r)
	if err != nil {
		return err
	}
	var configData []byte
	if zipPath := api.runPath(run.StoredFilename); fileExists(zipPath) {
		if content, found, err := readArchiveEntry(zipPath, isConfigName); err == nil && found {
			configData = content
		}
	}
	runJSON, err := json.MarshalIndent(run.toMap(), "", "  ")
	if err != nil {
		return err
	}
	var buffer bytes.Buffer
	archive := zip.NewWriter(&buffer)
	entries := []uploadedFile{{name: "run.json", content: runJSON}}
	if configData != nil {
		entries = append(entries, uploadedFile{name: "config.json", content: configData})
	}
	for _, entry := range entries {
		writer, err := archive.Create(entry.name)
		if err != nil {
			return err
		}
		if _, err := writer.Write(entry.content); err != nil {
			return err
		}
	}
	if err := archive.Close(); err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename=run_"+safeFilename(run.Title)+"_"+run.ID+".zip")
	w.Header().Set("Content-Length", strconv.Itoa(buffer.Len()))
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(buffer.Bytes())
	return err
}

func serveFile(w http.ResponseWriter, r *http.Request, path string, contentType string, downloadName string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", contentType)
	if downloadName != "" {
		w.Header().Set("Content-Disposition", "attachment; filename="+strconv.Quote(downloadName))
	}
	http.ServeContent(w, r, "", info.ModTime(), file)
	return nil
}

func (api *API) downloadRun(w http.ResponseWriter, r *http.Request) error {
	run, err := api.requireRun(r)
	if err != nil {
		return err
	}
	path := api.runPath(run.StoredFilename)
	if !fileExists(path) {
		return plainError(w, http.StatusNotFound, "File not found on server.")
	}
	if _, err := api.deps.Store.Pool.Exec(r.Context(),
		"UPDATE sim_runs SET download_count = COALESCE(download_count, 0) + 1 WHERE id = $1", run.ID); err != nil {
		return err
	}
	return serveFile(w, r, path, "application/zip", run.OriginalFilename)
}

func (api *API) runThumbnail(w http.ResponseWriter, r *http.Request) error {
	run, err := api.requireRun(r)
	if err != nil {
		return err
	}
	if run.ThumbnailFilename == nil || *run.ThumbnailFilename == "" || !fileExists(api.runPath(*run.ThumbnailFilename)) {
		if err := api.createMissingThumbnail(r, run); err != nil {
			return err
		}
	}
	if run.ThumbnailFilename == nil || *run.ThumbnailFilename == "" {
		return plainError(w, http.StatusNotFound, "No thumbnail available.")
	}
	path := api.runPath(*run.ThumbnailFilename)
	if !fileExists(path) {
		return plainError(w, http.StatusNotFound, "Thumbnail file not found on server.")
	}
	return serveFile(w, r, path, "image/jpeg", "")
}

func (api *API) createMissingThumbnail(r *http.Request, run *simRun) error {
	if run.VideoFilename == nil || *run.VideoFilename == "" || !fileExists(api.runPath(*run.VideoFilename)) {
		return nil
	}
	name := run.ID + ".jpg"
	if !api.extractThumbnail(r.Context(), api.runPath(*run.VideoFilename), api.runPath(name)) {
		return nil
	}
	if _, err := api.deps.Store.Pool.Exec(r.Context(), "UPDATE sim_runs SET thumbnail_filename = $2 WHERE id = $1", run.ID, name); err != nil {
		return err
	}
	run.ThumbnailFilename = &name
	return nil
}

func (api *API) runVideo(w http.ResponseWriter, r *http.Request) error {
	run, err := api.requireRun(r)
	if err != nil {
		return err
	}
	if run.VideoFilename != nil && *run.VideoFilename != "" {
		if path := api.runPath(*run.VideoFilename); fileExists(path) {
			return serveFile(w, r, path, "video/mp4", "")
		}
	}
	zipPath := api.runPath(run.StoredFilename)
	if !fileExists(zipPath) {
		return plainError(w, http.StatusNotFound, "Video file not found on server.")
	}
	video, found, err := readArchiveEntry(zipPath, isVideoName)
	if err != nil {
		return err
	}
	if !found {
		return plainError(w, http.StatusNotFound, "No video found in run archive.")
	}
	w.Header().Set("Content-Type", "video/mp4")
	http.ServeContent(w, r, "", time.Time{}, bytes.NewReader(video))
	return nil
}

func (api *API) runConfig(w http.ResponseWriter, r *http.Request) error {
	run, err := api.requireRun(r)
	if err != nil {
		return err
	}
	zipPath := api.runPath(run.StoredFilename)
	if !fileExists(zipPath) {
		return plainError(w, http.StatusNotFound, "Run archive not found on server.")
	}
	content, found, err := readArchiveEntry(zipPath, isConfigName)
	if err != nil {
		return plainError(w, http.StatusInternalServerError, "Failed to parse config: "+err.Error())
	}
	if !found {
		return plainError(w, http.StatusNotFound, "No config file found in run archive.")
	}
	config := decodeJSON(content)
	if config == nil && strings.TrimSpace(string(content)) != "null" {
		return plainError(w, http.StatusInternalServerError, "Failed to parse config: invalid JSON")
	}
	httpx.WriteJSON(w, http.StatusOK, config)
	return nil
}

func (api *API) deleteRun(w http.ResponseWriter, r *http.Request) error {
	if _, err := api.deps.Auth.RequireAdmin(r); err != nil {
		return err
	}
	run, err := api.requireRun(r)
	if err != nil {
		return err
	}
	stored := []string{run.StoredFilename}
	for _, optional := range []*string{run.ThumbnailFilename, run.VideoFilename} {
		if optional != nil && *optional != "" {
			stored = append(stored, *optional)
		}
	}
	for _, name := range stored {
		if path := api.runPath(name); fileExists(path) {
			if err := os.Remove(path); err != nil {
				return err
			}
		}
	}
	if _, err := api.deps.Store.Pool.Exec(r.Context(), "DELETE FROM sim_runs WHERE id = $1", run.ID); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}
