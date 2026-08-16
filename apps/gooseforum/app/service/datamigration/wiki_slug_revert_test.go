package datamigration

import (
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiPages"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func openSlugRevertDB(t *testing.T) *gorm.DB {
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

func insertSlugRevertPage(t *testing.T, conn *gorm.DB, id uint64, path, sourcePath, namespace string) {
	t.Helper()
	p := wikiPages.Entity{Id: id, TopicId: 200000 + id, Path: path, SourcePath: sourcePath, Namespace: namespace}
	if err := conn.Create(&p).Error; err != nil {
		t.Fatalf("insert page %q: %v", path, err)
	}
}

func scanSlugRevertPage(t *testing.T, conn *gorm.DB, id uint64) (path, namespace string) {
	t.Helper()
	var p wikiPages.Entity
	if err := conn.Unscoped().Model(&wikiPages.Entity{}).Select("path", "namespace").Where("id = ?", id).Scan(&p).Error; err != nil {
		t.Fatalf("scan page id %d: %v", id, err)
	}
	return p.Path, p.Namespace
}

// TestRevertWikiNamespaceSlugsMovesSlugPathBackToDirName slug 已生效的存量
// 页面（path 首段 = slug，namespace 列 = slug）：按 source_path（仓库真实
// 相对路径）迁回目录名，namespace 列同步。
func TestRevertWikiNamespaceSlugsMovesSlugPathBackToDirName(t *testing.T) {
	conn := openSlugRevertDB(t)
	// 模拟：中文目录"同济新手教程"曾声明 slug=freshman-guide，
	// 页面 path 首段/namespace 列为 slug，source_path 为仓库真实路径。
	insertSlugRevertPage(t, conn, 1, "freshman-guide/index", "同济新手教程/index", "freshman-guide")
	insertSlugRevertPage(t, conn, 2, "freshman-guide/academics/课程", "同济新手教程/academics/课程", "freshman-guide")

	result := RevertWikiNamespaceSlugsWithDB(conn)
	if result.Failed != 0 {
		t.Fatalf("RevertWikiNamespaceSlugsWithDB() failed = %d last=%s", result.Failed, result.LastFailed)
	}
	if result.Migrated != 2 {
		t.Fatalf("migrated = %d, want 2", result.Migrated)
	}
	if path, ns := scanSlugRevertPage(t, conn, 1); path != "同济新手教程/index" || ns != "同济新手教程" {
		t.Fatalf("page 1 after revert: path=%q namespace=%q, want 同济新手教程/index/同济新手教程", path, ns)
	}
	if path, ns := scanSlugRevertPage(t, conn, 2); path != "同济新手教程/academics/课程" || ns != "同济新手教程" {
		t.Fatalf("page 2 after revert: path=%q namespace=%q", path, ns)
	}
}

// TestRevertWikiNamespaceSlugsSkipsAlreadyDirNamePath path 已是仓库路径
// （未分配过 slug / 已迁回）的行零操作；source_path 为空（从未同步）的行跳过。
func TestRevertWikiNamespaceSlugsSkipsAlreadyDirNamePath(t *testing.T) {
	conn := openSlugRevertDB(t)
	insertSlugRevertPage(t, conn, 1, "guide/start", "guide/start", "guide")
	insertSlugRevertPage(t, conn, 2, "old-path/x", "", "old-ns") // source_path 为空，跳过

	result := RevertWikiNamespaceSlugsWithDB(conn)
	if result.Failed != 0 {
		t.Fatalf("RevertWikiNamespaceSlugsWithDB() failed = %d last=%s", result.Failed, result.LastFailed)
	}
	if result.Migrated != 0 {
		t.Fatalf("migrated = %d, want 0 (no slug-affected rows)", result.Migrated)
	}
	if path, ns := scanSlugRevertPage(t, conn, 1); path != "guide/start" || ns != "guide" {
		t.Fatalf("page 1 changed unexpectedly: path=%q namespace=%q", path, ns)
	}
}

// TestRevertWikiNamespaceSlugsIdempotent 重复运行零变更。
func TestRevertWikiNamespaceSlugsIdempotent(t *testing.T) {
	conn := openSlugRevertDB(t)
	insertSlugRevertPage(t, conn, 1, "freshman-guide/start", "同济新手教程/start", "freshman-guide")

	result := RevertWikiNamespaceSlugsWithDB(conn)
	if result.Failed != 0 || result.Migrated != 1 {
		t.Fatalf("first run failed=%d migrated=%d, want 1", result.Failed, result.Migrated)
	}
	result = RevertWikiNamespaceSlugsWithDB(conn)
	if result.Failed != 0 {
		t.Fatalf("second run failed = %d last=%s", result.Failed, result.LastFailed)
	}
	if result.Migrated != 0 {
		t.Fatalf("second run migrated = %d, want 0", result.Migrated)
	}
}
