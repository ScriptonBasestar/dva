//go:build unix

package compare

import (
	"context"
	"net"
	"syscall"
	"testing"
)

// TestPortBusySeesWildcardListenerBoundWithoutReuse pins the macOS hole a bind-only probe has:
// Go binds with SO_REUSEADDR, which on BSD lets 127.0.0.1:port bind while another process holds
// 0.0.0.0:port without it. That process is an ordinary dev server (python -m http.server), and
// dva must not hand its port to a variant.
func TestPortBusySeesWildcardListenerBoundWithoutReuse(t *testing.T) {
	lc := net.ListenConfig{Control: func(_, _ string, c syscall.RawConn) error {
		var serr error
		if err := c.Control(func(fd uintptr) {
			serr = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 0)
		}); err != nil {
			return err
		}
		return serr
	}}
	l, err := lc.Listen(context.Background(), "tcp4", "0.0.0.0:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = l.Close() }()
	port := l.Addr().(*net.TCPAddr).Port
	busy, err := PortBusy(port)
	if err != nil || !busy {
		t.Fatalf("a wildcard listener without SO_REUSEADDR on %d was not seen: busy=%v err=%v", port, busy, err)
	}
}
