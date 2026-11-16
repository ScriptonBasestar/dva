package cli

import (
	"errors"
	"net"
	"strconv"
	"strings"
	"testing"
)

func TestCompareSkippedVariantDoesNotProbePorts(t *testing.T) {
	host := newFakeHost()
	host.portErrors[41001] = errors.New("must not probe a running variant")
	old := compareHost
	compareHost = host
	t.Cleanup(func() { compareHost = old })
	problems := checkPorts([]*upVariant{{skip: true, ports: map[string]int{"WEB_PORT": 41001}}})
	if len(problems) != 0 || len(host.portChecks) != 0 {
		t.Fatalf("skipped variant: problems=%v probes=%v", problems, host.portChecks)
	}
}

// Hold the actual listener throughout the production check: no close/rebind gap,
// fixed port, fake occupancy state or skip-on-collision is involved.
func TestCompareProductionPortProbeSeesHeldListener(t *testing.T) {
	l, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = l.Close() })
	port := l.Addr().(*net.TCPAddr).Port
	t.Setenv("DVA_TEST_HELD_PORT", strconv.Itoa(port))
	old := compareHost
	compareHost = lifecycleRunner{}
	t.Cleanup(func() { compareHost = old })
	problems := checkPorts([]*upVariant{{spec: variantSpec{Name: "held"}, ports: map[string]int{"DVA_TEST_HELD_PORT": port}}})
	if len(problems) != 1 || !strings.Contains(problems[0], "already in use") {
		t.Fatalf("production probe did not reject listener: %v", problems)
	}
}
