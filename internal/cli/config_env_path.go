package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/ScriptonBasestar/dva/internal/config"
)

// bridgeTargetPath resolves only the plaintext target. sops_source remains a
// literal config-relative declaration; it is not an env-file loading path.
func bridgeTargetPath(c *config.Config, declared string) (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("env bridge: cannot determine working directory: %w", err)
	}
	path, failure := config.ResolveEnvFilePath(declared, c.EnvFileScopeAt(wd))
	if failure != nil {
		// The existing bridge code set is closed. A new resolution failure is
		// codeless rather than misclassified as a different frozen condition.
		return "", fmt.Errorf("env bridge: cannot resolve target %s: %s", declared, failure.Reason())
	}
	return path, nil
}

// declaredTargetError removes filesystem path payloads from errors for expanded
// targets, keeping the operation/cause and the declared selector. Post-rename
// failures must retain their distinct "already replaced" semantics.
func declaredTargetError(err error, declared string) error {
	if err == nil {
		return nil
	}
	var op string
	var reason error
	if le, ok := errors.AsType[*os.LinkError](err); ok {
		op, reason = le.Op, le.Err
	} else if pe, ok := errors.AsType[*fs.PathError](err); ok {
		op, reason = pe.Op, pe.Err
	} else {
		return err
	}
	cause := fmt.Errorf("%s %s: %w", op, declared, reason)
	if _, postRename := errors.AsType[*postRenameError](err); postRename {
		return &postRenameError{target: declared, err: cause}
	}
	return fmt.Errorf("env bridge: %w", cause)
}
