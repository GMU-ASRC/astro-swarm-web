package worker

import (
	"context"
	"log"
	"runtime"
	"sort"
	"sync"
	"time"

	"astroswarm/worker/internal/bench"
)

type Settings struct {
	ServerURL        string
	APIKey           string
	Name             string
	Hostname         string
	PollInterval     time.Duration
	CancelPoll       time.Duration
	JobTimeout       time.Duration
	ID               string
	IDFile           string
	Cores            int
	JobSlots         int
	CoresPerJob      int
	SimWorkers       int
	UploadChunkBytes int
	VariedSweepSpawn bool
}

type Worker struct {
	id       string
	settings Settings
	capacity Capacity
	cores    *bench.CorePool
	client   *Client
	logger   *log.Logger

	mutex       sync.Mutex
	runningJobs map[string]struct{}
	jobFinished chan struct{}
}

func New(settings Settings, logger *log.Logger) (*Worker, error) {
	id, err := loadOrCreateID(settings.ID, settings.IDFile, logger)
	if err != nil {
		return nil, err
	}
	capacity := planCapacity(settings)
	if capacity.Cores < runtime.GOMAXPROCS(0) {
		runtime.GOMAXPROCS(capacity.Cores)
	}
	return &Worker{
		id:          id,
		settings:    settings,
		capacity:    capacity,
		cores:       bench.NewCorePool(capacity.Cores),
		client:      NewClient(settings.ServerURL, settings.APIKey),
		logger:      logger,
		runningJobs: map[string]struct{}{},
		jobFinished: make(chan struct{}, 1),
	}, nil
}

func (w *Worker) ID() string {
	return w.id
}

func (w *Worker) Capacity() Capacity {
	return w.capacity
}

func (w *Worker) Run(ctx context.Context) {
	w.logger.Printf("worker starting; server=%s id=%s", w.settings.ServerURL, w.id)
	w.register(ctx)

	for {
		if ctx.Err() != nil {
			w.waitUntilIdle()
			return
		}

		w.fillFreeSlots(ctx)
		w.heartbeat()

		select {
		case <-ctx.Done():
		case <-w.jobFinished:
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

func (w *Worker) fillFreeSlots(ctx context.Context) {
	freeSlots := w.capacity.JobSlots - w.runningCount()
	if freeSlots < 1 {
		return
	}
	for _, job := range w.claim(ctx, freeSlots) {
		w.startJob(ctx, job)
	}
}

func (w *Worker) claim(ctx context.Context, slots int) []Job {
	response, err := w.client.Claim(ClaimRequest{WorkerID: w.id, Slots: slots})
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
	if len(response.Jobs) > slots {
		return response.Jobs[:slots]
	}
	return response.Jobs
}

func (w *Worker) startJob(parent context.Context, job Job) {
	w.mutex.Lock()
	w.runningJobs[job.EvaluationID] = struct{}{}
	w.mutex.Unlock()

	go func() {
		defer func() {
			w.mutex.Lock()
			delete(w.runningJobs, job.EvaluationID)
			w.mutex.Unlock()
			select {
			case w.jobFinished <- struct{}{}:
			default:
			}
		}()
		w.execute(parent, job)
	}()
}

func (w *Worker) runningCount() int {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	return len(w.runningJobs)
}

func (w *Worker) runningIDs() []string {
	w.mutex.Lock()
	ids := make([]string, 0, len(w.runningJobs))
	for id := range w.runningJobs {
		ids = append(ids, id)
	}
	w.mutex.Unlock()
	sort.Strings(ids)
	return ids
}

func (w *Worker) waitUntilIdle() {
	for w.runningCount() > 0 {
		time.Sleep(200 * time.Millisecond)
	}
}

func (w *Worker) heartbeat() {
	jobIDs := w.runningIDs()
	status := "idle"
	currentJob := ""
	if len(jobIDs) > 0 {
		status = "busy"
		currentJob = jobIDs[0]
	}

	stats := collectSystemStats()
	stats["usable_cores"] = w.capacity.Cores
	stats["job_slots"] = w.capacity.JobSlots
	stats["running_jobs"] = len(jobIDs)
	stats["busy_cores"] = w.cores.Busy()

	if _, err := w.client.Heartbeat(HeartbeatRequest{
		WorkerID:    w.id,
		Status:      status,
		CurrentJob:  currentJob,
		CurrentJobs: jobIDs,
		JobSlots:    w.capacity.JobSlots,
		SystemStats: stats,
	}); err != nil {
		w.logger.Printf("heartbeat failed: %v", err)
	}
}
