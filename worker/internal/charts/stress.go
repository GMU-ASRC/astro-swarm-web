package charts

import (
	"fmt"
	"image/color"
	"math"
	"os"
	"path/filepath"
	"sort"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"

	"astroswarm/worker/internal/stress"
)

var (
	stressMaxColor    = color.RGBA{R: 0x9c, G: 0x6a, B: 0xde, A: 0xff}
	stressRenderColor = color.RGBA{R: 0x52, G: 0x51, B: 0x4e, A: 0xff}
)

type StressInput struct {
	Result      stress.Result
	SquadColors []color.RGBA
	Subtitle    string
}

func WriteStress(directory string, input StressInput) ([]string, error) {
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return nil, err
	}
	if len(input.Result.Samples) == 0 {
		return nil, nil
	}

	steps := []struct {
		name  string
		build func(StressInput) (*plot.Plot, error)
	}{
		{"stress_tick_time.png", stressTickTime},
		{"stress_sim_fps.png", stressSimulationSpeed},
		{"stress_tick_vs_ships.png", stressTickVersusShips},
		{"stress_ships_alive.png", stressShipsAlive},
		{"stress_captures.png", stressCaptures},
		{"stress_memory.png", stressMemory},
	}

	written := []string{}
	for _, step := range steps {
		figure, err := step.build(input)
		if err != nil {
			return written, err
		}
		if figure == nil {
			continue
		}
		path := filepath.Join(directory, step.name)
		if err := savePlot(figure, path); err != nil {
			return written, err
		}
		written = append(written, path)
	}
	return written, nil
}

func stressTickTime(input StressInput) (*plot.Plot, error) {
	samples := input.Result.Samples
	p := newPlot("Tick time"+suffix(input.Subtitle), "Simulated time (s)", "Milliseconds per tick")
	mean := samplePoints(samples, func(sample stress.Sample) float64 { return sample.TickMeanMs })
	peak := samplePoints(samples, func(sample stress.Sample) float64 { return sample.TickMaxMs })
	budgetValue := 1000.0 / float64(input.Result.Settings.TickRate)
	budget := flatLine(samples, budgetValue)

	meanLine, err := addSeries(p, mean, simulatedColor)
	if err != nil {
		return nil, err
	}
	peakLine, err := addSeries(p, peak, stressMaxColor)
	if err != nil {
		return nil, err
	}
	budgetLine, err := addDashed(p, budget, secondaryInk)
	if err != nil {
		return nil, err
	}
	p.Legend.Add("Mean tick", meanLine)
	p.Legend.Add("Slowest tick", peakLine)
	p.Legend.Add(fmt.Sprintf("Real-time budget (%.1f ms)", budgetValue), budgetLine)
	marginX(p, mean)
	p.X.Tick.Marker = plot.DefaultTicks{}
	valueAxis(p, math.Max(maxY(peak), budgetValue))
	legendRoom(p)
	return p, nil
}

func stressSimulationSpeed(input StressInput) (*plot.Plot, error) {
	samples := input.Result.Samples
	p := newPlot("Simulation speed"+suffix(input.Subtitle), "Simulated time (s)", "Frames per second")
	simulated := samplePoints(samples, func(sample stress.Sample) float64 { return sample.SimFPS })
	target := flatLine(samples, float64(input.Result.Settings.TickRate))

	simulatedLine, err := addSeries(p, simulated, simulatedColor)
	if err != nil {
		return nil, err
	}
	targetLine, err := addDashed(p, target, secondaryInk)
	if err != nil {
		return nil, err
	}
	p.Legend.Add("Simulation fps", simulatedLine)
	ceiling := math.Max(maxY(simulated), float64(input.Result.Settings.TickRate))

	if input.Result.FramesRendered > 0 {
		rendered := samplePoints(samples, func(sample stress.Sample) float64 { return sample.RenderFPS })
		renderedLine, err := addSeries(p, rendered, stressRenderColor)
		if err != nil {
			return nil, err
		}
		p.Legend.Add("Video render fps", renderedLine)
		ceiling = math.Max(ceiling, maxY(rendered))
	}
	p.Legend.Add(fmt.Sprintf("Real time (%d fps)", input.Result.Settings.TickRate), targetLine)
	marginX(p, simulated)
	p.X.Tick.Marker = plot.DefaultTicks{}
	valueAxis(p, ceiling)
	legendRoom(p)
	return p, nil
}

func stressTickVersusShips(input StressInput) (*plot.Plot, error) {
	samples := input.Result.Samples
	points := make(plotter.XYs, 0, len(samples))
	for _, sample := range samples {
		points = append(points, plotter.XY{X: float64(sample.ShipsInWorld), Y: sample.TickMeanMs})
	}
	sort.Slice(points, func(first, second int) bool { return points[first].X < points[second].X })

	p := newPlot("Tick time by ships in the world"+suffix(input.Subtitle), "Ships in the world", "Mean milliseconds per tick")
	scatter, err := plotter.NewScatter(points)
	if err != nil {
		return nil, err
	}
	scatter.GlyphStyle = draw.GlyphStyle{Color: simulatedColor, Radius: vg.Points(2.5), Shape: draw.CircleGlyph{}}
	p.Add(scatter)
	marginX(p, points)
	valueAxis(p, maxY(points))
	return p, nil
}

func stressShipsAlive(input StressInput) (*plot.Plot, error) {
	samples := input.Result.Samples
	p := newPlot("Ships alive"+suffix(input.Subtitle), "Simulated time (s)", "Ships")
	ceiling := 1.0
	for index, squad := range input.Result.Squads {
		squadIndex := index
		points := samplePoints(samples, func(sample stress.Sample) float64 { return float64(sample.DefendersAlive[squadIndex]) })
		line, err := addSeries(p, points, squadColor(input, index))
		if err != nil {
			return nil, err
		}
		p.Legend.Add(squad.Name+" defenders", line)
		ceiling = math.Max(ceiling, maxY(points))
	}
	evaders := samplePoints(samples, func(sample stress.Sample) float64 { return float64(sample.EvadersInFlight) })
	evaderLine, err := addSeries(p, evaders, riskColor)
	if err != nil {
		return nil, err
	}
	p.Legend.Add("Evaders in flight", evaderLine)
	ceiling = math.Max(ceiling, maxY(evaders))
	marginX(p, evaders)
	p.X.Tick.Marker = plot.DefaultTicks{}
	countAxis(p, ceiling)
	legendRoom(p)
	return p, nil
}

func stressCaptures(input StressInput) (*plot.Plot, error) {
	samples := input.Result.Samples
	p := newPlot("Evaders stopped and breaches"+suffix(input.Subtitle), "Simulated time (s)", "Evaders (cumulative)")
	ceiling := 1.0
	for index, squad := range input.Result.Squads {
		squadIndex := index
		points := samplePoints(samples, func(sample stress.Sample) float64 { return float64(sample.Captures[squadIndex]) })
		line, err := addSeries(p, points, squadColor(input, index))
		if err != nil {
			return nil, err
		}
		p.Legend.Add(squad.Name+" captures", line)
		ceiling = math.Max(ceiling, maxY(points))
	}
	breaches := samplePoints(samples, func(sample stress.Sample) float64 { return float64(sample.Breaches) })
	breachLine, err := addSeries(p, breaches, riskColor)
	if err != nil {
		return nil, err
	}
	p.Legend.Add("Reached the planet", breachLine)
	ceiling = math.Max(ceiling, maxY(breaches))
	marginX(p, breaches)
	p.X.Tick.Marker = plot.DefaultTicks{}
	countAxis(p, ceiling)
	legendRoom(p)
	return p, nil
}

func stressMemory(input StressInput) (*plot.Plot, error) {
	samples := input.Result.Samples
	p := newPlot("Heap memory"+suffix(input.Subtitle), "Simulated time (s)", "Megabytes")
	heap := samplePoints(samples, func(sample stress.Sample) float64 { return sample.HeapMB })
	line, err := addSeries(p, heap, captureColor)
	if err != nil {
		return nil, err
	}
	p.Legend.Add("Heap in use", line)
	marginX(p, heap)
	p.X.Tick.Marker = plot.DefaultTicks{}
	valueAxis(p, maxY(heap))
	legendRoom(p)
	return p, nil
}

func samplePoints(samples []stress.Sample, value func(stress.Sample) float64) plotter.XYs {
	points := make(plotter.XYs, 0, len(samples))
	for _, sample := range samples {
		points = append(points, plotter.XY{X: sample.SimSeconds, Y: value(sample)})
	}
	return points
}

func flatLine(samples []stress.Sample, value float64) plotter.XYs {
	return plotter.XYs{
		{X: samples[0].SimSeconds, Y: value},
		{X: samples[len(samples)-1].SimSeconds, Y: value},
	}
}

func maxY(points plotter.XYs) float64 {
	highest := 0.0
	for _, point := range points {
		highest = math.Max(highest, point.Y)
	}
	return highest
}

func valueAxis(p *plot.Plot, ceiling float64) {
	p.Y.Min = 0
	p.Y.Max = math.Max(ceiling, 0.001) * (1 + countHeadroom)
}

func squadColor(input StressInput, index int) color.RGBA {
	if index < len(input.SquadColors) {
		return input.SquadColors[index]
	}
	return simulatedColor
}
