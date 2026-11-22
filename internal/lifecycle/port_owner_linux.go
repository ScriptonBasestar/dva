//go:build linux

package lifecycle

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// tcpListenState is the st column value of a listening socket in /proc/net/tcp{,6}.
const tcpListenState = "0A"

// lookupListenerPids finds the socket inodes listening on port in /proc/net/tcp and tcp6,
// then the pids whose fds point at them. A process this user cannot inspect leaves its fds
// unreadable; finding inodes without any owning pid is therefore unverified, not "free".
func lookupListenerPids(port int) ([]int, bool) {
	inodes := map[string]bool{}
	readable := false
	for _, table := range []string{"/proc/net/tcp", "/proc/net/tcp6"} {
		found, ok := listeningInodes(table, port)
		if !ok {
			continue
		}
		readable = true
		for inode := range found {
			inodes[inode] = true
		}
	}
	if !readable {
		return nil, false
	}
	if len(inodes) == 0 {
		return nil, true
	}
	procDirs, err := filepath.Glob("/proc/[0-9]*")
	if err != nil {
		return nil, false
	}
	var pids []int
	for _, dir := range procDirs {
		pid, err := strconv.Atoi(filepath.Base(dir))
		if err != nil {
			continue
		}
		fds, err := os.ReadDir(filepath.Join(dir, "fd"))
		if err != nil {
			continue
		}
		for _, fd := range fds {
			link, err := os.Readlink(filepath.Join(dir, "fd", fd.Name()))
			if err != nil || !strings.HasPrefix(link, "socket:[") {
				continue
			}
			if inodes[strings.TrimSuffix(strings.TrimPrefix(link, "socket:["), "]")] {
				pids = append(pids, pid)
				break
			}
		}
	}
	if len(pids) == 0 {
		return nil, false
	}
	return pids, true
}

// listeningInodes parses one /proc/net/tcp table: "sl local_address rem_address st ... inode".
func listeningInodes(path string, port int) (map[string]bool, bool) {
	file, err := os.Open(path)
	if err != nil {
		return nil, false
	}
	defer file.Close()
	want := strings.ToUpper(strconv.FormatInt(int64(port), 16))
	for len(want) < 4 {
		want = "0" + want
	}
	found := map[string]bool{}
	scanner := bufio.NewScanner(file)
	scanner.Scan() // header
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 10 || fields[3] != tcpListenState {
			continue
		}
		local := fields[1]
		if idx := strings.LastIndex(local, ":"); idx < 0 || local[idx+1:] != want {
			continue
		}
		found[fields[9]] = true
	}
	return found, scanner.Err() == nil
}

func lookupProcessGroup(pid int) (int, bool) {
	group, err := syscall.Getpgid(pid)
	return group, err == nil
}
