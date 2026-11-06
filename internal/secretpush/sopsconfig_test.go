package secretpush

import (
	"reflect"
	"testing"
)

// --config is a global sops flag and sops rejects it after the subcommand, so
// a resolved rules file has to lead the argv.
func TestDecryptArgsPlacesConfigFirst(t *testing.T) {
	got := decryptArgs("/p/.sops.yaml", "s.enc", "yaml", "json")
	want := []string{"--config", "/p/.sops.yaml", "--decrypt", "--input-type", "yaml", "--output-type", "json", "s.enc"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	if got := decryptArgs("", "s.enc", "dotenv", "dotenv"); got[0] != "--decrypt" {
		t.Errorf("empty config must add no flag: %v", got)
	}
}
