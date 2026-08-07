package ratelimit

import (
	"sync"
	"testing"
	"time"
)

func TestAllowWithinLimit(t *testing.T) {
	store := NewMemoryStore()
	for i := 0; i < 5; i++ {
		ok, retryAfter, count := store.Allow("a", 5, time.Minute)
		if !ok {
			t.Fatalf("attempt %d denied, want allowed", i+1)
		}
		if retryAfter != 0 {
			t.Fatalf("attempt %d retryAfter = %v, want 0", i+1, retryAfter)
		}
		if count != i+1 {
			t.Fatalf("attempt %d count = %d, want %d", i+1, count, i+1)
		}
	}
}

func TestDenyOverLimitAndRetryAfter(t *testing.T) {
	store := NewMemoryStore()
	for i := 0; i < 3; i++ {
		store.Allow("a", 3, time.Minute)
	}
	ok, retryAfter, count := store.Allow("a", 3, time.Minute)
	if ok {
		t.Fatal("4th attempt allowed, want denied")
	}
	if retryAfter <= 0 || retryAfter > time.Minute {
		t.Fatalf("retryAfter = %v, want (0, 1m]", retryAfter)
	}
	if count != 4 {
		t.Fatalf("count = %d, want 4", count)
	}
}

func TestWindowReset(t *testing.T) {
	store := NewMemoryStore()
	store.Allow("a", 1, 30*time.Millisecond)
	ok, _, _ := store.Allow("a", 1, 30*time.Millisecond)
	if ok {
		t.Fatal("second attempt within window allowed, want denied")
	}
	time.Sleep(40 * time.Millisecond)
	ok, retryAfter, count := store.Allow("a", 1, 30*time.Millisecond)
	if !ok {
		t.Fatal("attempt after window reset denied, want allowed")
	}
	if retryAfter != 0 {
		t.Fatalf("retryAfter = %v, want 0", retryAfter)
	}
	if count != 1 {
		t.Fatalf("count = %d, want 1", count)
	}
}

func TestSeparateKeys(t *testing.T) {
	store := NewMemoryStore()
	store.Allow("a", 1, time.Minute)
	ok, _, _ := store.Allow("b", 1, time.Minute)
	if !ok {
		t.Fatal("distinct key denied, want allowed")
	}
}

func TestZeroLimitOrWindowAlwaysAllowed(t *testing.T) {
	store := NewMemoryStore()
	ok, _, _ := store.Allow("a", 0, time.Minute)
	if !ok {
		t.Fatal("zero limit denied, want allowed")
	}
	ok, _, _ = store.Allow("b", 5, 0)
	if !ok {
		t.Fatal("zero window denied, want allowed")
	}
}

func TestResetAndResetAll(t *testing.T) {
	store := NewMemoryStore()
	store.Allow("a", 1, time.Minute)
	store.Allow("b", 1, time.Minute)
	store.Reset("a")
	if ok, _, _ := store.Allow("a", 1, time.Minute); !ok {
		t.Fatal("key after Reset denied, want allowed")
	}
	store.ResetAll()
	if ok, _, _ := store.Allow("b", 1, time.Minute); !ok {
		t.Fatal("key after ResetAll denied, want allowed")
	}
}

func TestCleanupRemovesExpired(t *testing.T) {
	store := NewMemoryStore()
	store.Allow("expired", 1, 20*time.Millisecond)
	store.Allow("live", 1, time.Hour)
	time.Sleep(30 * time.Millisecond)
	store.cleanup()
	store.mu.RLock()
	_, expired := store.data["expired"]
	_, live := store.data["live"]
	store.mu.RUnlock()
	if expired {
		t.Fatal("expired window still present after cleanup")
	}
	if !live {
		t.Fatal("live window removed by cleanup, want kept")
	}
}

func TestConcurrentAllow(t *testing.T) {
	store := NewMemoryStore()
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				store.Allow("c", 1000, time.Minute)
			}
		}()
	}
	wg.Wait()
	_, _, count := store.Allow("c", 1000, time.Minute)
	if count != 1001 {
		t.Fatalf("count = %d, want 1001", count)
	}
}
