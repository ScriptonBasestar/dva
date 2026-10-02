package config

// eachInteractionNode visits every command in the interaction tree — top-level entries and
// nested subcommands alike — passing each node with its dotted config path so a warning can
// name the exact YAML location.
//
// `subcommands` is recursive by construction (map[string]*InteractionCommand), the runner
// executes it to unbounded depth, and examples/full-stack.yml already ships three levels
// (`dva rails db migrate`). A check that walks only the first level therefore reports a
// mistake at the top and stays silent on the identical one below it, which is the reasoning
// warnInertProvisionSteps already records for its own walker.
//
// The visitor also receives the execution context inherited from the node's ancestors. That
// is not a convenience: runner.mergeInteraction copies every execution field parent → child
// and lets the child override only what it sets, so the node the runtime executes is not the
// node parsed from YAML. At depth 1 the two coincide — there is no ancestor — which is why
// checks written against raw nodes were correct until they were made to recurse, and wrong
// immediately afterwards. Any check that asks "does this node have X" must ask it of the
// inherited view. TASK-128.
func eachInteractionNode(interaction map[string]*InteractionCommand, visit func(path string, cmd *InteractionCommand, inherited inheritedExec)) {
	var walk func(path string, cmd *InteractionCommand, inherited inheritedExec)
	walk = func(path string, cmd *InteractionCommand, inherited inheritedExec) {
		if cmd == nil {
			return
		}
		visit(path, cmd, inherited)

		descend := inheritedExec{
			callable: inherited.callable || cmd.hasExecutionTarget(),
			runner:   firstNonEmptyStr(cmd.Runner, inherited.runner),
			pod:      firstNonEmptyStr(cmd.Pod, inherited.pod),
		}
		for subName, sub := range cmd.Subcommands {
			walk(path+".subcommands."+subName, sub, descend)
		}
	}

	for name, cmd := range interaction {
		walk("interaction."+name, cmd, inheritedExec{})
	}
}

// inheritedExec is what a node receives from its ancestors under
// runner.mergeInteraction's rules. Only the fields the warnings below actually consult are
// carried; adding a check that depends on another inherited field means adding it here too.
type inheritedExec struct {
	// callable records whether any ancestor supplies something to execute, which the child
	// inherits and can therefore be invoked with.
	callable bool
	// runner and pod are the nearest ancestor's effective values, i.e. what the child ends
	// up running under when it does not set its own.
	runner string
	pod    string
}

func firstNonEmptyStr(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
