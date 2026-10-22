package datamigration

import (
	"errors"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/markdown2html"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// BackfillEmptyTopicExcerpts repairs persisted summaries from the effective
// public first post, never from a pending edit. It does not reorder the feed.
func BackfillEmptyTopicExcerpts(conn *gorm.DB) error {
	var cursor uint64
	for {
		var batch []topics.Entity
		if err := conn.Model(&topics.Entity{}).Select("id").
			Where("id > ? AND excerpt = ?", cursor, "").
			Where(topics.PublicScopePredicate("topics")).
			Where(topics.FirstPostVisiblePredicate("topics")).
			Order("id ASC").Limit(200).Find(&batch).Error; err != nil {
			return err
		}
		for _, row := range batch {
			cursor = row.Id
			if err := conn.Transaction(func(tx *gorm.DB) error {
				var topic topics.Entity
				err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
					Where("id = ? AND excerpt = ?", row.Id, "").
					Where(topics.PublicScopePredicate("topics")).First(&topic).Error
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return nil
				}
				if err != nil {
					return err
				}
				var post posts.Entity
				err = tx.Clauses(clause.Locking{Strength: "UPDATE"}).
					Where("id = ? AND topic_id = ? AND visibility_status = ? AND process_status = ?", topic.FirstPostId, topic.Id, posts.VisibilityActive, posts.ProcessStatusNormal).
					First(&post).Error
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return nil
				}
				if err != nil {
					return err
				}
				excerpt := markdown2html.ExtractDescription(post.Content, 200)
				if excerpt == "" {
					return nil
				}
				return tx.Model(&topic).UpdateColumn("excerpt", excerpt).Error
			}); err != nil {
				return err
			}
		}
		if len(batch) < 200 {
			return nil
		}
	}
}
