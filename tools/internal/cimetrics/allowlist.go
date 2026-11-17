package cimetrics

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const module = "github.com/ScriptonBasestar/dva"

// LoadAllowed scans only the project's source roots; symbolic links are never followed.
func LoadAllowed(root string) (map[string]map[string]bool, error) {
	f, err := os.Open(filepath.Join(root, "go.mod"))
	if err != nil {
		return nil, err
	}
	s := bufio.NewScanner(f)
	valid := s.Scan() && strings.TrimSpace(s.Text()) == "module "+module
	err = s.Err()
	_ = f.Close()
	if err != nil {
		return nil, err
	}
	if !valid {
		return nil, fmt.Errorf("unexpected module")
	}
	allowed := make(map[string]map[string]bool)
	queue := []string{"cmd", "internal", "tools"}
	entries, total := 0, int64(0)
	for len(queue) > 0 {
		rel := queue[0]
		queue = queue[1:]
		dir := filepath.Join(root, rel)
		info, e := os.Lstat(dir)
		if e != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("allowlist directory unavailable")
		}
		f, e := os.Open(dir)
		if e != nil {
			return nil, fmt.Errorf("allowlist directory unavailable")
		}
		pkg := module + "/" + filepath.ToSlash(rel)
		allowed[pkg] = make(map[string]bool)
		for {
			batch, readErr := f.ReadDir(128)
			for _, d := range batch {
				entries++
				if entries > 16384 {
					_ = f.Close()
					return nil, fmt.Errorf("allowlist entry limit exceeded")
				}
				name := d.Name()
				if d.Type()&os.ModeSymlink != 0 {
					continue
				}
				if d.IsDir() {
					if name == "testdata" || name == "tmp" || name == "vendor" || strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") {
						continue
					}
					queue = append(queue, filepath.Join(rel, name))
					continue
				}
				if !strings.HasSuffix(name, "_test.go") || !d.Type().IsRegular() {
					continue
				}
				source, e := os.Open(filepath.Join(dir, name))
				if e != nil {
					_ = f.Close()
					return nil, fmt.Errorf("allowlist source unavailable")
				}
				data, e := io.ReadAll(io.LimitReader(source, (1<<20)+1))
				_ = source.Close()
				if e != nil {
					_ = f.Close()
					return nil, fmt.Errorf("allowlist source unavailable")
				}
				total += int64(len(data))
				if len(data) > 1<<20 || total > 32<<20 {
					_ = f.Close()
					return nil, fmt.Errorf("allowlist source limit exceeded")
				}
				file, e := parser.ParseFile(token.NewFileSet(), "", data, 0)
				if e != nil {
					_ = f.Close()
					return nil, fmt.Errorf("allowlist source invalid")
				}
				for _, decl := range file.Decls {
					fn, ok := decl.(*ast.FuncDecl)
					if !ok || fn.Recv != nil {
						continue
					}
					for _, prefix := range []string{"Test", "Benchmark", "Fuzz"} {
						if strings.HasPrefix(fn.Name.Name, prefix) && len(fn.Name.Name) > len(prefix) && len(fn.Name.Name) <= 96 {
							allowed[pkg][fn.Name.Name] = true
						}
					}
				}
			}
			if readErr == io.EOF {
				break
			}
			if readErr != nil {
				_ = f.Close()
				return nil, fmt.Errorf("allowlist directory unavailable")
			}
		}
		if e = f.Close(); e != nil {
			return nil, fmt.Errorf("allowlist directory unavailable")
		}
	}
	return allowed, nil
}
