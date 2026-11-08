package benchmarks

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"astroswarm/worker/internal/server/pytime"
	"astroswarm/worker/internal/server/store"
	"astroswarm/worker/internal/server/values"
)

const (
	RunKindTrial      = "trial"
	RunKindSweep      = "sweep"
	RunKindSweepTrial = "sweep_trial"
	extraPackedKey    = "inline_frames_packed_key"
	extraPlainKey     = "inline_frames_plain_key"
)

var (
	replayListKeys = map[string]bool{"runs": true, "sweep_runs": true, "pending_run": true}
	rateColumns    = []string{"success_rate", "attacker_rate", "defender_rate", "detection_rate", "capture_rate"}
)

type Builder struct {
	files *Files
}

func NewBuilder(files *Files) *Builder {
	return &Builder{files: files}
}

func replayMeta(replays map[string]any) map[string]any {
	meta := map[string]any{}
	for key, value := range replays {
		if !replayListKeys[key] {
			meta[key] = value
		}
	}
	return meta
}

func runDetails(run map[string]any) map[string]any {
	details := map[string]any{}
	for key, value := range run {
		if key != "frames_packed" && key != "frames" && key != "trial_runs" {
			details[key] = value
		}
	}
	return details
}

func wholeNumber(value any) *int64 {
	if whole, isWhole := values.Integer(value); isWhole {
		return &whole
	}
	if number, isNumber := values.Number(value); isNumber && number == float64(int64(number)) {
		whole := int64(number)
		return &whole
	}
	return nil
}

func outcomeColumn(value any) *string {
	text, isText := value.(string)
	if !isText {
		return nil
	}
	truncated := values.Truncate(text, 20)
	return &truncated
}

func (builder *Builder) framesColumns(evaluationID string, run map[string]any, fileName string, details map[string]any) (*string, *string, *int64, error) {
	framesFile, _ := run["frames_file"].(string)
	packed, _ := run["frames_packed"].(string)
	plain, hasPlain := run["frames"].([]any)
	var packedKey, plainKey string
	if packed != "" {
		data, err := UnpackBase64(packed)
		if err != nil {
			return nil, nil, nil, err
		}
		if packedKey, err = builder.files.WriteFile(evaluationID, InlineUploadID, fileName+"-packed", data); err != nil {
			return nil, nil, nil, err
		}
	}
	if hasPlain && len(plain) > 0 {
		data, err := encodePlain(plain)
		if err != nil {
			return nil, nil, nil, err
		}
		if plainKey, err = builder.files.WriteFile(evaluationID, InlineUploadID, fileName+"-plain", data); err != nil {
			return nil, nil, nil, err
		}
	}
	var mainKey, mainFormat string
	switch {
	case framesFile != "":
		mainKey, mainFormat = framesFile, FormatDelta
	case packedKey != "":
		mainKey, mainFormat = packedKey, FormatDelta
	case plainKey != "":
		mainKey, mainFormat = plainKey, FormatPlain
	default:
		return nil, nil, nil, nil
	}
	if packedKey != "" && packedKey != mainKey {
		details[extraPackedKey] = packedKey
	}
	if plainKey != "" && plainKey != mainKey {
		details[extraPlainKey] = plainKey
	}
	return &mainKey, &mainFormat, builder.files.Size(mainKey), nil
}

type runRow struct {
	kind           string
	position       int
	parentPosition *int
	sweepN         *int64
	run            map[string]any
	fileName       string
}

func (builder *Builder) Build(ctx context.Context, db store.Querier, evaluationID string, levelID string, results any, replays map[string]any, contentHash *string) (string, error) {
	runs := objectList(replays["runs"])
	sweepRuns := objectList(replays["sweep_runs"])
	rates, _ := results.(map[string]any)
	summary, err := store.JSONParameter(results)
	if err != nil {
		return "", err
	}
	meta, err := store.JSONParameter(replayMeta(replays))
	if err != nil {
		return "", err
	}
	rateValues := make([]any, len(rateColumns))
	for index, column := range rateColumns {
		if number, convertErr := values.ToFloat(rates[column]); convertErr == nil && rates[column] != nil {
			if _, isBool := rates[column].(bool); !isBool {
				rateValues[index] = number
			}
		}
	}
	benchmarkID := uuid.NewString()
	_, err = db.Exec(ctx, `INSERT INTO benchmarks (id, content_hash, level_id, success_rate, attacker_rate, defender_rate,
		detection_rate, capture_rate, summary, replay_meta, replays_shape, source_evaluation_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::jsonb, $10::jsonb, 'object', $11, $12)`,
		benchmarkID, contentHash, levelID, rateValues[0], rateValues[1], rateValues[2], rateValues[3], rateValues[4],
		summary, meta, evaluationID, pytime.Now())
	if err != nil {
		return "", err
	}

	rows := []runRow{}
	for position, run := range runs {
		rows = append(rows, runRow{kind: RunKindTrial, position: position, run: run, fileName: fmt.Sprintf("trial-%d", position)})
	}
	for position, sweepRun := range sweepRuns {
		sweepN := wholeNumber(sweepRun["n"])
		rows = append(rows, runRow{kind: RunKindSweep, position: position, sweepN: sweepN, run: sweepRun, fileName: fmt.Sprintf("sweep-%d", position)})
		for childPosition, child := range objectList(sweepRun["trial_runs"]) {
			parent := position
			rows = append(rows, runRow{
				kind: RunKindSweepTrial, position: childPosition, parentPosition: &parent, sweepN: sweepN, run: child,
				fileName: fmt.Sprintf("sweep-%d-trial-%d", position, childPosition),
			})
		}
	}
	for _, row := range rows {
		if err := builder.insertRun(ctx, db, evaluationID, benchmarkID, row); err != nil {
			return "", err
		}
	}
	return benchmarkID, nil
}

func (builder *Builder) insertRun(ctx context.Context, db store.Querier, evaluationID string, benchmarkID string, row runRow) error {
	details := runDetails(row.run)
	framesKey, framesFormat, framesBytes, err := builder.framesColumns(evaluationID, row.run, row.fileName, details)
	if err != nil {
		return err
	}
	sweepN := row.sweepN
	if row.kind == RunKindTrial {
		sweepN = wholeNumber(row.run["n"])
	}
	detailsJSON, err := store.JSONParameter(details)
	if err != nil {
		return err
	}
	_, err = db.Exec(ctx, `INSERT INTO benchmark_runs (benchmark_id, kind, position, parent_position, sweep_n, trial, outcome,
		details, frames_key, frames_format, frames_bytes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8::jsonb, $9, $10, $11)`,
		benchmarkID, row.kind, row.position, row.parentPosition, sweepN, wholeNumber(row.run["trial"]),
		outcomeColumn(row.run["outcome"]), detailsJSON, framesKey, framesFormat, framesBytes)
	return err
}

func FindByHash(ctx context.Context, db store.Querier, contentHash string) (string, bool, error) {
	if contentHash == "" {
		return "", false, nil
	}
	var benchmarkID string
	err := db.QueryRow(ctx, "SELECT id FROM benchmarks WHERE content_hash = $1", contentHash).Scan(&benchmarkID)
	if store.IsNoRows(err) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return benchmarkID, true, nil
}

func Release(ctx context.Context, db store.Querier, benchmarkID *string) error {
	if benchmarkID == nil || *benchmarkID == "" {
		return nil
	}
	var stillUsed bool
	if err := db.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM player_evaluations WHERE benchmark_id = $1)", *benchmarkID).Scan(&stillUsed); err != nil {
		return err
	}
	if stillUsed {
		return nil
	}
	if _, err := db.Exec(ctx, "DELETE FROM benchmark_runs WHERE benchmark_id = $1", *benchmarkID); err != nil {
		return err
	}
	_, err := db.Exec(ctx, "DELETE FROM benchmarks WHERE id = $1", *benchmarkID)
	return err
}
