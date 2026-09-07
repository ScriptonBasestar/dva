package main

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
)

// zone is where a task card physically sits, which is what decides whether it counts as
// closed. tasks/plan/ is never a card and is excluded before this type applies.
type zone int

const (
	zoneOpen zone = iota // tasks/todo/, tasks/doing/
	zoneClosed
	zoneOther // anywhere else under tasks/ (e.g. a future tasks/issue/) — found, but not closed
)

// taskIndex maps a normalized "TASK-N" id (no leading zeros) to every card file found for it.
// A well-formed tree has exactly one match per id; more than one is reported by the caller as
// part of the id's zone, not treated as an error here — planprogress checks plan/child
// consistency, not card uniqueness.
type taskIndex map[string]taskRecord

type taskRecord struct {
	paths  []string
	closed bool // true if ANY matching file sits under a closed zone
}

var cardFilenameRE = regexp.MustCompile(`^0*(\d+)-.+\.md$`)

// buildTaskIndex walks <root>/tasks, excluding tasks/plan/, and classifies every card file by
// id. A path under tasks/done/ or tasks/_archive/ (at any depth, e.g. tasks/_archive/done/) is
// closed; a path under tasks/todo/ or tasks/doing/ is open; anything else found under tasks/ is
// recorded but counted as neither.
func buildTaskIndex(root string) (taskIndex, error) {
	tasksDir := filepath.Join(root, "tasks")
	idx := taskIndex{}
	err := filepath.WalkDir(tasksDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(tasksDir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if isPlanPath(rel) {
			return nil
		}
		m := cardFilenameRE.FindStringSubmatch(filepath.Base(rel))
		if m == nil {
			return nil
		}
		n, err := strconv.Atoi(m[1])
		if err != nil {
			return nil
		}
		id := fmt.Sprintf("TASK-%d", n)
		rec := idx[id]
		rec.paths = append(rec.paths, rel)
		if zoneFromPath(rel) == zoneClosed {
			rec.closed = true
		}
		idx[id] = rec
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk %s: %w", tasksDir, err)
	}
	return idx, nil
}

// isPlanPath reports whether a tasks/-relative path is a plan card rather than a task card.
// Plans live in tasks/plan/ and, once complete, in tasks/_archive/plan/ — both have to be
// excluded from the task index. A plan filename ("003-command-surface-renewal-discovery.md")
// is indistinguishable from a task filename, so an archived plan would otherwise be indexed
// as TASK-3 and silently satisfy a real child of that id.
func isPlanPath(rel string) bool {
	return slices.Contains(strings.Split(rel, "/"), "plan")
}

func zoneFromPath(rel string) zone {
	switch {
	case rel == "done" || strings.HasPrefix(rel, "done/"):
		return zoneClosed
	case rel == "_archive" || strings.HasPrefix(rel, "_archive/"):
		return zoneClosed
	case rel == "todo" || strings.HasPrefix(rel, "todo/"):
		return zoneOpen
	case rel == "doing" || strings.HasPrefix(rel, "doing/"):
		return zoneOpen
	default:
		return zoneOther
	}
}

// plan is the frontmatter this tool cares about from one tasks/plan/*.md file.
type plan struct {
	path           string
	id             string
	children       []string
	totalTasks     int
	hasTotalTasks  bool
	completedTasks int
	hasCompleted   bool
	progress       int
	hasProgress    bool
}

var frontmatterFieldRE = regexp.MustCompile(`^([A-Za-z][A-Za-z0-9_-]*):\s*(.*)$`)

// loadPlans reads every tasks/plan/*.md file under root and parses the frontmatter fields
// planprogress checks. It is deliberately narrow — id, children, total-tasks, completed-tasks,
// progress — and ignores everything else in the card, including the body.
func loadPlans(root string) ([]plan, error) {
	planDir := filepath.Join(root, "tasks", "plan")
	entries, err := os.ReadDir(planDir)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", planDir, err)
	}
	var plans []plan
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		path := filepath.Join(planDir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
		p, err := parsePlan(string(data))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		p.path = filepath.ToSlash(path)
		plans = append(plans, p)
	}
	sort.Slice(plans, func(i, j int) bool { return plans[i].path < plans[j].path })
	return plans, nil
}

// parsePlan extracts frontmatter fields from one plan card's raw file content.
func parsePlan(data string) (plan, error) {
	lines := strings.Split(strings.TrimPrefix(data, "\uFEFF"), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return plan{}, fmt.Errorf("no opening frontmatter fence")
	}
	end := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			end = i
			break
		}
	}
	if end == -1 {
		return plan{}, fmt.Errorf("unterminated frontmatter")
	}

	var p plan
	for _, line := range lines[1:end] {
		if line == "" || line[0] == ' ' || line[0] == '\t' || line[0] == '#' {
			continue // indented/blank/comment lines are not top-level frontmatter fields
		}
		m := frontmatterFieldRE.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		key, val := m[1], strings.TrimSpace(m[2])
		switch key {
		case "id":
			p.id = strings.Trim(val, `"'`)
		case "children":
			children, err := parseChildren(val)
			if err != nil {
				return plan{}, fmt.Errorf("children: %w", err)
			}
			p.children = children
		case "total-tasks":
			n, err := strconv.Atoi(val)
			if err != nil {
				return plan{}, fmt.Errorf("total-tasks: not an integer: %q", val)
			}
			p.totalTasks, p.hasTotalTasks = n, true
		case "completed-tasks":
			n, err := strconv.Atoi(val)
			if err != nil {
				return plan{}, fmt.Errorf("completed-tasks: not an integer: %q", val)
			}
			p.completedTasks, p.hasCompleted = n, true
		case "progress":
			n, err := strconv.Atoi(val)
			if err != nil {
				return plan{}, fmt.Errorf("progress: not an integer: %q", val)
			}
			p.progress, p.hasProgress = n, true
		}
	}
	return p, nil
}

// parseChildren parses a bracketed, comma-separated frontmatter value such as
// "[TASK-244, TASK-245, TASK-246]" into its normalized ("TASK-N", no leading zeros) ids.
func parseChildren(val string) ([]string, error) {
	val = strings.TrimSpace(val)
	if val == "[]" || val == "" {
		return nil, nil
	}
	if !strings.HasPrefix(val, "[") || !strings.HasSuffix(val, "]") {
		return nil, fmt.Errorf("expected a bracketed list, got %q", val)
	}
	inner := strings.TrimSpace(val[1 : len(val)-1])
	if inner == "" {
		return nil, nil
	}
	var out []string
	for tok := range strings.SplitSeq(inner, ",") {
		id, err := normalizeTaskID(strings.TrimSpace(tok))
		if err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, nil
}

var taskIDRE = regexp.MustCompile(`^TASK-0*(\d+)$`)

func normalizeTaskID(tok string) (string, error) {
	m := taskIDRE.FindStringSubmatch(tok)
	if m == nil {
		return "", fmt.Errorf("not a TASK-N id: %q", tok)
	}
	n, err := strconv.Atoi(m[1])
	if err != nil {
		return "", fmt.Errorf("not a TASK-N id: %q", tok)
	}
	return fmt.Sprintf("TASK-%d", n), nil
}

// checkPlan compares one plan's declared frontmatter against the measured reality in idx and
// returns one human-readable defect message per problem found. A plan with no defects
// contributes nothing.
func checkPlan(p plan, idx taskIndex) []string {
	name := p.id
	if name == "" {
		name = p.path
	}

	var defects []string

	var missing []string
	closedCount := 0
	for _, child := range p.children {
		rec, ok := idx[child]
		if !ok {
			missing = append(missing, child)
			continue
		}
		if rec.closed {
			closedCount++
		}
	}
	for _, child := range missing {
		defects = append(defects, fmt.Sprintf("%s (%s): children lists %s, but no task card exists for it anywhere under tasks/", name, p.path, child))
	}

	wantTotal := len(p.children)
	if !p.hasTotalTasks {
		defects = append(defects, fmt.Sprintf("%s (%s): missing total-tasks", name, p.path))
	} else if p.totalTasks != wantTotal {
		defects = append(defects, fmt.Sprintf("%s (%s): total-tasks=%d, want %d (len(children))", name, p.path, p.totalTasks, wantTotal))
	}

	if !p.hasCompleted {
		defects = append(defects, fmt.Sprintf("%s (%s): missing completed-tasks", name, p.path))
	} else if p.completedTasks != closedCount {
		defects = append(defects, fmt.Sprintf("%s (%s): completed-tasks=%d, want %d (measured closed children)", name, p.path, p.completedTasks, closedCount))
	}

	if !p.hasProgress {
		defects = append(defects, fmt.Sprintf("%s (%s): missing progress", name, p.path))
	} else if p.hasTotalTasks && p.hasCompleted {
		wantProgress := roundPercent(p.completedTasks, p.totalTasks)
		if p.progress != wantProgress {
			defects = append(defects, fmt.Sprintf("%s (%s): progress=%d, want %d (round(completed-tasks/total-tasks*100))", name, p.path, p.progress, wantProgress))
		}
	}

	return defects
}

// roundPercent rounds completed/total*100 to the nearest integer, half away from zero,
// matching the convention already used by hand across tasks/plan/*.md. A zero total rounds to
// zero rather than dividing by zero — an empty plan is 0% complete, not NaN.
func roundPercent(completed, total int) int {
	if total == 0 {
		return 0
	}
	return int(math.Round(float64(completed) / float64(total) * 100))
}
