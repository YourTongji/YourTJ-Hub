package userTotpRecoveryCodes

import (
	"time"
)

// Create 创建恢复码记录
func Create(entity *Entity) error {
	return builder().Create(entity).Error
}

// GetUnusedByHash 根据哈希获取未使用的恢复码
func GetUnusedByHash(userID uint64, codeHash string) *Entity {
	var entity Entity
	err := builder().Where(fieldUserId, userID).Where(fieldCodeHash, codeHash).Where(fieldUsedAt + " IS NULL").First(&entity).Error
	if err != nil {
		return nil
	}
	return &entity
}

// MarkUsedIfUnused 原子地将恢复码标记为已使用；仅当该码此前未被使用时返回 true。
// 通过 UPDATE ... WHERE used_at IS NULL 避免并发双花。
func MarkUsedIfUnused(id uint64) bool {
	result := builder().Where(pid, id).Where(fieldUsedAt+" IS NULL").Update(fieldUsedAt, time.Now())
	return result.RowsAffected > 0
}

// DeleteByUserID 删除用户的全部恢复码
func DeleteByUserID(userID uint64) error {
	return builder().Where(fieldUserId, userID).Delete(&Entity{}).Error
}
