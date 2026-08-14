package localcache

import (
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/cacheconfig"
)

func TestCache_GetOrLoad(t *testing.T) {
	c := Cache[string]{}
	defer stopTestCache(&c)
	loads := 0
	a, _ := c.GetOrLoadE("", func() (string, error) {
		loads++
		return "a", nil
	}, time.Minute)
	b := c.GetOrLoad("", func() (string, error) {
		loads++
		return "b", nil
	}, time.Minute)

	if a != "a" || b != "a" || loads != 1 {
		t.Fatalf("expected cached value after first load, got %q/%q with %d loads", a, b, loads)
	}
}

func TestCache_LimitsEntries(t *testing.T) {
	c := Cache[int]{}
	defer stopTestCache(&c)
	defaultMaxEntries := cacheconfig.Current().DefaultLocal
	for i := range int(defaultMaxEntries) + 10 {
		key := "key-" + strconv.Itoa(i)
		_, _ = c.GetOrLoadE(key, func() (int, error) {
			return i, nil
		}, time.Minute)
	}

	count := c.cache.Len()
	if count > int(defaultMaxEntries) {
		t.Fatalf("cache retained too many entries: %d", count)
	}
}

func TestCache_UsesCustomMaxEntries(t *testing.T) {
	c := Cache[int]{MaxEntries: 3}
	defer stopTestCache(&c)
	for i := range 10 {
		key := "key-" + strconv.Itoa(i)
		_, _ = c.GetOrLoadE(key, func() (int, error) {
			return i, nil
		}, time.Minute)
	}

	count := c.cache.Len()
	if count > int(c.MaxEntries) {
		t.Fatalf("cache retained too many entries: %d", count)
	}
}

func TestCache_ExpiresEntries(t *testing.T) {
	c := Cache[string]{}
	defer stopTestCache(&c)

	loads := 0
	_, _ = c.GetOrLoadE("key", func() (string, error) {
		loads++
		return "a", nil
	}, 20*time.Millisecond)
	time.Sleep(60 * time.Millisecond)
	got, _ := c.GetOrLoadE("key", func() (string, error) {
		loads++
		return "b", nil
	}, time.Minute)

	if got != "b" || loads != 2 {
		t.Fatalf("expected expired entry to reload, got %q with %d loads", got, loads)
	}
}

func TestCache_GetOrLoadE_ReturnsLoaderError(t *testing.T) {
	c := Cache[string]{}
	defer stopTestCache(&c)

	want := errors.New("boom")
	_, err := c.GetOrLoadE("key", func() (string, error) {
		return "", want
	}, time.Minute)
	if !errors.Is(err, want) {
		t.Fatalf("GetOrLoadE() error = %v, want %v", err, want)
	}
}

func TestCache_Set(t *testing.T) {
	c := Cache[string]{}
	defer stopTestCache(&c)

	c.Set("key", "preset", time.Minute)
	got, _ := c.GetOrLoadE("key", func() (string, error) {
		return "loaded", nil
	}, time.Minute)

	if got != "preset" {
		t.Fatalf("Set() value = %q", got)
	}
}

func TestCache_UpdateIfPresent(t *testing.T) {
	c := Cache[int]{}
	defer stopTestCache(&c)

	updated := c.UpdateIfPresent("missing", func(value int) int {
		return value + 1
	}, time.Minute)
	if updated {
		t.Fatal("UpdateIfPresent() updated a missing key")
	}

	c.Set("key", 1, time.Minute)
	updated = c.UpdateIfPresent("key", func(value int) int {
		return value + 1
	}, time.Minute)
	if !updated {
		t.Fatal("UpdateIfPresent() did not update an existing key")
	}

	got, _ := c.GetOrLoadE("key", func() (int, error) {
		return 0, nil
	}, time.Minute)
	if got != 2 {
		t.Fatalf("UpdateIfPresent() value = %d, want 2", got)
	}
}

func TestCache_DeleteAndClear(t *testing.T) {
	c := Cache[string]{}
	defer stopTestCache(&c)

	c.Set("first", "a", time.Minute)
	c.Set("second", "b", time.Minute)
	c.Delete("first")

	got := c.GetOrLoad("first", func() (string, error) {
		return "loaded", nil
	}, time.Minute)
	if got != "loaded" {
		t.Fatalf("Delete() did not remove key, got %q", got)
	}

	c.Clear()
	got = c.GetOrLoad("second", func() (string, error) {
		return "after-clear", nil
	}, time.Minute)
	if got != "after-clear" {
		t.Fatalf("Clear() did not remove entries, got %q", got)
	}
}

func stopTestCache[V any](c *Cache[V]) {
	if c.cache != nil {
		c.cache.Stop()
	}
}

// TestCache_DeleteDiscardsInFlightLoad pins the epoch-based invalidation
// guarantee: a load that started before Delete must not write its (stale)
// result back into the cache. The next read observes the fresh value.
func TestCache_DeleteDiscardsInFlightLoad(t *testing.T) {
	c := Cache[string]{}
	defer stopTestCache(&c)

	firstLoadStarted := make(chan struct{})
	releaseFirstLoad := make(chan struct{})
	loadDone := make(chan struct{})
	loads := 0

	go func() {
		defer close(loadDone)
		_, _ = c.GetOrLoadE("key", func() (string, error) {
			loads++
			if loads == 1 {
				close(firstLoadStarted)
				<-releaseFirstLoad
				return "stale", nil
			}
			return "fresh", nil
		}, time.Minute)
	}()

	<-firstLoadStarted
	c.Delete("key")
	close(releaseFirstLoad)
	<-loadDone

	if loads != 2 {
		t.Fatalf("loads = %d, want 2 (stale load must be retried)", loads)
	}

	got, err := c.GetOrLoadE("key", func() (string, error) {
		return "fresh", nil
	}, time.Minute)
	if err != nil {
		t.Fatalf("GetOrLoadE() error = %v", err)
	}
	if got != "fresh" {
		t.Fatalf("cached value = %q, want fresh (stale in-flight load must be discarded)", got)
	}
}

// TestCache_ClearDiscardsInFlightLoad is the Clear counterpart of the Delete
// test, covering the same epoch guarantee for whole-cache invalidation.
func TestCache_ClearDiscardsInFlightLoad(t *testing.T) {
	c := Cache[string]{}
	defer stopTestCache(&c)

	firstLoadStarted := make(chan struct{})
	releaseFirstLoad := make(chan struct{})
	loadDone := make(chan struct{})
	loads := 0

	go func() {
		defer close(loadDone)
		_, _ = c.GetOrLoadE("key", func() (string, error) {
			loads++
			if loads == 1 {
				close(firstLoadStarted)
				<-releaseFirstLoad
				return "stale", nil
			}
			return "fresh", nil
		}, time.Minute)
	}()

	<-firstLoadStarted
	c.Clear()
	close(releaseFirstLoad)
	<-loadDone

	if loads != 2 {
		t.Fatalf("loads = %d, want 2 (stale load must be retried)", loads)
	}

	got, err := c.GetOrLoadE("key", func() (string, error) {
		return "fresh", nil
	}, time.Minute)
	if err != nil {
		t.Fatalf("GetOrLoadE() error = %v", err)
	}
	if got != "fresh" {
		t.Fatalf("cached value = %q, want fresh", got)
	}
}
