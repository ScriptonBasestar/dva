package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ScriptonBasestar/dva/internal/config"
	"github.com/ScriptonBasestar/dva/internal/output"
)

func TestRunDoctorCommandRecordsExecutionFacts(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DVA_DOCTOR_AMBIENT", "kept")

	if err := os.WriteFile(filepath.Join(dir, "marker"), []byte("ok"), 0o600); err != nil {
		t.Fatal(err)
	}
	passed, evidence := runDoctorCommand(config.DoctorCheck{
		Name:    "relative and ambient",
		Type:    "command",
		Command: "test \"$(pwd)\" = \"$PWD\" && test \"$DVA_DOCTOR_AMBIENT\" = kept && test -f marker",
	}, dir)
	if !passed {
		t.Fatalf("command did not pass: %+v", evidence)
	}
	if evidence.Cwd != dir || len(evidence.ShellArgv) != 3 || !filepath.IsAbs(evidence.ShellArgv[0]) {
		t.Fatalf("evidence = %+v, want cwd and shell argv", evidence)
	}
	if evidence.ExitCode == nil || *evidence.ExitCode != 0 {
		t.Fatalf("exit_code = %v, want pointer to 0", evidence.ExitCode)
	}
	encoded, err := json.Marshal(evidence)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), `"exit_code":0`) {
		t.Fatalf("successful exit code disappeared from JSON: %s", encoded)
	}
}

func TestRunDoctorCommandUsesOuterExitStatus(t *testing.T) {
	passed, evidence := runDoctorCommand(config.DoctorCheck{
		Name:    "outer quality",
		Type:    "command",
		Command: "true; true; exit 2",
	}, t.TempDir())
	if passed || evidence.ExitCode == nil || *evidence.ExitCode != 2 || evidence.ErrorClassification != "exit_status" {
		t.Fatalf("outer failure was not preserved: passed=%t evidence=%+v", passed, evidence)
	}

	passed, evidence = runDoctorCommand(config.DoctorCheck{Name: "nonzero", Type: "command", Command: "exit 17"}, t.TempDir())
	if passed || evidence.ExitCode == nil || *evidence.ExitCode != 17 {
		t.Fatalf("exit 17 was not preserved: passed=%t evidence=%+v", passed, evidence)
	}
}

func TestRunDoctorCommandLaunchErrorsAndEvidenceDoNotLeakOutput(t *testing.T) {
	t.Run("missing shell", func(t *testing.T) {
		t.Setenv("PATH", "")
		passed, evidence := runDoctorCommand(config.DoctorCheck{Name: "missing shell", Type: "command", Command: "true"}, t.TempDir())
		if passed || evidence.ErrorClassification != "start_error" || evidence.LaunchError != "shell_not_found" {
			t.Fatalf("missing shell = passed=%t evidence=%+v", passed, evidence)
		}
		if got := doctorCommandFinding("true", evidence); !strings.Contains(got, "shell_not_found") {
			t.Fatalf("finding = %q, want launch subtype", got)
		}
	})

	t.Run("unavailable cwd", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "missing")
		passed, evidence := runDoctorCommand(config.DoctorCheck{Name: "bad cwd", Type: "command", Command: "true"}, dir)
		if passed || evidence.ErrorClassification != "start_error" || evidence.LaunchError != "process_start_failed" {
			t.Fatalf("unavailable cwd = passed=%t evidence=%+v", passed, evidence)
		}
	})

	t.Run("output and ambient values omitted", func(t *testing.T) {
		const secret = "doctor-ambient-sentinel"
		t.Setenv("DVA_DOCTOR_SECRET", secret)
		_, evidence := runDoctorCommand(config.DoctorCheck{Name: "private output", Type: "command", Command: "printf '%s' \"$DVA_DOCTOR_SECRET\"; printf '%s' \"$DVA_DOCTOR_SECRET\" >&2; exit 17"}, t.TempDir())
		encoded, err := json.Marshal(evidence)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(encoded), secret) {
			t.Fatalf("command output or ambient value leaked: %s", encoded)
		}
	})
}

// This is the command's public path, not just the helper: two inner successes cannot
// turn a final outer exit 2 into a doctor success or hide that exit from --json.
func TestDoctorCommandJSONPreservesOuterExitFailure(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, config.FileName)
	if err := os.WriteFile(configPath, []byte("checks:\n  - name: outer quality\n    type: command\n    command: 'true; true; exit 2'\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(config.EnvFileKey, configPath)

	oldCfg, oldEnv, oldJSON := cfg, env, jsonOutput
	oldRunningCommand, oldDoctorFix, oldDoctorStrict := runningCommand, doctorFix, doctorStrict
	jsonFlag := rootCmd.PersistentFlags().Lookup("json")
	oldJSONFlag, oldJSONChanged := jsonFlag.Value.String(), jsonFlag.Changed
	cfg, env, jsonOutput = nil, nil, false
	defer func() {
		cfg, env, jsonOutput = oldCfg, oldEnv, oldJSON
		runningCommand, doctorFix, doctorStrict = oldRunningCommand, oldDoctorFix, oldDoctorStrict
		if err := jsonFlag.Value.Set(oldJSONFlag); err != nil {
			t.Errorf("restore --json flag: %v", err)
		}
		jsonFlag.Changed = oldJSONChanged
	}()
	output.ResetStdoutDocument()
	defer output.ResetStdoutDocument()

	if err := doctorCmd.Flags().Set("strict", "false"); err != nil {
		t.Fatal(err)
	}
	if err := doctorCmd.Flags().Set("fix", "false"); err != nil {
		t.Fatal(err)
	}
	rootCmd.SetArgs([]string{"--json", "doctor"})
	defer rootCmd.SetArgs(nil)
	stdout, _, err := captureValidateOutput(t, func() error {
		_, executeErr := rootCmd.ExecuteC()
		return executeErr
	})
	if err == nil || !strings.Contains(err.Error(), "1 user check(s) failed") {
		t.Fatalf("doctor error = %v, want user check failure", err)
	}
	var result struct {
		Checks []struct {
			Name    string `json:"name"`
			Passed  bool   `json:"passed"`
			Command struct {
				ExitCode *int `json:"exit_code"`
			} `json:"command"`
		} `json:"checks"`
	}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("doctor --json is not parseable: %v\n%s", err, stdout)
	}
	for _, check := range result.Checks {
		if check.Name == "outer quality" {
			if check.Passed || check.Command.ExitCode == nil || *check.Command.ExitCode != 2 {
				t.Fatalf("outer quality result = %+v, want failed exit 2", check)
			}
			return
		}
	}
	t.Fatalf("outer quality missing from doctor JSON: %s", stdout)
}
