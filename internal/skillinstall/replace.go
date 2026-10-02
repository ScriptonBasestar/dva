package skillinstall

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	bundled "github.com/ScriptonBasestar/dva/skills"
)

func replaceSkillDirectories(destination string, files []fileHash, replaceExisting bool) (func() error, func() error, error) {
	return replaceSkillDirectoriesWithRename(destination, files, replaceExisting, ownedSkillNames(files), os.Rename)
}

func replaceBundle(destination string, bundle skillBundle, replaceExisting bool, owned map[string]bool) (func() error, func() error, error) {
	if len(bundle.files) > 0 && !strings.Contains(bundle.files[0].Path, "/") {
		return replaceFlatFiles(destination, bundle, replaceExisting, owned)
	}
	return replaceSkillDirectoriesWithRename(destination, bundle.files, replaceExisting, owned, os.Rename)
}

func replaceFlatFiles(destination string, bundle skillBundle, replaceExisting bool, owned map[string]bool) (func() error, func() error, error) {
	stage, err := os.MkdirTemp(destination, ".dva-skill-stage-")
	if err != nil {
		return nil, nil, err
	}
	for _, file := range bundle.files {
		contents := bundle.contents[file.Path]
		if err := writeBytesSynced(filepath.Join(stage, filepath.FromSlash(file.Path)), contents, 0o644); err != nil {
			_ = os.RemoveAll(stage)
			return nil, nil, err
		}
	}
	type move struct{ final, backup string }
	moves := make([]move, 0, len(bundle.files))
	rollback := func() error {
		var rollbackErr error
		for index := range slices.Backward(moves) {
			if err := os.Remove(moves[index].final); err != nil && !errors.Is(err, os.ErrNotExist) && rollbackErr == nil {
				rollbackErr = err
			}
			if moves[index].backup != "" {
				if err := os.Rename(moves[index].backup, moves[index].final); err != nil && rollbackErr == nil {
					rollbackErr = err
				}
			}
		}
		if rollbackErr == nil {
			if err := os.RemoveAll(stage); err != nil {
				rollbackErr = err
			}
		}
		if err := syncDirectory(destination); err != nil && rollbackErr == nil {
			rollbackErr = err
		}
		return rollbackErr
	}
	fail := func(cause error) (func() error, func() error, error) {
		if rollbackErr := rollback(); rollbackErr != nil {
			return nil, nil, fmt.Errorf("%w (rollback also failed: %v; recovery stage: %s)", cause, rollbackErr, stage)
		}
		return nil, nil, cause
	}
	for _, file := range bundle.files {
		final := filepath.Join(destination, filepath.FromSlash(file.Path))
		backup := filepath.Join(stage, filepath.FromSlash(file.Path)+".backup")
		if _, err := os.Lstat(final); err == nil {
			if !replaceExisting || !owned[file.Path] {
				return fail(fmt.Errorf("refusing collision at %s; no DVA receipt exists", final))
			}
			if err := os.Rename(final, backup); err != nil {
				return fail(err)
			}
			moves = append(moves, move{final: final, backup: backup})
		} else if !errors.Is(err, os.ErrNotExist) {
			return fail(err)
		} else {
			moves = append(moves, move{final: final})
		}
		if err := os.Rename(filepath.Join(stage, filepath.FromSlash(file.Path)), final); err != nil {
			return fail(err)
		}
	}
	if err := syncDirectory(destination); err != nil {
		return fail(err)
	}
	return rollback, func() error { return os.RemoveAll(stage) }, nil
}

func replaceSkillDirectoriesWithRename(destination string, files []fileHash, replaceExisting bool, owned map[string]bool, rename func(string, string) error) (func() error, func() error, error) {
	stage, err := os.MkdirTemp(destination, ".dva-skill-stage-")
	if err != nil {
		return nil, nil, err
	}
	for _, file := range files {
		if err := writeEmbedded(filepath.Join(stage, filepath.FromSlash(file.Path)), file.Path); err != nil {
			_ = os.RemoveAll(stage)
			return nil, nil, err
		}
	}
	type move struct{ final, backup string }
	names := skillNames(skillBundle{files: files})
	moves := make([]move, 0, len(names))
	rollback := func() error {
		var rollbackErr error
		for index := range slices.Backward(moves) {
			if err := os.RemoveAll(moves[index].final); err != nil && rollbackErr == nil {
				rollbackErr = err
			}
			if moves[index].backup != "" {
				if err := os.Rename(moves[index].backup, moves[index].final); err != nil && rollbackErr == nil {
					rollbackErr = err
				}
			}
		}
		if rollbackErr == nil {
			if err := os.RemoveAll(stage); err != nil {
				rollbackErr = err
			}
		}
		if err := syncDirectory(destination); err != nil && rollbackErr == nil {
			rollbackErr = err
		}
		return rollbackErr
	}
	fail := func(cause error) (func() error, func() error, error) {
		if rollbackErr := rollback(); rollbackErr != nil {
			return nil, nil, fmt.Errorf("%w (rollback also failed: %v; recovery stage: %s)", cause, rollbackErr, stage)
		}
		return nil, nil, cause
	}
	for _, name := range names {
		final := filepath.Join(destination, name)
		backup := filepath.Join(stage, name+".backup")
		if _, err := os.Lstat(final); err == nil {
			if !replaceExisting || !owned[name] {
				return fail(fmt.Errorf("refusing collision at %s; no DVA receipt exists", final))
			}
			if err := rename(final, backup); err != nil {
				return fail(err)
			}
			moves = append(moves, move{final: final, backup: backup})
		} else if !errors.Is(err, os.ErrNotExist) {
			return fail(err)
		} else {
			moves = append(moves, move{final: final})
		}
		if err := rename(filepath.Join(stage, name), final); err != nil {
			return fail(err)
		}
	}
	if err := syncDirectory(destination); err != nil {
		return fail(err)
	}
	finalize := func() error { return os.RemoveAll(stage) }
	return rollback, finalize, nil
}

func writeEmbedded(destination, embeddedPath string) error {
	contents, err := bundled.Files.ReadFile(embeddedPath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}
	return writeBytesSynced(destination, contents, 0o644)
}

func writeBytesSynced(destination string, contents []byte, mode fs.FileMode) error {
	file, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return err
	}
	_, writeErr := file.Write(contents)
	syncErr := file.Sync()
	closeErr := file.Close()
	for _, candidate := range []error{writeErr, syncErr, closeErr} {
		if candidate != nil {
			return candidate
		}
	}
	return nil
}
func syncDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	syncErr := directory.Sync()
	closeErr := directory.Close()
	if syncErr != nil {
		return syncErr
	}
	return closeErr
}

func mkdirAllSynced(path string, mode fs.FileMode) error {
	path = filepath.Clean(path)
	var missing []string
	current := path
	for {
		info, err := os.Lstat(current)
		if err == nil {
			if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
				return fmt.Errorf("durable directory ancestor %s is not a regular directory", current)
			}
			break
		}
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		parent := filepath.Dir(current)
		if parent == current {
			return err
		}
		missing = append(missing, current)
		current = parent
	}
	for _, directory := range slices.Backward(missing) {
		if err := os.Mkdir(directory, mode); err != nil {
			return err
		}
		if err := syncDirectory(filepath.Dir(directory)); err != nil {
			return err
		}
	}
	return nil
}
