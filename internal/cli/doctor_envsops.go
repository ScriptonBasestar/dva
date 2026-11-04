package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/ScriptonBasestar/dva/internal/config"
	"github.com/ScriptonBasestar/dva/internal/lifecycle"
)

// checkEnvSopsDeclaration reports sops evidence that no env_file entry claims.
//
// sops_source never affects loading, so its absence cannot fail `up`; the first
// place it shows is `dva config env` refusing. Without evidence there is no row:
// most projects do not use sops, and a row for them would be noise.
func checkEnvSopsDeclaration(c *config.Config) []DoctorResult {
	if len(c.EncryptedEnvEntries()) > 0 {
		return nil
	}
	ev := c.DetectSopsEvidence()
	if !ev.Found() {
		return nil
	}
	return []DoctorResult{{
		Name:    "Encrypted env source declared",
		Finding: "sops files found but no env_file entry declares sops_source: " + sopsEvidenceSummary(ev),
		FixHint: "Declare the source in dva.yml, e.g. " + sopsDeclarationExample(c, ev),
	}}
}

// checkEnvUnsealed reports encrypted env_file targets that are absent. The loader
// skips them as optional, so `up` proceeds without their values.
func checkEnvUnsealed(c *config.Config, r *config.EnvInputReport) []DoctorResult {
	var results []DoctorResult
	for _, target := range c.UnsealedEnvTargets(r) {
		results = append(results, DoctorResult{
			Name:    fmt.Sprintf("Encrypted env target present: %s", target),
			Finding: unsealedEnvFinding(target),
			FixHint: "Run: " + unsealCommand(target),
		})
	}
	return results
}

func unsealCommand(target string) string {
	separator := ""
	if strings.HasPrefix(target, "-") {
		separator = "-- "
	}
	if target == "" || strings.IndexFunc(target, func(c rune) bool {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			return false
		}
		return !strings.ContainsRune("_./-", c)
	}) >= 0 {
		target = "'" + strings.ReplaceAll(target, "'", "'\"'\"'") + "'"
	}
	return "dva config env unseal " + separator + target
}

func unsealedEnvFinding(target string) string {
	return fmt.Sprintf("env_file %s declares sops_source but does not exist, so its values are not loaded", target)
}

// warnUnsealedEnv tells `up` what doctor reports. It cannot fail the run: the entry
// is optional, and sops_source stays invisible to loading (TASK-245). The recreate
// note matters because unsealing later changes the environment of running containers.
func warnUnsealedEnv(c *config.Config, r *config.EnvInputReport, label string) {
	for _, target := range c.UnsealedEnvTargets(r) {
		fmt.Fprintf(os.Stderr, "warning: %s%s; run `%s` in the project that declares it, then `dva up` again (compose containers using these values are recreated)\n", label, unsealedEnvFinding(target), unsealCommand(target))
	}
}

// warnCompositionUnsealed checks each child's owner, because children load their own
// env and a root env_file may not reach them. It inspects rather than resolves the
// child: resolvePlanRuntime merges plan vars into the environment the run will use.
func warnCompositionUnsealed(c *config.Config, comp *lifecycle.CompositionPlan) {
	seen := map[*config.Config]bool{}
	for _, entry := range comp.Entries {
		owner := entry.ChildPlan.OwnerConfig(c)
		if owner == nil || seen[owner] {
			continue
		}
		seen[owner] = true
		warnUnsealedEnv(owner, config.InspectEnvFiles(owner.EnvFile, owner.FileDir(), owner.EnvFileScope()), "[child: "+entry.ChildPlan.Name+"] ")
	}
}
