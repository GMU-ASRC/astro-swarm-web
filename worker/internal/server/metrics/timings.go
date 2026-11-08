package metrics

import (
	"sort"
	"sync"
	"time"
)

const (
	samplesPerSeries = 500
	maxSeries        = 300
)

type series struct {
	count   int64
	samples []float64
	next    int
	last    time.Time
}

func (current *series) add(milliseconds float64, now time.Time) {
	current.count++
	current.last = now
	if len(current.samples) < samplesPerSeries {
		current.samples = append(current.samples, milliseconds)
		return
	}
	current.samples[current.next] = milliseconds
	current.next = (current.next + 1) % samplesPerSeries
}

type Summary struct {
	Key     string  `json:"key"`
	Count   int64   `json:"count"`
	Average float64 `json:"average_ms"`
	Median  float64 `json:"p50_ms"`
	P95     float64 `json:"p95_ms"`
	Max     float64 `json:"max_ms"`
	LastAt  string  `json:"last_at"`
}

func (current *series) summary(key string) Summary {
	sorted := append([]float64{}, current.samples...)
	sort.Float64s(sorted)
	total := 0.0
	for _, value := range sorted {
		total += value
	}
	return Summary{
		Key: key, Count: current.count, Average: rounded(total / float64(len(sorted))),
		Median: rounded(percentile(sorted, 0.5)), P95: rounded(percentile(sorted, 0.95)),
		Max: rounded(sorted[len(sorted)-1]), LastAt: current.last.UTC().Format(time.RFC3339),
	}
}

func percentile(sorted []float64, fraction float64) float64 {
	index := int(fraction * float64(len(sorted)-1))
	return sorted[index]
}

func rounded(value float64) float64 {
	return float64(int64(value*10+0.5)) / 10
}

type Timings struct {
	lock    sync.Mutex
	entries map[string]*series
	started time.Time
}

func NewTimings() *Timings {
	return &Timings{entries: map[string]*series{}, started: time.Now()}
}

func (timings *Timings) Record(key string, milliseconds float64) {
	now := time.Now()
	timings.lock.Lock()
	defer timings.lock.Unlock()
	current := timings.entries[key]
	if current == nil {
		if len(timings.entries) >= maxSeries {
			return
		}
		current = &series{}
		timings.entries[key] = current
	}
	current.add(milliseconds, now)
}

func (timings *Timings) Summaries() []Summary {
	timings.lock.Lock()
	defer timings.lock.Unlock()
	output := make([]Summary, 0, len(timings.entries))
	for key, current := range timings.entries {
		output = append(output, current.summary(key))
	}
	sort.Slice(output, func(left, right int) bool {
		if output[left].Count != output[right].Count {
			return output[left].Count > output[right].Count
		}
		return output[left].Key < output[right].Key
	})
	return output
}

func (timings *Timings) Since() time.Time {
	return timings.started
}
