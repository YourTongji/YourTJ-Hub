package users

import (
	"strconv"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/localcache"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/cacheconfig"
)

// blockedPeersCacheTTL bounds how long a cached reader-level answer may
// outlive an out-of-band relation change. It matches the shared public list
// cache window (hotdataserve topicListCacheTTL), so a relation change that
// never reaches the write path lags a list refresh by at most one list-cache
// window. Relations changed through SetBlockedUser are invalidated instead of
// waiting out this TTL.
const blockedPeersCacheTTL = 5 * time.Second

var blockedPeersCache = localcache.Cache[bool]{MaxEntries: cacheconfig.Current().BlockedPeers}

// HasBlockedPeersCached is the public-list variant of HasBlockedPeers: it
// answers from a short-lived per-user cache so the shared list cache-hit path
// performs no per-request existence query (issue #1129). Block writes
// invalidate the entry, so new or removed relations are visible immediately;
// the TTL only bounds changes that bypass SetBlockedUser. Loader errors are
// not cached and reach the caller, which fails open.
func HasBlockedPeersCached(userID uint64) (bool, error) {
	if userID == 0 {
		return false, nil
	}
	return blockedPeersCache.GetOrLoadE(blockedPeersCacheKey(userID), func() (bool, error) {
		return HasBlockedPeers(userID)
	}, blockedPeersCacheTTL)
}

// InvalidateBlockedPeersCache drops the cached answers for the given users.
// Callers pass both sides of a relation change: a block in either direction
// flips the public-list decision of both users. localcache also rejects a load
// that raced the invalidation, so a stale answer can never be written back.
func InvalidateBlockedPeersCache(userIDs ...uint64) {
	for _, userID := range userIDs {
		if userID == 0 {
			continue
		}
		blockedPeersCache.Delete(blockedPeersCacheKey(userID))
	}
}

func blockedPeersCacheKey(userID uint64) string {
	return "user:blocked-peers:" + strconv.FormatUint(userID, 10)
}
