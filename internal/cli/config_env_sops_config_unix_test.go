//go:build !windows

package cli

import (
	"reflect"
	"testing"
)

// seal's row-23 check and the encrypt it guards use one resolution. Before,
// row 23 walked for a .sops.yaml on its own, so a project that names its rules
// file with sops.config (and has no .sops.yaml anywhere) was refused even
// though the declared file exists.
func TestConfigEnvSealUsesDeclaredRulesFile(t *testing.T) {
	yaml := bridgeEnabledYAML("secrets.env.enc", true, false) + "sops:\n  config: ops/sops.yaml\n"
	f := newBridgeFixture(t, yaml, map[string]string{".env": bridgePayload(), "ops/sops.yaml": sopsCreationRuleYAML})
	f.install(false)

	var err error
	captureStreams(t, func() { err = runEnvSeal("", true) })
	if err != nil {
		t.Fatalf("seal: %v", err)
	}
	if got, want := f.sops.configsUsed(), []string{f.path("ops/sops.yaml")}; !reflect.DeepEqual(got, want) {
		t.Errorf("--config per call = %v, want %v", got, want)
	}
}

// lookup: root ignores a nested rules file, so a source whose only rules sit
// beside it has none under root mode and seal refuses before sops runs.
func TestConfigEnvSealRootLookupIgnoresNestedRules(t *testing.T) {
	yaml := bridgeEnabledYAML("svc/secrets.env.enc", true, false) + "sops:\n  lookup: root\n"
	f := newBridgeFixture(t, yaml, map[string]string{".env": bridgePayload(), "svc/.sops.yaml": sopsCreationRuleYAML})
	f.install(false)

	var err error
	captureStreams(t, func() { err = runEnvSeal("", true) })
	requireCode(t, err, codeSopsCreationRuleMissing)
	if calls := f.sops.callsMade(); len(calls) != 0 {
		t.Errorf("sops calls = %v, want none", calls)
	}
}
