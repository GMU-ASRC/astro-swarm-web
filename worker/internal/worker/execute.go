package worker

import (
	"context"
	"errors"
	"sync"
	"time"

	"astroswarm/worker/internal/bench"
)

const (
	resultPostAttempts = 3               // count, tries before the job is reported as failed
	resultPostBackoff  = 5 * time.Second // seconds added between retries
	finalizeStage      = "Saving results"
)

func (w *Worker) execute(parent context.Context, job Job) {
	started := time.Now()
	ctx, cancel := context.WithTimeout(parent, w.settings.JobTimeout)
	defer cancel()

	state := &progressState{}
	stopPolling := make(chan struct{})
	var pollOnce sync.Once
	stop := func() { pollOnce.Do(func() { close(stopPolling) }) }
	defer stop()

	cancelled := make(chan struct{})
	var cancelOnce sync.Once
	markCancelled := func() {
		cancelOnce.Do(func() {
			close(cancelled)
			cancel()
		})
	}

	go w.pollCancellation(ctx, job, state, stopPolling, markCancelled)

	result, err := w.executeJob(ctx, job, state.set)

	select {
	case <-cancelled:
		stop()
		w.logger.Printf("job %s: cancelled", job.EvaluationID)
		return
	default:
	}

	if err != nil {
		stop()
		if errors.Is(err, ErrCancelled) || errors.Is(ctx.Err(), context.Canceled) {
			w.logger.Printf("job %s: cancelled", job.EvaluationID)
			return
		}
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			err = errors.New("job exceeded the configured timeout")
		}
		w.logger.Printf("job %s: failed: %v", job.EvaluationID, err)
		w.reportFailure(job, err.Error())
		return
	}

	simulated := time.Since(started)
	uploadStarted := time.Now()
	uploadID, uploadErr := w.uploadReplays(ctx, job, &result, state)
	if uploadErr == nil {
		state.set(state.doneCount(), finalizeStage)
		uploadErr = w.postResult(job, uploadID, result)
	}
	stop()

	select {
	case <-cancelled:
		w.logger.Printf("job %s: cancelled during upload", job.EvaluationID)
		return
	default:
	}
	if errors.Is(uploadErr, errJobReleased) {
		w.logger.Printf("job %s: dropped, the server reassigned or cancelled it", job.EvaluationID)
		return
	}
	if uploadErr != nil {
		w.logger.Printf("job %s: result never reached the server: %v", job.EvaluationID, uploadErr)
		w.reportFailure(job, "result upload failed: "+uploadErr.Error())
		return
	}
	w.logger.Printf("job %s: simulated in %.1fs, uploaded in %.1fs (%d runs, %d sweep points)",
		job.EvaluationID, simulated.Seconds(), time.Since(uploadStarted).Seconds(), len(result.Runs), len(result.SweepRuns))
}

func (w *Worker) reportFailure(job Job, message string) {
	if err := w.client.Fail(job.JobID, FailRequest{WorkerID: w.id, Error: truncate(message, 400)}); err != nil {
		w.logger.Printf("job %s: fail post failed: %v", job.JobID, err)
	}
}

func (w *Worker) postResult(job Job, uploadID string, result bench.JobResult) error {
	body, err := encodeGzipJSON(ResultRequest{WorkerID: w.id, UploadID: uploadID, Result: result})
	if err != nil {
		return err
	}
	for attempt := 1; attempt <= resultPostAttempts; attempt++ {
		var accepted bool
		accepted, err = w.client.PostResult(job.JobID, body)
		if err == nil {
			if !accepted {
				return errJobReleased
			}
			return nil
		}
		w.logger.Printf("job %s: result post attempt %d/%d failed: %v", job.JobID, attempt, resultPostAttempts, err)
		if attempt < resultPostAttempts {
			time.Sleep(resultPostBackoff * time.Duration(attempt))
		}
	}
	return err
}

func (w *Worker) pollCancellation(ctx context.Context, job Job, state *progressState, stop <-chan struct{}, markCancelled func()) {
	ticker := time.NewTicker(w.settings.CancelPoll)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			done, stage := state.get()
			cancel, err := w.client.Progress(job.JobID, ProgressRequest{
				WorkerID: w.id,
				Done:     done,
				Stage:    stage,
			})
			if err != nil {
				w.logger.Printf("job %s: progress post failed: %v", job.JobID, err)
				continue
			}
			if cancel {
				markCancelled()
				return
			}
		}
	}
}

type progressState struct {
	mutex sync.Mutex
	done  int
	stage string
}

func (p *progressState) set(done int, stage string) {
	p.mutex.Lock()
	p.done = done
	if stage != "" {
		p.stage = stage
	}
	p.mutex.Unlock()
}

func (p *progressState) get() (int, string) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	return p.done, p.stage
}

func (p *progressState) doneCount() int {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	return p.done
}
