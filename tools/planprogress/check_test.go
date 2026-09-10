package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckPlanDefectKinds(t *testing.T) {
	baseIndex := taskIndex{
		"TASK-1": {paths: []string{"done/1-a.md"}, closed: true},
		"TASK-2": {paths: []string{"todo/2-b.md"}, closed: false},
		"TASK-3": {paths: []string{"_archive/3-c.md"}, closed: true},
		"TASK-4": {paths: []string{"todo/4-d.md"}, closed: false},
	}

	tests := []struct {
		name    string
		p       plan
		idx     taskIndex
		wantAny []string // substrings every one of these must appear in some defect
		wantN   int      // expected number of defects
	}{
		{
			name: "clean plan has no defects",
			p: plan{
				id:             "PLAN-CLEAN",
				children:       []string{"TASK-1", "TASK-2", "TASK-3"},
				totalTasks:     3,
				hasTotalTasks:  true,
				completedTasks: 2,
				hasCompleted:   true,
				progress:       66,
				hasProgress:    true,
			},
			idx:   baseIndex,
			wantN: 0,
		},
		{
			name: "total-tasks disagrees with len(children)",
			p: plan{
				id:             "PLAN-A",
				children:       []string{"TASK-1", "TASK-2", "TASK-3"},
				totalTasks:     4, // wrong: len(children) is 3
				hasTotalTasks:  true,
				completedTasks: 2,
				hasCompleted:   true,
				progress:       50, // consistent with the (wrong) declared total-tasks: 2*100/4
				hasProgress:    true,
			},
			idx:     baseIndex,
			wantAny: []string{"total-tasks=4, want 3"},
			wantN:   1,
		},
		{
			name: "completed-tasks disagrees with measured closed children",
			p: plan{
				id:             "PLAN-B",
				children:       []string{"TASK-1", "TASK-2", "TASK-3"},
				totalTasks:     3,
				hasTotalTasks:  true,
				completedTasks: 3, // wrong: only TASK-1 and TASK-3 are closed
				hasCompleted:   true,
				progress:       100,
				hasProgress:    true,
			},
			idx:     baseIndex,
			wantAny: []string{"completed-tasks=3, want 2"},
			// progress is checked against the *declared* completed-tasks/total-tasks
			// arithmetic (3*100/3 == 100 == declared progress), so this case
			// surfaces only the completed-tasks vs. measured-reality defect.
			wantN: 1,
		},
		{
			name: "progress disagrees with truncated completed/total*100",
			p: plan{
				id:             "PLAN-C",
				children:       []string{"TASK-1", "TASK-2", "TASK-3"},
				totalTasks:     3,
				hasTotalTasks:  true,
				completedTasks: 2,
				hasCompleted:   true,
				progress:       88, // wrong: truncated 2*100/3 is 66
				hasProgress:    true,
			},
			idx:     baseIndex,
			wantAny: []string{"progress=88, want 66"},
			wantN:   1,
		},
		{
			name: "a child in children has no card file anywhere",
			p: plan{
				id:             "PLAN-D",
				children:       []string{"TASK-1", "TASK-999"},
				totalTasks:     2,
				hasTotalTasks:  true,
				completedTasks: 1,
				hasCompleted:   true,
				progress:       50,
				hasProgress:    true,
			},
			idx:     baseIndex,
			wantAny: []string{"children lists TASK-999, but no task card exists"},
			wantN:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := checkPlan(tt.p, tt.idx)
			if len(got) != tt.wantN {
				t.Fatalf("checkPlan() = %v (%d defects), want %d", got, len(got), tt.wantN)
			}
			for _, want := range tt.wantAny {
				found := false
				for _, d := range got {
					if strings.Contains(d, want) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("checkPlan() = %v, want a defect containing %q", got, want)
				}
			}
		})
	}
}

func TestParsePlanFrontmatter(t *testing.T) {
	data := `---
id: PLAN-006
title: "Work the devbox dogfood follow-up queue in dependency order"
type: plan
scope: "TASK-311..323 from the 2026-09-05 mydevbox migration"
progress: 37
total-tasks: 19
completed-tasks: 7
children: [TASK-328, TASK-329, TASK-312]
target-date: "2026-10-31"
created: 2026-09-05
---

## Goal

body text is ignored, including a fake field: total-tasks: 999
`
	p, err := parsePlan(data)
	if err != nil {
		t.Fatalf("parsePlan() error = %v", err)
	}
	if p.id != "PLAN-006" {
		t.Errorf("id = %q, want PLAN-006", p.id)
	}
	if !p.hasTotalTasks || p.totalTasks != 19 {
		t.Errorf("totalTasks = %d, %v; want 19, true", p.totalTasks, p.hasTotalTasks)
	}
	if !p.hasCompleted || p.completedTasks != 7 {
		t.Errorf("completedTasks = %d, %v; want 7, true", p.completedTasks, p.hasCompleted)
	}
	if !p.hasProgress || p.progress != 37 {
		t.Errorf("progress = %d, %v; want 37, true", p.progress, p.hasProgress)
	}
	wantChildren := []string{"TASK-328", "TASK-329", "TASK-312"}
	if len(p.children) != len(wantChildren) {
		t.Fatalf("children = %v, want %v", p.children, wantChildren)
	}
	for i, want := range wantChildren {
		if p.children[i] != want {
			t.Errorf("children[%d] = %q, want %q", i, p.children[i], want)
		}
	}
}

func TestParsePlanMissingFrontmatterFence(t *testing.T) {
	if _, err := parsePlan("no frontmatter here\n"); err == nil {
		t.Fatal("expected an error for a document with no opening fence")
	}
}

func TestParsePlanUnterminatedFrontmatter(t *testing.T) {
	if _, err := parsePlan("---\nid: PLAN-X\n"); err == nil {
		t.Fatal("expected an error for unterminated frontmatter")
	}
}

func TestParseChildrenNormalizesLeadingZeros(t *testing.T) {
	got, err := parseChildren("[TASK-056, TASK-244]")
	if err != nil {
		t.Fatalf("parseChildren() error = %v", err)
	}
	want := []string{"TASK-56", "TASK-244"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("parseChildren() = %v, want %v", got, want)
	}
}

func TestParseChildrenEmptyList(t *testing.T) {
	got, err := parseChildren("[]")
	if err != nil {
		t.Fatalf("parseChildren() error = %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("parseChildren() = %v, want empty", got)
	}
}

func TestZoneFromPath(t *testing.T) {
	tests := []struct {
		path string
		want zone
	}{
		{"todo/249-x.md", zoneOpen},
		{"doing/249-x.md", zoneOpen},
		{"done/248-x.md", zoneClosed},
		{"_archive/251-x.md", zoneClosed},
		{"_archive/done/244-x.md", zoneClosed},
		{"issue/1-x.md", zoneOther},
	}
	for _, tt := range tests {
		if got := zoneFromPath(tt.path); got != tt.want {
			t.Errorf("zoneFromPath(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestBuildTaskIndexWalksNestedArchiveAndSkipsPlanDir(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"tasks/todo/249-open.md":                   "open card",
		"tasks/done/248-closed.md":                 "closed card",
		"tasks/_archive/251-flat.md":               "archived directly under _archive/, not nested",
		"tasks/_archive/done/244-nested.md":        "archived under _archive/done/",
		"tasks/plan/006-should-be-ignored.md":      "plan cards are not task cards",
		"tasks/_archive/plan/003-archived-plan.md": "an archived plan is still a plan, not TASK-3",
	}
	for rel, content := range files {
		full := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	idx, err := buildTaskIndex(root)
	if err != nil {
		t.Fatalf("buildTaskIndex() error = %v", err)
	}

	if _, ok := idx["TASK-6"]; ok {
		t.Error("buildTaskIndex() indexed a card under tasks/plan/, which is not a task zone")
	}
	if _, ok := idx["TASK-3"]; ok {
		t.Error("buildTaskIndex() indexed a card under tasks/_archive/plan/; an archived plan would then satisfy a real TASK-3 child")
	}
	if rec, ok := idx["TASK-249"]; !ok || rec.closed {
		t.Errorf("TASK-249 = %+v, ok=%v; want open (not closed)", idx["TASK-249"], ok)
	}
	if rec, ok := idx["TASK-248"]; !ok || !rec.closed {
		t.Errorf("TASK-248 = %+v, ok=%v; want closed", idx["TASK-248"], ok)
	}
	if rec, ok := idx["TASK-251"]; !ok || !rec.closed {
		t.Errorf("TASK-251 = %+v, ok=%v; want closed (flat under _archive/)", idx["TASK-251"], ok)
	}
	if rec, ok := idx["TASK-244"]; !ok || !rec.closed {
		t.Errorf("TASK-244 = %+v, ok=%v; want closed (nested under _archive/done/)", idx["TASK-244"], ok)
	}
}

func TestTruncatedPercent(t *testing.T) {
	tests := []struct {
		completed, total, want int
	}{
		{15, 16, 93},
		{16, 26, 61},
		{7, 19, 36},
		{2, 3, 66},
		{0, 0, 0},
		{5, 5, 100},
	}
	for _, tt := range tests {
		if got := truncatedPercent(tt.completed, tt.total); got != tt.want {
			t.Errorf("truncatedPercent(%d, %d) = %d, want %d", tt.completed, tt.total, got, tt.want)
		}
	}
}
