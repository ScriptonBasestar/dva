//go:build darwin || linux

package cirun

import (
	"errors"
	"io"
	"os"

	"golang.org/x/sys/unix"
)

var errNativeStageUnsupported = errors.New("secure native-stage collection unsupported")

func nativeStagePlatformUnsupported(err error) bool { return errors.Is(err, errNativeStageUnsupported) }

func openNativeEventsDir(parentPath, runID string) (*os.File, error) {
	if !nativeInvocationPattern.MatchString(runID) {
		return nil, errors.New("invalid run id")
	}
	parent, err := openSummaryState(parentPath)
	if err != nil {
		return nil, err
	}
	fd, err := unix.Openat(int(parent.Fd()), runID, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, closeSummaryRejected(parent, err)
	}
	dir := os.NewFile(uintptr(fd), runID)
	if err := parent.Close(); err != nil {
		return nil, closeSummaryRejected(dir, errors.Join(errSummaryClose, err))
	}
	info, err := dir.Stat()
	if err != nil || !info.IsDir() || info.Mode().Perm() != 0700 || !ownedByCurrentUser(info) {
		return nil, closeSummaryRejected(dir, errors.New("native events directory is not private and owned"))
	}
	return dir, nil
}

func readNativeEvent(dir *os.File, name string) ([]byte, error) {
	fd, err := unix.Openat(int(dir.Fd()), name, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(fd), name)
	info, err := file.Stat()
	if err != nil {
		return nil, closeSummaryRejected(file, err)
	}
	if !info.Mode().IsRegular() {
		return nil, closeSummaryRejected(file, errors.New("not regular"))
	}
	if info.Mode().Perm() != 0600 || !ownedByCurrentUser(info) {
		return nil, closeSummaryRejected(file, errors.New("not private and owned"))
	}
	if info.Size() < 0 || info.Size() > maxNativeStageFileBytes {
		return nil, closeSummaryRejected(file, errors.New("oversized"))
	}
	data, readErr := io.ReadAll(io.LimitReader(file, maxNativeStageFileBytes+1))
	closeErr := file.Close()
	if readErr != nil || closeErr != nil {
		return nil, errors.Join(readErr, closeErr)
	}
	if len(data) > maxNativeStageFileBytes {
		return nil, errors.New("oversized")
	}
	return data, nil
}
