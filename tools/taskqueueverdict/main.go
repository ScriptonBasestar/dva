// taskqueueverdict classifies the read-only TaskChain queue without selecting work.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode"
)

const (
	verdictVersion = 1
	queueTimeout   = 10 * time.Second
	maxQueueOutput = 1 << 20
)

var errOutputLimit = errors.New("task queue output exceeds limit")

type queue struct {
	OutputVersion      int               `json:"outputVersion"`
	RunnableCount      int               `json:"runnableCount"`
	AgentRunnableCount int               `json:"agentRunnableCount"`
	Runnable           []json.RawMessage `json:"runnable"`
	AgentRunnable      []json.RawMessage `json:"agentRunnable"`
}

type queueItem struct {
	Card struct {
		ID string `json:"id"`
	} `json:"card"`
	Path          string   `json:"path"`
	ExecutionMode string   `json:"executionMode"`
	NeedsHuman    bool     `json:"needsHuman"`
	AllowedPaths  []string `json:"allowedPaths"`
}

type verdict struct {
	VerdictVersion     int               `json:"verdictVersion"`
	State              string            `json:"state"`
	RunnableCount      int               `json:"runnableCount"`
	AgentRunnableCount int               `json:"agentRunnableCount"`
	Runnable           []json.RawMessage `json:"runnable"`
	AgentRunnable      []json.RawMessage `json:"agentRunnable"`
	Candidate          json.RawMessage   `json:"candidate"`
}

func main() {
	dir := flag.String("dir", "tasks", "TaskChain board directory")
	startType := flag.String("start-type", "", "CE task branch type (feat|fix|refactor|docs|test|chore|perf)")
	flag.Parse()
	if flag.NArg() != 0 {
		fail(fmt.Errorf("unexpected arguments: %s", strings.Join(flag.Args(), " ")))
	}

	if err := execute(context.Background(), *dir, *startType, os.Stdout); err != nil {
		if startErr, ok := errors.AsType[*runStartError](err); ok {
			failCode(err, startErr.exitCode)
		}
		fail(err)
	}
}

func fail(err error) {
	failCode(err, 1)
}

func failCode(err error, code int) {
	fmt.Fprintln(os.Stderr, "task-queue-verdict:", err)
	os.Exit(code)
}

// execute preserves the original verdict output unless an operator explicitly
// selects a CE branch type. A start request is deliberately a second phase:
// queue validation completes before CE, and every non-candidate verdict fails
// without calling CE's lifecycle writer.
func execute(ctx context.Context, dir, startType string, stdout io.Writer) error {
	if startType != "" && !validStartType(startType) {
		return fmt.Errorf("invalid --start-type %q (want feat, fix, refactor, docs, test, chore, or perf)", startType)
	}
	if startType != "" && dir != "tasks" {
		return fmt.Errorf("CE start is bound to the repository tasks board, got --dir %q", dir)
	}

	queueCtx, cancelQueue := context.WithTimeout(ctx, queueTimeout)
	q, err := loadQueue(queueCtx, dir)
	cancelQueue()
	if err != nil {
		return err
	}
	v := classify(q)
	if startType == "" {
		encoded, err := json.Marshal(v)
		if err != nil {
			return fmt.Errorf("encode verdict: %w", err)
		}
		_, err = fmt.Fprintln(stdout, string(encoded))
		return err
	}
	if v.State != "candidate" {
		return fmt.Errorf("CE run-start requires one candidate; queue verdict is %q", v.State)
	}
	key, err := runtimeKey(v.Candidate)
	if err != nil {
		return err
	}
	output, err := runStart(ctx, key, startType)
	if len(output) > 0 {
		if _, writeErr := stdout.Write(output); writeErr != nil {
			return writeErr
		}
	}
	if err != nil {
		return err
	}
	return nil
}

func validStartType(value string) bool {
	switch value {
	case "feat", "fix", "refactor", "docs", "test", "chore", "perf":
		return true
	default:
		return false
	}
}

func runtimeKey(raw json.RawMessage) (string, error) {
	var item queueItem
	if err := json.Unmarshal(raw, &item); err != nil {
		return "", fmt.Errorf("decode candidate: %w", err)
	}
	normalized, ok := normalizedQueueID(item.Card.ID)
	if !ok {
		return "", errors.New("candidate has invalid card id")
	}
	return strings.ToLower(normalized), nil
}

// runStart forwards successful CE JSON without interpreting its versioned
// response envelope. CE owns that contract and any lifecycle state it writes.
func runStart(ctx context.Context, key, startType string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "ce", "task", "run-start", key, "--type", startType, "--json")
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	if err != nil {
		code := 1
		if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
			code = exitErr.ExitCode()
		}
		if stderr.Len() > 0 {
			return stdout.Bytes(), &runStartError{exitCode: code, err: fmt.Errorf("CE run-start failed: %w: %s", err, strings.TrimSpace(stderr.String()))}
		}
		return stdout.Bytes(), &runStartError{exitCode: code, err: fmt.Errorf("CE run-start failed: %w", err)}
	}
	if stderr.Len() > 0 {
		_, _ = os.Stderr.Write(stderr.Bytes())
	}
	return stdout.Bytes(), nil
}

type runStartError struct {
	exitCode int
	err      error
}

func (e *runStartError) Error() string { return e.err.Error() }
func (e *runStartError) Unwrap() error { return e.err }

func loadQueue(ctx context.Context, dir string) (queue, error) {
	cmd := exec.CommandContext(ctx, "taskchain-task-manager", "queue", "--dir", dir, "--json")
	var stdout, stderr limitedBuffer
	stdout.limit, stderr.limit = maxQueueOutput, maxQueueOutput
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return queue{}, fmt.Errorf("task queue timed out after %s", queueTimeout)
	}
	if errors.Is(err, errOutputLimit) || stdout.exceeded || stderr.exceeded {
		return queue{}, fmt.Errorf("%w (%d bytes per stream)", errOutputLimit, maxQueueOutput)
	}
	if err != nil {
		if stderr.Len() > 0 {
			return queue{}, fmt.Errorf("task queue failed: %w: %s", err, strings.TrimSpace(stderr.String()))
		}
		return queue{}, fmt.Errorf("task queue failed: %w", err)
	}
	if stderr.Len() > 0 {
		return queue{}, fmt.Errorf("task queue wrote to stderr: %s", strings.TrimSpace(stderr.String()))
	}
	q, err := parseQueue(stdout.Bytes())
	if err != nil {
		return queue{}, err
	}
	return q, nil
}

type limitedBuffer struct {
	bytes.Buffer
	limit    int
	exceeded bool
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	if b.Len()+len(p) > b.limit {
		remaining := b.limit - b.Len()
		if remaining > 0 {
			_, _ = b.Buffer.Write(p[:remaining])
		}
		b.exceeded = true
		return 0, errOutputLimit
	}
	return b.Buffer.Write(p)
}

func parseQueue(data []byte) (queue, error) {
	if err := rejectDuplicateJSONKeys(data); err != nil {
		return queue{}, fmt.Errorf("invalid task queue JSON: %w", err)
	}
	var raw map[string]json.RawMessage
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&raw); err != nil {
		return queue{}, fmt.Errorf("invalid task queue JSON: %w", err)
	}
	if err := ensureEOF(decoder); err != nil {
		return queue{}, fmt.Errorf("invalid task queue JSON: %w", err)
	}
	for _, name := range []string{"outputVersion", "runnableCount", "agentRunnableCount", "runnable", "agentRunnable"} {
		value, ok := raw[name]
		if !ok {
			return queue{}, fmt.Errorf("task queue JSON missing %q", name)
		}
		if bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return queue{}, fmt.Errorf("task queue JSON %q must not be null", name)
		}
	}
	var q queue
	if err := json.Unmarshal(data, &q); err != nil {
		return queue{}, fmt.Errorf("invalid task queue JSON: %w", err)
	}
	if q.OutputVersion != 1 {
		return queue{}, fmt.Errorf("unsupported task queue outputVersion %d", q.OutputVersion)
	}
	if q.Runnable == nil || q.AgentRunnable == nil {
		return queue{}, errors.New("task queue runnable arrays must be arrays")
	}
	if q.RunnableCount < 0 || q.AgentRunnableCount < 0 || q.RunnableCount != len(q.Runnable) || q.AgentRunnableCount != len(q.AgentRunnable) {
		return queue{}, errors.New("task queue counts do not match runnable arrays")
	}
	if err := validateItems("runnable", q.Runnable, false); err != nil {
		return queue{}, err
	}
	if err := validateItems("agentRunnable", q.AgentRunnable, true); err != nil {
		return queue{}, err
	}
	if err := validateAgentSubset(q); err != nil {
		return queue{}, err
	}
	return q, nil
}

// rejectDuplicateJSONKeys rejects the ambiguity encoding/json otherwise resolves
// by silently keeping the final member. It walks nested card objects too.
func rejectDuplicateJSONKeys(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := scanJSONValue(decoder); err != nil {
		return err
	}
	return ensureEOF(decoder)
}

func scanJSONValue(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delim, isDelim := token.(json.Delim)
	if !isDelim {
		return nil
	}
	switch delim {
	case '{':
		keys := make(map[string]struct{})
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return errors.New("object key is not a string")
			}
			if _, exists := keys[key]; exists {
				return fmt.Errorf("duplicate JSON object key %q", key)
			}
			keys[key] = struct{}{}
			if err := scanJSONValue(decoder); err != nil {
				return err
			}
		}
		_, err = decoder.Token()
		return err
	case '[':
		for decoder.More() {
			if err := scanJSONValue(decoder); err != nil {
				return err
			}
		}
		_, err = decoder.Token()
		return err
	default:
		return fmt.Errorf("unexpected JSON delimiter %q", delim)
	}
}

func ensureEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err == io.EOF {
		return nil
	} else if err != nil {
		return err
	}
	return errors.New("multiple JSON values")
}

func validateItems(name string, rawItems []json.RawMessage, agent bool) error {
	ids, paths := make(map[string]struct{}), make(map[string]struct{})
	for i, raw := range rawItems {
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(raw, &fields); err != nil || fields == nil {
			return fmt.Errorf("%s[%d] must be an object", name, i)
		}
		for _, key := range []string{"card", "path", "executionMode", "needsHuman", "allowedPaths"} {
			value, ok := fields[key]
			if !ok || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
				return fmt.Errorf("%s[%d] needs non-null %q", name, i, key)
			}
		}
		var item queueItem
		if err := json.Unmarshal(raw, &item); err != nil {
			return fmt.Errorf("%s[%d] is invalid: %w", name, i, err)
		}
		identity, valid := normalizedQueueID(item.Card.ID)
		if !valid || !safeQueuePath(item.Path) {
			return fmt.Errorf("%s[%d] has invalid card id or board path", name, i)
		}
		if _, exists := ids[identity]; exists {
			return fmt.Errorf("%s has duplicate id %q", name, item.Card.ID)
		}
		if _, exists := paths[item.Path]; exists {
			return fmt.Errorf("%s has duplicate path %q", name, item.Path)
		}
		ids[identity], paths[item.Path] = struct{}{}, struct{}{}
		switch item.ExecutionMode {
		case "implementation":
			if len(item.AllowedPaths) == 0 {
				return fmt.Errorf("%s[%d] implementation has no allowedPaths", name, i)
			}
			for _, path := range item.AllowedPaths {
				if !safeAllowedPath(path) {
					return fmt.Errorf("%s[%d] has unsafe allowedPath %q", name, i, path)
				}
			}
		case "external", "decision":
			if !item.NeedsHuman || len(item.AllowedPaths) != 0 {
				return fmt.Errorf("%s[%d] has inconsistent human route", name, i)
			}
		default:
			return fmt.Errorf("%s[%d] has unknown executionMode %q", name, i, item.ExecutionMode)
		}
		if agent && (item.ExecutionMode != "implementation" || item.NeedsHuman) {
			return fmt.Errorf("agentRunnable[%d] is not an executable implementation candidate", i)
		}
	}
	return nil
}

func validateAgentSubset(q queue) error {
	expected := make([]any, 0, len(q.Runnable))
	for _, raw := range q.Runnable {
		var item queueItem
		if err := json.Unmarshal(raw, &item); err != nil {
			return err
		}
		if item.ExecutionMode == "implementation" && !item.NeedsHuman {
			value, err := semanticJSON(raw)
			if err != nil {
				return fmt.Errorf("invalid runnable JSON: %w", err)
			}
			expected = append(expected, value)
		}
	}
	if len(expected) != len(q.AgentRunnable) {
		return errors.New("agentRunnable does not match executable runnable projection")
	}
	for i, raw := range q.AgentRunnable {
		candidate, err := semanticJSON(raw)
		if err != nil {
			return fmt.Errorf("invalid agentRunnable JSON: %w", err)
		}
		if !semanticEqual(expected[i], candidate) {
			return fmt.Errorf("agentRunnable[%d] does not match executable runnable projection", i)
		}
	}
	return nil
}

func semanticJSON(raw json.RawMessage) (any, error) {
	var value any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	return value, ensureEOF(decoder)
}

func semanticEqual(left, right any) bool {
	return reflect.DeepEqual(left, right)
}

func normalizedQueueID(id string) (string, bool) {
	prefix, digits, ok := strings.Cut(id, "-")
	if !ok || digits == "" {
		return "", false
	}
	switch prefix {
	case "TASK", "ISSUE":
	default:
		return "", false
	}
	for _, digit := range digits {
		if digit < '0' || digit > '9' {
			return "", false
		}
	}
	number, err := strconv.ParseUint(digits, 10, 64)
	if err != nil {
		return "", false
	}
	return prefix + "-" + strconv.FormatUint(number, 10), true
}

func safeQueuePath(name string) bool {
	if name == "" || path.IsAbs(name) || path.Clean(name) != name || strings.Contains(name, "\\") {
		return false
	}
	if len(name) >= 2 && name[1] == ':' && ((name[0] >= 'a' && name[0] <= 'z') || (name[0] >= 'A' && name[0] <= 'Z')) {
		return false
	}
	for _, r := range name {
		if unicode.IsControl(r) {
			return false
		}
	}
	for segment := range strings.SplitSeq(name, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return false
		}
	}
	return true
}

func safeAllowedPath(path string) bool {
	if path == "" || path == "tasks" || strings.HasPrefix(path, "tasks/") || strings.HasPrefix(path, "/") || strings.Contains(path, "\\") {
		return false
	}
	for _, r := range path {
		if unicode.IsSpace(r) || unicode.IsControl(r) || strings.ContainsRune("*?[]", r) {
			return false
		}
	}
	if len(path) >= 2 && path[1] == ':' && ((path[0] >= 'a' && path[0] <= 'z') || (path[0] >= 'A' && path[0] <= 'Z')) {
		return false
	}
	for segment := range strings.SplitSeq(path, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return false
		}
	}
	return true
}

func classify(q queue) verdict {
	v := verdict{
		VerdictVersion:     verdictVersion,
		RunnableCount:      q.RunnableCount,
		AgentRunnableCount: q.AgentRunnableCount,
		Runnable:           q.Runnable,
		AgentRunnable:      q.AgentRunnable,
		Candidate:          json.RawMessage("null"),
	}
	switch len(q.AgentRunnable) {
	case 0:
		if len(q.Runnable) == 0 {
			v.State = "empty"
		} else {
			v.State = "human_required"
		}
	case 1:
		v.State, v.Candidate = "candidate", q.AgentRunnable[0]
	default:
		v.State = "selection_required"
	}
	return v
}
