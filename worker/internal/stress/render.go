package stress

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"time"

	"git.sr.ht/~sbinet/gg"
	"github.com/go-fonts/liberation/liberationmonoregular"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"

	"astroswarm/worker/internal/blocks"
	"astroswarm/worker/internal/sim"
)

var (
	backgroundColor = color.RGBA{R: 0x0a, G: 0x0e, B: 0x1a, A: 0xff}
	gridColor       = color.RGBA{R: 0x16, G: 0x20, B: 0x3a, A: 0xff}
	planetFill      = color.RGBA{R: 0x3a, G: 0x7d, B: 0x5a, A: 0xff}
	planetEdge      = color.RGBA{R: 0x7f, G: 0xc8, B: 0x9c, A: 0xff}
	evaderColor     = color.RGBA{R: 0xff, G: 0x5a, B: 0x4a, A: 0xff}
	panelColor      = color.NRGBA{R: 0x05, G: 0x07, B: 0x0f, A: 0xc8}
	panelTextColor  = color.RGBA{R: 0xe8, G: 0xec, B: 0xf5, A: 0xff}
	panelDimColor   = color.RGBA{R: 0x8b, G: 0xa3, B: 0xc9, A: 0xff}
	panelGoodColor  = color.RGBA{R: 0x4a, G: 0xde, B: 0x80, A: 0xff}
	panelWarnColor  = color.RGBA{R: 0xfb, G: 0xbf, B: 0x24, A: 0xff}
)

const (
	referenceVideoWidth = 1920.0
	baseFontSize        = 20.0
	gridStep            = 200.0
	minimumShipRadius   = 1.5
	headingScale        = 2.2
	coneFillAlpha       = 0x14
	coneEdgeAlpha       = 0x5a
	panelMargin         = 16.0
	panelPadding        = 14.0
	panelRadius         = 6.0
	swatchGap           = 10.0
)

type overlayLine struct {
	text   string
	shade  color.Color
	swatch color.Color
}

type Renderer struct {
	options    Options
	configs    []blocks.ShipConfig
	width      int
	height     int
	scale      float64
	sizeFactor float64
	canvas     *image.RGBA
	context    *gg.Context
	lineHeight float64
	showCones  bool
	started    time.Time
}

func VideoSize(options Options, width int) (int, int) {
	evenWidth := evenDimension(width)
	height := float64(evenWidth) * options.ArenaHeight / options.ArenaWidth
	return evenWidth, evenDimension(int(math.Round(height)))
}

func NewRenderer(options Options, width int, showCones bool) (*Renderer, error) {
	videoWidth, videoHeight := VideoSize(options, width)
	parsed, err := opentype.Parse(liberationmonoregular.TTF)
	if err != nil {
		return nil, fmt.Errorf("loading the overlay font: %w", err)
	}
	sizeFactor := math.Max(0.5, float64(videoWidth)/referenceVideoWidth)
	face, err := opentype.NewFace(parsed, &opentype.FaceOptions{Size: baseFontSize * sizeFactor, DPI: 72, Hinting: font.HintingFull})
	if err != nil {
		return nil, fmt.Errorf("sizing the overlay font: %w", err)
	}

	canvas := image.NewRGBA(image.Rect(0, 0, videoWidth, videoHeight))
	context := gg.NewContextForRGBA(canvas)
	context.SetFontFace(face)

	configs := make([]blocks.ShipConfig, len(options.Squads))
	for index, squad := range options.Squads {
		configs[index] = SquadConfig(squad)
	}

	return &Renderer{
		options:    options,
		configs:    configs,
		width:      videoWidth,
		height:     videoHeight,
		scale:      float64(videoWidth) / options.ArenaWidth,
		sizeFactor: sizeFactor,
		canvas:     canvas,
		context:    context,
		lineHeight: float64(face.Metrics().Height.Ceil()) * 1.15,
		showCones:  showCones,
		started:    time.Now(),
	}, nil
}

func (r *Renderer) Size() (int, int) {
	return r.width, r.height
}

func (r *Renderer) Render(snapshot Snapshot, latest Sample) *image.RGBA {
	r.context.SetColor(backgroundColor)
	r.context.Clear()
	r.drawGrid()
	r.drawPlanet()
	if r.showCones {
		r.drawCones(snapshot)
	}
	r.drawDefenders(snapshot)
	r.drawEvaders(snapshot)
	r.drawOverlay(snapshot, latest)
	return r.canvas
}

func (r *Renderer) drawGrid() {
	context := r.context
	context.SetColor(gridColor)
	context.SetLineWidth(1)
	for x := 0.0; x <= r.options.ArenaWidth; x += gridStep {
		context.MoveTo(x*r.scale, 0)
		context.LineTo(x*r.scale, float64(r.height))
	}
	for y := 0.0; y <= r.options.ArenaHeight; y += gridStep {
		context.MoveTo(0, y*r.scale)
		context.LineTo(float64(r.width), y*r.scale)
	}
	context.Stroke()
}

func (r *Renderer) drawPlanet() {
	center := planetCenter(r.options)
	context := r.context
	context.DrawCircle(center.X*r.scale, center.Y*r.scale, r.options.PlanetRadius*r.scale)
	context.SetColor(planetFill)
	context.FillPreserve()
	context.SetColor(planetEdge)
	context.SetLineWidth(2 * r.sizeFactor)
	context.Stroke()
}

func (r *Renderer) drawCones(snapshot Snapshot) {
	context := r.context
	for squadIndex, config := range r.configs {
		halfAngle := config.FovDegrees * math.Pi / 360
		radius := config.ViewDistance * r.scale
		for _, ship := range snapshot.Defenders {
			if ship.Squad != squadIndex {
				continue
			}
			x := ship.X * r.scale
			y := ship.Y * r.scale
			context.MoveTo(x, y)
			context.DrawArc(x, y, radius, ship.Rotation-halfAngle, ship.Rotation+halfAngle)
			context.ClosePath()
		}
		shade := r.options.Squads[squadIndex].Color
		context.SetColor(color.NRGBA{R: shade.R, G: shade.G, B: shade.B, A: coneFillAlpha})
		context.FillPreserve()
		context.SetColor(color.NRGBA{R: shade.R, G: shade.G, B: shade.B, A: coneEdgeAlpha})
		context.SetLineWidth(math.Max(1, r.sizeFactor))
		context.Stroke()
	}
}

func (r *Renderer) drawDefenders(snapshot Snapshot) {
	context := r.context
	for squadIndex, config := range r.configs {
		radius := math.Max(minimumShipRadius, config.HullRadius*r.scale)
		for _, ship := range snapshot.Defenders {
			if ship.Squad == squadIndex {
				context.DrawCircle(ship.X*r.scale, ship.Y*r.scale, radius)
			}
		}
		shade := r.options.Squads[squadIndex].Color
		context.SetColor(shade)
		context.Fill()

		for _, ship := range snapshot.Defenders {
			if ship.Squad != squadIndex {
				continue
			}
			x := ship.X * r.scale
			y := ship.Y * r.scale
			context.MoveTo(x, y)
			context.LineTo(x+math.Cos(ship.Rotation)*radius*headingScale, y+math.Sin(ship.Rotation)*radius*headingScale)
		}
		context.SetLineWidth(math.Max(1, radius*0.45))
		context.Stroke()
	}
}

func (r *Renderer) drawEvaders(snapshot Snapshot) {
	context := r.context
	radius := math.Max(minimumShipRadius, sim.DefaultHullRadius*r.scale)
	for _, ship := range snapshot.Evaders {
		context.DrawCircle(ship.X*r.scale, ship.Y*r.scale, radius)
	}
	context.SetColor(evaderColor)
	context.Fill()
}

func (r *Renderer) drawOverlay(snapshot Snapshot, latest Sample) {
	lines := r.overlayLines(snapshot, latest)
	context := r.context
	swatchSize := r.lineHeight * 0.55

	panelWidth := 0.0
	for _, line := range lines {
		width, _ := context.MeasureString(line.text)
		if line.swatch != nil {
			width += swatchSize + swatchGap
		}
		panelWidth = math.Max(panelWidth, width)
	}
	margin := panelMargin * r.sizeFactor
	padding := panelPadding * r.sizeFactor
	panelWidth += padding * 2
	panelHeight := r.lineHeight*float64(len(lines)) + padding*2

	context.DrawRoundedRectangle(margin, margin, panelWidth, panelHeight, panelRadius*r.sizeFactor)
	context.SetColor(panelColor)
	context.Fill()

	ascent := r.lineHeight * 0.75
	for index, line := range lines {
		x := margin + padding
		baseline := margin + padding + r.lineHeight*float64(index) + ascent
		if line.swatch != nil {
			context.DrawRectangle(x, baseline-swatchSize, swatchSize, swatchSize)
			context.SetColor(line.swatch)
			context.Fill()
			x += swatchSize + swatchGap
		}
		context.SetColor(line.shade)
		context.DrawString(line.text, x, baseline)
	}
}

func (r *Renderer) overlayLines(snapshot Snapshot, latest Sample) []overlayLine {
	options := r.options
	totals := snapshot.Totals
	lines := []overlayLine{
		{text: fmt.Sprintf("ASTROSWARM STRESS TEST  %d ships", options.Ships), shade: panelTextColor},
		{text: fmt.Sprintf("sim %6.1f / %.0f s   wall %s", snapshot.SimSeconds, options.DurationSeconds, clock(time.Since(r.started))), shade: panelDimColor},
	}

	if latest.SimFPS > 0 {
		speedShade := panelGoodColor
		if latest.RealtimeFactor < 1 {
			speedShade = panelWarnColor
		}
		lines = append(lines,
			overlayLine{text: fmt.Sprintf("sim fps %7.0f   target %d   realtime %.1fx", latest.SimFPS, options.TickRate, latest.RealtimeFactor), shade: speedShade},
			overlayLine{text: fmt.Sprintf("tick %6.2f ms avg   %6.2f ms max", latest.TickMeanMs, latest.TickMaxMs), shade: panelTextColor},
			overlayLine{text: fmt.Sprintf("render fps %5.1f   video %d fps   frame %d", latest.RenderFPS, options.VideoFPS, latest.FramesRendered), shade: panelTextColor},
			overlayLine{text: fmt.Sprintf("heap %6.1f MB   gc %d   world %d ships", latest.HeapMB, latest.GCCycles, latest.ShipsInWorld), shade: panelDimColor},
		)
	} else {
		lines = append(lines, overlayLine{text: "measuring performance...", shade: panelDimColor})
	}

	fleet := fmt.Sprintf("defenders %d   evaders %d   breaches %d", totals.AliveDefenders(), totals.EvadersInFlight, totals.Breaches)
	if offScreen := r.offScreenCount(snapshot); offScreen > 0 {
		fleet += fmt.Sprintf("   off screen %d", offScreen)
	}
	lines = append(lines, overlayLine{text: fleet, shade: panelTextColor})
	for index, squad := range options.Squads {
		config := r.configs[index]
		lines = append(lines,
			overlayLine{
				text:   fmt.Sprintf("%s  alive %d  captures %d", squad.Name, totals.DefendersAlive[index], totals.Captures[index]),
				shade:  panelTextColor,
				swatch: squad.Color,
			},
			overlayLine{
				text: fmt.Sprintf("speed %.2f m/s  vision %.1f m  fov %.0f deg  turn %.0f deg/s",
					config.Speed/blocks.PixelsPerMeter, config.ViewDistance/blocks.PixelsPerMeter, config.FovDegrees, config.TurnSpeed*180/math.Pi),
				shade: panelDimColor,
			},
		)
	}
	return lines
}

func (r *Renderer) offScreenCount(snapshot Snapshot) int {
	count := 0
	for _, ship := range snapshot.Defenders {
		if r.offScreen(ship) {
			count++
		}
	}
	for _, ship := range snapshot.Evaders {
		if r.offScreen(ship) {
			count++
		}
	}
	return count
}

func (r *Renderer) offScreen(ship ShipView) bool {
	return ship.X < 0 || ship.Y < 0 || ship.X > r.options.ArenaWidth || ship.Y > r.options.ArenaHeight
}

func clock(elapsed time.Duration) string {
	seconds := int(elapsed.Seconds())
	return fmt.Sprintf("%02d:%02d", seconds/60, seconds%60)
}

func evenDimension(value int) int {
	if value < 2 {
		return 2
	}
	if value%2 == 1 {
		return value + 1
	}
	return value
}
