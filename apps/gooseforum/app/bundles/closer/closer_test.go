package closer

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

type testCloser struct {
	closed *bool
}

func (c testCloser) Close() error {
	*c.closed = true
	return nil
}

func resetCloserForTest(t *testing.T) {
	t.Helper()
	mu.Lock()
	entries = nil
	nextSeq = 0
	mu.Unlock()
	closeTimeout = 10 * time.Second
}

func TestCloseAllRunsCallbacksInReverseOrderWithinPriorityAndClears(t *testing.T) {
	resetCloserForTest(t)
	t.Cleanup(func() {
		resetCloserForTest(t)
	})

	var order []int
	Register(func() error {
		order = append(order, 1)
		return nil
	})
	Register(func() error {
		order = append(order, 2)
		return errors.New("ignored")
	})

	CloseAll()

	if want := []int{2, 1}; !reflect.DeepEqual(order, want) {
		t.Fatalf("close order = %#v, want %#v", order, want)
	}

	order = nil
	CloseAll()
	if len(order) != 0 {
		t.Fatalf("CloseAll should clear callbacks after first run")
	}
}

func TestCloseAllRunsCallbacksByPriority(t *testing.T) {
	resetCloserForTest(t)
	t.Cleanup(func() {
		resetCloserForTest(t)
	})

	var order []int
	RegisterPriority(PriorityLogger, func() error {
		order = append(order, 4)
		return nil
	})
	RegisterPriority(PriorityProducer, func() error {
		order = append(order, 1)
		return nil
	})
	RegisterPriority(PriorityFlush, func() error {
		order = append(order, 2)
		return nil
	})
	RegisterPriority(PriorityDatabase, func() error {
		order = append(order, 3)
		return nil
	})

	CloseAll()

	if want := []int{1, 2, 3, 4}; !reflect.DeepEqual(order, want) {
		t.Fatalf("close order = %#v, want %#v", order, want)
	}
}

func TestBindRegistersCloseMethod(t *testing.T) {
	resetCloserForTest(t)
	t.Cleanup(func() {
		resetCloserForTest(t)
	})

	closed := false
	Bind(testCloser{closed: &closed})
	CloseAll()

	if !closed {
		t.Fatalf("bound closer was not closed")
	}
}

func TestCloseAllTimesOutBlockedCallback(t *testing.T) {
	resetCloserForTest(t)
	t.Cleanup(func() {
		resetCloserForTest(t)
	})

	closeTimeout = 20 * time.Millisecond
	RegisterContext(func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	})

	start := time.Now()
	CloseAll()

	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("CloseAll waited too long for blocked callback: %s", elapsed)
	}

	ran := false
	Register(func() error {
		ran = true
		return nil
	})
	CloseAll()
	if !ran {
		t.Fatalf("CloseAll should continue working after a timed out callback")
	}
}

func TestCloseAllHardTimeoutReturnsFromUncooperativeCallback(t *testing.T) {
	resetCloserForTest(t)
	t.Cleanup(func() {
		resetCloserForTest(t)
	})

	closeTimeout = 20 * time.Millisecond
	started := make(chan struct{})
	release := make(chan struct{})
	RegisterContext(func(context.Context) error {
		close(started)
		<-release
		return nil
	})

	done := make(chan struct{})
	go func() {
		CloseAll()
		close(done)
	}()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("uncooperative callback did not start")
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("CloseAll did not enforce its hard timeout")
	}
	close(release)
}

func TestCloseWithTimeoutIncludesEntryDetails(t *testing.T) {
	resetCloserForTest(t)
	t.Cleanup(func() {
		resetCloserForTest(t)
	})

	closeTimeout = time.Millisecond
	err := closeWithTimeout(context.Background(), closerEntry{
		f: func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		},
		caller:   "test.go:42",
		priority: PriorityFlush,
	})
	if err == nil {
		t.Fatal("expected timeout error")
	}

	message := err.Error()
	for _, want := range []string{"close timed out after", "priority=200", "registered_at=test.go:42"} {
		if !strings.Contains(message, want) {
			t.Fatalf("timeout error %q does not contain %q", message, want)
		}
	}
}

func TestCloseAllJoinsErrorsAndPassesCancellationContext(t *testing.T) {
	resetCloserForTest(t)
	t.Cleanup(func() {
		resetCloserForTest(t)
	})

	first := errors.New("first close error")
	second := errors.New("second close error")
	RegisterContext(func(ctx context.Context) error {
		if ctx == nil {
			t.Fatal("close callback received nil context")
		}
		return first
	})
	RegisterContext(func(ctx context.Context) error {
		if ctx == nil {
			t.Fatal("close callback received nil context")
		}
		return second
	})

	err := CloseAll()
	if !errors.Is(err, first) || !errors.Is(err, second) {
		t.Fatalf("CloseAll() error = %v, want both callback errors", err)
	}
}
