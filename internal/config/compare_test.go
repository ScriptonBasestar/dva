package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// compareBase declares a plan split by the compare tests: web runs per variant, api and db are
// shared. The web entry declares no compose project_name, so per-variant isolation holds.
const compareBase = `
stack:
  api:
    default_runner: native
    runners:
      native:
        run: go run ./cmd/api
  db:
    default_runner: compose
    runners:
      compose:
        files: [db.yml]
        project_name: shared-db
  web:
    default_runner: compose
    runners:
      compose:
        files: [web.yml]
plans:
  web-dev:
    entries:
      - name: api
      - name: db
      - name: web
  web-dev-alias:
    alias: web-dev
  web-dev-extra:
    extends: web-dev
    entries:
      - name: api
        order: 5
`

func loadCompare(t *testing.T, compare string) (*Config, error) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte(compareBase+compare), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	return cfg, cfg.Validate()
}

func TestCompareParsesDeclaration(t *testing.T) {
	cfg, err := loadCompare(t, `compare:
  web-layout:
    plan: web-dev
    shared: [api, db]
    per_variant: [web]
    ports:
      WEB_PORT: [3001, 3002, 3003]
    variants: { a: main, b: feat/layout-a }
`)
	if err != nil {
		t.Fatalf("valid declaration rejected: %v", err)
	}
	cmp := cfg.Compare["web-layout"]
	if cmp == nil {
		t.Fatal("compare.web-layout not parsed")
	}
	if cmp.Plan != "web-dev" || strings.Join(cmp.Shared, ",") != "api,db" || strings.Join(cmp.PerVariant, ",") != "web" {
		t.Errorf("parsed = %+v", cmp)
	}
	if got := cmp.Ports["WEB_PORT"]; len(got) != 3 || got[0] != 3001 {
		t.Errorf("ports = %v", cmp.Ports)
	}
	if cmp.Variants["b"] != "feat/layout-a" {
		t.Errorf("variants = %v", cmp.Variants)
	}
}

// TestCompareFollowsAliasAndExtends: the split is checked against the entries the plan
// actually runs, so an alias resolves to its target and extends unions the parent's entries.
func TestCompareFollowsAliasAndExtends(t *testing.T) {
	for _, plan := range []string{"web-dev-alias", "web-dev-extra"} {
		t.Run(plan, func(t *testing.T) {
			_, err := loadCompare(t, "compare:\n  c:\n    plan: "+plan+"\n    shared: [api, db]\n    per_variant: [web]\n")
			if err != nil {
				t.Fatalf("plan %s rejected: %v", plan, err)
			}
		})
	}
}

func TestCompareRejectsInvalidDeclarations(t *testing.T) {
	cases := []struct {
		name    string
		compare string
		want    string
	}{
		{"unknown plan", "    plan: ghost\n    shared: [api]\n",
			`compare.c.plan: plan "ghost" not found`},
		{"overlap", "    plan: web-dev\n    shared: [api, db, web]\n    per_variant: [web]\n",
			`compare.c: entry "web" is in both shared and per_variant`},
		{"missing entry", "    plan: web-dev\n    shared: [api]\n    per_variant: [web]\n",
			`compare.c: plan "web-dev" entries db are in neither shared nor per_variant`},
		{"foreign entry", "    plan: web-dev\n    shared: [api, db, cache]\n    per_variant: [web]\n",
			`compare.c.shared: entry "cache" is not an entry of plan "web-dev"`},
		{"fixed project_name", "    plan: web-dev\n    shared: [api]\n    per_variant: [db, web]\n",
			`compare.c.per_variant: stack entry "db" sets project_name "shared-db"`},
		{"ports shorter than default variants", "    plan: web-dev\n    shared: [api, db]\n    per_variant: [web]\n    ports: { WEB_PORT: [3001] }\n    variants: { a: main, b: dev }\n",
			`compare.c.variants: 2 variants need 2 ports per list, but WEB_PORT has 1`},
		{"duplicate port", "    plan: web-dev\n    shared: [api, db]\n    per_variant: [web]\n    ports: { API_PORT: [3001], WEB_PORT: [3001] }\n",
			`compare.c.ports.WEB_PORT: port 3001 is already assigned under ports.API_PORT`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := loadCompare(t, "compare:\n  c:\n"+tc.compare)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v\nwant substring %q", err, tc.want)
			}
		})
	}
}

func TestCompareRejectsCompositionPlan(t *testing.T) {
	c := &Config{
		Plans: map[string]*PlanConfig{
			"composed": {Composes: []CompositionEntry{{}}},
		},
		Compare: map[string]*CompareConfig{"c": {Plan: "composed"}},
	}
	errs := c.validateCompare()
	if len(errs) != 1 || !strings.Contains(errs[0].Error(), `"composed" is a composition plan`) {
		t.Fatalf("errs = %v", errs)
	}
}

func TestCheckPortCapacity(t *testing.T) {
	cmp := &CompareConfig{Ports: map[string][]int{"WEB_PORT": {1, 2, 3}, "API_PORT": {4, 5}}}
	if err := cmp.CheckPortCapacity(2); err != nil {
		t.Errorf("2 variants: %v", err)
	}
	err := cmp.CheckPortCapacity(3)
	if err == nil || !strings.Contains(err.Error(), "API_PORT has 2") || strings.Contains(err.Error(), "WEB_PORT") {
		t.Errorf("3 variants: %v", err)
	}
	if err := (&CompareConfig{}).CheckPortCapacity(5); err != nil {
		t.Errorf("no ports declared must not limit variants: %v", err)
	}
}

// TestCompareMergeReplacesWholeEntry: a module redefining a comparison replaces it whole, so
// the result is always a declaration some file actually wrote.
func TestCompareMergeReplacesWholeEntry(t *testing.T) {
	base := &Config{Compare: map[string]*CompareConfig{
		"c":    {Plan: "web-dev", Shared: []string{"api", "db"}, PerVariant: []string{"web"}},
		"keep": {Plan: "web-dev"},
	}}
	base.mergeFrom(&Config{Compare: map[string]*CompareConfig{"c": {Plan: "other"}}})
	if got := base.Compare["c"]; got.Plan != "other" || got.Shared != nil {
		t.Errorf("c = %+v, want whole replacement", got)
	}
	if base.Compare["keep"] == nil {
		t.Error("unrelated comparison dropped by merge")
	}
}
