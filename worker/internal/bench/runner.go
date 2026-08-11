package bench

import (
	"context"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"astroswarm/worker/internal/blocks"
	"astroswarm/worker/internal/godot"
)

type Options struct {
	LevelID          string
	Algorithm        []blocks.Script
	Placements       []Placement
	Trials           int
	TrialStart       int
	TrialCount       int
	Seed             int64
	SweepMax         int
	SweepTrials      int
	NStart           int
	NCount           int
	MatchSeconds     float64
	GoalTailSeconds  float64
	EnemyStart       godot.Vec
	HasEnemyStart    bool
	Collisions       bool
	SinglePrecision  bool
	VariedSweepSpawn bool
	Record           bool
	Workers          int
	Context          context.Context
	Progress         func(done, total int)
}

func (o *Options) applyDefaults() {
	if o.LevelID == "" {
		o.LevelID = "farp1"
	}
	if o.Trials <= 0 {
		o.Trials = DefaultTrials
	}
	if o.SweepMax <= 0 {
		o.SweepMax = DefaultSweepMax
	}
	if o.SweepTrials <= 0 {
		o.SweepTrials = DefaultSweepTrials
	}
	if o.Seed == 0 {
		o.Seed = DefaultSeed
	}
	if o.MatchSeconds <= 0 {
		o.MatchSeconds = DefaultMatchSeconds
	}
	if o.GoalTailSeconds < 0 {
		o.GoalTailSeconds = DefaultGoalTailSeconds
	}
	if o.NStart < 1 {
		o.NStart = 1
	}
	if o.TrialCount < 0 {
		o.TrialCount = o.Trials
	}
	if o.NCount < 0 {
		o.NCount = o.SweepMax
	}
	if o.Workers <= 0 {
		o.Workers = runtime.NumCPU()
	}
	if o.Context == nil {
		o.Context = context.Background()
	}
	if len(o.Placements) > MaxDefenders {
		o.Placements = o.Placements[:MaxDefenders]
	}
}

type job struct {
	sweep      bool
	trial      int
	defenders  int
	seed       int64
	placements []Placement
	spawn      godot.Vec
	record     bool
}

type jobResult struct {
	job    job
	output MatchOutput
	ran    bool
}

func Run(options Options) Report {
	report, _ := RunJob(options)
	return report
}

func RunJob(options Options) (Report, JobResult) {
	options.applyDefaults()
	started := time.Now()

	spawnPoints := SpawnPoints(options.Seed, options.Trials)
	enemyStart := options.EnemyStart
	if !options.HasEnemyStart {
		enemyStart = spawnPoints[0]
	}

	matchFrames := int(options.MatchSeconds * PhysicsTicksPerSecond)
	goalTailFrames := int(options.GoalTailSeconds * PhysicsTicksPerSecond)

	placementLayout := options.Placements
	if len(placementLayout) == 0 && LevelNumber(options.LevelID) == 2 {
		placementLayout = ScatterLayout(options.Seed, RingCount)
	}

	jobs := buildJobs(options, placementLayout, spawnPoints, enemyStart)
	results := runJobs(jobs, options, matchFrames, goalTailFrames)

	report := Report{
		LevelID:         options.LevelID,
		Defenders:       len(placementLayout),
		Seed:            options.Seed,
		SweepTrials:     options.SweepTrials,
		SweepSpawn:      sweepSpawnLabel(options.VariedSweepSpawn),
		Collisions:      options.Collisions,
		SinglePrecision: options.SinglePrecision,
		MatchSeconds:    options.MatchSeconds,
		MatchesRun:      len(jobs),
	}

	placement, sweep := splitResults(results, &report)
	fillPlacementResults(&report, placement)
	report.Results.Sweep = summarizeSweep(sweep)
	report.DurationSeconds = time.Since(started).Seconds()

	jobResult := JobResult{
		Runs:      packPlacementRuns(placement),
		SweepRuns: packSweepRuns(sweep),
		Meta: JobMeta{
			FPS:       PhysicsTicksPerSecond,
			Defenders: report.Defenders,
			View:      int(report.ViewDistance),
			Fov:       int(report.FovDegrees),
			Speed:     int(report.Speed),
			Hull:      int(report.HullRadius),
			Planet:    []int{int(PlanetX), int(PlanetY), int(PlanetRadius)},
			Arena:     []int{int(ArenaWidth), int(ArenaHeight)},
		},
	}
	return report, jobResult
}

func buildJobs(options Options, placementLayout []Placement, spawnPoints []godot.Vec, enemyStart godot.Vec) []job {
	jobs := make([]job, 0, options.TrialCount+options.NCount*options.SweepTrials)

	for offset := 0; offset < options.TrialCount; offset++ {
		trial := options.TrialStart + offset
		jobs = append(jobs, job{
			trial:      trial,
			seed:       options.Seed + int64(trial),
			placements: placementLayout,
			spawn:      spawnPoints[trial%len(spawnPoints)],
			record:     options.Record,
		})
	}

	var sweepSpawns []godot.Vec
	if options.VariedSweepSpawn {
		sweepSpawns = SweepSpawnPoints(options.Seed, options.SweepTrials)
	}
	for index := 0; index < options.NCount; index++ {
		defenders := options.NStart + index
		for trial := 0; trial < options.SweepTrials; trial++ {
			spawn := enemyStart
			if len(sweepSpawns) > 0 {
				spawn = sweepSpawns[trial%len(sweepSpawns)]
			}
			jobs = append(jobs, job{
				sweep:      true,
				trial:      trial,
				defenders:  defenders,
				seed:       SweepTrialSeed(options.Seed, trial, options.SweepTrials) + SweepMatchOffset + int64(defenders),
				placements: RingPlacements(options.Seed, trial, options.SweepTrials, defenders),
				spawn:      spawn,
				record:     options.Record && isReplayTrial(defenders, trial),
			})
		}
	}
	return jobs
}

func isReplayTrial(defenders int, trial int) bool {
	if trial == 0 {
		return true
	}
	return defenders < ReplaySweepNMax && trial < ReplaySweepTrials
}

func isTrialReplay(defenders int, trial int) bool {
	return defenders < ReplaySweepNMax && trial < ReplaySweepTrials
}

func splitResults(results []jobResult, report *Report) (placement []jobResult, sweep []jobResult) {
	for _, item := range results {
		if !item.ran {
			continue
		}
		if report.ViewDistance == 0 {
			report.ViewDistance = item.output.ViewDistance
			report.FovDegrees = item.output.FovDegrees
			report.Speed = item.output.Speed
			report.HullRadius = item.output.HullRadius
		}
		if item.job.sweep {
			sweep = append(sweep, item)
		} else {
			placement = append(placement, item)
		}
	}
	sort.SliceStable(placement, func(a, b int) bool {
		return placement[a].job.trial < placement[b].job.trial
	})
	return placement, sweep
}

func fillPlacementResults(report *Report, placement []jobResult) {
	outcomes := make([]string, 0, len(placement))
	detectionTimes := make([]float64, 0, len(placement))
	captureTimes := make([]float64, 0, len(placement))
	goalTimes := make([]float64, 0, len(placement))
	counts := OutcomeCounts{}
	goalTotal := 0.0
	goalSamples := 0

	for _, item := range placement {
		output := item.output
		report.Runs = append(report.Runs, TrialRun{
			Trial:         item.job.trial,
			Outcome:       output.Outcome,
			DetectionTime: output.DetectionTime,
			CaptureTime:   output.CaptureTime,
			GoalTime:      output.GoalTime,
		})
		outcomes = append(outcomes, output.Outcome)
		detectionTimes = append(detectionTimes, output.DetectionTime)
		captureTimes = append(captureTimes, output.CaptureTime)
		goalTimes = append(goalTimes, output.GoalTime)
		switch output.Outcome {
		case OutcomeWin:
			counts.Captured++
		case OutcomeLose:
			counts.ReachedPlanet++
		default:
			counts.TimedOut++
		}
		if output.GoalTime >= 0.0 {
			goalTotal += output.GoalTime
			goalSamples++
		}
	}

	total := len(outcomes)
	if total < 1 {
		total = 1
	}
	meanGoal := 0.0
	if goalSamples > 0 {
		meanGoal = round1(goalTotal / float64(goalSamples))
	}

	report.Results = Results{
		Trials:          len(outcomes),
		SuccessRate:     round1(100.0 * float64(counts.Captured) / float64(total)),
		DetectionRate:   rate(detectionTimes),
		CaptureRate:     rate(captureTimes),
		OutcomeCounts:   counts,
		MeanGoalSeconds: meanGoal,
		GoalSamples:     goalSamples,
		Outcomes:        outcomes,
		DetectionTimes:  detectionTimes,
		CaptureTimes:    captureTimes,
		GoalTimes:       goalTimes,
	}
}

type sweepBucket struct {
	total    int
	detected int
	captured int
	wins     int
	first    *MatchOutput
	replays  []jobResult
}

func groupSweep(sweep []jobResult) ([]int, map[int]*sweepBucket) {
	buckets := map[int]*sweepBucket{}
	order := []int{}
	for _, item := range sweep {
		defenders := item.job.defenders
		bucket, present := buckets[defenders]
		if !present {
			bucket = &sweepBucket{}
			buckets[defenders] = bucket
			order = append(order, defenders)
		}
		bucket.total++
		if item.output.DetectionTime >= 0.0 {
			bucket.detected++
		}
		if item.output.CaptureTime >= 0.0 {
			bucket.captured++
		}
		if item.output.Outcome == OutcomeWin {
			bucket.wins++
		}
		if item.job.trial == 0 {
			output := item.output
			bucket.first = &output
		}
		if len(item.output.Frames) > 0 && isTrialReplay(defenders, item.job.trial) {
			bucket.replays = append(bucket.replays, item)
		}
	}
	sort.Ints(order)
	return order, buckets
}

func summarizeSweep(sweep []jobResult) []SweepPoint {
	order, buckets := groupSweep(sweep)
	points := make([]SweepPoint, 0, len(order))
	for _, defenders := range order {
		bucket := buckets[defenders]
		trials := bucket.total
		if trials < 1 {
			trials = 1
		}
		point := SweepPoint{
			N:             defenders,
			Trials:        bucket.total,
			DetectionRate: round1(100.0 * float64(bucket.detected) / float64(trials)),
			CaptureRate:   round1(100.0 * float64(bucket.captured) / float64(trials)),
			WinRate:       round1(100.0 * float64(bucket.wins) / float64(trials)),
		}
		point.SuccessRate = point.CaptureRate
		points = append(points, point)
	}
	return points
}

func packPlacementRuns(placement []jobResult) []ReplayRun {
	runs := make([]ReplayRun, 0, len(placement))
	for _, item := range placement {
		runs = append(runs, ReplayRun{
			Trial:         item.job.trial,
			Outcome:       item.output.Outcome,
			DetectionTime: item.output.DetectionTime,
			CaptureTime:   item.output.CaptureTime,
			GoalTime:      item.output.GoalTime,
			FramesPacked:  PackFrames(item.output.Frames),
		})
	}
	return runs
}

func packSweepRuns(sweep []jobResult) []ReplaySweepRun {
	order, buckets := groupSweep(sweep)
	runs := make([]ReplaySweepRun, 0, len(order))
	for _, defenders := range order {
		bucket := buckets[defenders]
		trials := bucket.total
		if trials < 1 {
			trials = 1
		}

		run := ReplaySweepRun{
			N:             defenders,
			Defenders:     defenders,
			Outcome:       OutcomeTimeout,
			DetectionTime: -1.0,
			CaptureTime:   -1.0,
			GoalTime:      -1.0,
			DetectionRate: round1(100.0 * float64(bucket.detected) / float64(trials)),
			CaptureRate:   round1(100.0 * float64(bucket.captured) / float64(trials)),
			TrialRuns:     []ReplayRun{},
		}
		if bucket.first != nil {
			run.Outcome = bucket.first.Outcome
			run.DetectionTime = bucket.first.DetectionTime
			run.CaptureTime = bucket.first.CaptureTime
			run.GoalTime = bucket.first.GoalTime
			run.FramesPacked = PackFrames(bucket.first.Frames)
		}
		sort.SliceStable(bucket.replays, func(a, b int) bool {
			return bucket.replays[a].job.trial < bucket.replays[b].job.trial
		})
		for _, item := range bucket.replays {
			run.TrialRuns = append(run.TrialRuns, ReplayRun{
				Trial:         item.job.trial,
				Outcome:       item.output.Outcome,
				DetectionTime: item.output.DetectionTime,
				CaptureTime:   item.output.CaptureTime,
				GoalTime:      item.output.GoalTime,
				FramesPacked:  PackFrames(item.output.Frames),
			})
		}
		runs = append(runs, run)
	}
	return runs
}

func runJobs(jobs []job, options Options, matchFrames, goalTailFrames int) []jobResult {
	results := make([]jobResult, len(jobs))
	queue := make(chan int)
	var group sync.WaitGroup
	var done int64

	for worker := 0; worker < options.Workers; worker++ {
		group.Add(1)
		go func() {
			defer group.Done()
			for index := range queue {
				if options.Context.Err() != nil {
					continue
				}
				current := jobs[index]
				output := RunMatch(MatchInput{
					Algorithm:       options.Algorithm,
					Placements:      current.placements,
					EnemyStart:      current.spawn,
					Seed:            current.seed,
					MatchFrames:     matchFrames,
					GoalTailFrames:  goalTailFrames,
					Collisions:      options.Collisions,
					SinglePrecision: options.SinglePrecision,
					Record:          current.record,
				})
				results[index] = jobResult{job: current, output: output, ran: true}
				if options.Progress != nil {
					options.Progress(int(atomic.AddInt64(&done, 1)), len(jobs))
				}
			}
		}()
	}

	for index := range jobs {
		if options.Context.Err() != nil {
			break
		}
		queue <- index
	}
	close(queue)
	group.Wait()
	return results
}
