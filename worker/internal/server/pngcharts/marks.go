package pngcharts

import (
	"math"
	"strconv"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/text"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
)

func itoa(value int) string {
	return strconv.Itoa(value)
}

type integerTicks struct{}

func (integerTicks) Ticks(low, high float64) []plot.Tick {
	span := high - low
	step := 1.0
	for _, candidate := range []float64{1, 2, 5, 10, 20, 25, 50, 100, 200, 250, 500, 1000, 2000, 5000, 10000} {
		step = candidate
		if span/candidate <= 8 {
			break
		}
	}
	ticks := []plot.Tick{}
	for value := math.Ceil(low/step) * step; value <= high; value += step {
		ticks = append(ticks, plot.Tick{Value: value, Label: strconv.FormatFloat(value, 'f', -1, 64)})
	}
	return ticks
}

type dataText struct {
	x, y  float64
	label string
	style text.Style
}

type dataTexts []dataText

func (marks dataTexts) Plot(canvas draw.Canvas, chart *plot.Plot) {
	transformX, transformY := chart.Transforms(&canvas)
	for _, mark := range marks {
		canvas.FillText(mark.style, vg.Point{X: transformX(mark.x), Y: transformY(mark.y)}, mark.label)
	}
}

type cornerNote struct {
	label string
	style text.Style
}

func (note cornerNote) Plot(canvas draw.Canvas, chart *plot.Plot) {
	inset := vg.Points(4)
	canvas.FillText(note.style, vg.Point{X: canvas.Max.X - inset, Y: canvas.Min.Y + inset}, note.label)
}
