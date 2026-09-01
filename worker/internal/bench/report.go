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
	Risk          float64 `json:"risk"`
	Trials        int     `json:"trials"`
}

// One rung of the attrition curve: every evader that was launched while the
// line was this size, and how many of them the line stopped.
type AttritionPoint struct {
	Defenders   int     `json:"defenders"`
	Launched    int     `json:"launched"`
	Destroyed   int     `json:"destroyed"`
	CaptureRate float64 `json:"capture_rate"`
	Risk        float64 `json:"risk"`
}

// One ring size's own attrition curve: how the risk moved as a line that
// started at N traded itself down over the sweep trials at that size.
type AttritionSeries struct {
	N      int              `json:"n"`
	Points []AttritionPoint `json:"points"`
}

// One rung of a ring size's run: what the line had achieved and what was left
// of it by the time it had faced this many evaders, averaged over that ring's
// sweep trials. Every level produces these, including the ones without
// attrition, where the line simply never thins.
type SweepProgressPoint struct {
	Faced       int     `json:"faced"`
	CaptureRate float64 `json:"capture_rate"`
	Risk        float64 `json:"risk"`
	Defenders   float64 `json:"defenders"`
	Trials      int     `json:"trials"`
}

type SweepProgressSeries struct {
	N      int                  `json:"n"`
	Points []SweepProgressPoint `json:"points"`
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

	Risk                float64          `json:"risk"`
	EvadersResolved     int              `json:"evaders_resolved"`
	EvadersDestroyed    int              `json:"evaders_destroyed"`
	EvaderDestroyedRate float64          `json:"evader_destroyed_rate"`
	Breaches            int              `json:"breaches"`
	DefendersLost       int              `json:"defenders_lost"`
	TrialsHeldRate      float64          `json:"trials_held_rate"`
	ReplayTrials        int              `json:"replay_trials,omitempty"`
	TrialDestroyed      []int            `json:"trial_destroyed,omitempty"`
	TrialResolved       []int            `json:"trial_resolved,omitempty"`
	TrialBreaches       []int            `json:"trial_breaches,omitempty"`
	TrialLost           []int            `json:"trial_lost,omitempty"`
	Attrition           []AttritionPoint `json:"attrition,omitempty"`

	SweepAttrition []AttritionSeries     `json:"sweep_attrition,omitempty"`
	SweepProgress  []SweepProgressSeries `json:"sweep_progress,omitempty"`
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
	Results   *Results         `json:"results,omitempty"`
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
