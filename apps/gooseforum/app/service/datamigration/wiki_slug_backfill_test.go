package datamigration

import (
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiNamespaces"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func openSlugBackfillDB(t *testing.T) *gorm.DB {
	t.Helper()
	conn, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := conn.AutoMigrate(&wikiNamespaces.Entity{}); err != nil {
		t.Fatalf("auto migrate wiki_namespaces: %v", err)
	}
	return conn
}

func insertSlugBackfillNamespace(t *testing.T, conn *gorm.DB, name string, slug *string) uint64 {
	t.Helper()
	ns := wikiNamespaces.Entity{Name: name, Slug: slug}
	if err := conn.Create(&ns).Error; err != nil {
		t.Fatalf("insert namespace %q: %v", name, err)
	}
	return ns.Id
}

func scanSlugBackfillSlug(t *testing.T, conn *gorm.DB, id uint64) *string {
	t.Helper()
	var got *string
	if err := conn.Model(&wikiNamespaces.Entity{}).Select("slug").Where("id = ?", id).Scan(&got).Error; err != nil {
		t.Fatalf("scan slug for id %d: %v", id, err)
	}
	return got
}

func TestBackfillWikiNamespaceSlugsBackfillsASCIILowercaseName(t *testing.T) {
	conn := openSlugBackfillDB(t)
	id := insertSlugBackfillNamespace(t, conn, "guide", nil)
	insertSlugBackfillNamespace(t, conn, "deployment-guide", nil)

	result := BackfillWikiNamespaceSlugsWithDB(conn)
	if result.Failed != 0 {
		t.Fatalf("BackfillWikiNamespaceSlugsWithDB() failed = %d last=%s", result.Failed, result.LastFailed)
	}
	if result.Backfilled != 2 {
		t.Fatalf("backfilled = %d, want 2", result.Backfilled)
	}
	if got := scanSlugBackfillSlug(t, conn, id); got == nil || *got != "guide" {
		t.Fatalf("slug = %v, want guide", got)
	}
}

func TestBackfillWikiNamespaceSlugsSkipsNonASCIIChineseName(t *testing.T) {
	conn := openSlugBackfillDB(t)
	id := insertSlugBackfillNamespace(t, conn, "同济新手教程", nil)

	result := BackfillWikiNamespaceSlugsWithDB(conn)
	if result.Failed != 0 {
		t.Fatalf("BackfillWikiNamespaceSlugsWithDB() failed = %d last=%s", result.Failed, result.LastFailed)
	}
	if result.Backfilled != 0 {
		t.Fatalf("backfilled = %d, want 0 for chinese name", result.Backfilled)
	}
	if got := scanSlugBackfillSlug(t, conn, id); got != nil {
		t.Fatalf("slug = %v, want NULL for chinese name", got)
	}
}

func TestBackfillWikiNamespaceSlugsSkipsNonSlugFormASCII(t *testing.T) {
	conn := openSlugBackfillDB(t)
	for _, name := range []string{"Guide", "my_namespace", "带有中文123", "guide!"} {
		insertSlugBackfillNamespace(t, conn, name, nil)
	}

	result := BackfillWikiNamespaceSlugsWithDB(conn)
	if result.Failed != 0 {
		t.Fatalf("BackfillWikiNamespaceSlugsWithDB() failed = %d last=%s", result.Failed, result.LastFailed)
	}
	if result.Backfilled != 0 {
		t.Fatalf("backfilled = %d, want 0 for non-slug-form names", result.Backfilled)
	}
}

func TestBackfillWikiNamespaceSlugsSkipsAlreadyAssigned(t *testing.T) {
	conn := openSlugBackfillDB(t)
	assigned := "guide"
	id := insertSlugBackfillNamespace(t, conn, "guide", &assigned)

	result := BackfillWikiNamespaceSlugsWithDB(conn)
	if result.Failed != 0 {
		t.Fatalf("BackfillWikiNamespaceSlugsWithDB() failed = %d last=%s", result.Failed, result.LastFailed)
	}
	if result.Backfilled != 0 {
		t.Fatalf("backfilled = %d, want 0 (idempotent)", result.Backfilled)
	}
	if got := scanSlugBackfillSlug(t, conn, id); got == nil || *got != "guide" {
		t.Fatalf("slug = %v, want guide unchanged", got)
	}
}

func TestBackfillWikiNamespaceSlugsSkipsSlugConflictWithOtherRow(t *testing.T) {
	conn := openSlugBackfillDB(t)
	// 行 A 已占用 slug "guide"（如仓库同步分配）；行 B name 恰好也是 "guide" 形态。
	insertSlugBackfillNamespace(t, conn, "同济指南", &[]string{"guide"}[0])
	idB := insertSlugBackfillNamespace(t, conn, "guide", nil)

	result := BackfillWikiNamespaceSlugsWithDB(conn)
	if result.Failed != 0 {
		t.Fatalf("BackfillWikiNamespaceSlugsWithDB() failed = %d last=%s", result.Failed, result.LastFailed)
	}
	if result.Backfilled != 0 {
		t.Fatalf("backfilled = %d, want 0 on conflict", result.Backfilled)
	}
	if got := scanSlugBackfillSlug(t, conn, idB); got != nil {
		t.Fatalf("slug = %v, want NULL on conflict (keep old value)", got)
	}
}

func TestBackfillWikiNamespaceSlugsIdempotentSecondRun(t *testing.T) {
	conn := openSlugBackfillDB(t)
	insertSlugBackfillNamespace(t, conn, "guide", nil)

	result := BackfillWikiNamespaceSlugsWithDB(conn)
	if result.Failed != 0 || result.Backfilled != 1 {
		t.Fatalf("first run failed=%d backfilled=%d, want 1", result.Failed, result.Backfilled)
	}
	result = BackfillWikiNamespaceSlugsWithDB(conn)
	if result.Failed != 0 {
		t.Fatalf("second run failed = %d last=%s", result.Failed, result.LastFailed)
	}
	if result.Backfilled != 0 {
		t.Fatalf("second run backfilled = %d, want 0", result.Backfilled)
	}
}
