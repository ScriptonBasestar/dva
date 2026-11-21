package cli

import (
	"io"
	"os"
	"strings"
	"testing"
)

func TestResolveResourcePolicyPath(t *testing.T) {
	t.Run("flag wins over environment", func(t *testing.T) {
		t.Setenv("DVA_RESOURCE_POLICY", "/env/policy.json")
		got, err := resolveResourcePolicyPath("/flag/policy.json")
		if err != nil || got != "/flag/policy.json" {
			t.Fatalf("got %q, %v", got, err)
		}
	})
	t.Run("environment when no flag", func(t *testing.T) {
		t.Setenv("DVA_RESOURCE_POLICY", "/env/policy.json")
		got, err := resolveResourcePolicyPath("")
		if err != nil || got != "/env/policy.json" {
			t.Fatalf("got %q, %v", got, err)
		}
	})
	t.Run("explicitly empty environment", func(t *testing.T) {
		t.Setenv("DVA_RESOURCE_POLICY", "")
		_, err := resolveResourcePolicyPath("")
		if err == nil || !strings.Contains(err.Error(), "explicitly empty: set it to an absolute policy path, or unset it and pass --policy") {
			t.Fatalf("err = %v", err)
		}
	})
}

// Each resource command names a missing policy instead of reporting the
// empty path as "must be absolute".
func TestResourceCommandsNameMissingPolicy(t *testing.T) {
	// t.Setenv registers restoration; Unsetenv then makes the key absent.
	for _, key := range []string{"DVA_RESOURCE_POLICY", "DVA_RESOURCE_SUPERVISED", "DVA_CI_PARENT_RUN"} {
		t.Setenv(key, "")
		if err := os.Unsetenv(key); err != nil {
			t.Fatal(err)
		}
	}
	for _, args := range [][]string{
		{"status"},
		{"recover"},
		{"run", "--class", "light", "--", "true"},
	} {
		t.Run(args[0], func(t *testing.T) {
			cmd := newResourceCommand()
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			cmd.SetArgs(args)
			err := cmd.Execute()
			if err == nil || !strings.Contains(err.Error(), "no resource policy") {
				t.Fatalf("err = %v", err)
			}
		})
	}
}
