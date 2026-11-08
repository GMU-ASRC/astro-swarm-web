package benchmarks

import (
	"context"
	"fmt"

	"astroswarm/worker/internal/server/store"
)

const RetireLegacyFramesVersion = 8

func LegacyFramesRetired(ctx context.Context, db store.Querier) (bool, error) {
	var status *string
	err := db.QueryRow(ctx, "SELECT status FROM schema_migrations WHERE version = $1", RetireLegacyFramesVersion).Scan(&status)
	if store.IsNoRows(err) {
		return false, nil
	}
	return status != nil && *status == "done", err
}

type retireCandidate struct {
	id           int64
	framesKey    string
	framesFormat string
	binaryKey    string
}

type RetireResult struct {
	Removed  int
	Problems []string
}

func RetireLegacyBatch(ctx context.Context, db store.Querier, files *Files, afterID int64, limit int) (RetireResult, int64, error) {
	rows, err := db.Query(ctx, `SELECT id, frames_key, coalesce(frames_format, 'delta'), binary_key FROM benchmark_runs
		WHERE binary_key IS NOT NULL AND frames_key IS NOT NULL AND id > $1 ORDER BY id LIMIT $2`, afterID, limit)
	if err != nil {
		return RetireResult{}, afterID, err
	}
	candidates := []retireCandidate{}
	for rows.Next() {
		var candidate retireCandidate
		if err := rows.Scan(&candidate.id, &candidate.framesKey, &candidate.framesFormat, &candidate.binaryKey); err != nil {
			rows.Close()
			return RetireResult{}, afterID, err
		}
		candidates = append(candidates, candidate)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return RetireResult{}, afterID, err
	}
	result := RetireResult{}
	lastID := afterID
	for _, candidate := range candidates {
		lastID = candidate.id
		if err := files.VerifyBinary(candidate.framesKey, candidate.framesFormat, candidate.binaryKey); err != nil {
			result.Problems = append(result.Problems, fmt.Sprintf("run %d kept its old file: %v", candidate.id, err))
			continue
		}
		var shared bool
		if err := db.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM benchmark_runs WHERE frames_key = $1 AND id <> $2)",
			candidate.framesKey, candidate.id).Scan(&shared); err != nil {
			return result, lastID, err
		}
		if _, err := db.Exec(ctx, `UPDATE benchmark_runs SET frames_key = NULL, frames_format = NULL, frames_bytes = NULL
			WHERE id = $1 AND frames_key = $2`, candidate.id, candidate.framesKey); err != nil {
			return result, lastID, err
		}
		if !shared {
			if err := files.Remove(candidate.framesKey); err != nil {
				result.Problems = append(result.Problems, fmt.Sprintf("run %d: could not delete %s: %v", candidate.id, candidate.framesKey, err))
				continue
			}
		}
		result.Removed++
	}
	if len(candidates) < limit {
		return result, -1, nil
	}
	return result, lastID, nil
}

func RetireAllLegacy(ctx context.Context, db store.Querier, files *Files, batchSize int, progress func(removed int)) ([]string, error) {
	problems := []string{}
	afterID, removed := int64(0), 0
	for afterID >= 0 {
		result, nextID, err := RetireLegacyBatch(ctx, db, files, afterID, batchSize)
		problems = append(problems, result.Problems...)
		removed += result.Removed
		if err != nil {
			return problems, err
		}
		afterID = nextID
		progress(removed)
	}
	return problems, nil
}
