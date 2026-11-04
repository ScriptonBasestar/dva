package config

import (
	"fmt"
	"sort"
	"strings"
)

// walkLocalInteractionMetadata preserves declaration paths rather than flattening
// them: a literal key containing spaces is distinct from a nested command path.
// Imported metadata belongs to its declaring child, never the parent's namespace.
func (c *Config) walkLocalInteractionMetadata(visit func([]string, *InteractionCommand)) {
	var walk func([]string, map[string]*InteractionCommand)
	walk = func(parent []string, commands map[string]*InteractionCommand) {
		names := make([]string, 0, len(commands))
		for name := range commands {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			cmd := commands[name]
			if cmd == nil || cmd.SubprojectName != "" || cmd.OwnerConfig(c) != c {
				continue
			}
			path := append(append([]string(nil), parent...), name)
			visit(path, cmd)
			walk(path, cmd.Subcommands)
		}
	}
	walk(nil, c.Interaction)
}

func (c *Config) declaredSubprojectShadow(name string) bool {
	idx := strings.Index(name, ":")
	if idx <= 0 || IsReservedCommand(name[:idx]) {
		return false
	}
	_, exists := c.Subprojects[name[:idx]]
	return exists
}

func (c *Config) intentionalSubprojectShadow(name string, cmd *InteractionCommand) bool {
	return cmd != nil && cmd.SubprojectName == "" && cmd.OwnerConfig(c) == c &&
		cmd.IntentionalShadow != nil && *cmd.IntentionalShadow && c.declaredSubprojectShadow(name)
}

// IntentionalSubprojectShadows exposes acknowledged local collisions for the
// validator's mandatory suppression count and explanation, without changing routing.
func (c *Config) IntentionalSubprojectShadows() []string {
	var names []string
	for name, cmd := range c.Interaction {
		if c.intentionalSubprojectShadow(name, cmd) {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

func (c *Config) warnUnusedIntentionalShadows() []string {
	var warnings []string
	c.walkLocalInteractionMetadata(func(path []string, cmd *InteractionCommand) {
		if cmd.IntentionalShadow == nil || !*cmd.IntentionalShadow {
			return
		}
		if len(path) == 1 && c.declaredSubprojectShadow(path[0]) {
			return
		}
		warnings = append(warnings, fmt.Sprintf("interaction.%s.intentional_shadow: no local literal key shadows a declared subproject here; remove the stale declaration", strings.Join(path, ".subcommands.")))
	})
	return warnings
}

// validateSuggestionAliases enforces one declaring interaction per Makefile
// target. Target existence and route reachability belong to CLI discovery, which
// can inspect the actual Makefile and the resolved InteractionTree.
func (c *Config) validateSuggestionAliases() error {
	owners := make(map[string]string)
	var errs []error
	c.walkLocalInteractionMetadata(func(path []string, cmd *InteractionCommand) {
		if cmd.SuggestionAliases == nil {
			return
		}
		owner := "interaction." + strings.Join(path, ".subcommands.")
		for _, target := range cmd.SuggestionAliases.Makefile {
			if strings.TrimSpace(target) == "" {
				errs = append(errs, fmt.Errorf("%s.suggestion_aliases.makefile: target must not be blank", owner))
				continue
			}
			if previous, exists := owners[target]; exists {
				errs = append(errs, fmt.Errorf("makefile target %q is claimed by %s and %s; declare exactly one owning interaction", target, previous, owner))
				continue
			}
			owners[target] = owner
		}
	})
	return joinValidationErrors(errs)
}
