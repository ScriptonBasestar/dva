//go:build !darwin && !linux

package ociexec

import (
	"fmt"
	"os"
)

func openInputFile(*os.Root, string) (*os.File, error) {
	return nil, fmt.Errorf("OCI input descriptors are unsupported on this platform")
}
