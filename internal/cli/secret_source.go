package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ScriptonBasestar/dva/internal/config"
	"github.com/ScriptonBasestar/dva/internal/output"
	"github.com/spf13/cobra"
)

func init() {
	for _, verb := range []string{"edit", "reseal"} {
		cmd := &cobra.Command{Use: verb + " <source>", Short: verb + " a declared SOPS source", Args: cobra.ExactArgs(1)}
		if verb == "edit" {
			cmd.Long = "Open an existing secrets.sources alias in SOPS using its declared format (dotenv by default). The source's project field anchors its path in a declared subproject; --project instead selects a child's own dva.yml and refuses a second project field. SOPS owns the editor and its temporary plaintext. Interactive display uses the controlling terminal; child stdout and diagnostics are never copied into DVA results. --json and --dry-run are unsupported."
		} else {
			cmd.Long = "Re-encrypt an existing secrets.sources alias with the creation rules selected by its owning config's sops policy (nearest by default, root or an explicit config path when declared). Plaintext flows through a pipe, never a DVA plaintext file. Only after decryption and encryption both succeed is the ciphertext atomically replaced; a directory-flush failure after replacement is reported explicitly. --project selects a child's own declarations; a source project field cannot be combined with that flag. --json reports metadata only; --dry-run is unsupported."
		}
		cmd.Flags().String("project", "", "Use a declared child project's own secret sources")
		cmd.RunE = runSecretSource
		secretCmd.AddCommand(cmd)
	}
}

// openSecretSource anchors filesystem access at the declared source owner.
// The configuration owner remains separate: its sops policy and targets apply.
func openSecretSource(c *config.Config, name string) (*envRoot, config.SecretSource, error) {
	if c.Secrets == nil {
		return nil, config.SecretSource{}, fmt.Errorf("no secret sources are declared")
	}
	source, ok := c.Secrets.Sources[name]
	if !ok {
		return nil, source, fmt.Errorf("unknown secret source %q", name)
	}
	root, err := openSecretSourceRoot(c, source)
	if err != nil {
		return nil, source, err
	}
	state, info, err := root.checkPath(source.Sops)
	if err == nil && (state != pathPresent || !info.Mode().IsRegular()) {
		err = fmt.Errorf("secret source %q must be an existing regular file", name)
	}
	if err != nil {
		root.Close()
		return nil, source, err
	}
	return root, source, nil
}

func openSecretSourceRoot(c *config.Config, source config.SecretSource) (*envRoot, error) {
	dir, err := filepath.EvalSymlinks(c.FileDir())
	if err != nil {
		return nil, err
	}
	if source.Project != "" {
		sub, ok := c.Subprojects[source.Project]
		if !ok {
			return nil, fmt.Errorf("unknown project %q", source.Project)
		}
		path := sub.Path
		if !filepath.IsAbs(path) {
			path = filepath.Join(dir, path)
		}
		// Declared subprojects may be outside the parent, but may not redirect
		// writes through a symlinked directory component.
		if err := checkSecretDirectory(path); err != nil {
			return nil, err
		}
		dir = path
	}
	root, err := openEnvRoot(dir)
	if err != nil {
		return nil, err
	}
	return root, nil
}

func checkSecretDirectory(path string) error {
	for dir := filepath.Clean(path); ; dir = filepath.Dir(dir) {
		info, err := os.Lstat(dir)
		if err != nil {
			return err
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("secret project path contains a symlink or non-directory")
		}
		if filepath.Dir(dir) == dir {
			return nil
		}
	}
}

func secretRules(c *config.Config, root *envRoot, source config.SecretSource, encrypting bool) (string, error) {
	policyRoot, err := filepath.EvalSymlinks(c.FileDir())
	if err != nil {
		return "", err
	}
	rules, err := config.ResolveSopsConfig(policyRoot, c.Sops, root.abs(source.Sops))
	if errors.Is(err, config.ErrNoSopsConfig) && !encrypting {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	// The shared resolver rejects a symlinked leaf. For a write-capable
	// command, also reject a symlink in the configured rules' parent path.
	policy, err := openEnvRoot(policyRoot)
	if err != nil {
		return "", err
	}
	defer policy.Close()
	rel, err := filepath.Rel(policyRoot, rules)
	if err != nil {
		return "", err
	}
	state, info, err := policy.checkPath(rel)
	if err != nil {
		return "", err
	}
	if state != pathPresent || !info.Mode().IsRegular() {
		return "", fmt.Errorf("secret rules must be a regular file")
	}
	return rules, nil
}

func runSecretSource(cmd *cobra.Command, args []string) error {
	if dryRun {
		return fmt.Errorf("secret %s does not support --dry-run", cmd.Name())
	}
	if cmd.Name() == "edit" && jsonOutput {
		return fmt.Errorf("secret edit does not support --json")
	}
	if !supportedBridgePlatforms[bridgeGOOS] {
		return fmt.Errorf("secret writes are unsupported on this platform")
	}
	c, err := remoteOwner(cmd)
	if err != nil {
		return err
	}
	root, source, err := openSecretSource(c, args[0])
	if err != nil {
		return err
	}
	defer root.Close()
	rules, err := secretRules(c, root, source, cmd.Name() == "reseal")
	if err != nil {
		return err
	}
	ctx, stop := remoteContext(cmd)
	defer stop()
	format := c.SecretSourceFormat(args[0])
	if cmd.Name() == "edit" {
		if err := editSecretSOPS(ctx, root.dir, root.abs(source.Sops), rules, format); err != nil {
			return err
		}
	} else {
		anchor, err := root.openTargetAnchor(source.Sops)
		if err != nil {
			return err
		}
		defer anchor.Close()
		cipher, err := anchor.newTemp()
		if err != nil {
			return err
		}
		defer cipher.Abort()
		if err := resealSecretSOPS(ctx, root.dir, root.abs(source.Sops), rules, format, cipher.File()); err != nil {
			return err
		}
		info, err := cipher.File().Stat()
		if err != nil {
			return err
		}
		if info.Size() == 0 {
			return fmt.Errorf("secret_encrypt_empty_output")
		}
		if err := cipher.Commit(); err != nil {
			return err
		}
	}
	if jsonOutput {
		return output.PrintJSON(map[string]string{"action": cmd.Name(), "source": args[0], "result": "replaced"})
	}
	fmt.Printf("%s completed for secret source %s\n", cmd.Name(), args[0])
	return nil
}
