//go:build linux

package lifecycle

import (
	"os"
	"path/filepath"
	"testing"
)

// A /proc/net/tcp fixture: only LISTEN (0A) rows on the wanted port count, matched on the
// local port column, not on the remote one or on another state.
func TestListeningInodesFixture(t *testing.T) {
	table := `  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode
   0: 0100007F:0BB8 00000000:0000 0A 00000000:00000000 00:00000000 00000000  1000        0 111 1 0000000000000000 100 0 0 10 0
   1: 0100007F:0BB8 0100007F:D431 01 00000000:00000000 00:00000000 00000000  1000        0 222 1 0000000000000000 20 4 30 10 -1
   2: 0100007F:1F90 00000000:0000 0A 00000000:00000000 00:00000000 00000000  1000        0 333 1 0000000000000000 100 0 0 10 0
   3: 0100007F:D431 0100007F:0BB8 0A 00000000:00000000 00:00000000 00000000  1000        0 444 1 0000000000000000 100 0 0 10 0
`
	path := filepath.Join(t.TempDir(), "tcp")
	if err := os.WriteFile(path, []byte(table), 0o644); err != nil {
		t.Fatal(err)
	}
	found, ok := listeningInodes(path, 3000) // 0x0BB8
	if !ok || len(found) != 1 || !found["111"] {
		t.Fatalf("listeningInodes = %v, %v; want only inode 111", found, ok)
	}
	if _, ok := listeningInodes(filepath.Join(t.TempDir(), "missing"), 3000); ok {
		t.Fatal("missing table reported readable")
	}
}
