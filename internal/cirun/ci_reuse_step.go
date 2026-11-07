package cirun

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/ScriptonBasestar/dva/internal/config"
)

func runReusableStep(ctx context.Context, root string, step config.CIStep, inherited []string, workers int, source string, out io.Writer, startedAt time.Time) (r StepReport, ret error) {
	if step.Reuse == nil {
		return runStep(ctx, root, step, inherited, workers, source, out, startedAt)
	}
	r = StepReport{Name: step.Name, Status: "running", StartedAt: startedAt}
	stepCtx := ctx
	var cancel context.CancelFunc
	if step.Timeout != "" {
		d, err := time.ParseDuration(step.Timeout)
		if err != nil {
			return reusePreparationFailure(stepCtx, r, err)
		}
		stepCtx, cancel = context.WithTimeout(ctx, d)
		defer cancel()
	}
	runtime, _ := ctx.Value(reuseRuntimeKey{}).(reuseRuntime)
	if runtime.fresh {
		_, _ = fmt.Fprintf(out, "ci step %s reuse=BYPASSED reason=fresh\n", step.Name)
		r, ret = runStep(stepCtx, root, step, inherited, workers, source, out, startedAt)
		r.Reuse = &ReuseReport{Disposition: "bypassed", Reason: "fresh"}
		return r, ret
	}
	actualEnv := ciEnv(inherited, step.Environment, workers)
	for _, key := range step.Reuse.UnlessEnvironment {
		if environmentMap(actualEnv)[key] != "" {
			_, _ = fmt.Fprintf(out, "ci step %s reuse=BYPASSED reason=environment:%s\n", step.Name, key)
			r, ret = runStep(stepCtx, root, step, inherited, workers, source, out, startedAt)
			r.Reuse = &ReuseReport{Disposition: "bypassed", Reason: "environment:" + key}
			return r, ret
		}
	}
	plan, err := reuseFor(stepCtx, root, step, actualEnv)
	if err != nil {
		return reusePreparationFailure(stepCtx, r, err)
	}
	maxAge, _ := time.ParseDuration(step.Reuse.MaxAge) // config validation guarantees this.
	if stepCtx.Err() == nil {
		if record, reason := loadReuse(plan, maxAge, time.Now(), root); record != nil {
			_, _ = fmt.Fprintf(out, "ci step %s reuse=REUSED original_run=%s\n", step.Name, record.RunID)
			finishedAt := time.Now()
			return StepReport{Name: step.Name, Status: "succeeded", StartedAt: startedAt, FinishedAt: finishedAt, Duration: finishedAt.Sub(startedAt), Reuse: &ReuseReport{Key: plan.key, Disposition: "reused", OriginalRunID: record.RunID, FinishedAt: record.FinishedAt}}, nil
		} else {
			_, _ = fmt.Fprintf(out, "ci step %s reuse=EXECUTED reason=%s\n", step.Name, reason)
		}
	}
	r, ret = runStep(stepCtx, root, step, inherited, workers, source, out, startedAt)
	if r.Reuse == nil {
		disposition := "executed"
		reason := ""
		r.Reuse = &ReuseReport{Key: plan.key, Disposition: disposition, Reason: reason}
	}
	if ret != nil {
		return r, ret
	}
	if err := stepCtx.Err(); err != nil {
		return reusePreparationFailure(stepCtx, r, err)
	}
	after, err := reuseFor(stepCtx, root, step, actualEnv)
	if err != nil {
		return reuseStepFailure(r, &reuseIntegrityError{fmt.Errorf("ci step %q verify reuse inputs: %w", step.Name, err)})
	}
	if after.key != plan.key {
		return reuseStepFailure(r, &reuseIntegrityError{fmt.Errorf("ci step %q changed declared reuse inputs during execution", step.Name)})
	}
	if err := saveReuse(plan, root, runtime.runID, time.Now()); err != nil {
		return reuseStepFailure(r, &reuseIntegrityError{fmt.Errorf("ci step %q cache successful result: %w", step.Name, err)})
	}
	return r, nil
}

// reusePreparationFailure classifies a failure before or around the step command. A reuse
// probe or input command the step timeout kills returns only "signal: killed", which does not
// wrap the context error, so the step context is consulted too, as runStep does.
func reusePreparationFailure(ctx context.Context, r StepReport, err error) (StepReport, error) {
	r.Error = err.Error()
	r.FinishedAt = time.Now()
	r.Duration = r.FinishedAt.Sub(r.StartedAt)
	switch {
	case errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded):
		r.Status = "timed_out"
	case errors.Is(err, context.Canceled) || errors.Is(ctx.Err(), context.Canceled):
		r.Status = "canceled"
	default:
		r.Status = "failed"
	}
	return r, err
}

func reuseStepFailure(r StepReport, err error) (StepReport, error) {
	r.Status, r.Error, r.FinishedAt = "failed", err.Error(), time.Now()
	r.Duration = r.FinishedAt.Sub(r.StartedAt)
	return r, err
}
