package lifecycle

import (
	"maps"
	"strings"

	"github.com/ScriptonBasestar/dva/internal/config"
)

func mergeStringMap(dst map[string]string, src map[string]string) {
	if len(src) == 0 {
		return
	}
	maps.Copy(dst, src)
}

func copyStringSlice(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, len(in))
	copy(out, in)
	return out
}

func normalizeRunnerName(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	if name == "" {
		return ""
	}
	return strings.ReplaceAll(name, "_", "-")
}

func runnerDeclared(runners map[string]any, runner string) bool {
	runner = normalizeRunnerName(runner)
	for k := range runners {
		if normalizeRunnerName(k) == runner {
			return true
		}
	}
	return false
}

// runnerConfigDir reports the working directory one runner config declares. It returns ""
// both when the config is absent and when that runner shape has no directory of its own —
// compose, helm, script, docker and friends locate their work by file, not by directory —
// and the caller treats "" as "nothing to check" rather than as the config directory, so an
// entry that never named a directory is not skipped on the strength of an unrelated path.
//
// A typed nil pointer stored in an any is not a nil interface, so every case has to guard
// its own pointer: the flat-field callers below pass fields that are usually nil.
func runnerConfigDir(cfg any) string {
	switch c := cfg.(type) {
	case *config.NativeRunnerConfig:
		if c != nil {
			return c.Dir
		}
	case *config.ProcessPluginConfig:
		if c != nil {
			return c.Dir
		}
	case *config.KustomizePluginConfig:
		if c != nil {
			return c.Dir
		}
	case *config.TiltPluginConfig:
		if c != nil {
			return c.Dir
		}
	case *config.VagrantPluginConfig:
		if c != nil {
			return c.Dir
		}
	case *config.ServerlessPluginConfig:
		if c != nil {
			return c.Dir
		}
	}
	return ""
}

// optionalSkipDir picks the directory whose existence decides whether an optional entry
// survives plan resolution, or "" when nothing about the entry names one.
//
// The runner config the plan settled on is asked first and answers on its own whenever it
// declares a directory. Only when it declares none — compose, helm, script, docker and
// friends locate their work by file, not by directory — does source.path stand in, because
// a source: entry that was never checked out is exactly the case optional: was added for.
//
// "" means "nothing to check", not "the config directory": an entry that named no directory
// must not be skipped on the strength of an unrelated path.
func optionalSkipDir(e *config.LifecycleEntry, runnerConfig any) string {
	if dir := runnerConfigDir(runnerConfig); dir != "" {
		return dir
	}
	if e.Source != nil && e.Source.Path != "" {
		return e.Source.Path
	}
	return ""
}
