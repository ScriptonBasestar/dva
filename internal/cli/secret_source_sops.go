package cli

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

// editSecretSOPS leaves editor interaction to SOPS. Child diagnostics are never
// returned as DVA errors, because they may include decrypted content.
func editSecretSOPS(ctx context.Context, root, source, rules, format string) error {
	cmd := exec.CommandContext(ctx, "sops", sopsArgs(rules, "edit", "--input-type", format, "--output-type", format, source)...)
	cmd.Dir = root
	cmd.Stdin, cmd.Stdout = os.Stdin, io.Discard
	// Interactive display goes to the controlling terminal, never to captured
	// DVA stdout. Scripted editors can still run without a terminal.
	if terminal, err := os.OpenFile("/dev/tty", os.O_WRONLY, 0); err == nil {
		defer func() { _ = terminal.Close() }()
		cmd.Stdout = terminal
	}
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("secret_edit_failed")
	}
	return nil
}

type sopsStageResult struct {
	stage string
	err   error
}

// resealResultError only uses diagnostics after both waits have completed.
// Never return the diagnostic itself: it may contain decrypted data.
func resealResultError(first, second sopsStageResult, diagnostic string) error {
	if ((first.stage == "encrypt" && first.err != nil) || (second.stage == "encrypt" && second.err != nil)) &&
		strings.Contains(strings.ToLower(diagnostic), "creation rule") {
		return fmt.Errorf("secret_creation_rule_missing")
	}
	failed := first
	if failed.err == nil {
		failed = second
	}
	if failed.err != nil {
		return fmt.Errorf("secret_%s_failed", failed.stage)
	}
	return nil
}

// resealSecretSOPS connects two children with a kernel pipe. No plaintext file
// or plaintext buffer is created by DVA; only ciphertext reaches the writer.
// Both children must succeed before the caller may commit that ciphertext.
func resealSecretSOPS(ctx context.Context, root, source, rules, format string, out *os.File) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	r, w, err := os.Pipe()
	if err != nil {
		return err
	}
	defer func() { _ = r.Close(); _ = w.Close() }()
	decrypt := exec.CommandContext(ctx, "sops", sopsArgs(rules, "decrypt", "--input-type", format, "--output-type", format, source)...)
	encrypt := exec.CommandContext(ctx, "sops", sopsArgs(rules, "encrypt", "--input-type", format, "--output-type", format, "--filename-override", source, "/dev/stdin")...)
	decrypt.Dir, encrypt.Dir = root, root
	decrypt.Stdout, decrypt.Stderr = w, io.Discard
	encrypt.Stdin, encrypt.Stdout = r, out
	var diagnostic bytes.Buffer
	encrypt.Stderr = &limitedWriter{w: &diagnostic, remaining: sopsStderrLimit}
	if err := encrypt.Start(); err != nil {
		return fmt.Errorf("secret_encrypt_failed")
	}
	if err := decrypt.Start(); err != nil {
		cancel()
		_ = r.Close()
		_ = w.Close()
		_ = encrypt.Wait()
		return fmt.Errorf("secret_decrypt_failed")
	}
	_ = r.Close()
	_ = w.Close()
	// Cancel a peer promptly on the first failure, then collect both outcomes.
	finished := make(chan sopsStageResult, 2)
	go func() { finished <- sopsStageResult{"decrypt", decrypt.Wait()} }()
	go func() { finished <- sopsStageResult{"encrypt", encrypt.Wait()} }()
	first := <-finished
	if first.err != nil {
		cancel()
	}
	second := <-finished
	return resealResultError(first, second, diagnostic.String())
}
