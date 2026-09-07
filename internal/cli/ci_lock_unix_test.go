//go:build darwin || linux || freebsd || netbsd || openbsd

package cli

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

const ciLockHelperEnv = "DVA_CLI_CI_LOCK_HELPER"
const ciLockHolderID = "aabbccddeeff00112233445566778899"

// TestCIRootLockHelperProcess holds a real lock in a fresh process. A
// separate process exercises the same OS admission boundary as independently
// invoked dva binaries.
func TestCIRootLockHelperProcess(t *testing.T) {
	if os.Getenv(ciLockHelperEnv) != "1" {
		return
	}
	root, readyFD := os.Getenv("DVA_CLI_CI_LOCK_ROOT"), 3
	f, err := openAndLockCIRoot(root)
	if err != nil {
		fmt.Fprintf(os.Stdout, "ERROR %v\n", err)
		os.Exit(2)
	}
	defer func() {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		_ = f.Close()
	}()
	fmt.Fprintln(os.Stdout, "READY")
	_, _ = os.NewFile(uintptr(readyFD), "release").Read(make([]byte, 1))
	os.Exit(0)
}

type ciRootLockHolder struct {
	id      string
	cmd     *exec.Cmd
	release func()
}

func holdCIRootLock(t *testing.T, root string) *ciRootLockHolder {
	t.Helper()
	releaseR, releaseW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	readyR, readyW, err := os.Pipe()
	if err != nil {
		_ = releaseR.Close()
		_ = releaseW.Close()
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	t.Cleanup(cancel)
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestCIRootLockHelperProcess$")
	cmd.ExtraFiles = []*os.File{releaseR}
	cmd.Stdout, cmd.Stderr = readyW, os.Stderr
	cmd.Env = append(os.Environ(), ciLockHelperEnv+"=1", "DVA_CLI_CI_LOCK_ROOT="+root)
	if err := cmd.Start(); err != nil {
		_ = releaseR.Close()
		_ = releaseW.Close()
		_ = readyR.Close()
		_ = readyW.Close()
		t.Fatal(err)
	}
	_ = releaseR.Close()
	_ = readyW.Close()
	line, err := bufio.NewReader(readyR).ReadString('\n')
	_ = readyR.Close()
	if err != nil || strings.TrimSpace(line) != "READY" {
		_ = releaseW.Close()
		_ = cmd.Wait()
		t.Fatalf("CI lock helper was not ready: line=%q err=%v", line, err)
	}
	h := &ciRootLockHolder{id: ciLockHolderID, cmd: cmd}
	h.release = func() {
		if releaseW != nil {
			_ = releaseW.Close()
			releaseW = nil
		}
		if h.cmd != nil {
			_ = h.cmd.Wait()
			h.cmd = nil
		}
	}
	t.Cleanup(h.release)
	return h
}

func openAndLockCIRoot(root string) (*os.File, error) {
	u, err := user.Current()
	if err != nil {
		return nil, err
	}
	canonical, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, err
	}
	canonical, err = filepath.Abs(canonical)
	if err != nil {
		return nil, err
	}
	hash := sha256.Sum256([]byte(canonical))
	path := filepath.Join(u.HomeDir, ".local", "state", "dva", "ci", "locks", "root-"+hex.EncodeToString(hash[:])+".lock")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = f.Close()
		return nil, err
	}
	// Match cirun's occupancy metadata so conflict diagnostics can identify it.
	if err = f.Truncate(0); err == nil {
		_, err = f.WriteAt([]byte(ciLockHolderID+"\n"), 0)
	}
	if err != nil {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		_ = f.Close()
		return nil, err
	}
	return f, nil
}
