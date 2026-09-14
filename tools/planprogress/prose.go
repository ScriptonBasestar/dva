package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// This file checks a plan's *prose* against its own frontmatter. checkPlan compares
// total-tasks/completed-tasks/progress against where the child cards physically sit; nothing
// looked at what the card says in words, so a plan could name six children in `scope:` while
// `children:` carried seven and no gate noticed (TASK-380 D-2).
//
// Scope of the machine check (TASK-381, decided against the four live plans before writing
// any of it — the card's originally proposed rules were measured first and rejected):
//
//   - A lone "TASK-318" in prose is CONTEXT, not a membership claim. PLAN-008's scope names
//     TASK-318 as the re-review that spawned its children; flagging it as "not in children"
//     would be a false positive on a correct card. Only an *enumeration* — a range
//     ("TASK-358..365") or a run of two or more ids joined by "," or "·" — asserts membership.
//   - A count phrase ("8장", "일곱 장") only counts children when it sits beside such an
//     enumeration. PLAN-007's "done 카드 58장" and PLAN-006's "23개 devbox 저장소" count
//     something else entirely, and a rule that reads every numeral would reject both.
//   - Sharing a sentence is not enough on its own, and the first cut of this rule was wrong to
//     treat it as such (ISSUE-016). Two further conditions were added by measurement:
//     (a) only 장 pairs. 개/건/장 are all generic Korean counters, but in this corpus 장 is the
//     only one that ever counts cards — "필드 32개", "관련 문서 8건", "23개 devbox 저장소" count
//     fields, documents and repositories, and each of those sentences can legitimately carry an
//     id enumeration beside it. findCountPhrases still reports all three (a 건 count is a real
//     quantity; it is just not a card count), so the narrowing lives in the pairing, not in the
//     vocabulary.
//     (b) pairing is symmetric. A count and an enumeration pair only when each is the other's
//     nearest neighbour in the sentence, so "TASK-1·2와 TASK-3·4·5, 세 장이 겹친다" does not
//     hand the same "세 장" to the far enumeration as well. Symmetry narrows F3; it does not
//     remove it, and this comment said otherwise until review-planprogress measured the
//     counterexample. Reorder the same sentence and the false positive returns:
//     "다섯 장 — TASK-1·2와 TASK-3·4·5" (five children, written as two groups) still reports
//     "counts 5 card(s) beside an enumeration naming 2". Symmetry guarantees that only ONE
//     enumeration is blamed; it does not establish that the blamed one is the count's subject.
//     The residue is tracked on ISSUE-021, not claimed as fixed here.
//   - total-tasks is only comparable to prose in `scope:`, which by definition describes the
//     whole plan. In `## Goal` a count legitimately describes a subset — PLAN-009 says
//     "다섯 장(TASK-344·350·343·354·338)" about five of its seven children — so there the
//     count is checked against the enumeration beside it and nothing else.
//
// What this deliberately does NOT do: judge whether a free-form sentence asserting a remainder
// names the set it counts (the defect TASK-385 found in PLAN-006 §Session handoff). That is not
// a regex-decidable property and pretending otherwise would trade one silent gap for a noisy one.

// countedEnumeration is one enumeration of task ids found in a plan's prose, together with the
// count phrase nearest to it in the same sentence (if any).
type countedEnumeration struct {
	ids      []string // normalized "TASK-N" ids, in the order written
	count    int      // the paired count phrase's value
	hasCount bool
	text     string // the enumeration as written, for the defect message
}

var (
	// taskAnchorRE finds the first id of a potential enumeration. Leading zeros are
	// normalized away, matching normalizeTaskID.
	taskAnchorRE = regexp.MustCompile(`TASK-0*(\d+)`)
	// rangeTailRE continues an anchor as a range: "TASK-358..365".
	rangeTailRE = regexp.MustCompile(`^\.\.0*(\d+)`)
	// runTailRE continues an anchor as a separated run: "TASK-371, 344, 350" or
	// "TASK-344·350·343". Continuations may drop the "TASK-" prefix, which the live plans do.
	runTailRE = regexp.MustCompile(`^\s*[,·]\s*(?:TASK-)?0*(\d+)`)
	// countPhraseRE matches a counted quantity: a digit or a native Korean numeral followed by
	// 장/개/건. Sino-Korean numeral words (일, 이, 삼 …) are excluded on purpose: they are
	// indistinguishable from ordinary words at this level and the digit form covers the same
	// values. The counter itself is captured because only 장 counts cards here — see cardCounter.
	countPhraseRE = regexp.MustCompile(`(\d+|열다섯|열네|열세|열두|열한|다섯|여섯|일곱|여덟|아홉|한|두|세|네|열)\s*([장개건])`)
	// sentenceSplitRE bounds "the same sentence". An em dash does not end one — PLAN-009 writes
	// its enumeration and its count on either side of one.
	sentenceSplitRE = regexp.MustCompile(`\.\s|\n`)
)

var nativeNumerals = map[string]int{
	"한": 1, "두": 2, "세": 3, "네": 4, "다섯": 5, "여섯": 6, "일곱": 7,
	"여덟": 8, "아홉": 9, "열": 10, "열한": 11, "열두": 12, "열세": 13,
	"열네": 14, "열다섯": 15,
}

// maxRangeSpan bounds how many ids a "TASK-A..B" range may expand to. A malformed or reversed
// range should be ignored, not turned into thousands of membership claims.
const maxRangeSpan = 200

// cardCounter is the one Korean counter that this corpus uses for cards. 개 and 건 are matched
// as quantities but never paired with an enumeration — see the file comment (ISSUE-016).
const cardCounter = "장"

// findCountedEnumerations extracts every task-id enumeration in text, sentence by sentence,
// pairing each with the nearest count phrase in its own sentence.
func findCountedEnumerations(text string) []countedEnumeration {
	var out []countedEnumeration
	for _, sentence := range sentenceSplitRE.Split(text, -1) {
		enums := findEnumerations(sentence)
		if len(enums) == 0 {
			continue
		}
		counts := cardCounts(findCountPhrases(sentence))
		for i, e := range enums {
			// Symmetric pairing: the count must be this enumeration's nearest, and this
			// enumeration must be that count's nearest. One-way nearest handed the same count
			// to every enumeration in the sentence (ISSUE-016 F3).
			if ci, ok := nearestCount(e, counts); ok {
				if ei, ok := nearestEnum(counts[ci], enums); ok && ei == i {
					e.count, e.hasCount = counts[ci].n, true
				}
			}
			out = append(out, e.countedEnumeration)
		}
	}
	return out
}

// span records where a match sits in its sentence, so a count can be paired with the
// enumeration it is actually next to rather than one further away in the same sentence.
type span struct{ start, end int }

// gap is the character distance between two spans in the same sentence, 0 when they overlap
// (which only happens if a count sits inside a run).
func (s span) gap(o span) int {
	if g := o.start - s.end; g >= 0 {
		return g
	}
	if g := s.start - o.end; g >= 0 {
		return g
	}
	return 0
}

type enumMatch struct {
	countedEnumeration
	span
}

type countMatch struct {
	n    int
	unit string
	span
}

// findEnumerations returns the id enumerations in one sentence. A single isolated mention is
// context rather than a membership claim and is not returned.
func findEnumerations(sentence string) []enumMatch {
	var out []enumMatch
	for pos := 0; pos < len(sentence); {
		loc := taskAnchorRE.FindStringSubmatchIndex(sentence[pos:])
		if loc == nil {
			break
		}
		start := pos + loc[0]
		end := pos + loc[1]
		first, err := strconv.Atoi(sentence[pos+loc[2] : pos+loc[3]])
		if err != nil {
			pos = end
			continue
		}

		var ids []int
		if m := rangeTailRE.FindStringSubmatchIndex(sentence[end:]); m != nil {
			last, err := strconv.Atoi(sentence[end+m[2] : end+m[3]])
			if err == nil && last >= first && last-first < maxRangeSpan {
				for n := first; n <= last; n++ {
					ids = append(ids, n)
				}
			}
			// end advances past the range tail whether or not the range was accepted. A
			// reversed or oversized range contributes only its anchor id, but the text after
			// it is still ordinary prose: leaving end before the ".." made the run loop below
			// re-read the range's own tail and then stop, so "TASK-365..358, 366, 367" lost
			// 366 and 367 as well — a sound membership claim swallowed by a typo next to it
			// (ISSUE-021 B).
			end += m[1]
		}
		if ids == nil {
			ids = []int{first}
		}
		// A range may still be continued by a separated run ("TASK-358..365, 999"), so this
		// runs whether or not the range branch above matched.
		for {
			m := runTailRE.FindStringSubmatchIndex(sentence[end:])
			if m == nil {
				break
			}
			n, err := strconv.Atoi(sentence[end+m[2] : end+m[3]])
			if err != nil {
				break
			}
			ids = append(ids, n)
			end += m[1]
		}

		// Two or more ids *as written* is what makes this an enumeration rather than an
		// isolated mention; the distinct ids are what it claims to count. Keeping the two
		// apart matters for "TASK-1, 1": still an enumeration, but of one card, so a "두 장"
		// beside it is reported instead of matching the repeat (ISSUE-017 F5).
		if len(ids) >= 2 {
			e := enumMatch{span: span{start: start, end: end}}
			e.text = strings.TrimSpace(sentence[start:end])
			seen := make(map[int]bool, len(ids))
			for _, n := range ids {
				if seen[n] {
					continue
				}
				seen[n] = true
				e.ids = append(e.ids, fmt.Sprintf("TASK-%d", n))
			}
			out = append(out, e)
		}
		pos = end
	}
	return out
}

// findCountPhrases returns every "N장"/"일곱 장" style quantity in one sentence.
func findCountPhrases(sentence string) []countMatch {
	var out []countMatch
	for _, loc := range countPhraseRE.FindAllStringSubmatchIndex(sentence, -1) {
		tok := sentence[loc[2]:loc[3]]
		n, err := strconv.Atoi(tok)
		if err != nil {
			var ok bool
			if n, ok = nativeNumerals[tok]; !ok {
				continue
			}
		}
		out = append(out, countMatch{
			n:    n,
			unit: sentence[loc[4]:loc[5]],
			span: span{start: loc[0], end: loc[1]},
		})
	}
	return out
}

// cardCounts keeps only the quantities that could be counting cards. A 개/건 quantity is still a
// real quantity — it just counts fields, documents or repositories, never children (ISSUE-016).
func cardCounts(counts []countMatch) []countMatch {
	var out []countMatch
	for _, c := range counts {
		if c.unit == cardCounter {
			out = append(out, c)
		}
	}
	return out
}

// nearestCount returns the index of the count phrase closest to e by character gap. Sentences
// that carry more than one quantity ("여덟 장 중 세 장") would otherwise pair arbitrarily.
func nearestCount(e enumMatch, counts []countMatch) (int, bool) {
	best, bestGap := 0, -1
	for i, c := range counts {
		if gap := e.gap(c.span); bestGap == -1 || gap < bestGap {
			best, bestGap = i, gap
		}
	}
	return best, bestGap != -1
}

// nearestEnum is nearestCount's mirror: the index of the enumeration closest to c. Both
// directions are needed because pairing is symmetric — see findCountedEnumerations.
func nearestEnum(c countMatch, enums []enumMatch) (int, bool) {
	best, bestGap := 0, -1
	for i, e := range enums {
		if gap := c.gap(e.span); bestGap == -1 || gap < bestGap {
			best, bestGap = i, gap
		}
	}
	return best, bestGap != -1
}

// checkPlanProse reports where a plan's prose contradicts its own frontmatter. The two prose
// regions are held to different standards on purpose — see the file comment.
func checkPlanProse(p plan, name string) []string {
	var defects []string
	children := make(map[string]bool, len(p.children))
	for _, c := range p.children {
		children[c] = true
	}

	check := func(region, text string, compareTotal bool) {
		for _, e := range findCountedEnumerations(text) {
			var missing []string
			for _, id := range e.ids {
				if !children[id] {
					missing = append(missing, id)
				}
			}
			if len(missing) > 0 {
				defects = append(defects, fmt.Sprintf(
					"%s (%s): %s enumerates %s, which children: does not list (%q)",
					name, p.path, region, strings.Join(missing, ", "), e.text))
			}
			if !e.hasCount {
				continue
			}
			if e.count != len(e.ids) {
				defects = append(defects, fmt.Sprintf(
					"%s (%s): %s counts %d card(s) beside an enumeration naming %d (%q)",
					name, p.path, region, e.count, len(e.ids), e.text))
				continue
			}
			if compareTotal && p.hasTotalTasks && e.count != p.totalTasks {
				defects = append(defects, fmt.Sprintf(
					"%s (%s): %s enumerates and counts %d card(s), but total-tasks=%d (%q)",
					name, p.path, region, e.count, p.totalTasks, e.text))
			}
		}
	}

	check("scope:", p.scope, true)
	check("## Goal", p.goal, false)
	return defects
}
