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

type TopicExcerptBackfillResult struct {
	Updated    int
	Skipped    int
	Failed     int
	LastFailed string
}

// BackfillTopicExcerpt 按首帖正文重算存量话题摘要（列表卡片 description）。
// issue #1304：摘要生成曾漏掉 Markdown 转义与 HTML 实体解码，修复只影响之后
// 写入的摘要，存量行需一次性重算。幂等：摘要未变化不写库。
func BackfillTopicExcerpt(conn *gorm.DB) TopicExcerptBackfillResult {
	const batchSize = 200
	result := TopicExcerptBackfillResult{}
	if !conn.Migrator().HasTable(&topics.Entity{}) || !conn.Migrator().HasTable(&posts.Entity{}) {
		return result
	}

	var cursor uint64
	for {
		var batch []topics.Entity
		// Unscoped：软删除行也要重算，恢复后不会带着修复前的脏摘要回归。
		if err := conn.Unscoped().Model(&topics.Entity{}).
			Select("id", "excerpt", "first_post_id").
			Where("id > ?", cursor).
			Order("id ASC").
			Limit(batchSize).
			Find(&batch).Error; err != nil {
			result.Failed++
			result.LastFailed = err.Error()
			return result
		}
		if len(batch) == 0 {
			return result
		}

		postIDs := make([]uint64, 0, len(batch))
		for i := range batch {
			cursor = batch[i].Id
			if batch[i].FirstPostId != 0 {
				postIDs = append(postIDs, batch[i].FirstPostId)
			}
		}
		contents := map[uint64]string{}
		if len(postIDs) > 0 {
			var firstPosts []posts.Entity
			if err := conn.Unscoped().Model(&posts.Entity{}).
				Select("id", "content").
				Where("id IN ?", postIDs).
				Find(&firstPosts).Error; err != nil {
				result.Failed++
				result.LastFailed = err.Error()
				continue
			}
			for _, post := range firstPosts {
				contents[post.Id] = post.Content
			}
		}

		for i := range batch {
			item := &batch[i]
			content, ok := contents[item.FirstPostId]
			if !ok || content == "" {
				result.Skipped++
				continue
			}
			excerpt := markdown2html.ExtractDescription(content, 200)
			if excerpt == item.Excerpt {
				result.Skipped++
				continue
			}
			// Unscoped：与扫描一致，软删除行同样写入（否则作用域会静默跳过）。
			// UpdateColumn：避免扰动 topics.updated_at（首页默认排序键）。
			if err := conn.Unscoped().Model(&topics.Entity{}).Where("id = ?", item.Id).
				UpdateColumn("excerpt", excerpt).Error; err != nil {
				result.Failed++
				result.LastFailed = err.Error()
				continue
			}
			result.Updated++
		}
	}
}
