package searchservice

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/meiliconnect"
	"github.com/meilisearch/meilisearch-go"
)

// TestCleanupGhostDocumentsE2E 在真实 Meilisearch 上验证幽灵文档清理（issue #139）：
// 索引中存在、但期望集合（数据库）中不存在的文档会被删除；不存在的索引视为无文档可清理。
// 使用独立测试索引，不触碰生产 topics/users/categories 索引。
// 由 TEST_MEILI_URL 门控（缺省跳过，CI 默认不跑）。
func TestCleanupGhostDocumentsE2E(t *testing.T) {
	if os.Getenv("TEST_MEILI_URL") == "" {
		t.Skip("TEST_MEILI_URL not set; skipping ghost cleanup e2e test")
	}
	client := meiliconnect.GetClient()
	unique := time.Now().UnixNano() & 0x7fffffff
	indexName := fmt.Sprintf("ghost-cleanup-e2e-%d", unique%1000000)

	// 1. 创建隔离测试索引
	if _, err := client.CreateIndex(&meilisearch.IndexConfig{Uid: indexName, PrimaryKey: "id"}); err != nil {
		t.Fatalf("CreateIndex(%s) error: %v", indexName, err)
	}
	waitIndexTasks(t, indexName)
	defer func() {
		_, _ = client.DeleteIndex(indexName)
	}()

	index := client.Index(indexName)

	// 2. 写入 4 个文档（ID 1-4），其中 1、3 是幽灵文档（不在期望集合）
	pk := "id"
	if _, err := index.AddDocuments([]map[string]any{
		{"id": 1, "title": "ghost-one"},
		{"id": 2, "title": "kept-two"},
		{"id": 3, "title": "ghost-three"},
		{"id": 4, "title": "kept-four"},
	}, &meilisearch.DocumentOptions{PrimaryKey: &pk}); err != nil {
		t.Fatalf("AddDocuments error: %v", err)
	}
	waitIndexTasks(t, indexName)

	// 3. 清理：只保留 2、4
	deleted, err := cleanupGhostDocuments(index, map[string]struct{}{"2": {}, "4": {}}, nil)
	if err != nil {
		t.Fatalf("cleanupGhostDocuments error: %v", err)
	}
	if len(deleted) != 2 {
		t.Fatalf("removed = %d, want 2", len(deleted))
	}
	waitIndexTasks(t, indexName)

	// 4. 断言索引中只剩 2、4
	ids, err := fetchIndexDocumentIDs(index)
	if err != nil {
		t.Fatalf("fetchIndexDocumentIDs error: %v", err)
	}
	if len(ids) != 2 {
		t.Fatalf("docs after cleanup = %v, want [2 4]", ids)
	}
	seen := map[string]bool{}
	for _, id := range ids {
		seen[id] = true
	}
	if !seen["2"] || !seen["4"] {
		t.Fatalf("docs after cleanup = %v, want [2 4]", ids)
	}

	// 5. 不存在的索引视为无文档可清理（不报错）
	missingName := fmt.Sprintf("ghost-cleanup-missing-%d", unique%1000000)
	if _, err := cleanupGhostDocuments(client.Index(missingName), map[string]struct{}{"1": {}}, nil); err != nil {
		t.Fatalf("cleanup on missing index should not error: %v", err)
	}
}

// waitIndexTasks 轮询指定索引的任务直到全部进入终态。
// GetTasks 返回错误时立即失败（保留错误信息），避免连接/配置故障
// 被静默重试掩盖到超时（PR #151 review 建议）。
func waitIndexTasks(t *testing.T, indexUID string) {
	t.Helper()
	client := meiliconnect.GetClient()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		tasks, err := client.GetTasks(&meilisearch.TasksQuery{IndexUIDS: []string{indexUID}})
		if err != nil {
			t.Fatalf("GetTasks(%s) error: %v", indexUID, err)
		}
		done := true
		for _, task := range tasks.Results {
			if task.Status == meilisearch.TaskStatusEnqueued || task.Status == meilisearch.TaskStatusProcessing {
				done = false
				break
			}
			if task.Status == meilisearch.TaskStatusFailed {
				t.Fatalf("meilisearch task failed: %s", task.Error)
			}
		}
		if done {
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatal("meilisearch tasks did not settle within 10s")
}
