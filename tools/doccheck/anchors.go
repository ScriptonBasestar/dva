package main

import (
	"regexp"
	"slices"
	"strings"
	"unicode"
)

var reHeading = regexp.MustCompile(`(?m)^(#{1,6})[ \t]+(.+?)[ \t]*#*[ \t]*$`)

var (
	reHeadingLink = regexp.MustCompile(`\[([^\]]+)\]\([^)]*\)`)
	reHeadingImg  = regexp.MustCompile(`!\[([^\]]*)\]\([^)]*\)`)
)

// collectAnchors returns GitHub-style heading slug set for a markdown body.
// Fenced code blocks are ignored so headings inside fences are not anchors;
// inline code in real headings is retained (stripped only for slug text).
func collectAnchors(src string) map[string]struct{} {
	src = stripFencedRegions(src)
	anchors := make(map[string]struct{})
	seen := make(map[string]int)

	add := func(heading string) {
		base := githubAnchor(heading)
		if base == "" {
			return
		}
		n := seen[base]
		seen[base] = n + 1
		slug := base
		if n > 0 {
			slug = base + "-" + itoa(n)
		}
		anchors[slug] = struct{}{}
	}

	for _, m := range reHeading.FindAllStringSubmatch(src, -1) {
		add(stripHeadingInline(m[2]))
	}
	lines := strings.Split(src, "\n")
	for i := 1; i < len(lines); i++ {
		under := strings.TrimRight(lines[i], " \t\r")
		if under == "" || !isSetextUnderline(under) {
			continue
		}
		text := strings.TrimSpace(lines[i-1])
		if text == "" || strings.HasPrefix(text, "#") {
			continue
		}
		add(stripHeadingInline(text))
	}
	return anchors
}

func isSetextUnderline(s string) bool {
	if len(s) == 0 {
		return false
	}
	ch := s[0]
	if ch != '=' && ch != '-' {
		return false
	}
	for i := 1; i < len(s); i++ {
		if s[i] != ch {
			return false
		}
	}
	return true
}

func stripHeadingInline(s string) string {
	s = strings.TrimSpace(s)
	s = reHeadingLink.ReplaceAllString(s, "$1")
	s = reHeadingImg.ReplaceAllString(s, "$1")
	s = strings.ReplaceAll(s, "`", "")
	s = strings.ReplaceAll(s, "**", "")
	s = strings.ReplaceAll(s, "*", "")
	s = stripUnderscoreEmphasis(s)
	return strings.TrimSpace(s)
}

// stripUnderscoreEmphasis removes `_` delimiters only where GitHub would render
// emphasis, keeping intraword underscores (`sops_source`, `snake_case`)
// literal so heading slugs match GitHub anchors. CommonMark computes flanking
// per delimiter run, not per character: it reads the characters just
// outside a run to compute its left/right flanking, so the `__` in `a __ b`
// is a pure closer and the inner `__` of `x__y__z` is neither (an intraword
// `_` run can neither open nor close). Pairing walks a simplified delimiter
// stack: a closer run matches the nearest openable run above it and consumes
// min length of 2 (strong, when both runs still hold >= 2) or 1 (em)
// delimiters from each run's inner side; a match is vetoed when either run
// can both open and close and (openerLen+closerLen)%3 == 0 (the
// multiple-of-3 rule). Unlike full CommonMark there is no delimiter-stack
// pruning (no openers_bottom): after a multiple-of-3 veto this implementation
// keeps walking to lower openers where cmark would bound the search, so a
// few pathological punctuation-flanked headings can pair that cmark leaves
// literal. Fine for the heading slugs checked here.
func stripUnderscoreEmphasis(s string) string {
	rs := []rune(s)
	isPunct := func(r rune) bool { return unicode.IsPunct(r) || unicode.IsSymbol(r) }
	type run struct {
		start, length     int
		canOpen, canClose bool
	}
	var runs []run
	for i := 0; i < len(rs); {
		if rs[i] != '_' {
			i++
			continue
		}
		j := i
		for j < len(rs) && rs[j] == '_' {
			j++
		}
		prevWS, nextWS := true, true
		prevPunct, nextPunct := false, false
		if i > 0 {
			prevWS = unicode.IsSpace(rs[i-1])
			prevPunct = isPunct(rs[i-1])
		}
		if j < len(rs) {
			nextWS = unicode.IsSpace(rs[j])
			nextPunct = isPunct(rs[j])
		}
		leftFlanking := !nextWS && (!nextPunct || prevWS || prevPunct)
		rightFlanking := !prevWS && (!prevPunct || nextWS || nextPunct)
		r := run{start: i, length: j - i}
		r.canOpen = leftFlanking && (!rightFlanking || prevPunct)
		r.canClose = rightFlanking && (!leftFlanking || nextPunct)
		runs = append(runs, r)
		i = j
	}
	removed := make([]bool, len(rs))
	var openers []int // indexes into runs, most recent last
	for c := range runs {
		cr := &runs[c]
		for cr.canClose && cr.length > 0 {
			oi := -1
			for i, ri := range slices.Backward(openers) {
				o := &runs[ri]
				bothSide := (o.canOpen && o.canClose) || (cr.canOpen && cr.canClose)
				if bothSide && (o.length+cr.length)%3 == 0 {
					continue // multiple-of-3 veto; keep looking below
				}
				oi = i
				break
			}
			if oi < 0 {
				break
			}
			o := &runs[openers[oi]]
			use := 1
			if o.length >= 2 && cr.length >= 2 {
				use = 2
			}
			for n := 0; n < use; n++ { // consume each run from its inner side
				removed[o.start+o.length-1-n] = true
				removed[cr.start+n] = true
			}
			o.length -= use
			cr.length -= use
			if o.length == 0 {
				openers = append(openers[:oi], openers[oi+1:]...)
			}
		}
		if cr.canOpen {
			openers = append(openers, c)
		}
	}
	var b strings.Builder
	b.Grow(len(rs))
	for i, r := range rs {
		if removed[i] {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// githubAnchor approximates GitHub heading anchors for this repo's docs.
func githubAnchor(heading string) string {
	var b strings.Builder
	b.Grow(len(heading))
	prevHyphen := false
	for _, r := range strings.ToLower(heading) {
		switch {
		case unicode.IsLetter(r) || unicode.IsNumber(r):
			b.WriteRune(r)
			prevHyphen = false
		case r == '_' || r == '-':
			b.WriteRune(r)
			prevHyphen = r == '-'
		case unicode.IsSpace(r):
			if !prevHyphen {
				b.WriteByte('-')
				prevHyphen = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [12]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
