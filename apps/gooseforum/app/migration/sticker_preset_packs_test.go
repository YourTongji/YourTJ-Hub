package migration

import (
	"errors"
	"os"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/sticker"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/stickerservice"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestVersionedMigrationBackfillsPresetPacks(t *testing.T) {
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(&pageConfig.Entity{}, &sticker.Entity{}); err != nil {
		t.Fatal(err)
	}
	rows := seedLegacyPresetPacks(t, conn)
	for attempt := 0; attempt < 2; attempt++ {
		if err := pageConfig.SyncMigrationVersion(29); err != nil {
			t.Fatal(err)
		}
		if err := runVersionedDataMigrations(); err != nil {
			t.Fatal(err)
		}
		assertPresetPacks(t, conn, rows)
		if got := pageConfig.GetMigrationVersion(); got != 30 {
			t.Fatalf("migration version=%d, want 30", got)
		}
	}
}

func seedLegacyPresetPacks(t *testing.T, conn *gorm.DB) []sticker.Entity {
	t.Helper()
	rows := []sticker.Entity{
		{Name: "叶", FileName: "flower.png", IsOfficial: true, Pack: "official"},
		{Name: "232个国家还是你最成功", FileName: "emoji.jpg", IsOfficial: true, Pack: "official"},
		{Name: "嘿嘿", FileName: "custom.png", IsOfficial: true, Pack: "custom"},
		{Name: "日", FileName: "personal.png", IsOfficial: true, Pack: "official"},
		{Name: "nonpreset", FileName: "other.png", IsOfficial: true, Pack: "official"},
		{Name: "星", FileName: "admin.png", IsOfficial: true, Pack: "official", CreatedBy: 7},
		{Name: "cppmind", FileName: "wxmeme.png", IsOfficial: true, Pack: "official", SortOrder: 13},
	}
	if err := conn.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}
	ids := make([]uint64, len(rows))
	for index, row := range rows {
		ids[index] = row.Id
	}
	t.Cleanup(func() { conn.Where("id IN ?", ids).Delete(&sticker.Entity{}) })
	// Explicit false avoids GORM's default:true overriding the personal row.
	if err := conn.Model(&sticker.Entity{}).Where("id = ?", rows[3].Id).Update("is_official", false).Error; err != nil {
		t.Fatal(err)
	}
	rows[3].IsOfficial = false
	if err := conn.Model(&sticker.Entity{}).Where("id = ?", rows[6].Id).Update("is_enabled", false).Error; err != nil {
		t.Fatal(err)
	}
	rows[6].IsEnabled = false
	return rows
}

func assertPresetPacks(t *testing.T, conn *gorm.DB, rows []sticker.Entity) {
	t.Helper()
	for index, wantPack := range []string{"flowerhd", "emojipackage", "custom", "official", "official", "official", "wxmeme"} {
		var got sticker.Entity
		if err := conn.First(&got, rows[index].Id).Error; err != nil {
			t.Fatal(err)
		}
		if got.Pack != wantPack || got.Name != rows[index].Name || got.FileName != rows[index].FileName {
			t.Fatalf("row %q: pack=%q, want %q; name/file must stay unchanged", got.Name, got.Pack, wantPack)
		}
		if got.IsOfficial != rows[index].IsOfficial || got.IsEnabled != rows[index].IsEnabled || got.CreatedBy != rows[index].CreatedBy || got.SortOrder != rows[index].SortOrder {
			t.Fatalf("backfill altered non-pack fields for %q", got.Name)
		}
	}
}

func TestVersionedPresetPackMigrationFailureDoesNotAdvance(t *testing.T) {
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(&pageConfig.Entity{}, &sticker.Entity{}); err != nil {
		t.Fatal(err)
	}
	rows := seedLegacyPresetPacks(t, conn)
	if err := pageConfig.SyncMigrationVersion(29); err != nil {
		t.Fatal(err)
	}
	writes := 0
	if err := conn.Callback().Update().Before("gorm:update").Register("fail_preset_pack", func(tx *gorm.DB) {
		if tx.Statement.Table == "stickers" {
			writes++
			if writes == 2 {
				_ = tx.AddError(errors.New("forced pack update failure"))
			}
		}
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Callback().Update().Remove("fail_preset_pack") })
	if err := runVersionedDataMigrations(); err == nil {
		t.Fatal("expected migration failure")
	}
	if got := pageConfig.GetMigrationVersion(); got != 29 {
		t.Fatalf("failed migration advanced version to %d", got)
	}
	for _, row := range rows {
		var got sticker.Entity
		if err := conn.First(&got, row.Id).Error; err != nil {
			t.Fatal(err)
		}
		if got.Pack != row.Pack {
			t.Fatalf("failed migration retained partial update: %q = %q", got.Name, got.Pack)
		}
	}
}

func TestSchemaPresetPacksOnPostgreSQL(t *testing.T) {
	dsn := os.Getenv("YOURTJ_TEST_PG_URL")
	if dsn == "" {
		t.Skip("YOURTJ_TEST_PG_URL not set")
	}
	conn, err := gorm.Open(postgres.Open(dsn), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := conn.AutoMigrate(&sticker.Entity{}); err != nil {
		t.Fatal(err)
	}
	rows := seedLegacyPresetPacks(t, conn)
	for attempt, wantChanged := range []int64{3, 0} {
		changed, err := stickerservice.BackfillPresetPacks(conn)
		if err != nil || changed != wantChanged {
			t.Fatalf("attempt %d: changed=%d err=%v, want %d", attempt, changed, err, wantChanged)
		}
		assertPresetPacks(t, conn, rows)
	}
}
