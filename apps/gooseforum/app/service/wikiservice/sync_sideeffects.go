package wikiservice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiPages"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/fileusageservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/searchservice"
	"gorm.io/gorm"
)

const TaskTypeWikiProjection = "wiki-projection."

type wikiProjectionTask struct {
	HeadSHA string `json:"headSha"`
	PageID  uint64 `json:"pageId"`
	TopicID uint64 `json:"topicId"`
	Notify  bool   `json:"notify"`
}

func enqueueWikiProjectionTaskTx(tx *gorm.DB, headSHA string, pageID, topicID uint64, notify bool) error {
	payload, err := json.Marshal(wikiProjectionTask{
		HeadSHA: headSHA,
		PageID:  pageID,
		TopicID: topicID,
		Notify:  notify,
	})
	if err != nil {
		return err
	}
	return taskQueue.CreateTx(tx, &taskQueue.Entity{
		Type:     TaskTypeWikiProjection + "reconcile",
		Status:   taskQueue.StatusPending,
		TaskJson: string(payload),
	})
}

// RunWikiProjectionTask reconciles side effects from the current projection.
// Reading current rows makes retries monotonic: an old task cannot overwrite a
// newer Git projection after a later sync has committed.
func RunWikiProjectionTask(ctx context.Context, task *taskQueue.Entity) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	var payload wikiProjectionTask
	if err := json.Unmarshal([]byte(task.TaskJson), &payload); err != nil {
		return fmt.Errorf("decode wiki projection task: %w", err)
	}
	if payload.PageID == 0 || payload.TopicID == 0 {
		return errors.New("wiki projection task requires pageId and topicId")
	}

	var page wikiPages.Entity
	if err := dbconnect.Connect().Unscoped().Where("id = ?", payload.PageID).First(&page).Error; err != nil {
		return fmt.Errorf("load wiki page: %w", err)
	}
	topic := topics.UnscopedGet(payload.TopicID)
	if topic.Id == 0 || page.TopicId != topic.Id {
		return errors.New("wiki projection topic binding not found")
	}
	post := posts.UnscopedGet(topic.FirstPostId)
	if post.Id == 0 {
		return errors.New("wiki projection first post not found")
	}

	content := post.Content
	if page.DeletedAt.Valid || topic.DeletedAt.Valid || post.DeletedAt.Valid {
		content = ""
	}
	if err := fileusageservice.ReplaceTopicWithError(topic.Id, wikiSystemUserID, content); err != nil {
		return fmt.Errorf("reconcile wiki file usages: %w", err)
	}
	if _, err := searchservice.BuildSingleTopicSearchDocument(&topic, &post); err != nil {
		return fmt.Errorf("reconcile wiki search: %w", err)
	}
	if payload.Notify && !page.DeletedAt.Valid {
		if err := notifyWatchersThrottled(topic.Id, page.Path, page.Title, wikiSystemUserID); err != nil {
			return fmt.Errorf("notify wiki watchers: %w", err)
		}
	}
	return ctx.Err()
}
