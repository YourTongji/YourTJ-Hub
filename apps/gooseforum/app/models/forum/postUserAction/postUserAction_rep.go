package postUserAction

import (
	"time"

	"github.com/leancodebox/GooseForum/app/bundles/queryopt"
	"gorm.io/gorm/clause"
)

func SaveOrCreateById(entity *Entity) int64 {
	if entity.Id == 0 {
		return builder().Create(entity).RowsAffected
	}
	return builder().Save(entity).RowsAffected
}

func Get(id uint64) (entity Entity) {
	builder().First(&entity, id)
	return
}

func GetByPostId(userId, postId any) (entity Entity) {
	builder().Where(queryopt.Eq("user_id", userId)).Where(queryopt.Eq("post_id", postId)).First(&entity)
	return
}

func SetLiked(userId, postId uint64, liked bool) bool {
	return setAt(userId, postId, "liked_at", timeForState(liked))
}

func SetBookmarked(userId, postId uint64, bookmarked bool) bool {
	return setAt(userId, postId, "bookmarked_at", timeForState(bookmarked))
}

func setAt(userId, postId uint64, field string, value *time.Time) bool {
	if userId == 0 || postId == 0 {
		return false
	}
	if value == nil {
		result := builder().
			Where(queryopt.Eq("user_id", userId)).
			Where(queryopt.Eq("post_id", postId)).
			Where(field + " IS NOT NULL").
			Updates(map[string]any{field: nil, "updated_at": time.Now()})
		return result.Error == nil && result.RowsAffected > 0
	}

	result := builder().
		Where(queryopt.Eq("user_id", userId)).
		Where(queryopt.Eq("post_id", postId)).
		Where(field + " IS NULL").
		Updates(map[string]any{field: value, "updated_at": time.Now()})
	if result.Error != nil || result.RowsAffected > 0 {
		return result.Error == nil && result.RowsAffected > 0
	}

	result = builder().Create(&Entity{
		UserId:       userId,
		PostId:       postId,
		LikedAt:      valueForField(field, "liked_at", value),
		BookmarkedAt: valueForField(field, "bookmarked_at", value),
	})
	return result.Error == nil
}

func ensureAt(userId, postId uint64, field string, value *time.Time) bool {
	if userId == 0 || postId == 0 || value == nil {
		return false
	}
	result := builder().Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "post_id"}},
		DoUpdates: clause.Assignments(map[string]any{field: value}),
	}).Create(&Entity{
		UserId:       userId,
		PostId:       postId,
		LikedAt:      valueForField(field, "liked_at", value),
		BookmarkedAt: valueForField(field, "bookmarked_at", value),
	})
	return result.Error == nil
}

func timeForState(active bool) *time.Time {
	if !active {
		return nil
	}
	now := time.Now()
	return &now
}

func valueForField(field, target string, value *time.Time) *time.Time {
	if field != target {
		return nil
	}
	return value
}

// PostActionCount 楼层点赞数聚合行
type PostActionCount struct {
	PostId uint64 `gorm:"column:post_id"`
	Count  uint64 `gorm:"column:count"`
}

// BookmarkedPostRef 楼层收藏引用（个人主页收藏列表用）
type BookmarkedPostRef struct {
	ID           uint64    `gorm:"column:id"`
	PostID       uint64    `gorm:"column:post_id"`
	BookmarkedAt time.Time `gorm:"column:bookmarked_at"`
}

// ListBookmarkedPostRefsBeforeTime 时间游标分页：用户收藏过的楼层（按收藏时间倒序）。
// 与主题收藏共用时间游标，便于跨表合并排序展示。
func ListBookmarkedPostRefsBeforeTime(userId uint64, before time.Time, limit int) []BookmarkedPostRef {
	if userId == 0 || limit <= 0 {
		return nil
	}
	rows := make([]BookmarkedPostRef, 0, limit)
	query := builder().
		Select("id", "post_id", "bookmarked_at").
		Where(queryopt.Eq("user_id", userId)).
		Where("bookmarked_at IS NOT NULL")
	if !before.IsZero() {
		query = query.Where("bookmarked_at < ?", before)
	}
	query.Order("bookmarked_at DESC, id DESC").Limit(limit).Find(&rows)
	return rows
}

// CountLikesByPostIds 统计一批楼层的点赞数（单条 GROUP BY 查询）
func CountLikesByPostIds(postIds []uint64) map[uint64]uint64 {
	result := make(map[uint64]uint64, len(postIds))
	if len(postIds) == 0 {
		return result
	}
	rows := make([]PostActionCount, 0, len(postIds))
	builder().
		Select("post_id", "COUNT(*) AS count").
		Where("post_id IN ?", postIds).
		Where("liked_at IS NOT NULL").
		Group("post_id").
		Scan(&rows)
	for _, row := range rows {
		result[row.PostId] = row.Count
	}
	return result
}

// GetStateMapByUserAndPostIds 当前用户对一批楼层的点赞/收藏状态
func GetStateMapByUserAndPostIds(userId uint64, postIds []uint64) map[uint64]Entity {
	result := make(map[uint64]Entity, len(postIds))
	if userId == 0 || len(postIds) == 0 {
		return result
	}
	rows := make([]Entity, 0, len(postIds))
	builder().
		Where("user_id = ?", userId).
		Where("post_id IN ?", postIds).
		Find(&rows)
	for _, row := range rows {
		result[row.PostId] = row
	}
	return result
}
