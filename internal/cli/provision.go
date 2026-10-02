package cli

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ScriptonBasestar/dva/internal/config"
	"github.com/ScriptonBasestar/dva/internal/output"
)

var provisionList bool

var provisionCmd = &cobra.Command{
	Use:   "provision [PROFILE]",
	Short: "Execute the provisioning steps defined in 'dva.yml'",
	Long: `Run one profile's steps from dva.yml's provision: section — shell commands, plus
compose_up/compose_exec/compose_run steps executed through the stack's compose backend.

Without PROFILE it runs the "default" profile when declared, falling back to
default_profile, then to the single declared profile when there is exactly one; with
several profiles and no match it lists what is available with a "did you mean" hint.
--list prints the declared profiles without running any of them. --dry-run prints each
step's resolved command without executing it. A profile owned by a subprojects: entry
runs against that subproject's own vars/environment/env_file.

See USAGE.md's "provision" section for examples.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c := mustLoadConfig()

		// --list: show available profiles and exit
		if provisionList {
			return listProvisionProfiles(c)
		}

		requested := "default"
		explicit := len(args) > 0
		if explicit {
			requested = args[0]
		}

		profile, steps, err := resolveProvisionProfile(c.Provision.Profiles, c.Provision.DefaultProfile, requested, explicit)
		if err != nil {
			return err
		}

		// Owner and its environment are resolved after the profile name is settled and
		// before the first step runs. An imported profile executes against the child that
		// declared it, so its steps see the child's vars, environment: and env_file and run
		// from the child config directory; a root profile is unchanged. Resolving here also
		// keeps a broken root env_file from blocking a purely child-owned profile — the
		// warning-and-continue policy itself is TASK-248's to change (TASK-264).
		rt, err := resolveProvisionRuntime(c, profile)
		if err != nil {
			return err
		}
		e, owner := rt.env, rt.config

		if dryRun {
			fmt.Printf("🔍 DRY RUN — showing execution plan for profile: %s\n\n", profile)
		} else {
			fmt.Printf("🚀 Running provision profile: %s\n\n", profile)
		}

		// Execute steps with parallel batch support
		batches := groupParallelBatches(steps)
		stepOffset := 0
		for _, batch := range batches {
			if len(batch) == 1 || !batch[0].Parallel {
				// Sequential execution
				for _, step := range batch {
					if err := executeProvisionStep(e, owner, step, stepOffset, len(steps), dryRun); err != nil {
						return err
					}
					stepOffset++
				}
			} else {
				// Parallel execution
				if err := executeParallelBatch(e, owner, batch, stepOffset, len(steps), dryRun); err != nil {
					return err
				}
				stepOffset += len(batch)
			}
		}

		if dryRun {
			fmt.Println("\n🔍 Dry run complete — no commands were executed.")
		} else {
			// Write provision marker so `dva up` knows this profile was run
			writeProvisionMarker(c.FileDir(), profile)
			fmt.Println("\n✅ Provision complete!")
		}
		return nil
	},
}

func init() {
	provisionCmd.Flags().BoolVarP(&provisionList, "list", "l", false, "List available provision profiles")
}

// groupParallelBatches groups steps into batches. Consecutive steps with
// Parallel=true form a single batch; non-parallel steps are each their own batch.
func groupParallelBatches(steps []config.ProvisionItem) [][]config.ProvisionItem {
	var batches [][]config.ProvisionItem
	var current []config.ProvisionItem

	for _, step := range steps {
		if step.Parallel {
			current = append(current, step)
		} else {
			if len(current) > 0 {
				batches = append(batches, current)
				current = nil
			}
			batches = append(batches, []config.ProvisionItem{step})
		}
	}
	if len(current) > 0 {
		batches = append(batches, current)
	}
	return batches
}

// resolveProvisionProfile resolves which provision profile to use.
// Priority: exact match → default_profile alias → single-profile auto → error.
func resolveProvisionProfile(provision map[string][]config.ProvisionItem, defaultProfile string, requested string, explicit bool) (string, []config.ProvisionItem, error) {
	// Direct lookup
	if steps, ok := provision[requested]; ok {
		return requested, steps, nil
	}

	// No provision defined at all
	if len(provision) == 0 {
		return "", nil, fmt.Errorf("no provision commands defined in dva.yml")
	}

	// Implicit fallbacks only (user did not specify a profile name)
	if requested == "default" && !explicit {
		// default_profile alias (explicit config, works with any number of profiles)
		if defaultProfile != "" {
			if steps, ok := provision[defaultProfile]; ok {
				fmt.Fprintf(os.Stderr, "⚠ Profile 'default' not found, using '%s' (default_profile)\n\n", defaultProfile)
				return defaultProfile, steps, nil
			}
			fmt.Fprintf(os.Stderr, "⚠ default_profile '%s' not found in provision profiles — ignoring\n\n", defaultProfile)
		}

		// Auto-fallback: exactly 1 profile
		if len(provision) == 1 {
			for k, v := range provision {
				fmt.Fprintf(os.Stderr, "⚠ Profile 'default' not found, using '%s' (only available profile)\n\n", k)
				return k, v, nil
			}
		}
	}

	// Build available list
	available := make([]string, 0, len(provision))
	for k := range provision {
		available = append(available, k)
	}
	sort.Strings(available)

	// "Did you mean?" suggestion
	var suggestions []string
	for _, name := range available {
		if levenshtein(requested, name) <= 2 {
			suggestions = append(suggestions, name)
		}
	}

	var msg strings.Builder
	fmt.Fprintf(&msg, "provision profile '%s' not found. Available: %s", requested, strings.Join(available, ", "))
	if len(suggestions) > 0 {
		msg.WriteString("\n\nDid you mean?")
		for _, s := range suggestions {
			fmt.Fprintf(&msg, "\n  dva provision %s", s)
		}
	}

	return "", nil, fmt.Errorf("%s", msg.String())
}

// listProvisionProfiles prints provision profiles in the requested format.
func listProvisionProfiles(c *config.Config) error {
	if len(c.Provision.Profiles) == 0 {
		fmt.Println("No provision profiles defined.")
		return nil
	}

	keys := make([]string, 0, len(c.Provision.Profiles))
	for k := range c.Provision.Profiles {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	if jsonOutput {
		return printProvisionJSON(c, c.Provision.Profiles, c.Provision.DefaultProfile, keys)
	}
	return printProvisionTable(c.Provision.Profiles, c.Provision.DefaultProfile, keys)
}

func provisionAliasGroups(pc *config.ProvisionConfig) map[string][]string {
	groups := make(map[string][]string)
	if pc == nil {
		return groups
	}
	for k := range pc.Profiles {
		canonical := pc.ProfileCanonicalAddress(k)
		if canonical == "" || k == canonical {
			continue
		}
		groups[canonical] = append(groups[canonical], k)
	}
	for canonical := range groups {
		sort.Strings(groups[canonical])
	}
	return groups
}

func printProvisionTable(provision map[string][]config.ProvisionItem, defaultProfile string, keys []string) error {
	maxName := len("PROFILE")
	for _, k := range keys {
		if len(k) > maxName {
			maxName = len(k)
		}
	}
	if defaultProfile != "" {
		maxName += 2 // room for " *" suffix
	}

	fmt.Printf("%-*s  %5s  %s\n", maxName, "PROFILE", "STEPS", "FIRST STEP")
	for _, k := range keys {
		steps := provision[k]
		desc := firstStepDescription(steps)
		display := k
		if k == defaultProfile {
			display = k + " *"
		}
		fmt.Printf("%-*s  %5d  %s\n", maxName, display, len(steps), desc)
	}
	if defaultProfile != "" {
		fmt.Printf("\n* default profile\n")
	}
	return nil
}

func printProvisionJSON(c *config.Config, provision map[string][]config.ProvisionItem, defaultProfile string, keys []string) error {
	var aliasGroups map[string][]string
	if c != nil {
		aliasGroups = provisionAliasGroups(&c.Provision)
	}
	profiles := make(map[string]any, len(keys))
	for _, k := range keys {
		steps := provision[k]
		owner := rootOwnerName
		var canonical string
		if c != nil {
			if o := c.Provision.ProfileSubproject(k); o != "" {
				owner = o
			}
			canonical = c.Provision.ProfileCanonicalAddress(k)
		}
		prof := map[string]any{
			"steps":      len(steps),
			"first_step": firstStepDescription(steps),
			"owner":      owner,
		}
		if canonical != "" {
			if k == canonical {
				if aliases := aliasGroups[k]; len(aliases) > 0 {
					prof["aliases"] = aliases
				}
			} else {
				prof["alias_of"] = canonical
			}
		}
		profiles[k] = prof
	}
	result := map[string]any{"profiles": profiles}
	if defaultProfile != "" {
		result["default_profile"] = defaultProfile
	}
	return output.PrintJSON(result)
}

func firstStepDescription(steps []config.ProvisionItem) string {
	if len(steps) == 0 {
		return ""
	}
	s := steps[0]
	if s.Step != "" {
		return s.Step
	}
	if s.Raw != "" {
		return s.Raw
	}
	if s.Echo != "" {
		return s.Echo
	}
	return ""
}
