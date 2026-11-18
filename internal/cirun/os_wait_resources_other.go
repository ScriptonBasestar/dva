//go:build !darwin && !linux

package cirun

import "os"

func observeOSWaitResources(*os.ProcessState) *OSWaitResources {
	return &OSWaitResources{CollectionStatus: "unavailable"}
}
