package datamigration

import (
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/markdown2html"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/postRevisions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"gorm.io/gorm"
)

type PostRevisionBackfillResult struct {
	Seeded     int
	Skipped    int
	Failed     int
	LastFailed string
}

// BackfillPostRevisionSeeds 为部署前已存在、尚无任何版本的帖子回填 v1 快照
// （editor = 作者，内容取当前正文）。版本历史首次编辑时只追加新版本，不会
// 回看旧正文；不先回填 v1，存量帖子的原始正文会在首次编辑覆写后永久丢失。
// 幂等：已有版本的帖子跳过；content 为空的帖子（已删除/隐私擦除）跳过。
func BackfillPostRevisionSeeds() PostRevisionBackfillResult {
	return BackfillPostRevisionSeedsWithDB(dbconnect.Connect())
}

func BackfillPostRevisionSeedsWithDB(conn *gorm.DB) PostRevisionBackfillResult {
	const batchSize = 200
	result := PostRevisionBackfillResult{}
	if !conn.Migrator().HasTable(&posts.Entity{}) || !conn.Migrator().HasTable(&postRevisions.Entity{}) {
		return result
	}

	var cursor uint64
	for {
		var batch []posts.Entity
		err := conn.Model(&posts.Entity{}).
			Select("id", "user_id", "content", "rendered_html", "process_status").
			Where("id > ?", cursor).
			Order("id ASC").
			Limit(batchSize).
			Find(&batch).Error
		if err != nil {
			result.Failed++
			result.LastFailed = err.Error()
			return result
		}
		for i := range batch {
			item := &batch[i]
			cursor = item.Id
			if item.Content == "" {
				result.Skipped++
				continue
			}
			var count int64
			if err := conn.Model(&postRevisions.Entity{}).
				Where("post_id = ?", item.Id).
				Count(&count).Error; err != nil {
				result.Failed++
				result.LastFailed = err.Error()
				continue
			}
			if count > 0 {
				result.Skipped++
				continue
			}
			renderedHTML := item.RenderedHTML
			if renderedHTML == "" {
				renderedHTML = markdown2html.PostMarkdownToHTML(item.Content)
			}
			if err := conn.Create(&postRevisions.Entity{
				PostId:        item.Id,
				Version:       1,
				EditorId:      item.UserId,
				Content:       item.Content,
				RenderedHTML:  renderedHTML,
				ProcessStatus: item.ProcessStatus,
			}).Error; err != nil {
				result.Failed++
				result.LastFailed = err.Error()
				continue
			}
			result.Seeded++
		}
		if len(batch) < batchSize {
			return result
		}
	}
}
