package main

import (
	"strings"
	"testing"
)

// childrenOf builds the children list and the matching total-tasks for a fixture plan.
func childrenOf(ids ...string) plan {
	return plan{children: ids, totalTasks: len(ids), hasTotalTasks: true}
}

// The fixtures below are deliberately shaped like the live plan cards rather than invented
// minimal strings: the rules in prose.go were chosen by measuring those four cards, so a
// fixture that drops their punctuation ("—", "·", bare-number continuations) would stop
// testing the thing that was decided.
func TestCheckPlanProse(t *testing.T) {
	tests := []struct {
		name    string
		p       plan
		wantAny []string
		wantN   int
	}{
		{
			// The defect TASK-380 found: scope enumerates and counts six, children carries
			// seven. Every pre-TASK-381 gate passed this.
			name: "scope enumerates and counts fewer cards than total-tasks",
			p: func() plan {
				p := childrenOf("TASK-371", "TASK-344", "TASK-350", "TASK-343", "TASK-354", "TASK-338", "TASK-377")
				p.id = "PLAN-009"
				p.scope = "TASK-371, 344, 350, 343, 354, 338 — 여섯 장이 shared task progress contract를 고친다"
				return p
			}(),
			wantAny: []string{"enumerates and counts 6 card(s), but total-tasks=7"},
			wantN:   1,
		},
		{
			name: "the same scope once the seventh child is written in is clean",
			p: func() plan {
				p := childrenOf("TASK-371", "TASK-344", "TASK-350", "TASK-343", "TASK-354", "TASK-338", "TASK-377")
				p.id = "PLAN-009"
				p.scope = "TASK-371, 344, 350, 343, 354, 338, 377 — 일곱 장이 shared task progress contract를 고친다"
				return p
			}(),
			wantN: 0,
		},
		{
			// PLAN-008's real scope. TASK-318 is the re-review that spawned the children, not
			// a child — an isolated mention must stay context, or a correct card fails the gate.
			name: "an isolated id beside a range is context, not a membership claim",
			p: func() plan {
				p := childrenOf("TASK-358", "TASK-359", "TASK-360", "TASK-361", "TASK-362", "TASK-363", "TASK-364", "TASK-365")
				p.id = "PLAN-008"
				p.scope = "TASK-358..365 — TASK-318 재리뷰가 파생시킨 8장. 전부 한 파일에 걸린다"
				return p
			}(),
			wantN: 0,
		},
		{
			name: "an enumerated id that children does not list is reported",
			p: func() plan {
				p := childrenOf("TASK-1", "TASK-2", "TASK-3")
				p.id = "PLAN-X"
				p.scope = "TASK-1, 2, 999 — 세 장이 같은 파일을 고친다"
				return p
			}(),
			wantAny: []string{"children: does not list", "TASK-999"},
			wantN:   1,
		},
		{
			// PLAN-009's Goal counts five of its seven children. total-tasks is not the
			// yardstick outside scope:, only the enumeration standing next to the count.
			name: "a Goal count describing a subset is checked against its own enumeration",
			p: func() plan {
				p := childrenOf("TASK-371", "TASK-344", "TASK-350", "TASK-343", "TASK-354", "TASK-338", "TASK-377")
				p.id = "PLAN-009"
				p.goal = "TASK-371은 contract를 맞추고, 다섯 장(TASK-344·350·343·354·338)은 doccheck를 고친다"
				return p
			}(),
			wantN: 0,
		},
		{
			name: "a Goal count that disagrees with the enumeration beside it is reported",
			p: func() plan {
				p := childrenOf("TASK-1", "TASK-2", "TASK-3", "TASK-4")
				p.id = "PLAN-Y"
				p.goal = "네 장(TASK-1·2·3)이 같은 파일을 고친다"
				return p
			}(),
			wantAny: []string{"counts 4 card(s) beside an enumeration of 3"},
			wantN:   1,
		},
		{
			// PLAN-007 and PLAN-006 in miniature: quantities that count something other than
			// children, with no enumeration anywhere near them.
			name: "counts with no enumeration beside them are not children counts",
			p: func() plan {
				p := childrenOf("TASK-1", "TASK-2")
				p.id = "PLAN-Z"
				p.scope = "tasks/done/ 58장의 일회성 트리아지, 소유자 없는 결함 3건"
				p.goal = "2026-09-05 dogfood(23개 devbox 저장소를 이전)에서 나온 카드를 처리한다"
				return p
			}(),
			wantN: 0,
		},
		{
			// PLAN-006: a range with no count phrase asserts membership but claims nothing
			// about the plan's size, which is 29 against a 13-wide range.
			name: "a range without a count phrase is not compared to total-tasks",
			p: func() plan {
				p := plan{
					id:            "PLAN-006",
					children:      []string{"TASK-311", "TASK-312", "TASK-313", "TASK-328"},
					totalTasks:    29,
					hasTotalTasks: true,
					scope:         "TASK-311..313 from the migration, plus the needs-human cards that gate the rest",
				}
				return p
			}(),
			wantN: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := checkPlanProse(tt.p, tt.p.id)
			if len(got) != tt.wantN {
				t.Fatalf("checkPlanProse() = %v (%d defects), want %d", got, len(got), tt.wantN)
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
					t.Errorf("no defect contains %q; got %v", want, got)
				}
			}
		})
	}
}

func TestFindEnumerations(t *testing.T) {
	tests := []struct {
		name     string
		sentence string
		want     []string
	}{
		{"single mention is not an enumeration", "TASK-318 재리뷰가 파생시켰다", nil},
		{"comma run with bare continuations", "TASK-371, 344, 350", []string{"TASK-371", "TASK-344", "TASK-350"}},
		{"middot run", "TASK-344·350·343", []string{"TASK-344", "TASK-350", "TASK-343"}},
		{"range expands", "TASK-358..361", []string{"TASK-358", "TASK-359", "TASK-360", "TASK-361"}},
		{"leading zeros normalize", "TASK-007, 008", []string{"TASK-7", "TASK-8"}},
		// A range that is then continued by a run: the tail used to be dropped silently,
		// which let an extra id ride along unchecked inside a correct-looking range.
		{"range continued by a run", "TASK-358..360, 999", []string{"TASK-358", "TASK-359", "TASK-360", "TASK-999"}},
		{"reversed range is ignored", "TASK-365..358", nil},
		{"two mentions joined by prose stay isolated", "TASK-343 depends-on TASK-344", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findEnumerations(tt.sentence)
			if tt.want == nil {
				if len(got) != 0 {
					t.Fatalf("findEnumerations(%q) = %v, want none", tt.sentence, got)
				}
				return
			}
			if len(got) != 1 {
				t.Fatalf("findEnumerations(%q) returned %d enumerations, want 1", tt.sentence, len(got))
			}
			if strings.Join(got[0].ids, ",") != strings.Join(tt.want, ",") {
				t.Errorf("findEnumerations(%q) ids = %v, want %v", tt.sentence, got[0].ids, tt.want)
			}
		})
	}
}

func TestFindCountPhrases(t *testing.T) {
	tests := []struct {
		sentence string
		want     []int
	}{
		{"파생시킨 8장", []int{8}},
		{"일곱 장이", []int{7}},
		{"열두 개의", []int{12}},
		{"58장의 트리아지, 결함 3건", []int{58, 3}},
		{"367줄짜리 파일", nil}, // 줄 is not a card counter
		{"여러 장을 닫았다", nil}, // not a numeral
		{"일 개월", []int{}},  // sino-Korean numerals are deliberately not matched
	}

	for _, tt := range tests {
		t.Run(tt.sentence, func(t *testing.T) {
			got := findCountPhrases(tt.sentence)
			if len(got) != len(tt.want) {
				t.Fatalf("findCountPhrases(%q) = %v, want %v", tt.sentence, got, tt.want)
			}
			for i, w := range tt.want {
				if got[i].n != w {
					t.Errorf("findCountPhrases(%q)[%d] = %d, want %d", tt.sentence, i, got[i].n, w)
				}
			}
		})
	}
}

func TestExtractSection(t *testing.T) {
	body := strings.Split(strings.TrimSpace(`
## Goal

첫 문단이다.

두번째 문단이다.

## Order

이건 Goal이 아니다.
`), "\n")

	got := extractSection(body, "## Goal")
	if !strings.Contains(got, "첫 문단이다.") || !strings.Contains(got, "두번째 문단이다.") {
		t.Errorf("extractSection dropped Goal content: %q", got)
	}
	if strings.Contains(got, "이건 Goal이 아니다.") {
		t.Errorf("extractSection ran past the next heading: %q", got)
	}
	if extractSection(body, "## Missing") != "" {
		t.Errorf("an absent section should yield the empty string")
	}
}
