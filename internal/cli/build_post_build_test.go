package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ScriptonBasestar/dva/internal/config"
)

// postBuildPlanConfig declares runners.native.post_build alongside build. The two commands
// write different markers into the same entry dir, so the pair of files answers three
// questions at once: did post_build run, did it run in the entry's dir rather than the
// process cwd, and did runners.native.env reach it the way build's env does.
//
// ordered's build appends to one file and post_build appends to the same one, so the file's
// contents record the sequence — a post_build that ran first would still leave both markers.
const postBuildPlanConfig = `version: "0.1.44"
stack:
  api:
    default_runner: native
    runners:
      native:
        dir: services/api
        build: touch built-${DVA_TEST_BUILD_TAG}
        post_build: touch packaged-${DVA_TEST_BUILD_TAG}
        run: ./api
        env:
          DVA_TEST_BUILD_TAG: v9
  ordered:
    default_runner: native
    runners:
      native:
        dir: services/api
        build: sh -c 'echo build >> order.txt'
        post_build: sh -c 'echo post >> order.txt'
        run: ./ordered
  nopost:
    default_runner: native
    runners:
      native:
        dir: services/api
        build: touch nopost-built
        run: ./nopost
  failing:
    default_runner: native
    runners:
      native:
        dir: services/api
        build: exit 1
        post_build: touch SHOULD-NOT-EXIST
        run: ./failing
  failpost:
    default_runner: native
    runners:
      native:
        dir: services/api
        build: touch failpost-built
        post_build: exit 3
        run: ./failpost
plans:
  api:
    entries:
      - name: api
  ordered:
    entries:
      - name: ordered
  nopost:
    entries:
      - name: nopost
  failing:
    entries:
      - name: failing
  failpost:
    entries:
      - name: failpost
`

func postBuildTestConfig(t *testing.T) *config.Config {
	t.Helper()
	c := loadTestConfig(t, postBuildPlanConfig)
	if err := os.MkdirAll(filepath.Join(c.FileDir(), "services", "api"), 0o755); err != nil {
		t.Fatalf("mkdir entry dir: %v", err)
	}
	return c
}

// buildOnePlan runs the single-entry plan's build the way `dva build <plan>` does.
func buildOnePlan(t *testing.T, c *config.Config, plan string) error {
	t.Helper()
	targets := planBuildTargets(resolveBuildPlan(t, c, plan))
	if len(targets) != 1 {
		t.Fatalf("plan %q produced %d build targets, want 1", plan, len(targets))
	}
	return buildPlanEntry(buildTestEnv(c), c, targets[0], nil)
}

func entryFile(t *testing.T, c *config.Config, name string) (string, bool) {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(c.FileDir(), "services", "api", name))
	if os.IsNotExist(err) {
		return "", false
	}
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return string(b), true
}

// post_build runs after a successful build, in the entry's dir and with its env — the
// same three guarantees build itself carries. Declaring the field and having it silently
// do nothing is the failure this pins down.
func TestPostBuildRunsAfterSuccessfulBuild(t *testing.T) {
	c := postBuildTestConfig(t)
	if err := buildOnePlan(t, c, "api"); err != nil {
		t.Fatalf("build: %v", err)
	}
	if _, ok := entryFile(t, c, "built-v9"); !ok {
		t.Error("build command did not run")
	}
	if _, ok := entryFile(t, c, "packaged-v9"); !ok {
		t.Error("post_build did not run in the entry dir with runners.native.env applied")
	}
}

// Order is the contract: post_build is defined as running after build, so a build that
// produced nothing yet must not be packaged.
func TestPostBuildRunsAfterBuildNotBefore(t *testing.T) {
	c := postBuildTestConfig(t)
	if err := buildOnePlan(t, c, "ordered"); err != nil {
		t.Fatalf("build: %v", err)
	}
	got, ok := entryFile(t, c, "order.txt")
	if !ok {
		t.Fatal("neither command ran")
	}
	if got != "build\npost\n" {
		t.Errorf("command order = %q, want \"build\\npost\\n\"", got)
	}
}

// An entry with no post_build must build exactly as before.
func TestPostBuildAbsentIsNotAnError(t *testing.T) {
	c := postBuildTestConfig(t)
	if err := buildOnePlan(t, c, "nopost"); err != nil {
		t.Fatalf("build: %v", err)
	}
	if _, ok := entryFile(t, c, "nopost-built"); !ok {
		t.Error("build command did not run")
	}
}

// A failed build must not be packaged: post_build would otherwise run against a stale or
// absent artifact and hand back a success the build never earned.
func TestPostBuildSkippedWhenBuildFails(t *testing.T) {
	c := postBuildTestConfig(t)
	if err := buildOnePlan(t, c, "failing"); err == nil {
		t.Fatal("build: got nil error, want the failing build to be reported")
	}
	if _, ok := entryFile(t, c, "SHOULD-NOT-EXIST"); ok {
		t.Error("post_build ran after the build failed")
	}
}

// A failing post_build fails the build. Swallowing it would report a build that succeeded
// while the artifact it was supposed to produce is missing.
func TestPostBuildFailurePropagates(t *testing.T) {
	c := postBuildTestConfig(t)
	err := buildOnePlan(t, c, "failpost")
	if err == nil {
		t.Fatal("build: got nil error, want the failing post_build to be reported")
	}
	if _, ok := entryFile(t, c, "failpost-built"); !ok {
		t.Error("build command did not run")
	}
}
