//go:build !darwin && !linux

package lifecycle

// lookupListenerPids has no owner lookup on this platform; every address is unverified.
func lookupListenerPids(int) ([]int, bool) { return nil, false }

func lookupProcessGroup(int) (int, bool) { return 0, false }
