package migration

import (
	"os"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/sticker"
	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// TestStickerSchemaOnSQLite 验证 stickers 新表在 SQLite 内存库上的 AutoMigrate
// 建表（列齐全 + name 唯一索引生效）。PostgreSQL 建表由 TestSchema 门禁覆盖。
func TestStickerSchemaOnSQLite(t *testing.T) {
	conn, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := conn.AutoMigrate(&sticker.Entity{}); err != nil {
		t.Fatalf("migrate stickers: %v", err)
	}
	if !conn.Migrator().HasTable("stickers") {
		t.Fatal("stickers table missing after AutoMigrate")
	}
	for _, column := range []string{"id", "name", "file_name", "sort_order", "is_enabled", "created_by"} {
		if !conn.Migrator().HasColumn(&sticker.Entity{}, column) {
			t.Fatalf("stickers column %s missing", column)
		}
	}
	if err := conn.Create(&sticker.Entity{Name: "滑稽", FileName: "stickers/a.png", IsEnabled: true}).Error; err != nil {
		t.Fatalf("insert sticker: %v", err)
	}
	if err := conn.Create(&sticker.Entity{Name: "滑稽", FileName: "stickers/b.png", IsEnabled: true}).Error; err == nil {
		t.Fatal("duplicate sticker name accepted, want unique index rejection")
	}
}

// Defaults classify every pre-existing global sticker as official. User
// membership is additive and does not rewrite an existing token or file key.
type legacySticker struct {
	Id        uint64 `gorm:"primaryKey;autoIncrement"`
	Name      string `gorm:"type:varchar(64);not null;uniqueIndex"`
	FileName  string `gorm:"type:varchar(512);not null;default:''"`
	SortOrder int    `gorm:"not null;default:0;index"`
	IsEnabled bool   `gorm:"not null;default:true;index"`
	CreatedBy uint64 `gorm:"not null;default:0"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (legacySticker) TableName() string { return "stickers" }

func assertStickerUpgrade(t *testing.T, conn *gorm.DB) {
	t.Helper()
	if err := conn.Migrator().DropTable(&sticker.LibraryEntry{}, &sticker.LibraryOwner{}, &sticker.Entity{}); err != nil {
		t.Fatal(err)
	}
	if err := conn.AutoMigrate(&legacySticker{}); err != nil {
		t.Fatal(err)
	}
	old := legacySticker{Name: "legacy_smile", FileName: "legacy.png", IsEnabled: true}
	if err := conn.Create(&old).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.AutoMigrate(&sticker.Entity{}, &sticker.LibraryOwner{}, &sticker.LibraryEntry{}); err != nil {
		t.Fatal(err)
	}
	var upgraded sticker.Entity
	if err := conn.First(&upgraded, old.Id).Error; err != nil {
		t.Fatal(err)
	}
	if !upgraded.IsOfficial || upgraded.Pack != "official" || upgraded.Name != old.Name || upgraded.FileName != old.FileName {
		t.Fatalf("upgrade corrupted legacy sticker: %#v", upgraded)
	}
	entry := sticker.LibraryEntry{UserID: 1, StickerID: upgraded.Id}
	if err := conn.Create(&entry).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Create(&entry).Error; err == nil {
		t.Fatal("duplicate member accepted")
	}
	if err := conn.Create(&sticker.LibraryEntry{UserID: 2, StickerID: upgraded.Id}).Error; err != nil {
		t.Fatalf("distinct account cannot collect: %v", err)
	}
}

func TestSchemaStickerUpgradeOnSQLite(t *testing.T) {
	conn, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatal(err)
	}
	assertStickerUpgrade(t, conn)
}

func TestSchemaStickerUpgradeOnPostgreSQL(t *testing.T) {
	dsn := os.Getenv("YOURTJ_TEST_PG_URL")
	if dsn == "" {
		t.Skip("YOURTJ_TEST_PG_URL not set")
	}
	conn, err := gorm.Open(postgres.Open(dsn), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatal(err)
	}
	assertStickerUpgrade(t, conn)
}
