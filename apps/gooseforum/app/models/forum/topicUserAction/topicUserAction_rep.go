package topicUserAction

import (
	"strconv"
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

func GetByTopicId(userId, topicId any) (entity Entity) {
	builder().Where(queryopt.Eq("user_id", userId)).Where(queryopt.Eq("topic_id", topicId)).First(&entity)
	return
}

// SetLiked 设置主题点赞状态，返回是否发生了状态迁移（false = 状态未变化或写入失败）。
func SetLiked(userId, topicId uint64, liked bool) bool {
	return setAt(userId, topicId, "liked_at", timeForState(liked))
}

// SetBookmarked 设置主题收藏状态，返回是否发生了状态迁移。
func SetBookmarked(userId, topicId uint64, bookmarked bool) bool {
	return setAt(userId, topicId, "bookmarked_at", timeForState(bookmarked))
}

// SetWatched 设置主题关注状态，返回是否发生了状态迁移。
func SetWatched(userId, topicId uint64, watched bool) bool {
	return setAt(userId, topicId, "watched_at", timeForState(watched))
}

// SetLikedAt 强制设定主题点赞时间（幂等 upsert，仅覆盖点赞字段，不改变迁移语义）。
func SetLikedAt(userId, topicId uint64, likedAt *time.Time) bool {
	return upsertAt(userId, topicId, "liked_at", likedAt)
}

// SetBookmarkedAt 强制设定主题收藏时间（幂等 upsert）。
func SetBookmarkedAt(userId, topicId uint64, bookmarkedAt *time.Time) bool {
	return upsertAt(userId, topicId, "bookmarked_at", bookmarkedAt)
}

// SetWatchedAt 强制设定主题关注时间（幂等 upsert）。
func SetWatchedAt(userId, topicId uint64, watchedAt *time.Time) bool {
	return upsertAt(userId, topicId, "watched_at", watchedAt)
}

// upsertAt 按 user+topic 唯一键幂等写入指定状态时间（行不存在则创建）。
func upsertAt(userId, topicId uint64, field string, value *time.Time) bool {
	if userId == 0 || topicId == 0 || value == nil {
		return false
	}
	result := builder().Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "topic_id"}},
		DoUpdates: clause.Assignments(map[string]any{field: value}),
	}).Create(&Entity{
		UserId:       userId,
		TopicId:      topicId,
		LikedAt:      valueForField(field, "liked_at", value),
		BookmarkedAt: valueForField(field, "bookmarked_at", value),
		WatchedAt:    valueForField(field, "watched_at", value),
	})
	return result.Error == nil
}

// setAt 原子地设置状态并返回是否发生了迁移：
//   - 取消状态：UPDATE 命中已设置的行才算迁移；
//   - 设置状态：先 UPDATE 未设置的行（命中即迁移），未命中再 INSERT（冲突时静默），
//     只有真正插入新行才算迁移。
//
// 并发下只有一个请求能命中迁移，统计/通知副作用必须只在迁移时执行。
func setAt(userId, topicId uint64, field string, value *time.Time) bool {
	if userId == 0 || topicId == 0 {
		return false
	}
	if value == nil {
		result := builder().
			Where(queryopt.Eq("user_id", userId)).
			Where(queryopt.Eq("topic_id", topicId)).
			Where(field + " IS NOT NULL").
			Updates(map[string]any{field: nil, "updated_at": time.Now()})
		return result.Error == nil && result.RowsAffected > 0
	}

	// 1) 更新当前未设置的行：命中即发生 "未设置 → 已设置" 迁移
	result := builder().
		Where(queryopt.Eq("user_id", userId)).
		Where(queryopt.Eq("topic_id", topicId)).
		Where(field + " IS NULL").
		Updates(map[string]any{field: value, "updated_at": time.Now()})
	if result.Error == nil && result.RowsAffected > 0 {
		return true
	}
	// 2) 行不存在时插入；已存在（并发已设置）则冲突静默，不算迁移
	insert := builder().Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "topic_id"}},
		DoNothing: true,
	}).Create(&Entity{
		UserId:       userId,
		TopicId:      topicId,
		LikedAt:      valueForField(field, "liked_at", value),
		BookmarkedAt: valueForField(field, "bookmarked_at", value),
		WatchedAt:    valueForField(field, "watched_at", value),
	})
	return insert.Error == nil && insert.RowsAffected > 0
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

func ListActiveWatchUserIDsAfter(topicId, afterUserId uint64, excludeUserIds []uint64, limit int) []uint64 {
	if topicId == 0 || limit <= 0 {
		return nil
	}

	userIds := make([]uint64, 0, limit)
	query := builder().
		Select("user_id").
		Where(queryopt.Eq("topic_id", topicId)).
		Where("watched_at IS NOT NULL").
		Where("user_id > ?", afterUserId)
	if len(excludeUserIds) > 0 {
		query = query.Where("user_id NOT IN ?", excludeUserIds)
	}
	query.Order("user_id ASC").Limit(limit).Pluck("user_id", &userIds)
	return userIds
}

type LikedTopicRef struct {
	ID      uint64    `gorm:"column:id"`
	TopicID uint64    `gorm:"column:topic_id"`
	LikedAt time.Time `gorm:"column:liked_at"`
}

func ListLikedTopicRefsBefore(userId uint64, cursor string, limit int) ([]LikedTopicRef, string) {
	if userId == 0 || limit <= 0 {
		return nil, ""
	}

	rows := make([]LikedTopicRef, 0, limit+1)
	cursorID := parseLikedCursor(cursor)
	query := builder().
		Select("id", "topic_id", "liked_at").
		Where(queryopt.Eq("user_id", userId)).
		Where("liked_at IS NOT NULL")
	if cursorID > 0 {
		query = query.Where("id < ?", cursorID)
	}
	query.Order("id DESC").Limit(limit + 1).Find(&rows)

	hasNext := len(rows) > limit
	if hasNext {
		rows = rows[:limit]
	}
	if hasNext && len(rows) > 0 {
		return rows, formatLikedCursor(rows[len(rows)-1].ID)
	}
	return rows, ""
}

type BookmarkedTopicRef struct {
	ID           uint64    `gorm:"column:id"`
	TopicID      uint64    `gorm:"column:topic_id"`
	BookmarkedAt time.Time `gorm:"column:bookmarked_at"`
}

// ListBookmarkedTopicRefsBeforeTime 无损时间游标分页：用户收藏过的主题（按收藏时间倒序）。
// 与楼层收藏共用同一续页谓词（时间 + kindRank + 主键），保证跨表合并不重不漏。
// topic 表 kindRank 恒为 1（post 为 2，rank 小者先排序）。
func ListBookmarkedTopicRefsBeforeTime(userId uint64, before time.Time, beforeKindRank int, beforeID uint64, limit int) []BookmarkedTopicRef {
	if userId == 0 || limit <= 0 {
		return nil
	}
	rows := make([]BookmarkedTopicRef, 0, limit)
	query := builder().
		Select("id", "topic_id", "bookmarked_at").
		Where(queryopt.Eq("user_id", userId)).
		Where("bookmarked_at IS NOT NULL")
	if !before.IsZero() {
		query = query.Where(
			`bookmarked_at < ? OR (bookmarked_at = ? AND ? > ?) OR (bookmarked_at = ? AND ? = ? AND id < ?)`,
			before, before, 1, beforeKindRank, before, 1, beforeKindRank, beforeID,
		)
	}
	query.Order("bookmarked_at DESC, id DESC").Limit(limit).Find(&rows)
	return rows
}

// ListBookmarkedTopicRefsBefore 游标分页：用户收藏过的主题引用（按收藏时间倒序）
func ListBookmarkedTopicRefsBefore(userId uint64, cursor string, limit int) ([]BookmarkedTopicRef, string) {
	if userId == 0 || limit <= 0 {
		return nil, ""
	}

	rows := make([]BookmarkedTopicRef, 0, limit+1)
	cursorID := parseBookmarkCursor(cursor)
	query := builder().
		Select("id", "topic_id", "bookmarked_at").
		Where(queryopt.Eq("user_id", userId)).
		Where("bookmarked_at IS NOT NULL")
	if cursorID > 0 {
		query = query.Where("id < ?", cursorID)
	}
	query.Order("id DESC").Limit(limit + 1).Find(&rows)

	hasNext := len(rows) > limit
	if hasNext {
		rows = rows[:limit]
	}
	if hasNext && len(rows) > 0 {
		return rows, formatBookmarkCursor(rows[len(rows)-1].ID)
	}
	return rows, ""
}

func parseBookmarkCursor(cursor string) uint64 {
	id, err := strconv.ParseUint(cursor, 10, 64)
	if err != nil || id == 0 {
		return 0
	}
	return id
}

func formatBookmarkCursor(id uint64) string {
	if id == 0 {
		return ""
	}
	return strconv.FormatUint(id, 10)
}

func parseLikedCursor(cursor string) uint64 {
	id, err := strconv.ParseUint(cursor, 10, 64)
	if err != nil || id == 0 {
		return 0
	}
	return id
}

func formatLikedCursor(id uint64) string {
	if id == 0 {
		return ""
	}
	return strconv.FormatUint(id, 10)
}
