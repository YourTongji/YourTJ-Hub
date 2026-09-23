package users

import (
	"errors"
	"time"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const MaxPrivateNotes = 1000

var ErrPrivateNoteTarget = errors.New("invalid private note target")
var ErrPrivateNoteLimit = errors.New("private note limit reached")

// PrivateNoteEntity belongs to the author, never to the target's public profile.
type PrivateNoteEntity struct {
	OwnerID      uint64    `gorm:"primaryKey;autoIncrement:false;not null" json:"-"`
	TargetUserID uint64    `gorm:"primaryKey;autoIncrement:false;not null;index" json:"targetUserId"`
	Note         string    `gorm:"type:text;not null" json:"note"`
	UpdatedAt    time.Time `json:"-"`
}

func (PrivateNoteEntity) TableName() string { return "user_private_notes" }

type PrivateNote struct {
	TargetUserID uint64 `json:"targetUserId"`
	Username     string `json:"username"`
	Note         string `json:"note"`
}

func ListPrivateNotes(ownerID uint64) ([]PrivateNote, error) {
	result := make([]PrivateNote, 0)
	err := db.Connect().Table("user_private_notes AS n").Select("n.target_user_id, u.username, n.note").Joins("JOIN users AS u ON u.id = n.target_user_id AND u.deleted_at IS NULL").Where("n.owner_id = ?", ownerID).Order("n.target_user_id").Limit(MaxPrivateNotes).Scan(&result).Error
	return result, err
}

func SetPrivateNote(ownerID, targetID uint64, note string) error {
	if ownerID == 0 || targetID == 0 || ownerID == targetID {
		return ErrPrivateNoteTarget
	}
	return db.Connect().Transaction(func(tx *gorm.DB) error {
		// Deterministic user locks serialize quota checks and writes against account
		// closure. A closed target can never receive a new note after erasure.
		var people []EntityComplete
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id IN ?", []uint64{ownerID, targetID}).Order("id").Find(&people).Error; err != nil {
			return err
		}
		if note == "" {
			return tx.Where("owner_id = ? AND target_user_id = ?", ownerID, targetID).Delete(&PrivateNoteEntity{}).Error
		}
		if len(people) != 2 {
			return ErrPrivateNoteTarget
		}
		var existing int64
		if err := tx.Model(&PrivateNoteEntity{}).Where("owner_id = ? AND target_user_id = ?", ownerID, targetID).Count(&existing).Error; err != nil {
			return err
		}
		if existing == 0 {
			var count int64
			if err := tx.Model(&PrivateNoteEntity{}).Where("owner_id = ?", ownerID).Count(&count).Error; err != nil {
				return err
			}
			if count >= MaxPrivateNotes {
				return ErrPrivateNoteLimit
			}
		}
		return tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "owner_id"}, {Name: "target_user_id"}}, DoUpdates: clause.AssignmentColumns([]string{"note", "updated_at"})}).Create(&PrivateNoteEntity{OwnerID: ownerID, TargetUserID: targetID, Note: note}).Error
	})
}
