package cirun

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/user"
	"path/filepath"
	"time"

	"github.com/ScriptonBasestar/dva/internal/cirun/ociexec"
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
	Subproject string
	Profile    config.CIProfile
	// Fresh bypasses declared step result caches. The profile may also require it.
	Fresh bool
	Env   []string
	// WorkerEnvironment marks root-owned Cargo/Rust declarations. They remain
	// explicit even when numerically equal to an inherited runtime default.
	WorkerEnvironment map[string]bool
	Output            io.Writer
	lockDirectory     string // test seam; production callers cannot set it.
	resourceDirectory string // test seam; production callers share host state.
}

type StepReport struct {
	OutputObservation *OutputObservation    `json:"output_observation,omitempty"`
	Admission         *AdmissionObservation `json:"admission,omitempty"`
	OSWaitResources   *OSWaitResources      `json:"os_wait_resources,omitempty"`
	Workdir           string                `json:"workdir,omitempty"`
	Condition         *ConditionReport      `json:"condition,omitempty"`
	Reuse             *ReuseReport          `json:"reuse,omitempty"`
	CompositionBudget *CompositionBudget    `json:"composition_budget,omitempty"`
	OCI               *ociexec.Report       `json:"oci,omitempty"`
	Name              string                `json:"name"`
	Status            string                `json:"status"`
	Error             string                `json:"error,omitempty"`
	SkipReason        string                `json:"skip_reason,omitempty"`
	StartedAt         time.Time             `json:"started_at"`
	FinishedAt        time.Time             `json:"finished_at"`
	Duration          time.Duration         `json:"duration"`
	HostSlotWait      time.Duration         `json:"host_slot_wait,omitempty"`
	WorkerBudget      *WorkerBudget         `json:"worker_budget,omitempty"`
}

// OSWaitResources contains usage returned when DVA waits for the step process.
// The exact child-process coverage depends on the operating system.
type OSWaitResources struct {
	CollectionStatus string `json:"collection_status"`
	UserCPU          string `json:"user_cpu,omitempty"`
	SystemCPU        string `json:"system_cpu,omitempty"`
	PeakRSSBytes     *int64 `json:"peak_rss_bytes,omitempty"`
}

type ConditionReport struct {
	DefinitionSHA256 string `json:"definition_sha256"`
	Timeout          string `json:"timeout"`
	Outcome          string `json:"outcome"`
	Selected         *bool  `json:"selected,omitempty"`
}

// ReuseReport distinguishes a command that ran from a valid prior PASS. A
// reused step remains succeeded so dependency handling stays unchanged.
type ReuseReport struct {
	Key           string    `json:"key,omitempty"`
	Disposition   string    `json:"disposition"`
	OriginalRunID string    `json:"original_run_id,omitempty"`
	FinishedAt    time.Time `json:"finished_at,omitzero"`
	Reason        string    `json:"reason,omitempty"`
}

type CompositionBudget struct {
	Deadline  string `json:"deadline"`
	JobsLimit int    `json:"jobs_limit"`
}

type CompositionParent struct {
	RunID string `json:"run_id"`
	Root  string `json:"root"`
	Step  string `json:"step"`
}

// WorkerBudget records the numeric worker limits actually supplied to a CI step.
// Sources are fixed runtime labels; it intentionally excludes all other environment values.
type WorkerBudget struct {
	GoMaxProcs            *ObservedWorker `json:"go_max_procs,omitempty"`
	GoPackageJobs         *ObservedWorker `json:"go_package_jobs,omitempty"`
	PytestWorkers         *ObservedWorker `json:"pytest_workers,omitempty"`
	PytestMaxWorkers      *ObservedWorker `json:"pytest_max_workers,omitempty"`
	ToolJobs              int             `json:"tool_jobs,omitempty"`
	Source                string          `json:"source,omitempty"`
	DVAJobs               int             `json:"dva_ci_jobs"`
	CargoBuildJobs        int             `json:"cargo_build_jobs"`
	CargoBuildJobsSource  string          `json:"cargo_build_jobs_source,omitempty"`
	RustTestThreads       int             `json:"rust_test_threads"`
	RustTestThreadsSource string          `json:"rust_test_threads_source,omitempty"`
}

// ObservedWorker is an explicit numeric setting, not proof that a tool used it.
type ObservedWorker struct {
	Value  int    `json:"value"`
	Source string `json:"source"`
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
	Admission                          *AdmissionObservation
	Observability                      *Observability
	NativeStages                       *NativeStageObservation
	ID, Root, Profile, Status, LogPath string
	Subproject                         string
	Error                              string
	SkipReason                         string
	StartedAt, FinishedAt              time.Time
	Duration                           time.Duration
	Steps                              []StepReport
	Attestation                        Attestation
	Conflict                           *Conflict
	Parent                             *CompositionParent
	CompositionDepth                   int
	Fresh                              bool
}

func (r Report) MarshalJSON() ([]byte, error) {
	type wire struct {
		Admission        *AdmissionObservation   `json:"admission,omitempty"`
		Observability    *Observability          `json:"observability,omitempty"`
		NativeStages     *NativeStageObservation `json:"native_stages,omitempty"`
		ID               string                  `json:"id"`
		Root             string                  `json:"root"`
		Profile          string                  `json:"profile"`
		Subproject       string                  `json:"subproject,omitempty"`
		Status           string                  `json:"status"`
		LogPath          string                  `json:"log_path"`
		Error            string                  `json:"error,omitempty"`
		SkipReason       string                  `json:"skip_reason,omitempty"`
		StartedAt        time.Time               `json:"started_at"`
		FinishedAt       time.Time               `json:"finished_at"`
		Duration         *string                 `json:"duration"`
		Steps            []StepReport            `json:"steps"`
		Attestation      Attestation             `json:"attestation"`
		Conflict         *Conflict               `json:"conflict,omitempty"`
		Parent           *CompositionParent      `json:"parent,omitempty"`
		CompositionDepth int                     `json:"composition_depth,omitempty"`
		Fresh            bool                    `json:"fresh,omitempty"`
	}
	var duration *string
	if r.Status != "waiting" {
		value := r.Duration.String()
		duration = &value
	}
	return json.Marshal(wire{r.Admission, r.Observability, r.NativeStages, r.ID, r.Root, r.Profile, r.Subproject, r.Status, r.LogPath, r.Error, r.SkipReason, r.StartedAt, r.FinishedAt, duration, r.Steps, r.Attestation, r.Conflict, r.Parent, r.CompositionDepth, r.Fresh})
}

func (r *Report) UnmarshalJSON(data []byte) error {
	type wire struct {
		Admission        *AdmissionObservation   `json:"admission,omitempty"`
		Observability    *Observability          `json:"observability,omitempty"`
		NativeStages     *NativeStageObservation `json:"native_stages,omitempty"`
		ID               string                  `json:"id"`
		Root             string                  `json:"root"`
		Profile          string                  `json:"profile"`
		Subproject       string                  `json:"subproject,omitempty"`
		Status           string                  `json:"status"`
		LogPath          string                  `json:"log_path"`
		Error            string                  `json:"error,omitempty"`
		SkipReason       string                  `json:"skip_reason,omitempty"`
		StartedAt        time.Time               `json:"started_at"`
		FinishedAt       time.Time               `json:"finished_at"`
		Duration         *string                 `json:"duration"`
		Steps            []StepReport            `json:"steps"`
		Attestation      Attestation             `json:"attestation"`
		Conflict         *Conflict               `json:"conflict,omitempty"`
		Parent           *CompositionParent      `json:"parent,omitempty"`
		CompositionDepth int                     `json:"composition_depth,omitempty"`
		Fresh            bool                    `json:"fresh,omitempty"`
	}
	var in wire
	if err := json.Unmarshal(data, &in); err != nil {
		return err
	}
	var d time.Duration
	if in.Duration == nil {
		if in.Status != "waiting" {
			return fmt.Errorf("missing duration for %s receipt", in.Status)
		}
	} else {
		var err error
		d, err = time.ParseDuration(*in.Duration)
		if err != nil {
			return err
		}
	}
	*r = Report{Admission: in.Admission, Observability: in.Observability, NativeStages: in.NativeStages, ID: in.ID, Root: in.Root, Profile: in.Profile, Subproject: in.Subproject, Status: in.Status, LogPath: in.LogPath, Error: in.Error, SkipReason: in.SkipReason, StartedAt: in.StartedAt, FinishedAt: in.FinishedAt, Duration: d, Steps: in.Steps, Attestation: in.Attestation, Conflict: in.Conflict, Parent: in.Parent, CompositionDepth: in.CompositionDepth, Fresh: in.Fresh}
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
