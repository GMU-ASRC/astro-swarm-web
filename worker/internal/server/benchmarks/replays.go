package benchmarks

import (
	"context"
	"fmt"

	"astroswarm/worker/internal/server/store"
)

type Run struct {
	ID           int64
	BinaryKey    *string
	Position     int
	Details      map[string]any
	FramesKey    *string
	FramesFormat *string
}

func (run Run) Recorded() bool {
	return run.FramesKey != nil || run.BinaryKey != nil
}

func (run Run) detail(key string, fallback any) any {
	if value, found := run.Details[key]; found {
		return value
	}
	return fallback
}

type Reader struct {
	db    store.Querier
	files *Files
}

func NewReader(db store.Querier, files *Files) *Reader {
	return &Reader{db: db, files: files}
}

type Owner struct {
	BenchmarkID  *string
	GenomeColors []any
	FramesLink   bool
}

func (reader *Reader) setFrames(replay map[string]any, owner Owner, run Run) {
	if owner.FramesLink && run.Recorded() {
		replay["frames_url"] = fmt.Sprintf("/api/replay-frames/%d", run.ID)
		return
	}
	replay["frames"] = reader.Frames(run)
}

func (reader *Reader) Meta(ctx context.Context, benchmarkID *string) (map[string]any, error) {
	meta := map[string]any{}
	if benchmarkID == nil {
		return meta, nil
	}
	var raw []byte
	err := reader.db.QueryRow(ctx, "SELECT replay_meta FROM benchmarks WHERE id = $1", *benchmarkID).Scan(&raw)
	if store.IsNoRows(err) || raw == nil {
		return meta, nil
	}
	if err != nil {
		return nil, err
	}
	var decoded any
	if err := decodeJSON(raw, &decoded); err != nil {
		return nil, err
	}
	if object, isObject := decoded.(map[string]any); isObject {
		return object, nil
	}
	return meta, nil
}

func (reader *Reader) runs(ctx context.Context, benchmarkID *string, kind string, condition string, arguments ...any) ([]Run, error) {
	if benchmarkID == nil {
		return nil, nil
	}
	query := "SELECT id, binary_key, position, details, frames_key, frames_format FROM benchmark_runs WHERE benchmark_id = $1 AND kind = $2"
	if condition != "" {
		query += " AND " + condition
	}
	query += " ORDER BY position ASC"
	rows, err := reader.db.Query(ctx, query, append([]any{*benchmarkID, kind}, arguments...)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	runs := []Run{}
	for rows.Next() {
		var run Run
		var raw []byte
		if err := rows.Scan(&run.ID, &run.BinaryKey, &run.Position, &raw, &run.FramesKey, &run.FramesFormat); err != nil {
			return nil, err
		}
		var details any
		if err := decodeJSON(raw, &details); err != nil {
			return nil, err
		}
		run.Details, _ = details.(map[string]any)
		if run.Details == nil {
			run.Details = map[string]any{}
		}
		runs = append(runs, run)
	}
	return runs, rows.Err()
}

func (reader *Reader) firstRun(ctx context.Context, benchmarkID *string, kind string, condition string, argument any) (*Run, error) {
	runs, err := reader.runs(ctx, benchmarkID, kind, condition, argument)
	if err != nil || len(runs) == 0 {
		return nil, err
	}
	return &runs[0], nil
}

func (reader *Reader) Frames(run Run) []any {
	if run.BinaryKey != nil {
		if data, found := reader.files.Read(*run.BinaryKey); found {
			if frames, err := DecodeBinaryFrames(data); err == nil {
				return frames
			}
		}
	}
	if run.FramesKey == nil {
		return []any{}
	}
	data, found := reader.files.Read(*run.FramesKey)
	if !found {
		return []any{}
	}
	framesFormat := FormatDelta
	if run.FramesFormat != nil {
		framesFormat = *run.FramesFormat
	}
	frames, err := DecodeFrames(data, framesFormat)
	if err != nil {
		return []any{}
	}
	return frames
}

func (reader *Reader) ReplayIndex(ctx context.Context, owner Owner) (map[string]any, error) {
	meta, err := reader.Meta(ctx, owner.BenchmarkID)
	if err != nil {
		return nil, err
	}
	runs, err := reader.runs(ctx, owner.BenchmarkID, RunKindTrial, "")
	if err != nil {
		return nil, err
	}
	listed := make([]any, 0, len(runs))
	for _, run := range runs {
		listed = append(listed, map[string]any{"trial": run.Details["trial"], "outcome": run.Details["outcome"], "recorded": run.Recorded()})
	}
	return map[string]any{
		"fps": valueOr(meta, "fps", 12), "defenders": valueOr(meta, "defenders", 0), "view": valueOr(meta, "view", 300),
		"fov": valueOr(meta, "fov", 70), "planet": meta["planet"], "arena": meta["arena"], "runs": listed,
	}, nil
}

func timeFields(target map[string]any, run Run) {
	target["outcome"] = run.Details["outcome"]
	target["detection_time"] = run.detail("detection_time", -1)
	target["capture_time"] = run.detail("capture_time", -1)
	target["goal_time"] = run.detail("goal_time", -1)
}

func (reader *Reader) ReplayFor(ctx context.Context, owner Owner, trial int64) (map[string]any, error) {
	run, err := reader.firstRun(ctx, owner.BenchmarkID, RunKindTrial, "trial = $3", trial)
	if err != nil || run == nil {
		return nil, err
	}
	meta, err := reader.Meta(ctx, owner.BenchmarkID)
	if err != nil {
		return nil, err
	}
	replay := map[string]any{
		"trial": run.Details["trial"], "fps": valueOr(meta, "fps", 12), "defenders": valueOr(meta, "defenders", 0),
		"view": valueOr(meta, "view", 300), "fov": valueOr(meta, "fov", 70), "speed": meta["speed"], "hull": meta["hull"],
		"planet": meta["planet"], "arena": meta["arena"], "stats": run.detail("stats", map[string]any{}),
		"genome_switches": run.detail("genome_switches", []any{}), "genome_colors": owner.GenomeColors,
	}
	reader.setFrames(replay, owner, *run)
	timeFields(replay, *run)
	return replay, nil
}

func (reader *Reader) SweepIndex(ctx context.Context, owner Owner) ([]any, error) {
	runs, err := reader.runs(ctx, owner.BenchmarkID, RunKindSweep, "")
	if err != nil {
		return nil, err
	}
	childCounts, err := reader.childCounts(ctx, owner.BenchmarkID)
	if err != nil {
		return nil, err
	}
	rows := make([]any, 0, len(runs))
	for _, run := range runs {
		row := map[string]any{
			"n": run.Details["n"], "detection_rate": run.Details["detection_rate"], "capture_rate": run.Details["capture_rate"],
			"trial_count": childCounts[run.Position], "recorded": run.Recorded(),
		}
		timeFields(row, run)
		rows = append(rows, row)
	}
	return rows, nil
}

func (reader *Reader) childCounts(ctx context.Context, benchmarkID *string) (map[int]int, error) {
	counts := map[int]int{}
	if benchmarkID == nil {
		return counts, nil
	}
	rows, err := reader.db.Query(ctx, `SELECT parent_position, count(*) FROM benchmark_runs
		WHERE benchmark_id = $1 AND kind = $2 GROUP BY parent_position`, *benchmarkID, RunKindSweepTrial)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var parent *int
		var count int
		if err := rows.Scan(&parent, &count); err != nil {
			return nil, err
		}
		if parent != nil {
			counts[*parent] = count
		}
	}
	return counts, rows.Err()
}

func (reader *Reader) sweepRun(ctx context.Context, owner Owner, n int64) (*Run, error) {
	return reader.firstRun(ctx, owner.BenchmarkID, RunKindSweep, "sweep_n = $3", n)
}

func (reader *Reader) sweepChildren(ctx context.Context, owner Owner, sweepRun *Run) ([]Run, error) {
	return reader.runs(ctx, owner.BenchmarkID, RunKindSweepTrial, "parent_position = $3", sweepRun.Position)
}

func (reader *Reader) SweepTrialIndex(ctx context.Context, owner Owner, n int64) ([]any, error) {
	sweepRun, err := reader.sweepRun(ctx, owner, n)
	if err != nil || sweepRun == nil {
		return []any{}, err
	}
	children, err := reader.sweepChildren(ctx, owner, sweepRun)
	if err != nil {
		return nil, err
	}
	rows := make([]any, 0, len(children))
	for _, child := range children {
		row := map[string]any{"trial": child.Details["trial"], "recorded": child.Recorded()}
		timeFields(row, child)
		rows = append(rows, row)
	}
	return rows, nil
}

func (reader *Reader) SweepTrialReplayFor(ctx context.Context, owner Owner, n int64, trial int64) (map[string]any, error) {
	sweepRun, err := reader.sweepRun(ctx, owner, n)
	if err != nil || sweepRun == nil {
		return nil, err
	}
	children, err := reader.sweepChildren(ctx, owner, sweepRun)
	if err != nil {
		return nil, err
	}
	var child *Run
	for index := range children {
		if whole := wholeNumber(children[index].Details["trial"]); whole != nil && *whole == trial {
			child = &children[index]
			break
		}
	}
	if child == nil {
		return nil, nil
	}
	meta, err := reader.Meta(ctx, owner.BenchmarkID)
	if err != nil {
		return nil, err
	}
	replay := map[string]any{
		"n": sweepRun.Details["n"], "trial": child.Details["trial"], "fps": valueOr(meta, "fps", 12),
		"defenders": sweepRun.detail("defenders", 0), "view": valueOr(meta, "view", 300), "fov": valueOr(meta, "fov", 70),
		"hull": meta["hull"], "planet": meta["planet"], "arena": meta["arena"],
		"genome_switches": child.detail("genome_switches", []any{}), "genome_colors": owner.GenomeColors,
	}
	reader.setFrames(replay, owner, *child)
	timeFields(replay, *child)
	return replay, nil
}

func (reader *Reader) SweepReplayFor(ctx context.Context, owner Owner, n int64) (map[string]any, error) {
	sweepRun, err := reader.sweepRun(ctx, owner, n)
	if err != nil || sweepRun == nil {
		return nil, err
	}
	meta, err := reader.Meta(ctx, owner.BenchmarkID)
	if err != nil {
		return nil, err
	}
	replay := map[string]any{
		"n": sweepRun.Details["n"], "fps": valueOr(meta, "fps", 12), "defenders": sweepRun.detail("defenders", 0),
		"view": valueOr(meta, "view", 300), "fov": valueOr(meta, "fov", 70), "planet": meta["planet"], "arena": meta["arena"],
		"genome_switches": sweepRun.detail("genome_switches", []any{}), "genome_colors": owner.GenomeColors,
	}
	reader.setFrames(replay, owner, *sweepRun)
	timeFields(replay, *sweepRun)
	return replay, nil
}

func (reader *Reader) RunFrames(ctx context.Context, runID int64) ([]byte, error) {
	var binaryKey, framesKey, framesFormat *string
	err := reader.db.QueryRow(ctx, "SELECT binary_key, frames_key, frames_format FROM benchmark_runs WHERE id = $1", runID).
		Scan(&binaryKey, &framesKey, &framesFormat)
	if store.IsNoRows(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if binaryKey != nil {
		if data, found := reader.files.Read(*binaryKey); found {
			return data, nil
		}
	}
	if framesKey == nil {
		return nil, nil
	}
	frames := reader.Frames(Run{FramesKey: framesKey, FramesFormat: framesFormat})
	return EncodeBinaryFrames(frames)
}
