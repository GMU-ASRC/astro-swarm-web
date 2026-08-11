package bench

import "math"

type TrialRun struct {
	Trial         int     `json:"trial"`
	Outcome       string  `json:"outcome"`
	DetectionTime float64 `json:"detection_time"`
	CaptureTime   float64 `json:"capture_time"`
	GoalTime      float64 `json:"goal_time"`
}

type SweepPoint struct {
	N             int     `json:"n"`
	SuccessRate   float64 `json:"success_rate"`
	DetectionRate float64 `json:"detection_rate"`
	CaptureRate   float64 `json:"capture_rate"`
	WinRate       float64 `json:"win_rate"`
	Trials        int     `json:"trials"`
}

type OutcomeCounts struct {
	Captured      int `json:"captured"`
	ReachedPlanet int `json:"reached_planet"`
	TimedOut      int `json:"timed_out"`
}

type Results struct {
	Trials          int           `json:"trials"`
	SuccessRate     float64       `json:"success_rate"`
	DetectionRate   float64       `json:"detection_rate"`
	CaptureRate     float64       `json:"capture_rate"`
	OutcomeCounts   OutcomeCounts `json:"outcome_counts"`
	MeanGoalSeconds float64       `json:"mean_goal_seconds"`
	GoalSamples     int           `json:"goal_samples"`
	Outcomes        []string      `json:"outcomes"`
	DetectionTimes  []float64     `json:"detection_times"`
	CaptureTimes    []float64     `json:"capture_times"`
	GoalTimes       []float64     `json:"goal_times"`
	Sweep           []SweepPoint  `json:"sweep"`
}

type Report struct {
	LevelID         string     `json:"level_id"`
	Defenders       int        `json:"defenders"`
	ViewDistance    float64    `json:"view"`
	FovDegrees      float64    `json:"fov"`
	Speed           float64    `json:"speed"`
	HullRadius      float64    `json:"hull"`
	Seed            int64      `json:"seed"`
	SweepTrials     int        `json:"sweep_trials"`
	SweepSpawn      string     `json:"sweep_spawn"`
	Collisions      bool       `json:"collisions"`
	SinglePrecision bool       `json:"single_precision"`
	MatchSeconds    float64    `json:"match_seconds"`
	Runs            []TrialRun `json:"runs"`
	Results         Results    `json:"results"`
	DurationSeconds float64    `json:"duration_seconds"`
	MatchesRun      int        `json:"matches_run"`
}

type ReplayRun struct {
	Trial         int                `json:"trial"`
	Outcome       string             `json:"outcome"`
	DetectionTime float64            `json:"detection_time"`
	CaptureTime   float64            `json:"capture_time"`
	GoalTime      float64            `json:"goal_time"`
	FramesPacked  string             `json:"frames_packed"`
	Stats         map[string]float64 `json:"stats,omitempty"`
}

type ReplaySweepRun struct {
	N             int         `json:"n"`
	Outcome       string      `json:"outcome"`
	DetectionTime float64     `json:"detection_time"`
	CaptureTime   float64     `json:"capture_time"`
	GoalTime      float64     `json:"goal_time"`
	DetectionRate float64     `json:"detection_rate"`
	CaptureRate   float64     `json:"capture_rate"`
	Defenders     int         `json:"defenders"`
	TrialRuns     []ReplayRun `json:"trial_runs"`
	FramesPacked  string      `json:"frames_packed"`
}

type JobMeta struct {
	FPS       int   `json:"fps"`
	Defenders int   `json:"defenders"`
	View      int   `json:"view"`
	Fov       int   `json:"fov"`
	Speed     int   `json:"speed"`
	Hull      int   `json:"hull"`
	Planet    []int `json:"planet"`
	Arena     []int `json:"arena"`
}

type JobResult struct {
	Runs      []ReplayRun      `json:"runs"`
	SweepRuns []ReplaySweepRun `json:"sweep_runs"`
	Meta      JobMeta          `json:"meta"`
}

const (
	SweepSpawnFixed  = "fixed"
	SweepSpawnVaried = "varied"
)

func sweepSpawnLabel(varied bool) string {
	if varied {
		return SweepSpawnVaried
	}
	return SweepSpawnFixed
}

func rate(times []float64) float64 {
	if len(times) == 0 {
		return 0.0
	}
	hits := 0
	for _, value := range times {
		if value >= 0.0 {
			hits++
		}
	}
	return round1(100.0 * float64(hits) / float64(len(times)))
}

func round1(value float64) float64 {
	return math.Round(value*10.0) / 10.0
}
