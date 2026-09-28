package taskqueue

import (
	"bytes"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

//go:embed taskchain-pins.json
var pinFiles embed.FS

var embeddedPins = readEmbeddedPins()

type artifactPins struct {
	SchemaVersion int           `json:"schemaVersion"`
	Artifacts     []artifactPin `json:"artifacts"`
	loadErr       error
}
type artifactPin struct {
	GOOS               string `json:"goos"`
	GOARCH             string `json:"goarch"`
	SourceCommit       string `json:"sourceCommit"`
	SourceTree         string `json:"sourceTree"`
	GoVersion          string `json:"goVersion"`
	CGOEnabled         string `json:"cgoEnabled"`
	GOARM64            string `json:"goarm64"`
	BuildCommand       string `json:"buildCommand"`
	SHA256             string `json:"sha256"`
	Status             string `json:"status"`
	Distribution       string `json:"distribution"`
	MutationAuthorized bool   `json:"mutationAuthorized"`
}

func readEmbeddedPins() artifactPins {
	data, err := pinFiles.ReadFile("taskchain-pins.json")
	if err != nil {
		return artifactPins{loadErr: err}
	}
	var pins artifactPins
	if err := rejectDuplicateJSONKeys(data); err != nil {
		pins.loadErr = err
		return pins
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&pins); err != nil {
		pins.loadErr = err
	} else if err := ensureEOF(decoder); err != nil {
		pins.loadErr = err
	}
	return pins
}

func (pins artifactPins) activeForPlatform() (artifactPin, error) {
	platform := runtime.GOOS + "/" + runtime.GOARCH
	if pins.loadErr != nil || pins.SchemaVersion != 1 {
		return artifactPin{}, fmt.Errorf("TaskChain pin manifest invalid for %s: load=%v version=%d", platform, pins.loadErr, pins.SchemaVersion)
	}
	var active *artifactPin
	for i := range pins.Artifacts {
		pin := &pins.Artifacts[i]
		if !pin.MutationAuthorized || pin.GOOS != runtime.GOOS || pin.GOARCH != runtime.GOARCH {
			continue
		}
		if active != nil {
			return artifactPin{}, fmt.Errorf("multiple authorized TaskChain pins for %s", platform)
		}
		active = pin
	}
	if active == nil {
		return artifactPin{}, fmt.Errorf("no authorized TaskChain binary pin for %s; CE start is disabled", platform)
	}
	if active.Status != "published-approved" || active.Distribution != "published" || active.GoVersion == "" || active.BuildCommand == "" || (active.CGOEnabled != "0" && active.CGOEnabled != "1") {
		return artifactPin{}, fmt.Errorf("authorized TaskChain pin for %s lacks approved release/build provenance", platform)
	}
	for _, revision := range []string{active.SourceCommit, active.SourceTree} {
		if len(revision) != 40 {
			return artifactPin{}, fmt.Errorf("authorized TaskChain pin for %s has invalid source revision", platform)
		}
		if _, err := hex.DecodeString(revision); err != nil {
			return artifactPin{}, fmt.Errorf("authorized TaskChain pin for %s has invalid source revision: %w", platform, err)
		}
	}
	if len(active.SHA256) != 64 || strings.ToLower(active.SHA256) != active.SHA256 {
		return artifactPin{}, fmt.Errorf("invalid authorized TaskChain SHA-256 for %s", platform)
	}
	if _, err := hex.DecodeString(active.SHA256); err != nil {
		return artifactPin{}, fmt.Errorf("invalid authorized TaskChain SHA-256 for %s: %w", platform, err)
	}
	return *active, nil
}

// pinnedQueueBinary resolves one PATH entry and runs only a hashed snapshot of it.
func pinnedQueueBinary(pin artifactPin) (string, func() error, error) {
	selected, err := exec.LookPath("taskchain-task-manager")
	if err != nil {
		return "", nil, fmt.Errorf("find TaskChain binary for %s/%s: %w", pin.GOOS, pin.GOARCH, err)
	}
	selected, err = filepath.Abs(selected)
	if err != nil {
		return "", nil, fmt.Errorf("absolute TaskChain binary path: %w", err)
	}
	source, err := os.Open(selected)
	if err != nil {
		return "", nil, fmt.Errorf("open selected TaskChain binary %s: %w", selected, err)
	}
	defer func() { _ = source.Close() }()
	info, err := source.Stat()
	if err != nil {
		return "", nil, fmt.Errorf("stat selected TaskChain binary %s: %w", selected, err)
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0o111 == 0 {
		return "", nil, fmt.Errorf("selected TaskChain binary %s is not a regular executable: mode=%v", selected, info.Mode())
	}
	dir, err := os.MkdirTemp("", "dva-taskchain-pin-*")
	if err != nil {
		return "", nil, fmt.Errorf("create private TaskChain snapshot directory: %w", err)
	}
	snapshot := filepath.Join(dir, "taskchain-task-manager")
	cleanup := func() error {
		if err := os.Remove(snapshot); err != nil && !os.IsNotExist(err) {
			return err
		}
		return os.Remove(dir)
	}
	destination, err := os.OpenFile(snapshot, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o700)
	if err != nil {
		_ = os.Remove(dir)
		return "", nil, fmt.Errorf("create TaskChain snapshot: %w", err)
	}
	hash := sha256.New()
	_, copyErr := io.Copy(io.MultiWriter(destination, hash), source)
	closeErr := destination.Close()
	if copyErr != nil || closeErr != nil {
		_ = cleanup()
		return "", nil, fmt.Errorf("copy TaskChain snapshot: copy=%v close=%v", copyErr, closeErr)
	}
	actual := hex.EncodeToString(hash.Sum(nil))
	if actual != pin.SHA256 {
		_ = cleanup()
		return "", nil, fmt.Errorf("selected TaskChain binary %s SHA-256 mismatch for %s/%s: expected %s, observed %s", selected, pin.GOOS, pin.GOARCH, pin.SHA256, actual)
	}
	return snapshot, cleanup, nil
}
