package cli

import (
	"strings"
	"testing"

	"github.com/ScriptonBasestar/dva/internal/config"
)

const unresolvedPathConfig = `version: "0.1.45"
env_file: .env.${DVA_TEST_UNSET_X:?need}
stack:
  svc:
    default_runner: script
    runners:
      script:
        up: touch runner-up
        logs: touch runner-logs
plans:
  dev:
    entries: [{name: svc}]
`

func TestStatusJSONReportsUnresolvedPathKind(t *testing.T) {
	c := loadTestConfig(t, unresolvedPathConfig)
	withEnvPolicyGlobals(t, c, true)

	var err error
	stdout, _ := captureStreams(t, func() { err = statusCmd.RunE(statusCmd, []string{"dev"}) })
	if err == nil {
		t.Fatal("status returned nil on an unresolved env_file path")
	}
	doc := decodeSingleJSONDocument(t, stdout)
	environment, ok := doc["environment"].(map[string]any)
	if !ok {
		t.Fatalf("no environment block: %v", doc)
	}
	failures, ok := environment["failures"].([]any)
	if !ok || len(failures) != 1 {
		t.Fatalf("failures = %v", environment["failures"])
	}
	if kind := failures[0].(map[string]any)["kind"]; kind != string(config.EnvInputUnresolvedPath) {
		t.Errorf("kind = %v", kind)
	}
}

func TestLogsRefusesUnresolvedPath(t *testing.T) {
	c := loadTestConfig(t, unresolvedPathConfig)
	withEnvPolicyGlobals(t, c, false)

	var err error
	stdout, _ := captureStreams(t, func() { err = runPlanLogs(c, rootEnvLoad(c), "dev", nil) })
	if err == nil || !strings.Contains(err.Error(), "environment inputs are incomplete") {
		t.Fatalf("error = %v", err)
	}
	if stdout != "" {
		t.Errorf("logs wrote to stdout: %q", stdout)
	}
}
