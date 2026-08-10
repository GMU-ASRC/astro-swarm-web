//go:build !linux

package worker

import "runtime"

func collectSystemStats() map[string]any {
	return map[string]any{"cpu_count": runtime.NumCPU()}
}
