package datamigration

import (
	"log/slog"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
	"gorm.io/gorm"
)

type ContentTypeArticleBackfillResult struct {
	Updated    int64
	Failed     int
	LastFailed string
}

// BackfillLegacyPostsContentTypeArticle 将历史存量 content_type == 0 的帖子回填为文章类型 (ContentTypeArticle = 3)。
// 幂等：仅对 content_type == 0 的记录执行 UPDATE。
func BackfillLegacyPostsContentTypeArticle() ContentTypeArticleBackfillResult {
	conn := db.Connect()
	res := BackfillLegacyPostsContentTypeArticleWithDB(conn)
	if res.Failed == 0 && res.Updated > 0 {
		hotdataserve.ClearTopicListCache()
	}
	return res
}

// BackfillLegacyPostsContentTypeArticleWithDB 在指定数据库连接上执行迁移，便于单元测试。
func BackfillLegacyPostsContentTypeArticleWithDB(conn *gorm.DB) ContentTypeArticleBackfillResult {
	res := ContentTypeArticleBackfillResult{}
	if conn == nil || !conn.Migrator().HasTable("posts") {
		return res
	}

	result := conn.Unscoped().Model(&posts.Entity{}).
		Where("content_type = ?", posts.ContentTypeRegular).
		Update("content_type", posts.ContentTypeArticle)

	if result.Error != nil {
		res.Failed++
		res.LastFailed = result.Error.Error()
		slog.Error("backfill legacy posts content_type to article failed", "err", result.Error)
		return res
	}

	res.Updated = result.RowsAffected
	slog.Info("backfill legacy posts content_type to article done", "updated", res.Updated)
	return res
}
