package pngcharts

import (
	"fmt"

	"astroswarm/worker/internal/server/values"
)

func Line(outcomes []any, meta Meta) ([]byte, error) {
	wins := 0
	xs, ys := []float64{}, []float64{}
	for index, outcome := range outcomes {
		if outcome == "win" {
			wins++
		}
		xs = append(xs, float64(index+1))
		ys = append(ys, 100.0*float64(wins)/float64(index+1))
	}
	chart := newChart("Cumulative Win Rate", "Trial number (n)", "Cumulative win rate (%)")
	addGrid(chart, gridColor, true)
	if err := addLine(chart, xs, ys, "#1f77b4", lineOptions{label: "Cumulative win rate (%)"}); err != nil {
		return nil, err
	}
	percentAxis(chart, "")
	paddedX(chart, xs)
	return renderChart(chart, chartWidth, meta)
}

func sweepRateChart(rows []map[string]any, rateKey, timeKey, lineHex, title, yLabel string, meta Meta) ([]byte, error) {
	points := sortedBy(rows, "n")
	xs := column(points, "n", 0)
	ys := make([]float64, len(points))
	for index, point := range points {
		ys[index] = sweepRate(point, rateKey, timeKey)
	}
	chart := newChart(title, "Ring size (n)", yLabel)
	addGrid(chart, gridColor, true)
	if err := addLine(chart, xs, ys, lineHex, lineOptions{label: yLabel}); err != nil {
		return nil, err
	}
	percentAxis(chart, "")
	paddedX(chart, xs)
	return renderChart(chart, chartWidth, meta)
}

func DetectionRate(rows []map[string]any, meta Meta) ([]byte, error) {
	return sweepRateChart(rows, "detection_rate", "detection_time", "#2563eb", "Detection Rate by Ring Size", "Detection rate (%)", meta)
}

func CaptureRate(rows []map[string]any, meta Meta) ([]byte, error) {
	return sweepRateChart(rows, "capture_rate", "capture_time", "#dc2626", "Capture Rate by Ring Size", "Capture rate (%)", meta)
}

func Risk(rows []map[string]any, meta Meta) ([]byte, error) {
	points := sortedBy(rows, "n")
	xs := column(points, "n", 0)
	risks := make([]float64, len(points))
	captures := make([]float64, len(points))
	for index, point := range points {
		risks[index] = sweepRisk(point)
		captures[index] = sweepRate(point, "capture_rate", "capture_time")
	}
	chart := newChart("Risk by Ring Size", "Ring size (n)", "Risk (%)")
	addGrid(chart, gridColor, true)
	if err := addLine(chart, xs, risks, "#dc2626", lineOptions{label: "Risk (%)"}); err != nil {
		return nil, err
	}
	if err := addLine(chart, xs, captures, "#16a34a", lineOptions{label: "Capture rate (%)"}); err != nil {
		return nil, err
	}
	percentAxis(chart, "")
	paddedX(chart, xs)
	return renderChart(chart, chartWidth, meta)
}

func TrialRisk(destroyedValue, resolvedValue any, meta Meta) ([]byte, error) {
	destroyed, resolved := numbersOf(destroyedValue), numbersOf(resolvedValue)
	xs := make([]float64, len(destroyed))
	risks := make([]float64, len(destroyed))
	destroyedSum, resolvedSum := 0.0, 0.0
	for index := range destroyed {
		xs[index] = float64(index + 1)
		risks[index] = 100.0
		if index < len(resolved) && resolved[index] > 0 {
			risks[index] = 100.0 - 100.0*destroyed[index]/resolved[index]
		}
		destroyedSum += destroyed[index]
	}
	for _, count := range resolved {
		resolvedSum += count
	}
	overall := 100.0
	if resolvedSum != 0 {
		overall = 100.0 - 100.0*destroyedSum/resolvedSum
	}
	overallLine := make([]float64, len(xs))
	for index := range overallLine {
		overallLine[index] = overall
	}
	chart := newChart("Risk per Trial", "Trial number (n)", "Risk (%)")
	addGrid(chart, gridColor, true)
	if err := addLine(chart, xs, risks, "#dc2626", lineOptions{label: "Risk in this trial (%)"}); err != nil {
		return nil, err
	}
	overallLabel := fmt.Sprintf("Risk across every trial - %s%%", values.FloatRepr(values.RoundTo(overall, 1)))
	if err := addLine(chart, xs, overallLine, "#6b7280", lineOptions{label: overallLabel, dashed: true}); err != nil {
		return nil, err
	}
	percentAxis(chart, "")
	paddedX(chart, xs)
	return renderChart(chart, chartWidth, meta)
}

func Attrition(rows []map[string]any, meta Meta) ([]byte, error) {
	points := sortedBy(rows, "defenders")
	xs := column(points, "defenders", 0)
	ys := column(points, "risk", 0.0)
	chart := newChart("Risk as the Line Thins", "Defenders left (n)", "Risk (%)")
	addGrid(chart, gridColor, true)
	if err := addLine(chart, xs, ys, "#dc2626", lineOptions{label: "Risk (%)", marker: true}); err != nil {
		return nil, err
	}
	percentAxis(chart, "")
	paddedX(chart, xs)
	return renderChart(chart, chartWidth, meta)
}
