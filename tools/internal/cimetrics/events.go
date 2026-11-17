package cimetrics

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"maps"
	"math"
	"slices"
	"strings"
	"sync"
)

type Progress struct {
	Package        string   `json:"package,omitempty"`
	Test           string   `json:"test,omitempty"`
	Action         string   `json:"action"`
	ElapsedSeconds *float64 `json:"elapsed_seconds,omitempty"`
}
type Snapshot struct {
	Status                  string         `json:"status"`
	Reasons                 []string       `json:"reasons,omitempty"`
	Counts                  map[string]int `json:"counts"`
	Completed               []Progress     `json:"completed,omitempty"`
	Running                 []Progress     `json:"running,omitempty"`
	Paused                  []Progress     `json:"paused,omitempty"`
	Incomplete              []Progress     `json:"incomplete,omitempty"`
	TimingsTruncated        bool           `json:"timings_truncated,omitempty"`
	ActiveTruncated         bool           `json:"active_truncated,omitempty"`
	RunningCount            int            `json:"running_count"`
	PausedCount             int            `json:"paused_count"`
	IncompleteCount         int            `json:"incomplete_count"`
	PendingBuilds           []string       `json:"pending_builds,omitempty"`
	PendingBuildCount       int            `json:"pending_build_count"`
	PendingPackages         []string       `json:"pending_packages,omitempty"`
	PendingPackageCount     int            `json:"pending_package_count"`
	PendingDetailsTruncated bool           `json:"pending_details_truncated,omitempty"`
	// GoTestCached reports a textual hint; false means no hint was observed.
	GoTestCached bool `json:"go_test_cache_hint_seen"`
}
type Parser struct {
	mu              sync.Mutex
	allowed         map[string]map[string]bool
	emit            func(Progress)
	key             [32]byte
	noRandom        bool
	line            []byte
	overflow        bool
	finished        bool
	observed        bool
	snap            Snapshot
	active          map[string]Progress
	pendingPackages map[string]bool
	buildPending    map[string]bool
}

func NewParser(allowed map[string]map[string]bool, emit func(Progress)) *Parser {
	p := &Parser{allowed: allowed, emit: emit, active: make(map[string]Progress), pendingPackages: make(map[string]bool), buildPending: make(map[string]bool), snap: Snapshot{Status: "complete", Counts: make(map[string]int)}}
	if _, err := rand.Read(p.key[:]); err != nil {
		p.partial("randomness unavailable")
		p.noRandom = true
	}
	return p
}
func (p *Parser) partial(reason string) {
	p.snap.Status = "partial"
	if slices.Contains(p.snap.Reasons, reason) {
		return
	}
	p.snap.Reasons = append(p.snap.Reasons, reason)
}
func (p *Parser) opaque(s string) string {
	if p.noRandom {
		return "redacted"
	}
	h := hmac.New(sha256.New, p.key[:])
	_, _ = h.Write([]byte(s))
	return "id-" + hex.EncodeToString(h.Sum(nil)[:12])
}
func (p *Parser) labels(pkg, test string) (string, string) {
	tests, known := p.allowed[pkg]
	safePkg := pkg
	if !known || len(pkg) > 160 {
		safePkg = p.opaque(pkg)
	}
	if test == "" {
		return safePkg, ""
	}
	base, suffix, hasSuffix := strings.Cut(test, "/")
	if !known || !tests[base] || len(base) > 96 {
		return safePkg, p.opaque(test)
	}
	if hasSuffix {
		return safePkg, base + "/" + p.opaque(suffix)
	}
	return safePkg, base
}

// SafeLabels shares the same per-execution privacy identities with live output.
func (p *Parser) SafeLabels(pkg, test string) (string, string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.labels(pkg, test)
}

func (p *Parser) Write(data []byte) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	n := len(data)
	if p.finished {
		p.partial("write after finish")
		return n, nil
	}
	for _, c := range data {
		if c == '\n' {
			if !p.overflow {
				p.consume(p.line)
			}
			p.line = p.line[:0]
			p.overflow = false
			continue
		}
		if p.overflow {
			continue
		}
		if len(p.line) >= 65536 {
			p.partial("line limit exceeded")
			p.line = p.line[:0]
			p.overflow = true
			continue
		}
		p.line = append(p.line, c)
	}
	return n, nil
}
func (p *Parser) consume(line []byte) {
	var e struct {
		Action     string
		Package    string
		ImportPath string
		Test       string
		Output     string
		Elapsed    *float64
	}
	if len(bytes.TrimSpace(line)) == 0 || json.Unmarshal(line, &e) != nil {
		p.partial("malformed event")
		return
	}
	if e.Action == "output" {
		if e.Package == "" {
			p.partial("missing package")
			return
		}
		p.observed = true
		pkg, _ := p.labels(e.Package, "")
		p.pendingPackage(pkg)
		if e.Test == "" && strings.Contains(e.Output, "(cached)") {
			p.snap.GoTestCached = true
		}
		return
	}
	if e.Action == "build-output" {
		if e.ImportPath == "" {
			p.partial("missing build import path")
			return
		}
		p.observed = true
		pkg, _ := p.labels(e.ImportPath, "")
		if !p.buildPending[pkg] && len(p.buildPending) >= 16384 {
			p.partial("build entry limit exceeded")
		} else {
			p.buildPending[pkg] = true
		}
		return
	}
	if e.Action == "build-fail" {
		if e.ImportPath == "" {
			p.partial("missing build import path")
			return
		}
		pkg, _ := p.labels(e.ImportPath, "")
		p.observed = true
		delete(p.buildPending, pkg)
		p.snap.Counts[e.Action]++
		if p.emit != nil {
			p.emit(Progress{Package: pkg, Action: e.Action})
		}
		return
	}
	switch e.Action {
	case "start", "run", "pause", "cont", "pass", "fail", "skip":
	default:
		p.partial("unknown action")
		return
	}
	if e.Package == "" {
		p.partial("missing package")
		return
	}
	if e.Elapsed != nil && (math.IsNaN(*e.Elapsed) || math.IsInf(*e.Elapsed, 0) || *e.Elapsed < 0) {
		p.partial("invalid elapsed")
		return
	}
	pkg, test := p.labels(e.Package, e.Test)
	if test != "" || (e.Action != "pass" && e.Action != "fail" && e.Action != "skip") {
		p.pendingPackage(pkg)
	} else {
		delete(p.pendingPackages, pkg)
		delete(p.buildPending, pkg)
	}
	event := Progress{Package: pkg, Test: test, Action: e.Action, ElapsedSeconds: e.Elapsed}
	p.observed = true
	p.snap.Counts[e.Action]++
	key := pkg + "\x00" + test
	if e.Action == "run" || e.Action == "start" || e.Action == "cont" || e.Action == "pause" {
		if _, ok := p.active[key]; !ok && len(p.active) >= 16384 {
			p.partial("entry limit exceeded")
		} else {
			p.active[key] = event
		}
	} else {
		delete(p.active, key)
		if len(p.snap.Completed) < 2048 {
			p.snap.Completed = append(p.snap.Completed, event)
		} else {
			p.snap.TimingsTruncated = true
			min := 0
			for i := 1; i < len(p.snap.Completed); i++ {
				if elapsedRank(p.snap.Completed[i]) < elapsedRank(p.snap.Completed[min]) {
					min = i
				}
			}
			if elapsedRank(event) > elapsedRank(p.snap.Completed[min]) {
				p.snap.Completed[min] = event
			}
		}
	}
	if p.emit != nil {
		p.emit(copyProgress(event))
	}
}
func (p *Parser) pendingPackage(pkg string) {
	if !p.pendingPackages[pkg] && len(p.pendingPackages) >= 16384 {
		p.partial("package entry limit exceeded")
		return
	}
	p.pendingPackages[pkg] = true
}
func copyProgress(event Progress) Progress {
	if event.ElapsedSeconds != nil {
		elapsed := *event.ElapsedSeconds
		event.ElapsedSeconds = &elapsed
	}
	return event
}
func elapsedRank(event Progress) float64 {
	if event.ElapsedSeconds == nil {
		return -1
	}
	return *event.ElapsedSeconds
}
func (p *Parser) Finish() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.finished {
		return
	}
	p.finished = true
	if !p.observed {
		p.partial("no valid events")
	}
	if len(p.line) > 0 || p.overflow {
		p.partial("truncated event")
	}
	if len(p.active) > 0 {
		p.partial("unfinished events")
	}
	if len(p.pendingPackages) > 0 {
		p.partial("missing package terminal events")
	}
	if len(p.buildPending) > 0 {
		p.partial("build output without subsequent terminal event")
	}
}
func (p *Parser) Snapshot() Snapshot {
	p.mu.Lock()
	defer p.mu.Unlock()
	s := p.snap
	s.PendingBuildCount = len(p.buildPending)
	s.PendingPackageCount = len(p.pendingPackages)
	for pkg := range p.buildPending {
		if len(s.PendingBuilds) < 128 {
			s.PendingBuilds = append(s.PendingBuilds, pkg)
		} else {
			s.PendingDetailsTruncated = true
		}
	}
	for pkg := range p.pendingPackages {
		if len(s.PendingPackages) < 128 {
			s.PendingPackages = append(s.PendingPackages, pkg)
		} else {
			s.PendingDetailsTruncated = true
		}
	}
	s.Reasons = append([]string(nil), s.Reasons...)
	s.Completed = append([]Progress(nil), s.Completed...)
	for i := range s.Completed {
		s.Completed[i] = copyProgress(s.Completed[i])
	}
	s.Counts = make(map[string]int, len(p.snap.Counts))
	maps.Copy(s.Counts, p.snap.Counts)
	activeCount := 0
	for _, v := range p.active {
		if p.finished {
			s.IncompleteCount++
		} else if v.Action == "pause" {
			s.PausedCount++
		} else {
			s.RunningCount++
		}
		if activeCount >= 2048 {
			s.ActiveTruncated = true
			continue
		}
		activeCount++
		if p.finished {
			s.Incomplete = append(s.Incomplete, v)
		} else if v.Action == "pause" {
			s.Paused = append(s.Paused, v)
		} else {
			s.Running = append(s.Running, v)
		}
	}
	return s
}
