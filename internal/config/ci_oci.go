package config

import (
	"fmt"
	"path"
	"regexp"
	"strings"
)

type CIOCI struct {
	Image            string            `yaml:"image" json:"image"`
	Argv             []string          `yaml:"argv" json:"argv"`
	Source           string            `yaml:"source" json:"source"`
	Workdir          string            `yaml:"workdir,omitempty" json:"workdir,omitempty"`
	Environment      map[string]string `yaml:"environment,omitempty" json:"environment,omitempty"`
	PidsLimit        int64             `yaml:"pids_limit" json:"pids_limit"`
	MemoryBytes      int64             `yaml:"memory_bytes" json:"memory_bytes"`
	CPUs             int               `yaml:"cpus" json:"cpus"`
	OutputLimitBytes int64             `yaml:"output_limit_bytes" json:"output_limit_bytes"`
}

var ciOCIImage = regexp.MustCompile(`^(sha256:[0-9a-f]{64}|[A-Za-z0-9][A-Za-z0-9._:/-]*@sha256:[0-9a-f]{64})$`)
var ciOCIEnv = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func validateCIOCI(profile, step string, oci CIOCI) error {
	fail := func(message string) error { return fmt.Errorf("ci profile %q step %q oci: %s", profile, step, message) }
	if !ciOCIImage.MatchString(oci.Image) {
		return fail("image must be an immutable sha256 digest")
	}
	if len(oci.Argv) == 0 {
		return fail("argv must not be empty")
	}
	if oci.Argv[0] == "" {
		return fail("argv command must not be empty")
	}
	for _, arg := range oci.Argv {
		if arg == "__complete" || arg == "__completeNoDesc" {
			return fail("argv contains a reserved Docker transport completion token")
		}
		if strings.ContainsRune(arg, 0) {
			return fail("argv must not contain NUL")
		}
	}
	if strings.ContainsAny(oci.Source, "\x00\\,") || oci.Source == "." || oci.Source == "" || path.IsAbs(oci.Source) || path.Clean(oci.Source) != oci.Source || strings.HasPrefix(oci.Source, "../") || oci.Source == ".." {
		return fail("source must be a clean relative subdirectory")
	}
	if oci.Workdir != "" && (strings.ContainsAny(oci.Workdir, "\x00\\") || path.Clean(oci.Workdir) != oci.Workdir || (oci.Workdir != "/workspace" && !strings.HasPrefix(oci.Workdir, "/workspace/"))) {
		return fail("workdir must be contained in /workspace")
	}
	for key, value := range oci.Environment {
		if !ciOCIEnv.MatchString(key) || strings.ContainsAny(key, "\x00=") || strings.ContainsRune(value, 0) {
			return fail("environment contains an invalid key or value")
		}
		if key == "HOME" || key == "TMPDIR" || strings.HasPrefix(key, "XDG_") || key == CIParentRunEnv || key == CISubprojectEnv {
			return fail("environment sets a reserved variable")
		}
	}
	if oci.PidsLimit < 8 || oci.PidsLimit > 4096 {
		return fail("pids_limit must be between 8 and 4096")
	}
	if oci.MemoryBytes < 16<<20 {
		return fail("memory_bytes must be at least 16MiB")
	}
	if oci.CPUs < 1 || oci.CPUs > 128 {
		return fail("cpus must be between 1 and 128")
	}
	if oci.OutputLimitBytes < 1024 || oci.OutputLimitBytes > 64<<20 {
		return fail("output_limit_bytes must be between 1024 and 64MiB")
	}
	return nil
}
