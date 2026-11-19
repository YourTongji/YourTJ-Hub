package meiliconnect

import (
	"context"
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

	// availabilityProbeTimeout 是单次健康探测的请求上限。
	availabilityProbeTimeout = 3 * time.Second
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
//
// meilisearch-go 0.37 起 GetTask 需要 context，这里用同一个带 deadline 的 ctx
// 同时约束单次 HTTP 请求与整段轮询：超时边界不再由最后一次 time.Sleep 决定。
func WaitForTask(task *meilisearch.TaskInfo) error {
	if task == nil || client == nil {
		return nil
	}
	const waitTimeout = 30 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), waitTimeout)
	defer cancel()
	for {
		info, err := client.GetTask(ctx, task.TaskUID)
		if err != nil {
			if ctx.Err() != nil {
				return fmt.Errorf("task %d timed out after %s", task.TaskUID, waitTimeout)
			}
			return fmt.Errorf("get task %d: %w", task.TaskUID, err)
		}
		switch info.Status {
		case meilisearch.TaskStatusSucceeded:
			return nil
		case meilisearch.TaskStatusFailed, meilisearch.TaskStatusCanceled:
			return fmt.Errorf("task %d %s: %s", task.TaskUID, info.Status, info.Error.Message)
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("task %d timed out after %s", task.TaskUID, waitTimeout)
		case <-time.After(100 * time.Millisecond):
		}
	}
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

	// 健康探测是带缓存的最佳努力检查：调用方（含测试）没有可传的 ctx，
	// 用固定上限的局部 ctx 兜住单次请求，避免慢/挂起的 Meilisearch 拖住调用方。
	ctx, cancel := context.WithTimeout(context.Background(), availabilityProbeTimeout)
	defer cancel()
	_, err := client.Health(ctx)
	availabilityCheckedAt = time.Now()
	availabilityCached = err == nil
	return availabilityCached
}
