//go:build !darwin && !linux && !freebsd && !netbsd && !openbsd

package main

import (
	"context"
	"os"
	"os/signal"
)

func signalContext() (context.Context, func(), func() os.Signal) {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	return ctx, stop, func() os.Signal { return nil }
}
func childSignal(error) os.Signal { return nil }
func rethrow(os.Signal)           {}
