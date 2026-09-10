package datamigration

import (
	"errors"
	"log/slog"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiPageRevisions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiPages"
	"gorm.io/gorm"
)

type WikiSingleSourceBackfillResult struct {
	PagesSeeded     int
	RevisionsSeeded int
	TopicsSeeded    int
	PostsSeeded     int
	Skipped         int
	Failed          int
	LastFailed      string
}

// BackfillWikiSingleSource 单一事件源架构 v19 回填：
//
//  1. wiki_pages.published_revision_no ← 各页最新 approved 修订的 revision_no
//     （存量页面创建即发布，最大 approved revision_no 即当前发布版本）。
//     无 approved 修订的页面（异常态）跳过，指针保持 0。
//  2. wiki_page_revisions 无变化（已是事件源）。
//  3. topics.wiki_synced_revision_no ← published（标题/摘要/首图当前内容即最新版）。
//  4. posts.wiki_synced_revision_no ← published（首楼快照当前内容即最新版）。
//
// 幂等：指针已 >0 的页面跳过；水印已 = published 的行跳过。
func BackfillWikiSingleSource() WikiSingleSourceBackfillResult {
	return BackfillWikiSingleSourceWithDB(dbconnect.Connect())
}

func BackfillWikiSingleSourceWithDB(conn *gorm.DB) WikiSingleSourceBackfillResult {
	result := WikiSingleSourceBackfillResult{}
	if !conn.Migrator().HasTable(&wikiPages.Entity{}) || !conn.Migrator().HasTable(&wikiPageRevisions.Entity{}) {
		return result
	}

	const batchSize = 200
	var cursor uint64
	for {
		var batch []wikiPages.Entity
		err := conn.Model(&wikiPages.Entity{}).
			Select("id", "topic_id", "published_revision_no").
			Where("id > ?", cursor).
			Order("id ASC").
			Limit(batchSize).
			Find(&batch).Error
		if err != nil {
			result.Failed++
			result.LastFailed = "pages_scan:" + err.Error()
			slog.Error("wiki single source backfill: scan pages failed", "err", err)
			return result
		}
		for i := range batch {
			page := &batch[i]
			cursor = page.Id
			if page.PublishedRevisionNo > 0 {
				result.Skipped++
				continue
			}
			// 最新 approved 修订（无则跳过，异常态）。
			var latest wikiPageRevisions.Entity
			err := conn.Model(&wikiPageRevisions.Entity{}).
				Where("page_id = ?", page.Id).
				Where("status = ?", wikiPageRevisions.StatusApproved).
				Order("revision_no DESC").
				Order("id DESC").
				First(&latest).Error
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					result.Skipped++
					continue
				}
				result.Failed++
				result.LastFailed = "latest_rev:" + err.Error()
				slog.Error("wiki single source backfill: latest revision failed", "pageId", page.Id, "err", err)
				continue
			}
			if latest.RevisionNo == 0 {
				result.Skipped++
				continue
			}
			if err := conn.Model(&wikiPages.Entity{}).
				Where("id = ?", page.Id).
				Update("published_revision_no", latest.RevisionNo).Error; err != nil {
				result.Failed++
				result.LastFailed = "page_ptr:" + err.Error()
				slog.Error("wiki single source backfill: page pointer failed", "pageId", page.Id, "err", err)
				continue
			}
			result.PagesSeeded++
			// 3. topics 水印。
			if page.TopicId > 0 {
				topicRows := conn.Model(&topics.Entity{}).
					Where("id = ?", page.TopicId).
					Where("wiki_synced_revision_no < ?", latest.RevisionNo).
					Update("wiki_synced_revision_no", latest.RevisionNo)
				if topicRows.Error != nil {
					result.Failed++
					result.LastFailed = "topic_wm:" + topicRows.Error.Error()
					slog.Error("wiki single source backfill: topic watermark failed", "topicId", page.TopicId, "err", topicRows.Error)
					continue
				}
				result.TopicsSeeded += int(topicRows.RowsAffected)
				// 4. 首楼 post 水印。
				postRows := conn.Model(&posts.Entity{}).
					Where("topic_id = ?", page.TopicId).
					Where("post_no = 1").
					Where("wiki_synced_revision_no < ?", latest.RevisionNo).
					Update("wiki_synced_revision_no", latest.RevisionNo)
				if postRows.Error != nil {
					result.Failed++
					result.LastFailed = "post_wm:" + postRows.Error.Error()
					slog.Error("wiki single source backfill: post watermark failed", "topicId", page.TopicId, "err", postRows.Error)
					continue
				}
				result.PostsSeeded += int(postRows.RowsAffected)
			}
		}
		if len(batch) < batchSize {
			return result
		}
	}
}
