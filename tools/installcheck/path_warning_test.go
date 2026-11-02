package installcheck

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestInstallerOnlyTargetWarnsAboutPathResolution(t *testing.T) {
	repo := repositoryRoot(t)
	systemPath := os.Getenv("PATH")
	install := func(t *testing.T, pathFor func(fixture, localDir, goDir string) string) string {
		t.Helper()
		fixture := t.TempDir()
		source := fixtureDVA(t, filepath.Join(fixture, "source"))
		localDir := filepath.Join(fixture, "local", "bin")
		goDir := filepath.Join(fixture, "go", "bin")
		output, err := runInstallerResult(repo, source, []string{"PATH=" + pathFor(fixture, localDir, goDir)},
			"LOCAL_BIN_DIR="+localDir, "GO_BIN_DIR="+goDir)
		if err != nil {
			t.Fatalf("a PATH warning must not fail the install: %v\n%s", err, output)
		}
		return output
	}

	t.Run("earlier binary shadows the install", func(t *testing.T) {
		output := install(t, func(fixture, localDir, goDir string) string {
			shadowDir := filepath.Join(fixture, "shadow")
			writeExecutable(t, filepath.Join(shadowDir, "dva"), "#!/bin/sh\necho stale\n")
			if err := os.MkdirAll(localDir, 0o755); err != nil {
				t.Fatal(err)
			}
			return shadowDir + ":" + localDir + ":" + systemPath
		})
		if !strings.Contains(output, "WARNING") || !strings.Contains(output, "stale") || !strings.Contains(output, "shadows it") {
			t.Fatalf("missing shadow warning:\n%s", output)
		}
	})

	t.Run("later binary stays quiet", func(t *testing.T) {
		output := install(t, func(fixture, localDir, goDir string) string {
			laterDir := filepath.Join(fixture, "later")
			writeExecutable(t, filepath.Join(laterDir, "dva"), "#!/bin/sh\necho stale\n")
			if err := os.MkdirAll(localDir, 0o755); err != nil {
				t.Fatal(err)
			}
			return localDir + ":" + laterDir + ":" + systemPath
		})
		if strings.Contains(output, "WARNING") {
			t.Fatalf("unexpected warning:\n%s", output)
		}
	})

	t.Run("identical copy and dva-less directory ahead stay quiet", func(t *testing.T) {
		output := install(t, func(fixture, localDir, goDir string) string {
			sameDir := filepath.Join(fixture, "same")
			if err := os.Rename(fixtureDVA(t, sameDir), filepath.Join(sameDir, "dva")); err != nil {
				t.Fatal(err)
			}
			emptyDir := filepath.Join(fixture, "empty")
			for _, dir := range []string{localDir, filepath.Join(emptyDir, "dva")} {
				if err := os.MkdirAll(dir, 0o755); err != nil {
					t.Fatal(err)
				}
			}
			return sameDir + ":" + emptyDir + ":" + localDir + ":" + systemPath
		})
		if strings.Contains(output, "WARNING") {
			t.Fatalf("unexpected warning:\n%s", output)
		}
	})

	t.Run("Go destination and symlinked destination count as reachable", func(t *testing.T) {
		for _, useLink := range []bool{false, true} {
			output := install(t, func(fixture, localDir, goDir string) string {
				if err := os.MkdirAll(goDir, 0o755); err != nil {
					t.Fatal(err)
				}
				entry := goDir
				if useLink {
					entry = filepath.Join(fixture, "link")
					if err := os.Symlink(goDir, entry); err != nil {
						t.Fatal(err)
					}
				}
				return entry + ":" + systemPath
			})
			if strings.Contains(output, "WARNING") {
				t.Fatalf("unexpected warning (link=%v):\n%s", useLink, output)
			}
		}
	})

	t.Run("unreadable shadow warns without failing", func(t *testing.T) {
		output := install(t, func(fixture, localDir, goDir string) string {
			shadowDir := filepath.Join(fixture, "shadow")
			writeExecutableMode(t, filepath.Join(shadowDir, "dva"), "#!/bin/sh\n", 0o111)
			if err := os.MkdirAll(localDir, 0o755); err != nil {
				t.Fatal(err)
			}
			return shadowDir + ":" + localDir + ":" + systemPath
		})
		if !strings.Contains(output, "unknown version") {
			t.Fatalf("missing unknown-version warning:\n%s", output)
		}
	})

	t.Run("stdin-reading shadow cannot hang and warning goes to stderr", func(t *testing.T) {
		fixture := t.TempDir()
		source := fixtureDVA(t, filepath.Join(fixture, "source"))
		shadowDir := filepath.Join(fixture, "shadow")
		writeExecutable(t, filepath.Join(shadowDir, "dva"), "#!/bin/sh\ncat >/dev/null\n")
		localDir := filepath.Join(fixture, "local", "bin")
		if err := os.MkdirAll(localDir, 0o755); err != nil {
			t.Fatal(err)
		}
		held, writer, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		defer held.Close()
		defer writer.Close()
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, "make", "-C", repo, "--no-print-directory", "install-binary", "INSTALL_SOURCE="+source,
			"LOCAL_BIN_DIR="+localDir, "GO_BIN_DIR="+filepath.Join(fixture, "go", "bin"))
		cmd.Env = append(os.Environ(), "PATH="+shadowDir+":"+localDir+":"+systemPath)
		cmd.Stdin = held
		cmd.WaitDelay = 2 * time.Second
		var stdout, stderr strings.Builder
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		if err := cmd.Run(); err != nil {
			t.Fatalf("install failed or hung: %v\n%s", err, stderr.String())
		}
		if !strings.Contains(stderr.String(), "WARNING") || strings.Contains(stdout.String(), "WARNING") {
			t.Fatalf("warning must reach stderr only\nstdout: %s\nstderr: %s", stdout.String(), stderr.String())
		}
	})

	t.Run("no destination on PATH", func(t *testing.T) {
		output := install(t, func(fixture, localDir, goDir string) string { return systemPath })
		if !strings.Contains(output, "no install destination is on PATH") {
			t.Fatalf("missing unreachable warning:\n%s", output)
		}
	})
}

func TestInstallerOnlyTargetWarningNeverFailsAfterInstall(t *testing.T) {
	repo := repositoryRoot(t)
	systemPath := os.Getenv("PATH")
	prepare := func(t *testing.T, shadowBody string) (string, *exec.Cmd, context.CancelFunc) {
		t.Helper()
		fixture := t.TempDir()
		source := fixtureDVA(t, filepath.Join(fixture, "source"))
		shadowDir := filepath.Join(fixture, "shadow")
		writeExecutable(t, filepath.Join(shadowDir, "dva"), strings.ReplaceAll(shadowBody, "FIXTURE", fixture))
		localDir := filepath.Join(fixture, "local", "bin")
		writeExistingBinary(t, localDir)
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		cmd := exec.CommandContext(ctx, "make", "-C", repo, "--no-print-directory", "install-binary", "INSTALL_SOURCE="+source,
			"LOCAL_BIN_DIR="+localDir, "GO_BIN_DIR="+filepath.Join(fixture, "go", "bin"))
		cmd.Env = append(os.Environ(), "PATH="+shadowDir+":"+localDir+":"+systemPath)
		cmd.WaitDelay = 2 * time.Second
		return fixture, cmd, cancel
	}

	t.Run("looping writer ends with its pipe", func(t *testing.T) {
		_, cmd, cancel := prepare(t, "#!/bin/sh\nwhile :; do echo spam; done\n")
		defer cancel()
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("install failed or hung: %v\n%s", err, output)
		}
	})

	t.Run("stderr closed during the warning", func(t *testing.T) {
		fixture, cmd, cancel := prepare(t, "#!/bin/sh\n: > FIXTURE/mark\nwhile [ ! -e FIXTURE/gate ]; do sleep 0.05; done\necho stale\n")
		defer cancel()
		reader, writer, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		defer writer.Close()
		cmd.Stderr = writer
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		go func() { _, _ = io.Copy(io.Discard, reader) }()
		mark := filepath.Join(fixture, "mark")
		deadline := time.Now().Add(30 * time.Second)
		for {
			if _, err := os.Stat(mark); err == nil {
				break
			}
			if time.Now().After(deadline) {
				t.Fatal("shadow never ran")
			}
			time.Sleep(20 * time.Millisecond)
		}
		reader.Close()
		if err := os.WriteFile(filepath.Join(fixture, "gate"), nil, 0o644); err != nil {
			t.Fatal(err)
		}
		if err := cmd.Wait(); err != nil {
			t.Fatalf("closed stderr failed the install: %v", err)
		}
		leftovers, err := filepath.Glob(filepath.Join(fixture, "local", "bin", ".dva-install*"))
		if err != nil || len(leftovers) != 0 {
			t.Fatalf("leftovers %v (%v)", leftovers, err)
		}
	})
}
