package charts

import (
	"image/color"
	"math"
	"os"
	"strconv"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/vg/vgimg"
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
	outerEdge   = vg.Length(14) // border kept clear between the plot and the edge of the png

	percentHeadroom = 3.0  // percentage points the axis runs past 0 and 100
	contentMargin   = 0.05 // fraction of the x range left clear at each end
	legendRoomShare = 0.2  // extra fraction of the x range kept clear for the legend
	countHeadroom   = 0.08 // fraction of the tallest line left clear above a count axis
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

// A line sitting exactly on 0 or 100 is drawn half outside the plot area, so the
// scale runs a little past both ends while the ticks stay on the round numbers.
func percentAxis(p *plot.Plot) {
	p.Y.Min = -percentHeadroom
	p.Y.Max = 100 + percentHeadroom
	percentTicks(p)
}

// Bars are read against the zero line, so this one only takes headroom on top.
func percentAxisFromBaseline(p *plot.Plot) {
	p.Y.Min = 0
	p.Y.Max = 100 + percentHeadroom
	percentTicks(p)
}

// gonum draws a plot right out to the edge of the image, so the canvas is cropped
// by a fixed border first and the plot laid out inside what is left. Without it
// the axis labels sit flush against the edge of the png.
func savePlot(p *plot.Plot, path string) error {
	image := vgimg.PngCanvas{Canvas: vgimg.New(chartWidth, chartHeight)}
	p.Draw(draw.Crop(draw.New(image), outerEdge, -outerEdge, outerEdge, -outerEdge))

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	if _, err := image.WriteTo(file); err != nil {
		file.Close()
		return err
	}
	return file.Close()
}

// A count axis starts at zero and is labelled in whole ships, with the same
// headroom above the tallest line that the percent axis keeps above 100.
func countAxis(p *plot.Plot, ceiling float64) {
	p.Y.Min = 0
	p.Y.Max = math.Max(ceiling, 1) * (1 + countHeadroom)
	p.Y.Tick.Marker = wholeNumbers()
}

func wholeNumbers() plot.TickerFunc {
	return plot.TickerFunc(func(low, high float64) []plot.Tick {
		ticks := plot.DefaultTicks{}.Ticks(low, high)
		for index, tick := range ticks {
			if tick.Label == "" {
				continue
			}
			rounded := math.Round(tick.Value)
			if math.Abs(tick.Value-rounded) > 1e-9 {
				ticks[index].Label = ""
				continue
			}
			ticks[index].Label = strconv.FormatFloat(rounded, 'f', -1, 64)
		}
		return ticks
	})
}

// Every x axis these charts use is a count - trials, evaders faced, ring size,
// defenders standing - so its ticks are whole numbers however the range falls.
func countTicks(p *plot.Plot) {
	p.X.Tick.Marker = wholeNumbers()
}

func marginX(p *plot.Plot, series ...plotter.XYs) {
	countTicks(p)
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
	margin := (high - low) * contentMargin
	p.X.Min = low - margin
	p.X.Max = high + margin
}

// The legend is drawn inside the plot at the top right, where a chart with one
// line per ring is at its most crowded. Widening the x range past the last point
// gives it a clear corner to sit in.
func legendRoom(p *plot.Plot) {
	if p.X.Max <= p.X.Min {
		return
	}
	p.X.Max += (p.X.Max - p.X.Min) * legendRoomShare
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
