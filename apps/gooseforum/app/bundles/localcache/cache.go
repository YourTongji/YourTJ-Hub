package localcache

import (
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jellydator/ttlcache/v3"
	"github.com/leancodebox/GooseForum/app/bundles/closer"
	"github.com/leancodebox/GooseForum/app/cacheconfig"
	"golang.org/x/sync/singleflight"
)

// errCacheInvalidated is returned when a value was invalidated while its load
// was in flight and the bounded retry could not settle.
var errCacheInvalidated = errors.New("localcache: value invalidated during load")

// Cache is a small in-process cache facade backed by ttlcache.
//
// Invalidation is epoch-based so that a value loaded concurrently with an
// invalidation is never served afterwards: every mutation that can make
// in-flight loads stale (Set, Delete, Clear, UpdateIfPresent) bumps the cache
// epoch, and every entry records the epoch it was produced under. A read
// accepts an entry only when its epoch matches the current one, so a stale
// write that lands after a Delete is treated as a miss and reloaded instead
// of being served. Security-relevant caches (e.g. the user-info snapshot used
// for TokenVersion checks) rely on this under concurrent "revoke all
// sessions" style invalidations.
type Cache[V any] struct {
	MaxEntries uint64

	once  sync.Once
	cache *ttlcache.Cache[string, cachedEntry[V]]
	group singleflight.Group
	epoch atomic.Uint64
}

// cachedEntry couples a value with the cache epoch it was produced under.
// Entries whose epoch no longer matches the current one are stale and must
// not be served.
type cachedEntry[V any] struct {
	value V
	epoch uint64
}

func (c *Cache[V]) init() {
	c.once.Do(func() {
		c.cache = ttlcache.New[string, cachedEntry[V]](
			ttlcache.WithCapacity[string, cachedEntry[V]](c.maxEntries()),
			ttlcache.WithDisableTouchOnHit[string, cachedEntry[V]](),
		)
		go c.cache.Start()
		closer.RegisterPriority(closer.PriorityCache, func() error {
			c.cache.Stop()
			return nil
		})
	})
}

func (c *Cache[V]) maxEntries() uint64 {
	if c.MaxEntries == 0 {
		return cacheconfig.Current().DefaultLocal
	}
	return c.MaxEntries
}

func (c *Cache[V]) GetOrLoad(
	key string,
	getData func() (V, error),
	timeout time.Duration,
) (value V) {
	data, err := c.GetOrLoadE(key, getData, timeout)
	if err != nil {
		slog.Debug("localcache: load failed in GetOrLoad", "key", key, "err", err)
	}
	return data
}

func (c *Cache[V]) GetOrLoadE(
	key string,
	getData func() (V, error),
	timeout time.Duration,
) (V, error) {
	c.init()
	for attempt := 0; attempt < 2; attempt++ {
		if value, ok := c.getValid(key); ok {
			slog.Debug("localcache: hit", "key", key)
			return value, nil
		}
		slog.Debug("localcache: miss", "key", key)

		_, err, _ := c.group.Do(key, func() (any, error) {
			if value, ok := c.getValid(key); ok {
				slog.Debug("localcache: hit after singleflight wait", "key", key)
				return value, nil
			}
			epoch := c.epoch.Load()
			newVal, err := getData()
			if err != nil {
				slog.Debug("localcache: loader error", "key", key, "err", err)
				return *new(V), err
			}
			c.cache.Set(key, cachedEntry[V]{value: newVal, epoch: epoch}, timeout)
			slog.Debug("localcache: stored", "key", key, "ttl", timeout)
			return newVal, nil
		})
		if err != nil {
			slog.Debug("localcache: load failed", "key", key, "err", err)
			return *new(V), err
		}

		// singleflight shared the leader's result with waiters; it may have
		// been produced under a stale epoch, so re-validate before serving.
		if value, ok := c.getValid(key); ok {
			return value, nil
		}
		slog.Debug("localcache: discarding stale in-flight load", "key", key)
	}
	return *new(V), errCacheInvalidated
}

// getValid returns the cached value for key when it was produced under the
// current cache epoch, dropping the entry otherwise.
func (c *Cache[V]) getValid(key string) (V, bool) {
	item := c.cache.Get(key)
	if item == nil {
		return *new(V), false
	}
	entry := item.Value()
	if entry.epoch != c.epoch.Load() {
		slog.Debug("localcache: dropping stale entry", "key", key)
		c.cache.Delete(key)
		return *new(V), false
	}
	return entry.value, true
}

func (c *Cache[V]) Clear() {
	c.init()
	c.cache.DeleteAll()
	c.epoch.Add(1)
}

func (c *Cache[V]) Set(key string, value V, timeout time.Duration) {
	c.init()
	c.epoch.Add(1)
	c.cache.Set(key, cachedEntry[V]{value: value, epoch: c.epoch.Load()}, timeout)
	slog.Debug("localcache: set", "key", key, "ttl", timeout)
}

func (c *Cache[V]) UpdateIfPresent(key string, update func(V) V, timeout time.Duration) bool {
	c.init()
	item := c.cache.Get(key)
	if item == nil {
		return false
	}
	c.epoch.Add(1)
	c.cache.Set(key, cachedEntry[V]{value: update(item.Value().value), epoch: c.epoch.Load()}, timeout)
	slog.Debug("localcache: updated", "key", key, "ttl", timeout)
	return true
}

func (c *Cache[V]) Delete(key string) {
	c.init()
	c.cache.Delete(key)
	c.epoch.Add(1)
	slog.Debug("localcache: deleted", "key", key)
}
