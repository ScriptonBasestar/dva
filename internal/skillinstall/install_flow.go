package skillinstall

import (
	"errors"
	"fmt"
	"os"

	"github.com/ScriptonBasestar/dva/internal/skillclaim"
)

// Install copies the embedded skills into every selected runtime directory.
func Install(options Options) (Result, error) {
	resolved, destinations, err := resolve(options)
	if err != nil {
		return Result{}, err
	}
	// Inspect every destination before the first mutation. Claim locks repeat these
	// checks at the mutation boundary, but this pass avoids predictable partial work.
	for _, target := range destinations {
		if err := preflightInstall(resolved, target); err != nil {
			return Result{}, err
		}
	}
	result := Result{Scope: resolved.Scope, Destinations: make([]DestinationResult, 0, len(destinations))}
	for _, target := range destinations {
		entry, err := installDestination(resolved, target)
		if err != nil {
			return Result{}, err
		}
		result.Destinations = append(result.Destinations, entry)
	}
	return result, nil
}

func preflightInstall(options Options, target destination) error {
	bundle, err := bundleFor(target)
	if err != nil {
		return err
	}
	record, found, err := readReceipt(receiptPath(options.StateRoot, target.path))
	if err != nil {
		return fmt.Errorf("read receipt for %s: %w", target.path, err)
	}
	operationID := "preflight"
	projection, err := projectedClaims(target, options.Scope, target.runtimes, bundle, skillclaim.StateActive, operationID)
	if err != nil {
		return err
	}
	if !found || (record.Schema == receiptSchemaCurrent && record.Installation == "absent") {
		if err := ensureClaimsAbsentUnlocked(options.ClaimRoot, projection); err != nil {
			return err
		}
		if found && hasForeignCollision(target.path, bundle.files) {
			return fmt.Errorf("refusing collision at %s while a takeover backup is retained", target.path)
		}
		if !found {
			if err := ensureNoCollision(target.path, bundle.files); err != nil && !options.Takeover {
				return err
			}
			if options.Takeover {
				return validateTakeover(target, bundle)
			}
		}
		return nil
	}
	if err := validateReceipt(record, options.Scope, target); err != nil {
		return err
	}
	if err := verifyInstalled(target.path, record.Files); err != nil {
		return fmt.Errorf("refusing to update drifted DVA skill installation at %s: %w", target.path, err)
	}
	oldProjection, err := projectedClaims(target, options.Scope, record.Runtimes, skillBundle{files: record.Files}, skillclaim.StateActive, operationID)
	if err != nil {
		return err
	}
	if record.Schema < receiptSchemaCurrent {
		if err := ensureClaimsAbsentUnlocked(options.ClaimRoot, oldProjection); err != nil {
			return err
		}
	} else if err := verifyClaimsUnlocked(options.ClaimRoot, oldProjection); err != nil {
		return err
	}
	runtimes := unionRuntimes(record.Runtimes, target.runtimes)
	desired, err := projectedClaims(target, options.Scope, runtimes, bundle, skillclaim.StateActive, operationID)
	if err != nil {
		return err
	}
	_, added, err := additiveClaimUpdate(oldProjection, desired)
	if err != nil {
		return err
	}
	if err := ensureClaimsAbsentUnlocked(options.ClaimRoot, added); err != nil {
		return err
	}
	return ensureClaimDestinationsAbsent(added)
}

func installDestination(options Options, target destination) (entry DestinationResult, err error) {
	bundle, err := bundleFor(target)
	if err != nil {
		return DestinationResult{}, err
	}
	entry = resultEntry(target, options.Version, sourceBundleSHA(bundle.files))
	record, found, err := readReceipt(receiptPath(options.StateRoot, target.path))
	if err != nil {
		return DestinationResult{}, err
	}
	if options.DryRun {
		entry.Status = "would-install"
		setAllRuntimeStatuses(&entry, "would-install")
		if !found && options.Takeover && hasForeignCollision(target.path, bundle.files) {
			entry.Detail = "would back up foreign DVA-name skill before takeover"
			entry.BackupStatus = "would-backup"
		}
		return entry, nil
	}
	if err := ensureDestination(target.path); err != nil {
		return DestinationResult{}, err
	}
	destinations, err := claimDestinations(target, bundle)
	if err != nil {
		return DestinationResult{}, err
	}
	if found && record.Schema == receiptSchemaCurrent && record.Installation == "active" {
		if err := validateReceipt(record, options.Scope, target); err != nil {
			return DestinationResult{}, err
		}
		oldDestinations, err := claimDestinations(target, skillBundle{files: record.Files})
		if err != nil {
			return DestinationResult{}, err
		}
		destinations = unionClaimDestinations(destinations, oldDestinations)
	}
	store, err := skillclaim.Begin(options.ClaimRoot, destinations)
	if err != nil {
		return DestinationResult{}, err
	}
	defer func() {
		if closeErr := store.Close(); closeErr != nil {
			if err != nil {
				err = errors.Join(err, fmt.Errorf("close claim store: %w", closeErr))
			} else {
				entry = DestinationResult{}
				err = fmt.Errorf("close claim store: %w", closeErr)
			}
		}
	}()

	// Repeat receipt and file checks while the per-skill claim locks are held.
	record, found, err = readReceipt(receiptPath(options.StateRoot, target.path))
	if err != nil {
		return DestinationResult{}, err
	}
	if found && record.Schema == receiptSchemaCurrent && record.Installation == "active" && equalFiles(record.Files, bundle.files) && containsRuntimes(record.Runtimes, target.runtimes) {
		if err := verifyInstalled(target.path, record.Files); err != nil {
			return DestinationResult{}, err
		}
		projection, err := projectedClaims(target, options.Scope, record.Runtimes, bundle, skillclaim.StateActive, "up-to-date-check")
		if err != nil {
			return DestinationResult{}, err
		}
		current, err := readLockedClaims(store, projection)
		if err != nil {
			return DestinationResult{}, err
		}
		if err := verifyActiveClaims(current, projection); err != nil {
			return DestinationResult{}, err
		}
		entry.Status = "up-to-date"
		setAllRuntimeStatuses(&entry, "up-to-date")
		if len(record.Takeovers) > 0 {
			entry.BackupStatus, entry.TakeoverBackup = verifyTakeoverBackups(options.StateRoot, record)
		}
		return entry, nil
	}
	if !found || record.Schema < receiptSchemaCurrent || record.Installation == "absent" {
		return installWithReservations(options, target, bundle, store, record, found, entry)
	}
	return updateClaimedInstall(options, target, bundle, store, record, entry)
}

func installWithReservations(options Options, target destination, bundle skillBundle, store *skillclaim.LockedStore, previousReceipt receipt, receiptFound bool, entry DestinationResult) (DestinationResult, error) {
	if receiptFound {
		if err := validateReceipt(previousReceipt, options.Scope, target); err != nil {
			return DestinationResult{}, err
		}
		if previousReceipt.Schema < receiptSchemaCurrent {
			if err := verifyInstalled(target.path, previousReceipt.Files); err != nil {
				return DestinationResult{}, err
			}
		} else if hasForeignCollision(target.path, bundle.files) {
			return DestinationResult{}, errors.New("refusing reinstall because a backup-only destination is no longer empty")
		}
	}
	operationID, err := newClaimOperationID()
	if err != nil {
		return DestinationResult{}, err
	}
	runtimes := append([]Runtime(nil), target.runtimes...)
	if receiptFound && previousReceipt.Installation != "absent" {
		runtimes = unionRuntimes(previousReceipt.Runtimes, runtimes)
	}
	claims, err := projectedClaims(target, options.Scope, runtimes, bundle, skillclaim.StateReserved, operationID)
	if err != nil {
		return DestinationResult{}, err
	}
	if err := ensureClaimsAbsent(store, claims); err != nil {
		return DestinationResult{}, err
	}
	if !receiptFound {
		if err := ensureNoCollision(target.path, bundle.files); err != nil && !options.Takeover {
			return DestinationResult{}, err
		}
		if options.Takeover {
			if err := validateTakeover(target, bundle); err != nil {
				return DestinationResult{}, err
			}
		}
	}
	reserved, err := reserveClaims(store, claims)
	if err != nil {
		_ = rollbackClaimsToAbsent(store, reserved)
		return DestinationResult{}, err
	}
	takeovers := append([]takeoverBackup(nil), previousReceipt.Takeovers...)
	rollbackTakeover := func() error { return nil }
	finalizeTakeover := func() error { return nil }
	cleanupTakeover := func() error { return nil }
	takeoverRecovery := ""
	if !receiptFound && options.Takeover {
		takeovers, rollbackTakeover, finalizeTakeover, cleanupTakeover, takeoverRecovery, err = createTakeoverBackups(options.StateRoot, target, bundle)
		if err != nil {
			_ = rollbackClaimsToAbsent(store, reserved)
			return DestinationResult{}, err
		}
	}
	replaceExisting := receiptFound && previousReceipt.Installation != "absent"
	undo, finalize, err := replaceBundle(target.path, bundle, replaceExisting, ownedSkillNames(previousReceipt.Files))
	if err != nil {
		originalErr := rollbackTakeover()
		cleanupErr := error(nil)
		if originalErr == nil {
			cleanupErr = cleanupTakeover()
		}
		claimErr := rollbackClaimsToAbsent(store, reserved)
		return DestinationResult{}, fmt.Errorf("install captured takeover: %w (original rollback: %v; backup cleanup: %v; claim rollback: %v; recovery artifact: %s)", err, originalErr, cleanupErr, claimErr, takeoverRecovery)
	}
	newReceipt := receipt{
		Schema: receiptSchemaCurrent, Installation: "active", Format: targetReceiptFormat(target),
		Scope: options.Scope, Destination: target.path, Runtimes: runtimes, Version: options.Version,
		BundleSHA: sourceBundleSHA(bundle.files), Files: bundle.files, Takeovers: takeovers,
	}
	receiptFile := receiptPath(options.StateRoot, target.path)
	if err := writeReceipt(receiptFile, newReceipt); err != nil {
		rollbackErr := undo()
		originalErr := error(nil)
		cleanupErr := error(nil)
		if rollbackErr == nil {
			originalErr = rollbackTakeover()
		}
		if rollbackErr == nil && originalErr == nil {
			cleanupErr = cleanupTakeover()
		}
		claimErr := rollbackClaimsToAbsent(store, reserved)
		return DestinationResult{}, fmt.Errorf("write receipt: %w (file rollback: %v; original rollback: %v; backup cleanup: %v; claim rollback: %v; recovery artifact: %s)", err, rollbackErr, originalErr, cleanupErr, claimErr, takeoverRecovery)
	}
	if _, err := activateReservedClaims(store, reserved); err != nil {
		rollbackErr := undo()
		originalErr := error(nil)
		cleanupErr := error(nil)
		if rollbackErr == nil {
			originalErr = rollbackTakeover()
		}
		if rollbackErr == nil && originalErr == nil {
			cleanupErr = cleanupTakeover()
		}
		receiptErr := restoreReceipt(receiptFile, previousReceipt, receiptFound)
		claimErr := rollbackClaimsToAbsent(store, claims)
		return DestinationResult{}, fmt.Errorf("activate DVA claims: %w (file rollback: %v; original rollback: %v; receipt rollback: %v; backup cleanup: %v; claim rollback: %v; recovery artifact: %s)", err, rollbackErr, originalErr, receiptErr, cleanupErr, claimErr, takeoverRecovery)
	}
	if err := finalize(); err != nil {
		entry.Detail = fmt.Sprintf("installed; cleanup retained a temporary artifact: %v", err)
	}
	if err := finalizeTakeover(); err != nil {
		entry.Detail = fmt.Sprintf("installed; cleanup retained captured originals at %s: %v", takeoverRecovery, err)
	}
	entry.Status = "installed"
	setAllRuntimeStatuses(&entry, "installed")
	if len(takeovers) > 0 {
		entry.BackupStatus, entry.TakeoverBackup = verifyTakeoverBackups(options.StateRoot, newReceipt)
	}
	return entry, nil
}

func updateClaimedInstall(options Options, target destination, bundle skillBundle, store *skillclaim.LockedStore, record receipt, entry DestinationResult) (DestinationResult, error) {
	if err := validateReceipt(record, options.Scope, target); err != nil {
		return DestinationResult{}, err
	}
	if record.Installation != "active" {
		return DestinationResult{}, errors.New("recovery-required: claimed receipt is not active")
	}
	if err := verifyInstalled(target.path, record.Files); err != nil {
		return DestinationResult{}, err
	}
	operationID, err := newClaimOperationID()
	if err != nil {
		return DestinationResult{}, err
	}
	oldProjection, err := projectedClaims(target, options.Scope, record.Runtimes, skillBundle{files: record.Files}, skillclaim.StateActive, operationID)
	if err != nil {
		return DestinationResult{}, err
	}
	current, err := readLockedClaims(store, oldProjection)
	if err != nil {
		return DestinationResult{}, err
	}
	if err := verifyActiveClaims(current, oldProjection); err != nil {
		return DestinationResult{}, err
	}
	runtimes := unionRuntimes(record.Runtimes, target.runtimes)
	desired, err := projectedClaims(target, options.Scope, runtimes, bundle, skillclaim.StateActive, operationID)
	if err != nil {
		return DestinationResult{}, err
	}
	matched, added, err := additiveClaimUpdate(current, desired)
	if err != nil {
		return DestinationResult{}, err
	}
	if err := ensureClaimsAbsent(store, added); err != nil {
		return DestinationResult{}, err
	}
	if err := ensureClaimDestinationsAbsent(added); err != nil {
		return DestinationResult{}, err
	}
	reserved, err := reserveClaims(store, added)
	if err != nil {
		rollbackErr := rollbackClaimsToAbsent(store, reserved)
		return DestinationResult{}, fmt.Errorf("reserve added claims: %w (added claim rollback: %v)", err, rollbackErr)
	}
	updating, err := transitionActiveClaims(store, current, matched, skillclaim.StateUpdating, operationID)
	if err != nil {
		rollbackErr := rollbackClaimsToAbsent(store, reserved)
		return DestinationResult{}, fmt.Errorf("transition existing claims: %w (added claim rollback: %v)", err, rollbackErr)
	}
	undo, finalize := func() error { return nil }, func() error { return nil }
	if !equalFiles(record.Files, bundle.files) {
		undo, finalize, err = replaceBundle(target.path, bundle, true, ownedSkillNames(record.Files))
		if err != nil {
			claimErr := rollbackClaimsToActive(store, current)
			reserveErr := rollbackClaimsToAbsent(store, reserved)
			return DestinationResult{}, fmt.Errorf("replace managed skills after claim reservation: %w (claim rollback: %v; added claim rollback: %v)", err, claimErr, reserveErr)
		}
	}
	updated := receipt{
		Schema: receiptSchemaCurrent, Installation: "active", Format: targetReceiptFormat(target),
		Scope: options.Scope, Destination: target.path, Runtimes: runtimes, Version: options.Version,
		BundleSHA: sourceBundleSHA(bundle.files), Files: bundle.files, Takeovers: record.Takeovers,
	}
	receiptFile := receiptPath(options.StateRoot, target.path)
	if err := writeReceipt(receiptFile, updated); err != nil {
		fileErr := undo()
		claimErr := rollbackClaimsToActive(store, current)
		reserveErr := rollbackClaimsToAbsent(store, reserved)
		return DestinationResult{}, fmt.Errorf("update receipt: %w (file rollback: %v; claim rollback: %v; added claim rollback: %v)", err, fileErr, claimErr, reserveErr)
	}
	if err := activateUpdatedClaims(store, updating); err != nil {
		fileErr := undo()
		receiptErr := writeReceipt(receiptFile, record)
		claimErr := rollbackClaimsToActive(store, current)
		reserveErr := rollbackClaimsToAbsent(store, reserved)
		return DestinationResult{}, fmt.Errorf("activate updated claims: %w (file rollback: %v; receipt rollback: %v; claim rollback: %v; added claim rollback: %v)", err, fileErr, receiptErr, claimErr, reserveErr)
	}
	if _, err := activateReservedClaims(store, reserved); err != nil {
		fileErr := undo()
		receiptErr := writeReceipt(receiptFile, record)
		claimErr := rollbackClaimsToActive(store, current)
		reserveErr := rollbackClaimsToAbsent(store, reserved)
		return DestinationResult{}, fmt.Errorf("activate added claims: %w (file rollback: %v; receipt rollback: %v; claim rollback: %v; added claim rollback: %v)", err, fileErr, receiptErr, claimErr, reserveErr)
	}
	if err := finalize(); err != nil {
		return DestinationResult{}, err
	}
	entry.Status = "installed"
	setAllRuntimeStatuses(&entry, "installed")
	if len(updated.Takeovers) > 0 {
		entry.BackupStatus, entry.TakeoverBackup = verifyTakeoverBackups(options.StateRoot, updated)
	}
	return entry, nil
}

func ensureClaimDestinationsAbsent(claims []skillclaim.Claim) error {
	for _, claim := range claims {
		if _, err := os.Lstat(claim.Destination); err == nil {
			return fmt.Errorf("refusing collision at %s; newly bundled DVA skill has no matching receipt", claim.Destination)
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}

func restoreReceipt(path string, previous receipt, found bool) error {
	if found {
		return writeReceipt(path, previous)
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
