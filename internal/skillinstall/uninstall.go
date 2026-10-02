package skillinstall

import (
	"errors"
	"fmt"
	"os"
	"syscall"

	"github.com/ScriptonBasestar/dva/internal/skillclaim"
)

// Uninstall removes only a verified DVA-owned installation.
func Uninstall(options Options) (Result, error) {
	resolved, destinations, err := resolve(options)
	if err != nil {
		return Result{}, err
	}
	for _, target := range destinations {
		if err := preflightUninstall(resolved, target); err != nil {
			return Result{}, err
		}
	}
	result := Result{Scope: resolved.Scope, Destinations: make([]DestinationResult, 0, len(destinations))}
	for _, target := range destinations {
		entry, err := uninstallDestination(resolved, target)
		if err != nil {
			return Result{}, err
		}
		result.Destinations = append(result.Destinations, entry)
	}
	return result, nil
}

func preflightUninstall(options Options, target destination) error {
	record, found, err := readReceipt(receiptPath(options.StateRoot, target.path))
	if err != nil {
		return err
	}
	if !found {
		bundle, err := bundleFor(target)
		if err != nil {
			return err
		}
		projection, err := projectedClaims(target, options.Scope, target.runtimes, bundle, skillclaim.StateActive, "preflight")
		if err != nil {
			return err
		}
		return ensureClaimsAbsentUnlocked(options.ClaimRoot, projection)
	}
	if err := validateReceipt(record, options.Scope, target); err != nil {
		return err
	}
	if options.RestoreTakeoverBackup {
		if len(record.Takeovers) == 0 {
			return fmt.Errorf("no takeover backup exists for %s", target.path)
		}
		if status, _ := verifyTakeoverBackups(options.StateRoot, record); status != "available" {
			return fmt.Errorf("takeover backup for %s is %s", target.path, status)
		}
		if record.Installation == "active" && len(removeRuntimes(record.Runtimes, target.runtimes)) > 0 {
			return errors.New("restore requires selecting every consumer of the shared skill destination")
		}
	}
	if record.Schema == receiptSchemaCurrent && record.Installation == "absent" {
		bundle := skillBundle{files: record.Files}
		projection, err := projectedClaims(target, options.Scope, target.runtimes, bundle, skillclaim.StateActive, "preflight")
		if err != nil {
			return err
		}
		if err := ensureClaimsAbsentUnlocked(options.ClaimRoot, projection); err != nil {
			return err
		}
		if hasForeignCollision(target.path, record.Files) {
			return errors.New("backup-only destination contains a DVA-name collision")
		}
		return nil
	}
	if err := verifyInstalled(target.path, record.Files); err != nil {
		return fmt.Errorf("refusing to uninstall drifted DVA skill installation at %s: %w", target.path, err)
	}
	projection, err := projectedClaims(target, options.Scope, record.Runtimes, skillBundle{files: record.Files}, skillclaim.StateActive, "preflight")
	if err != nil {
		return err
	}
	if record.Schema < receiptSchemaCurrent {
		return ensureClaimsAbsentUnlocked(options.ClaimRoot, projection)
	}
	return verifyClaimsUnlocked(options.ClaimRoot, projection)
}

func uninstallDestination(options Options, target destination) (DestinationResult, error) {
	entry := resultEntry(target, "", "")
	receiptFile := receiptPath(options.StateRoot, target.path)
	record, found, err := readReceipt(receiptFile)
	if err != nil {
		return DestinationResult{}, err
	}
	if !found {
		bundle, bundleErr := bundleFor(target)
		if bundleErr != nil {
			return DestinationResult{}, bundleErr
		}
		if options.DryRun {
			entry.Status = "not-installed"
			setAllRuntimeStatuses(&entry, entry.Status)
			return entry, nil
		}
		claimPaths, claimErr := claimDestinations(target, bundle)
		if claimErr != nil {
			return DestinationResult{}, claimErr
		}
		store, claimErr := skillclaim.Begin(options.ClaimRoot, claimPaths)
		if claimErr != nil {
			return DestinationResult{}, claimErr
		}
		defer func() { _ = store.Close() }()
		projection, claimErr := projectedClaims(target, options.Scope, target.runtimes, bundle, skillclaim.StateActive, "uninstall-check")
		if claimErr != nil {
			return DestinationResult{}, claimErr
		}
		if claimErr := ensureClaimsAbsent(store, projection); claimErr != nil {
			return DestinationResult{}, claimErr
		}
		entry.Status = "not-installed"
		setAllRuntimeStatuses(&entry, entry.Status)
		return entry, nil
	}
	if err := validateReceipt(record, options.Scope, target); err != nil {
		return DestinationResult{}, err
	}
	if options.DryRun {
		if options.RestoreTakeoverBackup {
			entry.Status, entry.BackupStatus = "would-restore-takeover", "would-restore"
		} else if record.Installation == "absent" {
			entry.Status = "not-installed"
		} else {
			entry.Status = "would-uninstall"
		}
		setAllRuntimeStatuses(&entry, entry.Status)
		return entry, nil
	}
	if err := ensureDestination(target.path); err != nil {
		return DestinationResult{}, err
	}
	bundle := skillBundle{files: record.Files}
	destinations, err := claimDestinations(target, bundle)
	if err != nil {
		return DestinationResult{}, err
	}
	store, err := skillclaim.Begin(options.ClaimRoot, destinations)
	if err != nil {
		return DestinationResult{}, err
	}
	defer func() { _ = store.Close() }()
	record, found, err = readReceipt(receiptFile)
	if err != nil || !found {
		return DestinationResult{}, fmt.Errorf("receipt changed before uninstall: %w", err)
	}
	if record.Installation == "absent" {
		if !options.RestoreTakeoverBackup {
			entry.Status = "not-installed"
			setAllRuntimeStatuses(&entry, entry.Status)
			return entry, nil
		}
		return restoreBackupOnly(options, target, record, store, entry)
	}
	if err := verifyInstalled(target.path, record.Files); err != nil {
		return DestinationResult{}, err
	}
	operationID, err := newClaimOperationID()
	if err != nil {
		return DestinationResult{}, err
	}
	projection, err := projectedClaims(target, options.Scope, record.Runtimes, bundle, skillclaim.StateActive, operationID)
	if err != nil {
		return DestinationResult{}, err
	}
	installedRequested := intersectRuntimes(record.Runtimes, target.runtimes)
	if len(installedRequested) == 0 && record.Schema < receiptSchemaCurrent {
		entry.Status = "not-installed"
		setAllRuntimeStatuses(&entry, entry.Status)
		return entry, nil
	}
	var current []skillclaim.Claim
	if record.Schema < receiptSchemaCurrent {
		if err := ensureClaimsAbsent(store, projection); err != nil {
			return DestinationResult{}, err
		}
		reserved, err := reserveClaims(store, projection)
		if err != nil {
			_ = rollbackClaimsToAbsent(store, reserved)
			return DestinationResult{}, err
		}
		current, err = activateReservedClaims(store, reserved)
		if err != nil {
			return DestinationResult{}, fmt.Errorf("migrate legacy receipt claims: %w; recovery-required", err)
		}
	} else {
		current, err = readLockedClaims(store, projection)
		if err != nil {
			return DestinationResult{}, err
		}
		if err := verifyActiveClaims(current, projection); err != nil {
			return DestinationResult{}, err
		}
	}
	if len(installedRequested) == 0 {
		entry.Status = "not-installed"
		setAllRuntimeStatuses(&entry, entry.Status)
		return entry, nil
	}
	remaining := removeRuntimes(record.Runtimes, target.runtimes)
	if options.RestoreTakeoverBackup && len(remaining) > 0 {
		return DestinationResult{}, errors.New("restore requires selecting every consumer")
	}
	if len(remaining) > 0 {
		return unlinkConsumers(options, target, record, store, current, remaining, operationID, entry)
	}
	if options.RestoreTakeoverBackup {
		return restoreActiveTakeover(options, target, record, store, current, operationID, entry)
	}
	return removeLastConsumer(options, target, record, store, current, operationID, entry)
}

func unlinkConsumers(options Options, target destination, record receipt, store *skillclaim.LockedStore, current []skillclaim.Claim, remaining []Runtime, operationID string, entry DestinationResult) (DestinationResult, error) {
	desired, err := projectedClaims(target, options.Scope, remaining, skillBundle{files: record.Files}, skillclaim.StateActive, operationID)
	if err != nil {
		return DestinationResult{}, err
	}
	updating, err := transitionActiveClaims(store, current, desired, skillclaim.StateUpdating, operationID)
	if err != nil {
		return DestinationResult{}, err
	}
	updated := record
	updated.Schema = receiptSchemaCurrent
	updated.Installation = "active"
	updated.Format = targetReceiptFormat(target)
	updated.Runtimes = remaining
	if err := writeReceipt(receiptPath(options.StateRoot, target.path), updated); err != nil {
		claimErr := rollbackClaimsToActive(store, current)
		return DestinationResult{}, fmt.Errorf("update receipt: %w (claim rollback: %v)", err, claimErr)
	}
	if err := activateUpdatedClaims(store, updating); err != nil {
		receiptErr := writeReceipt(receiptPath(options.StateRoot, target.path), record)
		claimErr := rollbackClaimsToActive(store, current)
		return DestinationResult{}, fmt.Errorf("activate consumer update: %w (receipt rollback: %v; claim rollback: %v)", err, receiptErr, claimErr)
	}
	entry.Status = "unlinked"
	setMembershipStatuses(&entry, intersectRuntimes(record.Runtimes, target.runtimes), "unlinked", "not-installed")
	return entry, nil
}

func removeLastConsumer(options Options, target destination, record receipt, store *skillclaim.LockedStore, current []skillclaim.Claim, operationID string, entry DestinationResult) (DestinationResult, error) {
	releasing, err := transitionActiveClaims(store, current, current, skillclaim.StateReleasing, operationID)
	if err != nil {
		return DestinationResult{}, err
	}
	rollbackFiles, finalizeFiles, recoveryStage, err := stageManagedRemoval(target.path, record.Files)
	if err != nil {
		claimErr := rollbackClaimsToActive(store, current)
		return DestinationResult{}, fmt.Errorf("stage DVA skill removal: %w (claim rollback: %v)", err, claimErr)
	}
	if err := removeTransitionedClaims(store, releasing); err != nil {
		fileErr := rollbackFiles()
		claimErr := rollbackClaimsToActive(store, current)
		if fileErr != nil || claimErr != nil {
			return DestinationResult{}, fmt.Errorf("remove DVA claim: %w (file rollback: %v; claim rollback: %v; recovery stage: %s)", err, fileErr, claimErr, recoveryStage)
		}
		return DestinationResult{}, fmt.Errorf("remove DVA claim: %w", err)
	}
	receiptFile := receiptPath(options.StateRoot, target.path)
	if len(record.Takeovers) > 0 {
		tombstone := record
		tombstone.Schema = receiptSchemaCurrent
		tombstone.Installation = "absent"
		tombstone.Runtimes = nil
		if err := writeReceipt(receiptFile, tombstone); err != nil {
			fileErr := rollbackFiles()
			claimErr := rollbackClaimsToActive(store, current)
			return DestinationResult{}, fmt.Errorf("write backup-only receipt: %w (file rollback: %v; claim rollback: %v; recovery stage: %s)", err, fileErr, claimErr, recoveryStage)
		}
	} else if err := os.Remove(receiptFile); err != nil && !errors.Is(err, os.ErrNotExist) {
		fileErr := rollbackFiles()
		claimErr := rollbackClaimsToActive(store, current)
		return DestinationResult{}, fmt.Errorf("remove receipt: %w (file rollback: %v; claim rollback: %v; recovery stage: %s)", err, fileErr, claimErr, recoveryStage)
	}
	if err := finalizeFiles(); err != nil {
		entry.Detail = fmt.Sprintf("uninstalled; cleanup retained a recovery artifact at %s: %v", recoveryStage, err)
	}
	if hasRuntime(target.runtimes, RuntimeAgentMesh) {
		if err := os.Remove(target.path); err != nil && !errors.Is(err, os.ErrNotExist) && !errors.Is(err, syscall.ENOTEMPTY) {
			return DestinationResult{}, err
		}
	}
	entry.Status = "uninstalled"
	setMembershipStatuses(&entry, intersectRuntimes(record.Runtimes, target.runtimes), "uninstalled", "not-installed")
	if len(record.Takeovers) > 0 {
		entry.BackupStatus, entry.TakeoverBackup = verifyTakeoverBackups(options.StateRoot, receipt{Destination: record.Destination, Takeovers: record.Takeovers})
	}
	return entry, nil
}

func restoreActiveTakeover(options Options, target destination, record receipt, store *skillclaim.LockedStore, current []skillclaim.Claim, operationID string, entry DestinationResult) (DestinationResult, error) {
	if status, _ := verifyTakeoverBackups(options.StateRoot, record); status != "available" {
		return DestinationResult{}, fmt.Errorf("takeover backup is %s", status)
	}
	restoring, err := transitionActiveClaims(store, current, current, skillclaim.StateRestoring, operationID)
	if err != nil {
		return DestinationResult{}, err
	}
	rollbackRestore, finalizeRestore, err := replaceWithTakeoverBackups(options.StateRoot, record)
	if err != nil {
		claimErr := rollbackClaimsToActive(store, current)
		return DestinationResult{}, fmt.Errorf("restore takeover backup: %w (claim rollback: %v)", err, claimErr)
	}
	if err := removeTransitionedClaims(store, restoring); err != nil {
		fileErr := rollbackRestore()
		claimErr := rollbackClaimsToActive(store, current)
		return DestinationResult{}, fmt.Errorf("release restored claims: %w (file rollback: %v; claim rollback: %v)", err, fileErr, claimErr)
	}
	if err := os.Remove(receiptPath(options.StateRoot, target.path)); err != nil && !errors.Is(err, os.ErrNotExist) {
		fileErr := rollbackRestore()
		claimErr := rollbackClaimsToActive(store, current)
		return DestinationResult{}, fmt.Errorf("remove restored receipt: %w (file rollback: %v; claim rollback: %v)", err, fileErr, claimErr)
	}
	if err := finalizeRestore(); err != nil {
		entry.Detail = fmt.Sprintf("restored; cleanup retained a recovery artifact: %v", err)
	}
	entry.Status, entry.BackupStatus = "restored-takeover", "restored"
	setAllRuntimeStatuses(&entry, "restored-takeover")
	return entry, nil
}

func restoreBackupOnly(options Options, target destination, record receipt, store *skillclaim.LockedStore, entry DestinationResult) (DestinationResult, error) {
	if status, _ := verifyTakeoverBackups(options.StateRoot, record); status != "available" {
		return DestinationResult{}, fmt.Errorf("takeover backup is %s", status)
	}
	if hasForeignCollision(target.path, record.Files) {
		return DestinationResult{}, errors.New("refusing restore because a DVA-name destination reappeared")
	}
	operationID, err := newClaimOperationID()
	if err != nil {
		return DestinationResult{}, err
	}
	projection, err := projectedClaims(target, options.Scope, target.runtimes, skillBundle{files: record.Files}, skillclaim.StateReserved, operationID)
	if err != nil {
		return DestinationResult{}, err
	}
	if err := ensureClaimsAbsent(store, projection); err != nil {
		return DestinationResult{}, err
	}
	reserved, err := reserveClaims(store, projection)
	if err != nil {
		_ = rollbackClaimsToAbsent(store, reserved)
		return DestinationResult{}, err
	}
	active, err := activateReservedClaims(store, reserved)
	if err != nil {
		return DestinationResult{}, fmt.Errorf("reserve restore claims: %w; recovery-required", err)
	}
	restoring, err := transitionActiveClaims(store, active, active, skillclaim.StateRestoring, operationID+"-restore")
	if err != nil {
		return DestinationResult{}, err
	}
	rollbackRestore, finalizeRestore, err := replaceWithTakeoverBackups(options.StateRoot, record)
	if err != nil {
		claimErr := rollbackClaimsToAbsent(store, restoring)
		return DestinationResult{}, fmt.Errorf("restore takeover backup: %w (claim rollback: %v)", err, claimErr)
	}
	if err := removeTransitionedClaims(store, restoring); err != nil {
		fileErr := rollbackRestore()
		claimErr := rollbackClaimsToAbsent(store, restoring)
		return DestinationResult{}, fmt.Errorf("release restore claims: %w (file rollback: %v; claim rollback: %v)", err, fileErr, claimErr)
	}
	if err := os.Remove(receiptPath(options.StateRoot, target.path)); err != nil && !errors.Is(err, os.ErrNotExist) {
		fileErr := rollbackRestore()
		claimErr := rollbackClaimsToAbsent(store, restoring)
		return DestinationResult{}, fmt.Errorf("remove backup-only receipt: %w (file rollback: %v; claim rollback: %v)", err, fileErr, claimErr)
	}
	if err := finalizeRestore(); err != nil {
		entry.Detail = fmt.Sprintf("restored; cleanup retained a recovery artifact: %v", err)
	}
	entry.Status, entry.BackupStatus = "restored-takeover", "restored"
	setAllRuntimeStatuses(&entry, "restored-takeover")
	return entry, nil
}
