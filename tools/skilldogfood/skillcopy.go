package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type limitedBuffer struct {
	contents []byte
	limit    int
}

func (buffer *limitedBuffer) Write(value []byte) (int, error) {
	remaining := buffer.limit - len(buffer.contents)
	if remaining > 0 {
		if len(value) < remaining {
			remaining = len(value)
		}
		buffer.contents = append(buffer.contents, value[:remaining]...)
	}
	return len(value), nil
}

func (buffer *limitedBuffer) String() string { return string(buffer.contents) }

func immutableExecutableCopy(source, expected string) (path, digest string, cleanup func() error, err error) {
	directory, err := os.MkdirTemp("", "dva-skill-dogfood-bin-")
	if err != nil {
		return "", "", nil, fmt.Errorf("create immutable executable directory: %w", err)
	}
	cleanup = func() error { return removeAll("clean immutable executable directory "+directory, directory) }
	input, err := os.Open(source)
	if err != nil {
		return "", "", nil, errors.Join(fmt.Errorf("open DVA_BIN: %w", err), cleanup())
	}
	destination := filepath.Join(directory, "dva")
	output, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o700)
	if err != nil {
		return "", "", nil, errors.Join(fmt.Errorf("create immutable DVA copy: %w", err), input.Close(), cleanup())
	}
	hash := sha256.New()
	_, copyErr := io.Copy(io.MultiWriter(output, hash), input)
	closeErr := errors.Join(input.Close(), output.Close())
	if copyErr != nil || closeErr != nil {
		return "", "", nil, errors.Join(copyErr, closeErr, cleanup())
	}
	digest = hex.EncodeToString(hash.Sum(nil))
	if digest != strings.ToLower(expected) {
		return "", "", nil, errors.Join(fmt.Errorf("DVA_BIN SHA-256 %s does not match expected %s", digest, expected), cleanup())
	}
	return destination, digest, cleanup, nil
}

func removeAll(context, path string) error {
	if err := os.RemoveAll(path); err != nil {
		return fmt.Errorf("%s: %w", context, err)
	}
	return nil
}

func executableFile(path string) (string, error) {
	if path == "" {
		return "", errors.New("must be set to an absolute executable path")
	}
	if !filepath.IsAbs(path) {
		return "", fmt.Errorf("must be absolute, got %q", path)
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0o111 == 0 {
		return "", fmt.Errorf("is not an executable regular file: %s", path)
	}
	return filepath.EvalSymlinks(path)
}

func gitRoot(path string) (string, error) {
	if path == "" {
		return "", errors.New("must be set to an absolute Git repository root path")
	}
	if !filepath.IsAbs(path) {
		return "", fmt.Errorf("must be absolute, got %q", path)
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("is not a directory: %s", resolved)
	}
	rootOutput, err := commandOutput(nil, "git", "-C", resolved, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", fmt.Errorf("is not a Git repository root: %w", err)
	}
	root, err := filepath.EvalSymlinks(strings.TrimSpace(rootOutput))
	if err != nil {
		return "", err
	}
	if root != resolved {
		return "", fmt.Errorf("must name the repository root, not %s", resolved)
	}
	return root, nil
}
