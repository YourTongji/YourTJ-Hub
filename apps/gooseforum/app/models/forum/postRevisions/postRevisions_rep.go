package postRevisions

import (
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/queryopt"
	"gorm.io/gorm"
)

// CreateTx 事务内追加版本快照。
func CreateTx(tx *gorm.DB, entity *Entity) error {
	return tx.Table(tableName).Create(entity).Error
}

// Create 非事务追加（仅测试/修复场景使用）。
func Create(entity *Entity) error {
	return builder().Create(entity).Error
}

// ListByPostId 按版本号升序返回某帖全部版本（无分页，仅测试/内部统计使用）。
func ListByPostId(postId uint64) (entities []*Entity) {
	builder().
		Where(queryopt.Eq("post_id", postId)).
		Order(queryopt.Asc("version")).
		Find(&entities)
	return
}

// PageByPostId 按版本号游标分页返回历史（供只读历史接口使用）。
// beforeVersion=0 表示从最新版本开始；否则返回 version <= beforeVersion
// 的版本（cursor 为上一页最早版本号减一，包含语义保证翻页不漏行）。
// 返回按版本号升序的一页，hasMore 表示是否还有更早的版本。
func PageByPostId(postId uint64, beforeVersion uint64, limit int) (entities []*Entity, hasMore bool) {
	query := builder().
		Where(queryopt.Eq("post_id", postId))
	if beforeVersion > 0 {
		query = query.Where(queryopt.Le("version", beforeVersion))
	}
	rows := make([]*Entity, 0, limit+1)
	query.
		Order(queryopt.Desc("version")).
		Limit(limit + 1).
		Find(&rows)
	if len(rows) > limit {
		rows = rows[:limit]
		hasMore = true
	}
	// 反转为升序返回（与前端展示顺序一致，最新版本在末尾）。
	for i, j := 0, len(rows)-1; i < j; i, j = i+1, j-1 {
		rows[i], rows[j] = rows[j], rows[i]
	}
	return rows, hasMore
}

// NextVersionTx 返回该帖下一个版本号（当前最大版本 + 1），事务内使用。
func NextVersionTx(tx *gorm.DB, postId uint64) uint64 {
	var maxVersion uint64
	_ = tx.Table(tableName).
		Where(queryopt.Eq("post_id", postId)).
		Select("COALESCE(MAX(version), 0)").
		Scan(&maxVersion).Error
	return maxVersion + 1
}

// CountByPostIds 批量返回各帖版本数（用于楼层 VO 的 revisionCount）。
func CountByPostIds(postIds []uint64) map[uint64]int64 {
	result := make(map[uint64]int64, len(postIds))
	if len(postIds) == 0 {
		return result
	}
	type row struct {
		PostId uint64
		Count  int64
	}
	rows := make([]row, 0, len(postIds))
	_ = builder().
		Select("post_id, COUNT(*) AS count").
		Where(queryopt.In("post_id", postIds)).
		Group("post_id").
		Scan(&rows).Error
	for _, item := range rows {
		result[item.PostId] = item.Count
	}
	return result
}

// BlankContentByPostIdTx 事务内清空某帖全部版本的正文（永久删除/隐私擦除级联，
// 与 posts.MarkPurged/MarkPrivacyErased 的清正文语义一致，防止删除后
// 原文仍经版本历史留存）。必须与帖子行的状态/正文更新在同一事务内执行。
func BlankContentByPostIdTx(tx *gorm.DB, postId uint64) error {
	return tx.Table(tableName).
		Where(queryopt.Eq("post_id", postId)).
		Updates(map[string]any{
			"content":       "",
			"rendered_html": "",
		}).Error
}

// BlankContentByPostIdsTx 事务内批量清空（话题级联删除用，按 post_id 集合）。
func BlankContentByPostIdsTx(tx *gorm.DB, postIds []uint64) error {
	if len(postIds) == 0 {
		return nil
	}
	return tx.Table(tableName).
		Where(queryopt.In("post_id", postIds)).
		Updates(map[string]any{
			"content":       "",
			"rendered_html": "",
		}).Error
}
