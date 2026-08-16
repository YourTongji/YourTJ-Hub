package wikiPages

import (
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/queryopt"
	"gorm.io/gorm"
)

func Get(id uint64) (entity Entity) {
	builder().First(&entity, id)
	return
}

// GetTx 事务内按 id 获取页面（避免单连接测试库下事务内走全局连接死锁）。
func GetTx(tx *gorm.DB, id uint64) (entity Entity) {
	tx.Table(tableName).First(&entity, id)
	return
}

func GetByPath(path string) (entity Entity) {
	builder().Where(queryopt.Eq("path", path)).First(&entity)
	return
}

func GetByTopicId(topicId uint64) (entity Entity) {
	builder().Where(queryopt.Eq("topic_id", topicId)).First(&entity)
	return
}

// ListAll 返回全部页面（按 namespace/sort_order/id 升序）。
// 显式返回查询错误：公开读必须区分 DB 故障与真实空数据，不能吞错（issue #287）。
func ListAll() ([]*Entity, error) {
	var entities []*Entity
	err := builder().
		Order(queryopt.Asc("namespace")).
		Order(queryopt.Asc("sort_order")).
		Order(queryopt.Asc("id")).
		Find(&entities).Error
	return entities, err
}

// ListAllUnscopedTx 在调用方事务内返回全部页面（含软删）。
func ListAllUnscopedTx(tx *gorm.DB) (entities []*Entity, err error) {
	err = tx.Table(tableName).Unscoped().Order(queryopt.Asc("id")).Find(&entities).Error
	return
}

// ListByIDs 按 id 集合批量返回页面（审核队列取 path 用，避免 ListAll 全表扫，
// review N2/查询优化）。
func ListByIDs(ids []uint64) []*Entity {
	if len(ids) == 0 {
		return nil
	}
	var entities []*Entity
	builder().Where(queryopt.In("id", ids)).Find(&entities)
	return entities
}

func Create(entity *Entity) error {
	return builder().Create(entity).Error
}

func CreateTx(tx *gorm.DB, entity *Entity) error {
	return tx.Table(tableName).Create(entity).Error
}

func Save(entity *Entity) error {
	return builder().Save(entity).Error
}

func SaveTx(tx *gorm.DB, entity *Entity) error {
	return tx.Table(tableName).Save(entity).Error
}

// MovePathTx 暂存或移动页面路径。调用方负责在同一事务内写入最终路径。
func MovePathTx(tx *gorm.DB, id uint64, path string) error {
	return tx.Table(tableName).Unscoped().Where(queryopt.Eq("id", id)).Update("path", path).Error
}

func Delete(id uint64) error {
	return builder().Where(queryopt.Eq("id", id)).Delete(&Entity{}).Error
}

// PathExists 判断 path 是否已被占用（排除指定 id）。
func PathExists(path string, excludeID uint64) bool {
	var count int64
	q := builder().Where(queryopt.Eq("path", path))
	if excludeID != 0 {
		q = q.Where(queryopt.Ne("id", excludeID))
	}
	q.Count(&count)
	return count > 0
}

// CountChildren 统计某页面的直接子页面数。
func CountChildren(parentID uint64) int64 {
	var count int64
	builder().Where(queryopt.Eq("parent_id", parentID)).Count(&count)
	return count
}

// GetByPathUnscoped 按 path 取页面（含软删行）：GitHub 同步恢复被删页面时用，
// 复用原 topic/评论/点赞/订阅，而不是新建空页面。
func GetByPathUnscoped(path string) (entity Entity) {
	builder().Unscoped().Where(queryopt.Eq("path", path)).First(&entity)
	return
}

// GetBySourcePathUnscoped 按仓库真实路径取页面（含软删行）：命名空间删除后
// 重建且 URL key 变化时，旧软删页面 path 首段已是旧 key，无法按 path 匹配，
// 需按 source_path（仓库路径稳定）找回复用（review L5）。
func GetBySourcePathUnscoped(sourcePath string) (entity Entity) {
	builder().Unscoped().Where(queryopt.Eq("source_path", sourcePath)).First(&entity)
	return
}

// RestoreSoftDeleted 恢复软删页面（清除 deleted_at）。
func RestoreSoftDeleted(id uint64) error {
	return builder().Unscoped().Where(queryopt.Eq("id", id)).Update("deleted_at", nil).Error
}
