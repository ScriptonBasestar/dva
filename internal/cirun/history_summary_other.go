//go:build !darwin && !linux

package cirun

import (
	"errors"
	"os"
)

func openSummaryState(string) (*os.File, error) {
	return nil, errors.New("secure CI history reading unsupported on this platform")
}
func openSummaryReceipt(*os.File, string) (*os.File, string, error) {
	return nil, "unsupported", errors.New("secure CI history reading unsupported on this platform")
}
