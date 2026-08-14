package entry

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"astroswarm/worker/internal/bench"
	"astroswarm/worker/internal/blocks"
	"astroswarm/worker/internal/godot"
)

type Placement struct {
	X   float64 `json:"x"`
	Y   float64 `json:"y"`
	Rot float64 `json:"rot"`
}

type SweepPoint struct {
	N           int     `json:"n"`
	SuccessRate float64 `json:"success_rate"`
}

type Results struct {
	Trials         int          `json:"trials"`
	SuccessRate    *float64     `json:"success_rate"`
	DetectionRate  *float64     `json:"detection_rate"`
	CaptureRate    *float64     `json:"capture_rate"`
	Outcomes       []string     `json:"outcomes"`
	DetectionTimes []float64    `json:"detection_times"`
	CaptureTimes   []float64    `json:"capture_times"`
	GoalTimes      []float64    `json:"goal_times"`
	Sweep          []SweepPoint `json:"sweep"`

	SequentialRate      *float64 `json:"sequential_rate"`
	SimultaneousRate    *float64 `json:"simultaneous_rate"`
	EvaderDestroyedRate *float64 `json:"evader_destroyed_rate"`
}

type Entry struct {
	ID            string            `json:"id"`
	Username      string            `json:"username"`
	LevelID       string            `json:"level_id"`
	Status        string            `json:"status"`
	Trials        int               `json:"trials"`
	DefenderCount int               `json:"defender_count"`
	Algorithm     any               `json:"algorithm"`
	Placements    []Placement       `json:"placements"`
	Collisions    bool              `json:"collisions"`
	Results       *Results          `json:"results"`
	SweepIndex    []SweepIndexEntry `json:"sweep_index,omitempty"`
}

func Load(path string) (*Entry, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	parsed := &Entry{}
	if err := json.Unmarshal(raw, parsed); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	return parsed, nil
}

func Fetch(serverURL string, id string) (*Entry, error) {
	endpoint := fmt.Sprintf("%s/api/evaluations/%s", strings.TrimRight(serverURL, "/"), id)
	parsed := &Entry{}
	if err := getJSON(endpoint, parsed); err != nil {
		return nil, err
	}
	return parsed, nil
}

func getJSON(endpoint string, target any) error {
	client := &http.Client{Timeout: 30 * time.Second}
	response, err := client.Get(endpoint)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("%s returned %d: %s", endpoint, response.StatusCode, strings.TrimSpace(string(body)))
	}
	if err := json.Unmarshal(body, target); err != nil {
		return fmt.Errorf("parsing response from %s: %w", endpoint, err)
	}
	return nil
}

func (e *Entry) Scripts() []blocks.Script {
	return blocks.NormalizeToScripts(e.Algorithm)
}

func (e *Entry) BenchPlacements() []bench.Placement {
	placements := make([]bench.Placement, 0, len(e.Placements))
	for _, item := range e.Placements {
		placements = append(placements, bench.Placement{
			Position: godot.Vec{X: item.X, Y: item.Y},
			Rotation: item.Rot,
		})
	}
	return placements
}

func (e *Entry) Label() string {
	parts := []string{}
	if e.Username != "" {
		parts = append(parts, e.Username)
	}
	if e.LevelID != "" {
		parts = append(parts, e.LevelID)
	}
	if e.ID != "" {
		parts = append(parts, e.ID)
	}
	if len(parts) == 0 {
		return "entry"
	}
	return strings.Join(parts, " / ")
}
