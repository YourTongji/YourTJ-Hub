package userTotp

// GetByUserID 根据用户ID获取TOTP记录
func GetByUserID(userID uint64) *Entity {
	var entity Entity
	err := builder().Where(fieldUserId, userID).First(&entity).Error
	if err != nil {
		return nil
	}
	return &entity
}

// Create 创建TOTP记录
func Create(entity *Entity) error {
	return builder().Create(entity).Error
}

// Save 保存TOTP记录
func Save(entity *Entity) error {
	return builder().Save(entity).Error
}
