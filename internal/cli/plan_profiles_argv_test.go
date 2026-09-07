package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ScriptonBasestar/dva/internal/config"
)

// planProfilesCLIConfig is one compose entry read by two plans that differ only in how they
// select: `gated` names profiles and no services, `plain` names a service and no profiles.
//
// The pair is the whole fixture. `gated` is the shape that broke — a profiles-only plan has
// no service positionals, so a build or logs command that forwards only `services:` forwards
// nothing and addresses the whole file (or, for build, nothing at all). `plain` is the
// regression guard for every plan written before profiles existed: its argv must not move.
const planProfilesCLIConfig = `version: "0.1.44"
stack:
  core:
    default_runner: compose
    runners:
      compose:
        files: [compose.yml]
        project_name: profiles-demo
plans:
  gated:
    entries:
      - name: core
        profiles: [rust, monitoring]
  plain:
    entries:
      - name: core
        services: [db]
`

// planProfilesComposeFile gates api-rs behind the profile the `gated` plan activates, which
// is what makes the flag load-bearing: without --profile compose does not consider api-rs at
// all, so `compose build` has nothing to build and reports success.
const planProfilesComposeFile = `services:
  db:
    build: ./db
  api-rs:
    profiles: [rust]
    build: ./api
`

func planProfilesCLIConfigOnDisk(t *testing.T) *config.Config {
	t.Helper()
	c := loadTestConfig(t, planProfilesCLIConfig)
	writePlanProfilesComposeFile(t, c.FileDir())
	return c
}

func writePlanProfilesComposeFile(t *testing.T, dir string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "compose.yml"), []byte(planProfilesComposeFile), 0o644); err != nil {
		t.Fatalf("write compose file: %v", err)
	}
}

// planProfilesComposePrefix is the argv every invocation below shares: the -f flag with the
// resolved path and the project name, in the order ComposeArgv emits them. Spelling it out
// rather than calling the builder keeps the expectations independent of the code they check.
func planProfilesComposePrefix(c *config.Config) string {
	return fmt.Sprintf("docker compose -f %s --project-name profiles-demo", filepath.Join(c.FileDir(), "compose.yml"))
}

// planProfilesBuildDryRun runs `dva build <plan> --dry-run` and returns the previewed argv.
func planProfilesBuildDryRun(t *testing.T, plan string) (*config.Config, string) {
	t.Helper()
	enableDryRun(t)
	c := planProfilesCLIConfigOnDisk(t)

	var err error
	stderr := captureBothStreams(t, func() { err = runPlanBuild(c, planEnv(buildTestEnv(c)), plan, nil) })
	if err != nil {
		t.Fatalf("runPlanBuild(%s): %v", plan, err)
	}
	return c, strings.TrimSpace(stderr)
}

// TestPlanBuildArgvActivatesThePlanProfiles is the D1 closure.
//
// TASK-314 made `dva build <plan>` build the plan rather than the file by forwarding the
// plan's `services:` subset. A plan that selects by profile has no subset to forward, so on
// the code this test was written against `dva build gated` emitted a bare `compose build`
// against a project where api-rs does not exist, printed nothing, and exited 0 — while
// `dva up gated` started it. The profile flags are what put the service back in view.
func TestPlanBuildArgvActivatesThePlanProfiles(t *testing.T) {
	c, argv := planProfilesBuildDryRun(t, "gated")

	want := "[dry-run] core: " + planProfilesComposePrefix(c) + " --profile rust --profile monitoring build"
	if argv != want {
		t.Fatalf("build argv:\n got %q\nwant %q", argv, want)
	}
}

// TestPlanBuildArgvUnchangedWhenThePlanDeclaresNoProfiles: the argv is asserted whole, not
// merely searched for an absent --profile. A plan written before this field existed must
// reach docker byte-identical, and only a full comparison can say that — a `!Contains`
// check would pass on an argv that had grown or reordered anything else.
func TestPlanBuildArgvUnchangedWhenThePlanDeclaresNoProfiles(t *testing.T) {
	c, argv := planProfilesBuildDryRun(t, "plain")

	want := "[dry-run] core: " + planProfilesComposePrefix(c) + " build db"
	if argv != want {
		t.Fatalf("build argv:\n got %q\nwant %q", argv, want)
	}
}

// planProfilesLogsArgv runs `dva logs <plan>` against the docker shim and returns the single
// argv line docker received. The shim fixture sets forceSubprocess, which is what keeps the
// logs path — otherwise an ExecReplace — from replacing the test binary with docker.
func planProfilesLogsArgv(t *testing.T, plan string) (*config.Config, string) {
	t.Helper()
	dockerArgv := composePassthroughFixtureWith(t, planProfilesCLIConfig)
	writePlanProfilesComposeFile(t, ".")
	c := mustLoadConfig()
	writePlanProfilesComposeFile(t, c.FileDir())

	if err := runPlanLogs(c, planEnv(buildTestEnv(c)), plan, nil); err != nil {
		t.Fatalf("runPlanLogs(%s): %v", plan, err)
	}
	lines := dockerArgv()
	if len(lines) != 1 {
		t.Fatalf("docker was invoked %d times, want 1: %v", len(lines), lines)
	}
	return c, "docker " + lines[0]
}

// TestPlanLogsArgvActivatesThePlanProfiles.
//
// Unlike build this is not a visible break today: the docker compose in use (5.5.0) prints a
// running profile-gated container's logs without the profile being active. That is a property
// of one implementation, not of the interface — and a profiles-only plan has no service
// positionals either, so the unscoped form asks for the whole file and would answer a
// question about the plan with output from services the plan never started.
func TestPlanLogsArgvActivatesThePlanProfiles(t *testing.T) {
	c, argv := planProfilesLogsArgv(t, "gated")

	want := planProfilesComposePrefix(c) + " --profile rust --profile monitoring logs"
	if argv != want {
		t.Fatalf("logs argv:\n got %q\nwant %q", argv, want)
	}
}

// TestPlanLogsArgvUnchangedWhenThePlanDeclaresNoProfiles: same whole-argv comparison, same
// reason, for the path that carries every plan written before this field existed.
func TestPlanLogsArgvUnchangedWhenThePlanDeclaresNoProfiles(t *testing.T) {
	c, argv := planProfilesLogsArgv(t, "plain")

	want := planProfilesComposePrefix(c) + " logs db"
	if argv != want {
		t.Fatalf("logs argv:\n got %q\nwant %q", argv, want)
	}
}
