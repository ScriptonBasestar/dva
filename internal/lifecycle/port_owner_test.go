package lifecycle

import (
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/ScriptonBasestar/dva/internal/config"
)

func tcpCheck(port int) map[string]config.HealthCheckConfig {
	return map[string]config.HealthCheckConfig{"web": {Type: "tcp", Address: "127.0.0.1:" + strconv.Itoa(port)}}
}

func stubPortOwner(t *testing.T, pids []int, verified bool, groups map[int]int) {
	t.Helper()
	oldLookup, oldGroup := listenerLookup, processGroupOf
	t.Cleanup(func() { listenerLookup, processGroupOf = oldLookup, oldGroup })
	listenerLookup = func(int) ([]int, bool) { return pids, verified }
	processGroupOf = func(pid int) (int, bool) {
		group, ok := groups[pid]
		return group, ok
	}
}

func TestPortOwnerMismatchNamesAddressAndPid(t *testing.T) {
	stubPortOwner(t, []int{4242}, true, map[int]int{4242: 4242})
	report := checkHealthPortOwners(tcpCheck(3000), nil, 100)
	if len(report.Mismatched) != 1 || report.Mismatched[0] != "127.0.0.1:3000 (pid 4242)" {
		t.Fatalf("Mismatched = %v", report.Mismatched)
	}
	err := portOwnerError("web", 100, report.Mismatched)
	if !strings.Contains(err.Error(), "127.0.0.1:3000 (pid 4242)") || !strings.Contains(err.Error(), "owner mismatch") {
		t.Fatalf("error = %v", err)
	}
}

func TestPortOwnerMatchInTrackedGroup(t *testing.T) {
	// A child of the tracked process (e.g. a node server under npm) shares its group.
	stubPortOwner(t, []int{101}, true, map[int]int{101: 100})
	report := checkHealthPortOwners(tcpCheck(3000), nil, 100)
	if len(report.Mismatched) != 0 || len(report.Unverified) != 0 {
		t.Fatalf("report = %+v, want clean", report)
	}
}

func TestPortOwnerUnverifiedWithoutLookup(t *testing.T) {
	stubPortOwner(t, nil, false, nil)
	report := checkHealthPortOwners(tcpCheck(3000), nil, 100)
	if len(report.Mismatched) != 0 || len(report.Unverified) != 1 {
		t.Fatalf("report = %+v, want one unverified", report)
	}
}

func TestPortOwnerUnverifiedWhenGroupUnknown(t *testing.T) {
	stubPortOwner(t, []int{4242}, true, nil)
	report := checkHealthPortOwners(tcpCheck(3000), nil, 100)
	if len(report.Mismatched) != 0 || len(report.Unverified) != 1 {
		t.Fatalf("report = %+v, want unverified", report)
	}
}

// A remote host is never compared and never warned about on up, but status still lists it as
// unverified (TASK-633).
func TestPortOwnerRemoteHostSeparateFromUnverified(t *testing.T) {
	stubPortOwner(t, []int{4242}, true, map[int]int{4242: 4242})
	checks := map[string]config.HealthCheckConfig{"db": {Type: "tcp", Address: "db.example.com:5432"}}
	report := checkHealthPortOwners(checks, nil, 100)
	if len(report.Mismatched) != 0 || len(report.Unverified) != 0 || len(report.Remote) != 1 {
		t.Fatalf("report = %+v, want one remote and no unverified", report)
	}
	if got := report.extra()[portOwnerUnverifiedExtraKey]; got != "db.example.com:5432" {
		t.Fatalf("extra unverified = %q, want the remote address kept for status", got)
	}
}

// owner: external skips the comparison even when a foreign group holds the port.
func TestPortOwnerExternalSkipsMismatch(t *testing.T) {
	stubPortOwner(t, []int{4242}, true, map[int]int{4242: 4242})
	checks := tcpCheck(3000)
	web := checks["web"]
	web.Owner = config.HealthOwnerExternal
	checks["web"] = web
	report := checkHealthPortOwners(checks, nil, 100)
	if len(report.Mismatched) != 0 || len(report.Unverified) != 0 || len(report.External) != 1 {
		t.Fatalf("report = %+v, want one external and nothing else", report)
	}
}

// The declaration is about the port: another undeclared check on the same address must not
// bring the mismatch back, whichever map order Go picks.
func TestPortOwnerExternalWinsOnSharedAddress(t *testing.T) {
	stubPortOwner(t, []int{4242}, true, map[int]int{4242: 4242})
	checks := map[string]config.HealthCheckConfig{
		"a-plain": {Type: "tcp", Address: "127.0.0.1:3000"},
		"z-ext":   {Type: "http", URL: "http://127.0.0.1:3000/health", Owner: config.HealthOwnerExternal},
	}
	for range 20 {
		report := checkHealthPortOwners(checks, nil, 100)
		if len(report.Mismatched) != 0 || len(report.External) != 1 {
			t.Fatalf("report = %+v, want the shared address external", report)
		}
	}
}

func TestStatusPortOwnerExternalLine(t *testing.T) {
	out := captureStdout(func() {
		printServiceTable([]ServiceStatus{
			{Name: "c", State: "running", Extra: map[string]string{portOwnerExternalExtraKey: "127.0.0.1:5000"}},
		})
	})
	if !strings.Contains(out, "c: health port owner check skipped for 127.0.0.1:5000 (declared external)") {
		t.Fatalf("table = %q", out)
	}
}

func TestStatusPortOwnerTableLines(t *testing.T) {
	out := captureStdout(func() {
		printServiceTable([]ServiceStatus{
			{Name: "a", State: "error", Extra: map[string]string{portOwnerExtraKey: "127.0.0.1:3000 (pid 7)"}},
			{Name: "b", State: "running", Extra: map[string]string{portOwnerUnverifiedExtraKey: "127.0.0.1:4000"}},
		})
	})
	if !strings.Contains(out, "a: health port owner mismatch — 127.0.0.1:3000 (pid 7)") ||
		!strings.Contains(out, "b: health port owner unverified for 127.0.0.1:4000") {
		t.Fatalf("table = %q", out)
	}
}

// TestPortOwnerRealLookup exercises the OS lookup: this test process listens, and its group is
// not the fake tracked pid, so the real lookup must report a mismatch naming this pid.
func TestPortOwnerRealLookup(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("no owner lookup on " + runtime.GOOS)
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	port := ln.Addr().(*net.TCPAddr).Port
	pids, verified := lookupListenerPids(port)
	if !verified {
		t.Skip("owner lookup unavailable here (no lsof or /proc access)")
	}
	found := false
	for _, pid := range pids {
		found = found || pid == os.Getpid()
	}
	if !found {
		t.Fatalf("lookup pids = %v, want %d", pids, os.Getpid())
	}
	report := checkHealthPortOwners(tcpCheck(port), nil, 1)
	if len(report.Mismatched) != 1 || !strings.Contains(report.Mismatched[0], strconv.Itoa(os.Getpid())) {
		t.Fatalf("report = %+v", report)
	}
}
