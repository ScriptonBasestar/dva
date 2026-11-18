package cimetrics

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync/atomic"
	"time"
)

const recordSchema = "dva-project-ci-metrics/v2"
const legacyRecordSchema = "dva-project-ci-metrics/v1"

func supportedRecordSchema(schema string) bool {
	return schema == recordSchema || schema == legacyRecordSchema
}

const maxProgressBytes = 1024 * 1024
const progressFooterReserve = 4096
const maxSummaryBytes = 1024 * 1024
const maxFinishedBytes = 100 * 1024 * 1024
const maxFinishedCount = 1000
const maxFinishedAge = 30 * 24 * time.Hour

type Record struct {
	Schema            string             `json:"schema"`
	ID                string             `json:"measurement_id"`
	Check             string             `json:"check"`
	StartedAt         time.Time          `json:"started_at"`
	FinishedAt        time.Time          `json:"finished_at"`
	WallMS            *int64             `json:"wall_ms,omitempty"`
	Status            string             `json:"collection_status"`
	Reason            string             `json:"collection_reason,omitempty"`
	OutputDegraded    bool               `json:"output_degraded,omitempty"`
	Outcome           string             `json:"check_outcome"`
	ExitCode          *int               `json:"exit_code,omitempty"`
	Signal            string             `json:"signal,omitempty"`
	Metadata          Metadata           `json:"metadata"`
	Options           Options            `json:"options"`
	Events            *Snapshot          `json:"events,omitempty"`
	Hosted            *Hosted            `json:"hosted,omitempty"`
	ProgressRetention *ProgressRetention `json:"progress_retention,omitempty"`
}

// ProgressRetention describes intentional compaction separately from lost detail.
type ProgressRetention struct {
	Mode            string `json:"mode"`
	DetailStatus    string `json:"detail_status"`
	Omitted         uint64 `json:"omitted"`
	QueueDropped    uint64 `json:"queue_dropped"`
	ByteDropped     uint64 `json:"byte_dropped"`
	MarshalFailures uint64 `json:"marshal_failures"`
	WriteFailures   uint64 `json:"write_failures"`
}

type progressWriter struct {
	w     io.Writer
	size  int
	stats *ProgressRetention
}

func (w *progressWriter) write(v any, limit int) bool {
	b, err := json.Marshal(v)
	if err != nil {
		w.stats.MarshalFailures++
		return false
	}
	if len(b)+1 > limit-w.size {
		w.stats.ByteDropped++
		return false
	}
	b = append(b, '\n')
	n, err := w.w.Write(b)
	w.size += n
	if err != nil || n != len(b) {
		w.stats.WriteFailures++
		return false
	}
	return true
}

type Hosted struct {
	RunID   string `json:"run_id,omitempty"`
	Attempt int    `json:"attempt,omitempty"`
	Job     string `json:"job,omitempty"`
	Source  string `json:"source_sha,omitempty"`
}

type collector struct {
	id           string
	check        string
	start        time.Time
	events       chan Progress
	stop         chan struct{}
	final        chan Record
	done         chan struct{}
	dropped      atomic.Bool
	omitted      atomic.Uint64
	queueDropped atomic.Uint64
	failed       atomic.Bool
}

func newCollector(root, check, id string, start time.Time) *collector {
	c := &collector{id: id, check: check, start: start, events: make(chan Progress, 256), stop: make(chan struct{}), final: make(chan Record, 1), done: make(chan struct{})}
	go c.write(root)
	return c
}

// emit must never wait on a filesystem operation or a full queue.
// Test failures and package milestones remain visible; other test events are summarized.
func (c *collector) emit(p Progress) {
	if p.Test != "" && p.Action != "fail" {
		c.omitted.Add(1)
		return
	}
	select {
	case <-c.stop:
		return
	default:
	}
	select {
	case <-c.stop:
		return
	case c.events <- p:
	default:
		c.dropped.Store(true)
		c.queueDropped.Add(1)
	}
}

// finish bounds auxiliary work after the child exits. If storage cannot finish
// promptly, the active file remains partial and the caller still returns.
func (c *collector) finish(r Record) bool {
	incomplete := r.Status != "complete" || (r.Events != nil && r.Events.Status != "complete")
	c.final <- r
	close(c.stop)
	select {
	case <-c.done:
		return !c.failed.Load() && !c.dropped.Load() && !incomplete
	case <-time.After(100 * time.Millisecond):
		return false
	}
}

func (c *collector) write(root string) {
	defer close(c.done)
	key, ok := repositoryKey(root)
	if !ok {
		c.failed.Store(true)
		c.discard()
		return
	}
	dir, err := metricsDirectory(key)
	if err != nil {
		c.failed.Store(true)
		c.discard()
		return
	}
	// Refuse new records when the owned namespace cannot be inventoried within
	// a fixed bound. Do not delete unrecognized files to make room.
	if _, err := boundedEntries(dir); err != nil {
		c.failed.Store(true)
		c.discard()
		return
	}
	metadataReady := make(chan Metadata, 1)
	go func() { metadataReady <- collectMetadata(root, dir) }()
	activePath := filepath.Join(dir, c.id+".active.jsonl")
	f, err := openExclusive(activePath)
	if err != nil {
		c.failed.Store(true)
		c.discard()
		return
	}
	locked, err := tryLockFile(f)
	if err != nil || !locked {
		_ = f.Close()
		c.failed.Store(true)
		c.discard()
		return
	}
	defer func() {
		unlockFile(f)
		if err := f.Close(); err != nil {
			c.failed.Store(true)
		}
	}()
	stats := &ProgressRetention{Mode: "package-milestones-and-failures", DetailStatus: "compacted"}
	writer := &progressWriter{w: f, stats: stats}
	write := func(v any) bool { return writer.write(v, maxProgressBytes-progressFooterReserve) }
	if !write(struct {
		Schema    string    `json:"schema"`
		ID        string    `json:"measurement_id"`
		Check     string    `json:"check"`
		StartedAt time.Time `json:"started_at"`
	}{recordSchema, c.id, c.check, c.start}) {
		c.failed.Store(true)
	}
	consume := func(p Progress) {
		if !c.failed.Load() && !write(p) {
			c.dropped.Store(true)
			if stats.WriteFailures > 0 {
				c.failed.Store(true)
			}
		}
	}
	for {
		select {
		case p := <-c.events:
			consume(p)
		case <-c.stop:
			for {
				select {
				case p := <-c.events:
					consume(p)
				default:
					goto drained
				}
			}
		}
	}
drained:
	r := <-c.final
	stats.Omitted, stats.QueueDropped = c.omitted.Load(), c.queueDropped.Load()
	r.ProgressRetention = stats
	if c.failed.Load() || c.dropped.Load() {
		stats.DetailStatus = "partial"
	}
	select {
	case r.Metadata = <-metadataReady:
	default:
		r.Metadata = unknownMetadata()
	}
	if c.failed.Load() {
		r.Status, r.Reason = "partial", "progress storage failed"
	} else if c.dropped.Load() {
		r.Status, r.Reason = "partial", "progress detail lost"
	}
	if r.Events != nil && r.Events.Status != "complete" && r.Status == "complete" {
		r.Status, r.Reason = "partial", "incomplete event stream"
	}
	// The reserved footer gives bounded terminal context even after the detail limit.
	footer := struct {
		Type      string             `json:"type"`
		ID        string             `json:"measurement_id"`
		Outcome   string             `json:"check_outcome"`
		ExitCode  *int               `json:"exit_code,omitempty"`
		Status    string             `json:"collection_status"`
		Retention *ProgressRetention `json:"progress_retention"`
		Counts    map[string]int     `json:"counts,omitempty"`
	}{Type: "final", ID: r.ID, Outcome: r.Outcome, ExitCode: r.ExitCode, Status: r.Status, Retention: stats}
	if r.Events != nil {
		footer.Counts = r.Events.Counts
	}
	if !c.failed.Load() && !writer.write(footer, maxProgressBytes) {
		c.failed.Store(true)
		stats.DetailStatus = "partial"
		r.Status, r.Reason = "partial", "progress storage failed"
	}
	// Publication and retention share one nonblocking lock. If it is busy,
	// leave the active record for a later collector to recover as partial.
	lock, err := lockRetention(dir)
	if err != nil || lock == nil {
		c.failed.Store(true)
		return
	}
	defer unlockRetention(lock)
	// Publish progress before its summary. Retention only considers a valid
	// final summary, and never a still-active progress file.
	linked := false
	if !c.failed.Load() {
		eventsPath := filepath.Join(dir, c.id+".events.jsonl")
		if err := os.Link(activePath, eventsPath); err != nil {
			c.failed.Store(true)
		} else {
			linked = true
		}
	}
	if c.failed.Load() {
		r.Status, r.Reason = "partial", "progress storage failed"
	}
	if err := writeFinal(dir, r); err != nil {
		c.failed.Store(true)
		return
	}
	if linked {
		if err := os.Remove(activePath); err != nil {
			c.failed.Store(true)
		}
	}
	if err := pruneFinishedLocked(dir, time.Now()); err != nil {
		c.failed.Store(true)
	}
}

func (c *collector) discard() {
	for {
		select {
		case <-c.events:
		case <-c.stop:
			return
		}
	}
}

func unknownMetadata() Metadata {
	version := runtime.Version()
	if !safeGoVersion(version) {
		version = "unknown"
	}
	return Metadata{Revision: "unknown", Dirty: "unknown", SourceEvidence: "unknown", OS: runtime.GOOS, Arch: runtime.GOARCH, CPUs: runtime.NumCPU(), WrapperGo: version, GoBinaryBuild: "unknown", GOROOTVersion: "unknown", GoPairing: "unknown", Fresh: "unknown", Workspace: "unknown", ConfigStatus: "unknown"}
}

func metricsDirectory(key string) (string, error) {
	if len(key) != 64 || !isHex(key) {
		return "", errors.New("invalid repository identity")
	}
	base := os.Getenv("DVA_PROJECT_CI_METRICS_DIR")
	if base == "" {
		base = os.Getenv("XDG_STATE_HOME")
		if base == "" {
			home, e := os.UserHomeDir()
			if e != nil {
				return "", e
			}
			base = filepath.Join(home, ".local", "state")
		}
		base = filepath.Join(base, "dva-project-ci-metrics")
	}
	if !filepath.IsAbs(base) {
		return "", errors.New("metrics state directory must be absolute")
	}
	if err := os.MkdirAll(filepath.Dir(base), 0o700); err != nil {
		return "", err
	}
	if err := ensurePrivateDir(base); err != nil {
		return "", err
	}
	dir := filepath.Join(base, key)
	if err := ensurePrivateDir(dir); err != nil {
		return "", err
	}
	return dir, nil
}

func ensurePrivateDir(path string) error {
	if err := os.Mkdir(path, 0o700); err != nil && !errors.Is(err, os.ErrExist) {
		return err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0o077 != 0 || !ownedByUser(info) {
		return errors.New("metrics directory is not private and owned")
	}
	return nil
}

func writeFinal(dir string, r Record) error {
	b, err := json.Marshal(r)
	if err != nil {
		return err
	}
	if len(b) > maxSummaryBytes {
		r.Status, r.Reason = "partial", "summary byte limit exceeded"
		if r.Events != nil {
			r.Events.Completed = nil
		}
		b, err = json.Marshal(r)
		if err != nil || len(b) > maxSummaryBytes {
			return errors.New("summary byte limit exceeded")
		}
	}
	tmp, err := os.CreateTemp(dir, ".summary-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer func() { _ = os.Remove(name) }()
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(append(b, '\n')); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Link(name, filepath.Join(dir, r.ID+".json"))
}

type finishedRecord struct {
	id, summary string
	at          time.Time
	bytes       int64
	events      string
}

func pruneFinishedLocked(dir string, now time.Time) error {
	if err := reapAbandoned(dir, now); err != nil {
		return err
	}
	entries, err := boundedEntries(dir)
	if err != nil {
		return err
	}
	var records []finishedRecord
	var total int64
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		id := strings.TrimSuffix(name, ".json")
		if len(id) != 32 || !isHex(id) {
			continue
		}
		if _, err := os.Lstat(filepath.Join(dir, id+".active.jsonl")); err == nil || !errors.Is(err, os.ErrNotExist) {
			continue
		}
		p := filepath.Join(dir, name)
		r, size, err := readRecordHeader(p)
		if err != nil {
			continue
		}
		if !supportedRecordSchema(r.Schema) || r.ID != id || r.FinishedAt.IsZero() || r.FinishedAt.After(now) {
			continue
		}
		x := finishedRecord{id: id, summary: p, at: r.FinishedAt, bytes: size}
		eventPath := filepath.Join(dir, id+".events.jsonl")
		if size, err := progressHeaderSize(eventPath, id); err == nil {
			x.events = eventPath
			x.bytes += size
		}
		records = append(records, x)
		total += x.bytes
	}
	sort.Slice(records, func(i, j int) bool {
		if records[i].at.Equal(records[j].at) {
			return records[i].id < records[j].id
		}
		return records[i].at.Before(records[j].at)
	})
	remaining := len(records)
	for _, r := range records {
		if now.Sub(r.at) <= maxFinishedAge && remaining <= maxFinishedCount && total <= maxFinishedBytes {
			break
		}
		// Only fully valid own-schema summaries may be removed. Header-only
		// inventory avoids reading every test summary on every invocation.
		b, err := readOwnedRegular(r.summary, maxSummaryBytes)
		var verified Record
		if err != nil || json.Unmarshal(b, &verified) != nil || !supportedRecordSchema(verified.Schema) || verified.ID != r.id || verified.FinishedAt.IsZero() {
			continue
		}
		// Delete the event file only after verifying its own header and type.
		if r.events != "" {
			if err := os.Remove(r.events); err != nil && !errors.Is(err, os.ErrNotExist) {
				return err
			}
		}
		if err := os.Remove(r.summary); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		remaining--
		total -= r.bytes
	}
	if remaining > maxFinishedCount || total > maxFinishedBytes {
		return errors.New("metrics retention limit could not be enforced")
	}
	return nil
}

func readRecordHeader(path string) (Record, int64, error) {
	f, err := openNoFollow(path)
	if err != nil {
		return Record{}, 0, err
	}
	defer func() { _ = f.Close() }()
	info, err := f.Stat()
	if err != nil {
		return Record{}, 0, err
	}
	if !info.Mode().IsRegular() || !ownedByUser(info) || info.Mode().Perm()&0o077 != 0 || info.Size() > maxSummaryBytes {
		return Record{}, 0, errors.New("unrecognized metrics file")
	}
	d := json.NewDecoder(io.LimitReader(f, 8192))
	token, err := d.Token()
	if err != nil || token != json.Delim('{') {
		return Record{}, 0, errors.New("invalid metrics header")
	}
	var r Record
	for d.More() {
		token, err = d.Token()
		if err != nil {
			return Record{}, 0, err
		}
		key, ok := token.(string)
		if !ok {
			return Record{}, 0, errors.New("invalid metrics key")
		}
		switch key {
		case "schema":
			err = d.Decode(&r.Schema)
		case "measurement_id":
			err = d.Decode(&r.ID)
		case "finished_at":
			err = d.Decode(&r.FinishedAt)
		default:
			var discard json.RawMessage
			err = d.Decode(&discard)
		}
		if err != nil {
			return Record{}, 0, err
		}
		if r.Schema != "" && r.ID != "" && !r.FinishedAt.IsZero() {
			return r, info.Size(), nil
		}
	}
	return Record{}, 0, errors.New("incomplete metrics header")
}

func progressHeaderSize(path, id string) (int64, error) {
	f, err := openNoFollow(path)
	if err != nil {
		return 0, err
	}
	defer func() { _ = f.Close() }()
	info, err := f.Stat()
	if err != nil {
		return 0, err
	}
	if !info.Mode().IsRegular() || !ownedByUser(info) || info.Mode().Perm()&0o077 != 0 || info.Size() > maxProgressBytes {
		return 0, errors.New("unrecognized progress file")
	}
	b, err := io.ReadAll(io.LimitReader(f, 2048))
	if err != nil || !validProgressHeader(b, id) {
		return 0, errors.New("invalid progress header")
	}
	return info.Size(), nil
}

func validProgressHeader(b []byte, id string) bool {
	s := bufio.NewScanner(bytes.NewReader(b))
	if !s.Scan() {
		return false
	}
	var h struct {
		Schema string `json:"schema"`
		ID     string `json:"measurement_id"`
	}
	return json.Unmarshal(s.Bytes(), &h) == nil && supportedRecordSchema(h.Schema) && h.ID == id
}

// reapAbandoned acquires each active file's own non-blocking lock before
// classifying it. A live writer, an unrecognized header, and symlinks are not
// touched. Abandoned progress is published as partial with an unknown result.
func reapAbandoned(dir string, now time.Time) error {
	entries, err := boundedEntries(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".active.jsonl") {
			continue
		}
		id := strings.TrimSuffix(name, ".active.jsonl")
		if len(id) != 32 || !isHex(id) {
			continue
		}
		path := filepath.Join(dir, name)
		f, err := openNoFollowRW(path)
		if err != nil {
			continue
		}
		info, statErr := f.Stat()
		if statErr != nil || !info.Mode().IsRegular() || !ownedByUser(info) || info.Mode().Perm()&0o077 != 0 || info.Size() > maxProgressBytes || info.ModTime().After(now) {
			_ = f.Close()
			continue
		}
		locked, lockErr := tryLockFile(f)
		if lockErr != nil || !locked {
			_ = f.Close()
			continue
		}
		err = recoverOneActive(dir, path, id, f, info, now)
		unlockFile(f)
		closeErr := f.Close()
		if err != nil {
			return err
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return nil
}

func boundedEntries(dir string) ([]os.DirEntry, error) {
	f, err := openNoFollow(dir)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	info, err := f.Stat()
	if err != nil || !info.IsDir() || !ownedByUser(info) || info.Mode().Perm()&0o077 != 0 {
		return nil, errors.New("unrecognized metrics directory")
	}
	const limit = 10000
	entries := make([]os.DirEntry, 0, 512)
	for {
		batch, e := f.ReadDir(256)
		entries = append(entries, batch...)
		if len(entries) > limit {
			return nil, errors.New("metrics directory scan limit exceeded")
		}
		if errors.Is(e, io.EOF) {
			return entries, nil
		}
		if e != nil {
			return nil, e
		}
	}
}

func recoverOneActive(dir, path, id string, f *os.File, info os.FileInfo, now time.Time) error {
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return err
	}
	b, err := io.ReadAll(io.LimitReader(f, maxProgressBytes+1))
	if err != nil {
		return err
	}
	if len(b) > maxProgressBytes {
		return nil
	}
	s := bufio.NewScanner(bytes.NewReader(b))
	if !s.Scan() {
		return nil
	}
	var h struct {
		Schema    string    `json:"schema"`
		ID        string    `json:"measurement_id"`
		Check     string    `json:"check"`
		StartedAt time.Time `json:"started_at"`
	}
	if json.Unmarshal(s.Bytes(), &h) != nil || !supportedRecordSchema(h.Schema) || h.ID != id || !validCheck(h.Check) || h.StartedAt.IsZero() || h.StartedAt.After(now) {
		return nil
	}
	current, err := os.Lstat(path)
	if err != nil || !os.SameFile(info, current) {
		return nil
	}
	events := filepath.Join(dir, id+".events.jsonl")
	eventInfo, err := os.Lstat(events)
	if errors.Is(err, os.ErrNotExist) {
		if err := os.Link(path, events); err != nil {
			return err
		}
	} else if err != nil || !eventInfo.Mode().IsRegular() || !os.SameFile(info, eventInfo) {
		return nil
	}
	summaryPath := filepath.Join(dir, id+".json")
	if summaryBytes, err := readOwnedRegular(summaryPath, maxSummaryBytes); err == nil {
		var r Record
		if json.Unmarshal(summaryBytes, &r) != nil || !supportedRecordSchema(r.Schema) || r.ID != id || r.FinishedAt.IsZero() {
			return nil
		}
	} else if errors.Is(err, os.ErrNotExist) {
		finished := info.ModTime()
		if finished.Before(h.StartedAt) {
			finished = h.StartedAt
		}
		r := Record{Schema: h.Schema, ID: id, Check: h.Check, StartedAt: h.StartedAt, FinishedAt: finished, Status: "partial", Outcome: "unknown", Reason: "collector stopped before finalization", Metadata: unknownMetadata()}
		if err := writeFinal(dir, r); err != nil {
			return err
		}
	} else {
		return nil
	}
	current, err = os.Lstat(path)
	if err != nil || !os.SameFile(info, current) {
		return nil
	}
	return os.Remove(path)
}

func readOwnedRegular(path string, limit int64) ([]byte, error) {
	f, err := openNoFollow(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || !ownedByUser(info) || info.Mode().Perm()&0o077 != 0 || info.Size() > limit {
		return nil, fmt.Errorf("unrecognized metrics file")
	}
	b, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil || int64(len(b)) > limit {
		return nil, fmt.Errorf("metrics read failed or exceeded limit")
	}
	return b, nil
}
