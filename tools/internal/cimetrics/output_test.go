package cimetrics

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"
)

type failOutput struct{}

func (failOutput) Write([]byte) (int, error) { return 0, errors.New("fixture output failure") }

const outputFixturePackage = "example.test/safe"

func outputFixture(out io.Writer, verbose bool) (*Parser, *outputRenderer) {
	allowed := map[string]map[string]bool{outputFixturePackage: {"TestPass": true, "TestFail": true}}
	p := NewParser(allowed, nil)
	return p, newOutputRenderer(p, out, verbose)
}

func sendOutputEvent(t *testing.T, p *Parser, r *outputRenderer, event map[string]any) {
	t.Helper()
	b, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	b = append(b, '\n')
	// Exercise arbitrary pipe chunking, including splits inside JSON strings.
	w := io.MultiWriter(p, r)
	for len(b) > 0 {
		n := min(7, len(b))
		if _, err := w.Write(b[:n]); err != nil {
			t.Fatal(err)
		}
		b = b[n:]
	}
}

func event(action, test, text string) map[string]any {
	e := map[string]any{"Action": action, "Package": outputFixturePackage}
	if test != "" {
		e["Test"] = test
	}
	if text != "" {
		e["Output"] = text
	}
	return e
}

func TestMetricsOutputQuietInterleavedFailuresAndPassingSecrets(t *testing.T) {
	var out bytes.Buffer
	p, r := outputFixture(&out, false)
	for _, e := range []map[string]any{
		event("start", "", ""), event("run", "TestPass", ""), event("output", "TestPass", "=== RUN TestPass\n"),
		event("output", "TestPass", "passing-secret-fixture\n"), event("pause", "TestPass", ""),
		event("run", "TestFail", ""), event("output", "TestFail", "=== RUN TestFail\n"),
		event("output", "TestFail", "failure diagnostic\n"), event("fail", "TestFail", ""),
		event("cont", "TestPass", ""), event("pass", "TestPass", ""),
		event("output", "", "package-level diagnostic\n"), event("fail", "", ""),
	} {
		sendOutputEvent(t, p, r, e)
	}
	r.Finish()
	p.Finish()
	got := out.String()
	if strings.Contains(got, "passing-secret-fixture") || strings.Contains(got, "=== RUN") {
		t.Fatalf("passing/protocol output leaked: %q", got)
	}
	if !strings.Contains(got, "failure diagnostic") || !strings.Contains(got, "package-level diagnostic") || !strings.Contains(got, "FAIL\t"+outputFixturePackage) {
		t.Fatalf("failure output missing: %q", got)
	}
	if r.Degraded() || p.Snapshot().Status != "complete" {
		t.Fatalf("complete output marked partial: %t %+v", r.Degraded(), p.Snapshot())
	}
}

func TestMetricsOutputSuppressesPassingTestMainAndKeepsCoverage(t *testing.T) {
	var out bytes.Buffer
	p, r := outputFixture(&out, false)
	for _, e := range []map[string]any{
		event("start", "", ""), event("output", "", "testmain-secret-fixture\n"),
		event("output", "", "ok\texample.test/safe\t0.100s\tcoverage: 37.5% of statements\n"),
		{"Action": "pass", "Package": outputFixturePackage, "Elapsed": 0.0},
	} {
		sendOutputEvent(t, p, r, e)
	}
	r.Finish()
	p.Finish()
	got := out.String()
	if strings.Contains(got, "testmain-secret-fixture") || !strings.Contains(got, "0.000s") || !strings.Contains(got, "coverage: 37.5%") {
		t.Fatalf("quiet summary is wrong: %q", got)
	}
	if r.Degraded() {
		t.Fatal("complete package was marked degraded")
	}
}

func TestMetricsOutputMalformedOversizedAndUnfinishedNeverLeaks(t *testing.T) {
	var out bytes.Buffer
	p, r := outputFixture(&out, false)
	w := io.MultiWriter(p, r)
	for _, s := range []string{"malformed-secret\n", strings.Repeat("x", outputLineLimit+1) + "oversized-secret\n"} {
		if _, err := io.WriteString(w, s); err != nil {
			t.Fatal(err)
		}
	}
	sendOutputEvent(t, p, r, event("run", "TestPass", ""))
	sendOutputEvent(t, p, r, event("output", "TestPass", "unfinished-secret\n"))
	r.Finish()
	p.Finish()
	if got := out.String(); strings.Contains(got, "secret") || got != "" {
		t.Fatalf("untrusted/incomplete output leaked: %q", got)
	}
	if !r.Degraded() || p.Snapshot().Status != "partial" {
		t.Fatal("lost malformed, oversized or incomplete indication")
	}
}

func TestMetricsOutputOverlongNamesAndJSONFlagOverrides(t *testing.T) {
	var out bytes.Buffer
	p, r := outputFixture(&out, false)
	longName := "TestPass/" + strings.Repeat("n", 513)
	sendOutputEvent(t, p, r, event("run", longName, ""))
	sendOutputEvent(t, p, r, event("output", longName, "long-name-secret\n"))
	sendOutputEvent(t, p, r, event("pass", longName, ""))
	sendOutputEvent(t, p, r, event("pass", "", ""))
	r.Finish()
	p.Finish()
	if !r.Degraded() || strings.Contains(out.String(), "long-name-secret") {
		t.Fatalf("name limit lost: %q", out.String())
	}
	if hasJSONFlag([]string{"go", "test", "-json", "-json=false"}) {
		t.Fatal("later false JSON flag was ignored")
	}
	if !hasJSONFlag([]string{"go", "test", "-json=false", "-json=1"}) {
		t.Fatal("later true JSON flag was ignored")
	}
	if hasJSONFlag([]string{"go", "test", "-args", "-json"}) {
		t.Fatal("test-binary flag was mistaken for Go JSON")
	}
}

func TestMetricsOutputRepeatedTestWithoutTerminalDropsOldOutput(t *testing.T) {
	var out bytes.Buffer
	p, r := outputFixture(&out, false)
	for _, e := range []map[string]any{
		event("start", "", ""), event("run", "TestPass", ""), event("output", "TestPass", "prior-passing-secret\n"),
		event("run", "TestFail", ""), event("output", "TestFail", "parallel-failure-details\n"),
		event("run", "TestPass", ""), event("output", "TestPass", "new-failure-details\n"), event("fail", "TestPass", ""),
		event("fail", "TestFail", ""), event("fail", "", ""),
	} {
		sendOutputEvent(t, p, r, e)
	}
	r.Finish()
	p.Finish()
	got := out.String()
	if strings.Contains(got, "prior-passing-secret") || !strings.Contains(got, "new-failure-details") || !strings.Contains(got, "parallel-failure-details") {
		t.Fatalf("repeated test mixed associations or dropped parallel output: %q", got)
	}
	if !r.Degraded() {
		t.Fatal("missing terminal event was not reported")
	}
}

func TestMetricsOutputMalformedResetsAssociationsUntilNewRun(t *testing.T) {
	var out bytes.Buffer
	p, r := outputFixture(&out, false)
	for _, e := range []map[string]any{event("start", "", ""), event("run", "TestPass", ""), event("output", "TestPass", "old-secret\n")} {
		sendOutputEvent(t, p, r, e)
	}
	if _, err := io.WriteString(io.MultiWriter(p, r), "{malformed secret\n"); err != nil {
		t.Fatal(err)
	}
	for _, e := range []map[string]any{
		event("output", "TestPass", "unassociated-secret\n"), event("fail", "TestPass", ""),
		event("run", "TestFail", ""), event("output", "TestFail", "new-validated-failure\n"), event("fail", "TestFail", ""), event("fail", "", ""),
	} {
		sendOutputEvent(t, p, r, e)
	}
	r.Finish()
	p.Finish()
	got := out.String()
	if strings.Contains(got, "old-secret") || strings.Contains(got, "unassociated-secret") || strings.Contains(got, "malformed secret") || !strings.Contains(got, "new-validated-failure") {
		t.Fatalf("malformed stream mixed associations: %q", got)
	}
	if !r.Degraded() {
		t.Fatal("malformed stream not marked partial")
	}
}

func TestMetricsOutputRedactionFallbackDoesNotAliasUnknownTests(t *testing.T) {
	var out bytes.Buffer
	p, r := outputFixture(&out, false)
	p.mu.Lock()
	p.noRandom = true
	p.mu.Unlock()
	for _, e := range []map[string]any{
		event("start", "", ""), event("run", "TestUnknownA/private-a", ""), event("output", "TestUnknownA/private-a", "passing-secret\n"),
		event("run", "TestUnknownB/private-b", ""), event("output", "TestUnknownB/private-b", "independent-failure\n"),
		event("fail", "TestUnknownB/private-b", ""), event("pass", "TestUnknownA/private-a", ""), event("fail", "", ""),
	} {
		sendOutputEvent(t, p, r, e)
	}
	r.Finish()
	p.Finish()
	got := out.String()
	if strings.Contains(got, "passing-secret") || strings.Contains(got, "private-a") || strings.Contains(got, "private-b") || !strings.Contains(got, "independent-failure") {
		t.Fatalf("redacted test names aliased: %q", got)
	}
}

func TestMetricsOutputRestartedPackageDoesNotDiscardParallelPackage(t *testing.T) {
	var out bytes.Buffer
	p, r := outputFixture(&out, false)
	other := "example.test/parallel"
	e := func(action, test, text string) map[string]any {
		m := event(action, test, text)
		m["Package"] = other
		return m
	}
	for _, x := range []map[string]any{
		event("start", "", ""), event("run", "TestPass", ""), event("output", "TestPass", "prior-package-secret\n"),
		e("start", "", ""), e("run", "TestFail", ""), e("output", "TestFail", "parallel-package-failure\n"),
		event("start", "", ""), e("fail", "TestFail", ""), e("fail", "", ""), event("pass", "", ""),
	} {
		sendOutputEvent(t, p, r, x)
	}
	r.Finish()
	p.Finish()
	got := out.String()
	if strings.Contains(got, "prior-package-secret") || !strings.Contains(got, "parallel-package-failure") {
		t.Fatalf("package reset crossed boundaries: %q", got)
	}
	if !r.Degraded() {
		t.Fatal("restarted package was not marked partial")
	}
}

func TestMetricsOutputMalformedBuildEventNeverLeaks(t *testing.T) {
	var out bytes.Buffer
	p, r := outputFixture(&out, false)
	sendOutputEvent(t, p, r, map[string]any{"Action": "build-output", "Output": "malformed-build-secret\n"})
	sendOutputEvent(t, p, r, event("start", "", ""))
	sendOutputEvent(t, p, r, map[string]any{"Action": "build-output", "ImportPath": outputFixturePackage, "Output": "valid-compiler-diagnostic\n"})
	sendOutputEvent(t, p, r, event("fail", "", ""))
	r.Finish()
	p.Finish()
	got := out.String()
	if strings.Contains(got, "malformed-build-secret") || !strings.Contains(got, "valid-compiler-diagnostic") || !r.Degraded() {
		t.Fatalf("build event validation failed: %q", got)
	}
}

func TestMetricsOutputBoundsGlobalBuffersAndReportsTruncation(t *testing.T) {
	var out bytes.Buffer
	p, r := outputFixture(&out, false)
	sendOutputEvent(t, p, r, event("start", "", ""))
	for i := range 400 {
		name := fmt.Sprintf("TestPass/%04d", i)
		sendOutputEvent(t, p, r, event("run", name, ""))
		sendOutputEvent(t, p, r, event("output", name, strings.Repeat("x", 4096)+"sensitive-passing-marker\n"))
	}
	if !r.Degraded() || r.used > outputLimit || len(r.pending) > outputBufferLimit || len(r.packages) > outputPackageLimit {
		t.Fatalf("buffer limits not enforced: used=%d pending=%d packages=%d", r.used, len(r.pending), len(r.packages))
	}
	for i := range 400 {
		sendOutputEvent(t, p, r, event("pass", fmt.Sprintf("TestPass/%04d", i), ""))
	}
	sendOutputEvent(t, p, r, event("pass", "", ""))
	r.Finish()
	p.Finish()
	if strings.Contains(out.String(), "sensitive-passing-marker") {
		t.Fatal("passing output leaked after buffer limit")
	}
	if !r.Degraded() {
		t.Fatal("truncation not reported")
	}
}

func TestMetricsOutputBoundedFailureAndBuildDiagnostics(t *testing.T) {
	var out bytes.Buffer
	p, r := outputFixture(&out, false)
	sendOutputEvent(t, p, r, map[string]any{"Action": "build-output", "ImportPath": outputFixturePackage, "Output": "compiler diagnostic\n"})
	sendOutputEvent(t, p, r, event("run", "TestFail", ""))
	for range 2 {
		sendOutputEvent(t, p, r, event("output", "TestFail", strings.Repeat("d", 60000)))
	}
	sendOutputEvent(t, p, r, event("fail", "TestFail", ""))
	sendOutputEvent(t, p, r, event("fail", "", ""))
	r.Finish()
	p.Finish()
	if got := out.String(); !strings.Contains(got, "compiler diagnostic") || !strings.Contains(got, "failure output truncated") || len(got) > outputEntryLimit+1024 {
		t.Fatalf("missing/beyond-limit diagnostics; len=%d", len(got))
	}
	if !r.Degraded() {
		t.Fatal("truncated failure not marked degraded")
	}
}

func TestMetricsOutputVerboseOptInAndWriterErrors(t *testing.T) {
	var out bytes.Buffer
	p, r := outputFixture(&out, explicitVerbose([]string{"go", "test", "-json", "-v"}))
	for _, e := range []map[string]any{event("run", "TestPass", ""), event("output", "TestPass", "opted-in-output\n"), event("pass", "TestPass", ""), event("pass", "", "")} {
		sendOutputEvent(t, p, r, e)
	}
	r.Finish()
	p.Finish()
	if got := out.String(); !strings.Contains(got, "opted-in-output") || strings.Contains(got, `"Action"`) {
		t.Fatalf("verbose rendering incorrect: %q", got)
	}
	if explicitVerbose([]string{"-v", "-v=false"}) {
		t.Fatal("explicit quiet override ignored")
	}
	p, r = outputFixture(failOutput{}, false)
	b, _ := json.Marshal(event("pass", "", ""))
	b = append(b, '\n')
	if _, err := io.MultiWriter(p, r).Write(b); err == nil || !strings.Contains(err.Error(), "fixture output failure") {
		t.Fatalf("original output error hidden: %v", err)
	}
}

type blockingOutput struct {
	started chan struct{}
	release chan struct{}
}

func (w *blockingOutput) Write(b []byte) (int, error) {
	close(w.started)
	<-w.release
	return len(b), nil
}

func TestMetricsOutputFinishDoesNotWaitForBlockedWriter(t *testing.T) {
	w := &blockingOutput{started: make(chan struct{}), release: make(chan struct{})}
	p, r := outputFixture(w, false)
	done := make(chan struct{})
	go func() {
		b, _ := json.Marshal(event("pass", "", ""))
		b = append(b, '\n')
		_, _ = io.MultiWriter(p, r).Write(b)
		close(done)
	}()
	select {
	case <-w.started:
	case <-time.After(time.Second):
		t.Fatal("output did not start")
	}
	start := time.Now()
	r.Finish()
	if elapsed := time.Since(start); elapsed > 100*time.Millisecond {
		t.Errorf("Finish blocked on original writer: %s", elapsed)
	}
	if !r.Degraded() {
		t.Error("unfinished writer was not reported")
	}
	close(w.release)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("writer did not finish after release")
	}
}
