//go:build !linux && !darwin && !windows

package comms

import "context"

// systemResourceUsage returns OS-level CPU percent, memory used (bytes) and disk used (bytes).
// This is a stub for unsupported platforms.
func systemResourceUsage(_ context.Context) (cpuPct float64, memUsed uint64, diskUsed uint64) {
	return 0, 0, 0
}
