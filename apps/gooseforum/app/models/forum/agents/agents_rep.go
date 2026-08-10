package agents

import (
	"time"

	db "github.com/leancodebox/GooseForum/app/bundles/connect/dbconnect"
	"gorm.io/gorm"
)

func builder() *gorm.DB {
	return db.Connect().Table(tableName)
}

// GetByUserID 根据机器人用户id获取 Agent 记录。
func GetByUserID(userID uint64) *Entity {
	return GetByUserIDWithDB(nil, userID)
}

// GetByUserIDWithDB reads through tx when called from a transaction, so the
// lookup observes the transaction snapshot and does not contend for another
// SQLite connection.
func GetByUserIDWithDB(tx *gorm.DB, userID uint64) *Entity {
	if tx == nil {
		tx = db.Connect()
	}
	var entity Entity
	err := tx.Table(tableName).Where(fieldUserId+" = ?", userID).First(&entity).Error
	if err != nil {
		return nil
	}
	return &entity
}

// GetByTokenPrefix 按令牌前缀查找 Agent 记录（前缀非敏感，可建索引）。
func GetByTokenPrefix(prefix string) *Entity {
	var entity Entity
	err := builder().Where(fieldTokenPrefix+" = ?", prefix).First(&entity).Error
	if err != nil {
		return nil
	}
	return &entity
}

// UpdateColumns updates only fields owned by the current operation. This avoids
// stale full-row snapshots reverting a concurrent disable or token rotation.
func UpdateColumns(tx *gorm.DB, userID uint64, values map[string]any) error {
	if tx == nil {
		tx = db.Connect()
	}
	result := tx.Model(&Entity{}).Where(fieldUserId+" = ?", userID).Updates(values)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		var count int64
		if err := tx.Table(tableName).Where(fieldUserId+" = ?", userID).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			return gorm.ErrRecordNotFound
		}
	}
	return nil
}

// UpdateToken atomically replaces only the credential columns.
func UpdateToken(userID uint64, prefix, hash string) error {
	return UpdateColumns(nil, userID, map[string]any{
		fieldTokenPrefix: prefix,
		fieldTokenHash:   hash,
	})
}

// UpdateEnabled changes only the enabled state.
func UpdateEnabled(userID uint64, enabled int8) error {
	return UpdateColumns(nil, userID, map[string]any{fieldEnabled: enabled})
}

// List 按创建时间倒序返回全部 Agent 记录。
func List() []*Entity {
	var entities []*Entity
	builder().Order(fieldCreatedAt + " desc").Find(&entities)
	return entities
}

// TouchLastUsedAt 更新最近使用时间。
func TouchLastUsedAt(userID uint64, lastUsedAt time.Time) error {
	return builder().
		Where(fieldUserId+" = ?", userID).
		Update(fieldLastUsedAt, lastUsedAt).Error
}
