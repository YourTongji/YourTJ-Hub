package migration

import (
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/sticker"
	"github.com/glebarez/sqlite"
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
