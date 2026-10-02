package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	bundled "github.com/ScriptonBasestar/dva/skills"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"
)

func gitStatus(root string) (string, error) {
	return commandOutput(nil, "git", "-C", root, "status", "--porcelain=v1", "--untracked-files=all")
}

func snapshotGitTreeState(root string) (gitTreeState, error) {
	status, err := gitStatus(root)
	if err != nil {
		return gitTreeState{}, err
	}
	worktreeDiffSHA, err := commandOutputSHA256("", "git", "-C", root, "diff", "--no-ext-diff", "--no-textconv", "--binary", "HEAD", "--")
	if err != nil {
		return gitTreeState{}, fmt.Errorf("snapshot worktree diff: %w", err)
	}
	indexDiffSHA, err := commandOutputSHA256("", "git", "-C", root, "diff", "--cached", "--no-ext-diff", "--no-textconv", "--binary", "HEAD", "--")
	if err != nil {
		return gitTreeState{}, fmt.Errorf("snapshot index diff: %w", err)
	}
	untrackedOutput, err := commandOutput(nil, "git", "-C", root, "ls-files", "--others", "--exclude-standard", "-z")
	if err != nil {
		return gitTreeState{}, fmt.Errorf("list untracked files: %w", err)
	}
	untracked, err := snapshotListedFiles(root, strings.Split(strings.TrimSuffix(untrackedOutput, "\x00"), "\x00"))
	if err != nil {
		return gitTreeState{}, fmt.Errorf("snapshot untracked files: %w", err)
	}
	return gitTreeState{
		Status:           status,
		WorktreeDiffSHA:  worktreeDiffSHA,
		IndexDiffSHA:     indexDiffSHA,
		UntrackedEntries: untracked,
	}, nil
}

func snapshotListedFiles(root string, paths []string) ([]treeEntry, error) {
	entries := make([]treeEntry, 0, len(paths))
	for _, relative := range paths {
		if relative == "" {
			continue
		}
		path := filepath.Join(root, filepath.FromSlash(relative))
		info, err := os.Lstat(path)
		if err != nil {
			return nil, err
		}
		entry := treeEntry{Path: filepath.ToSlash(relative), Mode: info.Mode(), ModTime: info.ModTime()}
		switch {
		case info.Mode()&os.ModeSymlink != 0:
			target, err := os.Readlink(path)
			if err != nil {
				return nil, err
			}
			entry.Type, entry.LinkTarget = "symlink", target
		case info.Mode().IsRegular():
			digest, err := fileSHA256(path)
			if err != nil {
				return nil, err
			}
			entry.Type, entry.ContentSHA = "file", digest
		default:
			entry.Type = "other"
		}
		entries = append(entries, entry)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
	return entries, nil
}

func commandOutputSHA256(directory, command string, args ...string) (string, error) {
	cmd := exec.Command(command, args...)
	cmd.Dir = directory
	hash := sha256.New()
	stderr := limitedBuffer{limit: commandStderrLimit}
	cmd.Stdout = hash
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%s: %w\n%s", strings.Join(append([]string{command}, args...), " "), err, stderr.String())
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func formatGitTreeState(state gitTreeState) string {
	return fmt.Sprintf("status:\n%sworktree_diff_sha256=%s\nindex_diff_sha256=%s\nuntracked:\n%s", state.Status, state.WorktreeDiffSHA, state.IndexDiffSHA, formatSnapshot(state.UntrackedEntries))
}

func snapshotRuntimePaths(root string) ([]treeEntry, error) {
	var snapshot []treeEntry
	seen := make(map[string]bool)
	for _, relative := range []string{".agent-mesh/skills/dva", ".agents/skills", ".claude/skills", ".grok/skills", ".opencode/skills"} {
		if err := rejectSymlinkComponents(root, relative); err != nil {
			return nil, err
		}
		components := strings.Split(filepath.ToSlash(relative), "/")
		for index := 1; index < len(components); index++ {
			ancestor := strings.Join(components[:index], "/")
			if seen[ancestor] {
				continue
			}
			seen[ancestor] = true
			entry, err := snapshotSinglePath(root, ancestor)
			if err != nil {
				return nil, err
			}
			snapshot = append(snapshot, entry)
		}
		path := filepath.Join(root, relative)
		info, err := os.Lstat(path)
		if errors.Is(err, os.ErrNotExist) {
			snapshot = append(snapshot, treeEntry{Path: filepath.ToSlash(relative), Type: "missing"})
			continue
		}
		if err != nil {
			return nil, err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("refusing symlink runtime path %s", path)
		}
		err = filepath.WalkDir(path, func(current string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			info, err := entry.Info()
			if err != nil {
				return err
			}
			relativePath, err := filepath.Rel(root, current)
			if err != nil {
				return err
			}
			record := treeEntry{Path: filepath.ToSlash(relativePath), Mode: info.Mode(), ModTime: info.ModTime()}
			switch {
			case info.Mode()&os.ModeSymlink != 0:
				relativeToRuntime, err := filepath.Rel(path, current)
				if err != nil {
					return err
				}
				managed, _, _ := strings.Cut(filepath.ToSlash(relativeToRuntime), "/")
				if slices.Contains(bundled.Names, strings.TrimSuffix(managed, ".md")) {
					return fmt.Errorf("refusing symlink managed skill target %s", current)
				}
				target, err := os.Readlink(current)
				if err != nil {
					return err
				}
				record.Type, record.LinkTarget = "symlink", target
			case info.IsDir():
				record.Type = "directory"
			case info.Mode().IsRegular():
				digest, err := fileSHA256(current)
				if err != nil {
					return err
				}
				record.Type, record.ContentSHA = "file", digest
			default:
				record.Type = "other"
			}
			snapshot = append(snapshot, record)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Slice(snapshot, func(i, j int) bool { return snapshot[i].Path < snapshot[j].Path })
	return snapshot, nil
}

func snapshotSinglePath(root, relative string) (treeEntry, error) {
	path := filepath.Join(root, filepath.FromSlash(relative))
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return treeEntry{Path: filepath.ToSlash(relative), Type: "missing"}, nil
	}
	if err != nil {
		return treeEntry{}, err
	}
	entry := treeEntry{Path: filepath.ToSlash(relative), Mode: info.Mode(), ModTime: info.ModTime()}
	switch {
	case info.Mode()&os.ModeSymlink != 0:
		target, err := os.Readlink(path)
		if err != nil {
			return treeEntry{}, err
		}
		entry.Type, entry.LinkTarget = "symlink", target
	case info.IsDir():
		entry.Type = "directory"
	case info.Mode().IsRegular():
		digest, err := fileSHA256(path)
		if err != nil {
			return treeEntry{}, err
		}
		entry.Type, entry.ContentSHA = "file", digest
	default:
		entry.Type = "other"
	}
	return entry, nil
}

func rejectSymlinkComponents(root, relative string) error {
	current := root
	for component := range strings.SplitSeq(filepath.FromSlash(relative), string(filepath.Separator)) {
		current = filepath.Join(current, component)
		info, err := os.Lstat(current)
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("refusing symlink runtime path component %s", current)
		}
	}
	return nil
}

func formatSnapshot(snapshot []treeEntry) string {
	var lines []string
	for _, entry := range snapshot {
		line := fmt.Sprintf("%s type=%s mode=%#o modtime=%s", entry.Path, entry.Type, entry.Mode, entry.ModTime.UTC().Format(time.RFC3339Nano))
		if entry.ContentSHA != "" {
			line += " sha256=" + entry.ContentSHA
		}
		if entry.LinkTarget != "" {
			line += " target=" + entry.LinkTarget
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n") + "\n"
}
