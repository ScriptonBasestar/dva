// Command cimetrics collects optional, project-owned CI diagnostics.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/ScriptonBasestar/dva/tools/internal/cimetrics"
)

type result struct {
	code   int
	signal os.Signal
}

func main() {
	r := execute(os.Args[1:], os.Stdout, os.Stderr)
	if r.signal != nil {
		rethrow(r.signal)
	}
	os.Exit(r.code)
}

func execute(args []string, stdout, stderr io.Writer) result {
	if len(args) == 0 {
		_, _ = fmt.Fprintln(stderr, "usage: cimetrics run|summarize [options]")
		return result{code: 2}
	}
	switch args[0] {
	case "run":
		return runCommand(args[1:], stdout, stderr)
	case "summarize":
		return summarizeCommand(args[1:], stdout, stderr)
	default:
		_, _ = fmt.Fprintln(stderr, "ci-metrics: unknown command")
		return result{code: 2}
	}
}

func runCommand(args []string, stdout, stderr io.Writer) result {
	f := flag.NewFlagSet("run", flag.ContinueOnError)
	f.SetOutput(stderr)
	check := f.String("check", "", "fixed project check ID")
	if err := f.Parse(args); err != nil {
		return result{code: 2}
	}
	if *check == "" || len(f.Args()) == 0 {
		_, _ = fmt.Fprintln(stderr, "ci-metrics: run needs --check and a command")
		return result{code: 2}
	}
	ctx, stop, received := signalContext()
	err := cimetrics.Run(ctx, *check, f.Args(), stdout, stderr)
	stop()
	if sig := received(); sig != nil {
		return result{code: 1, signal: sig}
	}
	if err == nil {
		return result{}
	}
	_, _ = fmt.Fprintf(stderr, "ci-metrics: check failed: %v\n", err)
	if sig := childSignal(err); sig != nil {
		return result{code: 1, signal: sig}
	}
	var child *exec.ExitError
	if errors.As(err, &child) && child.ExitCode() >= 0 {
		return result{code: child.ExitCode()}
	}
	return result{code: 1}
}

func summarizeCommand(args []string, stdout, stderr io.Writer) result {
	f := flag.NewFlagSet("summarize", flag.ContinueOnError)
	f.SetOutput(stderr)
	check := f.String("check", "", "unit or integration")
	input := f.String("input", "", "existing Go JSONL file")
	output := f.String("output", "", "new summary file; omit for stdout")
	if err := f.Parse(args); err != nil {
		return result{code: 2}
	}
	if (*check != "unit" && *check != "integration") || *input == "" || len(f.Args()) != 0 {
		_, _ = fmt.Fprintln(stderr, "ci-metrics: summarize needs --check unit|integration and --input")
		return result{code: 2}
	}
	r, err := cimetrics.SummarizeFile(*check, *input)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "ci-metrics: summary unavailable")
		return result{}
	}
	if *output == "" {
		err = json.NewEncoder(stdout).Encode(r)
	} else {
		err = cimetrics.WriteHostedSummary(*output, r)
	}
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "ci-metrics: summary storage degraded")
		return result{}
	}
	if r.Status != "complete" {
		_, _ = fmt.Fprintln(stderr, "ci-metrics: summary is partial or unavailable")
	}
	return result{}
}
