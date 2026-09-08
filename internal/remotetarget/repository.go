// Package remotetarget binds remote writes to the owning checkout's GitHub origin.
package remotetarget

import (
	"bytes"
	"context"
	"errors"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

var repositoryPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]*/[A-Za-z0-9][A-Za-z0-9_.-]*$`)

// ValidRepository accepts only an explicit GitHub owner/repository identity.
func ValidRepository(repository string) bool {
	return len(repository) <= 200 && repositoryPattern.MatchString(repository)
}

// ParseOrigin accepts GitHub's canonical HTTPS and SSH remote forms. Aliases,
// alternate hosts, credentials in HTTPS URLs and local paths are not destinations.
func ParseOrigin(raw string) (string, error) {
	var repository string
	if value, ok := strings.CutPrefix(raw, "git@github.com:"); ok {
		repository = value
	} else {
		u, err := url.Parse(raw)
		if err != nil || u.Host != "github.com" || u.RawQuery != "" || u.Fragment != "" || u.RawPath != "" {
			return "", errors.New("remote_target_invalid_origin")
		}
		switch u.Scheme {
		case "https":
			if u.User != nil {
				return "", errors.New("remote_target_invalid_origin")
			}
		case "ssh":
			if u.User == nil || u.User.String() != "git" {
				return "", errors.New("remote_target_invalid_origin")
			}
		default:
			return "", errors.New("remote_target_invalid_origin")
		}
		repository = strings.TrimPrefix(u.Path, "/")
	}
	repository = strings.TrimSuffix(repository, ".git")
	if !ValidRepository(repository) {
		return "", errors.New("remote_target_invalid_origin")
	}
	return repository, nil
}

// Validate is read-only. A target declaration cannot redirect a secret or job
// into a different repository. A checkout's origin is the v1 trust anchor;
// changing Git config remains an explicit change to that trust anchor.
func Validate(ctx context.Context, root, repository string) error {
	if root == "" || !ValidRepository(repository) {
		return errors.New("remote_target_invalid_repository")
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "-C", root, "config", "--local", "--get", "remote.origin.url")
	cmd.WaitDelay = time.Second
	for _, entry := range os.Environ() {
		// A parent shell's GIT_DIR/WORK_TREE/config injection must not select a
		// different checkout while this command claims to validate root.
		if !strings.HasPrefix(entry, "GIT_") {
			cmd.Env = append(cmd.Env, entry)
		}
	}
	cmd.Env = append(cmd.Env, "GIT_TERMINAL_PROMPT=0")
	// Never relay git stderr, which may include credentials from a malformed URL.
	var stdout originOutput
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil || stdout.exceeded {
		return errors.New("remote_target_origin_unavailable")
	}
	origin, err := ParseOrigin(strings.TrimSpace(stdout.String()))
	if err != nil {
		return err
	}
	if !strings.EqualFold(origin, repository) {
		return errors.New("remote_target_repository_mismatch")
	}
	return nil
}

type originOutput struct {
	bytes.Buffer
	exceeded bool
}

func (b *originOutput) Write(p []byte) (int, error) {
	if b.Len()+len(p) > 4096 {
		b.exceeded = true
		return 0, errors.New("origin output too large")
	}
	return b.Buffer.Write(p)
}
