//go:build darwin || linux

package cirun

import (
	"errors"
	"golang.org/x/sys/unix"
	"os"
)

func openSummaryState(path string) (*os.File, error) {
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	f := os.NewFile(uintptr(fd), path)
	info, err := f.Stat()
	if err != nil || !info.IsDir() || info.Mode().Perm() != 0700 || !ownedByCurrentUser(info) {
		f.Close()
		return nil, errors.New("CI history state is not a private owned directory")
	}
	return f, nil
}

func openSummaryReceipt(dir *os.File, name string) (*os.File, string, error) {
	fd, err := unix.Openat(int(dir.Fd()), name, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
	if err != nil {
		if errors.Is(err, unix.ELOOP) {
			return nil, "non_regular", err
		}
		return nil, "unreadable", err
	}
	f := os.NewFile(uintptr(fd), name)
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, "unreadable", err
	}
	if !info.Mode().IsRegular() {
		f.Close()
		return nil, "non_regular", errors.New("not regular")
	}
	if info.Mode().Perm() != 0600 || !ownedByCurrentUser(info) {
		f.Close()
		return nil, "untrusted_owner_or_mode", errors.New("receipt is not private and owned")
	}
	if info.Size() > 4<<20 {
		f.Close()
		return nil, "oversized", errors.New("receipt too large")
	}
	return f, "", nil
}
