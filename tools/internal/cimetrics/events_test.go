package cimetrics

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"
)

func TestParserEventsPrivacyAndState(t *testing.T) {
	var events []Progress
	p := NewParser(map[string]map[string]bool{"example/pkg": {"TestSafe": true}}, func(v Progress) { events = append(events, v) })
	input := `{"Action":"run","Package":"example/pkg","Test":"TestSafe"}
{"Action":"pause","Package":"example/pkg","Test":"TestSafe"}
{"Action":"cont","Package":"example/pkg","Test":"TestSafe"}
{"Action":"run","Package":"example/pkg","Test":"TestSafe/private-SENTINEL"}
{"Action":"pass","Package":"example/pkg","Test":"TestSafe/private-SENTINEL","Elapsed":0.2}
{"Action":"fail","Package":"unknown-SENTINEL","Test":"secret-SENTINEL","Elapsed":0.3}
{"Action":"output","Package":"example/pkg","Output":"secret-SENTINEL (cached)"}
{"Action":"skip","Package":"example/pkg","Test":"TestSafe"}
{"Action":"pass","Package":"example/pkg"}
{"Action":"fail","Package":"unknown-SENTINEL"}
`
	if n, err := p.Write([]byte(input)); err != nil || n != len(input) {
		t.Fatal(n, err)
	}
	p.Finish()
	s := p.Snapshot()
	if s.Status != "complete" || !s.GoTestCached || s.Counts["pass"] != 2 || s.Counts["skip"] != 1 || len(events) != 9 {
		t.Fatalf("snapshot=%+v events=%v", s, events)
	}
	b, _ := json.Marshal(struct {
		S Snapshot
		E []Progress
	}{s, events})
	if strings.Contains(string(b), "SENTINEL") {
		t.Fatalf("leaked: %s", b)
	}
}

func TestParserIncompleteAndLimits(t *testing.T) {
	p := NewParser(nil, nil)
	_, _ = p.Write([]byte("{\"Action\":\"run\",\"Package\":\"x\",\"Test\":\"a\"}\n"))
	_, _ = p.Write([]byte("{broken}\n" + strings.Repeat("x", 65537) + "\n" + `{"Action":"surprise","Package":"x"}` + "\n" + "{"))
	p.Finish()
	s := p.Snapshot()
	if s.Status != "partial" || len(s.Incomplete) != 1 || len(s.Reasons) < 4 {
		t.Fatalf("%+v", s)
	}
	q := NewParser(nil, nil)
	for i := range 5000 {
		_, _ = fmt.Fprintf(q, "{\"Action\":\"pass\",\"Package\":\"x\",\"Test\":\"%d\"}\n", i)
	}
	_, _ = q.Write([]byte("{\"Action\":\"pass\",\"Package\":\"x\"}\n"))
	q.Finish()
	out, _ := json.Marshal(q.Snapshot())
	if len(out) >= 2<<20 || q.Snapshot().Status != "complete" || !q.Snapshot().TimingsTruncated || q.Snapshot().Counts["pass"] != 5001 {
		t.Fatal(len(out))
	}
}

func TestParserWholeLineIncompleteStreams(t *testing.T) {
	for _, input := range []string{
		"{\"Action\":\"output\",\"Package\":\"x\",\"Output\":\"(cached)\"}\n",
		"{\"Action\":\"build-output\",\"ImportPath\":\"x\"}\n",
		"{\"Action\":\"pass\",\"Package\":\"x\",\"Test\":\"TestX\"}\n",
		"{\"Action\":\"build-output\",\"ImportPath\":\"x\"}\n{\"Action\":\"pass\",\"Package\":\"y\"}\n",
		"{\"Action\":\"build-output\",\"ImportPath\":\"x\"}\n{\"Action\":\"build-output\",\"ImportPath\":\"z\"}\n{\"Action\":\"build-fail\",\"ImportPath\":\"x\"}\n",
		"{\"Action\":\"build-output\",\"ImportPath\":\"x [x.test]\"}\n{\"Action\":\"pass\",\"Package\":\"x\"}\n",
	} {
		p := NewParser(nil, nil)
		_, _ = p.Write([]byte(input))
		p.Finish()
		if p.Snapshot().Status != "partial" {
			t.Fatalf("%q: %+v", input, p.Snapshot())
		}
	}
	for _, input := range []string{
		"{\"Action\":\"build-output\",\"ImportPath\":\"x\"}\n{\"Action\":\"pass\",\"Package\":\"x\"}\n",
		"{\"Action\":\"build-output\",\"ImportPath\":\"x\"}\n{\"Action\":\"build-fail\",\"ImportPath\":\"x\"}\n",
		"{\"Action\":\"build-output\",\"ImportPath\":\"x\"}\n{\"Action\":\"build-output\",\"ImportPath\":\"z\"}\n{\"Action\":\"build-fail\",\"ImportPath\":\"x\"}\n{\"Action\":\"build-fail\",\"ImportPath\":\"z\"}\n",
		"{\"Action\":\"output\",\"Package\":\"x\",\"Output\":\"(cached)\"}\n{\"Action\":\"pass\",\"Package\":\"x\"}\n",
	} {
		p := NewParser(nil, nil)
		_, _ = p.Write([]byte(input))
		p.Finish()
		if p.Snapshot().Status != "complete" {
			t.Fatalf("%q: %+v", input, p.Snapshot())
		}
	}
}

func TestParserPendingDetailsBoundedAndPrivate(t *testing.T) {
	p := NewParser(nil, nil)
	for i := range 150 {
		_, _ = fmt.Fprintf(p, "{\"Action\":\"build-output\",\"ImportPath\":\"private-SENTINEL-%d\"}\n", i)
	}
	_, _ = p.Write([]byte("{\"Action\":\"output\",\"Package\":\"private-SENTINEL-package\"}\n"))
	p.Finish()
	s := p.Snapshot()
	b, _ := json.Marshal(s)
	if s.PendingBuildCount != 150 || len(s.PendingBuilds) != 128 || s.PendingPackageCount != 1 || !s.PendingDetailsTruncated || strings.Contains(string(b), "SENTINEL") || len(b) >= 2<<20 {
		t.Fatalf("%+v size=%d", s, len(b))
	}
}
func TestParserConcurrentSnapshot(t *testing.T) {
	p := NewParser(nil, nil)
	var wg sync.WaitGroup
	wg.Go(func() {
		for range 100 {
			p.Snapshot()
		}
	})
	for range 100 {
		_, _ = p.Write([]byte("{\"Action\":\"pass\",\"Package\":\"x\"}\n"))
	}
	wg.Wait()
	p.Finish()
}

func TestParserBuildEventsAndRandomFailure(t *testing.T) {
	var events []Progress
	p := NewParser(nil, func(v Progress) { events = append(events, v) })
	p.noRandom = true
	p.partial("randomness unavailable")
	_, _ = p.Write([]byte("{\"Action\":\"build-output\",\"ImportPath\":\"secret\",\"Output\":\"secret\"}\n{\"Action\":\"build-fail\",\"ImportPath\":\"secret\"}\n"))
	p.Finish()
	s := p.Snapshot()
	if s.Status != "partial" || s.Counts["build-fail"] != 1 || len(events) != 1 || events[0].Package != "redacted" || len(s.Completed) != 0 {
		t.Fatalf("%+v %+v", s, events)
	}
}

func TestParserMissingAndZeroElapsed(t *testing.T) {
	p := NewParser(nil, nil)
	_, _ = p.Write([]byte("{\"Action\":\"pass\",\"Package\":\"x\",\"Elapsed\":0}\n{\"Action\":\"skip\",\"Package\":\"y\"}\n"))
	p.Finish()
	s := p.Snapshot()
	if s.Status != "complete" || s.Completed[0].ElapsedSeconds == nil || *s.Completed[0].ElapsedSeconds != 0 || s.Completed[1].ElapsedSeconds != nil {
		t.Fatalf("%+v", s)
	}
	b, _ := json.Marshal(s.Completed)
	if !strings.Contains(string(b), `"elapsed_seconds":0`) {
		t.Fatalf("zero missing: %s", b)
	}
	*p.Snapshot().Completed[0].ElapsedSeconds = 4
	if *p.Snapshot().Completed[0].ElapsedSeconds != 0 {
		t.Fatal("snapshot exposed mutable state")
	}
}
func TestParserEmptyAndInvalidOutput(t *testing.T) {
	for _, input := range []string{"", "{\"Action\":\"output\",\"Output\":\"(cached)\"}\n", "{\"Action\":\"build-output\"}\n"} {
		p := NewParser(nil, nil)
		_, _ = p.Write([]byte(input))
		p.Finish()
		s := p.Snapshot()
		if s.Status != "partial" || s.GoTestCached {
			t.Fatalf("%q: %+v", input, s)
		}
	}
	p := NewParser(nil, nil)
	_, _ = p.Write([]byte("{\"Action\":\"start\",\"Package\":\"x\"}\n{\"Action\":\"skip\",\"Package\":\"x\"}\n"))
	p.Finish()
	if p.Snapshot().Status != "complete" {
		t.Fatal(p.Snapshot())
	}
}
