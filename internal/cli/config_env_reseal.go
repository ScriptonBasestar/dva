package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"time"

	"github.com/ScriptonBasestar/dva/internal/output"
	"github.com/spf13/cobra"
)

// configEnvResealCmd re-encrypts an existing sops_source with the current
// creation rules. It is the key-rotation path: edit changes plaintext inside
// the envelope, seal refuses an envelope that already exists, and neither one
// rewraps an existing envelope after .sops.yaml changes.
var configEnvResealCmd = &cobra.Command{
	Use:   "reseal [target]",
	Short: "Re-encrypt an env_file entry's sops_source in place",
	Long: `Re-encrypt an env_file entry's sops_source in place.

[target] selects the entry the same way 'unseal' does — by its declared 'path'.
The command decrypts that entry's sops_source and writes a new ciphertext over
the same path. It does not read or write the plaintext target.

The replacement is atomic. A failure leaves the existing encrypted source
byte-for-byte unchanged.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := ""
		if len(args) == 1 {
			target = args[0]
		}
		return runEnvReseal(target)
	},
}

func runEnvReseal(target string) error {
	c, err := loadConfig()
	if err != nil {
		return err
	}
	if !supportedBridgePlatforms[bridgeGOOS] {
		return bridgeErr(codeUnsupportedPlatform, "this platform is not supported for env bridge writes")
	}
	if err := checkWritableOrigin(c); err != nil {
		return err
	}
	entry, err := selectEncryptedEntry(c, target)
	if err != nil {
		return err
	}

	root, err := openEnvRoot(c.FileDir())
	if err != nil {
		return err
	}
	defer root.Close()

	state, info, err := root.checkPath(entry.SopsSource)
	if err != nil {
		return err
	}
	switch state {
	case pathMissingLeaf, pathMissingParent:
		return bridgeErr(codeSourceMissing, "encrypted source %s does not exist", entry.SopsSource)
	case pathPresent:
		if !info.Mode().IsRegular() {
			return bridgeErr(codeSourceNotRegular, "encrypted source %s is not a regular file", entry.SopsSource)
		}
	}
	src, err := root.root.Open(entry.SopsSource)
	if err != nil {
		return bridgeErr(codeSourceUnreadable, "cannot read encrypted source %s", entry.SopsSource)
	}
	_ = src.Close()

	if !bridgeSops.Available() {
		return bridgeErr(codeSopsNotFound, "sops is not installed or not on PATH")
	}

	anchor, err := root.openTargetAnchor(entry.SopsSource)
	if err != nil {
		if errors.Is(err, fs.ErrPermission) {
			return bridgeErr(codePermissionDenied, "permission denied writing %s", entry.SopsSource)
		}
		return err
	}
	defer anchor.Close()
	anchor.dir.reclaimStaleTemps(time.Now())

	plain, err := anchor.newTemp()
	if err != nil {
		if errors.Is(err, fs.ErrPermission) {
			return bridgeErr(codePermissionDenied, "permission denied writing %s", entry.SopsSource)
		}
		return err
	}
	defer plain.Abort()

	sourcePath := root.abs(entry.SopsSource)
	if err := bridgeSops.Decrypt(sourcePath, plain.File()); err != nil {
		return bridgeErr(codeDecryptFailed, "decryption failed for %s", entry.SopsSource)
	}
	if err := plain.File().Sync(); err != nil {
		return err
	}
	if err := validateDecrypted(&resolvedEntry{entry: entry, anchor: anchor}, plain); err != nil {
		return err
	}

	in, err := anchor.dir.root.Open(plain.name)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()

	cipher, err := anchor.newTemp()
	if err != nil {
		if errors.Is(err, fs.ErrPermission) {
			return bridgeErr(codePermissionDenied, "permission denied writing %s", entry.SopsSource)
		}
		return err
	}
	defer cipher.Abort()

	if err := bridgeSops.Encrypt(sourcePath, in, cipher.File()); err != nil {
		if errors.Is(err, errSopsCreationRuleMismatch) {
			return bridgeErr(codeSopsCreationRuleMissing,
				"no .sops.yaml creation rule matches %s", entry.SopsSource)
		}
		return bridgeErr(codeEncryptFailed, "encryption failed for %s", entry.SopsSource)
	}
	if err := cipher.File().Sync(); err != nil {
		return err
	}
	cipherInfo, err := anchor.dir.root.Stat(cipher.name)
	if err != nil {
		return err
	}
	if cipherInfo.Size() == 0 {
		return bridgeErr(codeEmptyOutput, "sops produced no output for %s", entry.SopsSource)
	}

	if err := cipher.Commit(); err != nil {
		if _, isPostRename := errors.AsType[*postRenameError](err); isPostRename {
			return err
		}
		if errors.Is(err, fs.ErrPermission) {
			return bridgeErr(codePermissionDenied, "permission denied writing %s", entry.SopsSource)
		}
		return err
	}

	if jsonOutput {
		return output.PrintJSON(map[string]any{
			"action": "reseal",
			"source": entry.SopsSource,
			"result": "replaced",
		})
	}
	fmt.Printf("resealed %s\n", entry.SopsSource)
	return nil
}
