//go:build linux

package worker

import (
	"math"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
)

type cpuSample struct {
	total float64
	idle  float64
	valid bool
}

var (
	cpuMutex sync.Mutex
	lastCPU  cpuSample
)

func collectSystemStats() map[string]any {
	stats := map[string]any{"cpu_count": runtime.NumCPU()}

	if percent, ok := cpuPercent(); ok {
		stats["cpu_percent"] = round(percent, 1)
	}
	if total, available, ok := memoryKB(); ok {
		used := total - available
		stats["memory_total_mb"] = int(math.Round(float64(total) / 1024.0))
		stats["memory_used_mb"] = int(math.Round(float64(used) / 1024.0))
		if total > 0 {
			stats["memory_percent"] = round(100.0*float64(used)/float64(total), 1)
		}
	}
	if total, used, ok := diskBytes("/"); ok {
		stats["disk_total_gb"] = round(float64(total)/(1024*1024*1024), 1)
		stats["disk_used_gb"] = round(float64(used)/(1024*1024*1024), 1)
		if total > 0 {
			stats["disk_percent"] = round(100.0*float64(used)/float64(total), 1)
		}
	}
	if load, ok := loadAverage(); ok {
		stats["load_avg_1m"] = round(load, 2)
	}
	return stats
}

func cpuPercent() (float64, bool) {
	raw, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0, false
	}
	line := ""
	for _, candidate := range strings.Split(string(raw), "\n") {
		if strings.HasPrefix(candidate, "cpu ") {
			line = candidate
			break
		}
	}
	if line == "" {
		return 0, false
	}

	fields := strings.Fields(line)[1:]
	total := 0.0
	idle := 0.0
	for index, field := range fields {
		value, err := strconv.ParseFloat(field, 64)
		if err != nil {
			continue
		}
		total += value
		if index == 3 || index == 4 {
			idle += value
		}
	}

	cpuMutex.Lock()
	previous := lastCPU
	lastCPU = cpuSample{total: total, idle: idle, valid: true}
	cpuMutex.Unlock()

	if !previous.valid {
		return 0, false
	}
	deltaTotal := total - previous.total
	deltaIdle := idle - previous.idle
	if deltaTotal <= 0 {
		return 0, false
	}
	return 100.0 * (deltaTotal - deltaIdle) / deltaTotal, true
}

func memoryKB() (total int64, available int64, ok bool) {
	raw, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, 0, false
	}
	for _, line := range strings.Split(string(raw), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		value, err := strconv.ParseInt(fields[1], 10, 64)
		if err != nil {
			continue
		}
		switch fields[0] {
		case "MemTotal:":
			total = value
		case "MemAvailable:":
			available = value
		}
	}
	return total, available, total > 0
}

func diskBytes(path string) (total uint64, used uint64, ok bool) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, 0, false
	}
	total = stat.Blocks * uint64(stat.Bsize)
	free := stat.Bavail * uint64(stat.Bsize)
	if total < free {
		return 0, 0, false
	}
	return total, total - free, true
}

func loadAverage() (float64, bool) {
	raw, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return 0, false
	}
	fields := strings.Fields(string(raw))
	if len(fields) == 0 {
		return 0, false
	}
	value, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0, false
	}
	return value, true
}

func round(value float64, places int) float64 {
	factor := math.Pow(10, float64(places))
	return math.Round(value*factor) / factor
}
