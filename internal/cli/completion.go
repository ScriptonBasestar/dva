package cli

import (
	"github.com/spf13/cobra"

	"github.com/ScriptonBasestar/dva/internal/config"
)

func init() {
	// Dynamic completion for 'run CMD': first arg suggests interaction command names.
	runCmd.ValidArgsFunction = func(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		c, err := config.Load(".")
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return completionCandidates(c), cobra.ShellCompDirectiveNoFileComp
	}
}

// completionCandidates returns every address form that actually routes to something,
// for `dva run`'s positional argument:
//   - every key literal in c.Interaction, which already covers root-declared entries,
//     the `<name>/<key>` import address (resolveSubprojectImports writes the canonical
//     name and any `as:` alias directly into this map), and reserved/builtin-shadowing
//     names — nothing more needs to be added for those two forms.
//   - the `<name>:<key>` shorthand for every subproject, computed here because run.go's
//     colon split (run.go:141) never becomes a literal c.Interaction key: it resolves
//     the project by name and looks the key up in that subproject's own config, subject
//     to that subproject's exclude_tags and RejectsInteractionKey rejection — a wider set
//     than what import: chose to expose under `<name>/<key>`. Each subproject is loaded
//     on its own and a failing one is skipped, so one absent checkout costs its own
//     colon forms and no others.
func completionCandidates(c *config.Config) []string {
	names := make([]string, 0, len(c.Interaction))
	for name := range c.Interaction {
		names = append(names, name)
	}

	for projectName, sub := range c.Subprojects {
		// One load per subproject, not one LoadSubprojects call over the whole map.
		// That call discards partial results on any failure (see its contract), so a
		// single sibling checkout absent on this machine — or one pinned to an
		// incompatible config version — removed every *other* subproject's `p:k` form
		// from completion while `dva run p:k` still routed it. Measured: two
		// subprojects, one path missing, completion offered zero colon forms.
		// run.go's own routing loads one at a time for the same reason
		// (loadSubprojectConfig), so this now matches what `dva run` actually accepts.
		subs, err := config.LoadSubprojects(c.FileDir(), map[string]config.SubprojectConfig{projectName: sub})
		if err != nil {
			continue
		}
		subCfg := subs[projectName]
		if subCfg == nil {
			continue
		}
		for key := range subCfg.FilterInteractions(sub.ExcludeTags) {
			if rejected, _, _ := subCfg.RejectsInteractionKey(key); rejected {
				continue
			}
			names = append(names, projectName+":"+key)
		}
	}
	return names
}

// subprojectNameCompletion offers `--project`'s value: every declared subproject name,
// whether or not it declares an import: block — `dva run --project <p> <key>` (and
// `dva ls --project <p>`) route by subproject name alone, independent of import.
//
// Names are offered as declared, without loading them, which is deliberately unlike
// completionCandidates above: a subproject whose path is absent still completes here, and
// `dva run --project <p> <key>` then reports the load error. That beats silence — the user
// wrote the name in their own dva.yml, and a diagnosis is more use than an empty list. The
// colon form cannot do the same because it needs the child's keys to offer anything at all.
//
// It is
// registered by runCmd's and lsCmd's own init() functions in run.go/list.go, right after
// each defines its "project" flag, not here — RegisterFlagCompletionFunc requires the
// flag to already exist, and Go does not guarantee this file's init() runs first.
func subprojectNameCompletion(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	c, err := config.Load(".")
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	names := make([]string, 0, len(c.Subprojects))
	for name := range c.Subprojects {
		names = append(names, name)
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}
