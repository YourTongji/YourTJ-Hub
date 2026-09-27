package sticker

import (
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// LibraryOwner serializes every account's membership write, including the empty
// library, so concurrent saves cannot bypass its quota or lose an ordering.
type LibraryOwner struct {
	UserID   uint64 `gorm:"primaryKey;autoIncrement:false;column:user_id"`
	Revision uint64 `gorm:"not null;default:0"`
	Closed   bool   `gorm:"not null;default:false"`
}

func (LibraryOwner) TableName() string { return "sticker_library_owners" }

// LibraryEntry is private account state. Deleting it never deletes the shared
// immutable asset or its file usage; previously sent tokens remain resolvable.
type LibraryEntry struct {
	UserID      uint64    `gorm:"primaryKey;autoIncrement:false;column:user_id"`
	StickerID   uint64    `gorm:"primaryKey;autoIncrement:false;column:sticker_id;index"`
	DisplayName string    `gorm:"type:varchar(64);not null;default:''"`
	SortOrder   int       `gorm:"not null;default:0"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
}

func (LibraryEntry) TableName() string { return "user_stickers" }

var ErrLibraryClosed = errors.New("sticker library closed")

func LockLibraryTx(tx *gorm.DB, userID uint64) error {
	owner := LibraryOwner{UserID: userID}
	if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&owner).Error; err != nil {
		return err
	}
	// A write also serializes SQLite; SELECT FOR UPDATE alone is ignored there.
	if err := tx.Model(&LibraryOwner{}).Where("user_id = ?", userID).Update("revision", gorm.Expr("revision + 1")).Error; err != nil {
		return err
	}
	if err := tx.First(&owner, "user_id = ?", userID).Error; err != nil {
		return err
	}
	if owner.Closed {
		return ErrLibraryClosed
	}
	return nil
}

func LibraryEntriesTx(tx *gorm.DB, userID uint64) ([]LibraryEntry, error) {
	entries := make([]LibraryEntry, 0)
	err := tx.Where("user_id = ?", userID).Order("sort_order ASC, sticker_id ASC").Find(&entries).Error
	return entries, err
}

func SaveLibraryEntryTx(tx *gorm.DB, entry *LibraryEntry) error { return tx.Save(entry).Error }

func RemoveLibraryEntryTx(tx *gorm.DB, userID, stickerID uint64) error {
	return tx.Where("user_id = ? AND sticker_id = ?", userID, stickerID).Delete(&LibraryEntry{}).Error
}

func EntitiesByIDsTx(tx *gorm.DB, ids []uint64) ([]Entity, error) {
	items := make([]Entity, 0)
	if len(ids) == 0 {
		return items, nil
	}
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id IN ?", ids).Order("id ASC").Find(&items).Error
	return items, err
}

func PersonalUploadTx(tx *gorm.DB, userID uint64, fileName string) (Entity, error) {
	var entity Entity
	err := tx.Where("created_by = ? AND file_name = ? AND is_official = ?", userID, fileName, false).First(&entity).Error
	return entity, err
}

func CountPersonalUploadsTx(tx *gorm.DB, userID uint64) (int64, error) {
	var count int64
	err := tx.Model(&Entity{}).Where("created_by = ? AND is_official = ?", userID, false).Count(&count).Error
	return count, err
}

func InsertPersonalTx(tx *gorm.DB, entity *Entity) error {
	// Clear the legacy true default inside the same transaction before publishing.
	if err := tx.Create(entity).Error; err != nil {
		return err
	}
	entity.IsOfficial = false
	return tx.Model(entity).Update("is_official", false).Error
}

// CloseLibraryTx removes private membership and fences requests that passed
// authentication before account closure. Shared assets keep their stable tokens.
func CloseLibraryTx(tx *gorm.DB, userID uint64) error {
	if err := LockLibraryTx(tx, userID); err != nil && !errors.Is(err, ErrLibraryClosed) {
		return err
	}
	if err := tx.Where("user_id = ?", userID).Delete(&LibraryEntry{}).Error; err != nil {
		return err
	}
	return tx.Model(&LibraryOwner{}).Where("user_id = ?", userID).Update("closed", true).Error
}
