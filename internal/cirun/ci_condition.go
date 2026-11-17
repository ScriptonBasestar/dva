package cirun

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/ScriptonBasestar/dva/internal/config"
)

const conditionOutputLimit = 4096

type conditionOutput struct{ bytes.Buffer }

func (w *conditionOutput) Write(p []byte) (int, error) {
	if w.Len()+len(p) > conditionOutputLimit {
		return 0, errors.New("condition output limit exceeded")
	}
	return w.Buffer.Write(p)
}

func conditionSelection(data []byte) (bool, error) {
	invalid := errors.New("condition returned invalid selection (output withheld)")
	decoder := json.NewDecoder(bytes.NewReader(data))
	start, err := decoder.Token()
	if err != nil || start != json.Delim('{') {
		return false, invalid
	}
	key, err := decoder.Token()
	if err != nil || key != "selected" {
		return false, invalid
	}
	var value json.RawMessage
	if err := decoder.Decode(&value); err != nil || (string(value) != "true" && string(value) != "false") {
		return false, invalid
	}
	end, err := decoder.Token()
	if err != nil || end != json.Delim('}') {
		return false, invalid
	}
	if _, err := decoder.Token(); err != io.EOF {
		return false, invalid
	}
	return string(value) == "true", nil
}

func runCondition(ctx context.Context, root string, step config.CIStep, inherited []string, workers int) (bool, error) {
	condition := step.Condition
	duration, err := time.ParseDuration(condition.Timeout)
	if err != nil || duration <= 0 || strings.TrimSpace(condition.Run) == "" {
		return false, errors.New("invalid condition declaration")
	}
	root, err = filepath.Abs(root)
	if err != nil {
		return false, errors.New("condition root unavailable")
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return false, errors.New("condition root unavailable")
	}
	workdir := root
	if step.Workdir != "" {
		workdir = filepath.Join(root, step.Workdir)
		if filepath.IsAbs(step.Workdir) {
			workdir = step.Workdir
		}
	}
	workdir, err = filepath.EvalSymlinks(workdir)
	if err != nil {
		return false, errors.New("condition workdir unavailable")
	}
	rel, err := filepath.Rel(root, workdir)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return false, errors.New("condition workdir escapes root")
	}
	conditionCtx, cancel := context.WithTimeout(ctx, duration)
	defer cancel()
	if err := conditionCtx.Err(); err != nil {
		return false, err
	}
	output := &conditionOutput{}
	cmd := exec.Command("/bin/sh", "-eu", "-c", condition.Run)
	cmd.Dir, cmd.Env, cmd.Stdout, cmd.Stderr = workdir, ciEnv(inherited, step.Environment, workers), output, io.Discard
	cmd.WaitDelay = time.Second
	if err := startProcess(cmd); err != nil {
		return false, errors.New("condition could not start")
	}
	pid := cmd.Process.Pid
	waited := make(chan error, 1)
	go func() { waited <- cmd.Wait() }()
	select {
	case err := <-waited:
		cleanupGroup(pid)
		if conditionCtx.Err() != nil {
			return false, conditionCtx.Err()
		}
		if err != nil {
			return false, errors.New("condition failed (output withheld)")
		}
	case <-conditionCtx.Done():
		terminateProcessGroup(pid, waited)
		return false, conditionCtx.Err()
	}
	return conditionSelection(output.Bytes())
}

func runConditionedStep(ctx context.Context, root string, step config.CIStep, inherited []string, workers int, source string, out io.Writer, startedAt time.Time) (StepReport, error) {
	if step.Condition == nil {
		return runReusableStep(ctx, root, step, inherited, workers, source, out, startedAt)
	}
	report := StepReport{Name: step.Name, Status: "running", StartedAt: startedAt}
	sum := sha256.Sum256([]byte(step.Condition.Run + "\x00" + step.Condition.Timeout))
	condition := &ConditionReport{DefinitionSHA256: hex.EncodeToString(sum[:]), Timeout: step.Condition.Timeout, Outcome: "failed"}
	report.Condition = condition
	stepCtx := ctx
	var cancel context.CancelFunc
	if step.Timeout != "" {
		d, err := time.ParseDuration(step.Timeout)
		if err != nil || d <= 0 {
			return reusePreparationFailure(ctx, report, errors.New("invalid step timeout"))
		}
		stepCtx, cancel = context.WithDeadline(ctx, startedAt.Add(d))
		defer cancel()
	}
	root, err := filepath.Abs(root)
	if err == nil {
		root, err = filepath.EvalSymlinks(root)
	}
	if err != nil {
		return reusePreparationFailure(stepCtx, report, errors.New("condition root unavailable"))
	}
	selected, err := runCondition(stepCtx, root, step, inherited, workers)
	if err != nil {
		return reusePreparationFailure(stepCtx, report, err)
	}
	condition.Selected = &selected
	if !selected {
		condition.Outcome = "not-selected"
		report.Status, report.SkipReason = "skipped", "condition-false"
		report.FinishedAt = time.Now()
		report.Duration = report.FinishedAt.Sub(startedAt)
		return report, nil
	}
	condition.Outcome = "selected"
	report, err = runReusableStep(stepCtx, root, step, inherited, workers, source, out, startedAt)
	report.Condition = condition
	return report, err
}
