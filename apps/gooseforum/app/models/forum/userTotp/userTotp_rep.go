package userTotp

import (
	"errors"

	"gorm.io/gorm"
)

// GetByUserID 根据用户ID获取TOTP记录
func GetByUserID(userID uint64) (*Entity, error) {
	var entity Entity
	err := builder().Where(fieldUserId, userID).First(&entity).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &entity, nil
}

// Create 创建TOTP记录
func Create(entity *Entity) error {
	return builder().Create(entity).Error
}

// Save 保存TOTP记录
func Save(entity *Entity) error {
	return builder().Save(entity).Error
}
