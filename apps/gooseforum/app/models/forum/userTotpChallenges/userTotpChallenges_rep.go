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

// MarkConsumedIfUnconsumed atomically marks a challenge token as consumed,
// returning whether it was still unconsumed. The WHERE consumed_at IS NULL
// guard closes the concurrent-replay race: only one request wins.
func MarkConsumedIfUnconsumed(userID uint64, jti string) bool {
	result := builder().Where(fieldUserId, userID).Where(fieldJti, jti).
		Where(fieldConsumedAt+" IS NULL").Update(fieldConsumedAt, time.Now())
	return result.RowsAffected > 0
}

func DeleteExpired() error {
	return builder().Where(fieldExpiresAt+" < ?", time.Now()).Delete(&Entity{}).Error
}
