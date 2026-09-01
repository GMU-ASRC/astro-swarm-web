package charts

import (
	"os"
	"path/filepath"
	"sort"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"

	"astroswarm/worker/internal/bench"
	"astroswarm/worker/internal/entry"
)

type Input struct {
	Report    bench.Report
	Published *entry.Entry
	Subtitle  string
}

func WriteAll(directory string, input Input) ([]string, error) {
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return nil, err
	}

	written := []string{}
	steps := []struct {
		name  string
		build func(Input) (*plot.Plot, error)
	}{
		{"detection_rate.png", detectionRateChart},
		{"detection_rate_vs_defenders.png", detectionVersusDefenders},
		{"capture_rate_vs_defenders.png", captureVersusDefenders},
		{"rates_vs_defenders.png", combinedVersusDefenders},
		{"risk_per_trial.png", riskPerTrial},
		{"risk_vs_defenders.png", riskVersusDefenders},
		{"risk_vs_attrition.png", riskVersusAttrition},
		{"risk_vs_attrition_by_ring.png", riskVersusSweepAttrition},
		{"capture_rate_by_ring.png", captureRateByRing},
		{"risk_by_ring.png", riskByRing},
		{"attrition_by_ring.png", attritionByRing},
	}

	for _, step := range steps {
		figure, err := step.build(input)
		if err != nil {
			return written, err
		}
		if figure == nil {
			continue
		}
		path := filepath.Join(directory, step.name)
		if err := figure.Save(chartWidth, chartHeight, path); err != nil {
			return written, err
		}
		written = append(written, path)
	}
	return written, nil
}

func detectionRateChart(input Input) (*plot.Plot, error) {
	results := input.Report.Results
	p := newPlot(
		"Outcome rates"+suffix(input.Subtitle),
		"",
		"Trials (%)",
	)
	percentAxisFromBaseline(p)
	p.NominalX("Evader detected", "Evader captured", "Defenders win")

	simulated := plotter.Values{results.DetectionRate, results.CaptureRate, results.SuccessRate}
	published, hasPublished := publishedHeadline(input.Published)

	barWidth := vg.Points(46)
	if hasPublished {
		barWidth = vg.Points(26)
	}

	simulatedBars, err := plotter.NewBarChart(simulated, barWidth)
	if err != nil {
		return nil, err
	}
	styleBars(simulatedBars, simulatedColor)

	if !hasPublished {
		p.Add(simulatedBars)
		if err := addValueLabels(p, simulated, 0); err != nil {
			return nil, err
		}
		return p, nil
	}

	publishedBars, err := plotter.NewBarChart(published, barWidth)
	if err != nil {
		return nil, err
	}
	styleBars(publishedBars, publishedColor)

	simulatedBars.Offset = -barWidth / 2
	publishedBars.Offset = barWidth / 2
	p.Add(simulatedBars, publishedBars)
	if err := addValueLabels(p, simulated, -barWidth/2); err != nil {
		return nil, err
	}
	if err := addValueLabels(p, published, barWidth/2); err != nil {
		return nil, err
	}
	p.Legend.Add("Simulated here", simulatedBars)
	p.Legend.Add("Published on the site", publishedBars)
	return p, nil
}

func detectionVersusDefenders(input Input) (*plot.Plot, error) {
	points := sweepPoints(input.Report, func(point bench.SweepPoint) float64 { return point.DetectionRate })
	if len(points) == 0 {
		return nil, nil
	}
	p := newPlot(
		"Detection rate by ring size"+suffix(input.Subtitle),
		"Ring size (n)",
		"Detection rate (%)",
	)
	percentAxis(p)

	line, err := addSeries(p, points, simulatedColor)
	if err != nil {
		return nil, err
	}
	published := publishedRates(input.Published, func(point entry.SweepIndexEntry) *float64 {
		return point.DetectionRate
	})
	if len(published) > 0 {
		reference, err := addSeries(p, published, publishedColor)
		if err != nil {
			return nil, err
		}
		p.Legend.Add("Simulated", line)
		p.Legend.Add("Published", reference)
	}
	marginX(p, points, published)
	return p, nil
}

func captureVersusDefenders(input Input) (*plot.Plot, error) {
	points := sweepPoints(input.Report, func(point bench.SweepPoint) float64 { return point.CaptureRate })
	if len(points) == 0 {
		return nil, nil
	}
	p := newPlot(
		"Capture rate by ring size"+suffix(input.Subtitle),
		"Ring size (n)",
		"Capture rate (%)",
	)
	percentAxis(p)

	line, err := addSeries(p, points, captureColor)
	if err != nil {
		return nil, err
	}
	published := publishedRates(input.Published, func(point entry.SweepIndexEntry) *float64 {
		return point.CaptureRate
	})
	if len(published) == 0 {
		published = publishedSweep(input.Published)
	}
	if len(published) > 0 {
		reference, err := addSeries(p, published, publishedColor)
		if err != nil {
			return nil, err
		}
		p.Legend.Add("Simulated", line)
		p.Legend.Add("Published", reference)
	}
	marginX(p, points, published)
	return p, nil
}

func combinedVersusDefenders(input Input) (*plot.Plot, error) {
	detection := sweepPoints(input.Report, func(point bench.SweepPoint) float64 { return point.DetectionRate })
	capture := sweepPoints(input.Report, func(point bench.SweepPoint) float64 { return point.CaptureRate })
	if len(detection) == 0 {
		return nil, nil
	}
	p := newPlot(
		"Detection and capture rates by ring size"+suffix(input.Subtitle),
		"Ring size (n)",
		"Rate (%)",
	)
	percentAxis(p)

	detectionLine, err := addSeries(p, detection, simulatedColor)
	if err != nil {
		return nil, err
	}
	captureLine, err := addSeries(p, capture, captureColor)
	if err != nil {
		return nil, err
	}
	p.Legend.Add("Detected", detectionLine)
	p.Legend.Add("Captured", captureLine)
	marginX(p, detection, capture)
	return p, nil
}

func sweepPoints(report bench.Report, pick func(bench.SweepPoint) float64) plotter.XYs {
	points := make(plotter.XYs, 0, len(report.Results.Sweep))
	for _, point := range report.Results.Sweep {
		points = append(points, plotter.XY{X: float64(point.N), Y: pick(point)})
	}
	return points
}

func publishedRates(published *entry.Entry, pick func(entry.SweepIndexEntry) *float64) plotter.XYs {
	if published == nil {
		return nil
	}
	points := make(plotter.XYs, 0, len(published.SweepIndex))
	for _, point := range published.SweepIndex {
		value := pick(point)
		if value == nil {
			continue
		}
		points = append(points, plotter.XY{X: float64(point.N), Y: *value})
	}
	sort.Slice(points, func(a, b int) bool { return points[a].X < points[b].X })
	return points
}

func publishedSweep(published *entry.Entry) plotter.XYs {
	if published == nil || published.Results == nil {
		return nil
	}
	points := make(plotter.XYs, 0, len(published.Results.Sweep))
	for _, point := range published.Results.Sweep {
		points = append(points, plotter.XY{X: float64(point.N), Y: point.SuccessRate})
	}
	return points
}

func publishedHeadline(published *entry.Entry) (plotter.Values, bool) {
	if published == nil || published.Results == nil {
		return nil, false
	}
	results := published.Results
	if results.DetectionRate == nil || results.CaptureRate == nil || results.SuccessRate == nil {
		return nil, false
	}
	return plotter.Values{*results.DetectionRate, *results.CaptureRate, *results.SuccessRate}, true
}

func suffix(subtitle string) string {
	if subtitle == "" {
		return ""
	}
	return " - " + subtitle
}
