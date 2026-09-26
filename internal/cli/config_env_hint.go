package cli

import (
	"fmt"
	"strings"

	"github.com/ScriptonBasestar/dva/internal/config"
)

// sopsDeclarationExample renders the env_file entry the author most likely meant,
// in the inline form USAGE.md documents, so the refusal and the doctor row both
// hand back something that can be pasted rather than a field name to look up.
//
// The target is the first declared plaintext entry (the file the project already
// loads) and the source is the first sops candidate found; either falls back to a
// conventional name when there is nothing to point at.
func sopsDeclarationExample(c *config.Config, ev config.SopsEvidence) string {
	target := ".env"
	if entries := c.AllEnvFileConfigs(); len(entries) > 0 && entries[0].Path != "" {
		target = entries[0].Path
	}
	source := target + ".enc"
	if len(ev.Candidates) > 0 {
		source = ev.Candidates[0]
	}
	return fmt.Sprintf("env_file: [{path: %s, sops_source: %s}]", target, source)
}

// sopsEvidenceSummary names what was found, or "" when nothing was.
func sopsEvidenceSummary(ev config.SopsEvidence) string {
	var parts []string
	if ev.CreationRules {
		parts = append(parts, ".sops.yaml")
	}
	parts = append(parts, ev.Candidates...)
	return strings.Join(parts, ", ")
}

// noEncryptedEntryMessage is the no_encrypted_env_entry text. The code stays the
// frozen TASK-245 one; only the prose grows a remedy.
func noEncryptedEntryMessage(c *config.Config) string {
	ev := c.DetectSopsEvidence()
	msg := "no env_file entry declares sops_source; declare one in dva.yml, e.g. " + sopsDeclarationExample(c, ev)
	if s := sopsEvidenceSummary(ev); s != "" {
		msg += " (sops files found: " + s + ")"
	}
	return msg
}
