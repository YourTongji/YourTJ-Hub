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
	var entity Entity
	err := builder().Where(fieldUserId, userID).First(&entity).Error
	if err != nil {
		return nil
	}
	return &entity
}

// GetByTokenPrefix 按令牌前缀查找 Agent 记录（前缀非敏感，可建索引）。
func GetByTokenPrefix(prefix string) *Entity {
	var entity Entity
	err := builder().Where(fieldTokenPrefix, prefix).First(&entity).Error
	if err != nil {
		return nil
	}
	return &entity
}

// Save 保存 Agent 记录。
func Save(entity *Entity) error {
	return builder().Save(entity).Error
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
		Where(fieldUserId, userID).
		Update(fieldLastUsedAt, lastUsedAt).Error
}
