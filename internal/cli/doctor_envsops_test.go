package cli

import (
	"strings"
	"testing"
)

func TestDoctorEnvSopsDeclaration(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		files   map[string]string
		wantRow bool
	}{
		{
			name:    "evidence without declaration reports a row",
			yaml:    "version: \"0.1.45\"\nenv_file: [.env]\n",
			files:   map[string]string{".sops.yaml": "creation_rules: []\n", "secrets.env.enc": "ENC"},
			wantRow: true,
		},
		{
			name:  "no evidence stays silent",
			yaml:  "version: \"0.1.45\"\nenv_file: [.env]\n",
			files: map[string]string{".env": "A=1\n"},
		},
		{
			name:  "a declared plaintext path is not a candidate",
			yaml:  "version: \"0.1.45\"\nenv_file: [.env.enc]\n",
			files: map[string]string{".env.enc": "A=1\n"},
		},
		{
			name:  "declared source stays silent",
			yaml:  simpleBridgeYAML(".env", "secrets.env.enc"),
			files: map[string]string{".sops.yaml": "creation_rules: []\n", "secrets.env.enc": "ENC"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newBridgeFixture(t, tt.yaml, tt.files)
			rows := checkEnvSopsDeclaration(f.cfg)
			if !tt.wantRow {
				if len(rows) != 0 {
					t.Fatalf("checkEnvSopsDeclaration() = %+v, want no rows", rows)
				}
				return
			}
			if len(rows) != 1 || rows[0].Passed {
				t.Fatalf("checkEnvSopsDeclaration() = %+v, want one failing row", rows)
			}
			if want := "{path: .env, sops_source: secrets.env.enc}"; !strings.Contains(rows[0].FixHint, want) {
				t.Fatalf("FixHint = %q, want it to contain %q", rows[0].FixHint, want)
			}
		})
	}
}

func TestNoEncryptedEntryMessageNamesEvidence(t *testing.T) {
	withEvidence := newBridgeFixture(t, "version: \"0.1.45\"\nenv_file: [.env.local]\n",
		map[string]string{"secrets.sops.env": "ENC"})
	_, err := selectEncryptedEntry(withEvidence.cfg, "")
	if got := errorCode(err); got != codeNoEncryptedEntry {
		t.Fatalf("code = %q, want %q", got, codeNoEncryptedEntry)
	}
	for _, want := range []string{"{path: .env.local, sops_source: secrets.sops.env}", "sops files found: secrets.sops.env"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q does not contain %q", err, want)
		}
	}

	bare := newBridgeFixture(t, "version: \"0.1.45\"\nenv_file: [.env]\n", nil)
	_, err = selectEncryptedEntry(bare.cfg, "")
	if msg := err.Error(); !strings.Contains(msg, "{path: .env, sops_source: .env.enc}") || strings.Contains(msg, "sops files found") {
		t.Fatalf("error without evidence = %q", msg)
	}
}
