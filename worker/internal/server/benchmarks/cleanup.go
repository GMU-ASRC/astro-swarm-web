package benchmarks

import (
	"context"
	"strings"

	"astroswarm/worker/internal/server/store"
)

func escapeLike(text string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(text)
}

func uploadReferenced(ctx context.Context, db store.Querier, evaluationID string, uploadID string) (bool, error) {
	pattern := escapeLike(evaluationID+"/"+uploadID+"/") + "%"
	var referenced bool
	err := db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM benchmark_runs WHERE frames_key LIKE $1 ESCAPE '\' OR binary_key LIKE $1 ESCAPE '\')`, pattern).Scan(&referenced)
	if err != nil || referenced || uploadID != InlineUploadID {
		return referenced, err
	}
	err = db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM benchmark_runs
		WHERE details->>'`+extraPackedKey+`' LIKE $1 ESCAPE '\' OR details->>'`+extraPlainKey+`' LIKE $1 ESCAPE '\')`, pattern).Scan(&referenced)
	return referenced, err
}

func RemoveUnusedUploads(ctx context.Context, db store.Querier, files *Files, evaluationID string, keepUploadID string) error {
	uploadIDs, err := files.UploadIDs(evaluationID)
	if err != nil {
		return nil
	}
	for _, uploadID := range uploadIDs {
		if uploadID == keepUploadID {
			continue
		}
		referenced, err := uploadReferenced(ctx, db, evaluationID, uploadID)
		if err != nil {
			return err
		}
		if !referenced {
			files.RemoveUpload(evaluationID, uploadID)
		}
	}
	return nil
}
