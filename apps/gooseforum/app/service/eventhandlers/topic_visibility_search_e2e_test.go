package eventhandlers

import (
	"os"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/searchservice"
)

// TestTopicVisibilitySearchSyncEndToEnd 在真实 Meilisearch 上验证公开性转变的
// 搜索索引同步（issue #132 核心回归）：
//  1. 发布（Status=1, ProcessStatus=0）→ 索引 upsert，公共搜索可命中；
//  2. 取消发布（Status→0）→ 索引删除，公共搜索不再命中；
//  3. 重新发布（Status→1）→ 索引恢复，公共搜索重新命中；
//  4. 送审（ProcessStatus→Pending）→ 索引删除，公共搜索不再命中；
//  5. 审核批准恢复（ProcessStatus→Normal）→ 索引恢复。
//
// 由 TEST_MEILI_URL 门控（缺省跳过，CI 默认不跑），与 topic_deleted_e2e_test.go
// 同模式。
func TestTopicVisibilitySearchSyncEndToEnd(t *testing.T) {
	if os.Getenv("TEST_MEILI_URL") == "" {
		t.Skip("TEST_MEILI_URL not set; skipping Meilisearch end-to-end test")
	}

	topicID := uint64(time.Now().UnixNano() & 0x7fffffff)
	title := "e2e-visibility-" + time.Now().Format("150405.000")
	topic := &topics.Entity{
		Id:               topicID,
		Title:            title,
		CategoryIds:      []uint64{1},
		Status:           1, // 发布状态
		ProcessStatus:    topics.ProcessStatusNormal,
		VisibilityStatus: topics.VisibilityActive,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
	firstPost := &posts.Entity{
		Id:            topicID + 1,
		TopicId:       topicID,
		PostNo:        1,
		Content:       "visibility e2e first post",
		ProcessStatus: posts.ProcessStatusNormal,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// 1. 发布 → 可搜
	task, err := searchservice.BuildSingleTopicSearchDocument(topic, firstPost)
	if err != nil {
		t.Fatalf("BuildSingleTopicSearchDocument(published) error: %v", err)
	}
	if task == nil {
		t.Fatal("published topic produced no Meilisearch task (service unavailable?)")
	}
	waitTask(t, task.TaskUID)
	if !topicExists(t, topicID, title) {
		t.Fatalf("topic %d not found in index after publish", topicID)
	}

	// 2. 取消发布（Status→0）→ 不可搜（同 UpdateTopicStatus 的下架路径）
	topic.Status = 0
	if _, err := searchservice.BuildSingleTopicSearchDocument(topic, firstPost); err != nil {
		t.Fatalf("BuildSingleTopicSearchDocument(unpublished) error: %v", err)
	}
	waitVisibilityGone(t, topicID, title, "unpublish")

	// 3. 重新发布（Status→1）→ 恢复可搜
	topic.Status = 1
	task, err = searchservice.BuildSingleTopicSearchDocument(topic, firstPost)
	if err != nil {
		t.Fatalf("BuildSingleTopicSearchDocument(re-publish) error: %v", err)
	}
	waitTask(t, task.TaskUID)
	if !topicExists(t, topicID, title) {
		t.Fatalf("topic %d not found in index after re-publish", topicID)
	}

	// 4. 送审（ProcessStatus→Pending）→ 不可搜（敏感内容审核前不外泄）
	topic.ProcessStatus = topics.ProcessStatusPending
	if _, err := searchservice.BuildSingleTopicSearchDocument(topic, firstPost); err != nil {
		t.Fatalf("BuildSingleTopicSearchDocument(pending) error: %v", err)
	}
	waitVisibilityGone(t, topicID, title, "pending review")

	// 5. 审核批准（ProcessStatus→Normal）→ 恢复可搜
	topic.ProcessStatus = topics.ProcessStatusNormal
	task, err = searchservice.BuildSingleTopicSearchDocument(topic, firstPost)
	if err != nil {
		t.Fatalf("BuildSingleTopicSearchDocument(approved) error: %v", err)
	}
	waitTask(t, task.TaskUID)
	if !topicExists(t, topicID, title) {
		t.Fatalf("topic %d not found in index after approval", topicID)
	}

	// 清理：删除索引文档，避免污染其他 e2e
	_, _ = searchservice.BuildSingleTopicSearchDocument(&topics.Entity{
		Id: topicID, Status: 0, ProcessStatus: topics.ProcessStatusBlocked,
	}, nil)
}

// waitVisibilityGone 轮询断言话题从索引移除（Meilisearch 任务异步生效）。
func waitVisibilityGone(t *testing.T, id uint64, title, stage string) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if !topicExists(t, id, title) {
			return // 已移除
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("topic %d (%s) still present in index 10s after %s", id, title, stage)
}
