package cmd

import (
	"fmt"
	"log/slog"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/stickerpresets"
	presetimages "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/console/stickerpresets"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/stickerservice"
	"github.com/spf13/cobra"
)

func init() {
	cmd := &cobra.Command{
		Use:   "seed-stickers",
		Short: "Seed the built-in preset sticker pack (idempotent by sticker name)",
		RunE:  runSeedStickers,
	}
	appendCommand(cmd)
}

// runSeedStickers imports the embedded preset pack through the standard
// storage pipeline (filedata row + provider mirror + sticker usage), so
// preset stickers behave exactly like admin-uploaded ones: deletable,
// editable and garbage-collectable after their sticker row is removed.
// Idempotent: existing sticker names are skipped, so re-runs only fill gaps.
func runSeedStickers(cmd *cobra.Command, args []string) error {
	presets, err := stickerpresets.Manifest()
	if err != nil {
		return fmt.Errorf("read preset manifest: %w", err)
	}
	imported, skipped, failed := 0, 0, 0
	for index, preset := range presets {
		data, err := presetimages.Load(preset.File)
		if err == nil {
			var skip bool
			skip, err = stickerservice.ImportImage(cmd.Context(), 0, data, preset.File, preset.Name, preset.Pack, index, true)
			if err == nil {
				if skip {
					skipped++
				} else {
					imported++
				}
				continue
			}
		}
		slog.Warn("import preset sticker failed", "name", preset.Name, "error", err)
		failed++
	}
	cmd.Printf("seed-stickers done: imported=%d skipped(existing)=%d failed=%d total=%d\n", imported, skipped, failed, len(presets))
	if failed > 0 {
		return fmt.Errorf("%d preset stickers failed to import", failed)
	}
	return nil
}
