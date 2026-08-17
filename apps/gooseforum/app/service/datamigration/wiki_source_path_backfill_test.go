package datamigration

import (
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiPages"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func openSourcePathBackfillDB(t *testing.T) *gorm.DB {
	t.Helper()
	conn, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := conn.AutoMigrate(&wikiPages.Entity{}); err != nil {
		t.Fatalf("auto migrate wiki_pages: %v", err)
	}
	return conn
}

// insertSourcePathPage 插入页面：topic_id 与 path 均有唯一索引
// （uniq_wiki_page_topic / uniq_wiki_page_path），每次调用用独立 id/path。
func insertSourcePathPage(t *testing.T, conn *gorm.DB, id uint64, path, sourcePath string) {
	t.Helper()
	p := wikiPages.Entity{Id: id, TopicId: 100000 + id, Path: path, SourcePath: sourcePath}
	if err := conn.Create(&p).Error; err != nil {
		t.Fatalf("insert page %q: %v", path, err)
	}
}

func scanSourcePath(t *testing.T, conn *gorm.DB, id uint64) string {
	t.Helper()
	var got string
	if err := conn.Unscoped().Model(&wikiPages.Entity{}).Select("source_path").Where("id = ?", id).Scan(&got).Error; err != nil {
		t.Fatalf("scan source_path for id %d: %v", id, err)
	}
	return got
}

func TestBackfillWikiPageSourcePathsBackfillsEmptyRows(t *testing.T) {
	conn := openSourcePathBackfillDB(t)
	insertSourcePathPage(t, conn, 1, "guide/start", "")
	insertSourcePathPage(t, conn, 2, "guide/content", "guide/content")

	result := BackfillWikiPageSourcePathsWithDB(conn)
	if result.Failed != 0 {
		t.Fatalf("BackfillWikiPageSourcePathsWithDB() failed = %d last=%s", result.Failed, result.LastFailed)
	}
	if result.Backfilled != 1 {
		t.Fatalf("backfilled = %d, want 1", result.Backfilled)
	}
	if got := scanSourcePath(t, conn, 1); got != "guide/start" {
		t.Fatalf("source_path = %q, want guide/start", got)
	}
}

func TestBackfillWikiPageSourcePathsIncludesSoftDeletedRows(t *testing.T) {
	conn := openSourcePathBackfillDB(t)
	insertSourcePathPage(t, conn, 1, "guide/archived", "")
	if err := conn.Where("id = ?", 1).Delete(&wikiPages.Entity{}).Error; err != nil {
		t.Fatalf("soft delete page: %v", err)
	}

	result := BackfillWikiPageSourcePathsWithDB(conn)
	if result.Failed != 0 {
		t.Fatalf("BackfillWikiPageSourcePathsWithDB() failed = %d last=%s", result.Failed, result.LastFailed)
	}
	if result.Backfilled != 1 {
		t.Fatalf("backfilled = %d, want 1", result.Backfilled)
	}
	if got := scanSourcePath(t, conn, 1); got != "guide/archived" {
		t.Fatalf("source_path = %q, want guide/archived", got)
	}
}
func TestBackfillWikiPageSourcePathsIdempotentSecondRun(t *testing.T) {
	conn := openSourcePathBackfillDB(t)
	insertSourcePathPage(t, conn, 1, "guide/start", "")

	if result := BackfillWikiPageSourcePathsWithDB(conn); result.Backfilled != 1 || result.Failed != 0 {
		t.Fatalf("first run backfilled=%d failed=%d, want 1/0", result.Backfilled, result.Failed)
	}
	result := BackfillWikiPageSourcePathsWithDB(conn)
	if result.Backfilled != 0 || result.Failed != 0 {
		t.Fatalf("second run backfilled=%d failed=%d, want 0/0 (idempotent)", result.Backfilled, result.Failed)
	}
}
