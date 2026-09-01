package main

import (
	"fmt"
	"net/url"
	"strings"

	"astroswarm/worker/internal/bench"
	"astroswarm/worker/internal/entry"
)

const DefaultServer = "https://astroswarm.autonomousrobotics.club"

func splitTarget(target string, fallbackServer string) (server string, id string, err error) {
	trimmed := strings.TrimSpace(target)
	if trimmed == "" {
		return "", "", fmt.Errorf("no entry id given")
	}

	if !strings.Contains(trimmed, "://") {
		return strings.TrimRight(fallbackServer, "/"), trimmed, nil
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", "", fmt.Errorf("cannot read %q as a url: %w", trimmed, err)
	}
	segments := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	last := segments[len(segments)-1]
	if last == "" {
		return "", "", fmt.Errorf("no entry id in %q", trimmed)
	}
	return fmt.Sprintf("%s://%s", parsed.Scheme, parsed.Host), last, nil
}

func resolveServer(options CommandOptions) string {
	if options.Target != "" {
		if server, _, err := splitTarget(options.Target, options.Server); err == nil {
			return server
		}
	}
	return strings.TrimRight(options.Server, "/")
}

func loadEntry(options CommandOptions, server string) (*entry.Entry, error) {
	if options.EntryFile != "" {
		return entry.Load(options.EntryFile)
	}

	_, id, err := splitTarget(options.Target, server)
	if err != nil {
		return nil, err
	}
	fmt.Printf("fetching entry %s from %s\n", id, server)
	fetched, err := entry.Fetch(server, id)
	if err != nil {
		return nil, err
	}

	index, err := entry.FetchSweepIndex(server, id)
	if err != nil {
		fmt.Printf("warning: could not read the published ring sweep (%v), charts will show simulated rates only\n", err)
		return fetched, nil
	}
	fetched.SweepIndex = index
	return fetched, nil
}

// The assault levels carry their own sweep budget on the server, so an entry on
// one of them has to adopt that budget rather than the level 1 and 2 one, or the
// numbers and the charts will not be the ones the site published.
func applyServerSettings(options *CommandOptions, server string, explicit map[string]bool, levelID string) {
	settings, err := entry.FetchSettings(server)
	if err != nil {
		fmt.Printf("warning: could not read %s/api/evaluations/settings (%v)\n", server, err)
		fmt.Printf("warning: falling back to built-in defaults, so seed %d may not be the one this entry was graded with\n", options.Seed)
		return
	}

	adopted := []string{}
	adopt := func(name string, apply func()) {
		if explicit[name] {
			return
		}
		apply()
		adopted = append(adopted, name)
	}

	if settings.Seed != 0 {
		adopt("seed", func() { options.Seed = settings.Seed })
	}
	sweepMax := settings.SweepMax
	sweepTrials := settings.SweepTrials
	sweepSource := "server"
	if bench.IsAssaultLevel(levelID) {
		sweepMax = bench.DefaultAssaultSweepMax
		sweepTrials = bench.DefaultAssaultSweepTrials
		sweepSource = "built-in assault budget"
		if settings.AssaultSweepMax > 0 && settings.AssaultSweepTrials > 0 {
			sweepMax = settings.AssaultSweepMax
			sweepTrials = settings.AssaultSweepTrials
			sweepSource = "server assault budget"
		}
	}
	if sweepMax > 0 {
		adopt("n-max", func() { options.SweepMax = sweepMax })
	}
	if sweepTrials > 0 {
		adopt("sweep-trials", func() { options.SweepTrials = sweepTrials })
	}
	if settings.SweepSpawn != "" {
		adopt("sweep-spawn", func() { options.SweepSpawn = settings.SweepSpawn })
	}
	if settings.MatchCapSeconds > 0 {
		adopt("match-seconds", func() { options.MatchSeconds = settings.MatchCapSeconds })
	}
	if settings.GoalTailSeconds >= 0 {
		adopt("goal-tail-seconds", func() { options.GoalTailSeconds = settings.GoalTailSeconds })
	}
	adopt("enemy-x", func() { options.EnemyX = settings.EnemyStartX })
	adopt("enemy-y", func() { options.EnemyY = settings.EnemyStartY })

	fmt.Printf("server settings: seed %d, sweep %d x %d trials from the %s (%s spawn), match cap %.0fs, evader spawn (%.0f, %.0f)\n",
		settings.Seed, sweepMax, sweepTrials, sweepSource, settings.SweepSpawn,
		settings.MatchCapSeconds, settings.EnemyStartX, settings.EnemyStartY)
	if sweepSource == "built-in assault budget" {
		fmt.Printf("warning: %s does not report an assault sweep budget, so its own sweep may not have been %d x %d\n",
			server, sweepMax, sweepTrials)
	}
	if len(adopted) > 0 {
		fmt.Printf("  adopted for this run: %s\n", strings.Join(adopted, ", "))
	}

	if settings.ArenaWidth > 0 && settings.ArenaWidth != bench.ArenaWidth {
		fmt.Printf("warning: server arena width %.0f differs from the simulated %.0f\n", settings.ArenaWidth, float64(bench.ArenaWidth))
	}
	if settings.ArenaHeight > 0 && settings.ArenaHeight != bench.ArenaHeight {
		fmt.Printf("warning: server arena height %.0f differs from the simulated %.0f\n", settings.ArenaHeight, float64(bench.ArenaHeight))
	}
}
