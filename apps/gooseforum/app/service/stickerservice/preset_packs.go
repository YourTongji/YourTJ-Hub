package stickerservice

import (
	"sort"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/stickerpresets"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/sticker"
	"gorm.io/gorm"
)

// BackfillPresetPacks repairs the default grouping of previously seeded assets.
// The same manifest and name sanitization drive seeding and upgrades; custom
// groups, administrator uploads and personal assets are never reclassified.
func BackfillPresetPacks(conn *gorm.DB) (int64, error) {
	presets, err := stickerpresets.Manifest()
	if err != nil {
		return 0, err
	}
	byPack := make(map[string][]string)
	var packs []string
	for _, preset := range presets {
		if _, exists := byPack[preset.Pack]; !exists {
			packs = append(packs, preset.Pack)
		}
		byPack[preset.Pack] = append(byPack[preset.Pack], SanitizeName(preset.Name))
	}
	sort.Strings(packs)
	var changed int64
	err = conn.Transaction(func(tx *gorm.DB) error {
		for _, pack := range packs {
			count, err := sticker.BackfillPresetPackTx(tx, pack, byPack[pack])
			if err != nil {
				return err
			}
			changed += count
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return changed, nil
}
