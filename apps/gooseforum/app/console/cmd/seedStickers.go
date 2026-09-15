package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"path"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/console/stickerpresets"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/filemodel/filedata"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/sticker"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/fileusageservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/stickerservice"
	"github.com/spf13/cobra"
)

func init() {
	cmd := &cobra.Command{
		Use:   "seed-stickers",
		Short: "Seed the built-in preset sticker pack (idempotent by sticker name)",
		Run:   runSeedStickers,
	}
	appendCommand(cmd)
}

// runSeedStickers imports the embedded preset pack through the standard
// storage pipeline (filedata row + provider mirror + sticker usage), so
// preset stickers behave exactly like admin-uploaded ones: deletable,
// editable and garbage-collectable after their sticker row is removed.
// Idempotent: existing sticker names are skipped, so re-runs only fill gaps.
func runSeedStickers(cmd *cobra.Command, args []string) {
	presets, err := stickerpresets.Manifest()
	if err != nil {
		fmt.Println("read preset manifest failed:", err)
		os.Exit(1)
	}
	imported, skipped, failed := 0, 0, 0
	for index, preset := range presets {
		if existing := sticker.GetByName(preset.Name); existing.Id != 0 {
			skipped++
			continue
		}
		data, err := stickerpresets.Load(preset.File)
		if err != nil {
			slog.Warn("load preset sticker failed", "name", preset.Name, "file", preset.File, "err", err)
			failed++
			continue
		}
		entity, err := filedata.SaveFileFromUpload(0, data, path.Base(preset.File), "stickers")
		if err != nil {
			slog.Warn("save preset sticker file failed", "name", preset.Name, "err", err)
			failed++
			continue
		}
		name, ok := stickerservice.UniqueName(preset.Name)
		if !ok {
			slog.Warn("preset sticker name unusable", "name", preset.Name, "file", preset.File)
			failed++
			continue
		}
		row := sticker.Entity{
			Name:      name,
			FileName:  entity.Name,
			SortOrder: index,
			IsEnabled: true,
		}
		if err := sticker.Save(&row); err != nil {
			slog.Warn("save preset sticker row failed", "name", preset.Name, "err", err)
			failed++
			continue
		}
		if err := fileusageservice.AddStickerUsage(0, entity.Name, row.Id); err != nil {
			slog.Warn("register preset sticker usage failed", "name", preset.Name, "err", err)
		}
		imported++
	}
	fmt.Printf("seed-stickers done: imported=%d skipped(existing)=%d failed=%d total=%d\n", imported, skipped, failed, len(presets))
}
