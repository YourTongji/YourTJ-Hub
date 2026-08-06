package eventhandlers

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/leancodebox/GooseForum/app/bundles/connect/meiliconnect"
	"github.com/leancodebox/GooseForum/app/models/forum/topics"
	"github.com/leancodebox/GooseForum/app/service/searchservice"
	"github.com/meilisearch/meilisearch-go"
)

// TestHandleTopicDeletedEndToEnd 在真实 Meilisearch 上验证删除主题事件链：
// 发布主题（Status=1, ProcessStatus=0）→ 索引可搜到 → 置 ProcessStatus=1 →
// handleTopicDeleted（事件链核心路径）→ 索引中该文档被移除。
// 由 TEST_MEILI_URL 门控（缺省跳过，CI 默认不跑）。该变量仅作为启用开关；
// Meilisearch 客户端实际连接目标由 config.toml [meilisearch] 决定。
func TestHandleTopicDeletedEndToEnd(t *testing.T) {
	if os.Getenv("TEST_MEILI_URL") == "" {
		t.Skip("TEST_MEILI_URL not set; skipping Meilisearch end-to-end test")
	}

	ctx := context.Background()
	topicID := uint64(time.Now().UnixNano() & 0x7fffffff)
	title := "e2e-delete-verification-" + time.Now().Format("150405.000")

	topic := &topics.Entity{
		Id:            topicID,
		Title:         title,
		CategoryIds:   []uint64{1},
		Status:        1, // 发布状态
		ProcessStatus: 0,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// 1. 发布：构建搜索文档（走 AddDocuments 分支）
	task, err := searchservice.BuildSingleTopicSearchDocument(topic, nil)
	if err != nil {
		t.Fatalf("BuildSingleTopicSearchDocument(published) error: %v", err)
	}
	if task == nil {
		t.Fatal("published topic produced no Meilisearch task (service unavailable?)")
	}
	waitTask(t, task.TaskUID)

	// 2. 断言发布后可以搜到该主题
	if !topicExists(t, topicID, title) {
		t.Fatalf("topic %d (%s) not found in index after publish", topicID, title)
	}

	// 3. 删除：置 ProcessStatus=1 后走 handleTopicDeleted（事件链核心路径）
	topic.ProcessStatus = 1
	if err := handleTopicDeleted(ctx, &TopicDeletedEvent{Topic: topic}); err != nil {
		t.Fatalf("handleTopicDeleted error: %v", err)
	}

	// 4. 轮询断言：删除后索引中该文档被移除（Meilisearch 任务异步生效）
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if !topicExists(t, topicID, title) {
			return // 已移除
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("topic %d (%s) still present in index 10s after delete event", topicID, title)
}

// topicExists 通过 SearchTopics 搜索标题并检查命中。
func topicExists(t *testing.T, id uint64, title string) bool {
	t.Helper()
	resp, err := searchservice.SearchTopics(searchservice.SearchRequest{
		Query: title,
		Limit: 10,
	})
	if err != nil {
		t.Fatalf("SearchTopics error: %v", err)
	}
	for _, hit := range resp.Results {
		if hit.ID == id {
			return true
		}
	}
	return false
}

// waitTask 轮询 Meilisearch 任务直到成功/失败。
func waitTask(t *testing.T, taskUID int64) {
	t.Helper()
	client := meiliconnect.GetClient()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		info, err := client.GetTask(taskUID)
		if err != nil {
			t.Fatalf("GetTask(%d) error: %v", taskUID, err)
		}
		switch info.Status {
		case meilisearch.TaskStatusSucceeded:
			return
		case meilisearch.TaskStatusFailed:
			t.Fatalf("Meilisearch task %d failed: %s", taskUID, info.Error)
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("Meilisearch task %d did not finish within 10s", taskUID)
}
