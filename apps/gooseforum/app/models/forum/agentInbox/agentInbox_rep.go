package agentInbox

import (
	"time"

	"github.com/leancodebox/GooseForum/app/bundles/pageutil"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// UpsertTx 按唯一键 (agent_id, topic_id, post_id) 插入或刷新一行，必须在
// 调用方事务内执行。冲突时重置为未读 + 待投递，并刷新事件/预览/actor/时间。
func UpsertTx(tx *gorm.DB, entity *Entity) error {
	return tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: fieldAgentId}, {Name: fieldTopicId}, {Name: fieldPostId}},
		DoUpdates: clause.AssignmentColumns([]string{
			"event_type",
			"actor_id",
			"content_preview",
			"status",
			"delivery_status",
			"attempts",
			"last_error",
			"read_at",
			"created_at",
			"updated_at",
		}),
	}).Create(entity).Error
}

// GetIdByKeyTx 在事务内按唯一键取回行 id（upsert 后主键不保证回填）。
func GetIdByKeyTx(tx *gorm.DB, agentID, topicID, postID uint64) (uint64, error) {
	var id uint64
	err := tx.Table(tableName).
		Where(fieldAgentId, agentID).
		Where(fieldTopicId, topicID).
		Where(fieldPostId, postID).
		Limit(1).
		Pluck("id", &id).Error
	return id, err
}

// GetByID 返回单行（worker 使用，不限定 Agent）。
func GetByID(id uint64) *Entity {
	var entity Entity
	if err := builder().Where("id", id).First(&entity).Error; err != nil {
		return nil
	}
	return &entity
}

// GetOwned 返回属于指定 Agent 的单行；跨 Agent 或不存在同样返回错误，
// 由调用方映射为统一的业务失败，避免存在性泄露。
func GetOwned(id, agentID uint64) (Entity, error) {
	var entity Entity
	err := builder().
		Where("id", id).
		Where(fieldAgentId, agentID).
		First(&entity).Error
	return entity, err
}

// PageResult 与 topics.Page 相同的 hasNext 分页形状。
type PageResult struct {
	Page     int
	PageSize int
	HasNext  bool
	Data     []Entity
}

// PageByAgent 列出某 Agent 的收件箱。status 为 nil 表示全部状态。
func PageByAgent(agentID uint64, status *uint8, page, pageSize int) PageResult {
	var list []Entity
	page = max(page-1, 0)
	pageSize = pageutil.BoundPageSize(pageSize)
	queryLimit := pageSize + 1
	b := builder().Where(fieldAgentId, agentID)
	if status != nil {
		b = b.Where(fieldStatus, *status)
	}
	b.Order("created_at desc").Order("id desc").
		Limit(queryLimit).Offset(pageSize * page).Find(&list)
	hasNext := len(list) > pageSize
	if hasNext {
		list = list[:pageSize]
	}
	return PageResult{Page: page + 1, PageSize: pageSize, HasNext: hasNext, Data: list}
}

// MarkRead 幂等标记已读（已读行再次调用仍成功）。
func MarkRead(id, agentID uint64) error {
	result := builder().
		Where("id", id).
		Where(fieldAgentId, agentID).
		Updates(map[string]any{
			fieldStatus: StatusRead,
			fieldReadAt: time.Now(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// MarkAllRead 将某 Agent 的全部收件箱标记已读（空收件箱也成功）。
func MarkAllRead(agentID uint64) error {
	return builder().
		Where(fieldAgentId, agentID).
		Updates(map[string]any{
			fieldStatus: StatusRead,
			fieldReadAt: time.Now(),
		}).Error
}

// DeleteOwned 删除某 Agent 的单个收件箱行。
func DeleteOwned(id, agentID uint64) error {
	result := builder().
		Where("id", id).
		Where(fieldAgentId, agentID).
		Delete(&Entity{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// DeleteAll 清空某 Agent 的全部收件箱（空收件箱也成功）。
func DeleteAll(agentID uint64) error {
	return builder().Where(fieldAgentId, agentID).Delete(&Entity{}).Error
}

// MarkDelivered 记录投递成功（终态）。
func MarkDelivered(id uint64) error {
	return builder().Where("id", id).Updates(map[string]any{
		"delivery_status": DeliveryDelivered,
		"last_error":      "",
	}).Error
}

// RecordFailure 增加一次失败尝试并记录清洗后的错误，返回新的尝试次数。
func RecordFailure(id uint64, sanitizedError string) (uint8, error) {
	err := builder().Where("id", id).UpdateColumn(fieldAttempts, gorm.Expr(fieldAttempts+" + 1")).Error
	if err != nil {
		return 0, err
	}
	err = builder().Where("id", id).Update("last_error", sanitizedError).Error
	if err != nil {
		return 0, err
	}
	var entity Entity
	if err := builder().Where("id", id).First(&entity).Error; err != nil {
		return 0, err
	}
	return entity.Attempts, nil
}

// MarkFailed 终态失败（不再自动重试；inbox 行保留失败事实）。
func MarkFailed(id uint64, sanitizedError string) error {
	return builder().Where("id", id).Updates(map[string]any{
		"delivery_status": DeliveryFailed,
		"last_error":      sanitizedError,
	}).Error
}

// MarkSkipped 终态跳过（Agent 禁用/删除/无端点等）。
func MarkSkipped(id uint64, reason string) error {
	return builder().Where("id", id).Updates(map[string]any{
		"delivery_status": DeliverySkipped,
		"last_error":      reason,
	}).Error
}
