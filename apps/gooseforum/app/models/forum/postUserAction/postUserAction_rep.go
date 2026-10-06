package postUserAction

import (
	"context"
	"fmt"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/feed"
	"gorm.io/gorm"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/queryopt"
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

// SetLiked 设置楼层点赞状态，返回是否发生了状态迁移（false = 状态未变化或写入失败）。
func SetLiked(userId, postId uint64, liked bool) bool {
	return setAt(userId, postId, "liked_at", timeForState(liked))
}

// SetBookmarked 设置楼层收藏状态，返回是否发生了状态迁移。
func SetBookmarked(userId, postId uint64, bookmarked bool) bool {
	return setAt(userId, postId, "bookmarked_at", timeForState(bookmarked))
}

// setAt 原子地设置状态并返回是否发生了迁移：
//   - 取消状态：UPDATE 命中已设置的行才算迁移；
//   - 设置状态：先 UPDATE 未设置的行（命中即迁移），未命中再 INSERT（冲突时静默），
//     只有真正插入新行才算迁移。
//
// 并发下（如双端同时点赞）只有一个请求能命中迁移，统计/通知副作用必须只在迁移时执行，
// 避免重复计数与重复通知导致统计永久漂移。
func setAt(userId, postId uint64, field string, value *time.Time) bool {
	changed, _ := SetState(context.Background(), userId, postId, 0, field, value != nil)
	return changed
}
func SetState(ctx context.Context, uid, postID, topicID uint64, field string, active bool) (bool, error) {
	if uid == 0 || postID == 0 {
		return false, nil
	}
	if field != "liked_at" && field != "bookmarked_at" {
		return false, fmt.Errorf("unsupported post action")
	}
	changed := false
	err := dbconnect.ConnectContext(ctx).Transaction(func(tx *gorm.DB) error {
		value := timeForState(active)
		condition := field + " IS NOT NULL"
		if active {
			condition = field + " IS NULL"
		}
		r := tx.Model(&Entity{}).Where("user_id = ? AND post_id = ?", uid, postID).Where(condition).Updates(map[string]any{field: value, "updated_at": time.Now()})
		if r.Error != nil {
			return r.Error
		}
		changed = r.RowsAffected > 0
		if !changed && active {
			r = tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "user_id"}, {Name: "post_id"}}, DoNothing: true}).Create(&Entity{UserId: uid, PostId: postID, LikedAt: valueForField(field, "liked_at", value), BookmarkedAt: valueForField(field, "bookmarked_at", value)})
			if r.Error != nil {
				return r.Error
			}
			changed = r.RowsAffected > 0
		}
		if !changed || topicID == 0 {
			return nil
		}
		kind := "post_like"
		if field == "bookmarked_at" {
			kind = "post_bookmark"
		}
		return feed.EventTx(tx, uid, topicID, postID, kind, active)
	})
	return changed && err == nil, err
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

// ListBookmarkedPostRefsBeforeTime 无损时间游标分页：用户收藏过的楼层（按收藏时间倒序）。
// 与主题收藏共用同一续页谓词（时间 + kindRank + 主键），保证跨表合并不重不漏。
// post 表 kindRank 恒为 2（topic 为 1，rank 小者先排序）。
func ListBookmarkedPostRefsBeforeTime(userId uint64, before time.Time, beforeKindRank int, beforeID uint64, limit int) []BookmarkedPostRef {
	if userId == 0 || limit <= 0 {
		return nil
	}
	rows := make([]BookmarkedPostRef, 0, limit)
	query := builder().
		Select("id", "post_id", "bookmarked_at").
		Where(queryopt.Eq("user_id", userId)).
		Where("bookmarked_at IS NOT NULL")
	if !before.IsZero() {
		query = query.Where(
			`bookmarked_at < ? OR (bookmarked_at = ? AND ? > ?) OR (bookmarked_at = ? AND ? = ? AND id < ?)`,
			before, before, 2, beforeKindRank, before, 2, beforeKindRank, beforeID,
		)
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
