package stress

import (
	"encoding/hex"
	"math"
	"runtime"
	"sort"
	"time"
)

const (
	StopDuration  = "duration"
	StopLineSpent = "every defender was destroyed"

	bytesPerMegabyte = 1024.0 * 1024.0
)

type Hooks struct {
	OnFrame  func(snapshot Snapshot, latest Sample) error
	OnSample func(sample Sample)
}

type Sample struct {
	SimSeconds     float64 `json:"sim_seconds"`
	WallSeconds    float64 `json:"wall_seconds"`
	TickMeanMs     float64 `json:"tick_mean_ms"`
	TickMaxMs      float64 `json:"tick_max_ms"`
	SimFPS         float64 `json:"sim_fps"`
	RealtimeFactor float64 `json:"realtime_factor"`
	RenderFPS      float64 `json:"render_fps"`
	FramesRendered int     `json:"frames_rendered"`
	ShipsInWorld   int     `json:"ships_in_world"`
	HeapMB         float64 `json:"heap_mb"`
	GCCycles       uint32  `json:"gc_cycles"`
	Totals
}

type SquadSummary struct {
	Name         string  `json:"name"`
	Color        string  `json:"color"`
	Ships        int     `json:"ships"`
	Captures     int     `json:"captures"`
	Lost         int     `json:"defenders_lost"`
	Survivors    int     `json:"survivors"`
	ViewDistance float64 `json:"view_distance"`
	FovDegrees   float64 `json:"fov_degrees"`
	Speed        float64 `json:"speed"`
	HullRadius   float64 `json:"hull_radius"`
}

type Settings struct {
	ArenaWidth       float64 `json:"arena_width"`
	ArenaHeight      float64 `json:"arena_height"`
	PlanetRadius     float64 `json:"planet_radius"`
	Spawn            string  `json:"spawn"`
	SpawnInner       float64 `json:"spawn_inner"`
	SpawnOuter       float64 `json:"spawn_outer"`
	SpawnSpacing     float64 `json:"spawn_spacing"`
	RingRadius       float64 `json:"ring_radius"`
	DurationSeconds  float64 `json:"duration_seconds"`
	TickRate         int     `json:"tick_rate"`
	Seed             int64   `json:"seed"`
	WaveInterval     float64 `json:"wave_interval"`
	EvadersPerWave   int     `json:"evaders_per_wave"`
	MaxEvaders       int     `json:"max_evaders"`
	EvaderSpeed      float64 `json:"evader_speed"`
	EvaderSpawn      string  `json:"evader_spawn"`
	EvaderRingRadius float64 `json:"evader_ring_radius"`
	Attrition        bool    `json:"attrition"`
	Collisions       bool    `json:"collisions"`
	KeepInArena      bool    `json:"keep_in_arena"`
	VideoFPS         int     `json:"video_fps"`
}

type Result struct {
	Ships             int            `json:"ships"`
	Settings          Settings       `json:"settings"`
	Squads            []SquadSummary `json:"squads"`
	StopReason        string         `json:"stop_reason"`
	Ticks             int            `json:"ticks"`
	SimulatedSeconds  float64        `json:"simulated_seconds"`
	WallSeconds       float64        `json:"wall_seconds"`
	SimulationSeconds float64        `json:"simulation_seconds"`
	RenderSeconds     float64        `json:"render_seconds"`
	FramesRendered    int            `json:"frames_rendered"`
	SimFPS            float64        `json:"sim_fps"`
	RealtimeFactor    float64        `json:"realtime_factor"`
	TickMeanMs        float64        `json:"tick_mean_ms"`
	TickP50Ms         float64        `json:"tick_p50_ms"`
	TickP95Ms         float64        `json:"tick_p95_ms"`
	TickP99Ms         float64        `json:"tick_p99_ms"`
	TickMaxMs         float64        `json:"tick_max_ms"`
	PeakHeapMB        float64        `json:"peak_heap_mb"`
	Totals            Totals         `json:"totals"`
	Samples           []Sample       `json:"samples"`
}

type recorder struct {
	options          Options
	started          time.Time
	windowTicks      int
	windowTickTime   time.Duration
	windowTickMax    time.Duration
	windowFrames     int
	windowRenderTime time.Duration
	totalTickTime    time.Duration
	totalRenderTime  time.Duration
	frames           int
	tickMilliseconds []float64
	samples          []Sample
	latest           Sample
	peakHeapMB       float64
}

func Run(options Options, hooks Hooks) (Result, error) {
	if err := options.Validate(); err != nil {
		return Result{}, err
	}

	simulation := newSimulation(options)
	delta := 1.0 / float64(options.TickRate)
	totalTicks := int(math.Ceil(options.DurationSeconds * float64(options.TickRate)))
	sampleTicks := maxInt(1, int(math.Round(options.SampleSeconds*float64(options.TickRate))))
	waveTicks := maxInt(1, int(math.Round(options.WaveInterval*float64(options.TickRate))))
	framesEnabled := hooks.OnFrame != nil && options.VideoFPS > 0
	frameTicks := 0.0
	if framesEnabled {
		frameTicks = float64(options.TickRate) / float64(options.VideoFPS)
	}

	record := &recorder{options: options, started: time.Now(), tickMilliseconds: make([]float64, 0, totalTicks)}
	simulation.launchWave()

	stopReason := StopDuration
	nextFrameTick := 0.0
	tick := 0
	for tick < totalTicks {
		if framesEnabled && float64(tick) >= nextFrameTick {
			renderStarted := time.Now()
			if err := hooks.OnFrame(simulation.snapshot(tick), record.latest); err != nil {
				return record.result(simulation, tick, stopReason), err
			}
			record.addFrame(time.Since(renderStarted))
			nextFrameTick += frameTicks
		}

		tickStarted := time.Now()
		if tick > 0 && tick%waveTicks == 0 {
			simulation.launchWave()
		}
		simulation.world.Step(delta)
		simulation.resolveContacts()
		record.addTick(time.Since(tickStarted))
		tick++

		if tick%sampleTicks == 0 {
			sample := record.takeSample(simulation, tick)
			if hooks.OnSample != nil {
				hooks.OnSample(sample)
			}
		}
		if len(simulation.defenders) == 0 {
			stopReason = StopLineSpent
			break
		}
	}

	if record.windowTicks > 0 {
		sample := record.takeSample(simulation, tick)
		if hooks.OnSample != nil {
			hooks.OnSample(sample)
		}
	}
	return record.result(simulation, tick, stopReason), nil
}

func (r *recorder) addTick(duration time.Duration) {
	r.windowTicks++
	r.windowTickTime += duration
	r.totalTickTime += duration
	if duration > r.windowTickMax {
		r.windowTickMax = duration
	}
	r.tickMilliseconds = append(r.tickMilliseconds, milliseconds(duration))
}

func (r *recorder) addFrame(duration time.Duration) {
	r.frames++
	r.windowFrames++
	r.windowRenderTime += duration
	r.totalRenderTime += duration
}

func (r *recorder) takeSample(simulation *swarmSimulation, tick int) Sample {
	var memory runtime.MemStats
	runtime.ReadMemStats(&memory)
	heapMB := float64(memory.HeapAlloc) / bytesPerMegabyte
	r.peakHeapMB = math.Max(r.peakHeapMB, heapMB)

	sample := Sample{
		SimSeconds:     float64(tick) / float64(r.options.TickRate),
		WallSeconds:    time.Since(r.started).Seconds(),
		TickMaxMs:      milliseconds(r.windowTickMax),
		FramesRendered: r.frames,
		ShipsInWorld:   len(simulation.world.Ships),
		HeapMB:         heapMB,
		GCCycles:       memory.NumGC,
		Totals:         simulation.totals.clone(),
	}
	if r.windowTicks > 0 {
		sample.TickMeanMs = milliseconds(r.windowTickTime) / float64(r.windowTicks)
	}
	if r.windowTickTime > 0 {
		sample.SimFPS = float64(r.windowTicks) / r.windowTickTime.Seconds()
		sample.RealtimeFactor = sample.SimFPS / float64(r.options.TickRate)
	}
	if r.windowRenderTime > 0 {
		sample.RenderFPS = float64(r.windowFrames) / r.windowRenderTime.Seconds()
	}

	r.samples = append(r.samples, sample)
	r.latest = sample
	r.windowTicks = 0
	r.windowTickTime = 0
	r.windowTickMax = 0
	r.windowFrames = 0
	r.windowRenderTime = 0
	return sample
}

func (r *recorder) result(simulation *swarmSimulation, tick int, stopReason string) Result {
	options := r.options
	result := Result{
		Ships: options.Ships,
		Settings: Settings{
			ArenaWidth:       options.ArenaWidth,
			ArenaHeight:      options.ArenaHeight,
			PlanetRadius:     options.PlanetRadius,
			Spawn:            options.Spawn,
			SpawnInner:       options.SpawnInner,
			SpawnOuter:       options.SpawnOuter,
			SpawnSpacing:     options.SpawnSpacing,
			RingRadius:       options.RingRadius,
			DurationSeconds:  options.DurationSeconds,
			TickRate:         options.TickRate,
			Seed:             options.Seed,
			WaveInterval:     options.WaveInterval,
			EvadersPerWave:   options.EvadersPerWave,
			MaxEvaders:       options.MaxEvaders,
			EvaderSpeed:      options.EvaderSpeed,
			EvaderSpawn:      options.EvaderSpawn,
			EvaderRingRadius: options.EvaderRingRadius,
			Attrition:        options.Attrition,
			Collisions:       options.Collisions,
			KeepInArena:      options.KeepInArena,
			VideoFPS:         options.VideoFPS,
		},
		StopReason:        stopReason,
		Ticks:             tick,
		SimulatedSeconds:  float64(tick) / float64(options.TickRate),
		WallSeconds:       time.Since(r.started).Seconds(),
		SimulationSeconds: r.totalTickTime.Seconds(),
		RenderSeconds:     r.totalRenderTime.Seconds(),
		FramesRendered:    r.frames,
		PeakHeapMB:        r.peakHeapMB,
		Totals:            simulation.totals.clone(),
		Samples:           r.samples,
	}
	if r.totalTickTime > 0 {
		result.SimFPS = float64(tick) / r.totalTickTime.Seconds()
		result.RealtimeFactor = result.SimFPS / float64(options.TickRate)
	}
	if tick > 0 {
		result.TickMeanMs = milliseconds(r.totalTickTime) / float64(tick)
	}

	sorted := append([]float64(nil), r.tickMilliseconds...)
	sort.Float64s(sorted)
	result.TickP50Ms = percentile(sorted, 0.50)
	result.TickP95Ms = percentile(sorted, 0.95)
	result.TickP99Ms = percentile(sorted, 0.99)
	if len(sorted) > 0 {
		result.TickMaxMs = sorted[len(sorted)-1]
	}

	squadCount := len(options.Squads)
	for index, squad := range options.Squads {
		config := SquadConfig(squad)
		ships := options.Ships / squadCount
		if index < options.Ships%squadCount {
			ships++
		}
		result.Squads = append(result.Squads, SquadSummary{
			Name:         squad.Name,
			Color:        "#" + hex.EncodeToString([]byte{squad.Color.R, squad.Color.G, squad.Color.B}),
			Ships:        ships,
			Captures:     simulation.totals.Captures[index],
			Lost:         simulation.totals.DefendersLost[index],
			Survivors:    simulation.totals.DefendersAlive[index],
			ViewDistance: config.ViewDistance,
			FovDegrees:   config.FovDegrees,
			Speed:        config.Speed,
			HullRadius:   config.HullRadius,
		})
	}
	return result
}

func percentile(sorted []float64, fraction float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	index := int(math.Ceil(fraction*float64(len(sorted)))) - 1
	if index < 0 {
		index = 0
	}
	return sorted[index]
}

func milliseconds(duration time.Duration) float64 {
	return float64(duration.Nanoseconds()) / 1e6
}

func maxInt(first, second int) int {
	if first > second {
		return first
	}
	return second
}
