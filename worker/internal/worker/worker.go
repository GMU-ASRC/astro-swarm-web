package worker

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Settings struct {
	ServerURL        string
	APIKey           string
	Name             string
	Hostname         string
	PollInterval     time.Duration
	CancelPoll       time.Duration
	JobTimeout       time.Duration
	IDFile           string
	SimWorkers       int
	VariedSweepSpawn bool
}

type Worker struct {
	id       string
	settings Settings
	client   *Client
	logger   *log.Logger

	mutex     sync.Mutex
	currentID string
}

func New(settings Settings, logger *log.Logger) (*Worker, error) {
	id, err := loadOrCreateID(settings.IDFile, logger)
	if err != nil {
		return nil, err
	}
	return &Worker{
		id:       id,
		settings: settings,
		client:   NewClient(settings.ServerURL, settings.APIKey),
		logger:   logger,
	}, nil
}

func (w *Worker) ID() string {
	return w.id
}

func loadOrCreateID(path string, logger *log.Logger) (string, error) {
	if path != "" {
		if raw, err := os.ReadFile(path); err == nil {
			if value := trimSpace(string(raw)); value != "" {
				return value, nil
			}
		}
	}
	value, err := newUUID()
	if err != nil {
		return "", err
	}
	if path == "" {
		return value, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		logger.Printf("could not create %s (%v); using an ephemeral worker id", filepath.Dir(path), err)
		return value, nil
	}
	if err := os.WriteFile(path, []byte(value), 0o644); err != nil {
		logger.Printf("could not persist the worker id to %s (%v); using an ephemeral id", path, err)
	}
	return value, nil
}

func newUUID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	bytes[6] = (bytes[6] & 0x0f) | 0x40
	bytes[8] = (bytes[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", bytes[0:4], bytes[4:6], bytes[6:8], bytes[8:10], bytes[10:16]), nil
}

func trimSpace(value string) string {
	start := 0
	end := len(value)
	for start < end && isSpace(value[start]) {
		start++
	}
	for end > start && isSpace(value[end-1]) {
		end--
	}
	return value[start:end]
}

func isSpace(character byte) bool {
	return character == ' ' || character == '\t' || character == '\n' || character == '\r'
}

func (w *Worker) Run(ctx context.Context) {
	w.logger.Printf("worker starting; server=%s id=%s", w.settings.ServerURL, w.id)
	w.register(ctx)

	for {
		if ctx.Err() != nil {
			w.waitUntilIdle()
			return
		}

		if !w.busy() {
			if claimed := w.claimOne(ctx); claimed != nil {
				w.startJob(ctx, *claimed)
			}
		}
		w.heartbeat()

		select {
		case <-ctx.Done():
		case <-time.After(w.settings.PollInterval):
		}
	}
}

func (w *Worker) register(ctx context.Context) {
	for ctx.Err() == nil {
		_, err := w.client.Register(RegisterRequest{
			WorkerID: w.id,
			Name:     w.settings.Name,
			Hostname: w.settings.Hostname,
		})
		if err == nil {
			w.logger.Printf("registered as %s (%s)", w.settings.Name, w.id)
			return
		}
		w.logger.Printf("register failed: %v", err)
		select {
		case <-ctx.Done():
		case <-time.After(w.settings.PollInterval):
		}
	}
}

func (w *Worker) claimOne(ctx context.Context) *Shard {
	response, err := w.client.Claim(ClaimRequest{WorkerID: w.id, Slots: 1})
	if err != nil {
		w.logger.Printf("claim failed: %v", err)
		return nil
	}
	if response.Known != nil && !*response.Known {
		w.logger.Printf("server does not know this worker; re-registering")
		w.register(ctx)
		return nil
	}
	if response.Enabled != nil && !*response.Enabled {
		return nil
	}
	if len(response.Shards) == 0 {
		return nil
	}
	return &response.Shards[0]
}

func (w *Worker) startJob(parent context.Context, shard Shard) {
	w.mutex.Lock()
	w.currentID = shard.EvaluationID
	w.mutex.Unlock()

	go func() {
		defer func() {
			w.mutex.Lock()
			w.currentID = ""
			w.mutex.Unlock()
		}()
		w.execute(parent, shard)
	}()
}

func (w *Worker) busy() bool {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	return w.currentID != ""
}

func (w *Worker) waitUntilIdle() {
	for w.busy() {
		time.Sleep(200 * time.Millisecond)
	}
}

func (w *Worker) execute(parent context.Context, shard Shard) {
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

	go w.pollCancellation(ctx, shard, state, stopPolling, markCancelled)

	report := func(done int, stage string) {
		state.set(done, stage)
	}

	result, err := w.executeShard(ctx, shard, report)
	stop()

	select {
	case <-cancelled:
		w.logger.Printf("job %s (shard %s): cancelled", shard.EvaluationID, shard.ShardID)
		return
	default:
	}

	if err != nil {
		if errors.Is(err, ErrCancelled) || errors.Is(ctx.Err(), context.Canceled) {
			w.logger.Printf("job %s (shard %s): cancelled", shard.EvaluationID, shard.ShardID)
			return
		}
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			err = errors.New("job exceeded the configured timeout")
		}
		w.logger.Printf("job %s (shard %s): failed: %v", shard.EvaluationID, shard.ShardID, err)
		if failErr := w.client.Fail(shard.ShardID, FailRequest{WorkerID: w.id, Error: truncate(err.Error(), 400)}); failErr != nil {
			w.logger.Printf("shard %s: fail post failed: %v", shard.ShardID, failErr)
		}
		return
	}

	if postErr := w.client.PostResult(shard.ShardID, ResultRequest{WorkerID: w.id, Result: result}); postErr != nil {
		w.logger.Printf("shard %s: result post failed: %v", shard.ShardID, postErr)
		return
	}
	w.logger.Printf("job %s (shard %s): done in %.1fs (%d runs, %d sweep points)",
		shard.EvaluationID, shard.ShardID, time.Since(started).Seconds(), len(result.Runs), len(result.SweepRuns))
}

func (w *Worker) pollCancellation(ctx context.Context, shard Shard, state *progressState, stop <-chan struct{}, markCancelled func()) {
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
			cancel, err := w.client.Progress(shard.ShardID, ProgressRequest{
				WorkerID: w.id,
				Done:     done,
				Stage:    stage,
			})
			if err != nil {
				w.logger.Printf("shard %s: progress post failed: %v", shard.ShardID, err)
				continue
			}
			if cancel {
				markCancelled()
				return
			}
		}
	}
}

func (w *Worker) heartbeat() {
	w.mutex.Lock()
	current := w.currentID
	w.mutex.Unlock()

	status := "idle"
	if current != "" {
		status = "busy"
	}

	if _, err := w.client.Heartbeat(HeartbeatRequest{
		WorkerID:    w.id,
		Status:      status,
		CurrentJob:  current,
		SystemStats: collectSystemStats(),
	}); err != nil {
		w.logger.Printf("heartbeat failed: %v", err)
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
