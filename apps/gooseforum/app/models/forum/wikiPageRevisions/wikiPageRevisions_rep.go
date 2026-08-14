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

// LatestByPages 批量返回多个页面的最新修订（每页一条）：给定页面 id 集合与
// 状态优先级列表（如 [StatusApproved, StatusPending]），取每种状态下 revision_no
// 最大的一条，命中更高优先级状态的页面不再回退。公开导航树/首页热路径用，
// 替代 pageTitles/approvedPageSet 的逐页 GetLatestApproved 查询（review N+1，
// 页面规模增长后这是放大的 DoS 面）。只查 title/page_id/revision_no/status 轻列，
// 不读 content/rendered_html/toc 大字段。
func LatestByPages(pageIDs []uint64, statuses ...int8) map[uint64]*Entity {
	result := make(map[uint64]*Entity, len(pageIDs))
	if len(pageIDs) == 0 {
		return result
	}
	// 一次 SQL 拉取这些页面的全部修订（仅轻列：不含 content/rendered_html/toc 大字段），
	// Go 侧按状态优先级取最新。created_at/editor_id 供首页排序与编辑者展示。
	var entities []*Entity
	builder().
		Select("id", "page_id", "revision_no", "title", "status", "editor_id", "created_at").
		Where(queryopt.In("page_id", pageIDs)).
		Order(queryopt.Desc("revision_no")).
		Order(queryopt.Desc("id")).
		Find(&entities)
	// 按页面分组，逐条写入，高优先级状态覆盖低优先级（rev_no 降序，首条即最新）。
	byPage := make(map[uint64]map[int8]*Entity, len(pageIDs))
	for _, e := range entities {
		if byPage[e.PageId] == nil {
			byPage[e.PageId] = make(map[int8]*Entity)
		}
		if _, exists := byPage[e.PageId][e.Status]; !exists {
			byPage[e.PageId][e.Status] = e
		}
	}
	for _, pageID := range pageIDs {
		for _, status := range statuses {
			if e, ok := byPage[pageID][status]; ok {
				result[pageID] = e
				break
			}
		}
	}
	return result
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

// ListApprovedByPage 返回某页面全部 approved 修订（降序）。
// SQL 层过滤 status，避免 ListByPage 把含 content/rendered_html 大字段的非 approved
// 修订也拉进内存（review：公开修订历史只展示 approved，读入即过滤是浪费）。
func ListApprovedByPage(pageID uint64) []*Entity {
	var entities []*Entity
	builder().
		Where(queryopt.Eq("page_id", pageID)).
		Where(queryopt.Eq("status", StatusApproved)).
		Order(queryopt.Desc("revision_no")).
		Order(queryopt.Desc("id")).
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

// CountByStatus 统计指定状态的修订总数（审核队列分页元信息）。
func CountByStatus(status int8) int64 {
	var count int64
	builder().Where(queryopt.Eq("status", status)).Count(&count)
	return count
}

// DeleteByPage 删除某页面的全部修订（页面删除时清理，避免 pending 修订
// 残留进审核队列且 Review 返回 ErrPageNotFound 的幽灵项，review P2）。
func DeleteByPage(pageID uint64) error {
	return builder().Where(queryopt.Eq("page_id", pageID)).Delete(&Entity{}).Error
}

// DeleteByEditor 删除某编辑者参与的全部修订（注销删号时清理他人页面上
// 残留的本人修订；DeleteByPage 只覆盖本人页面）。幂等：无匹配行即空操作。
func DeleteByEditor(editorID uint64) error {
	return builder().Where(queryopt.Eq("editor_id", editorID)).Delete(&Entity{}).Error
}
