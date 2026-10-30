package sticker

import (
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// PackOfficial is the default group of administrator stickers.
const PackOfficial = "official"

// OfficialPack names and orders one group of official stickers. Stickers join
// a group through Entity.Pack; a key without a row still groups its stickers
// and shows a default name, so seeded presets need no data migration.
type OfficialPack struct {
	Key            string    `gorm:"primaryKey;autoIncrement:false;column:pack_key;type:varchar(64);"`
	Name           string    `gorm:"column:name;type:varchar(64);not null;default:'';"`
	SortOrder      int       `gorm:"column:sort_order;not null;default:0;"`
	CoverStickerID uint64    `gorm:"column:cover_sticker_id;not null;default:0;"`
	CreatedAt      time.Time `gorm:"column:created_at;autoCreateTime;<-:create;"`
	UpdatedAt      time.Time `gorm:"column:updated_at;autoUpdateTime;"`
}

func (OfficialPack) TableName() string { return "sticker_packs" }

// OfficialPackCount is the number of official stickers holding one pack key.
type OfficialPackCount struct {
	Pack    string
	Total   int
	Enabled int
}

func OfficialPacks() ([]OfficialPack, error) {
	return OfficialPacksTx(marketDB())
}

func OfficialPacksTx(tx *gorm.DB) ([]OfficialPack, error) {
	rows := make([]OfficialPack, 0)
	err := tx.Order("sort_order ASC").Order("pack_key ASC").Find(&rows).Error
	return rows, err
}

func OfficialPackTx(tx *gorm.DB, key string) (OfficialPack, error) {
	var row OfficialPack
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("pack_key = ?", key).First(&row).Error
	return row, err
}

// SaveOfficialPackTx inserts or replaces the metadata of one pack key.
func SaveOfficialPackTx(tx *gorm.DB, row *OfficialPack) error {
	err := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "pack_key"}},
		DoUpdates: clause.AssignmentColumns([]string{"name", "sort_order", "cover_sticker_id", "updated_at"}),
	}).Create(row).Error
	if err == nil {
		notifyMutation()
	}
	return err
}

func DeleteOfficialPackTx(tx *gorm.DB, key string) error {
	err := tx.Where("pack_key = ?", key).Delete(&OfficialPack{}).Error
	if err == nil {
		notifyMutation()
	}
	return err
}

// OfficialPackCounts groups official stickers, enabled or not, by pack key.
func OfficialPackCounts(tx *gorm.DB) ([]OfficialPackCount, error) {
	rows := make([]OfficialPackCount, 0)
	err := tx.Model(&Entity{}).
		Select("pack, COUNT(*) AS total, SUM(CASE WHEN is_enabled THEN 1 ELSE 0 END) AS enabled").
		Where("is_official = ?", true).Group("pack").Scan(&rows).Error
	return rows, err
}

// MaxOfficialSortOrder is the largest sort order in a pack, or -1 when empty.
func MaxOfficialSortOrder(tx *gorm.DB, pack string) (int, error) {
	var order *int
	err := tx.Model(&Entity{}).Select("MAX(sort_order)").Where("is_official = ? AND pack = ?", true, pack).Scan(&order).Error
	if err != nil || order == nil {
		return -1, err
	}
	return *order, nil
}

// MoveOfficialStickersTx regroups every official sticker of one pack.
func MoveOfficialStickersTx(tx *gorm.DB, from, to string) (int64, error) {
	result := tx.Model(&Entity{}).Where("is_official = ? AND pack = ?", true, from).Update("pack", to)
	if result.Error == nil && result.RowsAffected > 0 {
		notifyMutation()
	}
	return result.RowsAffected, result.Error
}

// OfficialByIDsTx locks the official stickers among ids, in id order.
func OfficialByIDsTx(tx *gorm.DB, ids []uint64) ([]Entity, error) {
	rows := make([]Entity, 0, len(ids))
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id IN ? AND is_official = ?", ids, true).Order("id ASC").Find(&rows).Error
	return rows, err
}
