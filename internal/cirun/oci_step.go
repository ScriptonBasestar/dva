package cirun

import (
	"context"
	"errors"
	"io"

	"github.com/ScriptonBasestar/dva/internal/cirun/ociexec"
	"github.com/ScriptonBasestar/dva/internal/config"
)

type ociScopeKey struct{}
type ociScope struct{ state, runID string }

func runOCIStep(ctx context.Context, root string, step config.CIStep, out io.Writer) (*ociexec.Report, error) {
	scope, ok := ctx.Value(ociScopeKey{}).(ociScope)
	if !ok {
		return nil, errors.New("OCI step must run inside its owning CI execution")
	}
	result, err := ociexec.Run(ctx, ociexec.Options{Root: root, StateDir: scope.state, RunID: scope.runID, Step: step.Name, Config: *step.OCI, Output: out})
	return &result, err
}
