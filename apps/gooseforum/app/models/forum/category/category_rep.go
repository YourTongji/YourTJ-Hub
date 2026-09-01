package category

import (
	"context"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/queryopt"
	"gorm.io/gorm"
)

func SaveOrCreateById(entity *Entity) int64 {
	if entity.Id == 0 {
		return builder().Create(entity).RowsAffected
	}

	return builder().Save(entity).RowsAffected
}

// SaveTx persists a category inside a caller-owned transaction.
func SaveTx(tx *gorm.DB, entity *Entity) error {
	return tx.Table(tableName).Save(entity).Error
}

func Get(id uint64) (entity Entity) {
	builder().First(&entity, id)
	return
}

// GetWithError 返回实体与查询错误，供需要区分“记录不存在”与“查询失败”的调用方使用。
func GetWithError(id uint64) (entity Entity, err error) {
	err = builder().First(&entity, id).Error
	return
}

// GetWithContext is the cancellable worker/request variant of GetWithError.
func GetWithContext(ctx context.Context, id uint64) (entity Entity, err error) {
	err = dbconnect.ConnectContext(ctx).Table(tableName).First(&entity, id).Error
	return
}

func Count() int64 {
	var count int64
	builder().Count(&count)
	return count
}

func All() (entities []*Entity) {
	builder().Order(queryopt.Asc("sort")).Order(queryopt.Asc("id")).Find(&entities)
	return
}

func DeleteEntity(entity *Entity) int64 {
	return builder().Delete(entity).RowsAffected
}

// DeleteTx removes a category inside a caller-owned transaction.
func DeleteTx(tx *gorm.DB, entity *Entity) error {
	return tx.Table(tableName).Delete(entity).Error
}
