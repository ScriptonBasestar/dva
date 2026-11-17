//go:build !darwin && !linux && !freebsd && !netbsd && !openbsd

package cimetrics

import (
	"errors"
	"os"
)

var errUnsupportedStorage = errors.New("private metrics storage requires Unix file locking")

func ownedByUser(os.FileInfo) bool            { return false }
func openExclusive(string) (*os.File, error)  { return nil, errUnsupportedStorage }
func openNoFollow(string) (*os.File, error)   { return nil, errUnsupportedStorage }
func openNoFollowRW(string) (*os.File, error) { return nil, errUnsupportedStorage }
func tryLockFile(*os.File) (bool, error)      { return false, errUnsupportedStorage }
func unlockFile(*os.File)                     {}
func lockRetention(string) (*os.File, error)  { return nil, errUnsupportedStorage }
func unlockRetention(*os.File)                {}
