package bench

import (
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

const (
	WaveTrialSeedOffset   = 1100000
	WaveSweepSeedOffset   = 3300000
	WaveReplaySweepNMax   = 20
	WaveReplaySweepTrials = 4
)

type waveJob struct {
	sweep      bool
	trial      int
	defenders  int
	evaders    int
	seed       int64
	placements []Placement
	angles     []float64
	record     bool
}

type waveResult struct {
	job    waveJob
	output WaveOutput
	ran    bool
}

func waveTrialLayout(seed int64, trial int, defenders int) []Placement {
	return ScatterLayout(seed+int64(trial)*SweepSeedStride, defenders)
}

func buildWaveTrials(options Options) []waveJob {
	offset := int64(WaveTrialSeedOffset)
	defenders := len(options.Placements)
	if defenders < 1 {
		defenders = RingCount
	}
	jobs := make([]waveJob, 0, options.TrialCount)
	for index := 0; index < options.TrialCount; index++ {
		trial := options.TrialStart + index
		evaders := WaveEvaderCount(defenders)
		jobs = append(jobs, waveJob{
			trial:      trial,
			defenders:  defenders,
			evaders:    evaders,
			seed:       options.Seed + offset + int64(trial),
			placements: waveTrialLayout(options.Seed+offset, trial, defenders),
			angles:     WaveSpawnAngles(options.Seed+offset, trial, options.TrialCount, evaders),
			record:     options.Record,
		})
	}
	return jobs
}

func buildWaveSweepStep(options Options, defenders int) []waveJob {
	jobs := make([]waveJob, 0, options.SweepTrials)
	for trial := 0; trial < options.SweepTrials; trial++ {
		evaders := WaveEvaderCount(defenders)
		seed := options.Seed + WaveSweepSeedOffset + int64(defenders)*SweepSeedStride + int64(trial)
		jobs = append(jobs, waveJob{
			sweep:      true,
			trial:      trial,
			defenders:  defenders,
			evaders:    evaders,
			seed:       seed,
			placements: RingPlacements(options.Seed+WaveSweepSeedOffset, trial, options.SweepTrials, defenders),
			angles:     WaveSpawnAngles(options.Seed+WaveSweepSeedOffset+int64(defenders), trial, options.SweepTrials, evaders),
			record:     options.Record && (trial == 0 || (defenders <= WaveReplaySweepNMax && trial < WaveReplaySweepTrials)),
		})
	}
	return jobs
}

func runWaveJobs(jobs []waveJob, options Options, matchFrames int, destroysDefenders bool, progress func()) []waveResult {
	results := make([]waveResult, len(jobs))
	queue := make(chan int)
	var group sync.WaitGroup

	for worker := 0; worker < options.Workers; worker++ {
		group.Add(1)
		go func() {
			defer group.Done()
			for index := range queue {
				if options.Context.Err() != nil {
					continue
				}
				current := jobs[index]
				output := RunWaveMatch(WaveInput{
					Algorithm:         options.Algorithm,
					Placements:        current.placements,
					SpawnAngles:       current.angles,
					DestroysDefenders: destroysDefenders,
					Seed:              current.seed,
					MatchFrames:       matchFrames,
					SinglePrecision:   options.SinglePrecision,
					Record:            current.record,
				})
				results[index] = waveResult{job: current, output: output, ran: true}
				if progress != nil {
					progress()
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

func RunWaveJob(options Options) (Report, JobResult) {
	options.applyDefaults()
	started := time.Now()

	destroysDefenders := LevelNumber(options.LevelID) == 4
	matchFrames := int(options.MatchSeconds * PhysicsTicksPerSecond)

	if len(options.Placements) == 0 {
		options.Placements = ScatterLayout(options.Seed, RingCount)
	}

	var done int64
	estimate := int64(options.TrialCount + options.SweepMax*options.SweepTrials)
	tick := func() {
		if options.Progress == nil {
			return
		}
		options.Progress(int(atomic.AddInt64(&done, 1)), int(estimate))
	}

	trials := runWaveJobs(buildWaveTrials(options), options, matchFrames, destroysDefenders, tick)

	sweep, sweepPoints := runWaveSweep(options, matchFrames, destroysDefenders, tick)

	report := Report{
		LevelID:         options.LevelID,
		Defenders:       len(options.Placements),
		Seed:            options.Seed,
		SweepTrials:     options.SweepTrials,
		SweepSpawn:      sweepSpawnLabel(true),
		Collisions:      options.Collisions,
		SinglePrecision: options.SinglePrecision,
		MatchSeconds:    options.MatchSeconds,
		MatchesRun:      len(trials) + len(sweep),
	}

	fillWaveResults(&report, trials)
	report.Results.Sweep = sweepPoints
	report.DurationSeconds = time.Since(started).Seconds()

	if options.Progress != nil {
		options.Progress(int(estimate), int(estimate))
	}

	jobResult := JobResult{
		Runs:      packWaveRuns(trials),
		SweepRuns: packWaveSweepRuns(sweep),
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

func runWaveSweep(options Options, matchFrames int, destroysDefenders bool, tick func()) ([]waveResult, []SweepPoint) {
	all := []waveResult{}
	points := []SweepPoint{}
	clean := 0

	for defenders := options.NStart; defenders <= options.SweepMax; defenders++ {
		if options.Context.Err() != nil {
			break
		}
		step := runWaveJobs(buildWaveSweepStep(options, defenders), options, matchFrames, destroysDefenders, tick)
		all = append(all, step...)

		wins := 0
		detected := 0
		captured := 0
		ran := 0
		for _, item := range step {
			if !item.ran {
				continue
			}
			ran++
			if item.output.Outcome == OutcomeWin {
				wins++
			}
			if item.output.DetectionTime >= 0.0 {
				detected++
			}
			if item.output.CaptureTime >= 0.0 {
				captured++
			}
		}
		if ran < 1 {
			break
		}
		successRate := round1(100.0 * float64(wins) / float64(ran))
		points = append(points, SweepPoint{
			N:             defenders,
			Trials:        ran,
			SuccessRate:   successRate,
			WinRate:       successRate,
			DetectionRate: round1(100.0 * float64(detected) / float64(ran)),
			CaptureRate:   round1(100.0 * float64(captured) / float64(ran)),
		})

		if wins == ran {
			clean++
			if clean >= WaveConsecutiveMax {
				break
			}
		} else {
			clean = 0
		}
	}
	return all, points
}

func fillWaveResults(report *Report, trials []waveResult) {
	outcomes := make([]string, 0, len(trials))
	detectionTimes := make([]float64, 0, len(trials))
	captureTimes := make([]float64, 0, len(trials))
	goalTimes := make([]float64, 0, len(trials))
	counts := OutcomeCounts{}
	destroyed := 0
	evaderTotal := 0
	trialDestroyed := make([]int, 0, len(trials))
	trialEvaders := make([]int, 0, len(trials))
	detectedFirst := make([]int, 0, len(trials))
	detectedSecond := make([]int, 0, len(trials))

	for _, item := range trials {
		if !item.ran {
			continue
		}
		if report.ViewDistance == 0 {
			report.ViewDistance = item.output.ViewDistance
			report.FovDegrees = item.output.FovDegrees
			report.Speed = item.output.Speed
			report.HullRadius = item.output.HullRadius
		}
		report.Runs = append(report.Runs, TrialRun{
			Trial:         len(report.Runs),
			Outcome:       item.output.Outcome,
			DetectionTime: item.output.DetectionTime,
			CaptureTime:   item.output.CaptureTime,
			GoalTime:      item.output.GoalTime,
		})
		outcomes = append(outcomes, item.output.Outcome)
		detectionTimes = append(detectionTimes, item.output.DetectionTime)
		captureTimes = append(captureTimes, item.output.CaptureTime)
		goalTimes = append(goalTimes, item.output.GoalTime)
		destroyed += item.output.Destroyed
		evaderTotal += item.output.EvaderCount * WavePhases
		trialDestroyed = append(trialDestroyed, item.output.Destroyed)
		trialEvaders = append(trialEvaders, item.output.EvaderCount*WavePhases)
		detectedFirst = append(detectedFirst, boolCount(item.output.PhaseDetected[0]))
		detectedSecond = append(detectedSecond, boolCount(item.output.PhaseDetected[WavePhaseSimultaneous]))
		switch item.output.Outcome {
		case OutcomeWin:
			counts.Captured++
		case OutcomeLose:
			counts.ReachedPlanet++
		default:
			counts.TimedOut++
		}
	}

	total := len(outcomes)
	if total < 1 {
		total = 1
	}

	report.Results = Results{
		Trials:              len(outcomes),
		SuccessRate:         waveDestroyRate(destroyed, evaderTotal),
		TrialsHeldRate:      round1(100.0 * float64(counts.Captured) / float64(total)),
		DetectionRate:       rate(detectionTimes),
		CaptureRate:         rate(captureTimes),
		OutcomeCounts:       counts,
		Outcomes:            outcomes,
		DetectionTimes:      detectionTimes,
		CaptureTimes:        captureTimes,
		GoalTimes:           goalTimes,
		SequentialRate:      wavePhaseRate(trials, 0),
		SimultaneousRate:    wavePhaseRate(trials, WavePhaseSimultaneous),
		EvadersDestroyed:    destroyed,
		EvadersTotal:        evaderTotal,
		EvaderDestroyedRate: waveDestroyRate(destroyed, evaderTotal),
		TrialDestroyed:      trialDestroyed,
		TrialEvaders:        trialEvaders,
		TrialDetectedFirst:  detectedFirst,
		TrialDetectedSecond: detectedSecond,
		SequentialDetection: shareOf(detectedFirst),
		SimultaneousDetect:  shareOf(detectedSecond),
	}
}

func wavePhaseRate(results []waveResult, phase int) float64 {
	ran := 0
	held := 0
	for _, item := range results {
		if !item.ran {
			continue
		}
		ran++
		if item.output.PhaseHeld[phase] {
			held++
		}
	}
	if ran < 1 {
		return 0.0
	}
	return round1(100.0 * float64(held) / float64(ran))
}

func waveDestroyRate(destroyed int, total int) float64 {
	if total < 1 {
		return 0.0
	}
	return round1(100.0 * float64(destroyed) / float64(total))
}

func packWaveRuns(trials []waveResult) []ReplayRun {
	runs := make([]ReplayRun, 0, len(trials))
	index := 0
	for _, item := range trials {
		if !item.ran {
			continue
		}
		run := ReplayRun{
			Trial:         index,
			Outcome:       item.output.Outcome,
			DetectionTime: item.output.DetectionTime,
			CaptureTime:   item.output.CaptureTime,
			GoalTime:      item.output.GoalTime,
			Stats: map[string]float64{
				"evaders":         float64(item.output.EvaderCount),
				"destroyed":       float64(item.output.Destroyed),
				"breaches":        float64(item.output.Breaches),
				"defenders":       float64(len(item.job.placements)),
				"lost":            float64(item.output.DefendersLost),
				"held_first":      boolStat(item.output.PhaseHeld[0]),
				"held_second":     boolStat(item.output.PhaseHeld[WavePhaseSimultaneous]),
				"detected_first":  boolStat(item.output.PhaseDetected[0]),
				"detected_second": boolStat(item.output.PhaseDetected[WavePhaseSimultaneous]),
				"wave_two_time":   item.output.WaveTwoTime,
			},
		}
		if len(item.output.Frames) > 0 {
			run.FramesPacked = PackFrames(item.output.Frames)
		}
		runs = append(runs, run)
		index++
	}
	return runs
}

func packWaveSweepRuns(sweep []waveResult) []ReplaySweepRun {
	byDefenders := map[int][]waveResult{}
	order := []int{}
	for _, item := range sweep {
		if !item.ran {
			continue
		}
		if _, present := byDefenders[item.job.defenders]; !present {
			order = append(order, item.job.defenders)
		}
		byDefenders[item.job.defenders] = append(byDefenders[item.job.defenders], item)
	}
	sort.Ints(order)

	runs := make([]ReplaySweepRun, 0, len(order))
	for _, defenders := range order {
		items := byDefenders[defenders]
		detected := 0
		captured := 0
		for _, item := range items {
			if item.output.DetectionTime >= 0.0 {
				detected++
			}
			if item.output.CaptureTime >= 0.0 {
				captured++
			}
		}
		total := len(items)
		if total < 1 {
			total = 1
		}
		run := ReplaySweepRun{
			N:             defenders,
			Defenders:     defenders,
			Outcome:       items[0].output.Outcome,
			DetectionTime: items[0].output.DetectionTime,
			CaptureTime:   items[0].output.CaptureTime,
			GoalTime:      items[0].output.GoalTime,
			DetectionRate: round1(100.0 * float64(detected) / float64(total)),
			CaptureRate:   round1(100.0 * float64(captured) / float64(total)),
			TrialRuns:     []ReplayRun{},
		}
		if len(items[0].output.Frames) > 0 {
			run.FramesPacked = PackFrames(items[0].output.Frames)
		}
		for _, item := range items {
			if len(item.output.Frames) == 0 {
				continue
			}
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

func boolCount(value bool) int {
	if value {
		return 1
	}
	return 0
}

func shareOf(flags []int) float64 {
	if len(flags) == 0 {
		return 0.0
	}
	hits := 0
	for _, flag := range flags {
		hits += flag
	}
	return round1(100.0 * float64(hits) / float64(len(flags)))
}

func boolStat(value bool) float64 {
	if value {
		return 1.0
	}
	return 0.0
}
