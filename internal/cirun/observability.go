package cirun

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"

	"github.com/ScriptonBasestar/dva/internal/config"
)

// Observability contains diagnostic data only; it grants no authority.
type Observability struct {
	ExecutorVersion                  string `json:"executor_version"`
	ExecutorRevision                 string `json:"executor_revision"`
	ExecutorRevisionSource           string `json:"executor_revision_source"`
	RepositoryLocalID                string `json:"repository_local_id,omitempty"`
	Schema                           string `json:"schema"`
	Host                             string `json:"host,omitempty"`
	OS                               string `json:"os"`
	Arch                             string `json:"arch"`
	EventsDir                        string `json:"events_dir"`
	CacheCondition                   string `json:"cache_condition"`
	EventsStatus                     string `json:"events_status"`
	EventsError                      string `json:"events_error,omitempty"`
	ConfigurationShapeFingerprint    string `json:"configuration_shape_fingerprint"`
	ConfigurationFingerprintComplete bool   `json:"configuration_fingerprint_complete"`
}

func newObservability(events string) *Observability {
	host, _ := os.Hostname()
	commit := config.ResolvedCommit()
	return &Observability{ExecutorVersion: config.Version, ExecutorRevision: commit.Value, ExecutorRevisionSource: commit.Source, Schema: "dva-ci-observability/1", Host: host, OS: runtime.GOOS, Arch: runtime.GOARCH, EventsDir: events, CacheCondition: "unknown"}
}

func prepareObservability(state, id, profileName string, p config.CIProfile) *Observability {
	o := newObservability("")
	type stepShape struct {
		Name             string
		DependsOn        []string
		Timeout, Workdir string
	}
	shape := struct {
		Profile     string
		Timeout     string
		MaxParallel int
		ToolJobs    *int
		Steps       []stepShape
	}{Profile: profileName, Timeout: p.Timeout, MaxParallel: p.MaxParallel, ToolJobs: p.ToolJobs}
	for _, s := range p.Steps {
		shape.Steps = append(shape.Steps, stepShape{s.Name, s.DependsOn, s.Timeout, s.Workdir})
	}
	b, _ := json.Marshal(shape)
	sum := sha256.Sum256(b)
	o.ConfigurationShapeFingerprint = hex.EncodeToString(sum[:])
	parent := filepath.Join(state, "events")
	if err := os.MkdirAll(parent, 0700); err != nil {
		o.EventsStatus = "degraded"
		o.EventsError = err.Error()
		return o
	}
	info, err := os.Lstat(parent)
	if err != nil || !info.IsDir() || info.Mode().Perm() != 0700 || !ownedByCurrentUser(info) {
		o.EventsStatus = "degraded"
		o.EventsError = "events parent is not a private directory"
		return o
	}
	path := filepath.Join(parent, id)
	if err = os.Mkdir(path, 0700); err != nil {
		o.EventsStatus = "degraded"
		o.EventsError = err.Error()
		return o
	}
	info, err = os.Lstat(path)
	if err != nil || !info.IsDir() || info.Mode().Perm() != 0700 || !ownedByCurrentUser(info) {
		o.EventsStatus = "degraded"
		o.EventsError = "events directory is not private and current-user owned"
		return o
	}
	o.EventsDir = path
	o.EventsStatus = "available"
	return o
}
