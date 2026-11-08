package games

import (
	"fmt"
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
	maxMatchListLimit = 200
	matchColumns      = `id, player_id, username, game_version, outcome, duration, players, apm_bucket_seconds, apm_times,
		apm_player1, apm_player2, evaders_spawned, wild_remaining, first_target, created_at`
)

type surviveMatch struct {
	ID, PlayerID, Username                     string
	GameVersion, Outcome                       *string
	Duration, APMBucketSeconds                 *int
	EvadersSpawned, WildRemaining, FirstTarget *int
	Players, APMTimes, APMPlayer1, APMPlayer2  []byte
	CreatedAt                                  *time.Time
}

func scanMatch(row rowScanner) (*surviveMatch, error) {
	match := &surviveMatch{}
	err := row.Scan(&match.ID, &match.PlayerID, &match.Username, &match.GameVersion, &match.Outcome, &match.Duration,
		&match.Players, &match.APMBucketSeconds, &match.APMTimes, &match.APMPlayer1, &match.APMPlayer2,
		&match.EvadersSpawned, &match.WildRemaining, &match.FirstTarget, &match.CreatedAt)
	if store.IsNoRows(err) {
		return nil, nil
	}
	return match, err
}

func (match *surviveMatch) players() []any {
	list, isList := decodeJSON(match.Players).([]any)
	if !isList {
		return []any{}
	}
	return list
}

func (match *surviveMatch) player(slot int64) map[string]any {
	for _, item := range match.players() {
		entry, isObject := item.(map[string]any)
		if !isObject {
			continue
		}
		slotValue, found := entry["slot"]
		if !found {
			slotValue = int64(0)
		}
		if number, err := values.ToInt(slotValue); err == nil && number == slot {
			return entry
		}
	}
	return map[string]any{}
}

func (match *surviveMatch) playerName(slot int64) string {
	if name := match.player(slot)["name"]; values.Truthy(name) {
		return values.ToText(name)
	}
	return fmt.Sprintf("Player %d", slot)
}

func (match *surviveMatch) winnerName() any {
	switch stringOr(match.Outcome, "") {
	case "player1":
		return match.playerName(1)
	case "player2":
		return match.playerName(2)
	}
	return nil
}

func playerValue(player map[string]any, key string) any {
	if value, found := player[key]; found {
		return value
	}
	return 0
}

func (match *surviveMatch) toListMap() map[string]any {
	first, second := match.player(1), match.player(2)
	return map[string]any{
		"id": match.ID, "username": match.Username, "game_version": stringOr(match.GameVersion, "v0.0.5"),
		"outcome": stringOr(match.Outcome, "tie"), "winner": match.winnerName(), "duration": intOr(match.Duration, 0),
		"player1": match.playerName(1), "player2": match.playerName(2),
		"evaders_player1": playerValue(first, "evaders_through"), "evaders_player2": playerValue(second, "evaders_through"),
		"apm_player1": playerValue(first, "apm_average"), "apm_player2": playerValue(second, "apm_average"),
		"created_at": pytime.IsoformatOrNil(match.CreatedAt),
	}
}

func (match *surviveMatch) toMap() map[string]any {
	output := match.toListMap()
	output["player_id"] = match.PlayerID
	output["players"] = match.players()
	output["apm_bucket_seconds"] = intOr(match.APMBucketSeconds, 5)
	output["apm_times"] = listOrEmpty(match.APMTimes)
	output["apm_series_player1"] = listOrEmpty(match.APMPlayer1)
	output["apm_series_player2"] = listOrEmpty(match.APMPlayer2)
	output["evaders_spawned"] = intOr(match.EvadersSpawned, 0)
	output["wild_remaining"] = intOr(match.WildRemaining, 0)
	output["first_target"] = intOr(match.FirstTarget, 1)
	return output
}

func (api *API) findMatch(r *http.Request, matchID string) (*surviveMatch, error) {
	return scanMatch(api.deps.Store.Pool.QueryRow(r.Context(), "SELECT "+matchColumns+" FROM survive_matches WHERE id = $1", matchID))
}

func (api *API) requireMatch(r *http.Request) (*surviveMatch, error) {
	match, err := api.findMatch(r, r.PathValue("match_id"))
	if err != nil {
		return nil, err
	}
	if match == nil {
		return nil, httpx.NotFound("Match not found")
	}
	return match, nil
}

func (api *API) listMatches(w http.ResponseWriter, r *http.Request) error {
	limit := limitQuery(r, 100, maxMatchListLimit)
	conditions, arguments := []string{}, []any{}
	if playerID := r.URL.Query().Get("player_id"); playerID != "" {
		arguments = append(arguments, playerID)
		conditions = append(conditions, "WHERE player_id = $1")
	}
	arguments = append(arguments, limit)
	query := fmt.Sprintf("SELECT %s FROM survive_matches %s ORDER BY created_at DESC LIMIT $%d",
		matchColumns, strings.Join(conditions, " "), len(arguments))
	rows, err := api.deps.Store.Pool.Query(r.Context(), query, arguments...)
	if err != nil {
		return err
	}
	defer rows.Close()
	output := []any{}
	for rows.Next() {
		match, err := scanMatch(rows)
		if err != nil {
			return err
		}
		output = append(output, match.toListMap())
	}
	if err := rows.Err(); err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, output)
	return nil
}

func (api *API) getMatch(w http.ResponseWriter, r *http.Request) error {
	match, err := api.requireMatch(r)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, match.toMap())
	return nil
}

func (api *API) submitMatch(w http.ResponseWriter, r *http.Request) error {
	if _, err := api.deps.Auth.RequireAdmin(r); err != nil {
		return err
	}
	data, found := httpx.ReadJSONObject(r)
	if !found || len(data) == 0 {
		return httpx.BadRequest("Invalid JSON data")
	}
	submission, err := validate.ParseSurviveSubmission(data)
	if err != nil {
		return httpx.BadRequest(err.Error())
	}
	jsonValues := []any{}
	for _, value := range []any{submission.Players, submission.APMTimes, submission.APMPlayer1, submission.APMPlayer2} {
		encoded, err := store.JSONParameter(value)
		if err != nil {
			return err
		}
		jsonValues = append(jsonValues, encoded)
	}
	matchID := uuid.NewString()
	_, err = api.deps.Store.Pool.Exec(r.Context(), "INSERT INTO survive_matches ("+matchColumns+`) VALUES ($1, $2, $3, $4, $5, $6,
		$7::json, $8, $9::json, $10::json, $11::json, $12, $13, $14, $15)`,
		matchID, submission.PlayerID, submission.Username, submission.GameVersion, submission.Outcome, submission.Duration,
		jsonValues[0], submission.APMBucketSeconds, jsonValues[1], jsonValues[2], jsonValues[3],
		submission.EvadersSpawned, submission.WildRemaining, submission.FirstTarget, submission.CreatedAt)
	if err != nil {
		return err
	}
	match, err := api.findMatch(r, matchID)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusCreated, match.toMap())
	return nil
}

func (api *API) deleteMatch(w http.ResponseWriter, r *http.Request) error {
	if _, err := api.deps.Auth.RequireAdmin(r); err != nil {
		return err
	}
	match, err := api.requireMatch(r)
	if err != nil {
		return err
	}
	if _, err := api.deps.Store.Pool.Exec(r.Context(), "DELETE FROM survive_matches WHERE id = $1", match.ID); err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"deleted": match.ID})
	return nil
}
