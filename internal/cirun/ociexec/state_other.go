//go:build !darwin && !linux

package ociexec

import "fmt"

func secureStateDirectory(string) (string, error) {
	return "", fmt.Errorf("OCI state directories are unsupported on this platform")
}
