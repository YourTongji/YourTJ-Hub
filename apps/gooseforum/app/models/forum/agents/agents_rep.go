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

// UpdateTokenCAS replaces the credential columns only when the current
// token_prefix still matches oldPrefix. It returns the number of affected
// rows: 1 on success, 0 when a concurrent rotation already changed the
// prefix. This makes concurrent rotations fail loudly instead of silently
// discarding one of the two new tokens.
func UpdateTokenCAS(userID uint64, oldPrefix, newPrefix, hash string) (int64, error) {
	result := builder().
		Where(fieldUserId+" = ?", userID).
		Where(fieldTokenPrefix+" = ?", oldPrefix).
		Updates(map[string]any{
			fieldTokenPrefix: newPrefix,
			fieldTokenHash:   hash,
		})
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}

// RevokeCredential disables the agent and clears its token hash. The
// non-secret token prefix is retained as the unique index and the rotation
// CAS anchor; a revoked credential can never validate again, and re-enabling
// requires an explicit rotation first.
func RevokeCredential(userID uint64) error {
	return UpdateColumns(nil, userID, map[string]any{
		fieldEnabled:   StatusDisabled,
		fieldTokenHash: "",
	})
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
