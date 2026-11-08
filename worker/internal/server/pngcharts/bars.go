package pngcharts

import (
	"fmt"
	"math"

	"gonum.org/v1/plot/text"
)

const (
	categoryBarWidth = 0.8
	timesBarWidth    = 0.42
)

func Times(detectionValue, captureValue any, meta Meta) ([]byte, error) {
	detectionTimes, captureTimes := numbersOf(detectionValue), numbersOf(captureValue)
	count := max(len(detectionTimes), len(captureTimes))
	clamp := func(times []float64) []float64 {
		output := make([]float64, count)
		for index := range output {
			if index < len(times) && times[index] >= 0 {
				output[index] = times[index]
			}
		}
		return output
	}
	detections, captures := clamp(detectionTimes), clamp(captureTimes)
	chart := newChart("Detection and Capture Times", "Trial number (n)", "Time (s)")
	addGrid(chart, gridColor, false)
	tallest := 0.0
	for index := 0; index < count; index++ {
		trial := float64(index + 1)
		detectionBar, err := addBar(chart, trial-timesBarWidth, trial, detections[index], "#4ade80")
		if err != nil {
			return nil, err
		}
		captureBar, err := addBar(chart, trial, trial+timesBarWidth, captures[index], "#f87171")
		if err != nil {
			return nil, err
		}
		if index == 0 {
			chart.Legend.Add("Detection time (s)", detectionBar)
			chart.Legend.Add("Capture time (s)", captureBar)
		}
		tallest = max(tallest, detections[index], captures[index])
	}
	chart.Y.Min, chart.Y.Max = 0, 1.0
	if tallest > 0 {
		chart.Y.Max = tallest * 1.18
	}
	if count > 0 {
		low, high := 1.0-timesBarWidth, float64(count)+timesBarWidth
		padding := (high - low) * axisPadding
		chart.X.Min, chart.X.Max = low-padding, high+padding
	}
	return renderChart(chart, wideChartWidth, meta)
}

func Bar(outcomes []any, meta Meta) ([]byte, error) {
	counts := map[string]int{"win": 0, "lose": 0, "timeout": 0}
	for _, outcome := range outcomes {
		key, _ := outcome.(string)
		if _, known := counts[key]; !known {
			key = "timeout"
		}
		counts[key]++
	}
	total := float64(max(1, len(outcomes)))
	labels := []string{"Intercept", "Planet hit", "Timeout"}
	shares := []float64{
		100.0 * float64(counts["win"]) / total,
		100.0 * float64(counts["lose"]) / total,
		100.0 * float64(counts["timeout"]) / total,
	}
	fills := []string{"#4ade80", "#f87171", "#fbbf24"}
	chart := newChart("Outcome Breakdown", "Outcome", "Share of trials (%)")
	addGrid(chart, gridColor, false)
	marks := dataTexts{}
	for index, share := range shares {
		center := float64(index)
		bar, err := addBar(chart, center-categoryBarWidth/2, center+categoryBarWidth/2, share, fills[index])
		if err != nil {
			return nil, err
		}
		chart.Legend.Add(labels[index], bar)
		marks = append(marks, dataText{x: center, y: share + 1.5, label: fmt.Sprintf("%.0f%%", math.RoundToEven(share)),
			style: textStyle(9, hexColor("#374151"), false, text.XCenter)})
	}
	chart.Add(marks)
	chart.NominalX(labels...)
	percentAxis(chart, "")
	span := float64(len(labels)-1) + categoryBarWidth
	padding := span * axisPadding
	chart.X.Min, chart.X.Max = -categoryBarWidth/2-padding, float64(len(labels)-1)+categoryBarWidth/2+padding
	return renderChart(chart, chartWidth, meta)
}
