//go:build darwin || linux

package cirun

import (
	"math"
	"os"
	"syscall"
)

func observeOSWaitResources(state *os.ProcessState) *OSWaitResources {
	if state == nil {
		return nil
	}
	usage, ok := state.SysUsage().(*syscall.Rusage)
	if !ok || usage == nil || usage.Maxrss < 0 {
		return &OSWaitResources{CollectionStatus: "unavailable"}
	}
	maxRSS := int64(usage.Maxrss)
	if maxRSS > math.MaxInt64/bytesPerRusageMaxRSSUnit {
		return &OSWaitResources{CollectionStatus: "unavailable"}
	}
	maxRSS *= bytesPerRusageMaxRSSUnit
	return &OSWaitResources{
		CollectionStatus: "available",
		UserCPU:          state.UserTime().String(),
		SystemCPU:        state.SystemTime().String(),
		PeakRSSBytes:     &maxRSS,
	}
}
