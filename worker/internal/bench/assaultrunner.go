package bench

import (
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

const (
	AssaultTrialSeedOffset = 1100000 // rng seed offset for the placement trials of an assault job
	AssaultSweepSeedOffset = 3300000 // rng seed offset for the ring sweep of an assault job
	AssaultReplayTrials    = 25      // count, placement trials that keep a frame recording
	AssaultReplaySweepNMax = 20      // count, largest defender count that still records a replay
	AssaultReplaySweepMax  = 4       // count, trials per defender count that still record a replay
)

type assaultJob struct {
	sweep      bool
	trial      int
	defenders  int
	seed       int64
	placements []Placement
	record     bool
}

type assaultResult struct {
	job    assaultJob
	output AssaultOutput
	ran    bool
}

func AssaultMode(levelID string) string {
	if LevelNumber(levelID) == 5 {
		return AssaultModeSiege
	}
	return AssaultModeWaves
}

func assaultTrialLayout(seed int64, trial int, defenders int) []Placement {
	return ScatterLayout(seed+int64(trial)*SweepSeedStride, defenders)
}

func buildAssaultTrials(options Options) []assaultJob {
	defenders := len(options.Placements)
	if defenders < 1 {
		defenders = RingCount
	}
	jobs := make([]assaultJob, 0, options.TrialCount)
	for index := 0; index < options.TrialCount; index++ {
		trial := options.TrialStart + index
		jobs = append(jobs, assaultJob{
			trial:      trial,
			defenders:  defenders,
			seed:       options.Seed + AssaultTrialSeedOffset + int64(trial),
			placements: assaultTrialLayout(options.Seed+AssaultTrialSeedOffset, trial, defenders),
			record:     options.Record && trial < AssaultReplayTrials,
		})
	}
	return jobs
}

func buildAssaultSweepStep(options Options, defenders int) []assaultJob {
	jobs := make([]assaultJob, 0, options.SweepTrials)
	for trial := 0; trial < options.SweepTrials; trial++ {
		jobs = append(jobs, assaultJob{
			sweep:      true,
			trial:      trial,
			defenders:  defenders,
			seed:       options.Seed + AssaultSweepSeedOffset + int64(defenders)*SweepSeedStride + int64(trial),
			placements: RingPlacements(options.Seed+AssaultSweepSeedOffset, trial, options.SweepTrials, defenders),
			record:     options.Record && (trial == 0 || (defenders <= AssaultReplaySweepNMax && trial < AssaultReplaySweepMax)),
		})
	}
	return jobs
}

func runAssaultJobs(jobs []assaultJob, options Options, matchFrames int, mode string, destroysDefenders bool, progress func()) []assaultResult {
	results := make([]assaultResult, len(jobs))
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
				output := RunAssaultMatch(AssaultInput{
					Algorithm:         options.Algorithm,
					Placements:        current.placements,
					Mode:              mode,
					DestroysDefenders: destroysDefenders,
					Seed:              current.seed,
					MatchFrames:       matchFrames,
					SinglePrecision:   options.SinglePrecision,
					Record:            current.record,
				})
				results[index] = assaultResult{job: current, output: output, ran: true}
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

func RunAssaultJob(options Options) (Report, JobResult) {
	options.applyDefaults()
	started := time.Now()

	mode := AssaultMode(options.LevelID)
	destroysDefenders := IsAttritionLevel(options.LevelID)
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

	trials := runAssaultJobs(buildAssaultTrials(options), options, matchFrames, mode, destroysDefenders, tick)
	sweep, sweepPoints, sweepAttrition := runAssaultSweep(options, matchFrames, mode, destroysDefenders, tick)

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

	fillAssaultResults(&report, trials)
	report.Results.Sweep = sweepPoints
	report.Results.SweepAttrition = sweepAttrition
	report.DurationSeconds = time.Since(started).Seconds()

	if options.Progress != nil {
		options.Progress(int(estimate), int(estimate))
	}

	jobResult := JobResult{
		Runs:      packAssaultRuns(trials),
		SweepRuns: packAssaultSweepRuns(sweep),
		Meta: JobMeta{
			FPS:       PhysicsTicksPerSecond / AssaultRecordStride,
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

// The sweep grows the ring until the algorithm holds cleanly a few sizes in a
// row, so a strong entry is not charged for defender counts that add nothing.
func runAssaultSweep(options Options, matchFrames int, mode string, destroysDefenders bool, tick func()) ([]assaultResult, []SweepPoint, []AttritionSeries) {
	all := []assaultResult{}
	points := []SweepPoint{}
	series := []AttritionSeries{}
	clean := 0

	for defenders := options.NStart; defenders <= options.SweepMax; defenders++ {
		if options.Context.Err() != nil {
			break
		}
		step := runAssaultJobs(buildAssaultSweepStep(options, defenders), options, matchFrames, mode, destroysDefenders, tick)
		all = append(all, step...)

		wins := 0
		detected := 0
		ran := 0
		resolved := 0
		destroyed := 0
		samples := []AttritionSample{}
		for _, item := range step {
			if !item.ran {
				continue
			}
			ran++
			resolved += item.output.Resolved
			destroyed += item.output.Destroyed
			samples = append(samples, item.output.Samples...)
			if item.output.Outcome == OutcomeWin {
				wins++
			}
			if item.output.DetectionTime >= 0.0 {
				detected++
			}
		}
		if ran < 1 {
			break
		}
		captureRate := destroyRate(destroyed, resolved)
		points = append(points, SweepPoint{
			N:             defenders,
			Trials:        ran,
			SuccessRate:   captureRate,
			CaptureRate:   captureRate,
			Risk:          riskOf(captureRate),
			WinRate:       round1(100.0 * float64(wins) / float64(ran)),
			DetectionRate: round1(100.0 * float64(detected) / float64(ran)),
		})

		// A ring that never lost a ship has a single rung and no curve to draw,
		// which is every ring on a level without attrition.
		if curve := SummarizeAttrition(samples); len(curve) > 1 {
			series = append(series, AttritionSeries{N: defenders, Points: curve})
		}

		if wins == ran {
			clean++
			if clean >= WaveConsecutiveMax {
				break
			}
		} else {
			clean = 0
		}
	}
	return all, points, series
}

func fillAssaultResults(report *Report, trials []assaultResult) {
	outcomes := make([]string, 0, len(trials))
	detectionTimes := make([]float64, 0, len(trials))
	captureTimes := make([]float64, 0, len(trials))
	goalTimes := make([]float64, 0, len(trials))
	trialDestroyed := make([]int, 0, len(trials))
	trialResolved := make([]int, 0, len(trials))
	trialBreaches := make([]int, 0, len(trials))
	trialLost := make([]int, 0, len(trials))
	counts := OutcomeCounts{}
	resolved := 0
	destroyed := 0
	breaches := 0
	lost := 0
	recorded := 0
	samples := []AttritionSample{}

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
		trialDestroyed = append(trialDestroyed, item.output.Destroyed)
		trialResolved = append(trialResolved, item.output.Resolved)
		trialBreaches = append(trialBreaches, item.output.Breaches)
		trialLost = append(trialLost, item.output.DefendersLost)
		resolved += item.output.Resolved
		destroyed += item.output.Destroyed
		breaches += item.output.Breaches
		lost += item.output.DefendersLost
		samples = append(samples, item.output.Samples...)
		if len(item.output.Frames) > 0 {
			recorded++
		}
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
	captureRate := destroyRate(destroyed, resolved)

	report.Results = Results{
		Trials:              len(outcomes),
		SuccessRate:         captureRate,
		Risk:                riskOf(captureRate),
		TrialsHeldRate:      round1(100.0 * float64(counts.Captured) / float64(total)),
		DetectionRate:       rate(detectionTimes),
		CaptureRate:         captureRate,
		OutcomeCounts:       counts,
		Outcomes:            outcomes,
		DetectionTimes:      detectionTimes,
		CaptureTimes:        captureTimes,
		GoalTimes:           goalTimes,
		EvadersResolved:     resolved,
		EvadersDestroyed:    destroyed,
		EvaderDestroyedRate: captureRate,
		Breaches:            breaches,
		DefendersLost:       lost,
		ReplayTrials:        recorded,
		TrialDestroyed:      trialDestroyed,
		TrialResolved:       trialResolved,
		TrialBreaches:       trialBreaches,
		TrialLost:           trialLost,
		Attrition:           SummarizeAttrition(samples),
	}
}

// Pools every launched evader by the size of the line that was standing when it
// left the ring, so the curve reads as the risk a thinning defense carries.
func SummarizeAttrition(samples []AttritionSample) []AttritionPoint {
	buckets := map[int]*AttritionPoint{}
	order := []int{}
	for _, sample := range samples {
		bucket, present := buckets[sample.Defenders]
		if !present {
			bucket = &AttritionPoint{Defenders: sample.Defenders}
			buckets[sample.Defenders] = bucket
			order = append(order, sample.Defenders)
		}
		bucket.Launched++
		if sample.Destroyed {
			bucket.Destroyed++
		}
	}
	sort.Ints(order)
	points := make([]AttritionPoint, 0, len(order))
	for _, defenders := range order {
		bucket := buckets[defenders]
		bucket.CaptureRate = destroyRate(bucket.Destroyed, bucket.Launched)
		bucket.Risk = riskOf(bucket.CaptureRate)
		points = append(points, *bucket)
	}
	return points
}

func destroyRate(destroyed int, launched int) float64 {
	if launched < 1 {
		return 0.0
	}
	return round1(100.0 * float64(destroyed) / float64(launched))
}

func riskOf(captureRate float64) float64 {
	return round1(100.0 - captureRate)
}

func packAssaultRuns(trials []assaultResult) []ReplayRun {
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
			Stats:         assaultStats(item.output),
		}
		if len(item.output.Frames) > 0 {
			run.FramesPacked = PackFrames(item.output.Frames)
		}
		runs = append(runs, run)
		index++
	}
	return runs
}

func assaultStats(output AssaultOutput) map[string]float64 {
	return map[string]float64{
		"sent":      float64(output.Launched),
		"resolved":  float64(output.Resolved),
		"destroyed": float64(output.Destroyed),
		"breaches":  float64(output.Breaches),
		"defenders": float64(output.Defenders),
		"lost":      float64(output.DefendersLost),
		"end_time":  output.EndTime,
	}
}

func packAssaultSweepRuns(sweep []assaultResult) []ReplaySweepRun {
	byDefenders := map[int][]assaultResult{}
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
		resolved := 0
		destroyed := 0
		for _, item := range items {
			if item.output.DetectionTime >= 0.0 {
				detected++
			}
			resolved += item.output.Resolved
			destroyed += item.output.Destroyed
		}
		total := len(items)
		if total < 1 {
			total = 1
		}
		captureRate := destroyRate(destroyed, resolved)
		run := ReplaySweepRun{
			N:             defenders,
			Defenders:     defenders,
			Outcome:       items[0].output.Outcome,
			DetectionTime: items[0].output.DetectionTime,
			CaptureTime:   items[0].output.CaptureTime,
			GoalTime:      items[0].output.GoalTime,
			DetectionRate: round1(100.0 * float64(detected) / float64(total)),
			CaptureRate:   captureRate,
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
				Stats:         assaultStats(item.output),
			})
		}
		runs = append(runs, run)
	}
	return runs
}
