package cli

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ScriptonBasestar/dva/internal/compare"
	"github.com/ScriptonBasestar/dva/internal/config"
)

var compareWait bool

var compareCmd = &cobra.Command{
	Use:   "compare",
	Short: "Run one plan in several checkouts side by side",
	Long: `Start the same plan from several refs at once, look at them, and take them down.

  dva compare up <name> [variant=ref ...]   start the variants of compare.<name>
  dva compare status <name>                 read what is running, read-only
  dva compare down <name> [variant ...]     stop variants and reclaim their checkouts

<name> is a compare: declaration in dva.yml. Each variant=ref pair checks the ref out as a
detached snapshot under .sb/dva/variants/ and starts the declaration's per_variant entries
there, with the variant's own ports. The shared entries start once, from this checkout.
Without pairs, up uses the declaration's variants:.

A variant source of worktree:<branch> uses the existing work worktree that has that branch
checked out. dva never creates or removes it: up starts the per_variant entries there and down
stops them, leaving the worktree and branch in place. An unknown branch, several candidates or a
failed lookup start nothing.

up never moves a busy port and never rolls back: a variant that fails is reported with the
commands to continue or undo, and what already started keeps running. Run up again to start
only the variants that are missing. down removes a snapshot only when it is clean.
--dry-run shows what would happen; --json prints the same facts for a program.`,
}

var compareUpCmd = &cobra.Command{
	Use:   "up <name> [variant=ref ...]",
	Short: "Start the variants of a comparison",
	Long:  "Start compare.<name>: the shared entries once from this checkout, then each variant=ref as a\ndetached snapshot with its own ports. Pass name=ref pairs, or none to use the declaration\x27s variants.\nA busy port, an unknown ref or a short port list fails before anything is created. A variant that\nfails to start leaves the others running; run the printed down command to undo or up again to\nstart only what is missing.",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runCompareUp(compareContext(cmd), mustLoadConfig(), args[0], args[1:], compareWait)
	},
}

var compareStatusCmd = &cobra.Command{
	Use:   "status <name>",
	Short: "Show what a comparison started, read-only",
	Long:  "Read the record of an up and re-verify it against the checkouts, Git and the running entries.\nReports stale variants and sources that now name another commit. Changes nothing.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runCompareStatus(compareContext(cmd), mustLoadConfig(), args[0])
	},
}

var compareDownCmd = &cobra.Command{
	Use:   "down <name> [variant ...]",
	Short: "Stop variants and reclaim their snapshot checkouts",
	Long:  "Stop each variant\x27s per_variant entries, then remove its snapshot checkout. A worktree:\nvariant only has its entries stopped; its worktree stays. A snapshot with\nuncommitted or untracked changes is kept and reported. With no variant names it takes every\nvariant down and then stops the shared entries.",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runCompareDown(compareContext(cmd), mustLoadConfig(), args[0], args[1:])
	},
}

func init() {
	compareCmd.GroupID = "lifecycle"
	compareUpCmd.Flags().BoolVar(&compareWait, "wait", false, "Wait for each variant's entries to report ready")
	compareCmd.AddCommand(compareUpCmd, compareStatusCmd, compareDownCmd)
	setGroupParentBehavior(compareCmd)
	rootCmd.AddCommand(compareCmd)
}

func compareContext(cmd *cobra.Command) context.Context {
	if ctx := cmd.Context(); ctx != nil {
		return ctx
	}
	return context.Background()
}

// variantSpec is one `name=ref` request, in the order it was given.
type variantSpec struct {
	Name   string
	Source string
}

// comparePreflight is a failure before anything was created or started. result is optional: when
// the request was resolved far enough to show, it travels with the failure.
func comparePreflight(result *compareResult, format string, args ...any) error {
	return compareFailure(codeComparePreflight, result, format, args...)
}

// compareDeclaration finds the compare: entry a command names.
func compareDeclaration(c *config.Config, name string) (*config.CompareConfig, error) {
	if cmp := c.Compare[name]; cmp != nil {
		return cmp, nil
	}
	names := make([]string, 0, len(c.Compare))
	for n := range c.Compare {
		names = append(names, n)
	}
	sort.Strings(names)
	if len(names) == 0 {
		return nil, comparePreflight(nil, "no compare: declaration %q — dva.yml declares none", name)
	}
	return nil, comparePreflight(nil, "no compare: declaration %q — declared: %s", name, strings.Join(names, ", "))
}

// parseVariantSpecs reads `name=ref` arguments, or falls back to the declaration's variants:
// in name order. The order fixes which port position a new variant takes, so it must not
// depend on map iteration.
func parseVariantSpecs(compareName string, cmp *config.CompareConfig, args []string) ([]variantSpec, error) {
	var specs []variantSpec
	if len(args) == 0 {
		for n, src := range cmp.Variants {
			specs = append(specs, variantSpec{Name: n, Source: src})
		}
		if len(specs) == 0 {
			return nil, fmt.Errorf("no variants: pass name=ref pairs, or declare compare.%s.variants", compareName)
		}
		sort.Slice(specs, func(i, j int) bool { return specs[i].Name < specs[j].Name })
	}
	for _, a := range args {
		n, src, ok := strings.Cut(a, "=")
		if !ok || n == "" || src == "" {
			return nil, fmt.Errorf("variant argument %q must be name=ref", a)
		}
		specs = append(specs, variantSpec{Name: n, Source: src})
	}
	seen := map[string]bool{}
	for _, s := range specs {
		switch {
		case !compare.ValidName(s.Name):
			return nil, fmt.Errorf("variant name %q must start with a letter or digit and use only letters, digits, - and _", s.Name)
		case seen[s.Name]:
			return nil, fmt.Errorf("variant %q is given twice", s.Name)
		case compare.IsTargetSource(s.Source):
			// A git ref cannot contain ":", so this is a provider target such as worktree:ID.
			if _, err := compare.TargetID(s.Source); err != nil {
				return nil, fmt.Errorf("variant %s: %w", s.Name, err)
			}
		}
		seen[s.Name] = true
	}
	return specs, nil
}

var shellSafe = regexp.MustCompile(`^[A-Za-z0-9._/@=+-]+$`)

// shellWord quotes s for the commands compare prints for a person to paste.
func shellWord(s string) string {
	if shellSafe.MatchString(s) {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// downCommand is the exact command that takes variants of name down; no variants means all.
func downCommand(name string, variants []string) string {
	parts := append([]string{"dva", "compare", "down", name}, variants...)
	for i := range parts {
		parts[i] = shellWord(parts[i])
	}
	return strings.Join(parts, " ")
}

func varsOf(ports map[string]int) map[string]string {
	if len(ports) == 0 {
		return nil
	}
	vars := make(map[string]string, len(ports))
	for k, v := range ports {
		vars[k] = fmt.Sprint(v)
	}
	return vars
}
