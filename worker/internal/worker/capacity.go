package worker

import (
	"math"
	"os"
	"runtime"
	"strconv"
	"strings"
)

const defaultJobSlots = 4

type Capacity struct {
	Cores         int
	JobSlots      int
	MatchesPerJob int
}

func planCapacity(settings Settings) Capacity {
	cores := settings.Cores
	if cores < 1 {
		cores = usableCores()
	}
	jobSlots := settings.JobSlots
	if jobSlots < 1 && settings.CoresPerJob > 0 {
		jobSlots = maxInt(1, cores/settings.CoresPerJob)
	}
	if jobSlots < 1 {
		jobSlots = defaultJobSlots
	}
	matchesPerJob := settings.SimWorkers
	if matchesPerJob < 1 {
		matchesPerJob = cores
	}
	return Capacity{Cores: cores, JobSlots: jobSlots, MatchesPerJob: matchesPerJob}
}

func usableCores() int {
	cores := runtime.NumCPU()
	if quota, ok := cgroupCPUQuota(); ok && quota < cores {
		cores = quota
	}
	return maxInt(1, cores)
}

func cgroupCPUQuota() (int, bool) {
	if raw, err := os.ReadFile("/sys/fs/cgroup/cpu.max"); err == nil {
		fields := strings.Fields(string(raw))
		if len(fields) == 2 && fields[0] != "max" {
			return quotaCores(fields[0], fields[1])
		}
		return 0, false
	}
	quota, quotaErr := os.ReadFile("/sys/fs/cgroup/cpu/cpu.cfs_quota_us")
	period, periodErr := os.ReadFile("/sys/fs/cgroup/cpu/cpu.cfs_period_us")
	if quotaErr != nil || periodErr != nil {
		return 0, false
	}
	return quotaCores(strings.TrimSpace(string(quota)), strings.TrimSpace(string(period)))
}

func quotaCores(quotaText string, periodText string) (int, bool) {
	quota, quotaErr := strconv.ParseFloat(quotaText, 64)
	period, periodErr := strconv.ParseFloat(periodText, 64)
	if quotaErr != nil || periodErr != nil || quota <= 0 || period <= 0 {
		return 0, false
	}
	return maxInt(1, int(math.Ceil(quota/period))), true
}

func maxInt(first int, second int) int {
	if first > second {
		return first
	}
	return second
}
