package publicationservice

import (
	"context"
	"errors"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/postRevisions"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
)

// AdoptPending upgrades active legacy review entries once. Already versioned
// rows are skipped, including after a partially completed migration is retried.
func AdoptPending(ctx context.Context) error {
	var cursor uint64
	for {
		var batch []posts.Entity
		if err := db.ConnectContext(ctx).Where("id > ? AND latest_revision_id = 0 AND process_status = ? AND visibility_status = ?", cursor, posts.ProcessStatusPending, posts.VisibilityActive).Order("id").Limit(100).Find(&batch).Error; err != nil {
			return err
		}
		if len(batch) == 0 {
			return nil
		}
		for _, post := range batch {
			cursor = post.Id
			var topic topics.Entity
			if err := db.ConnectContext(ctx).First(&topic, post.TopicId).Error; errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			} else if err != nil {
				return err
			}
			if !available(topic, post) || (post.PostNo > 1 && topic.ProcessStatus != topics.ProcessStatusNormal) {
				continue
			}
			if err := Submit(ctx, &topic, &post); err != nil {
				return err
			}
		}
	}
}

// AdoptRejected supplies an exact immutable appeal target for legacy blocked
// forum content. No model evaluation or publication is triggered by migration.
func AdoptRejected(ctx context.Context) error {
	var cursor uint64
	for {
		var batch []posts.Entity
		if err := db.ConnectContext(ctx).Where("id > ? AND latest_revision_id = 0 AND process_status = ? AND visibility_status = ?", cursor, posts.ProcessStatusBlocked, posts.VisibilityActive).Order("id").Limit(100).Find(&batch).Error; err != nil {
			return err
		}
		if len(batch) == 0 {
			return nil
		}
		for _, item := range batch {
			cursor = item.Id
			if err := db.ConnectContext(ctx).Transaction(func(tx *gorm.DB) error {
				var post posts.Entity
				if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&post, item.Id).Error; errors.Is(err, gorm.ErrRecordNotFound) {
					return nil
				} else if err != nil {
					return err
				}
				var topic topics.Entity
				if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&topic, post.TopicId).Error; errors.Is(err, gorm.ErrRecordNotFound) {
					return nil
				} else if err != nil {
					return err
				}
				if post.LatestRevisionId != 0 || post.ProcessStatus != posts.ProcessStatusBlocked || !available(topic, post) || (topic.TopicType == topics.TopicTypeWiki && post.PostNo == 1) || post.RetentionStatus == posts.RetentionPurged || topic.RetentionStatus == topics.RetentionPurged {
					return nil
				}
				revision := snapshot(&topic, &post)
				revision.Version = postRevisions.NextVersionTx(tx, post.Id)
				if err := tx.Create(&revision).Error; err != nil {
					return err
				}
				return tx.Model(&post).Update("latest_revision_id", revision.Id).Error
			}); err != nil {
				return err
			}
		}
	}
}
