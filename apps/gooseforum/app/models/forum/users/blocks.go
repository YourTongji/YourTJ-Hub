package users

import (
	"errors"
	"time"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const MaxUserBlocks = 1000

var ErrBlockTarget = errors.New("invalid block target")
var ErrBlockLimit = errors.New("block limit reached")
var ErrInteractionBlocked = errors.New("interaction unavailable")

type BlockEntity struct {
	OwnerID      uint64    `gorm:"primaryKey;autoIncrement:false;not null" json:"-"`
	TargetUserID uint64    `gorm:"primaryKey;autoIncrement:false;not null;index" json:"targetUserId"`
	CreatedAt    time.Time `json:"-"`
}

func (BlockEntity) TableName() string { return "user_blocks" }

type BlockedUser struct {
	TargetUserID uint64 `json:"targetUserId"`
	Username     string `json:"username"`
}

func ListBlockedUsers(ownerID uint64) ([]BlockedUser, error) {
	result := make([]BlockedUser, 0)
	err := db.Connect().Table("user_blocks AS b").Select("b.target_user_id, COALESCE(u.username, '') AS username").Joins("LEFT JOIN users AS u ON u.id = b.target_user_id AND u.deleted_at IS NULL").Where("b.owner_id = ?", ownerID).Order("b.target_user_id").Limit(MaxUserBlocks).Scan(&result).Error
	return result, err
}

// LockInteractionUsers serializes block changes and message writes. All callers
// lock the two user rows in ID order before acquiring any conversation locks.
func LockInteractionUsers(tx *gorm.DB, a, b uint64) error {
	var ids []uint64
	return tx.Model(&EntityComplete{}).Select("id").Clauses(clause.Locking{Strength: "UPDATE"}).Where("id IN ?", []uint64{a, b}).Order("id").Find(&ids).Error
}
func interactionBlocked(tx *gorm.DB, a, b uint64) (bool, error) {
	if a == 0 || b == 0 || a == b {
		return false, nil
	}
	var count int64
	err := tx.Model(&BlockEntity{}).Where("(owner_id = ? AND target_user_id = ?) OR (owner_id = ? AND target_user_id = ?)", a, b, b, a).Count(&count).Error
	return count > 0, err
}
func InteractionBlocked(a, b uint64) (bool, error) { return interactionBlocked(db.Connect(), a, b) }
func CheckInteractionAllowed(tx *gorm.DB, a, b uint64) error {
	if err := LockInteractionUsers(tx, a, b); err != nil {
		return err
	}
	blocked, err := interactionBlocked(tx, a, b)
	if err != nil {
		return err
	}
	if blocked {
		return ErrInteractionBlocked
	}
	return nil
}
func SetBlockedUser(ownerID, targetID uint64, blocked bool) error {
	if ownerID == 0 || targetID == 0 || ownerID == targetID {
		return ErrBlockTarget
	}
	return db.Connect().Transaction(func(tx *gorm.DB) error {
		if err := LockInteractionUsers(tx, ownerID, targetID); err != nil {
			return err
		}
		if !blocked {
			return tx.Where("owner_id = ? AND target_user_id = ?", ownerID, targetID).Delete(&BlockEntity{}).Error
		}
		var owner EntityComplete
		if err := tx.First(&owner, ownerID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrBlockTarget
			}
			return err
		}
		var target EntityComplete
		if err := tx.First(&target, targetID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrBlockTarget
			}
			return err
		}
		var existing int64
		if err := tx.Model(&BlockEntity{}).Where("owner_id = ? AND target_user_id = ?", ownerID, targetID).Count(&existing).Error; err != nil {
			return err
		}
		if existing > 0 {
			return nil
		}
		var count int64
		if err := tx.Model(&BlockEntity{}).Where("owner_id = ?", ownerID).Count(&count).Error; err != nil {
			return err
		}
		if count >= MaxUserBlocks {
			return ErrBlockLimit
		}
		return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&BlockEntity{OwnerID: ownerID, TargetUserID: targetID}).Error
	})
}

// FilterInteractionRecipients makes block filtering bounded per fan-out, and
// fails closed on storage errors. The reverse relation is never exposed to users.
func FilterInteractionRecipients(actor uint64, recipients []uint64) ([]uint64, error) {
	if actor == 0 || len(recipients) == 0 {
		return recipients, nil
	}
	var rows []BlockEntity
	err := db.Connect().Where("(owner_id = ? AND target_user_id IN ?) OR (target_user_id = ? AND owner_id IN ?)", actor, recipients, actor, recipients).Find(&rows).Error
	if err != nil {
		return nil, err
	}
	blocked := make(map[uint64]bool, len(rows))
	for _, row := range rows {
		if row.OwnerID == actor {
			blocked[row.TargetUserID] = true
		} else {
			blocked[row.OwnerID] = true
		}
	}
	result := make([]uint64, 0, len(recipients))
	for _, id := range recipients {
		if !blocked[id] {
			result = append(result, id)
		}
	}
	return result, nil
}
