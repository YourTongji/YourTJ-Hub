package topicUserStat

import (
	"log/slog"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ReplierStat 单个回复者的重建聚合值。LastReplyAt 必须来自历史回复的
// MAX(created_at)，绝不重放增量路径的 time.Now()（PR #575 review P2）。
type ReplierStat struct {
	UserID      uint64
	ReplyCount  uint32
	LastReplyAt time.Time
}

func SaveOrCreateById(entity *Entity) int64 {
	if entity.Id == 0 {
		return builder().Create(entity).RowsAffected
	}

	return builder().Save(entity).RowsAffected
}

func IncrementUserPost(topicId, userId uint64) error {
	return IncrementUserPostTx(builder(), topicId, userId)
}

// IncrementUserPostTx keeps participant counts in the post write transaction.
func IncrementUserPostTx(tx *gorm.DB, topicId, userId uint64) error {
	now := time.Now()
	return tx.Table(tableName).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "topic_id"}, {Name: "user_id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"reply_count":   gorm.Expr("reply_count + 1"),
			"last_reply_at": now,
		}),
	}).Create(map[string]any{
		"topic_id":      topicId,
		"user_id":       userId,
		"reply_count":   1,
		"last_reply_at": now,
	}).Error
}

func DecrementUserPost(topicId, userId uint64) error {
	return builder().
		Where("topic_id = ? AND user_id = ? AND reply_count > 0", topicId, userId).
		Update("reply_count", gorm.Expr("reply_count - 1")).
		Error
}
func SyncTopicPosters(topicId, excludeUserId uint64) []uint64 {
	ids, err := syncTopicPosters(builder(), topicId, excludeUserId)
	if err != nil {
		slog.Error("sync topic posters failed", "topicId", topicId, "err", err)
		return nil
	}
	return ids
}

// SyncTopicPostersTx 是 SyncTopicPosters 的事务内变体，供绝对重建在
// 调用方事务中读取一致快照；查询失败必须传播错误让重建事务整体回滚，
// 绝不允许把查询失败伪装成"空参与者"静默落库（PR #575 review P1）。
func SyncTopicPostersTx(tx *gorm.DB, topicId, excludeUserId uint64) ([]uint64, error) {
	return syncTopicPosters(tx, topicId, excludeUserId)
}

func syncTopicPosters(b *gorm.DB, topicId, excludeUserId uint64) ([]uint64, error) {
	var activeUserIDs []uint64
	err := b.Model(&Entity{}).
		Where("topic_id = ?", topicId).
		Where("user_id <> ?", excludeUserId).
		Order("reply_count DESC").
		Order("last_reply_at DESC").
		Order("user_id ASC").
		Limit(3).
		Pluck("user_id", &activeUserIDs).Error
	return activeUserIDs, err
}
func DeleteByTopicIDTx(tx *gorm.DB, topicID uint64) error {
	return tx.Where("topic_id = ?", topicID).Delete(&Entity{}).Error
}

// BulkUpsertRepliersTx 集合化写入全部回复者统计：每批最多 500 行的多值 INSERT，
// 控制 PostgreSQL/SQLite 参数数量并避免逐回复 upsert。last_reply_at 用
// 调用方聚合的历史值显式写入，冲突时同样覆盖——重建路径绝不把历史时间
// 戳替换为部署时间。
func BulkUpsertRepliersTx(tx *gorm.DB, topicID uint64, repliers []ReplierStat) error {
	if len(repliers) == 0 {
		return nil
	}
	rows := make([]Entity, 0, len(repliers))
	for _, replier := range repliers {
		rows = append(rows, Entity{
			TopicId:     topicID,
			UserId:      replier.UserID,
			ReplyCount:  replier.ReplyCount,
			LastReplyAt: replier.LastReplyAt,
		})
	}
	return tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "topic_id"}, {Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"reply_count", "last_reply_at"}),
	}).CreateInBatches(&rows, 500).Error
}
