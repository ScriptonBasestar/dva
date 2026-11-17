package cirun

import (
	"fmt"
	"io"
	"sync"
	"time"
)

// OutputObservation is diagnostic only. A nil observation means output was not measured.
type OutputObservation struct {
	Scope          string     `json:"scope"`
	StartedAt      time.Time  `json:"started_at"`
	LastOutputAt   *time.Time `json:"last_output_at,omitempty"`
	WarningEmitted bool       `json:"warning_emitted"`
}

type outputWatch struct {
	mu                  sync.Mutex
	out                 io.Writer
	name                string
	threshold, interval time.Duration
	started             time.Time
	last                *time.Time
	warned, everWarned  bool
	done                chan struct{}
	finished            chan struct{}
}

func newOutputWatch(out io.Writer, name string, threshold, interval time.Duration) *outputWatch {
	return &outputWatch{out: out, name: name, threshold: threshold, interval: interval, done: make(chan struct{}), finished: make(chan struct{})}
}

func (w *outputWatch) Write(p []byte) (int, error) {
	if len(p) > 0 {
		now := time.Now()
		w.mu.Lock()
		w.last = &now
		w.warned = false
		w.mu.Unlock()
	}
	return w.out.Write(p)
}

func (w *outputWatch) start() {
	w.mu.Lock()
	w.started = time.Now()
	w.mu.Unlock()
	go func() {
		defer close(w.finished)
		ticker := time.NewTicker(w.interval)
		defer ticker.Stop()
		for {
			select {
			case now := <-ticker.C:
				w.check(now)
			case <-w.done:
				return
			}
		}
	}()
}

func (w *outputWatch) check(now time.Time) {
	w.mu.Lock()
	defer w.mu.Unlock()
	last := w.started
	if w.last != nil && w.last.After(last) {
		last = *w.last
	}
	if !w.warned && now.Sub(last) >= w.threshold {
		w.warned = true
		w.everWarned = true
		_, _ = fmt.Fprintf(w.out, "ci output idle step=%s elapsed=%s idle=%s (warning only)\n", w.name, now.Sub(w.started).Round(time.Second), now.Sub(last).Round(time.Second))
	}
}

func (w *outputWatch) stop() *OutputObservation {
	close(w.done)
	<-w.finished
	w.mu.Lock()
	defer w.mu.Unlock()
	return &OutputObservation{Scope: "native_leaf_child_output", StartedAt: w.started, LastOutputAt: w.last, WarningEmitted: w.everWarned}
}
