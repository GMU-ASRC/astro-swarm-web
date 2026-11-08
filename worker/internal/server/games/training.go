package games

import (
	"net/http"
	"time"

	"github.com/google/uuid"

	"astroswarm/worker/internal/server/httpx"
	"astroswarm/worker/internal/server/pytime"
	"astroswarm/worker/internal/server/store"
	"astroswarm/worker/internal/server/validate"
	"astroswarm/worker/internal/server/values"
)

const (
	maxTrainingLimit     = 200
	defaultTrainingLimit = 100
	trainingColumns      = "id, player_id, username, game_version, circliness, defenders, algorithm, placements, updated_at"
)

type trainingScore struct {
	ID, PlayerID, Username string
	GameVersion            *string
	Circliness             float64
	Defenders              *int
	Algorithm, Placements  []byte
	UpdatedAt              *time.Time
}

func scanTrainingScore(row rowScanner) (*trainingScore, error) {
	score := &trainingScore{}
	err := row.Scan(&score.ID, &score.PlayerID, &score.Username, &score.GameVersion, &score.Circliness, &score.Defenders,
		&score.Algorithm, &score.Placements, &score.UpdatedAt)
	if store.IsNoRows(err) {
		return nil, nil
	}
	return score, err
}

func (score *trainingScore) toMap(rank any) map[string]any {
	return map[string]any{
		"id": score.ID, "rank": rank, "player_id": score.PlayerID, "username": score.Username,
		"circliness": values.RoundTo(score.Circliness, 4), "defenders": intOr(score.Defenders, 0),
		"game_version": score.GameVersion, "updated_at": pytime.IsoformatOrNil(score.UpdatedAt),
	}
}

func (score *trainingScore) toDetailMap() map[string]any {
	detail := score.toMap(nil)
	detail["algorithm"] = listOrEmpty(score.Algorithm)
	detail["placements"] = listOrEmpty(score.Placements)
	return detail
}

func (api *API) findTrainingScore(r *http.Request, playerID string) (*trainingScore, error) {
	return scanTrainingScore(api.deps.Store.Pool.QueryRow(r.Context(),
		"SELECT "+trainingColumns+" FROM training_scores WHERE player_id = $1", playerID))
}

func (api *API) trainingLeaderboard(w http.ResponseWriter, r *http.Request) error {
	limit := limitQuery(r, defaultTrainingLimit, maxTrainingLimit)
	rows, err := api.deps.Store.Pool.Query(r.Context(),
		"SELECT "+trainingColumns+" FROM training_scores ORDER BY circliness DESC, updated_at ASC LIMIT $1", limit)
	if err != nil {
		return err
	}
	defer rows.Close()
	output := []any{}
	for rows.Next() {
		score, err := scanTrainingScore(rows)
		if err != nil {
			return err
		}
		output = append(output, score.toMap(len(output)+1))
	}
	if err := rows.Err(); err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, output)
	return nil
}

func (api *API) getTrainingScore(w http.ResponseWriter, r *http.Request) error {
	score, err := api.findTrainingScore(r, r.PathValue("player_id"))
	if err != nil {
		return err
	}
	if score == nil {
		return httpx.NotFound("Training score not found")
	}
	httpx.WriteJSON(w, http.StatusOK, score.toDetailMap())
	return nil
}

func (api *API) submitTrainingScore(w http.ResponseWriter, r *http.Request) error {
	if _, err := api.deps.Auth.RequireAdmin(r); err != nil {
		return err
	}
	data, found := httpx.ReadJSONObject(r)
	if !found || len(data) == 0 {
		return httpx.BadRequest("Invalid JSON data")
	}
	submission, err := validate.ParseTrainingSubmission(data)
	if err != nil {
		return httpx.BadRequest(err.Error())
	}
	ctx := r.Context()
	pool := api.deps.Store.Pool
	score, err := api.findTrainingScore(r, submission.PlayerID)
	if err != nil {
		return err
	}
	status := http.StatusOK
	if score != nil && submission.Circliness <= score.Circliness {
		if _, err := pool.Exec(ctx, "UPDATE training_scores SET username = $2 WHERE id = $1", score.ID, submission.Username); err != nil {
			return err
		}
	} else {
		algorithm, err := store.JSONParameter(submission.Algorithm)
		if err != nil {
			return err
		}
		placements, err := store.JSONParameter(submission.Placements)
		if err != nil {
			return err
		}
		if score == nil {
			status = http.StatusCreated
			_, err = pool.Exec(ctx, "INSERT INTO training_scores ("+trainingColumns+") VALUES ($1, $2, $3, $4, $5, $6, $7::json, $8::json, $9)",
				uuid.NewString(), submission.PlayerID, submission.Username, submission.GameVersion, submission.Circliness,
				len(submission.Placements), algorithm, placements, submission.CreatedAt)
		} else {
			_, err = pool.Exec(ctx, `UPDATE training_scores SET username = $2, game_version = $3, circliness = $4, defenders = $5,
				algorithm = $6::json, placements = $7::json, updated_at = $8 WHERE id = $1`,
				score.ID, submission.Username, submission.GameVersion, submission.Circliness, len(submission.Placements),
				algorithm, placements, submission.CreatedAt)
		}
		if err != nil {
			return err
		}
	}
	saved, err := api.findTrainingScore(r, submission.PlayerID)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, status, saved.toMap(nil))
	return nil
}
