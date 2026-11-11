package cli

import (
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"

	"github.com/ScriptonBasestar/dva/internal/compare"
	"github.com/ScriptonBasestar/dva/internal/config"
	"github.com/ScriptonBasestar/dva/internal/lifecycle"
)

// variantVar is the plan var a config-mode variant gets, holding the variant's name. A
// per_variant entry uses it to keep whatever it writes into the shared checkout apart from the
// other variants' (a build output directory, a cache path, a database file).
const variantVar = "DVA_VARIANT"

// variantVars are the plan vars of one variant: its assigned ports, plus DVA_VARIANT in config
// mode. The other modes have a checkout of their own and need no such name.
func variantVars(v *compare.Variant) map[string]string {
	vars := varsOf(v.Ports)
	if v.Mode != compare.ModeConfig {
		return vars
	}
	if vars == nil {
		vars = map[string]string{}
	}
	vars[variantVar] = v.Name
	return vars
}

// buildStateTools are the commands known to keep build state inside the checkout (`.next`,
// `.nuxt`, `.astro`, `.cache`) and to fail or corrupt each other when two run from one
// directory. dva cannot see what an arbitrary command writes, so this is a list of ones it
// knows about, not a proof that every other command is safe.
var buildStateTools = regexp.MustCompile(`(^|[\s;&|/])(next|nuxt|nuxi|astro|gatsby)(\s|$)`)

// configModeProblems lists the reasons the config-mode variants of an up cannot run as asked.
// Two variants in one directory share its build output, and dva does not split that for a
// command it knows to write there: it stops before starting anything and names the entry.
func configModeProblems(c *config.Config, cmp *config.CompareConfig, variants []*upVariant) []string {
	var names []string
	for _, v := range variants {
		if v.target != nil && v.target.Mode == compare.ModeConfig && !v.skip {
			names = append(names, v.spec.Name)
		}
	}
	if len(names) == 0 || len(cmp.PerVariant) == 0 {
		return nil
	}
	var problems []string
	if ev, ok := os.LookupEnv(variantVar); ok {
		// The OS environment has the highest var priority, so it would replace every variant's
		// own name and put the variants back on one build state. Same refusal as a port var.
		problems = append(problems, fmt.Sprintf("config variants %s: environment sets %s=%s, which would override each variant's own value; unset it", strings.Join(names, ","), variantVar, ev))
	}
	plan, err := lifecycle.ResolvePlan(c, cmp.Plan, nil)
	if err != nil {
		return append(problems, fmt.Sprintf("config variants %s: cannot inspect plan %q for shared build state: %v", strings.Join(names, ","), cmp.Plan, err))
	}
	for _, e := range plan.Entries {
		if !slices.Contains(cmp.PerVariant, e.Name) {
			continue
		}
		text := runnerText(e.RunnerConfig)
		if tool := buildStateTools.FindStringSubmatch(text); tool != nil && !strings.Contains(text, variantVar) {
			problems = append(problems, fmt.Sprintf(
				"entry %s runs %s, which keeps build state in the checkout, so config variants %s would share it: "+
					"make its command, dir or env use $%s (for example a per-variant build directory), or use snapshot or worktree variants",
				e.Name, tool[2], strings.Join(names, ","), variantVar))
		}
	}
	return problems
}

// runnerText is the declared command text of a process-like runner: everything that can name a
// tool or the variant var, before interpolation.
func runnerText(rc any) string {
	switch r := rc.(type) {
	case *config.NativeRunnerConfig:
		parts := []string{r.Dir, r.Build, r.PostBuild, r.Run}
		for k, v := range r.Env {
			parts = append(parts, k+"="+v)
		}
		return strings.Join(parts, "\n")
	case *config.ProcessPluginConfig:
		return strings.Join([]string{r.Dir, r.Command}, "\n")
	}
	return ""
}
