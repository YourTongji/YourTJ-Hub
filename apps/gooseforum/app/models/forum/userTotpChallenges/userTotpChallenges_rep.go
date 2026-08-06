package userTotpChallenges

import (
	"time"
)

func Create(entity *Entity) error {
	return builder().Create(entity).Error
}

func GetByUserIDAndJti(userID uint64, jti string) *Entity {
	var entity Entity
	err := builder().Where(fieldUserId, userID).Where(fieldJti, jti).First(&entity).Error
	if err != nil {
		return nil
	}
	return &entity
}

func MarkConsumed(userID uint64, jti string) error {
	return builder().Where(fieldUserId, userID).Where(fieldJti, jti).Update(fieldConsumedAt, time.Now()).Error
}

func DeleteExpired() error {
	return builder().Where(fieldExpiresAt+" < ?", time.Now()).Delete(&Entity{}).Error
}
