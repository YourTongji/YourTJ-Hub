// Package closer stores process-wide shutdown callbacks.
package closer

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime"
	"slices"
	"sync"
	"time"
)

// CloseFunc is a cooperative shutdown callback. Implementations must stop
// work and return when ctx is canceled; Go cannot interrupt a callback that
// ignores its context.
type CloseFunc func(context.Context) error

type Priority int

const (
	PriorityProducer Priority = 100
	PriorityFlush    Priority = 200
	PriorityCache    Priority = 300
	PriorityDefault  Priority = 500
	PriorityDatabase Priority = 900
	PriorityLogger   Priority = 1000
)

var (
	mu           sync.Mutex
	entries      []closerEntry
	closeTimeout = 10 * time.Second
	nextSeq      uint64
)

type closerEntry struct {
	f        CloseFunc
	caller   string
	priority Priority
	seq      uint64
}

// Register adds a legacy non-context callback to the process shutdown list.
// New code should use RegisterContext so shutdown can cancel owned work.
func Register(f func() error) {
	RegisterContext(func(context.Context) error { return f() })
}

// RegisterContext adds a cooperative shutdown callback to the process list.
func RegisterContext(f CloseFunc) {
	register(1, PriorityDefault, f)
}

// RegisterPriority adds a legacy non-context callback with an explicit close phase.
func RegisterPriority(priority Priority, f func() error) {
	RegisterPriorityContext(priority, func(context.Context) error { return f() })
}

// RegisterPriorityContext adds a cooperative callback with an explicit close phase.
func RegisterPriorityContext(priority Priority, f CloseFunc) {
	register(1, priority, f)
}

// Bind registers the Close method of c as a shutdown callback.
func Bind(c interface{ Close() error }) {
	Register(c.Close)
}

// BindContext registers a context-aware Close method as a shutdown callback.
func BindContext(c interface{ Close(context.Context) error }) {
	RegisterContext(c.Close)
}

// BindPriority registers the Close method of c with an explicit close phase.
func BindPriority(priority Priority, c interface{ Close() error }) {
	RegisterPriority(priority, c.Close)
}

// BindPriorityContext registers a context-aware Close method with a phase.
func BindPriorityContext(priority Priority, c interface{ Close(context.Context) error }) {
	RegisterPriorityContext(priority, c.Close)
}

func register(skip int, priority Priority, f CloseFunc) {
	mu.Lock()
	defer mu.Unlock()

	_, file, line, ok := runtime.Caller(skip + 1)
	caller := "unknown"
	if ok {
		caller = fmt.Sprintf("%s:%d", file, line)
	}

	entries = append(entries, closerEntry{
		f:        f,
		caller:   caller,
		priority: priority,
		seq:      nextSeq,
	})
	nextSeq++
	slog.Info("closer: registered resource",
		"caller", caller,
		"priority", priority,
		"total", len(entries),
	)
}

// CloseAll runs all registered shutdown callbacks in deterministic priority
// order and waits synchronously for each callback up to the hard timeout. The
// variadic form preserves callers that do not yet have a root context while
// allowing the server to provide one.
func CloseAll(parentContexts ...context.Context) error {
	parent := context.Background()
	if len(parentContexts) > 0 && parentContexts[0] != nil {
		parent = parentContexts[0]
	}

	mu.Lock()
	items := append([]closerEntry(nil), entries...)
	entries = nil
	mu.Unlock()

	slices.SortStableFunc(items, func(a, b closerEntry) int {
		return cmp.Or(cmp.Compare(a.priority, b.priority), cmp.Compare(b.seq, a.seq))
	})

	slog.Info("closer: starting to close all registered resources", "count", len(items))
	var closeErrors []error

	for i := range items {
		entry := items[i]
		slog.Info("closer: closing resource",
			"index", i,
			"priority", entry.priority,
			"registered_at", entry.caller,
		)
		if err := closeWithTimeout(parent, entry); err != nil {
			closeErrors = append(closeErrors, err)
			slog.Error("closer: failed to close resource",
				"index", i,
				"priority", entry.priority,
				"error", err,
				"registered_at", entry.caller,
			)
		}
	}

	slog.Info("closer: all resources closed")
	return errors.Join(closeErrors...)
}

func closeWithTimeout(parent context.Context, entry closerEntry) error {
	ctx, cancel := context.WithTimeoutCause(parent, closeTimeout, errCloseTimeout)
	defer cancel()

	result := make(chan error, 1)
	go func() { result <- entry.f(ctx) }()
	select {
	case err := <-result:
		if errors.Is(context.Cause(ctx), errCloseTimeout) {
			return timeoutError(entry)
		}
		if err != nil {
			return fmt.Errorf("close failed: priority=%d registered_at=%s: %w", entry.priority, entry.caller, err)
		}
		return nil
	case <-ctx.Done():
		return timeoutError(entry)
	}
}

var errCloseTimeout = errors.New("closer: callback deadline exceeded")

func timeoutError(entry closerEntry) error {
	return fmt.Errorf("close timed out after %s: priority=%d registered_at=%s: %w", closeTimeout, entry.priority, entry.caller, errCloseTimeout)
}
