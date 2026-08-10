package verify

import (
	"fmt"
	"math"

	"astroswarm/worker/internal/bench"
	"astroswarm/worker/internal/entry"
)

type Comparison struct {
	Metric    string  `json:"metric"`
	Published float64 `json:"published"`
	Simulated float64 `json:"simulated"`
	Delta     float64 `json:"delta"`
	WithinTol bool    `json:"within_tolerance"`
}

type Result struct {
	Tolerance   float64      `json:"tolerance"`
	Passed      bool         `json:"passed"`
	Comparisons []Comparison `json:"comparisons"`
	Notes       []string     `json:"notes"`
}

func Compare(published *entry.Entry, report bench.Report, tolerance float64) Result {
	result := Result{Tolerance: tolerance, Passed: true}
	if published == nil || published.Results == nil {
		result.Notes = append(result.Notes, "entry carries no published results to compare against")
		return result
	}

	source := published.Results
	add := func(metric string, value *float64, simulated float64) {
		if value == nil {
			result.Notes = append(result.Notes, fmt.Sprintf("published entry has no %s", metric))
			return
		}
		comparison := Comparison{
			Metric:    metric,
			Published: *value,
			Simulated: simulated,
			Delta:     simulated - *value,
		}
		comparison.WithinTol = math.Abs(comparison.Delta) <= tolerance
		if !comparison.WithinTol {
			result.Passed = false
		}
		result.Comparisons = append(result.Comparisons, comparison)
	}

	add("success_rate", source.SuccessRate, report.Results.SuccessRate)
	add("detection_rate", source.DetectionRate, report.Results.DetectionRate)
	add("capture_rate", source.CaptureRate, report.Results.CaptureRate)

	if source.Trials != 0 && source.Trials != report.Results.Trials {
		result.Notes = append(result.Notes, fmt.Sprintf(
			"published trial count %d differs from simulated %d", source.Trials, report.Results.Trials))
	}

	simulatedByN := map[int]float64{}
	for _, point := range report.Results.Sweep {
		simulatedByN[point.N] = point.SuccessRate
	}
	mismatched := 0
	worst := 0.0
	for _, point := range source.Sweep {
		simulated, present := simulatedByN[point.N]
		if !present {
			continue
		}
		delta := math.Abs(simulated - point.SuccessRate)
		if delta > worst {
			worst = delta
		}
		if delta > tolerance {
			mismatched++
		}
	}
	comparableSweep := report.SweepSpawn != bench.SweepSpawnVaried
	if len(source.Sweep) > 0 {
		result.Notes = append(result.Notes, fmt.Sprintf(
			"ring sweep: %d of %d points outside tolerance, worst delta %.1f", mismatched, len(source.Sweep), worst))
		if mismatched > 0 && comparableSweep {
			result.Passed = false
		}
	}
	if !comparableSweep {
		result.Notes = append(result.Notes,
			"ring sweep used a per-trial evader spawn while the server uses one fixed spawn, so the sweep curves are not a like-for-like comparison and were not counted in the verdict")
	}

	return result
}
