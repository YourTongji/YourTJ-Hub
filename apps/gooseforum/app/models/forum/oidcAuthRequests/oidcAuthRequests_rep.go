package oidcAuthRequests

import "time"

// Create 创建OIDC授权请求记录
func Create(entity *Entity) error {
	return builder().Create(entity).Error
}

// GetByRequestId 根据授权请求ID获取记录
func GetByRequestId(requestId string) *Entity {
	var entity Entity
	err := builder().Where("request_id = ?", requestId).First(&entity).Error
	if err != nil {
		return nil
	}
	return &entity
}

// GetByAuthCode 根据授权码获取记录（token 端点兑换用）
func GetByAuthCode(authCode string) *Entity {
	var entity Entity
	err := builder().Where("auth_code = ?", authCode).First(&entity).Error
	if err != nil {
		return nil
	}
	return &entity
}

// DeleteExpired 删除已过期的授权请求记录。
func DeleteExpired(now time.Time) int64 {
	result := builder().Where("expires_at < ?", now).Delete(&Entity{})
	return result.RowsAffected
}

// MarkDone 标记登录已完成并写入用户身份（幂等：仅当尚未完成时更新）。
// 返回更新行数，用于并发安全的一次性完成语义。
func MarkDone(requestId string, subject uint64, authTime time.Time) int64 {
	result := builder().
		Where("request_id = ? AND done = ?", requestId, false).
		Updates(map[string]any{
			"done":      true,
			"subject":   subject,
			"auth_time": authTime,
		})
	return result.RowsAffected
}

// UpdateAuthCode 写入授权码（AuthorizeCallback 生成 code 后调用）。
func UpdateAuthCode(requestId, authCode string) error {
	return builder().
		Where("request_id = ?", requestId).
		Update("auth_code", authCode).Error
}

// DeleteByRequestId 删除授权请求记录（token 兑换完成后调用）。
func DeleteByRequestId(requestId string) error {
	return builder().Where("request_id = ?", requestId).Delete(&Entity{}).Error
}

// MarkUsed 将授权码标记为已使用。返回更新行数，用于原子单次兑换。
func MarkUsed(id uint64) int64 {
	result := builder().
		Where("id = ? AND used = ?", id, false).
		Update("used", true)
	return result.RowsAffected
}
