package cimetrics

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"unicode/utf8"
)

// The output renderer never stores raw JSON. It holds at most one bounded
// line and bounded, in-memory diagnostic buffers while tests are in flight.
const outputLimit = 1024 * 1024
const outputLineLimit = 64 * 1024
const outputEntryLimit = 64 * 1024
const outputPackageLimit = 4096
const outputBufferLimit = 4096
const outputEntryOverhead = 128

// Reserve for both input lines, decoded and filtered event text, and one
// buffer growth copy before allocating diagnostic buffers.
const outputReserved = 5 * outputLineLimit

type pendingOutput struct {
	data      []byte
	truncated bool
}

type outputPackage struct {
	coverage      float64
	coverageKnown bool
	cacheHint     bool
	started       bool
}

// Raw identities are scoped to in-memory association only. A tuple keeps
// embedded NULs and separately redacted unknown names from aliasing.
type outputKey struct{ pkg, test string }

type outputRenderer struct {
	mu       sync.Mutex
	parser   *Parser
	out      io.Writer
	verbose  bool
	line     []byte
	overflow bool
	used     int
	packages map[string]*outputPackage
	pending  map[outputKey]*pendingOutput
	err      error
	unsafe   bool
	closed   atomic.Bool
	degraded atomic.Bool
}

func newOutputRenderer(p *Parser, out io.Writer, verbose bool) *outputRenderer {
	return &outputRenderer{parser: p, out: out, verbose: verbose,
		line: make([]byte, 0, outputLineLimit), used: outputReserved,
		packages: make(map[string]*outputPackage), pending: make(map[outputKey]*pendingOutput)}
}

// Write consumes malformed and oversized lines without echoing their bytes.
// A real stdout write failure is returned to exec.Cmd, just as before.
func (r *outputRenderer) Write(p []byte) (int, error) {
	if r.closed.Load() {
		return len(p), nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed.Load() {
		return len(p), nil
	}
	if r.err != nil {
		return 0, r.err
	}
	for i, ch := range p {
		if r.closed.Load() {
			return len(p), nil
		}
		if ch == '\n' {
			if !r.overflow {
				r.consume(r.line)
			}
			r.line = r.line[:0]
			r.overflow = false
			if r.err != nil {
				return i + 1, r.err
			}
			continue
		}
		if r.overflow {
			continue
		}
		if len(r.line) >= outputLineLimit {
			r.invalidate()
			r.line = r.line[:0]
			r.overflow = true
			continue
		}
		r.line = append(r.line, ch)
	}
	return len(p), nil
}

type outputEvent struct {
	Action     string
	Package    string
	ImportPath string
	Test       string
	Output     string
	Elapsed    *float64
}

func (r *outputRenderer) consume(line []byte) {
	var e outputEvent
	if len(bytes.TrimSpace(line)) == 0 || json.Unmarshal(line, &e) != nil {
		r.invalidate()
		return
	}
	if e.Action == "build-output" {
		if e.ImportPath == "" || len(e.ImportPath) > 512 {
			r.invalidate()
			return
		}
		if r.unsafe {
			r.degraded.Store(true)
		} else {
			r.write(e.Output)
		}
		return
	}
	if e.Action == "build-fail" {
		if e.ImportPath == "" || len(e.ImportPath) > 512 {
			r.invalidate()
		}
		return
	}
	switch e.Action {
	case "start", "run", "pause", "cont", "output", "pass", "fail", "skip":
	default:
		r.invalidate()
		return
	}
	if e.Package == "" {
		r.invalidate()
		return
	}
	if len(e.Package) > 512 || len(e.Test) > 512 {
		r.invalidate()
		return
	}
	if e.Elapsed != nil && (math.IsNaN(*e.Elapsed) || math.IsInf(*e.Elapsed, 0) || *e.Elapsed < 0) {
		r.invalidate()
		return
	}
	pkg, test := e.Package, e.Test
	if e.Action == "start" {
		if test != "" {
			r.invalidate()
			return
		}
		r.clearPackage(pkg, true)
		p := r.packageState(pkg)
		if p != nil {
			p.started = true
			r.unsafe = false
		}
		return
	}
	if e.Action == "run" {
		if test == "" {
			r.invalidate()
			return
		}
		if r.packageState(pkg) == nil {
			return
		}
		key := outputKey{pkg: pkg, test: test}
		if _, exists := r.pending[key]; exists {
			r.release(pkg, test, false)
			r.degraded.Store(true)
		}
		if r.addPending(key) != nil {
			r.unsafe = false
		}
		return
	}
	if e.Action == "pause" || e.Action == "cont" {
		if test == "" || r.pending[outputKey{pkg: pkg, test: test}] == nil {
			r.invalidate()
		}
		return
	}
	if e.Action == "output" {
		if test != "" && r.pending[outputKey{pkg: pkg, test: test}] == nil {
			r.degraded.Store(true)
			return
		}
		if test == "" && (r.packages[pkg] == nil || !r.packages[pkg].started) {
			r.degraded.Store(true)
			return
		}
		if r.verbose {
			r.write(e.Output)
			return
		}
		if test == "" {
			p := r.packages[pkg]
			if coverage, ok := extractCoverage(e.Output); ok {
				p.coverage, p.coverageKnown = coverage, true
			}
			if strings.Contains(e.Output, "(cached)") {
				p.cacheHint = true
			}
		}
		if test == "" {
			r.buffer(pkg, test, stripPackageProtocol(e.Output, pkg))
		} else {
			r.buffer(pkg, test, stripVerboseProtocol(e.Output))
		}
		return
	}
	if e.Action != "pass" && e.Action != "fail" && e.Action != "skip" {
		return
	}
	if test != "" {
		if r.pending[outputKey{pkg: pkg, test: test}] == nil {
			r.degraded.Store(true)
			return
		}
		if e.Action == "fail" {
			_, safeTest := r.parser.SafeLabels(pkg, test)
			if !r.verbose {
				r.write("--- FAIL: " + safeTest + "\n")
			}
			r.release(pkg, test, true)
		} else {
			r.release(pkg, test, false)
		}
		return
	}
	if !r.verbose {
		if r.packages[pkg] == nil || !r.packages[pkg].started {
			r.degraded.Store(true)
		}
		if e.Action == "fail" && r.packages[pkg] != nil && r.packages[pkg].started {
			r.release(pkg, "", true)
		} else {
			r.release(pkg, "", false)
		}
		r.packageSummary(pkg, e.Action, e.Elapsed)
	}
	r.clearPackage(pkg, false)
}

func (r *outputRenderer) packageState(pkg string) *outputPackage {
	if p := r.packages[pkg]; p != nil {
		return p
	}
	if len(r.packages) >= outputPackageLimit || !r.reserve(len(pkg)+outputEntryOverhead) {
		r.degraded.Store(true)
		return nil
	}
	p := &outputPackage{}
	r.packages[pkg] = p
	return p
}

func (r *outputRenderer) reserve(n int) bool {
	if n < 0 || n > outputLimit-r.used {
		return false
	}
	r.used += n
	return true
}

func (r *outputRenderer) buffer(pkg, test, text string) {
	if text == "" {
		return
	}
	key := outputKey{pkg: pkg, test: test}
	b := r.pending[key]
	if b == nil {
		if test != "" {
			r.degraded.Store(true)
			return
		}
		b = r.addPending(key)
		if b == nil {
			return
		}
	}
	available := outputEntryLimit - len(b.data)
	if len(text) > available {
		text = safePrefix(text, available)
		b.truncated = true
		r.degraded.Store(true)
	}
	if len(text) == 0 {
		return
	}
	need := len(b.data) + len(text)
	if need > cap(b.data) {
		capacity := min(max(cap(b.data)*2, need), outputEntryLimit)
		if !r.reserve(capacity - cap(b.data)) {
			// The existing capacity can still retain a prefix without breaching
			// the global budget; no other in-flight test is evicted or printed.
			text = safePrefix(text, cap(b.data)-len(b.data))
			b.truncated = true
			r.degraded.Store(true)
		} else {
			grown := make([]byte, len(b.data), capacity)
			copy(grown, b.data)
			b.data = grown
		}
	}
	b.data = append(b.data, text...)
}

func (r *outputRenderer) addPending(key outputKey) *pendingOutput {
	if len(r.pending) >= outputBufferLimit || !r.reserve(len(key.pkg)+len(key.test)+outputEntryOverhead) {
		r.degraded.Store(true)
		return nil
	}
	b := &pendingOutput{}
	r.pending[key] = b
	return b
}

func safePrefix(s string, limit int) string {
	if limit <= 0 {
		return ""
	}
	if len(s) <= limit {
		return s
	}
	s = s[:limit]
	for !utf8.ValidString(s) && len(s) > 0 {
		s = s[:len(s)-1]
	}
	return s
}

func (r *outputRenderer) release(pkg, test string, report bool) {
	key := outputKey{pkg: pkg, test: test}
	b := r.pending[key]
	if b == nil {
		return
	}
	if report {
		r.write(string(b.data))
		if b.truncated {
			r.write("\nci-metrics: failure output truncated\n")
		}
	}
	r.used -= len(key.pkg) + len(key.test) + outputEntryOverhead + cap(b.data)
	delete(r.pending, key)
}

func (r *outputRenderer) clearPackage(pkg string, reset bool) {
	for key, b := range r.pending {
		if key.pkg == pkg {
			r.used -= len(key.pkg) + len(key.test) + outputEntryOverhead + cap(b.data)
			delete(r.pending, key)
			r.degraded.Store(true) // nonterminal test/output, never flush it
		}
	}
	if state, ok := r.packages[pkg]; ok {
		if reset && state.started {
			r.degraded.Store(true)
		}
		r.used -= len(pkg) + outputEntryOverhead
		delete(r.packages, pkg)
	}
}

// Lost or untrusted framing invalidates all in-flight associations. Do not
// release any buffered diagnostics until an explicit new start/run arrives.
func (r *outputRenderer) invalidate() {
	r.degraded.Store(true)
	for key, b := range r.pending {
		r.used -= len(key.pkg) + len(key.test) + outputEntryOverhead + cap(b.data)
		delete(r.pending, key)
	}
	for pkg := range r.packages {
		r.used -= len(pkg) + outputEntryOverhead
		delete(r.packages, pkg)
	}
	r.unsafe = true
}

func (r *outputRenderer) packageSummary(pkg, action string, elapsed *float64) {
	duration := "?"
	if elapsed != nil && *elapsed <= 24*60*60 {
		duration = fmt.Sprintf("%.3fs", *elapsed)
	}
	if elapsed != nil && *elapsed > 24*60*60 {
		r.degraded.Store(true)
	}
	label := "ok"
	switch action {
	case "fail":
		label = "FAIL"
	case "skip":
		label = "?"
	}
	if p := r.packages[pkg]; p != nil {
		if p.cacheHint {
			duration = "(cached hint)"
		}
		if p.coverageKnown {
			duration += fmt.Sprintf("\tcoverage: %.1f%% of statements", p.coverage)
		}
	}
	safePkg, _ := r.parser.SafeLabels(pkg, "")
	r.write(fmt.Sprintf("%s\t%s\t%s\n", label, safePkg, duration))
}

var coveragePattern = regexp.MustCompile(`(?:^|[\t ])coverage: ([0-9]{1,3}(?:\.[0-9]{1,3})?)% of statements(?:[\t \r\n]|$)`)

func extractCoverage(s string) (float64, bool) {
	m := coveragePattern.FindStringSubmatch(s)
	if len(m) != 2 {
		return 0, false
	}
	v, err := strconv.ParseFloat(m[1], 64)
	return v, err == nil && v >= 0 && v <= 100
}

func stripVerboseProtocol(s string) string {
	if !strings.Contains(s, "=== ") && !strings.Contains(s, "--- ") {
		return s
	}
	var out strings.Builder
	out.Grow(len(s))
	for len(s) > 0 {
		part := s
		if i := strings.IndexByte(s, '\n'); i >= 0 {
			part, s = s[:i+1], s[i+1:]
		} else {
			s = ""
		}
		if strings.HasPrefix(part, "=== RUN ") || strings.HasPrefix(part, "=== PAUSE ") || strings.HasPrefix(part, "=== CONT ") || strings.HasPrefix(part, "=== NAME ") || strings.HasPrefix(part, "--- FAIL: ") || strings.HasPrefix(part, "--- PASS: ") || strings.HasPrefix(part, "--- SKIP: ") {
			continue
		}
		out.WriteString(part)
	}
	return out.String()
}

func stripPackageProtocol(s, pkg string) string {
	if !strings.Contains(s, "PASS") && !strings.Contains(s, "FAIL") && !strings.Contains(s, "ok") && !strings.Contains(s, "?") {
		return s
	}
	var out strings.Builder
	out.Grow(len(s))
	for len(s) > 0 {
		part := s
		if i := strings.IndexByte(s, '\n'); i >= 0 {
			part, s = s[:i+1], s[i+1:]
		} else {
			s = ""
		}
		trimmed := strings.TrimSpace(part)
		if trimmed == "PASS" || trimmed == "FAIL" || packageFooter(trimmed, pkg) {
			continue
		}
		out.WriteString(part)
	}
	return out.String()
}

func packageFooter(line, pkg string) bool {
	for _, prefix := range []string{"FAIL", "ok", "?"} {
		if !strings.HasPrefix(line, prefix) {
			continue
		}
		rest := strings.TrimPrefix(line, prefix)
		if len(rest) == 0 || (rest[0] != ' ' && rest[0] != '\t') {
			continue
		}
		rest = strings.TrimLeft(rest, " \t")
		if rest == pkg || (strings.HasPrefix(rest, pkg) && len(rest) > len(pkg) && (rest[len(pkg)] == ' ' || rest[len(pkg)] == '\t')) {
			return true
		}
	}
	return false
}

func (r *outputRenderer) write(s string) {
	if s == "" || r.err != nil || r.closed.Load() {
		return
	}
	n, err := io.WriteString(r.out, s)
	if err == nil && n != len(s) {
		err = io.ErrShortWrite
	}
	if err != nil {
		r.err = err
		r.degraded.Store(true)
	}
}

// Finish never flushes unfinished tests or waits on a blocked output writer.
func (r *outputRenderer) Finish() {
	r.closed.Store(true)
	if !r.mu.TryLock() {
		r.degraded.Store(true)
		return
	}
	defer r.mu.Unlock()
	if len(r.line) != 0 || r.overflow || len(r.pending) != 0 {
		r.degraded.Store(true)
	}
	r.line = nil
	r.pending = nil
	r.packages = nil
	r.used = 0
}

func (r *outputRenderer) Degraded() bool { return r.degraded.Load() }

func explicitVerbose(argv []string) bool {
	verbose := false
	for _, a := range argv {
		switch a {
		case "-v", "-v=true", "-test.v", "-test.v=true":
			verbose = true
		case "-v=false", "-test.v=false":
			verbose = false
		}
	}
	return verbose
}
