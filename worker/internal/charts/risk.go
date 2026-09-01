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

const (
	rampStartHue = 220.0 // degrees, the hue given to the smallest ring
	rampEndHue   = 0.0   // degrees, the hue given to the largest ring
	legendMax    = 8     // count, ring sizes the legend can name before it fills the figure
)

// A legend cannot hold forty entries, but forty unnamed lines are unreadable, so
// it names an even spread of them instead. They are anchors on the ramp: any
// line can be placed by the two named ones its color sits between.
func legendAnchors(count int) map[int]bool {
	anchors := map[int]bool{}
	if count <= legendMax {
		for index := 0; index < count; index++ {
			anchors[index] = true
		}
		return anchors
	}
	step := float64(count-1) / float64(legendMax-1)
	for index := 0; index < legendMax; index++ {
		anchors[int(math.Round(float64(index)*step))] = true
	}
	return anchors
}

// Ring size is an ordered quantity, so a chart with one line per n reads as a
// ramp rather than as a set of unrelated colors: the smallest ring is blue and
// the largest is red.
func seriesRamp(count int) []color.RGBA {
	if count < 2 {
		return []color.RGBA{hueColor(rampEndHue)}
	}
	span := rampStartHue - rampEndHue
	out := make([]color.RGBA, 0, count)
	for index := 0; index < count; index++ {
		out = append(out, hueColor(rampStartHue-span*float64(index)/float64(count-1)))
	}
	return out
}

func hueColor(hue float64) color.RGBA {
	const saturation = 0.7
	const lightness = 0.45
	chroma := (1.0 - math.Abs(2.0*lightness-1.0)) * saturation
	section := hue / 60.0
	second := chroma * (1.0 - math.Abs(math.Mod(section, 2.0)-1.0))
	base := lightness - chroma/2.0

	var red, green, blue float64
	switch {
	case section < 1.0:
		red, green, blue = chroma, second, 0.0
	case section < 2.0:
		red, green, blue = second, chroma, 0.0
	case section < 3.0:
		red, green, blue = 0.0, chroma, second
	case section < 4.0:
		red, green, blue = 0.0, second, chroma
	case section < 5.0:
		red, green, blue = second, 0.0, chroma
	default:
		red, green, blue = chroma, 0.0, second
	}
	return color.RGBA{
		R: uint8(math.Round((red + base) * 255.0)),
		G: uint8(math.Round((green + base) * 255.0)),
		B: uint8(math.Round((blue + base) * 255.0)),
		A: 0xff,
	}
}

// Risk trial by trial, so a run that averages well but collapses on a handful of
// scatters can be told apart from one that holds the same line every time.
func riskPerTrial(input Input) (*plot.Plot, error) {
	results := input.Report.Results
	if len(results.TrialResolved) == 0 {
		return nil, nil
	}
	points := make(plotter.XYs, 0, len(results.TrialResolved))
	overall := make(plotter.XYs, 0, len(results.TrialResolved))
	mean := riskOf(destroyRateOf(results.EvadersDestroyed, results.EvadersResolved))
	for index, resolved := range results.TrialResolved {
		risk := 100.0
		if resolved > 0 && index < len(results.TrialDestroyed) {
			risk = riskOf(100.0 * float64(results.TrialDestroyed[index]) / float64(resolved))
		}
		points = append(points, plotter.XY{X: float64(index + 1), Y: risk})
		overall = append(overall, plotter.XY{X: float64(index + 1), Y: mean})
	}

	p := newPlot(
		"Risk per trial"+suffix(input.Subtitle),
		"Trial",
		"Risk (%)",
	)
	percentAxis(p)

	line, err := addSeries(p, points, riskColor)
	if err != nil {
		return nil, err
	}
	meanLine, err := addSeries(p, overall, secondaryInk)
	if err != nil {
		return nil, err
	}
	p.Legend.Add("Risk in this trial", line)
	p.Legend.Add(fmt.Sprintf("Risk across every trial - %.1f%%", mean), meanLine)
	marginX(p, points)
	return p, nil
}

func destroyRateOf(destroyed int, resolved int) float64 {
	if resolved < 1 {
		return 0.0
	}
	return 100.0 * float64(destroyed) / float64(resolved)
}

func riskOf(captureRate float64) float64 {
	return 100.0 - captureRate
}

// Risk is the share of evaders that were not stopped, so it reads as the
// complement of the capture success rate on the same axis.
func riskVersusDefenders(input Input) (*plot.Plot, error) {
	points := sweepPoints(input.Report, riskOfPoint)
	if len(points) == 0 {
		return nil, nil
	}
	p := newPlot(
		"Risk by ring size"+suffix(input.Subtitle),
		"Ring size (n)",
		"Risk (%)",
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
	p.Legend.Add("Capture rate", captureLine)
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
		"Risk as the line thins"+suffix(input.Subtitle),
		"Defenders standing",
		"Risk (%)",
	)
	percentAxis(p)

	line, err := addSeries(p, points, riskColor)
	if err != nil {
		return nil, err
	}
	p.Legend.Add("Risk", line)
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
	series := input.Report.Results.SweepAttrition
	if len(series) == 0 {
		return nil, nil
	}
	colors := seriesRamp(len(series))
	anchors := legendAnchors(len(series))
	p := newPlot(
		"Risk as the line thins, per ring"+suffix(input.Subtitle),
		"Defenders standing",
		"Risk (%)",
	)
	percentAxis(p)

	drawn := make([]plotter.XYs, 0, len(series))
	for index, entry := range series {
		points := make(plotter.XYs, 0, len(entry.Points))
		for _, point := range entry.Points {
			points = append(points, plotter.XY{X: float64(point.Defenders), Y: point.Risk})
		}
		line, err := addSeries(p, points, colors[index])
		if err != nil {
			return nil, err
		}
		if anchors[index] {
			p.Legend.Add(fmt.Sprintf("n = %d", entry.N), line)
		}
		drawn = append(drawn, points)
	}
	marginX(p, drawn...)
	legendRoom(p)
	return p, nil
}

// The three ring-sweep curves the site draws side by side: how each ring size's
// capture rate settles, the risk that leaves, and how much of the line is still
// standing, all read wave by wave rather than in total. Every ring size gets its
// own line, so the sizes can be compared as runs rather than as end numbers.
func captureRateByRing(input Input) (*plot.Plot, error) {
	return progressChart(input,
		"Capture rate, wave by wave"+suffix(input.Subtitle),
		"Capture rate (%)",
		func(point bench.SweepProgressPoint) float64 { return point.CaptureRate })
}

func riskByRing(input Input) (*plot.Plot, error) {
	return progressChart(input,
		"Risk, wave by wave"+suffix(input.Subtitle),
		"Risk (%)",
		func(point bench.SweepProgressPoint) float64 { return point.Risk })
}

// A count of ships rather than a rate, so the biggest ring sets the axis and the
// smaller ones sit low on it, which is the comparison worth seeing.
func attritionByRing(input Input) (*plot.Plot, error) {
	return progressChart(input,
		"Defenders remaining, wave by wave"+suffix(input.Subtitle),
		"Defenders remaining",
		nil)
}

func progressChart(input Input, title string, yLabel string, pick func(bench.SweepProgressPoint) float64) (*plot.Plot, error) {
	series := input.Report.Results.SweepProgress
	if len(series) == 0 {
		return nil, nil
	}
	colors := seriesRamp(len(series))
	anchors := legendAnchors(len(series))
	p := newPlot(title, "Evaders faced", yLabel)
	counting := pick == nil
	if !counting {
		percentAxis(p)
	}

	ceiling := 0.0
	drawn := make([]plotter.XYs, 0, len(series))
	for index, entry := range series {
		points := make(plotter.XYs, 0, len(entry.Points))
		for _, point := range entry.Points {
			value := point.Defenders
			if pick != nil {
				value = pick(point)
			}
			ceiling = math.Max(ceiling, value)
			points = append(points, plotter.XY{X: float64(point.Faced), Y: value})
		}
		line, err := addSeries(p, points, colors[index])
		if err != nil {
			return nil, err
		}
		if anchors[index] {
			p.Legend.Add(fmt.Sprintf("n = %d", entry.N), line)
		}
		drawn = append(drawn, points)
	}
	if counting {
		countAxis(p, ceiling)
	}
	marginX(p, drawn...)
	legendRoom(p)
	return p, nil
}
