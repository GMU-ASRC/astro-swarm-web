package charts

import (
	"fmt"
	"image/color"
	"math"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"

	"astroswarm/worker/internal/bench"
)

var riskColor = color.RGBA{R: 0xd9, G: 0x3a, B: 0x3a, A: 0xff}

// Enough lines to read a handful of ring sizes apart without the legend taking
// over the figure.
var seriesColors = []color.RGBA{
	{R: 0x25, G: 0x63, B: 0xeb, A: 0xff},
	{R: 0x16, G: 0xa3, B: 0x4a, A: 0xff},
	{R: 0xd9, G: 0x77, B: 0x06, A: 0xff},
	{R: 0x93, G: 0x33, B: 0xea, A: 0xff},
	{R: 0x08, G: 0x91, B: 0xb2, A: 0xff},
	{R: 0xdb, G: 0x27, B: 0x77, A: 0xff},
}

const maxAttritionSeries = 6

// Risk is the share of evaders that were not stopped, so it reads as the
// complement of the capture success rate on the same axis.
func riskVersusDefenders(input Input) (*plot.Plot, error) {
	points := sweepPoints(input.Report, riskOfPoint)
	if len(points) == 0 {
		return nil, nil
	}
	p := newPlot(
		"Risk vs number of defenders"+suffix(input.Subtitle),
		"Defenders on the ring (n)",
		"Risk = 1 - capture success rate (%)",
	)
	percentAxis(p)

	line, err := addSeries(p, points, riskColor)
	if err != nil {
		return nil, err
	}
	capture := sweepPoints(input.Report, func(point bench.SweepPoint) float64 { return point.CaptureRate })
	captureLine, err := addSeries(p, capture, captureColor)
	if err != nil {
		return nil, err
	}
	p.Legend.Add("Risk", line)
	p.Legend.Add("Capture success rate", captureLine)
	marginX(p, points, capture)
	return p, nil
}

// The attrition curve reads right to left: a full line sits at the high end of
// the axis, and every trade moves the algorithm one rung down it.
func riskVersusAttrition(input Input) (*plot.Plot, error) {
	attrition := input.Report.Results.Attrition
	if len(attrition) < 2 {
		return nil, nil
	}
	points := make(plotter.XYs, 0, len(attrition))
	for _, point := range attrition {
		points = append(points, plotter.XY{X: float64(point.Defenders), Y: point.Risk})
	}

	p := newPlot(
		"Risk as the defender line thins"+suffix(input.Subtitle),
		"Defenders still standing when the evader launched",
		"Risk = 1 - capture success rate (%)",
	)
	percentAxis(p)

	line, err := addSeries(p, points, riskColor)
	if err != nil {
		return nil, err
	}
	p.Legend.Add("Risk at this line size", line)
	marginX(p, points)
	return p, nil
}

func riskOfPoint(point bench.SweepPoint) float64 {
	if point.Risk > 0.0 {
		return point.Risk
	}
	return 100.0 - point.CaptureRate
}

// One curve per ring size: a line that started at n and traded itself down, so
// two algorithms can be compared on what their risk does as the line thins
// rather than on a single number at full strength.
func riskVersusSweepAttrition(input Input) (*plot.Plot, error) {
	series := spreadSeries(input.Report.Results.SweepAttrition, maxAttritionSeries)
	if len(series) == 0 {
		return nil, nil
	}
	p := newPlot(
		"Risk as the line thins, by ring size"+suffix(input.Subtitle),
		"Defenders still standing when the evader launched",
		"Risk = 1 - capture success rate (%)",
	)
	percentAxis(p)

	drawn := make([]plotter.XYs, 0, len(series))
	for index, entry := range series {
		points := make(plotter.XYs, 0, len(entry.Points))
		for _, point := range entry.Points {
			points = append(points, plotter.XY{X: float64(point.Defenders), Y: point.Risk})
		}
		line, err := addSeries(p, points, seriesColors[index%len(seriesColors)])
		if err != nil {
			return nil, err
		}
		p.Legend.Add(fmt.Sprintf("n = %d", entry.N), line)
		drawn = append(drawn, points)
	}
	marginX(p, drawn...)
	return p, nil
}

// Ring sizes are consecutive, so drawing every one of them is unreadable. Take
// an even spread across the sweep and always keep the largest ring, which is
// the one the sweep stopped at.
func spreadSeries(series []bench.AttritionSeries, limit int) []bench.AttritionSeries {
	if len(series) <= limit {
		return series
	}
	out := make([]bench.AttritionSeries, 0, limit)
	step := float64(len(series)-1) / float64(limit-1)
	for index := 0; index < limit; index++ {
		out = append(out, series[int(math.Round(float64(index)*step))])
	}
	return out
}
