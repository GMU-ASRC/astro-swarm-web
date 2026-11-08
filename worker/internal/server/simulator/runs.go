package simulator

import (
	"net/http"
	"time"

	"astroswarm/worker/internal/server/httpx"
	"astroswarm/worker/internal/server/pytime"
	"astroswarm/worker/internal/server/store"
	"astroswarm/worker/internal/server/values"
)

const runColumns = `id, title, description, author, original_filename, stored_filename, thumbnail_filename, video_filename,
	file_size, species, robot_count, frame_count, arena_width, arena_height, download_count, created_at`

type simRun struct {
	ID, Title, OriginalFilename, StoredFilename string
	Description, Author                         *string
	ThumbnailFilename, VideoFilename            *string
	FileSize                                    int
	Species                                     []byte
	RobotCount, FrameCount, DownloadCount       *int
	ArenaWidth, ArenaHeight                     *float64
	CreatedAt                                   *time.Time
}

func scanRun(row rowScanner) (*simRun, error) {
	run := &simRun{}
	err := row.Scan(&run.ID, &run.Title, &run.Description, &run.Author, &run.OriginalFilename, &run.StoredFilename,
		&run.ThumbnailFilename, &run.VideoFilename, &run.FileSize, &run.Species, &run.RobotCount, &run.FrameCount,
		&run.ArenaWidth, &run.ArenaHeight, &run.DownloadCount, &run.CreatedAt)
	if store.IsNoRows(err) {
		return nil, nil
	}
	return run, err
}

func (run *simRun) baseMap() map[string]any {
	return map[string]any{
		"id": run.ID, "title": run.Title, "description": run.Description, "author": run.Author,
		"thumbnail_filename": run.ThumbnailFilename, "video_filename": run.VideoFilename, "file_size": run.FileSize,
		"species": jsonOr(run.Species, []any{}), "robot_count": run.RobotCount,
		"duration_seconds": values.RoundTo(float64(countOr(run.FrameCount))*0.1, 1),
		"download_count": run.DownloadCount, "created_at": pytime.IsoformatOrNil(run.CreatedAt),
	}
}

func (run *simRun) toMap() map[string]any {
	output := run.baseMap()
	output["original_filename"] = run.OriginalFilename
	output["frame_count"] = run.FrameCount
	output["arena_width"] = run.ArenaWidth
	output["arena_height"] = run.ArenaHeight
	return output
}

func (api *API) findRun(r *http.Request, runID string) (*simRun, error) {
	return scanRun(api.deps.Store.Pool.QueryRow(r.Context(), "SELECT "+runColumns+" FROM sim_runs WHERE id = $1", runID))
}

func (api *API) requireRun(r *http.Request) (*simRun, error) {
	run, err := api.findRun(r, r.PathValue("run_id"))
	if err != nil {
		return nil, err
	}
	if run == nil {
		return nil, httpx.NotFound("Run not found.")
	}
	return run, nil
}

func (api *API) listRuns(w http.ResponseWriter, r *http.Request) error {
	page := queryInt(r, "page", 1)
	pageSize := queryInt(r, "page_size", 20)
	queryPage, queryPageSize := max(1, page), pageSize
	if queryPageSize < 1 {
		queryPageSize = 20
	}
	ctx := r.Context()
	var total int
	if err := api.deps.Store.Pool.QueryRow(ctx, "SELECT count(*) FROM sim_runs").Scan(&total); err != nil {
		return err
	}
	rows, err := api.deps.Store.Pool.Query(ctx, "SELECT "+runColumns+" FROM sim_runs ORDER BY created_at DESC LIMIT $1 OFFSET $2",
		queryPageSize, (queryPage-1)*queryPageSize)
	if err != nil {
		return err
	}
	defer rows.Close()
	items := []any{}
	for rows.Next() {
		run, err := scanRun(rows)
		if err != nil {
			return err
		}
		items = append(items, run.baseMap())
	}
	if err := rows.Err(); err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items, "total": total, "page": page, "page_size": pageSize})
	return nil
}

func (api *API) getRun(w http.ResponseWriter, r *http.Request) error {
	run, err := api.requireRun(r)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, run.toMap())
	return nil
}
