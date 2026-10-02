package skillinstall

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	bundled "github.com/ScriptonBasestar/dva/skills"
)

func bundledBundle() (skillBundle, error) {
	bundle := skillBundle{contents: make(map[string][]byte)}
	for _, name := range bundled.Names {
		err := fs.WalkDir(bundled.Files, name, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				return nil
			}
			contents, err := bundled.Files.ReadFile(path)
			if err != nil {
				return err
			}
			digest := sha256.Sum256(contents)
			path = filepath.ToSlash(path)
			bundle.files = append(bundle.files, fileHash{Path: path, SHA: hex.EncodeToString(digest[:])})
			bundle.contents[path] = contents
			return nil
		})
		if err != nil {
			return skillBundle{}, err
		}
	}
	sort.Slice(bundle.files, func(i, j int) bool { return bundle.files[i].Path < bundle.files[j].Path })
	return bundle, nil
}

func bundledFiles() ([]fileHash, error) {
	bundle, err := bundledBundle()
	return bundle.files, err
}

func bundleFor(target destination) (skillBundle, error) {
	if hasRuntime(target.runtimes, RuntimeAgentMesh) {
		return agentMeshBundle()
	}
	return bundledBundle()
}

func hasRuntime(runtimes []Runtime, wanted Runtime) bool {
	return slices.Contains(runtimes, wanted)
}

func ensureDestination(path string) error {
	if info, err := os.Lstat(path); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refusing symlink skill destination %s", path)
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return os.MkdirAll(path, 0o755)
}

func ensureNoCollision(destination string, files []fileHash) error {
	if info, err := os.Lstat(destination); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refusing symlink skill destination %s", destination)
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	for _, path := range collisionPaths(destination, files) {
		if _, err := os.Lstat(path); err == nil {
			return fmt.Errorf("refusing collision at %s; no DVA receipt exists", path)
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}

func hasForeignCollision(destination string, files []fileHash) bool {
	for _, path := range collisionPaths(destination, files) {
		if _, err := os.Lstat(path); err == nil {
			return true
		}
	}
	return false
}

func skillNames(bundle skillBundle) []string {
	if len(bundle.files) == 0 {
		return nil
	}
	flat := !strings.Contains(bundle.files[0].Path, "/")
	present := make(map[string]bool, len(bundle.files))
	for _, file := range bundle.files {
		name := file.Path
		if flat {
			name = strings.TrimSuffix(name, ".md")
		} else if first, _, found := strings.Cut(name, "/"); found {
			name = first
		}
		present[name] = true
	}
	names := make([]string, 0, len(present))
	for _, name := range bundled.Names {
		if present[name] {
			if flat {
				names = append(names, name+".md")
			} else {
				names = append(names, name)
			}
		}
	}
	return names
}

func ownedSkillNames(files []fileHash) map[string]bool {
	owned := make(map[string]bool)
	for _, name := range skillNames(skillBundle{files: files}) {
		owned[name] = true
	}
	return owned
}
