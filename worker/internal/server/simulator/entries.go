package simulator

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"astroswarm/worker/internal/server/httpx"
	"astroswarm/worker/internal/server/pytime"
	"astroswarm/worker/internal/server/store"
	"astroswarm/worker/internal/server/validate"
	"astroswarm/worker/internal/server/values"
)

const (
	maxEntryListLimit = 200
	entryColumns      = `id, player_id, username, title, description, game_version, arena_width, arena_height, species, behaviors,
		arena_program, variables, obstacles, spawn_zones, placement_count, peak_robot_count, frame_count, record_interval, created_at`
)

type simulatorEntry struct {
	ID, PlayerID, Username, Title                                  string
	Description, GameVersion                                       *string
	ArenaWidth, ArenaHeight, RecordInterval                        *float64
	Species, Behaviors, ArenaProgram, Variables, Obstacles, Spawns []byte
	PlacementCount, PeakRobotCount, FrameCount                     *int
	CreatedAt                                                      *time.Time
}

func scanEntry(row rowScanner) (*simulatorEntry, error) {
	entry := &simulatorEntry{}
	err := row.Scan(&entry.ID, &entry.PlayerID, &entry.Username, &entry.Title, &entry.Description, &entry.GameVersion,
		&entry.ArenaWidth, &entry.ArenaHeight, &entry.Species, &entry.Behaviors, &entry.ArenaProgram, &entry.Variables,
		&entry.Obstacles, &entry.Spawns, &entry.PlacementCount, &entry.PeakRobotCount, &entry.FrameCount,
		&entry.RecordInterval, &entry.CreatedAt)
	if store.IsNoRows(err) {
		return nil, nil
	}
	return entry, err
}

func textOr(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func countOr(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}

func (entry *simulatorEntry) recordInterval() float64 {
	if entry.RecordInterval == nil || *entry.RecordInterval == 0 {
		return 0.1
	}
	return *entry.RecordInterval
}

func (entry *simulatorEntry) speciesSummary() []any {
	list, _ := decodeJSON(entry.Species).([]any)
	summary := []any{}
	for _, item := range list {
		species, _ := item.(map[string]any)
		summary = append(summary, map[string]any{"id": species["id"], "name": species["name"], "color": species["color"]})
	}
	return summary
}

func (entry *simulatorEntry) toListMap() map[string]any {
	spawnZones, _ := decodeJSON(entry.Spawns).([]any)
	return map[string]any{
		"id": entry.ID, "player_id": entry.PlayerID, "username": entry.Username, "title": entry.Title,
		"description": textOr(entry.Description), "game_version": textOr(entry.GameVersion), "species": entry.speciesSummary(),
		"placement_count": countOr(entry.PlacementCount), "peak_robot_count": countOr(entry.PeakRobotCount),
		"spawn_zone_count": len(spawnZones),
		"duration_seconds": values.RoundTo(float64(countOr(entry.FrameCount))*entry.recordInterval(), 1),
		"created_at": pytime.IsoformatOrNil(entry.CreatedAt),
	}
}

func (entry *simulatorEntry) toMap() map[string]any {
	output := entry.toListMap()
	output["arena_width"] = entry.ArenaWidth
	output["arena_height"] = entry.ArenaHeight
	output["species"] = jsonOr(entry.Species, []any{})
	output["behaviors"] = jsonOr(entry.Behaviors, map[string]any{})
	output["arena_program"] = jsonOr(entry.ArenaProgram, []any{})
	output["variables"] = jsonOr(entry.Variables, []any{})
	output["obstacles"] = jsonOr(entry.Obstacles, []any{})
	output["spawn_zones"] = jsonOr(entry.Spawns, []any{})
	output["frame_count"] = countOr(entry.FrameCount)
	output["record_interval"] = entry.recordInterval()
	return output
}

func (api *API) findEntry(r *http.Request, entryID string) (*simulatorEntry, error) {
	return scanEntry(api.deps.Store.Pool.QueryRow(r.Context(), "SELECT "+entryColumns+" FROM simulator_entries WHERE id = $1", entryID))
}

func (api *API) requireEntry(r *http.Request) (*simulatorEntry, error) {
	entry, err := api.findEntry(r, r.PathValue("entry_id"))
	if err != nil {
		return nil, err
	}
	if entry == nil {
		return nil, httpx.NotFound("Simulator entry not found")
	}
	return entry, nil
}

func (api *API) listEntries(w http.ResponseWriter, r *http.Request) error {
	limit := min(maxEntryListLimit, max(1, queryInt(r, "limit", 100)))
	conditions, arguments := []string{}, []any{}
	if playerID := r.URL.Query().Get("player_id"); playerID != "" {
		arguments = append(arguments, playerID)
		conditions = append(conditions, "WHERE player_id = $1")
	}
	arguments = append(arguments, limit)
	query := fmt.Sprintf("SELECT %s FROM simulator_entries %s ORDER BY created_at DESC LIMIT $%d",
		entryColumns, strings.Join(conditions, " "), len(arguments))
	rows, err := api.deps.Store.Pool.Query(r.Context(), query, arguments...)
	if err != nil {
		return err
	}
	defer rows.Close()
	output := []any{}
	for rows.Next() {
		entry, err := scanEntry(rows)
		if err != nil {
			return err
		}
		output = append(output, entry.toListMap())
	}
	if err := rows.Err(); err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, output)
	return nil
}

func (api *API) getEntry(w http.ResponseWriter, r *http.Request) error {
	entry, err := api.requireEntry(r)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, entry.toMap())
	return nil
}

func unpackReplay(packed *string) any {
	empty := map[string]any{"interval": 0.1, "robots": []any{}, "frames": []any{}}
	if packed == nil || *packed == "" {
		return empty
	}
	compressed, err := base64.StdEncoding.DecodeString(*packed)
	if err != nil {
		return empty
	}
	reader, err := zlib.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return empty
	}
	defer reader.Close()
	raw, err := io.ReadAll(reader)
	if err != nil {
		return empty
	}
	return decodeJSON(raw)
}

func packReplay(replay validate.SimulatorReplay) (string, error) {
	encoded, err := json.Marshal(replay)
	if err != nil {
		return "", err
	}
	var buffer bytes.Buffer
	writer, err := zlib.NewWriterLevel(&buffer, 6)
	if err != nil {
		return "", err
	}
	if _, err := writer.Write(encoded); err != nil {
		return "", err
	}
	if err := writer.Close(); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(buffer.Bytes()), nil
}

func (api *API) getEntryReplay(w http.ResponseWriter, r *http.Request) error {
	entry, err := api.requireEntry(r)
	if err != nil {
		return err
	}
	var packed *string
	err = api.deps.Store.Pool.QueryRow(r.Context(), "SELECT replay_packed FROM simulator_entries WHERE id = $1", entry.ID).Scan(&packed)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, unpackReplay(packed))
	return nil
}

func (api *API) submitEntry(w http.ResponseWriter, r *http.Request) error {
	if _, err := api.deps.Auth.RequireAdmin(r); err != nil {
		return err
	}
	data, found := httpx.ReadJSONObject(r)
	if !found {
		return httpx.BadRequest("Invalid JSON data")
	}
	submission, err := validate.ParseSimulatorSubmission(data)
	if err != nil {
		return httpx.BadRequest(err.Error())
	}
	packed, err := packReplay(submission.Replay)
	if err != nil {
		return err
	}
	jsonValues := []any{}
	for _, value := range []any{submission.Species, submission.Behaviors, submission.ArenaProgram, submission.Variables,
		submission.Obstacles, submission.SpawnZones} {
		encoded, err := store.JSONParameter(value)
		if err != nil {
			return err
		}
		jsonValues = append(jsonValues, encoded)
	}
	entryID := uuid.NewString()
	_, err = api.deps.Store.Pool.Exec(r.Context(), "INSERT INTO simulator_entries ("+entryColumns+`, replay_packed) VALUES
		($1, $2, $3, $4, $5, $6, $7, $8, $9::json, $10::json, $11::json, $12::json, $13::json, $14::json, $15, $16, $17, $18, $19, $20)`,
		entryID, submission.PlayerID, submission.Username, submission.Title, submission.Description, submission.GameVersion,
		submission.ArenaWidth, submission.ArenaHeight, jsonValues[0], jsonValues[1], jsonValues[2], jsonValues[3], jsonValues[4],
		jsonValues[5], submission.PlacementCount, submission.PeakRobotCount, submission.FrameCount, submission.RecordInterval,
		submission.CreatedAt, packed)
	if err != nil {
		return err
	}
	entry, err := api.findEntry(r, entryID)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusCreated, entry.toMap())
	return nil
}

func (api *API) deleteEntry(w http.ResponseWriter, r *http.Request) error {
	if _, err := api.deps.Auth.RequireAdmin(r); err != nil {
		return err
	}
	entry, err := api.requireEntry(r)
	if err != nil {
		return err
	}
	if _, err := api.deps.Store.Pool.Exec(r.Context(), "DELETE FROM simulator_entries WHERE id = $1", entry.ID); err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"deleted": entry.ID})
	return nil
}
