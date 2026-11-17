//go:build darwin || linux || freebsd || netbsd || openbsd

package main

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"os/signal"
	"sync/atomic"
	"syscall"
)

func signalContext() (context.Context, func(), func() os.Signal) {
	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	var received atomic.Int32
	go func() {
		select {
		case s := <-ch:
			if sig, ok := s.(syscall.Signal); ok {
				received.Store(int32(sig))
			}
			cancel()
		case <-ctx.Done():
		}
	}()
	stop := func() { signal.Stop(ch); cancel() }
	return ctx, stop, func() os.Signal {
		n := received.Load()
		if n == 0 {
			return nil
		}
		return syscall.Signal(n)
	}
}

func childSignal(err error) os.Signal {
	if e, ok := errors.AsType[*exec.ExitError](err); ok {
		if status, ok := e.Sys().(syscall.WaitStatus); ok && status.Signaled() {
			return status.Signal()
		}
	}
	return nil
}

func rethrow(sig os.Signal) {
	signal.Reset(sig)
	if s, ok := sig.(syscall.Signal); ok {
		if err := syscall.Kill(os.Getpid(), s); err != nil {
			os.Exit(128 + int(s))
		}
		// A delivered default signal terminates the process. Do not race that
		// delivery with an immediate numeric exit that discards signal status.
		select {}
	}
}
