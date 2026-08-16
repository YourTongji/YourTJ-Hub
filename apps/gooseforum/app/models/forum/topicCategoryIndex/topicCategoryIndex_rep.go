package topicCategoryIndex

import (
	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/queryopt"
	"github.com/samber/lo"
	"gorm.io/gorm"
)


func GetByTopicId(topicId uint64) (entities []*Entity) {
	builder().Where("topic_id = ?", topicId).Find(&entities)
	return
}


func GetOneByCategoryId(categoryId uint64) (entity Entity) {
	builder().
		Where(queryopt.Eq("category_id", categoryId)).
		Where(queryopt.Eq("effective", 1)).
		First(&entity)
	return
}

// ReplaceTopicCategories 替换话题分类索引（非事务入口）：委托事务变体保证
// 既有行软切换 effective 1/0 与新增分类插入在同一事务内，任一步失败整体回滚。
func ReplaceTopicCategories(topicId uint64, categoryIDs []uint64) error {
	return db.Connect().Transaction(func(tx *gorm.DB) error {
		return ReplaceTopicCategoriesTx(tx, topicId, categoryIDs)
	})
}

// ReplaceTopicCategoriesTx 事务内替换话题分类索引（同 ReplaceTopicCategories，使用 tx 作用域）。
// 既有行软切换 effective 1/0，新增分类插入新行，任一步失败整体回滚。
// 注意：tx 必须是真实事务（db.Connect().Transaction 的 tx），不能传 builder()——
// 后者已携带 Table 子句，再 .Table() 会生成重复表导致 SQL 歧义。
func ReplaceTopicCategoriesTx(tx *gorm.DB, topicId uint64, categoryIDs []uint64) error {
	categoryIDMap := lo.SliceToMap(categoryIDs, func(id uint64) (uint64, bool) {
		return id, true
	})
	var existing []*Entity
	if err := tx.Table(tableName).Where("topic_id = ?", topicId).Find(&existing).Error; err != nil {
		return err
	}
	for _, item := range existing {
		if _, ok := categoryIDMap[item.CategoryId]; ok {
			item.Effective = 1
			if err := tx.Table(tableName).Save(item).Error; err != nil {
				return err
			}
			delete(categoryIDMap, item.CategoryId)
			continue
		}
		item.Effective = 0
		if err := tx.Table(tableName).Save(item).Error; err != nil {
			return err
		}
	}
	for id := range categoryIDMap {
		rs := &Entity{TopicId: topicId, CategoryId: id, Effective: 1}
		if err := tx.Table(tableName).Create(rs).Error; err != nil {
			return err
		}
	}
	return nil
}
