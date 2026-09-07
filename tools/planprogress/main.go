// Command planprogress keeps tasks/plan/*.md frontmatter (children, total-tasks,
// completed-tasks, progress) in sync with where each child task card actually lives.
//
// Nothing recomputes a plan's counters when a child card closes, so they drift silently —
// a card moves from tasks/todo/ to tasks/done/ or tasks/_archive/ (directly, or nested
// under tasks/_archive/done/) and the plan keeps reporting the old numbers until a human
// recounts by hand. This program does the count itself and fails when the declared
// frontmatter disagrees with the measured reality, or with its own arithmetic.
//
// This is the read-time half of the fix, not the whole fix. ce-agent-kit TASK-133 ("ce task
// validate 가 워크북 host-schema 와 plan 진행률 계산을 흡수한다", open as of 2026-09-07) owns the
// write-time half: making `ce task move` recompute a parent plan's progress/total-tasks/
// completed-tasks when a child moves to done/_archive. The two do not make each other
// redundant — a write-time recompute only protects cards that tool moved, while this CI check
// catches drift from any source (a hand edit, another agent, a plain `git mv`) regardless of
// how the card got there. Keep both after TASK-133 lands.
package main

import (
	"fmt"
	"os"
)

func main() {
	root := "."
	if len(os.Args) > 1 {
		root = os.Args[1]
	}

	index, err := buildTaskIndex(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "planprogress: %v\n", err)
		os.Exit(2)
	}
	plans, err := loadPlans(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "planprogress: %v\n", err)
		os.Exit(2)
	}

	var defects []string
	for _, p := range plans {
		defects = append(defects, checkPlan(p, index)...)
	}

	fmt.Printf("plans_checked: %d\n", len(plans))
	fmt.Printf("task_cards_indexed: %d\n", len(index))

	if len(defects) == 0 {
		fmt.Println("planprogress: OK — every plan's total-tasks/completed-tasks/progress matches its children")
		return
	}
	for _, d := range defects {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", d)
	}
	fmt.Println("planprogress: FAIL")
	os.Exit(1)
}
