package games

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"astroswarm/worker/internal/server/httpx"
	"astroswarm/worker/internal/server/pytime"
	"astroswarm/worker/internal/server/store"
	"astroswarm/worker/internal/server/validate"
)

const leaderboardColumns = "id, player_id, username, time_seconds, algorithm, created_at"

type leaderboardEntry struct {
	ID          string
	PlayerID    string
	Username    string
	TimeSeconds float64
	Algorithm   []byte
	CreatedAt   *time.Time
}

func (entry *leaderboardEntry) toMap() map[string]any {
	return map[string]any{
		"id": entry.ID, "username": entry.Username, "time_seconds": entry.TimeSeconds,
		"algorithm": listOrEmpty(entry.Algorithm), "created_at": pytime.IsoformatOrNil(entry.CreatedAt),
	}
}

type rowScanner interface {
	Scan(destinations ...any) error
}

func scanLeaderboardEntry(row rowScanner) (*leaderboardEntry, error) {
	entry := &leaderboardEntry{}
	err := row.Scan(&entry.ID, &entry.PlayerID, &entry.Username, &entry.TimeSeconds, &entry.Algorithm, &entry.CreatedAt)
	if store.IsNoRows(err) {
		return nil, nil
	}
	return entry, err
}

func (api *API) findLeaderboardEntry(r *http.Request, column string, value string) (*leaderboardEntry, error) {
	return scanLeaderboardEntry(api.deps.Store.Pool.QueryRow(r.Context(),
		"SELECT "+leaderboardColumns+" FROM leaderboard_entries WHERE "+column+" = $1", value))
}

func (api *API) requireLeaderboardEntry(r *http.Request) (*leaderboardEntry, error) {
	entry, err := api.findLeaderboardEntry(r, "id", r.PathValue("entry_id"))
	if err != nil {
		return nil, err
	}
	if entry == nil {
		return nil, httpx.BadRequest("Entry not found")
	}
	return entry, nil
}

func (api *API) listLeaderboard(w http.ResponseWriter, r *http.Request) error {
	rows, err := api.deps.Store.Pool.Query(r.Context(),
		"SELECT "+leaderboardColumns+" FROM leaderboard_entries ORDER BY time_seconds ASC LIMIT 100")
	if err != nil {
		return err
	}
	defer rows.Close()
	output := []any{}
	for rows.Next() {
		entry, err := scanLeaderboardEntry(rows)
		if err != nil {
			return err
		}
		output = append(output, entry.toMap())
	}
	if err := rows.Err(); err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, output)
	return nil
}

func (api *API) submitLeaderboard(w http.ResponseWriter, r *http.Request) error {
	keys := r.Header.Values("X-API-Key")
	if len(keys) == 0 || keys[0] != api.deps.Config.APISecretKey {
		return httpx.Unauthorized("Invalid API key")
	}
	data, found := httpx.ReadJSONObject(r)
	if !found || len(data) == 0 {
		return httpx.BadRequest("Invalid JSON data")
	}
	submission, err := validate.ParseLeaderboardSubmission(data)
	if err != nil {
		return httpx.BadRequest(err.Error())
	}
	algorithm, err := store.JSONParameter(submission.Algorithm)
	if err != nil {
		return err
	}
	ctx := r.Context()
	entry, err := api.findLeaderboardEntry(r, "player_id", submission.PlayerID)
	if err != nil {
		return err
	}
	if entry != nil {
		if submission.TimeSeconds < entry.TimeSeconds {
			_, err := api.deps.Store.Pool.Exec(ctx, `UPDATE leaderboard_entries SET time_seconds = $2, username = $3,
				algorithm = $4::json WHERE id = $1`, entry.ID, submission.TimeSeconds, submission.Username, algorithm)
			if err != nil {
				return err
			}
			if entry, err = api.findLeaderboardEntry(r, "id", entry.ID); err != nil {
				return err
			}
		}
		httpx.WriteJSON(w, http.StatusOK, entry.toMap())
		return nil
	}
	entryID := uuid.NewString()
	_, err = api.deps.Store.Pool.Exec(ctx, "INSERT INTO leaderboard_entries ("+leaderboardColumns+") VALUES ($1, $2, $3, $4, $5::json, $6)",
		entryID, submission.PlayerID, submission.Username, submission.TimeSeconds, algorithm, pytime.Now())
	if err != nil {
		return err
	}
	if entry, err = api.findLeaderboardEntry(r, "id", entryID); err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusCreated, entry.toMap())
	return nil
}

func (api *API) getLeaderboardEntry(w http.ResponseWriter, r *http.Request) error {
	entry, err := api.requireLeaderboardEntry(r)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, entry.toMap())
	return nil
}

func (api *API) deleteLeaderboardEntry(w http.ResponseWriter, r *http.Request) error {
	if _, err := api.deps.Auth.RequireAdmin(r); err != nil {
		return err
	}
	entry, err := api.requireLeaderboardEntry(r)
	if err != nil {
		return err
	}
	if _, err := api.deps.Store.Pool.Exec(r.Context(), "DELETE FROM leaderboard_entries WHERE id = $1", entry.ID); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

var unsafeFilenameCharacters = regexp.MustCompile(`[^A-Za-z0-9_-]+`)

func safeFilename(value string) string {
	cleaned := strings.Trim(unsafeFilenameCharacters.ReplaceAllString(value, "_"), "_")
	if cleaned == "" {
		return "entry"
	}
	return cleaned
}

func (api *API) exportLeaderboardEntry(w http.ResponseWriter, r *http.Request) error {
	entry, err := api.requireLeaderboardEntry(r)
	if err != nil {
		return err
	}
	content, err := json.MarshalIndent(entry.toMap(), "", "  ")
	if err != nil {
		return err
	}
	var buffer bytes.Buffer
	archive := zip.NewWriter(&buffer)
	file, err := archive.Create("entry.json")
	if err != nil {
		return err
	}
	if _, err := file.Write(content); err != nil {
		return err
	}
	if err := archive.Close(); err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename=leaderboard_"+safeFilename(entry.Username)+"_"+entry.ID+".zip")
	w.Header().Set("Content-Length", strconv.Itoa(buffer.Len()))
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(buffer.Bytes())
	return err
}
