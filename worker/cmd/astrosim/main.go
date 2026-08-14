package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"astroswarm/worker/internal/bench"
	"astroswarm/worker/internal/charts"
	"astroswarm/worker/internal/entry"
	"astroswarm/worker/internal/godot"
	"astroswarm/worker/internal/verify"
)

type CommandOptions struct {
	Target          string
	Server          string
	EntryFile       string
	Output          string
	Trials          int
	Seed            int64
	SweepMax        int
	SweepTrials     int
	MatchSeconds    float64
	GoalTailSeconds float64
	EnemyX          float64
	EnemyY          float64
	SweepSpawn      string
	Collisions      bool
	SinglePrecision bool
	Workers         int
	Tolerance       float64
	SkipCharts      bool
	SkipSettings    bool
	Quiet           bool
}

func main() {
	options := CommandOptions{}
	flags := flag.NewFlagSet("astrosim", flag.ExitOnError)
	flags.StringVar(&options.Server, "server", DefaultServer, "base url of the AstroSwarm web server")
	flags.StringVar(&options.EntryFile, "file", "", "read the entry from a local json file instead of the server")
	flags.StringVar(&options.Output, "out", "", "directory for results.json and the charts (default out/<entry id>)")
	flags.IntVar(&options.Trials, "trials", 0, "placement trials to simulate (default: the entry's own trial count)")
	flags.Int64Var(&options.Seed, "seed", bench.DefaultSeed, "evaluation seed the server used")
	flags.IntVar(&options.SweepMax, "n-max", bench.DefaultSweepMax, "largest defender count in the ring sweep")
	flags.IntVar(&options.SweepTrials, "sweep-trials", bench.DefaultSweepTrials, "trials per ring-sweep defender count")
	flags.Float64Var(&options.MatchSeconds, "match-seconds", bench.DefaultMatchSeconds, "match time cap in seconds")
	flags.Float64Var(&options.GoalTailSeconds, "goal-tail-seconds", bench.DefaultGoalTailSeconds, "extra seconds simulated after the evader reaches the planet")
	flags.Float64Var(&options.EnemyX, "enemy-x", 1920, "ring-sweep evader spawn x")
	flags.Float64Var(&options.EnemyY, "enemy-y", 40, "ring-sweep evader spawn y")
	flags.StringVar(&options.SweepSpawn, "sweep-spawn", bench.SweepSpawnVaried, "ring-sweep evader spawn: varied stratifies the approach angle per trial, fixed pins it at the enemy start (the server setting wins unless this is set)")
	flags.BoolVar(&options.Collisions, "collisions", false, "enable ship and planet collisions")
	flags.BoolVar(&options.SinglePrecision, "single-precision", false, "round positions and headings to float32 the way Godot stores them")
	flags.IntVar(&options.Workers, "workers", runtime.NumCPU(), "parallel matches")
	flags.Float64Var(&options.Tolerance, "tolerance", 1.0, "percentage points a published rate may differ by before it is flagged")
	flags.BoolVar(&options.SkipCharts, "no-charts", false, "skip writing the png charts")
	flags.BoolVar(&options.SkipSettings, "no-settings", false, "do not read the seed and sweep settings from the server")
	flags.BoolVar(&options.Quiet, "quiet", false, "suppress the progress line")
	flags.Usage = usage(flags)

	if err := flags.Parse(os.Args[1:]); err != nil {
		os.Exit(2)
	}
	if flags.NArg() > 0 {
		options.Target = flags.Arg(0)
	}
	if options.Target == "" && options.EntryFile == "" {
		flags.Usage()
		os.Exit(2)
	}

	explicit := map[string]bool{}
	flags.Visit(func(set *flag.Flag) { explicit[set.Name] = true })

	if err := run(options, explicit); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func usage(flags *flag.FlagSet) func() {
	return func() {
		fmt.Fprintf(os.Stderr, "astrosim re-simulates a published FARP entry and checks the numbers on the site.\n\n")
		fmt.Fprintf(os.Stderr, "usage:\n")
		fmt.Fprintf(os.Stderr, "  astrosim <entry-id|entry-url> [flags]\n")
		fmt.Fprintf(os.Stderr, "  astrosim -file entry.json [flags]\n\n")
		fmt.Fprintf(os.Stderr, "examples:\n")
		fmt.Fprintf(os.Stderr, "  astrosim 13569541-180c-4c51-bcfe-d4ab038359af\n")
		fmt.Fprintf(os.Stderr, "  astrosim %s/levels/13569541-180c-4c51-bcfe-d4ab038359af\n\n", DefaultServer)
		fmt.Fprintf(os.Stderr, "flags:\n")
		flags.PrintDefaults()
	}
}

func run(options CommandOptions, explicit map[string]bool) error {
	server := resolveServer(options)
	published, err := loadEntry(options, server)
	if err != nil {
		return err
	}
	if !options.SkipSettings {
		applyServerSettings(&options, server, explicit)
	}

	algorithm := published.Scripts()
	if len(algorithm) == 0 {
		return fmt.Errorf("entry carries no algorithm blocks")
	}

	levelID := published.LevelID
	if levelID == "" {
		levelID = "farp1"
	}
	if bench.IsPilotLevel(levelID) {
		return fmt.Errorf("level %d entries are piloted recordings, there is nothing to re-simulate", bench.LevelNumber(levelID))
	}

	placements := published.BenchPlacements()
	if len(placements) == 0 && bench.LevelNumber(levelID) != 2 && !bench.IsWaveLevel(levelID) {
		return fmt.Errorf("entry carries no defender placements")
	}

	trials := options.Trials
	if trials <= 0 {
		trials = published.Trials
	}
	if trials <= 0 {
		trials = bench.DefaultTrials
	}

	output := options.Output
	if output == "" {
		name := published.ID
		if name == "" {
			name = "entry"
		}
		output = filepath.Join("out", name)
	}
	if err := os.MkdirAll(output, 0o755); err != nil {
		return err
	}

	if options.SweepSpawn != bench.SweepSpawnFixed && options.SweepSpawn != bench.SweepSpawnVaried {
		return fmt.Errorf("-sweep-spawn must be %q or %q", bench.SweepSpawnVaried, bench.SweepSpawnFixed)
	}
	variedSweepSpawn := options.SweepSpawn == bench.SweepSpawnVaried

	benchOptions := bench.Options{
		LevelID:          levelID,
		Algorithm:        algorithm,
		Placements:       placements,
		Trials:           trials,
		TrialStart:       0,
		TrialCount:       trials,
		Seed:             options.Seed,
		SweepMax:         options.SweepMax,
		SweepTrials:      options.SweepTrials,
		NStart:           1,
		NCount:           options.SweepMax,
		MatchSeconds:     options.MatchSeconds,
		GoalTailSeconds:  options.GoalTailSeconds,
		EnemyStart:       godot.Vec{X: options.EnemyX, Y: options.EnemyY},
		HasEnemyStart:    true,
		Collisions:       options.Collisions || published.Collisions,
		SinglePrecision:  options.SinglePrecision,
		VariedSweepSpawn: variedSweepSpawn,
		Workers:          options.Workers,
	}
	if !options.Quiet {
		benchOptions.Progress = progressPrinter(bench.IsWaveLevel(levelID), trials)
	}

	fmt.Printf("simulating %s\n", published.Label())
	if bench.IsWaveLevel(levelID) {
		fmt.Printf("  level %s, %d defenders, %d wave trials of up to %d evaders, defender sweep from n=1 up to %d x %d trials\n",
			levelID, len(benchOptions.Placements), trials, bench.WaveMaxEvaders, options.SweepMax, options.SweepTrials)
		fmt.Printf("  each trial runs both waves: one after another, then all at once against the same layout\n")
	} else {
		fmt.Printf("  level %s, %d defenders, %d placement trials, ring sweep n=1..%d x %d trials\n",
			levelID, len(benchOptions.Placements), trials, options.SweepMax, options.SweepTrials)
	}
	fmt.Printf("  seed %d, sweep spawn %s, %d workers\n", options.Seed, options.SweepSpawn, benchOptions.Workers)

	report := bench.Run(benchOptions)
	if !options.Quiet {
		fmt.Println()
	}

	checked := verify.Compare(published, report, options.Tolerance)
	printSummary(report, checked, bench.IsWaveLevel(levelID))

	payload := map[string]any{
		"entry": map[string]any{
			"id":       published.ID,
			"username": published.Username,
			"level_id": published.LevelID,
			"status":   published.Status,
		},
		"simulated":    report,
		"verification": checked,
	}
	resultsPath := filepath.Join(output, "results.json")
	if err := writeJSON(resultsPath, payload); err != nil {
		return err
	}
	fmt.Printf("\nwrote %s\n", resultsPath)

	if options.SkipCharts {
		return nil
	}
	written, err := charts.WriteAll(output, charts.Input{
		Report:    report,
		Published: published,
		Subtitle:  chartSubtitle(published),
	})
	for _, path := range written {
		fmt.Printf("wrote %s\n", path)
	}
	return err
}

func chartSubtitle(published *entry.Entry) string {
	parts := []string{}
	if published.Username != "" {
		parts = append(parts, published.Username)
	}
	if published.LevelID != "" {
		parts = append(parts, published.LevelID)
	}
	return strings.Join(parts, " ")
}

func progressPrinter(wave bool, trials int) func(done, total int) {
	var last int64
	return func(done, total int) {
		now := time.Now().UnixMilli()
		previous := atomic.LoadInt64(&last)
		if done < total && now-previous < 200 {
			return
		}
		if !atomic.CompareAndSwapInt64(&last, previous, now) {
			return
		}
		if wave {
			if done <= trials {
				fmt.Printf("\r  wave trial %d/%d          ", done, trials)
			} else {
				fmt.Printf("\r  defender sweep: %d runs   ", done-trials)
			}
			return
		}
		fmt.Printf("\r  %d/%d matches", done, total)
	}
}

func printSummary(report bench.Report, checked verify.Result, wave bool) {
	results := report.Results
	counts := results.OutcomeCounts

	label := "placement trials"
	if wave {
		label = "wave trials"
	}
	fmt.Printf("\n%s: %d in %.1fs (%d matches total)\n\n",
		label, results.Trials, report.DurationSeconds, report.MatchesRun)

	if wave {
		fmt.Printf("  %-7s Trials held — both waves stopped with nothing through\n", formatPercent(results.SuccessRate))
		breakdown := fmt.Sprintf("%d held, %d let an evader through", counts.Captured, counts.ReachedPlanet)
		if counts.TimedOut > 0 {
			breakdown += fmt.Sprintf(", %d timeouts", counts.TimedOut)
		}
		fmt.Printf("  %-7s %s\n", "", breakdown)
		fmt.Printf("  %-7s First wave held — the evaders one after another\n", formatPercent(results.SequentialRate))
		fmt.Printf("  %-7s Second wave held — the evaders all at once\n", formatPercent(results.SimultaneousRate))
		fmt.Printf("  %-7s Evaders destroyed (%d of %d across both waves)\n",
			formatPercent(results.EvaderDestroyedRate), results.EvadersDestroyed, results.EvadersTotal)
		fmt.Printf("  %-7s Detection rate — a defender saw an evader\n", formatPercent(results.DetectionRate))
		fmt.Printf("  %-7s Capture rate — a defender touched an evader\n", formatPercent(results.CaptureRate))
		fmt.Printf("  %-7d Defenders placed\n", report.Defenders)
		if len(results.Sweep) > 0 {
			last := results.Sweep[len(results.Sweep)-1]
			fmt.Printf("  %-7d Defenders the sweep reached before it stopped\n", last.N)
		}
		return
	}

	fmt.Printf("  %-7s Success rate\n", formatPercent(results.SuccessRate))
	breakdown := fmt.Sprintf("%d captured, %d reached the planet", counts.Captured, counts.ReachedPlanet)
	if counts.TimedOut > 0 {
		breakdown += fmt.Sprintf(", %d timeouts", counts.TimedOut)
	}
	fmt.Printf("  %-7s %s\n", "", breakdown)
	fmt.Printf("  %-7s Detection rate — a defender saw the evader\n", formatPercent(results.DetectionRate))
	fmt.Printf("  %-7s Capture rate — a defender touched the evader\n", formatPercent(results.CaptureRate))
	fmt.Printf("  %-7s Mean time for the evader to reach the planet\n", formatGoalTime(results.MeanGoalSeconds, results.GoalSamples))
	fmt.Printf("  %-7d Defenders placed\n", report.Defenders)

	if len(checked.Comparisons) > 0 {
		fmt.Printf("\nagainst the published entry (tolerance %.1f points):\n", checked.Tolerance)
		for _, comparison := range checked.Comparisons {
			marker := "ok"
			if !comparison.WithinTol {
				marker = "MISMATCH"
			}
			fmt.Printf("  %-15s published %6.1f  simulated %6.1f  delta %+6.1f  %s\n",
				comparison.Metric, comparison.Published, comparison.Simulated, comparison.Delta, marker)
		}
	}
	for _, note := range checked.Notes {
		fmt.Printf("  note: %s\n", note)
	}
	if len(checked.Comparisons) > 0 {
		if checked.Passed {
			fmt.Println("\nthe published numbers reproduce")
		} else {
			fmt.Println("\nthe published numbers do not reproduce")
		}
	}
}

func formatPercent(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64) + "%"
}

func formatGoalTime(seconds float64, samples int) string {
	if samples == 0 {
		return "n/a"
	}
	return strconv.FormatFloat(seconds, 'f', 1, 64) + "s"
}

func writeJSON(path string, payload any) error {
	encoded, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, encoded, 0o644)
}
