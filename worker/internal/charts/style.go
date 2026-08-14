package charts

import (
	"image/color"
	"math"
	"strconv"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
)

var (
	surfaceColor   = color.RGBA{R: 0xfc, G: 0xfc, B: 0xfb, A: 0xff}
	primaryInk     = color.RGBA{R: 0x0b, G: 0x0b, B: 0x0b, A: 0xff}
	secondaryInk   = color.RGBA{R: 0x52, G: 0x51, B: 0x4e, A: 0xff}
	gridInk        = color.RGBA{R: 0xe5, G: 0xe4, B: 0xe0, A: 0xff}
	simulatedColor = color.RGBA{R: 0x2a, G: 0x78, B: 0xd6, A: 0xff}
	publishedColor = color.RGBA{R: 0xeb, G: 0x68, B: 0x34, A: 0xff}
	captureColor   = color.RGBA{R: 0x1b, G: 0xaf, B: 0x7a, A: 0xff}
)

const (
	chartWidth  = 9 * vg.Inch
	chartHeight = 5 * vg.Inch
	markWidth   = vg.Length(2)
)

func newPlot(title, xLabel, yLabel string) *plot.Plot {
	p := plot.New()
	p.BackgroundColor = surfaceColor

	p.Title.Text = title
	p.Title.TextStyle.Color = primaryInk
	p.Title.TextStyle.Font.Size = vg.Points(14)
	p.Title.Padding = vg.Points(8)

	p.X.Label.Text = xLabel
	p.Y.Label.Text = yLabel
	for _, axis := range []*plot.Axis{&p.X, &p.Y} {
		axis.Label.TextStyle.Color = secondaryInk
		axis.Label.TextStyle.Font.Size = vg.Points(11)
		axis.Tick.Label.Color = secondaryInk
		axis.Tick.Label.Font.Size = vg.Points(10)
		axis.Tick.Color = gridInk
		axis.Tick.Width = vg.Points(0.75)
		axis.Color = gridInk
		axis.Width = vg.Points(0.75)
		axis.Padding = vg.Points(6)
	}

	p.Legend.TextStyle.Color = secondaryInk
	p.Legend.TextStyle.Font.Size = vg.Points(10)
	p.Legend.Top = true
	p.Legend.ThumbnailWidth = vg.Points(14)
	p.Legend.Padding = vg.Points(4)

	grid := plotter.NewGrid()
	grid.Horizontal = draw.LineStyle{Color: gridInk, Width: vg.Points(0.75)}
	grid.Vertical = draw.LineStyle{Color: color.Transparent, Width: 0}
	p.Add(grid)

	return p
}

func percentTicks(p *plot.Plot) {
	p.Y.Tick.Marker = plot.ConstantTicks([]plot.Tick{
		{Value: 0, Label: "0"},
		{Value: 25, Label: "25"},
		{Value: 50, Label: "50"},
		{Value: 75, Label: "75"},
		{Value: 100, Label: "100"},
	})
}

func percentAxis(p *plot.Plot) {
	p.Y.Min = 0
	p.Y.Max = 100
	percentTicks(p)
}

func percentAxisFromBaseline(p *plot.Plot) {
	p.Y.Min = 0
	p.Y.Max = 100
	percentTicks(p)
}

func marginX(p *plot.Plot, series ...plotter.XYs) {
	low := math.Inf(1)
	high := math.Inf(-1)
	for _, points := range series {
		for _, point := range points {
			low = math.Min(low, point.X)
			high = math.Max(high, point.X)
		}
	}
	if math.IsInf(low, 1) || high <= low {
		return
	}
	margin := (high - low) * 0.03
	p.X.Min = low - margin
	p.X.Max = high + margin
}

func styleBars(bars *plotter.BarChart, shade color.Color) {
	bars.Color = shade
	bars.LineStyle = draw.LineStyle{Color: surfaceColor, Width: markWidth}
}

func addSeries(p *plot.Plot, points plotter.XYs, shade color.Color) (*plotter.Line, error) {
	line, err := plotter.NewLine(points)
	if err != nil {
		return nil, err
	}
	line.LineStyle = draw.LineStyle{Color: shade, Width: markWidth}
	p.Add(line)
	return line, nil
}

func addValueLabels(p *plot.Plot, values plotter.Values, offset vg.Length) error {
	data := plotter.XYLabels{
		XYs:    make(plotter.XYs, 0, len(values)),
		Labels: make([]string, 0, len(values)),
	}
	for index, value := range values {
		data.XYs = append(data.XYs, plotter.XY{X: float64(index), Y: value})
		data.Labels = append(data.Labels, formatPercent(value))
	}

	labels, err := plotter.NewLabels(data)
	if err != nil {
		return err
	}
	labels.Offset = vg.Point{X: offset, Y: vg.Points(4)}
	for index := range labels.TextStyle {
		labels.TextStyle[index].Color = primaryInk
		labels.TextStyle[index].Font.Size = vg.Points(10)
		labels.TextStyle[index].XAlign = draw.XCenter
	}
	p.Add(labels)
	return nil
}

func formatPercent(value float64) string {
	return strconv.FormatFloat(value, 'f', 1, 64) + "%"
}
