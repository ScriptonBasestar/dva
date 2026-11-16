package cli

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestResealResultErrorPreservesCreationRule(t *testing.T) {
	failure := errors.New("private error sentinel")
	decrypt := sopsStageResult{stage: "decrypt", err: failure}
	encrypt := sopsStageResult{stage: "encrypt", err: failure}
	goodDecrypt := sopsStageResult{stage: "decrypt"}
	goodEncrypt := sopsStageResult{stage: "encrypt"}
	for _, tc := range []struct {
		name       string
		first      sopsStageResult
		second     sopsStageResult
		diagnostic string
		want       string
	}{
		{"decrypt first", decrypt, encrypt, "No matching CREATION RULE private sentinel", "secret_creation_rule_missing"},
		{"encrypt first", encrypt, decrypt, "no matching creation rule private sentinel", "secret_creation_rule_missing"},
		{"success", goodDecrypt, goodEncrypt, "", ""},
		{"decrypt failed", decrypt, goodEncrypt, "", "secret_decrypt_failed"},
		{"encrypt failed", goodDecrypt, encrypt, "private sentinel", "secret_encrypt_failed"},
		{"unrelated text in successful encrypt", decrypt, goodEncrypt, "creation rule private sentinel", "secret_decrypt_failed"},
		{"both generic decrypt first", decrypt, encrypt, "private sentinel", "secret_decrypt_failed"},
		{"both generic encrypt first", encrypt, decrypt, "private sentinel", "secret_encrypt_failed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := resealResultError(tc.first, tc.second, tc.diagnostic)
			if tc.want == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil || err.Error() != tc.want || strings.Contains(err.Error(), "sentinel") {
				t.Fatalf("got %v, want %q without diagnostics", err, tc.want)
			}
		})
	}
}

func TestResealCancelsFailedStagePeer(t *testing.T) {
	dir := t.TempDir()
	stub := `#!/bin/sh
case " $* " in
  *" decrypt "*) exec sleep 30 ;;
  *" encrypt "*) printf 'no matching creation rule private sentinel' >&2; exit 23 ;;
esac
`
	if err := os.WriteFile(filepath.Join(dir, "sops"), []byte(stub), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	out, err := os.CreateTemp(dir, "ciphertext-")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = out.Close() }()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = resealSecretSOPS(ctx, dir, "source.enc", "", "dotenv", out)
	if err == nil || err.Error() != "secret_creation_rule_missing" {
		t.Fatalf("expected classified error: %v", err)
	}
	if ctx.Err() != nil {
		t.Fatalf("peer was not canceled before caller deadline: %v", ctx.Err())
	}
}
