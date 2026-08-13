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

// ListByPostId 按版本号升序返回某帖全部版本。
func ListByPostId(postId uint64) (entities []*Entity) {
	builder().
		Where(queryopt.Eq("post_id", postId)).
		Order(queryopt.Asc("version")).
		Find(&entities)
	return
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

// BlankContentByPostId 清空某帖全部版本的正文（永久删除/隐私擦除级联，
// 与 posts.MarkPurged/MarkPrivacyErased 的清正文语义一致，防止删除后
// 原文仍经版本历史留存）。
func BlankContentByPostId(postId uint64) error {
	return builder().
		Where(queryopt.Eq("post_id", postId)).
		Updates(map[string]any{
			"content":       "",
			"rendered_html": "",
		}).Error
}

// BlankContentByPostIds 批量清空（话题级联删除用，按 post_id 集合）。
func BlankContentByPostIds(postIds []uint64) error {
	if len(postIds) == 0 {
		return nil
	}
	return builder().
		Where(queryopt.In("post_id", postIds)).
		Updates(map[string]any{
			"content":       "",
			"rendered_html": "",
		}).Error
}
