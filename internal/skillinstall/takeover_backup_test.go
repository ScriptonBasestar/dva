package skillinstall

import (
	"crypto/sha256"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ScriptonBasestar/dva/internal/skillclaim"
)

func TestRestoreTakeoverBacksUpAndExplicitRestoreReturnsExactTree(t *testing.T) {
	t.Parallel()
	options := testOptions(t, ScopeUser, RuntimeCodex)
	_, destinations, err := resolve(options)
	if err != nil {
		t.Fatal(err)
	}
	foreign := filepath.Join(destinations[0].path, "dva")
	empty := filepath.Join(foreign, "nested", "empty")
	if err := os.MkdirAll(empty, 0o711); err != nil {
		t.Fatal(err)
	}
	original := filepath.Join(foreign, "nested", "original.txt")
	if err := os.WriteFile(original, []byte("exact original"), 0o640); err != nil {
		t.Fatal(err)
	}
	if _, err := Install(options); err == nil {
		t.Fatal("receipt-less collision installed without takeover")
	}
	options.Takeover = true
	installed, err := Install(options)
	if err != nil {
		t.Fatal(err)
	}
	if installed.Destinations[0].BackupStatus != "available" {
		t.Fatalf("backup status = %q", installed.Destinations[0].BackupStatus)
	}
	if _, err := os.Stat(original); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("foreign tree remains after takeover: %v", err)
	}
	options.Takeover = false
	if _, err := Uninstall(options); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(original); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("ordinary uninstall restored the original automatically: %v", err)
	}
	status, err := Status(options)
	if err != nil {
		t.Fatal(err)
	}
	if status.Destinations[0].Status != "backup-only" || status.Destinations[0].BackupStatus != "available" {
		t.Fatalf("backup-only status = %#v", status.Destinations[0])
	}
	options.RestoreTakeoverBackup = true
	if _, err := Uninstall(options); err != nil {
		t.Fatal(err)
	}
	contents, err := os.ReadFile(original)
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != "exact original" {
		t.Fatalf("restored contents = %q", contents)
	}
	info, err := os.Stat(original)
	if err != nil || info.Mode().Perm() != 0o640 {
		t.Fatalf("restored file mode = %v, err=%v", info, err)
	}
	info, err = os.Stat(empty)
	if err != nil || !info.IsDir() || info.Mode().Perm() != 0o711 {
		t.Fatalf("empty directory was not restored exactly: %v, err=%v", info, err)
	}
}

func TestTakeoverRefusesOtherProducerClaim(t *testing.T) {
	t.Parallel()
	options := testOptions(t, ScopeUser, RuntimeCodex)
	_, destinations, err := resolve(options)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(destinations[0].path, "dva")
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, "SKILL.md"), []byte("foreign"), 0o600); err != nil {
		t.Fatal(err)
	}
	// A well-formed foreign claim is the authoritative refusal signal.
	claimRoot := filepath.Dir(options.StateRoot)
	canonical, err := skillclaim.CanonicalDestination(path)
	if err != nil {
		t.Fatal(err)
	}
	files := []skillclaim.FileHash{{Path: "SKILL.md", SHA: strings.Repeat("0", sha256.Size*2)}}
	sourceDigest, err := skillclaim.ManifestDigest(files)
	if err != nil {
		t.Fatal(err)
	}
	claim := skillclaim.Claim{Schema: skillclaim.Schema, Name: "dva", Kind: skillclaim.KindDirectory, State: skillclaim.StateReserved, OperationID: "foreign-test", Generation: 1, Destination: canonical, Producer: "other", Format: "agent-skills-directory", Scope: "user", Consumers: []string{"codex"}, SourceDigest: sourceDigest, Files: files}
	store, err := skillclaim.Begin(claimRoot, []string{canonical})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Reserve(claim); err != nil {
		t.Fatal(err)
	}
	previous, err := skillclaim.Digest(claim)
	if err != nil {
		t.Fatal(err)
	}
	claim.State, claim.Generation = skillclaim.StateActive, 2
	if err := store.CompareAndSwap(claim, 1, previous); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	options.Takeover = true
	if _, err := Install(options); err == nil {
		t.Fatal("takeover accepted other producer claim")
	}
}

func TestCorruptTakeoverBackupBlocksRestore(t *testing.T) {
	t.Parallel()
	options := testOptions(t, ScopeUser, RuntimeCodex)
	_, destinations, err := resolve(options)
	if err != nil {
		t.Fatal(err)
	}
	foreign := filepath.Join(destinations[0].path, "dva")
	if err := os.MkdirAll(foreign, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(foreign, "marker"), []byte("foreign"), 0o600); err != nil {
		t.Fatal(err)
	}
	options.Takeover = true
	if _, err := Install(options); err != nil {
		t.Fatal(err)
	}
	record, found, err := readReceipt(receiptPath(options.StateRoot, destinations[0].path))
	if err != nil || !found || len(record.Takeovers) == 0 {
		t.Fatalf("takeover receipt = (%#v, %t, %v)", record, found, err)
	}
	backup, err := takeoverBackupPath(options.StateRoot, record.Destination, record.Takeovers[0])
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Join(backup, "marker"), 0o644); err != nil {
		t.Fatal(err)
	}
	status, err := Status(options)
	if err != nil {
		t.Fatal(err)
	}
	if status.Destinations[0].BackupStatus != "corrupt" {
		t.Fatalf("corrupt backup status = %#v", status.Destinations[0])
	}
	options.Takeover, options.RestoreTakeoverBackup = false, true
	if _, err := Uninstall(options); err == nil {
		t.Fatal("restore accepted corrupt takeover backup")
	}
}

func TestRestoreRejectsIncompleteBackupInventory(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(t *testing.T, options Options, target destination, record *receipt)
	}{
		{
			name: "omitted receipt record",
			mutate: func(t *testing.T, options Options, target destination, record *receipt) {
				t.Helper()
				record.Takeovers = append([]takeoverBackup(nil), record.Takeovers[:1]...)
				if err := writeReceipt(receiptPath(options.StateRoot, target.path), *record); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "extra backup sibling",
			mutate: func(t *testing.T, options Options, target destination, record *receipt) {
				t.Helper()
				path, err := takeoverBackupPath(options.StateRoot, record.Destination, record.Takeovers[0])
				if err != nil {
					t.Fatal(err)
				}
				if err := os.Mkdir(filepath.Join(filepath.Dir(path), "unlisted"), 0o700); err != nil {
					t.Fatal(err)
				}
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			options := testOptions(t, ScopeProject, RuntimeCodex)
			_, targets, err := resolve(options)
			if err != nil {
				t.Fatal(err)
			}
			target := targets[0]
			for _, name := range []string{"dva", "dva-config"} {
				root := filepath.Join(target.path, name)
				if err := os.MkdirAll(root, 0o750); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(root, "original.txt"), []byte("foreign-"+name), 0o640); err != nil {
					t.Fatal(err)
				}
			}
			options.Takeover = true
			if _, err := Install(options); err != nil {
				t.Fatal(err)
			}
			record, found, err := readReceipt(receiptPath(options.StateRoot, target.path))
			if err != nil || !found || len(record.Takeovers) != 2 {
				t.Fatalf("takeover receipt = (%#v, %t, %v)", record, found, err)
			}
			backupRoot := filepath.Join(takeoverDestinationRoot(options.StateRoot, record.Destination), record.Takeovers[0].BackupID)
			test.mutate(t, options, target, &record)
			options.Takeover, options.RestoreTakeoverBackup = false, true
			if _, err := Uninstall(options); err == nil {
				t.Fatal("restore accepted incomplete backup inventory")
			}
			if _, err := os.Stat(filepath.Join(target.path, "dva", "SKILL.md")); err != nil {
				t.Fatalf("failed restore changed installed DVA skill: %v", err)
			}
			if _, err := os.Stat(filepath.Join(backupRoot, "dva-config", "original.txt")); err != nil {
				t.Fatalf("failed restore deleted unlisted original: %v", err)
			}
		})
	}
}

func TestTakeoverBackupCapturesLiveEntryBeforeReplacement(t *testing.T) {
	t.Parallel()
	options := testOptions(t, ScopeProject, RuntimeCodex)
	_, targets, err := resolve(options)
	if err != nil {
		t.Fatal(err)
	}
	target := targets[0]
	if err := ensureDestination(target.path); err != nil {
		t.Fatal(err)
	}
	foreign := filepath.Join(target.path, "dva")
	if err := os.MkdirAll(foreign, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(foreign, "original"), []byte("exact foreign bytes"), 0o640); err != nil {
		t.Fatal(err)
	}
	bundle, err := bundleFor(target)
	if err != nil {
		t.Fatal(err)
	}
	records, rollback, _, cleanup, recovery, err := createTakeoverBackups(options.StateRoot, target, bundle)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 || records[0].Skill != "dva" {
		t.Fatalf("captured takeover records = %#v", records)
	}
	if _, err := os.Lstat(foreign); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("foreign entry was not atomically captured: %v", err)
	}
	backup, err := os.ReadFile(filepath.Join(recovery, "dva", "original"))
	if err != nil || string(backup) != "exact foreign bytes" {
		t.Fatalf("durable snapshot = %q, %v", backup, err)
	}
	if err := rollback(); err != nil {
		t.Fatal(err)
	}
	restored, err := os.ReadFile(filepath.Join(foreign, "original"))
	if err != nil || string(restored) != "exact foreign bytes" {
		t.Fatalf("restored original = %q, %v", restored, err)
	}
	if err := cleanup(); err != nil {
		t.Fatal(err)
	}
}

func TestTakeoverRollbackCollisionRetainsDurableRecoveryArtifact(t *testing.T) {
	t.Parallel()
	options := testOptions(t, ScopeProject, RuntimeCodex)
	_, targets, err := resolve(options)
	if err != nil {
		t.Fatal(err)
	}
	target := targets[0]
	if err := ensureDestination(target.path); err != nil {
		t.Fatal(err)
	}
	foreign := filepath.Join(target.path, "dva")
	if err := os.MkdirAll(foreign, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(foreign, "original"), []byte("recover me"), 0o600); err != nil {
		t.Fatal(err)
	}
	bundle, err := bundleFor(target)
	if err != nil {
		t.Fatal(err)
	}
	_, rollback, _, _, recovery, err := createTakeoverBackups(options.StateRoot, target, bundle)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(foreign, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := rollback(); err == nil {
		t.Fatal("rollback accepted a late collision")
	}
	contents, err := os.ReadFile(filepath.Join(recovery, "dva", "original"))
	if err != nil || string(contents) != "recover me" {
		t.Fatalf("recovery artifact = %q, %v", contents, err)
	}
}

func TestClaimRollbackRestoresActiveSetAfterPartialRelease(t *testing.T) {
	t.Parallel()
	options := testOptions(t, ScopeProject, RuntimeCodex)
	installed, err := Install(options)
	if err != nil {
		t.Fatal(err)
	}
	target := destination{path: installed.Destinations[0].Destination, runtimes: options.Runtimes}
	record, found, err := readReceipt(receiptPath(options.StateRoot, target.path))
	if err != nil || !found {
		t.Fatal(err)
	}
	bundle := skillBundle{files: record.Files}
	paths, err := claimDestinations(target, bundle)
	if err != nil {
		t.Fatal(err)
	}
	store, err := skillclaim.Begin(filepath.Dir(options.StateRoot), paths)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()
	projection, err := projectedClaims(target, options.Scope, record.Runtimes, bundle, skillclaim.StateActive, "rollback-test")
	if err != nil {
		t.Fatal(err)
	}
	current, err := readLockedClaims(store, projection)
	if err != nil {
		t.Fatal(err)
	}
	releasing, err := transitionActiveClaims(store, current, current, skillclaim.StateReleasing, "partial-release")
	if err != nil {
		t.Fatal(err)
	}
	previous, err := skillclaim.Digest(releasing[0])
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Remove(releasing[0].Destination, releasing[0].Producer, releasing[0].OperationID, releasing[0].Generation, previous); err != nil {
		t.Fatal(err)
	}
	if err := rollbackClaimsToActive(store, current); err != nil {
		t.Fatal(err)
	}
	restored, err := readLockedClaims(store, projection)
	if err != nil {
		t.Fatal(err)
	}
	if err := verifyActiveClaims(restored, projection); err != nil {
		t.Fatal(err)
	}
}

func TestClaimRollbackRemovesPartialTemporaryRestoreSet(t *testing.T) {
	t.Parallel()
	options := testOptions(t, ScopeProject, RuntimeCodex)
	_, targets, err := resolve(options)
	if err != nil {
		t.Fatal(err)
	}
	target := targets[0]
	if err := ensureDestination(target.path); err != nil {
		t.Fatal(err)
	}
	bundle, err := bundleFor(target)
	if err != nil {
		t.Fatal(err)
	}
	paths, err := claimDestinations(target, bundle)
	if err != nil {
		t.Fatal(err)
	}
	store, err := skillclaim.Begin(filepath.Dir(options.StateRoot), paths)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()
	reserved, err := projectedClaims(target, options.Scope, options.Runtimes, bundle, skillclaim.StateReserved, "temporary-restore")
	if err != nil {
		t.Fatal(err)
	}
	reserved, err = reserveClaims(store, reserved)
	if err != nil {
		t.Fatal(err)
	}
	active, err := activateReservedClaims(store, reserved)
	if err != nil {
		t.Fatal(err)
	}
	restoring, err := transitionActiveClaims(store, active, active, skillclaim.StateRestoring, "temporary-restore-mutation")
	if err != nil {
		t.Fatal(err)
	}
	previous, err := skillclaim.Digest(restoring[0])
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Remove(restoring[0].Destination, restoring[0].Producer, restoring[0].OperationID, restoring[0].Generation, previous); err != nil {
		t.Fatal(err)
	}
	if err := rollbackClaimsToAbsent(store, restoring); err != nil {
		t.Fatal(err)
	}
	if err := ensureClaimsAbsent(store, reserved); err != nil {
		t.Fatal(err)
	}
}
