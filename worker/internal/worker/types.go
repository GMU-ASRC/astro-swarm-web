package worker

import (
	"astroswarm/worker/internal/bench"
	"astroswarm/worker/internal/entry"
	"astroswarm/worker/internal/godot"
)

type JobConfig struct {
	Seed            int64   `json:"seed"`
	SweepMax        int     `json:"sweep_max"`
	SweepTrials     int     `json:"sweep_trials"`
	MatchSeconds    float64 `json:"match_seconds"`
	GoalTailSeconds float64 `json:"goal_tail_seconds"`
	EnemyX          float64 `json:"enemy_x"`
	EnemyY          float64 `json:"enemy_y"`
	LevelID         string  `json:"level_id"`
	Collisions      bool    `json:"collisions"`
}

type PilotRun struct {
	Frames        [][]int            `json:"frames"`
	Outcome       string             `json:"outcome"`
	DetectionTime float64            `json:"detection_time"`
	CaptureTime   float64            `json:"capture_time"`
	GoalTime      float64            `json:"goal_time"`
	FPS           int                `json:"fps"`
	Defenders     int                `json:"defenders"`
	View          int                `json:"view"`
	Fov           int                `json:"fov"`
	Speed         int                `json:"speed"`
	Planet        []int              `json:"planet"`
	Arena         []int              `json:"arena"`
	Stats         map[string]float64 `json:"stats"`
}

type Job struct {
	JobID        string            `json:"job_id"`
	EvaluationID string            `json:"evaluation_id"`
	Algorithm    any               `json:"algorithm"`
	Placements   []entry.Placement `json:"placements"`
	Run          *PilotRun         `json:"run"`
	Trials       int               `json:"trials"`
	TotalUnits   int               `json:"total_units"`
	Config       JobConfig         `json:"config"`
}

func (j Job) benchPlacements() []bench.Placement {
	placements := make([]bench.Placement, 0, len(j.Placements))
	for _, item := range j.Placements {
		placements = append(placements, bench.Placement{
			Position: godot.Vec{X: item.X, Y: item.Y},
			Rotation: item.Rot,
		})
	}
	return placements
}

type RegisterRequest struct {
	WorkerID string `json:"worker_id"`
	Name     string `json:"name"`
	Hostname string `json:"hostname"`
}

type RegisterResponse struct {
	Enabled bool `json:"enabled"`
}

type HeartbeatRequest struct {
	WorkerID    string         `json:"worker_id"`
	Status      string         `json:"status"`
	CurrentJob  string         `json:"current_job"`
	SystemStats map[string]any `json:"system_stats"`
}

type HeartbeatResponse struct {
	Enabled bool  `json:"enabled"`
	Known   *bool `json:"known"`
}

type ClaimRequest struct {
	WorkerID string `json:"worker_id"`
}

type ClaimResponse struct {
	Jobs    []Job   `json:"jobs"`
	Enabled *bool   `json:"enabled"`
	Known   *bool   `json:"known"`
}

type ProgressRequest struct {
	WorkerID string `json:"worker_id"`
	Done     int    `json:"done"`
	Stage    string `json:"stage,omitempty"`
}

type ProgressResponse struct {
	Cancel bool `json:"cancel"`
}

type ResultRequest struct {
	WorkerID string            `json:"worker_id"`
	Result   bench.JobResult `json:"result"`
}

type ResultResponse struct {
	Accepted bool `json:"accepted"`
}

type FailRequest struct {
	WorkerID string `json:"worker_id"`
	Error    string `json:"error"`
}
