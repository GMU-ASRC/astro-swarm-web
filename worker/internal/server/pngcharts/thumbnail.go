package pngcharts

import (
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/text"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/vg/vgimg"

	"astroswarm/worker/internal/server/entrycard"
	"astroswarm/worker/internal/server/values"
)

const (
	thumbnailWidth      = 12.0
	thumbnailHeight     = 6.3
	thumbnailDPI        = 100
	thumbnailBackground = "#0a0e1a"
	thumbnailBrand      = "#7c9eff"
	thumbnailHeading    = "#ffffff"
	thumbnailMuted      = "#8ba3c9"
	thumbnailPositive   = "#4ade80"
	thumbnailNegative   = "#f87171"
	thumbnailGrid       = "#16203a"
	thumbnailSpine      = "#22304d"
	longNameLength      = 12
	longHeadlineLength  = 7
)

type thumbnailCanvas struct {
	full          draw.Canvas
	width, height vg.Length
}

func (canvas thumbnailCanvas) write(x, y float64, label string, size float64, hex string, bold bool) {
	canvas.full.FillText(textStyle(size, hexColor(hex), bold, text.XLeft),
		vg.Point{X: canvas.width * vg.Length(x), Y: canvas.height * vg.Length(y)}, label)
}

func Thumbnail(username string, card entrycard.Card) ([]byte, error) {
	width, height := vg.Length(thumbnailWidth)*vg.Inch, vg.Length(thumbnailHeight)*vg.Inch
	image := vgimg.NewWith(vgimg.UseWH(width, height), vgimg.UseDPI(thumbnailDPI), vgimg.UseBackgroundColor(hexColor(thumbnailBackground)))
	canvas := thumbnailCanvas{full: draw.New(image), width: width, height: height}

	nameSize := 52.0
	if values.Length(username) > longNameLength {
		nameSize = 36.0
	}
	headlineSize := 72.0
	if values.Length(card.Headline) > longHeadlineLength {
		headlineSize = 44.0
	}
	headlineColor := thumbnailNegative
	if card.Positive {
		headlineColor = thumbnailPositive
	}
	canvas.write(0.06, 0.84, "ASTROSWARM", 26, thumbnailBrand, true)
	canvas.write(0.06, 0.66, username, nameSize, thumbnailHeading, true)
	canvas.write(0.06, 0.55, card.LevelName, 20, thumbnailMuted, false)
	canvas.write(0.06, 0.30, card.Headline, headlineSize, headlineColor, true)
	canvas.write(0.06, 0.20, card.Caption, 16, thumbnailMuted, false)

	if len(card.Trend) > 0 {
		chart, err := trendChart(card)
		if err != nil {
			return nil, err
		}
		chart.Draw(canvasArea(image, width, height, 0.53, 0.1, 0.96, 0.86))
	} else {
		for index, fact := range card.Facts {
			y := 0.74 - float64(index)*0.19
			canvas.write(0.62, y, fact.Label, 16, thumbnailMuted, false)
			canvas.write(0.62, y-0.08, fact.Value, 30, thumbnailHeading, true)
		}
	}
	return encodePNG(image)
}

func trendChart(card entrycard.Card) (*plot.Plot, error) {
	trials := make([]float64, len(card.Trend))
	rates := make([]float64, len(card.Trend))
	for index, point := range card.Trend {
		trials[index], rates[index] = float64(point.Trial), point.Rate
	}
	chart := plot.New()
	chart.BackgroundColor = hexColor(thumbnailBackground)
	chart.Title.Text = card.TrendTitle
	chart.Title.TextStyle.Font = face(14, false)
	chart.Title.TextStyle.Color = hexColor(thumbnailMuted)
	chart.X.Label.Text = "Trial number (n)"
	chart.Y.Label.Text = "Rate (%)"
	for _, axis := range []*plot.Axis{&chart.X, &chart.Y} {
		axis.Label.TextStyle.Font = face(12, false)
		axis.Label.TextStyle.Color = hexColor(thumbnailMuted)
		axis.Tick.Label.Font = face(10, false)
		axis.Tick.Label.Color = hexColor(thumbnailMuted)
		axis.Tick.LineStyle.Color = hexColor(thumbnailMuted)
		axis.LineStyle.Color = hexColor(thumbnailSpine)
	}
	addGrid(chart, thumbnailGrid, true)
	if err := addLine(chart, trials, rates, thumbnailPositive, lineOptions{marker: len(trials) == 1, width: 3}); err != nil {
		return nil, err
	}
	percentAxis(chart, "%")
	paddedX(chart, trials)
	chart.X.Tick.Marker = integerTicks{}
	return chart, nil
}
