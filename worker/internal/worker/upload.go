package worker

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"astroswarm/worker/internal/bench"
)

const (
	replayUploadWorkers   = 4
	chunkUploadAttempts   = 4
	chunkUploadBackoff    = 2 * time.Second
	defaultUploadChunkKiB = 512
)

var errJobReleased = errors.New("the server no longer assigns this job to this worker")

type replayUpload struct {
	key    string
	packed *string
	file   *string
}

func collectReplayUploads(result *bench.JobResult) []replayUpload {
	uploads := []replayUpload{}
	add := func(key string, packed *string, file *string) {
		if *packed != "" {
			uploads = append(uploads, replayUpload{key: key, packed: packed, file: file})
		}
	}
	for runIndex := range result.Runs {
		run := &result.Runs[runIndex]
		add(fmt.Sprintf("run-%d", runIndex), &run.FramesPacked, &run.FramesFile)
	}
	for sweepIndex := range result.SweepRuns {
		sweep := &result.SweepRuns[sweepIndex]
		add(fmt.Sprintf("sweep-%d", sweepIndex), &sweep.FramesPacked, &sweep.FramesFile)
		for trialIndex := range sweep.TrialRuns {
			trial := &sweep.TrialRuns[trialIndex]
			add(fmt.Sprintf("sweep-%d-trial-%d", sweepIndex, trialIndex), &trial.FramesPacked, &trial.FramesFile)
		}
	}
	return uploads
}

func (w *Worker) uploadReplays(ctx context.Context, job Job, result *bench.JobResult, state *progressState) (string, error) {
	uploadID, err := newUUID()
	if err != nil {
		return "", err
	}
	uploads := collectReplayUploads(result)
	total := len(uploads)
	state.set(state.doneCount(), fmt.Sprintf("Uploading replays 0/%d", total))

	queue := make(chan replayUpload)
	var group sync.WaitGroup
	var uploaded int64
	var firstErr error
	var errOnce sync.Once
	uploadCtx, cancelUploads := context.WithCancel(ctx)
	defer cancelUploads()

	for workerIndex := 0; workerIndex < replayUploadWorkers; workerIndex++ {
		group.Add(1)
		go func() {
			defer group.Done()
			for upload := range queue {
				if uploadCtx.Err() != nil {
					continue
				}
				path, err := w.uploadReplayFile(uploadCtx, job, uploadID, upload)
				if err != nil {
					errOnce.Do(func() {
						firstErr = fmt.Errorf("replay %s: %w", upload.key, err)
						cancelUploads()
					})
					continue
				}
				*upload.file = path
				*upload.packed = ""
				done := atomic.AddInt64(&uploaded, 1)
				state.set(state.doneCount(), fmt.Sprintf("Uploading replays %d/%d", done, total))
			}
		}()
	}

	for _, upload := range uploads {
		if uploadCtx.Err() != nil {
			break
		}
		queue <- upload
	}
	close(queue)
	group.Wait()

	if firstErr != nil {
		return "", firstErr
	}
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	return uploadID, nil
}

func (w *Worker) uploadReplayFile(ctx context.Context, job Job, uploadID string, upload replayUpload) (string, error) {
	data, err := base64.StdEncoding.DecodeString(*upload.packed)
	if err != nil {
		return "", err
	}
	chunkSize := w.settings.UploadChunkBytes
	if chunkSize < 1 {
		chunkSize = defaultUploadChunkKiB * 1024
	}

	path := ""
	offset := 0
	stalls := 0
	for offset < len(data) {
		end := offset + chunkSize
		if end > len(data) {
			end = len(data)
		}
		response, err := w.sendChunk(ctx, job, uploadID, upload.key, offset, data[offset:end])
		if err != nil {
			return "", err
		}
		if response.Size <= offset {
			stalls++
			if stalls > chunkUploadAttempts {
				return "", fmt.Errorf("upload stalled at byte %d", offset)
			}
		}
		offset = response.Size
		path = response.Path
	}
	return path, nil
}

func (w *Worker) sendChunk(ctx context.Context, job Job, uploadID string, key string, offset int, chunk []byte) (ReplayChunkResponse, error) {
	var lastErr error
	for attempt := 1; attempt <= chunkUploadAttempts; attempt++ {
		if ctx.Err() != nil {
			return ReplayChunkResponse{}, ctx.Err()
		}
		response, err := w.client.UploadReplayChunk(job.JobID, uploadID, key, offset, w.id, chunk)
		if err == nil {
			if !response.Accepted {
				return ReplayChunkResponse{}, errJobReleased
			}
			return response, nil
		}
		lastErr = err
		w.logger.Printf("job %s: replay %s chunk at %d attempt %d/%d failed: %v", job.EvaluationID, key, offset, attempt, chunkUploadAttempts, err)
		select {
		case <-ctx.Done():
			return ReplayChunkResponse{}, ctx.Err()
		case <-time.After(chunkUploadBackoff * time.Duration(attempt)):
		}
	}
	return ReplayChunkResponse{}, lastErr
}
