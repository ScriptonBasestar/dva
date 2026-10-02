package cli

import (
	"strconv"
	"strings"
	"testing"
)

// --- TASK-115: a command: that splits to no words ---

// Each of these passes JSON-schema validation — to the schema they are strings — and each
// one made every compose runner index into a nil slice. `dva validate` reporting them is
// the difference between finding out at edit time and finding out from a stack trace.
func TestDetectUnrunnableComposeCommands_Reports(t *testing.T) {
	for _, command := range []string{"   ", "\t", `''`, `""`} {
		t.Run(strings.ReplaceAll(command, "\t", "<tab>"), func(t *testing.T) {
			c := loadTestConfig(t, "version: \"0.1.22\"\nstack:\n  db:\n    default_runner: compose\n    runners:\n      compose:\n        command: "+strconv.Quote(command)+"\n        files: [compose.yml]\n")

			problems := detectUnrunnableComposeCommands(c)
			if len(problems) != 1 {
				t.Fatalf("problems = %v, want exactly 1 for command %q", problems, command)
			}
			// The author has to be able to find the line. A message that says only "invalid
			// command" sends them looking through the whole file.
			for _, want := range []string{"stack.db", "command"} {
				if !strings.Contains(problems[0], want) {
					t.Errorf("problem does not mention %q:\n%s", want, problems[0])
				}
			}
		})
	}
}

func TestDetectUnrunnableComposeCommands_AcceptsRealCommands(t *testing.T) {
	for _, command := range []string{"", "docker compose", "podman-compose", "docker --context remote compose"} {
		t.Run(command, func(t *testing.T) {
			c := loadTestConfig(t, "version: \"0.1.22\"\nstack:\n  db:\n    default_runner: compose\n    runners:\n      compose:\n        command: "+strconv.Quote(command)+"\n        files: [compose.yml]\n")

			if problems := detectUnrunnableComposeCommands(c); len(problems) != 0 {
				t.Errorf("command %q was rejected: %v", command, problems)
			}
		})
	}
}
