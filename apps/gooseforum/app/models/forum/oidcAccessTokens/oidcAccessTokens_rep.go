package oidcAccessTokens

import "time"

// Create 创建OIDC访问令牌记录
func Create(entity *Entity) error {
	return builder().Create(entity).Error
}

// GetByTokenId 根据令牌ID获取记录
func GetByTokenId(tokenId string) *Entity {
	var entity Entity
	err := builder().Where("token_id = ?", tokenId).First(&entity).Error
	if err != nil {
		return nil
	}
	return &entity
}

// MarkRevoked 将令牌标记为已撤销
func MarkRevoked(tokenId string) error {
	return builder().Where("token_id = ?", tokenId).Update("revoked", true).Error
}

// DeleteExpired 删除已过期的令牌记录
func DeleteExpired(now time.Time) int64 {
	result := builder().Where("expires_at < ?", now).Delete(&Entity{})
	return result.RowsAffected
}
