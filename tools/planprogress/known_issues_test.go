//go:build knownbroken

package main

import (
	"strings"
	"testing"
)

// This file holds executable reproductions of open board defects. Every test in it is EXPECTED
// TO FAIL on the current code — that is the point, not an accident. They are the machine
// binding for the first Resolution Criterion of ISSUE-016 and ISSUE-017, kept behind the
// "knownbroken" build tag so the default `go test ./...` (and therefore `make test`) stays
// green. Do not remove a failing test here to make it pass; the only correct way to turn one
// green is to fix the defect it reproduces. When a defect is fixed, move its test into
// prose_test.go (dropping the tag) rather than deleting it here — deleting it to clear the
// tag without a corresponding fix is exactly the failure mode this file exists to catch. Both
// issues' last Resolution Criterion is bound to exactly that move: the test name must appear in
// prose_test.go AND be gone from this file, so dropping the tag alone does not satisfy it.
//
// ISSUE-016: TestIssue016FalsePositives — a generic Korean counter (개/건/장) anywhere in the
// same sentence as an id enumeration is read as a card count today, even when it counts
// something else entirely (a struct field count, a line range, a document count).
//
// ISSUE-017: TestIssue017Deduplication — a repeated id in a comma run is not deduplicated
// today, so "TASK-1, 1, 2" reports three ids for two distinct cards.

// TestIssue016FalsePositives reproduces review-381's three measured false positives (F1,
// 2026-09-14): a numeral of the counter shape (개/건/장) beside an id enumeration is paired
// with it as a card count even when it counts something other than cards. Expected to FAIL
// until the counter vocabulary or the pairing rule is narrowed — do not silence these by
// loosening wantN.
func TestIssue016FalsePositives(t *testing.T) {
	tests := []struct {
		name  string
		p     plan
		wantN int
	}{
		{
			name: "a struct field count in Goal is not an enumeration count",
			p: func() plan {
				p := childrenOf("TASK-1", "TASK-2")
				p.id = "PLAN-ISSUE-016-A"
				p.goal = "TASK-1·2는 check.go의 Result 구조체(필드 32개)를 고친다"
				return p
			}(),
			wantN: 0,
		},
		{
			name: "a struct field count beside a line range in scope is not an enumeration count",
			p: func() plan {
				p := childrenOf("TASK-1", "TASK-2", "TASK-3")
				p.id = "PLAN-ISSUE-016-B"
				p.scope = "TASK-1, 2, 3 — check.go(19-51행, 필드 32개)"
				return p
			}(),
			wantN: 0,
		},
		{
			name: "a document count in scope is not an enumeration count",
			p: func() plan {
				p := childrenOf("TASK-1", "TASK-2", "TASK-3")
				p.id = "PLAN-ISSUE-016-C"
				p.scope = "TASK-1, 2, 3 — 관련 문서 8건을 정리한다"
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
		})
	}
}

// TestIssue017Deduplication reproduces review-381's F5 (2026-09-14): findEnumerations does
// not deduplicate a repeated id in a comma run, so "TASK-1, 1, 2" is read as three ids for two
// distinct cards. Expected to FAIL until findEnumerations deduplicates.
func TestIssue017Deduplication(t *testing.T) {
	sentence := "TASK-1, 1, 2"
	want := []string{"TASK-1", "TASK-2"}

	got := findEnumerations(sentence)
	if len(got) != 1 {
		t.Fatalf("findEnumerations(%q) returned %d enumerations, want 1", sentence, len(got))
	}
	if strings.Join(got[0].ids, ",") != strings.Join(want, ",") {
		t.Errorf("findEnumerations(%q) ids = %v, want %v", sentence, got[0].ids, want)
	}
}
