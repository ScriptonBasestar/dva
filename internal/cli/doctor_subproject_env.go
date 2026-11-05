package cli

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ScriptonBasestar/dva/internal/config"
)

// checkSubprojectEncryptedEnv inspects only directly declared child owners. It
// never runs a child's backend/user doctor checks or recursively visits children.
func checkSubprojectEncryptedEnv(c *config.Config) []DoctorResult {
	names := make([]string, 0, len(c.Subprojects))
	for name := range c.Subprojects {
		names = append(names, name)
	}
	sort.Strings(names)
	var results []DoctorResult
	for _, name := range names {
		dir := c.Subprojects[name].Path
		if !filepath.IsAbs(dir) {
			dir = filepath.Join(c.FileDir(), dir)
		}
		file, found := config.ConfigFileInDir(dir)
		if !found {
			results = append(results, subprojectEnvUnavailable(name))
			continue
		}
		// Explicit selection prevents DVA_FILE or ancestor discovery from
		// silently substituting the parent's configuration for a missing child.
		child, err := config.Load(dir, config.FromFile(file), config.ForEnvInspection())
		if err != nil {
			results = append(results, subprojectEnvUnavailable(name))
			continue
		}
		encrypted := make(map[string]bool)
		for _, entry := range child.EncryptedEnvEntries() {
			encrypted[entry.Path] = true
		}
		if len(encrypted) == 0 {
			continue
		}
		report := config.InspectEnvFiles(child.EnvFile, child.FileDir(), child.EnvFileScope())
		// Keep unrelated plaintext-input diagnostics in their own child doctor.
		// Failed encrypted inputs are visible even when absence cannot be decided.
		var failed []config.EnvInputEntry
		for _, entry := range report.Entries {
			if encrypted[entry.File] && entry.Status == config.EnvInputFailed {
				failed = append(failed, entry)
			}
		}
		rows := envInputResults(&config.EnvInputReport{Entries: failed})
		rows = append(rows, checkEnvUnsealed(child, report)...)
		for _, row := range rows {
			row.Name = fmt.Sprintf("[child: %s] %s", name, row.Name)
			row.FixHint = fmt.Sprintf("In subproject %q: %s", name, strings.TrimPrefix(row.FixHint, "Run: "))
			results = append(results, row)
		}
	}
	return results
}

func subprojectEnvUnavailable(name string) DoctorResult {
	return DoctorResult{
		Name:    fmt.Sprintf("[child: %s] Encrypted env inspection available", name),
		Finding: "Child configuration could not be loaded; encrypted env targets were not inspected",
		FixHint: "Run dva config validate from this subproject's declared directory",
	}
}
