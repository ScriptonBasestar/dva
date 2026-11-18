//go:build !darwin && !linux

package cirun

import (
	"errors"
	"os"
)

var errNativeStageUnsupported = errors.New("secure native-stage collection unsupported")

func nativeStagePlatformUnsupported(error) bool            { return true }
func openNativeEventsDir(string, string) (*os.File, error) { return nil, errNativeStageUnsupported }
func readNativeEvent(*os.File, string) ([]byte, error)     { return nil, errNativeStageUnsupported }
