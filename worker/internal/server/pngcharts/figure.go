package pngcharts

import (
	"bytes"
	"image/color"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/text"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/vg/vgimg"
)

const (
	chartDPI         = 110
	chartWidth       = 6.4
	wideChartWidth   = 7.2
	chartHeight      = 3.8
	percentCeiling   = 100.0
	percentHeadroom  = 3.0
	countHeadroom    = 0.08
	axisPadding      = 0.05
	gridColor        = "#e5e7eb"
	captionColor     = "#6b7280"
	outerLeft        = 0.02
	outerBottom      = 0.06
	outerRight       = 0.98
	outerTop         = 0.96
	captionBaseline  = 0.005
	legendTextSize   = 8.0
	captionTextSize  = 8.0
	axisTextSize     = 10.0
	titleTextSize    = 12.0
	lineWidthPoints  = 2.0
	markerSizePoints = 4.0
)

type Meta struct {
	Username  string
	LevelID   string
	EvalID    string
	DateLabel string
}

func (meta Meta) caption() string {
	shortID := meta.EvalID
	if len(shortID) > 8 {
		shortID = shortID[:8]
	}
	return meta.Username + "  ·  level: " + meta.LevelID + "  ·  id: " + shortID + "  ·  " + meta.DateLabel
}

func newChart(title, xLabel, yLabel string) *plot.Plot {
	chart := plot.New()
	chart.Title.Text = title
	chart.Title.TextStyle.Font = face(titleTextSize, false)
	chart.X.Label.Text = xLabel
	chart.Y.Label.Text = yLabel
	for _, axis := range []*plot.Axis{&chart.X, &chart.Y} {
		axis.Label.TextStyle.Font = face(axisTextSize, false)
		axis.Tick.Label.Font = face(axisTextSize, false)
	}
	chart.Legend.Top = true
	chart.Legend.TextStyle.Font = face(legendTextSize, false)
	return chart
}

func addGrid(chart *plot.Plot, gridHex string, vertical bool) {
	grid := plotter.NewGrid()
	grid.Horizontal.Color = hexColor(gridHex)
	grid.Vertical.Color = hexColor(gridHex)
	if !vertical {
		grid.Vertical.Color = nil
	}
	chart.Add(grid)
}

func percentTicks(suffix string) plot.ConstantTicks {
	ticks := plot.ConstantTicks{}
	for _, value := range []int{0, 25, 50, 75, 100} {
		ticks = append(ticks, plot.Tick{Value: float64(value), Label: itoa(value) + suffix})
	}
	return ticks
}

func percentAxis(chart *plot.Plot, suffix string) {
	chart.Y.Min = -percentHeadroom
	chart.Y.Max = percentCeiling + percentHeadroom
	chart.Y.Tick.Marker = percentTicks(suffix)
}

func countAxis(chart *plot.Plot, drawn []float64) {
	ceiling := 1.0
	if len(drawn) > 0 {
		ceiling = drawn[0]
		for _, value := range drawn {
			ceiling = max(ceiling, value)
		}
	}
	if ceiling <= 0 {
		ceiling = 1.0
	}
	chart.Y.Min = 0
	chart.Y.Max = ceiling * (1.0 + countHeadroom)
	chart.Y.Tick.Marker = integerTicks{}
}

func paddedX(chart *plot.Plot, xs []float64) {
	if len(xs) == 0 {
		return
	}
	low, high := xs[0], xs[0]
	for _, value := range xs {
		low, high = min(low, value), max(high, value)
	}
	padding := (high - low) * axisPadding
	if padding == 0 {
		padding = 1.0
	}
	chart.X.Min = low - padding
	chart.X.Max = high + padding
}

func pointsOf(xs, ys []float64) plotter.XYs {
	points := make(plotter.XYs, min(len(xs), len(ys)))
	for index := range points {
		points[index] = plotter.XY{X: xs[index], Y: ys[index]}
	}
	return points
}

type lineOptions struct {
	label  string
	dashed bool
	marker bool
	width  float64
}

func addLine(chart *plot.Plot, xs, ys []float64, lineHex string, options lineOptions) error {
	points := pointsOf(xs, ys)
	line, err := plotter.NewLine(points)
	if err != nil {
		return err
	}
	lineColor := hexColor(lineHex)
	line.Color = lineColor
	line.Width = vg.Points(lineWidthPoints)
	if options.width > 0 {
		line.Width = vg.Points(options.width)
	}
	if options.dashed {
		line.Dashes = []vg.Length{vg.Points(7), vg.Points(3)}
	}
	chart.Add(line)
	thumbnails := []plot.Thumbnailer{line}
	if options.marker {
		scatter, err := plotter.NewScatter(points)
		if err != nil {
			return err
		}
		scatter.Shape = draw.CircleGlyph{}
		scatter.Radius = vg.Points(markerSizePoints / 2)
		scatter.Color = lineColor
		chart.Add(scatter)
		thumbnails = append(thumbnails, scatter)
	}
	if options.label != "" {
		chart.Legend.Add(options.label, thumbnails...)
	}
	return nil
}

func addBar(chart *plot.Plot, left, right, height float64, fillHex string) (*plotter.Polygon, error) {
	bar, err := plotter.NewPolygon(plotter.XYs{{X: left, Y: 0}, {X: right, Y: 0}, {X: right, Y: height}, {X: left, Y: height}})
	if err != nil {
		return nil, err
	}
	bar.Color = hexColor(fillHex)
	bar.LineStyle.Width = 0
	chart.Add(bar)
	return bar, nil
}

func canvasArea(canvas vg.Canvas, width, height vg.Length, left, bottom, right, top float64) draw.Canvas {
	return draw.Canvas{Canvas: canvas, Rectangle: vg.Rectangle{
		Min: vg.Point{X: width * vg.Length(left), Y: height * vg.Length(bottom)},
		Max: vg.Point{X: width * vg.Length(right), Y: height * vg.Length(top)},
	}}
}

func encodePNG(canvas *vgimg.Canvas) ([]byte, error) {
	var buffer bytes.Buffer
	if _, err := (vgimg.PngCanvas{Canvas: canvas}).WriteTo(&buffer); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func renderChart(chart *plot.Plot, widthInches float64, meta Meta) ([]byte, error) {
	width, height := vg.Length(widthInches)*vg.Inch, vg.Length(chartHeight)*vg.Inch
	canvas := vgimg.NewWith(vgimg.UseWH(width, height), vgimg.UseDPI(chartDPI), vgimg.UseBackgroundColor(color.White))
	chart.Draw(canvasArea(canvas, width, height, outerLeft, outerBottom, outerRight, outerTop))
	full := draw.New(canvas)
	full.FillText(textStyle(captionTextSize, hexColor(captionColor), false, text.XCenter),
		vg.Point{X: width / 2, Y: height * captionBaseline}, meta.caption())
	return encodePNG(canvas)
}
