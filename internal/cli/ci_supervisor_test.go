package cli

import (
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestCISupervisorPreservesArgumentsAndDoesNotReenter(t *testing.T) {
	prevArgs, prevReplace, prevDry, prevLookup := os.Args, replaceCIProcess, dryRun, ciLookup
	t.Cleanup(func() { os.Args, replaceCIProcess, dryRun, ciLookup = prevArgs, prevReplace, prevDry, prevLookup })
	dryRun, ciLookup = false, false
	t.Setenv(ciSupervisorEnv, "/host tools/ci-budget")
	os.Args = []string{"dva", "--json", "ci", "dev", "--subproject", "name with spaces"}
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	want := append([]string{"/host tools/ci-budget", self}, os.Args[1:]...)
	rejected := errors.New("budget exhausted")
	replaceCIProcess = func(args, env []string) error {
		if !reflect.DeepEqual(args, want) {
			t.Fatalf("forwarded args %q, want %q", args, want)
		}
		for _, v := range env {
			if strings.HasPrefix(v, ciSupervisorEnv+"=") {
				t.Fatalf("recursive supervisor in child environment")
			}
		}
		return rejected
	}
	if err := ciCmd.RunE(ciCmd, []string{"dev"}); !errors.Is(err, rejected) {
		t.Fatalf("expected budget refusal, got %v", err)
	}
}
