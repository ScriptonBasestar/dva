package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ScriptonBasestar/dva/internal/config"
	"github.com/ScriptonBasestar/dva/internal/output"
)

// validateNoticeWriter is where validate writes [warn]/[fixed]/[error] lines that
// accompany a successful (or soft-fail) pass.
//
// Rule (TASK-142): on the human path, notices that qualify the ✅ verdict share stdout
// with it so a reader of one stream sees both. On --json, stdout is reserved for the
// single document and notices stay on stderr so the document is not corrupted by prose.
// Errors that abort validation still use stderr directly where they are emitted.
func validateNoticeWriter() io.Writer {
	if jsonOutput {
		return os.Stderr
	}
	return os.Stdout
}

var validateStrict bool

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate the syntax and schema of 'dva.yml'",
	// Set here in the struct literal, not from a later init(): validate_alias.go copies
	// this Long by value into the top-level 'dva validate' alias inside its own init(), so
	// assigning it afterward would leave that alias's Long empty while this command's own
	// Long looked fixed.
	Long: `Check dva.yml against its JSON schema, then run semantic checks: unrunnable
compose runner commands (hard failure), compose file project-name mismatches, interaction
name collisions, config drift, and other semantic warnings. Reached as both
'dva validate' and 'dva config validate'.

--fix rewrites compose file 'name:' mismatches (and creates a missing devcontainer.json
when a devcontainer: section is declared and enabled) instead of only reporting them.
--strict turns config-drift, semantic, and interaction-collision warnings into a failing
exit code; without it those are reported but do not fail validation.

See USAGE.md's "config validate" section for the full list of semantic checks.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c := loadConfigForValidate()
		report := newValidateReport(c)

		// Hard errors are collected, not returned, so one malformed entry does not hide
		// the next one or the warnings below it. Every hard error is printed together at
		// the end (TASK-305); the exit code is unchanged.
		var hard []error
		for _, p := range c.LoadProblems() {
			hard = append(hard, errors.New(p))
		}

		if err := c.Validate(); err != nil {
			hard = append(hard, err)
		}

		// env_bridge's origin and version rules (TASK-281 §3-2) report only from
		// here and from `dva config env seal/show` — never from an ordinary
		// lifecycle command, so a policy declaration about the secret surface
		// cannot brick `dva up`.
		if err := checkEnvBridgeOriginAndVersion(c); err != nil {
			hard = append(hard, err)
		}

		// A hard failure rather than a warning. The schema accepts this config, and then
		// every compose runner rejects it at the moment it tries to run — so `dva validate`
		// exiting 0 here is the whole defect: a green check that is evidence about the
		// checker, not the config.
		if problems := detectUnrunnableComposeCommands(c); len(problems) > 0 {
			for _, p := range problems {
				fmt.Fprintf(os.Stderr, "[error] compose: %s\n", p)
			}
			hard = append(hard, fmt.Errorf("%d compose runner command(s) contain no command word", len(problems)))
		}

		notice := validateNoticeWriter()

		// Check compose file project name alignment
		warnings := c.ValidateComposeProjectNames()
		fix, _ := cmd.Flags().GetBool("fix")

		if fix {
			fixComposeNameWarnings(c, warnings)
		} else {
			printComposeNameWarnings(notice, warnings)
			// Only when they were reported: --fix rewrote the files, so the mismatch no
			// longer exists and putting it in the report would describe a fixed state as
			// an outstanding warning.
			for _, w := range warnings {
				report.addComposeNameWarning(w)
			}
		}

		// Semantic warnings (version, health checks, duplicate commands, etc.)
		semanticWarnings := c.ValidateWarnings()
		for _, w := range semanticWarnings {
			_, _ = fmt.Fprintf(notice, "[warn] semantic: %s\n", w)
		}
		report.add("semantic", semanticWarnings...)

		collisionWarnings := detectInteractionCollisionWarnings(c)
		for _, w := range collisionWarnings {
			_, _ = fmt.Fprintf(notice, "[warn] interaction: %s\n", w)
		}
		report.add("interaction_collision", collisionWarnings...)

		suppressed := &suppressionSummary{}

		driftWarnings, driftSuppressed := detectConfigDriftWarningsWithSuppressions(c)
		suppressed.merge(driftSuppressed)
		printConfigDriftWarnings(notice, driftWarnings)
		report.add("config_drift", driftWarnings...)

		suggestionWarnings, suggestionSuppressed := detectConfigSuggestionWarningsWithSuppressions(c)
		suppressed.merge(suggestionSuppressed)
		printConfigSuggestionWarnings(notice, suggestionWarnings)
		report.add("config_suggestion", suggestionWarnings...)

		// A pattern that hides nothing is reported whether or not --show-ignored was asked
		// for: it is a defect in dva.yml, not a detail about this run (docs/56 §6-5).
		printStaleIgnoreWarnings(notice, suppressed.stale)
		report.add("ignore_stale", suppressed.stale...)

		if showIgnored, _ := cmd.Flags().GetBool("show-ignored"); showIgnored {
			printSuppressedItems(notice, suppressed)
		}
		report.addSuppressions(suppressed)

		if err := failValidation(&report, dedupeErrors(hard)); err != nil {
			return err
		}

		if validateStrict && (len(driftWarnings) > 0 || len(semanticWarnings) > 0 || len(collisionWarnings) > 0) {
			return report.fail(fmt.Errorf("config warnings detected; review warnings above or run 'am run dva-improve'"))
		}

		// Check devcontainer sync
		if len(c.Devcontainer) > 0 && isDevcontainerEnabled(c.Devcontainer) {
			dcPath := filepath.Join(c.FileDir(), ".devcontainer", "devcontainer.json")
			if _, err := os.Stat(dcPath); os.IsNotExist(err) {
				if fix {
					if err := writeDevcontainerFiles(c.Devcontainer, c.AllComposeFiles(), c.FileDir()); err != nil {
						fmt.Fprintf(os.Stderr, "[error] devcontainer: %v\n", err)
					} else {
						_, _ = fmt.Fprintf(notice, "[fixed] created .devcontainer/devcontainer.json\n")
					}
				} else {
					_, _ = fmt.Fprintf(notice, "[warn] devcontainer section found but .devcontainer/devcontainer.json missing\n")
					_, _ = fmt.Fprintf(notice, "       → run: dva config validate --fix\n")
					report.add("devcontainer", "devcontainer section found but .devcontainer/devcontainer.json missing\n  → run: dva config validate --fix")
				}
			}
		}

		if suggestIgnore, _ := cmd.Flags().GetBool("suggest-ignore"); suggestIgnore && !jsonOutput {
			// Printed raw, and only on the human path: this block is meant to be copied
			// into dva.yml rather than read. On this path the notice writer is stdout
			// anyway; the guard is about --json, which reserves stdout for the single
			// document that a YAML fragment beside it would corrupt.
			fmt.Print(suggestIgnoreBlock(suppressed.suggested))
		}

		if jsonOutput {
			return output.PrintJSON(report)
		}
		// The suffix is not optional and has no flag to turn it off. It is what lets every
		// ignore surface above exist: an ignored finding stays counted, so "valid" never
		// silently means "valid once N findings were hidden" (docs/56 §6-4).
		fmt.Printf("✅ dva.yml is valid%s\n", suppressed.summarySuffix())
		return nil
	},
}

// loadConfigForValidate loads the config the way every other command does, and when that
// fails on a stack-entry shape problem, loads it again recording those problems instead
// (config.CollectEntryProblems) so the rest of the diagnostics still run. A file that
// cannot be parsed at all, or fails a check the lenient load does not defer, exits
// through mustLoadConfig's path exactly as before.
func loadConfigForValidate() *config.Config {
	c, err := loadConfig()
	if err == nil {
		return c
	}
	cfg = nil
	lenient, lenientErr := config.Load(".", config.CollectEntryProblems())
	if lenientErr != nil {
		return mustLoadConfig()
	}
	cfg = lenient
	checkGitignoreForWarning(lenient.FileDir())
	return lenient
}

// dedupeErrors flattens joined errors (config.Validate returns one ValidationErrors for
// all its findings) so each finding is its own numbered item, and drops repeated
// messages: the lenient load and Validate both run validateEntrySource, so a missing
// source would otherwise be listed twice.
func dedupeErrors(errs []error) []error {
	seen := make(map[string]bool, len(errs))
	var out []error
	var walk func([]error)
	walk = func(list []error) {
		for _, err := range list {
			if joined, ok := err.(interface{ Unwrap() []error }); ok {
				walk(joined.Unwrap())
				continue
			}
			if seen[err.Error()] {
				continue
			}
			seen[err.Error()] = true
			out = append(out, err)
		}
	}
	walk(errs)
	return out
}

// failValidation returns the failing error, or nil when there are none. Several hard
// errors come back as one numbered list (validationFailure) so root's single "ERROR:"
// line shows every problem after the warnings, and `--json` still gets each member
// through Unwrap.
func failValidation(report *validateReport, hard []error) error {
	switch len(hard) {
	case 0:
		return nil
	case 1:
		return report.fail(hard[0])
	}
	return report.fail(validationFailure(hard))
}

// validationFailure renders several hard errors as a numbered list with a count, so a
// twelve-line schema error and a one-line hook error next to it read as two items.
type validationFailure []error

func (v validationFailure) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%d errors found in this config:", len(v))
	for i, err := range v {
		fmt.Fprintf(&b, "\n\n[%d] %s", i+1, err.Error())
	}
	return b.String()
}

func (v validationFailure) Unwrap() []error { return v }

func init() {
	addValidateFlags(validateCmd)
	configCmd.AddCommand(validateCmd)
}
