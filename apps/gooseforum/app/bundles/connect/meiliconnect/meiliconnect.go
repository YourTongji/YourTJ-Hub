package meiliconnect

import (
	"sync"
	"time"

	"github.com/leancodebox/GooseForum/app/bundles/preferences"
	"github.com/meilisearch/meilisearch-go"
)

var (
	client = getClient()

	// availabilityTTL 控制 IsAvailable 结果缓存时长，var 便于测试覆盖
	availabilityTTL = 30 * time.Second
)

var (
	availabilityMu        sync.Mutex
	availabilityCheckedAt time.Time
	availabilityCached    bool
)

func getClient() meilisearch.ServiceManager {
	url := preferences.Get("meilisearch.url")
	if url == "" {
		return nil
	}
	if preferences.Get("meilisearch.masterkey") != "" {
		key := meilisearch.WithAPIKey(preferences.Get("meilisearch.masterkey"))
		return meilisearch.New(url, key)
	}

	return meilisearch.New(url)
}

func GetClient() meilisearch.ServiceManager {
	return client
}

// IsAvailable 检查 Meilisearch 是否可用，结果缓存 availabilityTTL 时长
func IsAvailable() bool {
	if client == nil {
		return false
	}

	availabilityMu.Lock()
	defer availabilityMu.Unlock()

	if time.Since(availabilityCheckedAt) < availabilityTTL {
		return availabilityCached
	}

	_, err := client.Health()
	availabilityCheckedAt = time.Now()
	availabilityCached = err == nil
	return availabilityCached
}
