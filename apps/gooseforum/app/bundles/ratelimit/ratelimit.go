// Package ratelimit provides in-process fixed-window rate limiting for
// write operations. It is intentionally dependency-free so it can back
// middleware and business rules (e.g. captcha escalation) alike.
//
// A Store counts requests per key inside a fixed window. The default
// implementation is an in-memory map guarded by a RWMutex with a periodic
// cleaner; a Redis-backed Store can be swapped in later without touching
// callers (multi-instance deployments).
package ratelimit

import (
	"context"
	"sync"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/closer"
	paniclog "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/recovery"
)

// Store counts attempts per key inside fixed windows.
type Store interface {
	// Allow records one attempt for key and reports whether it fits within
	// limit per window. When denied, retryAfter is the time until the current
	// window resets; count is the number of attempts recorded in the window.
	Allow(key string, limit int, window time.Duration) (ok bool, retryAfter time.Duration, count int)
	// Increment records one attempt for key without a limit and returns the
	// new count inside window. Used by business rules such as captcha
	// escalation that count successful actions.
	Increment(key string, window time.Duration) int
	// Count returns the current number of attempts for key inside window,
	// without recording a new attempt. Expired windows report 0.
	Count(key string) int
	// Reset drops all state for key (used by tests and config reloads).
	Reset(key string)
	// ResetAll drops all state.
	ResetAll()
}

type memoryEntry struct {
	count       int
	windowStart time.Time
	window      time.Duration
}

// MemoryStore is the default in-process Store.
type MemoryStore struct {
	mu   sync.RWMutex
	data map[string]*memoryEntry
}

// NewMemoryStore returns an empty MemoryStore.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{data: make(map[string]*memoryEntry)}
}

// Allow records one attempt and applies the fixed-window policy.
func (s *MemoryStore) Allow(key string, limit int, window time.Duration) (bool, time.Duration, int) {
	if limit <= 0 || window <= 0 {
		return true, 0, 0
	}
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.data[key]
	if !ok || now.Sub(entry.windowStart) >= entry.window {
		s.data[key] = &memoryEntry{count: 1, windowStart: now, window: window}
		return true, 0, 1
	}
	entry.count++
	if entry.count > limit {
		retryAfter := entry.window - now.Sub(entry.windowStart)
		return false, retryAfter, entry.count
	}
	return true, 0, entry.count
}

// Increment records one attempt without a limit and returns the new count.
func (s *MemoryStore) Increment(key string, window time.Duration) int {
	if window <= 0 {
		window = time.Minute
	}
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.data[key]
	if !ok || now.Sub(entry.windowStart) >= entry.window {
		s.data[key] = &memoryEntry{count: 1, windowStart: now, window: window}
		return 1
	}
	entry.count++
	return entry.count
}

// Count returns the current count for key inside window without recording.
func (s *MemoryStore) Count(key string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entry, ok := s.data[key]
	if !ok || time.Since(entry.windowStart) >= entry.window {
		return 0
	}
	return entry.count
}

// Reset drops all state for key.
func (s *MemoryStore) Reset(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, key)
}

// ResetAll drops all state.
func (s *MemoryStore) ResetAll() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data = make(map[string]*memoryEntry)
}

// cleanup drops entries whose window has fully elapsed, bounding memory.
func (s *MemoryStore) cleanup() {
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	for key, entry := range s.data {
		if now.Sub(entry.windowStart) >= entry.window {
			delete(s.data, key)
		}
	}
}

var (
	defaultStore  = NewMemoryStore()
	cleanupStopCh = make(chan struct{})
	cleanupOnce   sync.Once
	cleanupWg     sync.WaitGroup
)

// Default returns the process-wide Store used by middleware.
func Default() Store {
	return defaultStore
}

// StartCleanup starts the periodic expired-window cleanup worker.
func StartCleanup() {
	cleanupOnce.Do(func() {
		closer.RegisterPriorityContext(closer.PriorityCache, func(context.Context) error {
			return StopCleanup()
		})
		cleanupWg.Go(func() {
			defer paniclog.Recover("ratelimit_cleanup")
			ticker := time.NewTicker(time.Minute)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					defaultStore.cleanup()
				case <-cleanupStopCh:
					return
				}
			}
		})
	})
}

// StopCleanup stops the periodic cleanup worker.
func StopCleanup() error {
	select {
	case <-cleanupStopCh:
	default:
		close(cleanupStopCh)
	}
	cleanupWg.Wait()
	return nil
}
