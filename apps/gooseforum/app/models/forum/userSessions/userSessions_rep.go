package userSessions

import (
	"time"

	"gorm.io/gorm"
)

// Create 创建会话记录
func Create(entity *Entity) error {
	return builder().Create(entity).Error
}

// GetByJti 根据jti获取会话记录
func GetByJti(jti string) *Entity {
	var entity Entity
	err := builder().Where(fieldJti, jti).First(&entity).Error
	if err != nil {
		return nil
	}
	return &entity
}

// DeleteByJtiAndUserID deletes one session owned by the user, keyed by jti.
func DeleteByJtiAndUserID(userID uint64, jti string) error {
	return builder().Where(fieldJti, jti).Where(fieldUserId, userID).Delete(&Entity{}).Error
}

// DeleteByID 根据id删除会话记录（限用户）
func DeleteByID(userID uint64, id uint64) error {
	return builder().Where(pid, id).Where(fieldUserId, userID).Delete(&Entity{}).Error
}

// ListByUserID 获取用户全部会话，按创建时间倒序
func ListByUserID(userID uint64) ([]Entity, error) {
	var entities []Entity
	err := builder().Where(fieldUserId, userID).Where(fieldExpiresAt+" > ?", time.Now()).Order(fieldCreatedAt + " desc").Find(&entities).Error
	return entities, err
}

// DeleteExpired 清理已过期会话
func DeleteExpired() error {
	return builder().Where(fieldExpiresAt+" < ?", time.Now()).Delete(&Entity{}).Error
}

// DeleteAllByUserIDWithDB deletes every session of the user through the supplied database handle.
func DeleteAllByUserIDWithDB(conn *gorm.DB, userID uint64) error {
	return conn.Where(fieldUserId, userID).Delete(&Entity{}).Error
}

// UpdateExpiresAtByJti 更新会话过期时间（续签时保持同一会话）
func UpdateExpiresAtByJti(jti string, expiresAt time.Time) error {
	return builder().Where(fieldJti, jti).Update(fieldExpiresAt, expiresAt).Error
}
