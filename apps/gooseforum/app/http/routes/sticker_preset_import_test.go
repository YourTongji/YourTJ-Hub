package routes

import (
	"context"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/stickerpresets"
	presetimages "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/console/stickerpresets"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/filemodel/filedata"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/sticker"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/stickerservice"
)

func TestStickerPresetImportRetainsManifestPack(t *testing.T) {
	_, _ = setupAdminStickersContractTest(t)
	presets, err := stickerpresets.Manifest()
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, preset := range presets {
		if seen[preset.Pack] {
			continue
		}
		seen[preset.Pack] = true
		t.Run(preset.Pack, func(t *testing.T) {
			data, err := presetimages.Load(preset.File)
			if err != nil {
				t.Fatal(err)
			}
			skipped, err := stickerservice.ImportImage(context.Background(), 0, data, preset.File, preset.Name, preset.Pack, 0, true)
			if err != nil || skipped {
				t.Fatalf("import: skipped=%v, err=%v", skipped, err)
			}
			row, err := sticker.GetByName(stickerservice.SanitizeName(preset.Name))
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = filedata.DeleteByName(row.FileName) })
			if row.Pack != preset.Pack || !row.IsOfficial {
				t.Fatalf("preset %q: pack=%q official=%v, want pack=%q official=true", row.Name, row.Pack, row.IsOfficial, preset.Pack)
			}
		})
	}
}
