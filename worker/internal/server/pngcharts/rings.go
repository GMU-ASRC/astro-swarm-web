package pngcharts

import (
	"fmt"
	"math"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/text"

	"astroswarm/worker/internal/server/values"
)

const (
	rampStartHue = 220.0
	rampEndHue   = 0.0
	legendMax    = 8
)

func hueComponent(low, high, hue float64) float64 {
	hue = math.Mod(hue, 1.0)
	if hue < 0 {
		hue += 1.0
	}
	switch {
	case hue < 1.0/6.0:
		return low + (high-low)*hue*6.0
	case hue < 0.5:
		return high
	case hue < 2.0/3.0:
		return low + (high-low)*(2.0/3.0-hue)*6.0
	}
	return low
}

func hueColor(hue float64) string {
	lightness, saturation := 0.45, 0.7
	high := lightness * (1.0 + saturation)
	if lightness > 0.5 {
		high = lightness + saturation - lightness*saturation
	}
	low := 2.0*lightness - high
	fraction := hue / 360.0
	red := hueComponent(low, high, fraction+1.0/3.0)
	green := hueComponent(low, high, fraction)
	blue := hueComponent(low, high, fraction-1.0/3.0)
	return fmt.Sprintf("#%02x%02x%02x", int(math.RoundToEven(red*255)), int(math.RoundToEven(green*255)), int(math.RoundToEven(blue*255)))
}

func ramp(count int) []string {
	if count < 2 {
		return []string{hueColor(rampEndHue)}
	}
	span := rampStartHue - rampEndHue
	colors := make([]string, count)
	for index := range colors {
		colors[index] = hueColor(rampStartHue - span*float64(index)/float64(count-1))
	}
	return colors
}

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
		anchors[int(math.RoundToEven(float64(index)*step))] = true
	}
	return anchors
}

func rampNote(chart *plot.Plot, count int) {
	if count > legendMax {
		chart.Add(cornerNote{
			label: fmt.Sprintf("%d rings drawn, blue smallest to red largest", count),
			style: textStyle(legendTextSize, hexColor(captionColor), false, text.XRight),
		})
	}
}

func ringLabel(entry map[string]any) string {
	return "n = " + values.ToText(entry["n"])
}

type ringChart struct {
	title, xLabel, yLabel string
	xKey                  string
	pick                  func(point map[string]any) float64
	percent               bool
}

func (spec ringChart) render(seriesValue any, meta Meta) ([]byte, error) {
	chosen := sortedBy(objects(seriesValue), "n")
	colors := ramp(len(chosen))
	anchors := legendAnchors(len(chosen))
	chart := newChart(spec.title, spec.xLabel, spec.yLabel)
	addGrid(chart, gridColor, true)
	everything, drawn := []float64{}, []float64{}
	for index, entry := range chosen {
		points := sortedBy(objects(entry["points"]), spec.xKey)
		xs := column(points, spec.xKey, 0)
		ys := make([]float64, len(points))
		for pointIndex, point := range points {
			ys[pointIndex] = spec.pick(point)
		}
		everything = append(everything, xs...)
		drawn = append(drawn, ys...)
		label := ""
		if anchors[index] {
			label = ringLabel(entry)
		}
		if err := addLine(chart, xs, ys, colors[index%len(colors)], lineOptions{label: label}); err != nil {
			return nil, err
		}
	}
	rampNote(chart, len(chosen))
	if spec.percent {
		percentAxis(chart, "")
	} else {
		countAxis(chart, drawn)
	}
	paddedX(chart, everything)
	return renderChart(chart, chartWidth, meta)
}

func pick(key string) func(point map[string]any) float64 {
	return func(point map[string]any) float64 { return numberOr(point, key, 0.0) }
}

func SweepAttrition(series any, meta Meta) ([]byte, error) {
	return ringChart{title: "Risk as the Line Thins, per Ring", xLabel: "Defenders left (n)", yLabel: "Risk (%)",
		xKey: "defenders", pick: pick("risk"), percent: true}.render(series, meta)
}

func RingCapture(series any, meta Meta) ([]byte, error) {
	return ringChart{title: "Capture Rate, Wave by Wave", xLabel: "Evaders faced (n)", yLabel: "Capture rate (%)",
		xKey: "faced", pick: pick("capture_rate"), percent: true}.render(series, meta)
}

func RingRisk(series any, meta Meta) ([]byte, error) {
	return ringChart{title: "Risk, Wave by Wave", xLabel: "Evaders faced (n)", yLabel: "Risk (%)",
		xKey: "faced", pick: pick("risk"), percent: true}.render(series, meta)
}

func RingAttrition(series any, meta Meta) ([]byte, error) {
	return ringChart{title: "Defenders Remaining, Wave by Wave", xLabel: "Evaders faced (n)", yLabel: "Defenders remaining",
		xKey: "faced", pick: pick("defenders"), percent: false}.render(series, meta)
}
