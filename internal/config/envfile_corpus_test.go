package config

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"testing"
)

// envFileEntryKey matches the keys normalizeEnvFileConfig reads from one entry map.
var envFileEntryKey = regexp.MustCompile(`it\["([a-z_]+)"\]`)

// TestEnvFileEntryKeysTaughtBySchema fails when the parser accepts an env_file
// entry key that the schema reference never writes in YAML key position.
//
// The schema reference is what the agent-mesh flows and the dva-config skill
// teach from. sops_source was parsed for months without appearing there, so no
// generated or improved dva.yml ever declared it and `dva config env` refused on
// first use. Reading the keys from the parser source, rather than listing them
// here, is what makes a future field fail this test instead of repeating that.
func TestEnvFileEntryKeysTaughtBySchema(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	dir := filepath.Dir(file)
	src, err := os.ReadFile(filepath.Join(dir, "envfile.go"))
	if err != nil {
		t.Fatal(err)
	}
	keys := envFileEntryKey.FindAllStringSubmatch(string(src), -1)
	if len(keys) == 0 {
		t.Fatal("found no entry keys in envfile.go; the pattern no longer matches the parser")
	}
	schema, err := os.ReadFile(filepath.Join(dir, "..", "..", "agent-mesh-flows", "shared", "library", "dva-schema.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, k := range keys {
		if !regexp.MustCompile(`(?m)^\s+(- )?` + k[1] + `:`).Match(schema) {
			t.Errorf("env_file entry key %q is parsed but never written as a YAML key in dva-schema.md", k[1])
		}
	}
}
