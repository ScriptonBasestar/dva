package lifecycle

import (
	"context"
	"strings"
	"testing"
)

type downObservationExecutor struct {
	*fakeChildExecutor
	queryErrors []error
}

func (e *downObservationExecutor) IsUp(ctx context.Context, child *ExecutionPlan) (bool, error) {
	e.queryErrors = append(e.queryErrors, ctx.Err())
	if err := ctx.Err(); err != nil {
		return false, err
	}
	return e.fakeChildExecutor.IsUp(ctx, child)
}

func TestCompositionDownDoesNotTreatCancelledObservationAsRunning(t *testing.T) {
	plan := resolveTestComposition(t)
	executor := &downObservationExecutor{fakeChildExecutor: newFakeChildExecutor()}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	report, err := newTestCompositionOrchestrator(t, plan, executor).Down(ctx, CompositionDownOptions{})
	if err != nil || report.Outcome != CompositionOutcomeDown {
		t.Fatalf("best-effort down changed: report=%+v err=%v", report, err)
	}
	if len(executor.queryErrors) != len(plan.Entries) {
		t.Fatalf("observed %d children, want %d", len(executor.queryErrors), len(plan.Entries))
	}
	for _, err := range executor.queryErrors {
		if err != nil {
			t.Fatalf("cleanup observation inherited cancellation: %v", err)
		}
	}
	for _, call := range executor.recorded() {
		if strings.HasPrefix(call, "down:") {
			t.Fatalf("cancelled read caused teardown of an already-down child: %v", executor.recorded())
		}
	}
}
