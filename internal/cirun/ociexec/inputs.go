package ociexec

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// The first executor scope is small, explicitly selected project fixtures.
const maxInputBytes = 64 << 20
const maxInputFiles = 10000

func snapshotInput(root, source, destination string) (string, error) {
	if source == "" || source == "." || filepath.IsAbs(source) || strings.ContainsAny(source, "\\\x00") || filepath.Clean(source) != source || strings.HasPrefix(source, ".."+string(os.PathSeparator)) || source == ".." {
		return "", errors.New("OCI source must be a project subdirectory")
	}
	project, err := os.OpenRoot(root)
	if err != nil {
		return "", err
	}
	defer func() { _ = project.Close() }()
	parts := strings.Split(filepath.ToSlash(source), "/")
	current := project
	for _, part := range parts {
		info, err := current.Lstat(part)
		if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return "", errors.New("OCI source must not traverse symlinks")
		}
		next, err := current.OpenRoot(part)
		if err != nil {
			return "", err
		}
		if current != project {
			_ = current.Close()
		}
		current = next
	}
	defer func() {
		if current != project {
			_ = current.Close()
		}
	}()
	dest, err := os.OpenRoot(destination)
	if err != nil {
		return "", err
	}
	defer func() { _ = dest.Close() }()
	if entries, err := readDirPage(dest, "."); err != nil || len(entries) != 0 {
		return "", errors.New("OCI snapshot destination must be a fresh empty directory")
	}
	digest := sha256.New()
	var total int64
	files := 0
	var walk func(*os.Root, *os.Root, string, int) error
	walk = func(src, out *os.Root, prefix string, depth int) error {
		if depth > 32 {
			return errors.New("OCI input exceeds 32 directories deep")
		}
		file, err := src.Open(".")
		if err != nil {
			return err
		}
		defer func() { _ = file.Close() }()
		var all []os.DirEntry
		for {
			entries, err := file.ReadDir(64)
			if err != nil && err != io.EOF {
				return err
			}
			all = append(all, entries...)
			if len(all) > maxInputFiles {
				return errors.New("OCI input exceeds 10000 entries")
			}
			if err == io.EOF {
				break
			}
		}
		slices.SortFunc(all, func(left, right os.DirEntry) int { return strings.Compare(left.Name(), right.Name()) })
		for _, entry := range all {
			files++
			if files > maxInputFiles {
				return errors.New("OCI input exceeds 10000 entries")
			}
			name := entry.Name()
			rel := name
			if prefix != "" {
				rel = prefix + "/" + name
			}
			info, err := src.Lstat(name)
			if err != nil || info.Mode()&os.ModeSymlink != 0 || (!info.IsDir() && !info.Mode().IsRegular()) {
				return fmt.Errorf("OCI input %q is not a regular file or directory", rel)
			}
			if info.IsDir() {
				if err := out.Mkdir(name, 0o755); err != nil {
					return err
				}
				child, err := src.OpenRoot(name)
				if err != nil {
					return err
				}
				target, err := out.OpenRoot(name)
				if err != nil {
					_ = child.Close()
					return err
				}
				err = walk(child, target, rel, depth+1)
				_ = child.Close()
				_ = target.Close()
				if err != nil {
					return err
				}
				writeRecord(digest, rel, 'd', 0o755, 0, nil)
				continue
			}
			if info.Size() > maxInputBytes-total {
				return errors.New("OCI input exceeds 64 MiB")
			}
			in, err := openInputFile(src, name)
			if err != nil {
				return err
			}
			opened, err := in.Stat()
			if err != nil || !opened.Mode().IsRegular() || !os.SameFile(info, opened) || opened.Size() != info.Size() {
				_ = in.Close()
				return errors.New("OCI input changed while snapshotting")
			}
			mode := os.FileMode(0o644)
			if info.Mode().Perm()&0o111 != 0 {
				mode = 0o755
			}
			of, err := out.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
			if err != nil {
				_ = in.Close()
				return err
			}
			h := sha256.New()
			n, copyErr := io.Copy(io.MultiWriter(of, h), io.LimitReader(in, maxInputBytes-total+1))
			closeErr := errors.Join(in.Close(), of.Close())
			if copyErr != nil {
				return copyErr
			}
			if closeErr != nil {
				return closeErr
			}
			total += n
			if n != info.Size() {
				return errors.New("OCI input changed while snapshotting")
			}
			writeRecord(digest, rel, 'f', mode, n, h.Sum(nil))
		}
		return nil
	}
	err = walk(current, dest, "", 0)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}

func readDirPage(root *os.Root, name string) ([]os.DirEntry, error) {
	f, err := root.Open(name)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	entries, err := f.ReadDir(64)
	if err == io.EOF {
		err = nil
	}
	return entries, err
}
func writeRecord(w io.Writer, path string, kind byte, mode os.FileMode, size int64, sum []byte) {
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], uint64(len(path)))
	_, _ = w.Write(b[:])
	_, _ = w.Write([]byte(path))
	_, _ = w.Write([]byte{kind})
	binary.BigEndian.PutUint64(b[:], uint64(mode))
	_, _ = w.Write(b[:])
	binary.BigEndian.PutUint64(b[:], uint64(size))
	_, _ = w.Write(b[:])
	binary.BigEndian.PutUint64(b[:], uint64(len(sum)))
	_, _ = w.Write(b[:])
	_, _ = w.Write(sum)
}

// removeSnapshotAt removes only the fixed input child after refusing anything
// other than the regular files/directories created by snapshotInput.
func removeSnapshotAt(parent *os.Root) error {
	before, err := parent.Lstat("input")
	if err != nil {
		return err
	}
	if !before.IsDir() || before.Mode()&os.ModeSymlink != 0 {
		return errors.New("OCI snapshot root was replaced")
	}
	input, err := parent.OpenRoot("input")
	if err != nil {
		return err
	}
	defer func() { _ = input.Close() }()
	opened, err := input.Stat(".")
	if err != nil || !os.SameFile(before, opened) {
		return errors.New("OCI snapshot root changed during cleanup")
	}
	var remove func(*os.Root) error
	remove = func(root *os.Root) error {
		file, err := root.Open(".")
		if err != nil {
			return err
		}
		defer func() { _ = file.Close() }()
		for {
			entries, err := file.ReadDir(64)
			if err != nil && err != io.EOF {
				return err
			}
			for _, entry := range entries {
				info, statErr := root.Lstat(entry.Name())
				if statErr != nil || info.Mode()&os.ModeSymlink != 0 || (!info.IsDir() && !info.Mode().IsRegular()) {
					return errors.New("OCI snapshot contains unexpected object")
				}
				if info.IsDir() {
					child, openErr := root.OpenRoot(entry.Name())
					if openErr != nil {
						return openErr
					}
					removeErr := remove(child)
					_ = child.Close()
					if removeErr != nil {
						return removeErr
					}
				}
				if err := root.Remove(entry.Name()); err != nil {
					return err
				}
			}
			if err == io.EOF {
				break
			}
		}
		return nil
	}
	if err := remove(input); err != nil {
		return err
	}
	return parent.Remove("input")
}

func persistIntent(path string, value intent) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(path+".new", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	_, writeErr := file.Write(append(data, '\n'))
	syncErr := file.Sync()
	closeErr := file.Close()
	if err := errors.Join(writeErr, syncErr, closeErr); err != nil {
		return err
	}
	if err := os.Rename(path+".new", path); err != nil {
		return err
	}
	dir, err := os.Open(filepath.Dir(path))
	if err != nil {
		return err
	}
	defer func() { _ = dir.Close() }()
	return dir.Sync()
}
