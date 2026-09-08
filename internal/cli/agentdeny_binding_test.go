package cli

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/ScriptonBasestar/dva/internal/agentdeny"
)

// TestGatedCommandsCoverEveryGatedSurface binds internal/agentdeny.GatedCommands to the
// command tree that actually exists (TASK-337, closing the follow-up TASK-286 §criterion 1
// named but could not implement).
//
// `make check-generate` proves only that docs/agent-deny-rules.md matches GatedCommands —
// the generated document agreeing with the list it was generated from. Nothing compared
// that list to the CLI, so adding a command behind an env_bridge gate and forgetting its
// deny rule stayed green through every gate this repo has. The command would ship gated
// against a human and wide open to an agent runtime, which is the one thing the deny-rule
// policy exists to prevent.
//
// The surface is derived from the source rather than declared, on purpose. A registry the
// author has to remember to update just moves the forgetting one level up; the point of
// this check is that there is nothing new to remember. What a gated command cannot avoid
// doing is refusing with a `*_not_enabled` code — that refusal IS the gate — so this reads
// the package's own AST, finds every function that reaches such a refusal, walks up to the
// cobra commands whose RunE reaches it, and reconstructs each one's argv from the same
// AddCommand calls that build the real tree.
func TestGatedCommandsCoverEveryGatedSurface(t *testing.T) {
	pkg := parseCLIPackage(t)

	gateCodes := gateCodeConstants(pkg)
	if len(gateCodes) == 0 {
		t.Fatal("no `*_not_enabled` code constant found; the gate detector below would report an empty surface and pass vacuously")
	}

	gatedFuncs := gatedFunctions(pkg, gateCodes)
	if len(gatedFuncs) == 0 {
		t.Fatal("no function reaches a `*_not_enabled` refusal; the detector is broken, not the code")
	}

	surface := gatedCommandArgv(t, pkg, gatedFuncs)
	if len(surface) == 0 {
		t.Fatal("no cobra command reaches a gate; the detector is broken, not the code")
	}

	// Forward direction — the one that matters. A gated command with no rule is a command
	// an agent runtime is never told to refuse.
	for _, argv := range surface {
		if !argvIsGated(argv) {
			t.Errorf("`%s` is gated in internal/cli but has no internal/agentdeny.GatedCommands entry:\n"+
				"  no deny rule is generated for it, so an agent runtime is never told to refuse it.\n"+
				"  Add a GatedCommand with Argv: %q to internal/agentdeny/rules.go and run `make generate`.",
				argv, argv)
		}
	}

	// Reverse direction. TASK-337 called this "worth reporting" and not blocking because it
	// fails safe; it is a hard failure here anyway. A rule naming an argv the tree does not
	// have denies a command that cannot be run, which is exactly as untrue as a missing rule
	// and is how docs/agent-deny-rules.md's coverage table starts describing a CLI that no
	// longer exists. Nothing legitimately needs a rule for an unregistered command: the
	// patterns are written into a settings file for this binary, this version.
	inSurface := make(map[string]bool, len(surface))
	for _, argv := range surface {
		inSurface[argv] = true
	}
	for _, command := range agentdeny.GatedCommands {
		if !inSurface[command.Argv] {
			t.Errorf("GatedCommands entry %q names argv `%s`, which no gated command in internal/cli produces:\n"+
				"  the gated surface is %v.\n"+
				"  Either the command was renamed or removed (drop the entry and run `make generate`), "+
				"or its gate no longer refuses with a `*_not_enabled` code (fix the gate).",
				command.ID, command.Argv, surface)
		}
	}
}

func argvIsGated(argv string) bool {
	for _, command := range agentdeny.GatedCommands {
		if command.Argv == argv {
			return true
		}
	}
	return false
}

// parseCLIPackage parses this package's own non-test sources. `go test` runs with the
// package directory as the working directory, so "." is internal/cli. Test files are
// excluded because this file would otherwise be part of its own input — every argv string
// in the messages above would read as a call and a fixture command as a real one.
//
// Reading the directory rather than the build's file list is deliberate twice over. A
// build-tagged file counts on every platform, which is what a deny-rule policy wants: a
// rule must cover a command that exists on Linux even when the check runs on macOS. And it
// means `go test -overlay` cannot mutate this check — an overlay feeds the compiler, not
// os.ReadDir — so the mutation proving a new gated command fails the gate (TASK-337) has to
// write the fixture command to disk and delete it again.
func parseCLIPackage(t *testing.T) []*ast.File {
	t.Helper()

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read internal/cli: %v", err)
	}

	fset := token.NewFileSet()
	var files []*ast.File
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || filepath.Ext(name) != ".go" || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		if file.Name.Name != "cli" {
			continue
		}
		files = append(files, file)
	}
	if len(files) == 0 {
		t.Fatal("no non-test source parsed from internal/cli; every check below would pass vacuously")
	}
	return files
}

// gateCodeConstants returns the names of the string constants whose value ends in
// `_not_enabled`.
//
// The value, not the name, is the test: the constant identifiers are an internal spelling
// (`codeSealNotEnabled`), while the wire code is the frozen part — TASK-281 §3-7 froze
// `seal_not_enabled` and `show_not_enabled`, and a later gate that refuses with anything
// but a `*_not_enabled` code is not the shape this check claims to cover.
func gateCodeConstants(files []*ast.File) map[string]bool {
	codes := map[string]bool{}
	for _, file := range files {
		for _, decl := range file.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.CONST {
				continue
			}
			for _, spec := range gen.Specs {
				value, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for i, name := range value.Names {
					if i >= len(value.Values) {
						continue
					}
					lit, ok := value.Values[i].(*ast.BasicLit)
					if !ok || lit.Kind != token.STRING {
						continue
					}
					text, err := strconv.Unquote(lit.Value)
					if err != nil {
						continue
					}
					if strings.HasSuffix(text, "_not_enabled") {
						codes[name.Name] = true
					}
				}
			}
		}
	}
	return codes
}

// gatedFunctions returns every package-level function that reaches a gate refusal, directly
// or through another package-level function.
//
// Transitive because the gate is never in the command literal itself: seal's RunE calls
// runEnvSeal, which calls checkSealEnabled, which is the one that refuses. Fixpoint rather
// than a fixed depth so an extracted helper does not silently drop a command off the
// surface — an extraction is exactly the refactor that would otherwise turn this check off
// without anyone touching it.
func gatedFunctions(files []*ast.File, gateCodes map[string]bool) map[string]bool {
	bodies := map[string]*ast.FuncDecl{}
	for _, file := range files {
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil || fn.Body == nil {
				continue
			}
			bodies[fn.Name.Name] = fn
		}
	}

	gated := map[string]bool{}
	for name, fn := range bodies {
		if refusesWithGateCode(fn.Body, gateCodes) {
			gated[name] = true
		}
	}

	for changed := true; changed; {
		changed = false
		for name, fn := range bodies {
			if gated[name] {
				continue
			}
			for callee := range calledIdents(fn.Body) {
				if gated[callee] {
					gated[name] = true
					changed = true
					break
				}
			}
		}
	}
	return gated
}

// refusesWithGateCode reports whether node contains a `bridgeErr(<gate code>, ...)` call —
// the literal act of refusing because a command is disabled.
func refusesWithGateCode(node ast.Node, gateCodes map[string]bool) bool {
	found := false
	ast.Inspect(node, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || len(call.Args) == 0 {
			return true
		}
		fn, ok := call.Fun.(*ast.Ident)
		if !ok || fn.Name != "bridgeErr" {
			return true
		}
		if code, ok := call.Args[0].(*ast.Ident); ok && gateCodes[code.Name] {
			found = true
			return false
		}
		return true
	})
	return found
}

// calledIdents returns the names of every plain-identifier call in node. Selector calls
// (x.F()) are deliberately ignored: a gate lives in this package, and a method or
// cross-package call cannot be one.
func calledIdents(node ast.Node) map[string]bool {
	names := map[string]bool{}
	ast.Inspect(node, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok {
			if fn, ok := call.Fun.(*ast.Ident); ok {
				names[fn.Name] = true
			}
		}
		return true
	})
	return names
}

// commandDecl is one `var x = &cobra.Command{...}` declaration.
type commandDecl struct {
	use  string // first word of the Use field: the command's own name
	lit  *ast.CompositeLit
	name string // the Go variable name, for error messages
}

// gatedCommandArgv returns the sorted argv of every cobra command whose literal reaches a
// gated function — "dva config env seal" and so on, spelled exactly as a user types it and
// exactly as GatedCommands.Argv must spell it.
//
// The parent chain comes from the same `parent.AddCommand(child)` calls that build the real
// tree at init time, so a command moved to a different parent moves here too. Commands that
// never reach rootCmd are skipped rather than failed: an unattached command literal is
// unreachable from the CLI, so it is not part of any surface.
func gatedCommandArgv(t *testing.T, files []*ast.File, gatedFuncs map[string]bool) []string {
	t.Helper()

	commands := map[string]commandDecl{}
	for _, file := range files {
		for _, decl := range file.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.VAR {
				continue
			}
			for _, spec := range gen.Specs {
				value, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for i, name := range value.Names {
					if i >= len(value.Values) {
						continue
					}
					lit, ok := cobraCommandLit(value.Values[i])
					if !ok {
						continue
					}
					commands[name.Name] = commandDecl{use: useName(lit), lit: lit, name: name.Name}
				}
			}
		}
	}
	if _, ok := commands["rootCmd"]; !ok {
		t.Fatal("rootCmd literal not found; every argv below would be rooted at nothing")
	}

	parent := map[string]string{}
	for _, file := range files {
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || sel.Sel.Name != "AddCommand" {
				return true
			}
			owner, ok := sel.X.(*ast.Ident)
			if !ok {
				return true
			}
			for _, arg := range call.Args {
				if child, ok := arg.(*ast.Ident); ok {
					parent[child.Name] = owner.Name
				}
			}
			return true
		})
	}

	var surface []string
	for name, command := range commands {
		if !reachesGate(command.lit, gatedFuncs) {
			continue
		}
		argv, ok := argvOf(name, commands, parent)
		if !ok {
			continue
		}
		surface = append(surface, argv)
	}
	sort.Strings(surface)
	return surface
}

func cobraCommandLit(expr ast.Expr) (*ast.CompositeLit, bool) {
	unary, ok := expr.(*ast.UnaryExpr)
	if !ok || unary.Op != token.AND {
		return nil, false
	}
	lit, ok := unary.X.(*ast.CompositeLit)
	if !ok {
		return nil, false
	}
	sel, ok := lit.Type.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Command" {
		return nil, false
	}
	pkg, ok := sel.X.(*ast.Ident)
	if !ok || pkg.Name != "cobra" {
		return nil, false
	}
	return lit, true
}

// useName is the first word of the Use field — cobra's own rule, since Use carries the
// argument sketch too ("show [target]").
func useName(lit *ast.CompositeLit) string {
	for _, elt := range lit.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		key, ok := kv.Key.(*ast.Ident)
		if !ok || key.Name != "Use" {
			continue
		}
		str, ok := kv.Value.(*ast.BasicLit)
		if !ok || str.Kind != token.STRING {
			continue
		}
		text, err := strconv.Unquote(str.Value)
		if err != nil {
			continue
		}
		fields := strings.Fields(text)
		if len(fields) == 0 {
			return ""
		}
		return fields[0]
	}
	return ""
}

func reachesGate(lit *ast.CompositeLit, gatedFuncs map[string]bool) bool {
	for callee := range calledIdents(lit) {
		if gatedFuncs[callee] {
			return true
		}
	}
	return false
}

// argvOf walks from a command variable up to rootCmd, returning the invocation a user
// types. Reports false for a command whose chain does not reach rootCmd (unattached, or
// attached in a loop the AST cannot follow), which is not part of the reachable CLI.
func argvOf(name string, commands map[string]commandDecl, parent map[string]string) (string, bool) {
	var words []string
	for seen := map[string]bool{}; ; {
		if seen[name] {
			return "", false
		}
		seen[name] = true

		command, ok := commands[name]
		if !ok || command.use == "" {
			return "", false
		}
		words = append([]string{command.use}, words...)
		if name == "rootCmd" {
			return strings.Join(words, " "), true
		}
		next, ok := parent[name]
		if !ok {
			return "", false
		}
		name = next
	}
}
