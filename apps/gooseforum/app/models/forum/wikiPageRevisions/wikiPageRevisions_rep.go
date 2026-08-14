package wikiPageRevisions

import (
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/queryopt"
	"gorm.io/gorm"
)

func Get(id uint64) (entity Entity) {
	builder().First(&entity, id)
	return
}

// GetLatestApproved 返回某页面最新 approved 修订；无则返回零值。
func GetLatestApproved(pageID uint64) (entity Entity) {
	builder().
		Where(queryopt.Eq("page_id", pageID)).
		Where(queryopt.Eq("status", StatusApproved)).
		Order(queryopt.Desc("revision_no")).
		Order(queryopt.Desc("id")).
		First(&entity)
	return
}

// GetLatestApprovedTx 事务内返回某页面最新 approved 修订；无则返回零值。
func GetLatestApprovedTx(tx *gorm.DB, pageID uint64) (entity Entity) {
	tx.Table(tableName).
		Where(queryopt.Eq("page_id", pageID)).
		Where(queryopt.Eq("status", StatusApproved)).
		Order(queryopt.Desc("revision_no")).
		Order(queryopt.Desc("id")).
		First(&entity)
	return
}

// GetLatestPending 返回某页面最新 pending 修订；无则返回零值。
func GetLatestPending(pageID uint64) (entity Entity) {
	builder().
		Where(queryopt.Eq("page_id", pageID)).
		Where(queryopt.Eq("status", StatusPending)).
		Order(queryopt.Desc("revision_no")).
		Order(queryopt.Desc("id")).
		First(&entity)
	return
}

// ListByPage 返回某页面全部修订（降序）。
func ListByPage(pageID uint64) []*Entity {
	var entities []*Entity
	builder().
		Where(queryopt.Eq("page_id", pageID)).
		Order(queryopt.Desc("revision_no")).
		Order(queryopt.Desc("id")).
		Find(&entities)
	return entities
}

// ListByPageTx 事务内返回某页面全部修订（降序）。
func ListByPageTx(tx *gorm.DB, pageID uint64) []*Entity {
	var entities []*Entity
	tx.Table(tableName).
		Where(queryopt.Eq("page_id", pageID)).
		Order(queryopt.Desc("revision_no")).
		Order(queryopt.Desc("id")).
		Find(&entities)
	return entities
}

// ListPending 分页返回全部 pending 修订（降序）。
func ListPending(page, pageSize int) []*Entity {
	var entities []*Entity
	builder().
		Where(queryopt.Eq("status", StatusPending)).
		Order(queryopt.Desc("created_at")).
		Order(queryopt.Desc("id")).
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&entities)
	return entities
}

// ListByStatus 分页返回指定状态的修订（降序）。
func ListByStatus(status int8, page, pageSize int) []*Entity {
	var entities []*Entity
	builder().
		Where(queryopt.Eq("status", status)).
		Order(queryopt.Desc("created_at")).
		Order(queryopt.Desc("id")).
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&entities)
	return entities
}

func Create(entity *Entity) error {
	return builder().Create(entity).Error
}

func CreateTx(tx *gorm.DB, entity *Entity) error {
	return tx.Table(tableName).Create(entity).Error
}

// UpdateStatusTx 事务内更新修订状态（审核流转）。
func UpdateStatusTx(tx *gorm.DB, id uint64, status int8, reviewedBy uint64, reviewedAt interface{}) error {
	updates := map[string]any{
		"status":      status,
		"reviewed_by": reviewedBy,
	}
	if reviewedAt != nil {
		updates["reviewed_at"] = reviewedAt
	}
	return tx.Table(tableName).Where(queryopt.Eq("id", id)).Updates(updates).Error
}

// SupersedePendingTx 将某页面所有 pending 修订置为 superseded（新编辑提交时）。
func SupersedePendingTx(tx *gorm.DB, pageID uint64) error {
	return tx.Table(tableName).
		Where(queryopt.Eq("page_id", pageID)).
		Where(queryopt.Eq("status", StatusPending)).
		Update("status", StatusSuperseded).Error
}

// CountPending 统计全部 pending 修订数。
func CountPending() int64 {
	var count int64
	builder().Where(queryopt.Eq("status", StatusPending)).Count(&count)
	return count
}

// DeleteByPage 删除某页面的全部修订（页面删除时清理，避免 pending 修订
// 残留进审核队列且 Review 返回 ErrPageNotFound 的幽灵项，review P2）。
func DeleteByPage(pageID uint64) error {
	return builder().Where(queryopt.Eq("page_id", pageID)).Delete(&Entity{}).Error
}
