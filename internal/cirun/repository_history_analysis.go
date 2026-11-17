package cirun

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

func gitCommonDirectory(ctx context.Context, root string) (string, error) {
	common, err := gitStdout(ctx, root, "rev-parse", "--git-common-dir")
	if err != nil {
		return "", fmt.Errorf("locate git common directory: %w", gitError(err))
	}
	dir := strings.TrimSpace(string(common))
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(root, dir)
	}
	dir, err = filepath.EvalSymlinks(dir)
	if err != nil {
		return "", fmt.Errorf("canonicalize git common directory: %w", err)
	}
	return dir, nil
}

// RepositoryLocalID identifies a local clone and its linked worktrees only.
func RepositoryLocalID(ctx context.Context, root string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	dir, err := gitCommonDirectory(ctx, root)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256([]byte(dir))
	return hex.EncodeToString(sum[:]), nil
}

type RepositoryHistoryRun struct {
	Root          string         `json:"root"`
	Commit        string         `json:"commit,omitempty"`
	Observability *Observability `json:"observability,omitempty"`
	HistoryRun
}

type RepositoryHistoryAnalysis struct {
	RepositoryLocalID string                 `json:"repository_local_id"`
	Profile           string                 `json:"profile"`
	Comparability     string                 `json:"comparability"`
	ProfileUnassigned int                    `json:"profile_unassigned_repository_id"`
	Runs              []RepositoryHistoryRun `json:"runs"`
	Observations      []HistoryObservation   `json:"observations"`
}

func AnalyzeRepositoryHistory(reports []Report, id, profile string) RepositoryHistoryAnalysis {
	a := RepositoryHistoryAnalysis{RepositoryLocalID: id, Profile: profile, Comparability: "unknown: complete host, configuration and cache comparability is not established", Runs: []RepositoryHistoryRun{}, Observations: []HistoryObservation{}}
	remaining := maxHistoryStepObservations
	for _, r := range reports {
		if r.Profile != profile {
			continue
		}
		if r.Observability == nil || r.Observability.RepositoryLocalID == "" {
			a.ProfileUnassigned++
			continue
		}
		if r.Observability.RepositoryLocalID != id {
			continue
		}
		a.Runs = append(a.Runs, RepositoryHistoryRun{Root: r.Root, Commit: r.Attestation.Commit, Observability: r.Observability, HistoryRun: projectHistoryRun(r)})
		a.Observations = append(a.Observations, projectHistoryObservation(r, &remaining))
	}
	return a
}
