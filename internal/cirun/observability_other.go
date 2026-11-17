//go:build !darwin && !linux

package cirun

import "os"

func ownedByCurrentUser(os.FileInfo) bool { return false }
