package charts

import (
	"fmt"
	"image/color"
	"math"
	"os"
	"path/filepath"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
)

// Two or three algorithms read against each other on one axis: how the risk each
// one carries falls as the ring it is given grows.
var compareColors = []color.RGBA{
	{R: 0x25, G: 0x63, B: 0xeb, A: 0xff},
	{R: 0xd9, G: 0x77, B: 0x06, A: 0xff},
	{R: 0x16, G: 0xa3, B: 0x4a, A: 0xff},
	{R: 0x93, G: 0x33, B: 0xea, A: 0xff},
}

const fitSamples = 120 // points drawn along a fitted curve

type FitPoint struct {
	N    int
	Risk float64
}

type FitCurve struct {
	Label  string
	Points []FitPoint
}

// risk(n) = amplitude * e^(-lambda * n)
type Fit struct {
	Amplitude float64
	Lambda    float64
	RSquared  float64
	Samples   int
}

func (f Fit) String() string {
	return fmt.Sprintf("risk = %.1f e^(-%.4f n)", f.Amplitude, f.Lambda)
}

func (f Fit) At(n float64) float64 {
	return f.Amplitude * math.Exp(-f.Lambda*n)
}

// Least squares on the log of the risk, which is what turns the exponential into
// a straight line: ln(risk) = ln(amplitude) - lambda * n. A ring that let nothing
// through has no log to take and sits out of the fit, so the fit describes the
// part of the sweep where there was still risk to measure. R squared is reported
// against the risks themselves rather than their logs, so it reads as the share
// of the curve the fit accounts for.
func FitExponential(points []FitPoint) (Fit, bool) {
	sumN, sumLog, sumNN, sumNLog := 0.0, 0.0, 0.0, 0.0
	used := 0
	for _, point := range points {
		if point.Risk <= 0.0 {
			continue
		}
		n := float64(point.N)
		logRisk := math.Log(point.Risk)
		sumN += n
		sumLog += logRisk
		sumNN += n * n
		sumNLog += n * logRisk
		used++
	}
	if used < 2 {
		return Fit{}, false
	}

	count := float64(used)
	denominator := count*sumNN - sumN*sumN
	if math.Abs(denominator) < 1e-12 {
		return Fit{}, false
	}
	slope := (count*sumNLog - sumN*sumLog) / denominator
	intercept := (sumLog - slope*sumN) / count

	fit := Fit{Amplitude: math.Exp(intercept), Lambda: -slope, Samples: used}
	fit.RSquared = rSquared(points, fit)
	return fit, true
}

func rSquared(points []FitPoint, fit Fit) float64 {
	mean := 0.0
	for _, point := range points {
		mean += point.Risk
	}
	mean /= float64(len(points))

	residual, total := 0.0, 0.0
	for _, point := range points {
		predicted := fit.At(float64(point.N))
		residual += (point.Risk - predicted) * (point.Risk - predicted)
		total += (point.Risk - mean) * (point.Risk - mean)
	}
	if total == 0.0 {
		return 0.0
	}
	return 1.0 - residual/total
}

// The comparison figure is written on its own rather than through WriteAll: it
// is built from several entries, where every other chart describes just one.
func WriteRiskFit(directory string, curves []FitCurve, subtitle string, name string) (string, []Fit, error) {
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return "", nil, err
	}
	figure, fits, err := RiskFitChart(curves, subtitle)
	if err != nil || figure == nil {
		return "", fits, err
	}
	path := filepath.Join(directory, name)
	if err := savePlot(figure, path); err != nil {
		return "", fits, err
	}
	return path, fits, nil
}

func RiskFitChart(curves []FitCurve, subtitle string) (*plot.Plot, []Fit, error) {
	if len(curves) == 0 {
		return nil, nil, nil
	}
	p := newPlot("Risk by ring size"+suffix(subtitle), "Ring size (n)", "Risk (%)")
	percentAxis(p)

	fits := make([]Fit, 0, len(curves))
	drawn := make([]plotter.XYs, 0, len(curves)*2)
	for index, curve := range curves {
		shade := compareColors[index%len(compareColors)]
		measured := make(plotter.XYs, 0, len(curve.Points))
		for _, point := range curve.Points {
			measured = append(measured, plotter.XY{X: float64(point.N), Y: point.Risk})
		}
		line, err := addSeries(p, measured, shade)
		if err != nil {
			return nil, nil, err
		}
		p.Legend.Add(curve.Label, line)
		drawn = append(drawn, measured)

		fit, ok := FitExponential(curve.Points)
		if !ok {
			fits = append(fits, Fit{})
			continue
		}
		fits = append(fits, fit)
		fitted, err := addDashed(p, fitCurve(fit, measured), shade)
		if err != nil {
			return nil, nil, err
		}
		p.Legend.Add(fmt.Sprintf("  %s", fit), fitted)
	}

	marginX(p, drawn...)
	legendRoom(p)
	return p, fits, nil
}

func fitCurve(fit Fit, measured plotter.XYs) plotter.XYs {
	low, high := measured[0].X, measured[0].X
	for _, point := range measured {
		low = math.Min(low, point.X)
		high = math.Max(high, point.X)
	}
	points := make(plotter.XYs, 0, fitSamples)
	for index := 0; index < fitSamples; index++ {
		n := low + (high-low)*float64(index)/float64(fitSamples-1)
		points = append(points, plotter.XY{X: n, Y: fit.At(n)})
	}
	return points
}
