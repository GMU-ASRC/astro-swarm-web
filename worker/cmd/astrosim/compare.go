package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"astroswarm/worker/internal/bench"
	"astroswarm/worker/internal/charts"
)

const (
	compareChartName = "risk_fit.png"
	compareDataName  = "risk_fit.json"
)

// A target is a sim id, an entry url, an entry json, or the results.json an
// earlier run wrote. Reading a results.json back is what makes a second look at a
// figure free: the sweep it already measured is on disk, so nothing has to be
// simulated again. A target may be written "Label=target" to name its line.
type target struct {
	label string
	value string
}

func splitTargets(list string) []target {
	targets := []target{}
	for _, piece := range strings.Split(list, ",") {
		trimmed := strings.TrimSpace(piece)
		if trimmed == "" {
			continue
		}
		label := ""
		if name, rest, found := strings.Cut(trimmed, "="); found {
			label = strings.TrimSpace(name)
			trimmed = strings.TrimSpace(rest)
		}
		targets = append(targets, target{label: label, value: trimmed})
	}
	return targets
}

type storedRun struct {
	Entry struct {
		ID       string `json:"id"`
		Username string `json:"username"`
		LevelID  string `json:"level_id"`
	} `json:"entry"`
	Simulated bench.Report `json:"simulated"`
}

func loadStoredRun(path string) (*storedRun, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	stored := &storedRun{}
	if err := json.Unmarshal(raw, stored); err != nil {
		return nil, err
	}
	if len(stored.Simulated.Results.Sweep) == 0 {
		return nil, fmt.Errorf("%s carries no ring sweep", path)
	}
	return stored, nil
}

func compareCurve(label string, sweep []bench.SweepPoint) charts.FitCurve {
	points := make([]charts.FitPoint, 0, len(sweep))
	for _, point := range sweep {
		points = append(points, charts.FitPoint{N: point.N, Risk: point.Risk})
	}
	return charts.FitCurve{Label: label, Points: points}
}

func curveLabel(username string, levelID string, rate float64) string {
	name := strings.TrimSpace(username)
	if name == "" {
		name = "entry"
	}
	return fmt.Sprintf("%s %s - %.1f%%", name, levelID, rate)
}

func runComparison(options CommandOptions, explicit map[string]bool, targets []target) error {
	server := resolveServer(options)
	curves := make([]charts.FitCurve, 0, len(targets))
	levels := map[string]bool{}

	for _, item := range targets {
		curve, levelID, err := comparisonCurve(options, explicit, server, item)
		if err != nil {
			return err
		}
		curves = append(curves, curve)
		levels[levelID] = true
	}

	if len(levels) > 1 {
		fmt.Printf("warning: the entries are not all on one level, so their risks are not measured under the same rules\n")
	}

	output := options.Output
	if output == "" {
		output = filepath.Join("out", "compare")
	}
	if err := os.MkdirAll(output, 0o755); err != nil {
		return err
	}

	subtitle := ""
	for level := range levels {
		subtitle = level
	}
	if len(levels) > 1 {
		subtitle = ""
	}

	path, fits, err := charts.WriteRiskFit(output, curves, subtitle, compareChartName)
	if err != nil {
		return err
	}

	dataPath := filepath.Join(output, compareDataName)
	if err := writeJSON(dataPath, fitPayload(curves, fits, subtitle)); err != nil {
		return err
	}

	fmt.Println()
	for index, curve := range curves {
		fit := fits[index]
		if fit.Samples < 2 {
			fmt.Printf("  %-34s no fit: fewer than two ring sizes carried any risk\n", curve.Label)
			continue
		}
		fmt.Printf("  %-34s %s   lambda %.4f per defender, R2 %.3f over %d ring sizes\n",
			curve.Label, fit, fit.Lambda, fit.RSquared, fit.Samples)
	}
	fmt.Printf("\nwrote %s\nwrote %s\n", path, dataPath)
	return nil
}

// The measured points and the fit beside them, so the figure can be redrawn, or
// its numbers adjusted, without simulating anything again.
func fitPayload(curves []charts.FitCurve, fits []charts.Fit, levelID string) map[string]any {
	entries := make([]map[string]any, 0, len(curves))
	for index, curve := range curves {
		fit := fits[index]
		points := make([]map[string]any, 0, len(curve.Points))
		for _, point := range curve.Points {
			points = append(points, map[string]any{"n": point.N, "risk": point.Risk})
		}
		entries = append(entries, map[string]any{
			"label":  curve.Label,
			"points": points,
			"fit": map[string]any{
				"form":      "risk = amplitude * exp(-lambda * n)",
				"amplitude": fit.Amplitude,
				"lambda":    fit.Lambda,
				"r_squared": fit.RSquared,
				"samples":   fit.Samples,
			},
		})
	}
	return map[string]any{"level_id": levelID, "curves": entries}
}

func comparisonCurve(options CommandOptions, explicit map[string]bool, server string, item target) (charts.FitCurve, string, error) {
	onDisk := fileExists(item.value)
	if onDisk {
		if stored, err := loadStoredRun(item.value); err == nil {
			fmt.Printf("reading %s\n", item.value)
			results := stored.Simulated.Results
			label := named(item.label, curveLabel(stored.Entry.Username, stored.Entry.LevelID, results.SuccessRate))
			return compareCurve(label, results.Sweep), stored.Entry.LevelID, nil
		}
	}

	single := options
	single.Quiet = true
	if onDisk {
		single.Target = ""
		single.EntryFile = item.value
	} else {
		single.Target = item.value
		single.EntryFile = ""
	}

	report, published, err := simulateTarget(single, explicit, server)
	if err != nil {
		return charts.FitCurve{}, "", err
	}
	label := named(item.label, curveLabel(published.Username, published.LevelID, report.Results.SuccessRate))
	return compareCurve(label, report.Results.Sweep), published.LevelID, nil
}

func named(label string, fallback string) string {
	if label == "" {
		return fallback
	}
	return label
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
