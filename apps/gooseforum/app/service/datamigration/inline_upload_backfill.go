package datamigration

import (
	"errors"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/markdown2html"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/fileUsage"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/postRevisions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/fileusageservice"
	"gorm.io/gorm"
)

// Historical inline writers did not resolve publicUrlPrefix and logged reference
// failures after saving content. Repair tracked uploads before enabling orphan GC.
// Retained/deleted content also pins bytes, without granting public access.
func backfillInlineUploadUsages(conn *gorm.DB, result *FileUsageChatCoverResult) {
	hasTopics := conn.Migrator().HasTable(&topics.Entity{})
	hasPosts := conn.Migrator().HasTable(&posts.Entity{})
	if hasTopics {
		var batch []topics.Entity
		err := conn.Unscoped().FindInBatches(&batch, fileUsageChatCoverBatch, func(_ *gorm.DB, _ int) error {
			for _, topic := range batch {
				status := inlineBackfillStatus(topic.DeletedAt.Valid, topic.VisibilityStatus, topic.RetentionStatus, topic.Status == 1 && topic.ProcessStatus == topics.ProcessStatusNormal)
				if status == fileUsage.UsageStatusActive && topic.FirstPostId != 0 {
					var first posts.Entity
					if !hasPosts {
						status = fileUsage.UsageStatusPending
					} else {
						err := conn.Unscoped().First(&first, "id = ?", topic.FirstPostId).Error
						if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
							return err
						}
						status = inlineBackfillStatus(first.Id == 0 || first.DeletedAt.Valid, first.VisibilityStatus, first.RetentionStatus, first.ProcessStatus == posts.ProcessStatusNormal)
					}
				}
				backfillInlineReferences(conn, result, topic.UserId, fileUsage.TargetTopic, topic.Id, topic.ImageUrls, status)
			}
			return nil
		}).Error
		if err != nil {
			failFileUsageBackfill(result, "topic_inline_scan", err)
		}
	}
	if !hasPosts {
		return
	}
	var batch []posts.Entity
	err := conn.Unscoped().FindInBatches(&batch, fileUsageChatCoverBatch, func(_ *gorm.DB, _ int) error {
		for _, post := range batch {
			var topic topics.Entity
			if hasTopics {
				if err := conn.Unscoped().First(&topic, "id = ?", post.TopicId).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
					return err
				}
			}
			topicStatus := inlineBackfillStatus(topic.Id == 0 || topic.DeletedAt.Valid, topic.VisibilityStatus, topic.RetentionStatus, topic.Status == 1 && topic.ProcessStatus == topics.ProcessStatusNormal)
			status := inlineBackfillStatus(post.DeletedAt.Valid, post.VisibilityStatus, post.RetentionStatus, post.ProcessStatus == posts.ProcessStatusNormal && topicStatus == fileUsage.UsageStatusActive)
			if topicStatus == fileUsage.UsageStatusPurged {
				status = fileUsage.UsageStatusPurged
			}
			backfillInlineReferences(conn, result, post.UserId, fileUsage.TargetPost, post.Id, markdown2html.ExtractImageURLs(post.Content), status)
		}
		return nil
	}).Error
	if err != nil {
		failFileUsageBackfill(result, "post_inline_scan", err)
	}
	if !conn.Migrator().HasTable(&postRevisions.Entity{}) {
		return
	}
	var revisions []postRevisions.Entity
	err = conn.FindInBatches(&revisions, fileUsageChatCoverBatch, func(_ *gorm.DB, _ int) error {
		for _, revision := range revisions {
			var post posts.Entity
			if err := conn.Unscoped().First(&post, "id = ?", revision.PostId).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					continue
				}
				return err
			}
			urls := append(markdown2html.ExtractImageURLs(revision.Content), revision.ImageUrls...)
			backfillInlineReferences(conn, result, post.UserId, fileUsage.TargetPostRevision, revision.Id, urls, fileUsage.UsageStatusPending)
		}
		return nil
	}).Error
	if err != nil {
		failFileUsageBackfill(result, "revision_inline_scan", err)
	}
}

func inlineBackfillStatus(deleted bool, visibility, retention string, public bool) string {
	if deleted || (visibility != "" && visibility != posts.VisibilityActive) || retention == posts.RetentionPurged {
		return fileUsage.UsageStatusPurged
	}
	if public {
		return fileUsage.UsageStatusActive
	}
	return fileUsage.UsageStatusPending
}

func backfillInlineReferences(conn *gorm.DB, result *FileUsageChatCoverResult, userID uint64, target string, targetID uint64, urls []string, status string) {
	result.Processed++
	seen := make(map[string]bool)
	for _, value := range urls {
		name := fileusageservice.FileNameFromURL(value)
		if name == "" || len(name) > 512 || seen[name] {
			continue
		}
		seen[name] = true
		var owners int64
		if err := conn.Model(&fileUsage.Entity{}).Where("file_name = ? AND usage_type = ? AND user_id = ?", name, fileUsage.UsageUploadOwner, userID).Count(&owners).Error; err != nil {
			failFileUsageBackfill(result, "inline_owner_lookup", err)
			continue
		}
		// Untracked legacy files are not candidates for orphan GC. Do not change
		// their visibility or grant references to another user's tracked upload.
		if owners == 0 {
			continue
		}
		owned, err := ownedBackfillFile(name, func(ownerID uint64) bool { return ownerID == userID })
		if err != nil {
			failFileUsageBackfill(result, "inline_file_lookup", err)
			continue
		}
		if !owned {
			continue
		}
		insertUsageIfAbsent(conn, result, "inline_insert", fileUsage.Entity{FileName: name, TargetType: target, TargetId: targetID, UserId: userID, UsageType: fileUsage.UsageInlineImage, Status: status})
	}
}
