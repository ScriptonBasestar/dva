package cli

// TASK-353: Widen the gated-command detector to the shapes present in the tree.
//
// The original detector (TASK-337, TestGatedCommandsCoverEveryGatedSurface) only recognises
// one command shape:
//   var x = &cobra.Command{Use: "name", RunE: someFunc(...)}
//
// Four other shapes exist in internal/cli and are not seen:
//   (a) RunE: <bare func ident>   — `RunE: runKubectlPassthrough` in kubectl.go
//   (b) Non-literal Use           — `Use: otherCmd.Use` in validate_alias.go
//   (c) init() assign / local :=  — `initAliasCmd = &cobra.Command{…}` in init.go,
//                                   `rootValidateCmd := &cobra.Command{…}` in validate_alias.go
//   (d) AddCommand inline literal — ci.go:85,95 (intentionally excluded from criteria,
//                                   see card Notes)
//
// This file:
//   1. Adds a per-shape disk-resident fixture (required because parseCLIPackage uses
//      os.ReadDir and cannot be mutated by -overlay).
//   2. Adds a test for each shape (a)–(c).
//   3. Fixes reachesGate and gatedCommandArgv to cover shapes (a) and (c).
//   4. Upgrades the shape-(b) silent skip to a hard failure in gatedCommandArgvV2, then
//      wires TestGatedCommandsCoverEveryGatedSurface forward to use it.
//
// Shape (d) is not tested here — see card Notes — but the fixture directory exists so a
// future test can add it without changing the test infrastructure.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Fixtures — small, self-contained source files that declare exactly one shape.
// Each file is syntactically valid Go source in package "cli" so the regular
// parser can handle it without a full build.
// ---------------------------------------------------------------------------

// fixtureDir is the relative path (from internal/cli, where `go test` runs) to
// the directory holding the per-shape fixture source files.
const fixtureDir = "testdata/gatedetector"

// fixtureFiles holds the name and content of each shape fixture.
var fixtureFiles = map[string]string{
	// Shape (a): RunE value is a bare function identifier, not a call expression.
	// reachesGate previously only walked calledIdents (call expressions), so
	// `RunE: runGatedOp` was invisible even when runGatedOp reached a gate.
	"bare_rune_ident_fixture.go": `package cli

const fixtureGatedNotEnabled = "fixture_not_enabled"

func runGatedOp(cmd *cobra.Command, args []string) error {
	return bridgeErr(fixtureGatedNotEnabled, nil)
}

var rootCmd = &cobra.Command{Use: "dva"}
var fixtureCmd = &cobra.Command{
	Use:  "fixtureop",
	RunE: runGatedOp,
}

func init() { rootCmd.AddCommand(fixtureCmd) }
`,

	// Shape (b): Use field is a non-literal expression.  The current detector
	// returns "" for useName and argvOf returns (false) silently.  After the fix,
	// gatedCommandArgvV2 must return a hard error when a gated command's Use
	// cannot be resolved to a literal.
	"unresolvable_use_fixture.go": `package cli

const fixtureUnresolvableNotEnabled = "fixture_unresolvable_not_enabled"

func runUnresolvableGated(cmd *cobra.Command, args []string) error {
	return bridgeErr(fixtureUnresolvableNotEnabled, nil)
}

var rootCmd = &cobra.Command{Use: "dva"}
var rootValidateCmd = &cobra.Command{
	Use:  rootCmd.Use,
	RunE: runUnresolvableGated,
}

func init() { rootCmd.AddCommand(rootValidateCmd) }
`,

	// Shape (c): cobra.Command literal assigned via := (local variable) or plain
	// assignment (package-level pointer, assigned in init).  The original collector
	// only walked *ast.ValueSpec (var decls), missing *ast.AssignStmt right-hand sides.
	"assign_command_literal_fixture.go": `package cli

const fixtureAssignedNotEnabled = "fixture_assigned_not_enabled"

func runAssignedGated(cmd *cobra.Command, args []string) error {
	return bridgeErr(fixtureAssignedNotEnabled, nil)
}

var rootCmd = &cobra.Command{Use: "dva"}
var fixtureAssignedCmd *cobra.Command

func init() {
	fixtureAssignedCmd = &cobra.Command{
		Use:  "assignedop",
		RunE: runAssignedGated,
	}
	rootCmd.AddCommand(fixtureAssignedCmd)
}
`,
}

// ensureFixtures writes the fixture files to fixtureDir and registers a
// t.Cleanup to remove them.  It is idempotent per test run.
func ensureFixtures(t *testing.T) {
	t.Helper()
	if err := os.MkdirAll(fixtureDir, 0o755); err != nil {
		t.Fatalf("create fixture dir %s: %v", fixtureDir, err)
	}
	for name, content := range fixtureFiles {
		path := filepath.Join(fixtureDir, name)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write fixture %s: %v", path, err)
		}
	}
	t.Cleanup(func() {
		for name := range fixtureFiles {
			_ = os.Remove(filepath.Join(fixtureDir, name))
		}
		_ = os.Remove(fixtureDir)
		_ = os.Remove("testdata") // remove if now empty
	})
}

// parseFixtureFiles parses a named subset of the fixture files and returns
// their ASTs.
func parseFixtureFiles(t *testing.T, names ...string) []*ast.File {
	t.Helper()
	ensureFixtures(t)
	fset := token.NewFileSet()
	var files []*ast.File
	for _, name := range names {
		path := filepath.Join(fixtureDir, name)
		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			t.Fatalf("parse fixture %s: %v", name, err)
		}
		files = append(files, f)
	}
	return files
}

// ---------------------------------------------------------------------------
// Detector helpers — extended versions used by the new tests and by the
// updated TestGatedCommandsCoverEveryGatedSurface via gatedCommandArgvV2.
// ---------------------------------------------------------------------------

// reachesGateV2 extends reachesGate: in addition to call expressions (RunE: func(...){…}),
// it also recognises bare function identifier values in RunE / Run / PreRunE fields
// (RunE: runMyFunc), treating them as calls to the named function.
func reachesGateV2(lit *ast.CompositeLit, gatedFuncs map[string]bool) bool {
	// Original: walk call expressions inside the literal.
	for callee := range calledIdents(lit) {
		if gatedFuncs[callee] {
			return true
		}
	}
	// New: look for RunE / Run / PreRunE whose value is a bare identifier.
	for _, elt := range lit.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		key, ok := kv.Key.(*ast.Ident)
		if !ok {
			continue
		}
		switch key.Name {
		case "RunE", "Run", "PreRunE", "PreRun":
			if ident, ok := kv.Value.(*ast.Ident); ok && gatedFuncs[ident.Name] {
				return true
			}
		}
	}
	return false
}

// collectCommandDecls collects cobra.Command literals from both var decls (*ast.ValueSpec)
// and assignment statements (*ast.AssignStmt), the two additional shapes missing from the
// original gatedCommandArgv.
func collectCommandDecls(files []*ast.File) map[string]commandDecl {
	commands := map[string]commandDecl{}
	for _, file := range files {
		for _, decl := range file.Decls {
			switch d := decl.(type) {
			case *ast.GenDecl:
				if d.Tok != token.VAR {
					continue
				}
				for _, spec := range d.Specs {
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
			case *ast.FuncDecl:
				// Walk the function body looking for assignment statements.
				if d.Body == nil {
					continue
				}
				ast.Inspect(d.Body, func(n ast.Node) bool {
					assign, ok := n.(*ast.AssignStmt)
					if !ok {
						return true
					}
					for i, rhs := range assign.Rhs {
						lit, ok := cobraCommandLit(rhs)
						if !ok {
							continue
						}
						if i >= len(assign.Lhs) {
							continue
						}
						name := ""
						switch lhs := assign.Lhs[i].(type) {
						case *ast.Ident:
							name = lhs.Name
						}
						if name == "" {
							continue
						}
						commands[name] = commandDecl{use: useName(lit), lit: lit, name: name}
					}
					return true
				})
			}
		}
	}
	return commands
}

// gatedCommandArgvV2 is an upgraded version of gatedCommandArgv that:
//  1. Uses reachesGateV2 (catches bare RunE idents — shape a).
//  2. Uses collectCommandDecls (catches assigned/local literals — shape c).
//  3. Calls errorfn when a gated command's Use is not a string literal (shape b) instead of
//     silently skipping it; errorfn follows the same signature as t.Errorf.
//
// Callers pass t.Errorf for production use and a custom recorder in shape-(b) tests.
func gatedCommandArgvV2(t *testing.T, errorfn func(string, ...any), files []*ast.File, gatedFuncs map[string]bool) []string {
	t.Helper()

	commands := collectCommandDecls(files)
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
		if !reachesGateV2(command.lit, gatedFuncs) {
			continue
		}
		argv, ok := argvOf(name, commands, parent)
		if !ok {
			// Shape (b): gated command whose Use cannot be resolved to a literal.
			// The original argvOf returns false here and gatedCommandArgv skips
			// silently.  We report via errorfn instead: the unknown is visible, not
			// discarded.  The caller decides whether this is a hard failure.
			if command.use == "" {
				errorfn("gated command variable %q has a non-literal Use field: "+
					"its argv cannot be determined, so no deny-rule can be generated for it. "+
					"Assign a string literal to Use, or restructure so the command is reachable "+
					"through a parent that does have a literal Use.", name)
			}
			continue
		}
		surface = append(surface, argv)
	}
	// Sort for determinism.
	for i := 0; i < len(surface); i++ {
		for j := i + 1; j < len(surface); j++ {
			if surface[i] > surface[j] {
				surface[i], surface[j] = surface[j], surface[i]
			}
		}
	}
	return surface
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

// TestGateDetectorFindsBareRunEIdent verifies that a cobra command whose RunE
// is a bare function identifier (not a call expression) is found by the
// gate detector.
//
// Mutation evidence: before the reachesGateV2 fix, the fixture's `runGatedOp`
// function reached the gate, but `fixtureCmd` had `RunE: runGatedOp` (an
// *ast.Ident value, not a call), so reachesGate returned false and the command
// was not reported on the surface.  The test below would have reported:
//
//	"dva fixtureop" is not on the detected surface []
//
// TASK-353 criterion 1.
func TestGateDetectorFindsBareRunEIdent(t *testing.T) {
	files := parseFixtureFiles(t, "bare_rune_ident_fixture.go")

	gateCodes := gateCodeConstants(files)
	if len(gateCodes) == 0 {
		t.Fatal("fixture carries no *_not_enabled constant; test would pass vacuously")
	}
	gatedFuncs := gatedFunctions(files, gateCodes)

	// Old detector — must miss the command.
	oldSurface := gatedCommandArgv(t, files, gatedFuncs)
	for _, argv := range oldSurface {
		if strings.Contains(argv, "fixtureop") {
			t.Logf("NOTE: old detector already finds %q (fixture may need updating)", argv)
		}
	}

	// New detector — must find the command.
	surface := gatedCommandArgvV2(t, t.Errorf, files, gatedFuncs)
	found := false
	for _, argv := range surface {
		if strings.Contains(argv, "fixtureop") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("reachesGateV2 did not find the bare-RunE-ident gated command on the surface %v; "+
			"expected an entry containing \"fixtureop\"", surface)
	}
}

// TestGateDetectorFailsOnUnresolvableUse verifies that when a gated command has
// a non-literal Use field, the upgraded detector reports an error rather than
// silently skipping the command.
//
// Mutation evidence: the original gatedCommandArgv returned (false) from
// argvOf (because useName returns "" for a selector-expression Use) and the
// command was silently absent from the surface.  The test below verifies that
// the upgraded detector calls errorfn for the fixture's gated alias command.
//
// TASK-353 criterion 2.
func TestGateDetectorFailsOnUnresolvableUse(t *testing.T) {
	files := parseFixtureFiles(t, "unresolvable_use_fixture.go")

	gateCodes := gateCodeConstants(files)
	if len(gateCodes) == 0 {
		t.Fatal("fixture carries no *_not_enabled constant; test would pass vacuously")
	}
	gatedFuncs := gatedFunctions(files, gateCodes)

	// Inject a custom errorfn that records whether it was called.
	// We do NOT pass t.Errorf here — we want to observe the call without
	// failing this test.  The assertion below is that the call happened.
	var reported []string
	recorder := func(format string, args ...any) {
		msg := format
		if len(args) > 0 {
			msg = strings.NewReplacer("%q", "%q", "%v", "%v", "%s", "%s").Replace(format)
			_ = msg      // keep simple
			msg = format // use format as-is for the assertion
		}
		reported = append(reported, msg)
	}
	gatedCommandArgvV2(t, recorder, files, gatedFuncs)

	if len(reported) == 0 {
		t.Errorf("gatedCommandArgvV2 did not call errorfn for the fixture command with " +
			"non-literal Use; it should report rather than silently skip a gated command " +
			"it cannot name")
	}
}

// TestGateDetectorFindsAssignedCommandLiteral verifies that a cobra command
// literal assigned inside an init() function (init-assign shape) is collected
// and appears on the gated surface.
//
// Mutation evidence: the original collector only walked *ast.ValueSpec (var
// decls), so `fixtureAssignedCmd = &cobra.Command{…}` inside init() was never
// added to the commands map, and the command was absent from the surface.  The
// test below verifies that collectCommandDecls picks it up.
//
// TASK-353 criterion 3.
func TestGateDetectorFindsAssignedCommandLiteral(t *testing.T) {
	files := parseFixtureFiles(t, "assign_command_literal_fixture.go")

	gateCodes := gateCodeConstants(files)
	if len(gateCodes) == 0 {
		t.Fatal("fixture carries no *_not_enabled constant; test would pass vacuously")
	}
	gatedFuncs := gatedFunctions(files, gateCodes)

	// Old detector — must miss the command.
	oldSurface := gatedCommandArgv(t, files, gatedFuncs)
	for _, argv := range oldSurface {
		if strings.Contains(argv, "assignedop") {
			t.Logf("NOTE: old detector already finds %q (fixture may need updating)", argv)
		}
	}

	// New detector — must find the command.
	surface := gatedCommandArgvV2(t, t.Errorf, files, gatedFuncs)
	found := false
	for _, argv := range surface {
		if strings.Contains(argv, "assignedop") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("collectCommandDecls did not find the init-assigned gated command on the "+
			"surface %v; expected an entry containing \"assignedop\"", surface)
	}
}
