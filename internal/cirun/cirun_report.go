package cirun

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/user"
	"path/filepath"
	"time"

	"github.com/ScriptonBasestar/dva/internal/config"
)

var ErrBusy = errors.New("a conflicting dva ci run is active")

// Conflict identifies the lock preventing admission. RunID is omitted when unknown.
type Conflict struct {
	Kind  string `json:"kind"`
	Key   string `json:"key"`
	RunID string `json:"run_id,omitempty"`
}

type BusyError struct {
	ActiveRunID string
	Conflict    Conflict
}

func (e *BusyError) Error() string {
	message := fmt.Sprintf("%s (%s %q)", ErrBusy, e.Conflict.Kind, e.Conflict.Key)
	if e.ActiveRunID != "" {
		message += fmt.Sprintf(" (active run %s)", e.ActiveRunID)
	}
	return message
}
func (e *BusyError) Unwrap() error { return ErrBusy }

type Options struct {
	Root, ProfileName, StateDir string
	// Subproject is the declared child named by `dva ci --subproject`. Run is
	// the only source of config.CISubprojectEnv for steps and records it in the
	// receipt, so the receipt names the scope the steps actually received.
	Subproject    string
	Profile       config.CIProfile
	Env           []string
	Output        io.Writer
	lockDirectory string // test seam; production callers cannot set it.
}

type StepReport struct {
	Name         string        `json:"name"`
	Status       string        `json:"status"`
	Error        string        `json:"error,omitempty"`
	StartedAt    time.Time     `json:"started_at"`
	FinishedAt   time.Time     `json:"finished_at"`
	Duration     time.Duration `json:"duration"`
	WorkerBudget *WorkerBudget `json:"worker_budget,omitempty"`
}

// WorkerBudget records the numeric worker limits actually supplied to a CI step.
// Sources are fixed runtime labels; it intentionally excludes all other environment values.
type WorkerBudget struct {
	ToolJobs              int    `json:"tool_jobs,omitempty"`
	Source                string `json:"source,omitempty"`
	DVAJobs               int    `json:"dva_ci_jobs"`
	CargoBuildJobs        int    `json:"cargo_build_jobs"`
	CargoBuildJobsSource  string `json:"cargo_build_jobs_source,omitempty"`
	RustTestThreads       int    `json:"rust_test_threads"`
	RustTestThreadsSource string `json:"rust_test_threads_source,omitempty"`
}

// Attestation describes the Git inputs a run measured. Reason explains an
// unavailable attestation. Commit is HEAD at run start and Dirty whether the
// root's inputs differed from it; both are absent when unavailable, when HEAD
// is unborn, and in receipts written before they existed.
type Attestation struct {
	Available bool   `json:"available"`
	Reason    string `json:"reason,omitempty"`
	Before    string `json:"before,omitempty"`
	After     string `json:"after,omitempty"`
	Commit    string `json:"commit,omitempty"`
	Dirty     *bool  `json:"dirty,omitempty"`
}

type Report struct {
	ID, Root, Profile, Status, LogPath string
	Subproject                         string
	Error                              string
	StartedAt, FinishedAt              time.Time
	Duration                           time.Duration
	Steps                              []StepReport
	Attestation                        Attestation
	Conflict                           *Conflict
}

func (r Report) MarshalJSON() ([]byte, error) {
	type wire struct {
		ID          string       `json:"id"`
		Root        string       `json:"root"`
		Profile     string       `json:"profile"`
		Subproject  string       `json:"subproject,omitempty"`
		Status      string       `json:"status"`
		LogPath     string       `json:"log_path"`
		Error       string       `json:"error,omitempty"`
		StartedAt   time.Time    `json:"started_at"`
		FinishedAt  time.Time    `json:"finished_at"`
		Duration    string       `json:"duration"`
		Steps       []StepReport `json:"steps"`
		Attestation Attestation  `json:"attestation"`
		Conflict    *Conflict    `json:"conflict,omitempty"`
	}
	return json.Marshal(wire{r.ID, r.Root, r.Profile, r.Subproject, r.Status, r.LogPath, r.Error, r.StartedAt, r.FinishedAt, r.Duration.String(), r.Steps, r.Attestation, r.Conflict})
}

func (r *Report) UnmarshalJSON(data []byte) error {
	type wire struct {
		ID          string       `json:"id"`
		Root        string       `json:"root"`
		Profile     string       `json:"profile"`
		Subproject  string       `json:"subproject,omitempty"`
		Status      string       `json:"status"`
		LogPath     string       `json:"log_path"`
		Error       string       `json:"error,omitempty"`
		StartedAt   time.Time    `json:"started_at"`
		FinishedAt  time.Time    `json:"finished_at"`
		Duration    string       `json:"duration"`
		Steps       []StepReport `json:"steps"`
		Attestation Attestation  `json:"attestation"`
		Conflict    *Conflict    `json:"conflict,omitempty"`
	}
	var in wire
	if err := json.Unmarshal(data, &in); err != nil {
		return err
	}
	d, err := time.ParseDuration(in.Duration)
	if err != nil {
		return err
	}
	*r = Report{ID: in.ID, Root: in.Root, Profile: in.Profile, Subproject: in.Subproject, Status: in.Status, LogPath: in.LogPath, Error: in.Error, StartedAt: in.StartedAt, FinishedAt: in.FinishedAt, Duration: d, Steps: in.Steps, Attestation: in.Attestation, Conflict: in.Conflict}
	return nil
}

func stateDirectory(dir string) (string, error) {
	if dir != "" {
		return dir, nil
	}
	u, err := user.Current()
	if err != nil {
		return "", err
	}
	// Do not use HOME/XDG_CACHE_HOME: sessions must share UID-owned resource locks.
	return filepath.Join(u.HomeDir, ".local", "state", "dva", "ci"), nil
}
