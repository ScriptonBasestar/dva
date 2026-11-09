package compare

import (
	"net"
	"strings"
	"testing"

	"github.com/ScriptonBasestar/dva/internal/config"
)

func portCfg(lists map[string][]int) *config.CompareConfig {
	return &config.CompareConfig{Plan: "p", Ports: lists}
}

func TestAssignPortsTakesLowestFreePositionAndKeepsExisting(t *testing.T) {
	cmp := portCfg(map[string][]int{"WEB_PORT": {3001, 3002, 3003}, "API_PORT": {4001, 4002, 4003}})
	got, err := AssignPorts(cmp, nil, []string{"a", "b"})
	if err != nil {
		t.Fatal(err)
	}
	if got["a"]["WEB_PORT"] != 3001 || got["a"]["API_PORT"] != 4001 || got["b"]["WEB_PORT"] != 3002 {
		t.Fatalf("first assignment = %v", got)
	}

	// Only b is recorded, at position 1: a new variant takes the free position 0.
	recorded := []Variant{{Name: "b", Ports: got["b"]}}
	later, err := AssignPorts(cmp, recorded, []string{"c"})
	if err != nil {
		t.Fatal(err)
	}
	if later["c"]["WEB_PORT"] != 3001 || later["c"]["API_PORT"] != 4001 {
		t.Errorf("c must take the lowest free position, got %v", later["c"])
	}
	both := []Variant{{Name: "a", Ports: got["a"]}, {Name: "b", Ports: got["b"]}}
	next, _ := AssignPorts(cmp, both, []string{"c"})
	if next["c"]["WEB_PORT"] != 3003 {
		t.Errorf("c = %v, want position 2", next["c"])
	}
}

func TestAssignPortsFailsWhenListIsShort(t *testing.T) {
	cmp := portCfg(map[string][]int{"WEB_PORT": {3001, 3002}})
	_, err := AssignPorts(cmp, nil, []string{"a", "b", "c"})
	if err == nil || !strings.Contains(err.Error(), "WEB_PORT") {
		t.Fatalf("err = %v", err)
	}
}

func TestAssignPortsWithoutPortsAssignsNone(t *testing.T) {
	got, err := AssignPorts(portCfg(nil), nil, []string{"a"})
	if err != nil || len(got["a"]) != 0 {
		t.Fatalf("got %v, %v", got, err)
	}
}

func TestPortBusyDetectsListenerOnEitherAddress(t *testing.T) {
	for _, host := range []string{"127.0.0.1", ""} {
		l, err := net.Listen("tcp", net.JoinHostPort(host, "0"))
		if err != nil {
			t.Fatal(err)
		}
		port := l.Addr().(*net.TCPAddr).Port
		busy, err := PortBusy(port)
		_ = l.Close()
		if err != nil || !busy {
			t.Errorf("listener on %q: busy=%v err=%v", host, busy, err)
		}
		if busy, err := PortBusy(port); err != nil || busy {
			t.Errorf("after close port %d must be free: busy=%v err=%v", port, busy, err)
		}
	}
}

func TestPortBusyRejectsOutOfRange(t *testing.T) {
	for _, p := range []int{0, 70000} {
		if _, err := PortBusy(p); err == nil {
			t.Errorf("port %d accepted", p)
		}
	}
}

func TestValidNameMatchesConfigRule(t *testing.T) {
	for name, want := range map[string]bool{"a": true, "web-1": true, "9x": true, "": false, "../x": false, "-a": false, "a b": false} {
		if ValidName(name) != want {
			t.Errorf("ValidName(%q) = %v, want %v", name, !want, want)
		}
	}
}
