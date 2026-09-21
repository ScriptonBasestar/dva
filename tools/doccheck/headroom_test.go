package main

import (
	"strings"
	"testing"
)

func TestIsHeadroom_thresholds(t *testing.T) {
	cases := []struct {
		name   string
		lines  int
		nbytes int
		want   bool
	}{
		{"well under", 10, 100, false},
		{"exactly 80% lines", 400, 100, false},
		{"one line past 80%", 401, 100, true},
		{"exactly 80% bytes", 10, 8192, false},
		{"one byte past 80%", 10, 8193, true},
		{"at hard byte limit", 10, 10240, true},
		{"one byte over limit", 10, 10241, false},
		{"at hard line limit", 500, 100, true},
		{"one line over limit", 501, 100, false},
		{"both over", 501, 10241, false},
	}
	for _, c := range cases {
		if got := isHeadroom(c.lines, c.nbytes); got != c.want {
			t.Errorf("isHeadroom(%d lines, %d bytes)=%v want %v (%s)",
				c.lines, c.nbytes, got, c.want, c.name)
		}
	}
}

func headroomBody(t *testing.T, self string, filler string) string {
	t.Helper()
	return "# T\n\nSee [s](" + self + ").\n" + filler
}

func TestHeadroom_belowThresholdIsSilent(t *testing.T) {
	root := t.TempDir()
	path := "docs/ok.md"
	writeFile(t, root, path, headroomBody(t, "ok.md", "small body\n"))
	inv := mustInventory(t, root, path)

	res := Check(CheckInput{Root: root, Inventory: inv})
	if !res.OK {
		t.Fatalf("expected OK, errors=%v", res.Errors)
	}
	if res.HeadroomDocs != 0 {
		t.Fatalf("headroom_docs=%d want 0", res.HeadroomDocs)
	}
	if res.OversizedDocs != 0 {
		t.Fatalf("oversized_docs=%d want 0", res.OversizedDocs)
	}
}

func TestHeadroom_warnsPast80PercentBytesWithoutFailing(t *testing.T) {
	root := t.TempDir()
	path := "docs/by-bytes.md"
	body := headroomBody(t, "by-bytes.md", strings.Repeat("pad pad pad pad pad pad pad pad pad pad\n", 210))
	if len(body) <= headroomByteWarn || len(body) > maxDocBytes {
		t.Fatalf("fixture bytes=%d want in (%d, %d]", len(body), headroomByteWarn, maxDocBytes)
	}
	if countLines(body) > maxDocLines {
		t.Fatalf("fixture lines=%d want under %d", countLines(body), maxDocLines)
	}
	writeFile(t, root, path, body)
	inv := mustInventory(t, root, path)

	res := Check(CheckInput{Root: root, Inventory: inv})
	if !res.OK {
		t.Fatalf("headroom must not fail the gate, errors=%v", res.Errors)
	}
	if res.HeadroomDocs != 1 {
		t.Fatalf("headroom_docs=%d want 1", res.HeadroomDocs)
	}
	if res.OversizedDocs != 0 {
		t.Fatalf("oversized_docs=%d want 0", res.OversizedDocs)
	}
	if len(res.HeadroomDetail) != 1 || !strings.Contains(res.HeadroomDetail[0], path) {
		t.Fatalf("headroom detail must name %q, got %v", path, res.HeadroomDetail)
	}
}

func TestHeadroom_warnsPast80PercentLinesWithoutFailing(t *testing.T) {
	root := t.TempDir()
	path := "docs/by-lines.md"
	body := headroomBody(t, "by-lines.md", strings.Repeat("line\n", 420)+strings.Repeat("x", 7000)+"\n")
	if countLines(body) <= headroomLineWarn || countLines(body) > maxDocLines {
		t.Fatalf("fixture lines=%d want in (%d, %d]", countLines(body), headroomLineWarn, maxDocLines)
	}
	if len(body) > maxDocBytes {
		t.Fatalf("fixture bytes=%d want under %d", len(body), maxDocBytes)
	}
	writeFile(t, root, path, body)
	inv := mustInventory(t, root, path)

	res := Check(CheckInput{Root: root, Inventory: inv})
	if !res.OK {
		t.Fatalf("headroom must not fail the gate, errors=%v", res.Errors)
	}
	if res.HeadroomDocs != 1 {
		t.Fatalf("headroom_docs=%d want 1", res.HeadroomDocs)
	}
}

func TestHeadroom_exactlyAtLimitWarnsButPasses(t *testing.T) {
	root := t.TempDir()
	path := "docs/at-limit.md"
	header := headroomBody(t, "at-limit.md", "")
	rest := maxDocBytes - len(header) - 1
	if rest <= headroomByteWarn-len(header) {
		t.Fatalf("header too large to build an at-limit fixture: %d", len(header))
	}
	body := header + strings.Repeat("p", rest) + "\n"
	if len(body) != maxDocBytes {
		t.Fatalf("fixture bytes=%d want exactly %d", len(body), maxDocBytes)
	}
	writeFile(t, root, path, body)
	inv := mustInventory(t, root, path)

	res := Check(CheckInput{Root: root, Inventory: inv})
	if !res.OK {
		t.Fatalf("exactly %d bytes must pass, errors=%v", maxDocBytes, res.Errors)
	}
	if res.OversizedDocs != 0 {
		t.Fatalf("oversized_docs=%d want 0", res.OversizedDocs)
	}
	if res.HeadroomDocs != 1 {
		t.Fatalf("headroom_docs=%d want 1", res.HeadroomDocs)
	}
}

func TestHeadroom_oversizedIsNotHeadroom(t *testing.T) {
	root := t.TempDir()
	path := "docs/over.md"
	body := strings.Repeat("x", maxDocBytes+1) + "\n"
	writeFile(t, root, path, body)
	inv := mustInventory(t, root, path)

	res := Check(CheckInput{Root: root, Inventory: inv})
	if res.OK {
		t.Fatal("expected oversized failure")
	}
	if res.OversizedDocs != 1 {
		t.Fatalf("oversized_docs=%d want 1", res.OversizedDocs)
	}
	if res.HeadroomDocs != 0 {
		t.Fatalf("headroom_docs=%d want 0: oversized docs fail, they do not warn", res.HeadroomDocs)
	}
}

func TestNearLimit_sortedAndScoped(t *testing.T) {
	root := t.TempDir()
	small := "docs/small.md"
	writeFile(t, root, small, headroomBody(t, "small.md", "tiny\n"))
	mid := "docs/mid.md"
	writeFile(t, root, mid, headroomBody(t, "mid.md", strings.Repeat("pad pad pad pad pad pad pad pad pad pad\n", 205)))
	big := "workflows/big.md"
	writeFile(t, root, big, "# T\n\nSee [s](big.md).\n"+strings.Repeat("pad pad pad pad pad pad pad pad pad pad\n", 220))
	// Outside the size-enforced prefixes: large but never listed.
	outside := "notes/huge.md"
	writeFile(t, root, outside, strings.Repeat("z", maxDocBytes*2)+"\n")
	// Oversized: fails the gate, never listed as near-limit.
	over := "docs/over.md"
	writeFile(t, root, over, strings.Repeat("x", maxDocBytes+10)+"\n")
	inv := mustInventory(t, root, small, mid, big, outside, over)

	entries, err := collectNearLimit(root, inv)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("near-limit entries=%d want 2 (mid, big), got %+v", len(entries), entries)
	}
	if entries[0].path != big || entries[1].path != mid {
		t.Fatalf("want bytes-descending [big mid], got %+v", entries)
	}
	for _, e := range entries {
		if e.path == small || e.path == outside || e.path == over {
			t.Fatalf("unexpected entry %q in %+v", e.path, entries)
		}
	}
}
