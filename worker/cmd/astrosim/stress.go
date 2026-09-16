package main

import (
	"flag"
	"fmt"
	"image/color"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"astroswarm/worker/internal/bench"
	"astroswarm/worker/internal/blocks"
	"astroswarm/worker/internal/charts"
	"astroswarm/worker/internal/entry"
	"astroswarm/worker/internal/stress"
)

const (
	stressProgressInterval = 250 * time.Millisecond
	defaultVideoWidth      = 1920
	defaultVideoCodec      = "libx264"
)

var (
	levelOneSquadColor  = color.RGBA{R: 0x2a, G: 0x78, B: 0xd6, A: 0xff}
	levelFourSquadColor = color.RGBA{R: 0xeb, G: 0x68, B: 0x34, A: 0xff}
	stressLevels        = map[int]color.RGBA{1: levelOneSquadColor, 4: levelFourSquadColor}
)

type StressCommandOptions struct {
	Entry          string
	Server         string
	Output         string
	Ships          int
	ArenaWidth     float64
	ArenaHeight    float64
	PlanetRadius   float64
	Spawn          string
	SpawnInner     float64
	SpawnOuter     float64
	SpawnSpacing   float64
	RingRadius     float64
	Seconds        float64
	TickRate       int
	Seed           int64
	WaveInterval   float64
	EvadersPerWave int
	MaxEvaders     int
	EvaderSpeed    float64
	EvaderSpawn    string
	EvaderRing     float64
	Attrition      bool
	KeepInArena    bool
	Collisions     bool
	SampleSeconds  float64
	VideoFPS       int
	VideoWidth     int
	VisionCones    bool
	SkipVideo      bool
	SkipCharts     bool
	FFmpeg         string
	Codec          string
	Quiet          bool
}

func runStress(arguments []string) error {
	options := StressCommandOptions{}
	flags := flag.NewFlagSet("astrosim stress", flag.ExitOnError)
	flags.StringVar(&options.Entry, "entry", "", "a level 1 or level 4 entry: an id, an entry url, or a local entry json file (can also be given as the first argument)")
	flags.StringVar(&options.Server, "server", DefaultServer, "base url used when an entry is a bare id")
	flags.StringVar(&options.Output, "out", "", "directory for stress.json, the charts and the video (default out/stress-<ships>-<time>)")
	flags.IntVar(&options.Ships, "ships", stress.DefaultShips, "defenders to spawn, all flying the entry's algorithm")
	flags.Float64Var(&options.ArenaWidth, "arena-width", stress.DefaultArenaWidth, "arena width in pixels (40 pixels per meter)")
	flags.Float64Var(&options.ArenaHeight, "arena-height", stress.DefaultArenaHeight, "arena height in pixels")
	flags.Float64Var(&options.PlanetRadius, "planet-radius", stress.DefaultPlanetRadius, "radius of the planet at the arena center in pixels")
	flags.StringVar(&options.Spawn, "spawn", stress.SpawnArea, "defender layout: area places each ship at random in the band around the planet like the benchmark scatter, arena spreads them over the whole arena, ring spaces them evenly on a ring")
	flags.Float64Var(&options.SpawnInner, "spawn-inner", stress.DefaultSpawnInner, "inner radius of the spawn band in pixels from the planet center; for -spawn=arena, the radius kept clear")
	flags.Float64Var(&options.SpawnOuter, "spawn-outer", stress.DefaultSpawnOuter, "outer radius of the spawn band in pixels from the planet center")
	flags.Float64Var(&options.SpawnSpacing, "spawn-spacing", stress.DefaultSpawnSpacing, "preferred gap between spawned ships in pixels; a ship is placed anyway after the benchmark's retry limit")
	flags.Float64Var(&options.RingRadius, "ring-radius", stress.DefaultRingRadius, "ring radius in pixels for -spawn=ring")
	flags.Float64Var(&options.Seconds, "seconds", stress.DefaultDurationSeconds, "simulated seconds to run")
	flags.IntVar(&options.TickRate, "tick-rate", stress.DefaultTickRate, "physics ticks per simulated second")
	flags.Int64Var(&options.Seed, "seed", stress.DefaultSeed, "seed for the layout, the evader bearings and the ships' random walks")
	flags.Float64Var(&options.WaveInterval, "wave-interval", stress.DefaultWaveInterval, "simulated seconds between evader waves")
	flags.IntVar(&options.EvadersPerWave, "evaders-per-wave", stress.DefaultEvadersPerWave, "evaders launched from the arena edges each wave")
	flags.IntVar(&options.MaxEvaders, "max-evaders", stress.DefaultMaxEvaders, "most evaders allowed in flight at once")
	flags.Float64Var(&options.EvaderSpeed, "evader-speed", stress.DefaultEvaderSpeed, "evader speed in pixels per second")
	flags.StringVar(&options.EvaderSpawn, "evader-spawn", stress.EvaderSpawnRing, "where evaders launch: ring around the planet like the benchmark, or edge of the arena")
	flags.Float64Var(&options.EvaderRing, "evader-ring-radius", stress.DefaultEvaderRingRadius, "radius in pixels of the evader launch ring")
	flags.BoolVar(&options.Attrition, "attrition", false, "a capture destroys the defender that made it (default: on for a level 4 entry, off for a level 1 entry)")
	flags.BoolVar(&options.Collisions, "collisions", false, "enable ship and planet collisions")
	flags.BoolVar(&options.KeepInArena, "keep-in-arena", false, "hold ships inside the arena instead of letting them fly off past the edge")
	flags.Float64Var(&options.SampleSeconds, "sample-seconds", stress.DefaultSampleSeconds, "simulated seconds between performance samples")
	flags.IntVar(&options.VideoFPS, "video-fps", stress.DefaultVideoFPS, "frames per second of the video")
	flags.IntVar(&options.VideoWidth, "video-width", defaultVideoWidth, "video width in pixels; the height follows the arena's shape")
	flags.BoolVar(&options.VisionCones, "cones", true, "draw every defender's vision cone in the video (-cones=false hides them)")
	flags.BoolVar(&options.SkipVideo, "no-video", false, "skip rendering the video")
	flags.BoolVar(&options.SkipCharts, "no-charts", false, "skip writing the png charts")
	flags.StringVar(&options.FFmpeg, "ffmpeg", "ffmpeg", "ffmpeg binary used to encode the video")
	flags.StringVar(&options.Codec, "codec", defaultVideoCodec, "ffmpeg video codec, for example libx264 or libopenh264")
	flags.BoolVar(&options.Quiet, "quiet", false, "suppress the progress line")
	flags.Usage = stressUsage(flags)

	if err := flags.Parse(arguments); err != nil {
		return err
	}
	for flags.NArg() > 0 {
		if options.Entry != "" {
			return fmt.Errorf("unexpected argument %q, the stress test takes a single entry", flags.Arg(0))
		}
		options.Entry = flags.Arg(0)
		if err := flags.Parse(flags.Args()[1:]); err != nil {
			return err
		}
	}
	if options.Entry == "" {
		flags.Usage()
		return fmt.Errorf("a level 1 or level 4 entry is required")
	}

	explicit := map[string]bool{}
	flags.Visit(func(set *flag.Flag) { explicit[set.Name] = true })

	server := strings.TrimRight(options.Server, "/")
	loaded, level, err := loadStressEntry(options.Entry, server)
	if err != nil {
		return err
	}
	if !explicit["attrition"] {
		options.Attrition = level == 4
	}
	squadColor := stressLevels[level]

	simulationOptions := stress.Options{
		Squads: []stress.Squad{
			{Name: squadName(level, loaded), Algorithm: loaded.Scripts(), Color: squadColor},
		},
		Ships:            options.Ships,
		ArenaWidth:       options.ArenaWidth,
		ArenaHeight:      options.ArenaHeight,
		PlanetRadius:     options.PlanetRadius,
		Spawn:            options.Spawn,
		SpawnInner:       options.SpawnInner,
		SpawnOuter:       options.SpawnOuter,
		SpawnSpacing:     options.SpawnSpacing,
		RingRadius:       options.RingRadius,
		DurationSeconds:  options.Seconds,
		TickRate:         options.TickRate,
		Seed:             options.Seed,
		WaveInterval:     options.WaveInterval,
		EvadersPerWave:   options.EvadersPerWave,
		MaxEvaders:       options.MaxEvaders,
		EvaderSpeed:      options.EvaderSpeed,
		EvaderSpawn:      options.EvaderSpawn,
		EvaderRingRadius: options.EvaderRing,
		Attrition:        options.Attrition,
		Collisions:       options.Collisions,
		KeepInArena:      options.KeepInArena,
		SampleSeconds:    options.SampleSeconds,
		VideoFPS:         options.VideoFPS,
	}
	if options.SkipVideo {
		simulationOptions.VideoFPS = 0
	}
	if err := simulationOptions.Validate(); err != nil {
		return err
	}

	output := options.Output
	if output == "" {
		output = filepath.Join("out", fmt.Sprintf("stress-%d-%s", options.Ships, time.Now().Format("20060102-150405")))
	}
	if err := os.MkdirAll(output, 0o755); err != nil {
		return err
	}

	hooks := stress.Hooks{}
	var encoder *stress.Encoder
	videoPath := filepath.Join(output, "stress.mp4")
	if simulationOptions.VideoFPS > 0 {
		renderer, err := stress.NewRenderer(simulationOptions, options.VideoWidth, options.VisionCones)
		if err != nil {
			return err
		}
		width, height := renderer.Size()
		encoder, err = stress.StartEncoder(options.FFmpeg, videoPath, options.Codec, width, height, simulationOptions.VideoFPS)
		if err != nil {
			return err
		}
		hooks.OnFrame = func(snapshot stress.Snapshot, latest stress.Sample) error {
			return encoder.WriteFrame(renderer.Render(snapshot, latest))
		}
	}
	if !options.Quiet {
		hooks.OnSample = stressProgressPrinter(simulationOptions)
	}

	printStressHeader(simulationOptions, output)
	printProgram(simulationOptions.Squads[0])
	result, runErr := stress.Run(simulationOptions, hooks)
	if !options.Quiet {
		fmt.Println()
	}
	if encoder != nil {
		if closeErr := encoder.Close(); closeErr != nil && runErr == nil {
			runErr = closeErr
		}
	}
	if runErr != nil {
		return runErr
	}

	printStressSummary(result)

	resultsPath := filepath.Join(output, "stress.json")
	payload := map[string]any{
		"entry":  stressEntryInfo(loaded),
		"result": result,
	}
	if err := writeJSON(resultsPath, payload); err != nil {
		return err
	}
	fmt.Printf("\nwrote %s\n", resultsPath)
	if encoder != nil {
		fmt.Printf("wrote %s\n", videoPath)
	}

	if options.SkipCharts {
		return nil
	}
	written, err := charts.WriteStress(output, charts.StressInput{
		Result:      result,
		SquadColors: []color.RGBA{squadColor},
		Subtitle:    fmt.Sprintf("%d ships", options.Ships),
	})
	for _, path := range written {
		fmt.Printf("wrote %s\n", path)
	}
	return err
}

func stressUsage(flags *flag.FlagSet) func() {
	return func() {
		fmt.Fprintf(os.Stderr, "astrosim stress fills one arena with defenders flying a level 1 or level 4 algorithm\n")
		fmt.Fprintf(os.Stderr, "against waves of evaders, and records charts and a video with live performance stats.\n\n")
		fmt.Fprintf(os.Stderr, "usage:\n")
		fmt.Fprintf(os.Stderr, "  astrosim stress <entry> [flags]\n\n")
		fmt.Fprintf(os.Stderr, "examples:\n")
		fmt.Fprintf(os.Stderr, "  astrosim stress 13569541-180c-4c51-bcfe-d4ab038359af -ships 1000\n")
		fmt.Fprintf(os.Stderr, "  astrosim stress %s/levels/<id> -ships 2000 -arena-width 7680 -arena-height 4320 -no-video\n\n", DefaultServer)
		fmt.Fprintf(os.Stderr, "flags:\n")
		flags.PrintDefaults()
	}
}

func loadStressEntry(value string, server string) (*entry.Entry, int, error) {
	var loaded *entry.Entry
	var err error
	if info, statErr := os.Stat(value); statErr == nil && !info.IsDir() {
		loaded, err = entry.Load(value)
	} else {
		entryServer, id, splitErr := splitTarget(value, server)
		if splitErr != nil {
			return nil, 0, splitErr
		}
		fmt.Printf("fetching entry %s from %s\n", id, entryServer)
		loaded, err = entry.Fetch(entryServer, id)
	}
	if err != nil {
		return nil, 0, fmt.Errorf("loading the entry: %w", err)
	}

	level := bench.LevelNumber(loaded.LevelID)
	if _, supported := stressLevels[level]; !supported {
		return nil, 0, fmt.Errorf("the stress test takes a level 1 or level 4 entry, but %s is a level %d entry", loaded.Label(), level)
	}
	if len(loaded.Scripts()) == 0 {
		return nil, 0, fmt.Errorf("the entry %s carries no algorithm blocks", loaded.Label())
	}
	return loaded, level, nil
}

func squadName(level int, loaded *entry.Entry) string {
	if loaded.Username == "" {
		return fmt.Sprintf("Level %d", level)
	}
	return fmt.Sprintf("Level %d (%s)", level, loaded.Username)
}

func stressEntryInfo(loaded *entry.Entry) map[string]any {
	return map[string]any{
		"id":       loaded.ID,
		"username": loaded.Username,
		"level_id": loaded.LevelID,
	}
}

func printStressHeader(options stress.Options, output string) {
	fmt.Printf("stress test: %d ships in a %.0f x %.0f arena for %.0f simulated seconds at %d ticks per second\n",
		options.Ships, options.ArenaWidth, options.ArenaHeight, options.DurationSeconds, options.TickRate)
	for _, squad := range options.Squads {
		fmt.Printf("  %s flies %d ships\n", squad.Name, options.Ships/len(options.Squads))
	}
	switch options.Spawn {
	case stress.SpawnArea:
		fmt.Printf("  ships spawn at random %.0f to %.0f px from the planet, %.0f px apart where there is room\n",
			options.SpawnInner, options.SpawnOuter, options.SpawnSpacing)
	case stress.SpawnArena:
		fmt.Printf("  ships spawn at random across the arena, at least %.0f px from the planet\n", options.SpawnInner)
	case stress.SpawnRing:
		fmt.Printf("  ships spawn evenly on a %.0f px ring around the planet\n", options.RingRadius)
	}
	fmt.Printf("  %d evaders every %.1fs, at most %d in flight, attrition %t, collisions %t, seed %d\n",
		options.EvadersPerWave, options.WaveInterval, options.MaxEvaders, options.Attrition, options.Collisions, options.Seed)
	if options.VideoFPS > 0 {
		fmt.Printf("  recording a %d fps video\n", options.VideoFPS)
	}
	fmt.Printf("  writing to %s\n", output)
}

func printProgram(squad stress.Squad) {
	config := stress.SquadConfig(squad)
	fmt.Printf("  every ship: speed %.2f m/s, turn %.0f deg/s, vision %.2f m, fov %.0f deg, hull %.0f px\n",
		config.Speed/blocks.PixelsPerMeter, config.TurnSpeed*180/math.Pi, config.ViewDistance/blocks.PixelsPerMeter, config.FovDegrees, config.HullRadius)
	fmt.Printf("  and runs this program:\n")
	for _, script := range squad.Algorithm {
		printBlocks(script.Blocks, 2)
	}
}

func printBlocks(list []blocks.Block, depth int) {
	for _, block := range list {
		fmt.Printf("%s%s%s\n", strings.Repeat("  ", depth), block.Type, formatBlockParams(block.Params))
		printBlocks(block.Children, depth+1)
	}
}

func formatBlockParams(params map[string]any) string {
	if len(params) == 0 {
		return ""
	}
	keys := make([]string, 0, len(params))
	for key := range params {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%v", key, params[key]))
	}
	return " (" + strings.Join(parts, ", ") + ")"
}

func stressProgressPrinter(options stress.Options) func(stress.Sample) {
	var lastPrinted time.Time
	return func(sample stress.Sample) {
		if time.Since(lastPrinted) < stressProgressInterval {
			return
		}
		lastPrinted = time.Now()
		fmt.Printf("\r  sim %6.1f/%.0fs  %7.0f fps  tick %6.2f ms  defenders %4d  evaders %3d   ",
			sample.SimSeconds, options.DurationSeconds, sample.SimFPS, sample.TickMeanMs, sample.AliveDefenders(), sample.EvadersInFlight)
	}
}

func printStressSummary(result stress.Result) {
	fmt.Printf("\n%d ships, %d ticks (%.1f simulated seconds) in %.1fs of wall time, stopped by %s\n\n",
		result.Ships, result.Ticks, result.SimulatedSeconds, result.WallSeconds, result.StopReason)
	fmt.Printf("  simulation  %.1fs   %.0f fps   %.1fx real time\n", result.SimulationSeconds, result.SimFPS, result.RealtimeFactor)
	fmt.Printf("  tick        mean %.2f ms   p50 %.2f   p95 %.2f   p99 %.2f   max %.2f\n",
		result.TickMeanMs, result.TickP50Ms, result.TickP95Ms, result.TickP99Ms, result.TickMaxMs)
	if result.FramesRendered > 0 && result.RenderSeconds > 0 {
		fmt.Printf("  video       %d frames rendered in %.1fs (%.1f fps)\n",
			result.FramesRendered, result.RenderSeconds, float64(result.FramesRendered)/result.RenderSeconds)
	}
	fmt.Printf("  peak heap   %.1f MB\n", result.PeakHeapMB)
	fmt.Printf("  evaders     %d launched   %d reached the planet   %d still in flight\n",
		result.Totals.Launched, result.Totals.Breaches, result.Totals.EvadersInFlight)
	for _, squad := range result.Squads {
		fmt.Printf("  %-24s %d ships   %d captures   %d lost   %d survived\n",
			squad.Name, squad.Ships, squad.Captures, squad.Lost, squad.Survivors)
	}
}
