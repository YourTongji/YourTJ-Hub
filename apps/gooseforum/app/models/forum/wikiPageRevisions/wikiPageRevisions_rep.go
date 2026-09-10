package wikiPageRevisions

import (
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/queryopt"
	"gorm.io/gorm"
)

func Get(id uint64) (entity Entity) {
	builder().First(&entity, id)
	return
}

// GetByPageAndRevisionNo 按页面 + 版本号取修订（回滚目标/diff 单侧用）。
func GetByPageAndRevisionNo(pageID uint64, revisionNo int) (entity Entity) {
	builder().
		Where(queryopt.Eq("page_id", pageID)).
		Where(queryopt.Eq("revision_no", revisionNo)).
		First(&entity)
	return
}

// GetByPageAndRevisionNoTx 事务内按页面 + 版本号取修订（回滚锁内重校验用，
// 避免锁外读取的 target 在并发回滚后被物理删除）。
func GetByPageAndRevisionNoTx(tx *gorm.DB, pageID uint64, revisionNo int) (entity Entity) {
	tx.Table(tableName).
		Where(queryopt.Eq("page_id", pageID)).
		Where(queryopt.Eq("revision_no", revisionNo)).
		First(&entity)
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

// ListRecent 分页返回全站修订（可选按页面过滤；管理端版本历史数据源，降序）。
// 写即发布后不再有状态队列，版本历史 = 全部修订；pageID>0 时只列该页历史。
func ListRecent(pageID uint64, page, pageSize int) []*Entity {
	var entities []*Entity
	b := builder()
	if pageID > 0 {
		b = b.Where(queryopt.Eq("page_id", pageID))
	}
	b.Order(queryopt.Desc("created_at")).
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
