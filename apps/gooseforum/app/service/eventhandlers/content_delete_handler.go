package eventhandlers

import (
	"context"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/llmsservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/searchservice"
	"gorm.io/gorm"
)

// ContentDeletedEvent 内容删除事件（Issue #94）：话题/回复删除后广播，
// 供搜索索引、LLMS 投影缓存、通知等下游清理。
type ContentDeletedEvent struct {
	ContentType  string `json:"contentType"` // "topic" | "post"
	TopicId      uint64 `json:"topicId"`
	PostId       uint64 `json:"postId"`
	DeletedBy    uint64 `json:"deletedBy"`
	DeleteReason string `json:"deleteReason,omitempty"`
}

// handleContentDeleted 删除后的下游清理：
//   - 话题：按当前状态重建或删除搜索索引文档（已删则删，恢复由 ContentRestored 处理）。
//   - 回复：清除 LLMS 投影缓存（正文可能内嵌在导出的单主题 markdown 中）。
func handleContentDeleted(ctx context.Context, event *ContentDeletedEvent) error {
	if event == nil || event.TopicId == 0 {
		return nil
	}
	llmsservice.ClearCache()

	if event.ContentType == "topic" {
		return enqueueSearchProjection(ctx, func(tx *gorm.DB) error {
			return searchservice.EnqueueTopicSearchTask(tx, event.TopicId)
		})
	}
	return nil
}

// ContentRestoredEvent 内容恢复事件：恢复后重建搜索索引与缓存。
type ContentRestoredEvent struct {
	ContentType string `json:"contentType"`
	TopicId     uint64 `json:"topicId"`
	PostId      uint64 `json:"postId"`
}

// handleContentRestored 恢复后的下游重建。
func handleContentRestored(ctx context.Context, event *ContentRestoredEvent) error {
	if event == nil || event.TopicId == 0 {
		return nil
	}
	llmsservice.ClearCache()

	if event.ContentType == "topic" {
		return enqueueSearchProjection(ctx, func(tx *gorm.DB) error {
			return searchservice.EnqueueTopicSearchTask(tx, event.TopicId)
		})
	}
	return nil
}
