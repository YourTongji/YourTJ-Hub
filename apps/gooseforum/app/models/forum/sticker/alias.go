package sticker

import (
	"time"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Alias keeps a renamed sticker's previous token name resolvable (issue #1428).
// Sent content stores [:sticker:name:] text, so a rename records the old name
// here instead of rewriting posts, excerpts, revisions or messages. Names and
// aliases share one namespace: an alias points at exactly one sticker and no
// other sticker may take it while it is held.
type Alias struct {
	Name      string    `gorm:"primaryKey;column:name;type:varchar(64);not null;" json:"name"`
	StickerID uint64    `gorm:"column:sticker_id;not null;index;" json:"stickerId"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime;<-:create;" json:"createdAt"`
}

func (Alias) TableName() string { return "sticker_aliases" }

// AliasByNameTx returns the alias row holding name, or gorm.ErrRecordNotFound.
func AliasByNameTx(tx *gorm.DB, name string) (Alias, error) {
	var row Alias
	err := tx.Where("name = ?", name).First(&row).Error
	return row, err
}

// RecordAliasTx points an old name at stickerID. Re-recording the same pair is
// idempotent; the caller has already checked that no other sticker holds it.
func RecordAliasTx(tx *gorm.DB, name string, stickerID uint64) error {
	err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "name"}}, DoUpdates: clause.Assignments(map[string]any{"sticker_id": stickerID})}).
		Create(&Alias{Name: name, StickerID: stickerID}).Error
	if err == nil {
		notifyMutation()
	}
	return err
}

// ReleaseAliasTx frees name when its own sticker takes it back as the current name.
func ReleaseAliasTx(tx *gorm.DB, name string, stickerID uint64) error {
	return tx.Where("name = ? AND sticker_id = ?", name, stickerID).Delete(&Alias{}).Error
}

// AliasNamesBySticker lists previous names per sticker, oldest first.
func AliasNamesBySticker(ids []uint64) (map[uint64][]string, error) {
	names := make(map[uint64][]string, len(ids))
	if len(ids) == 0 {
		return names, nil
	}
	rows := make([]Alias, 0)
	if err := db.Connect().Model(&Alias{}).Where("sticker_id IN ?", ids).Order("created_at ASC").Order("name ASC").Find(&rows).Error; err != nil {
		return names, err
	}
	for _, row := range rows {
		names[row.StickerID] = append(names[row.StickerID], row.Name)
	}
	return names, nil
}

// EnabledByAliases resolves old names to enabled, approved stickers. The map
// key is the requested old name so callers can answer in the caller's terms.
func EnabledByAliases(names []string) (map[string]Entity, error) {
	resolved := make(map[string]Entity, len(names))
	if len(names) == 0 {
		return resolved, nil
	}
	aliases := make([]Alias, 0)
	if err := db.Connect().Model(&Alias{}).Where("name IN ?", names).Find(&aliases).Error; err != nil || len(aliases) == 0 {
		return resolved, err
	}
	ids := make([]uint64, 0, len(aliases))
	for _, alias := range aliases {
		ids = append(ids, alias.StickerID)
	}
	entities := make([]Entity, 0)
	if err := builder().Where("id IN ? AND is_enabled = ? AND review_status = ?", ids, true, ReviewStatusApproved).Find(&entities).Error; err != nil {
		return resolved, err
	}
	byID := make(map[uint64]Entity, len(entities))
	for _, entity := range entities {
		byID[entity.Id] = entity
	}
	for _, alias := range aliases {
		if entity, ok := byID[alias.StickerID]; ok {
			resolved[alias.Name] = entity
		}
	}
	return resolved, nil
}
