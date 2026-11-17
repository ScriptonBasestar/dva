// Package resourceadmission defines the host's cooperative resource budget.
package resourceadmission

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"time"
)

// Policy is an explicit host configuration; it is not inferred from CPU count.
type Policy struct {
	Capacity      int            `json:"capacity"`
	MaxRunning    int            `json:"max_running"`
	MaxHeavy      int            `json:"max_heavy"`
	Weights       map[string]int `json:"weights"`
	WorkerLimits  map[string]int `json:"worker_limits,omitempty"`
	AgingInterval string         `json:"aging_interval"`
}

// ReadPolicy rejects relative paths, symlinks, non-regular files and malformed
// input. The returned digest is a hash of normalized validated policy values.
func ReadPolicy(path string) (Policy, string, error) {
	var p Policy
	if !filepath.IsAbs(path) {
		return p, "", errors.New("resource policy path must be absolute")
	}
	f, err := openPolicy(path)
	if err != nil {
		return p, "", err
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil {
		return p, "", err
	}
	if !opened.Mode().IsRegular() || !owned(opened) || opened.Size() > 65536 {
		return p, "", errors.New("resource policy must be a current-user-owned regular file of at most 64KiB")
	}
	data, err := io.ReadAll(io.LimitReader(f, 65537))
	if err != nil {
		return p, "", err
	}
	if len(data) > 65536 {
		return p, "", errors.New("resource policy exceeds size limit")
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err = d.Decode(&p); err != nil {
		return p, "", fmt.Errorf("invalid resource policy: %w", err)
	}
	var extra any
	if err = d.Decode(&extra); err != io.EOF {
		return p, "", errors.New("resource policy must contain one JSON value")
	}
	if p.Capacity < 1 || p.MaxRunning < 1 || p.MaxHeavy < 0 || p.MaxHeavy > p.MaxRunning || len(p.Weights) != 2 || p.Weights["light"] < 1 || p.Weights["heavy"] < p.Weights["light"] || p.Weights["light"] > p.Capacity || p.Weights["heavy"] > p.Capacity {
		return p, "", errors.New("invalid resource capacity, limits or weights")
	}
	for class, limit := range p.WorkerLimits {
		if (class != "light" && class != "heavy") || limit < 1 {
			return p, "", errors.New("invalid worker_limits")
		}
	}
	if duration, err := time.ParseDuration(p.AgingInterval); err != nil || duration <= 0 {
		return p, "", errors.New("aging_interval must be a positive duration")
	}
	canonical, err := json.Marshal(p)
	if err != nil {
		return p, "", err
	}
	sum := sha256.Sum256(canonical)
	return p, hex.EncodeToString(sum[:]), nil
}
