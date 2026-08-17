package meiliconnect

import (
	"fmt"
	"sync"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
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

// WaitForTask 轮询等待 Meilisearch 任务完成（最多 waitTimeout）。
// 用于删除/写入任务必须顺序执行的场景（如全量重建先清空后写入）。
func WaitForTask(task *meilisearch.TaskInfo) error {
	if task == nil || client == nil {
		return nil
	}
	const waitTimeout = 30 * time.Second
	deadline := time.Now().Add(waitTimeout)
	for time.Now().Before(deadline) {
		info, err := client.GetTask(task.TaskUID)
		if err != nil {
			return fmt.Errorf("get task %d: %w", task.TaskUID, err)
		}
		switch info.Status {
		case meilisearch.TaskStatusSucceeded:
			return nil
		case meilisearch.TaskStatusFailed, meilisearch.TaskStatusCanceled:
			return fmt.Errorf("task %d %s: %s", task.TaskUID, info.Status, info.Error.Message)
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("task %d timed out after %s", task.TaskUID, waitTimeout)
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
