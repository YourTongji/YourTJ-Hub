package datamigration

import (
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiNamespaces"
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
	if err := conn.AutoMigrate(&wikiPages.Entity{}, &wikiNamespaces.Entity{}); err != nil {
		t.Fatalf("auto migrate wiki tables: %v", err)
	}
	return conn
}

func insertSlugRevertPage(t *testing.T, conn *gorm.DB, id uint64, path, sourcePath, namespace string) {
	t.Helper()
	p := wikiPages.Entity{Id: id, TopicId: 200000 + id, Path: path, SourcePath: sourcePath, Namespace: namespace, ContentHash: "hash-" + path}
	if err := conn.Create(&p).Error; err != nil {
		t.Fatalf("insert page %q: %v", path, err)
	}
}

func scanSlugRevertPage(t *testing.T, conn *gorm.DB, id uint64) (path, namespace, contentHash string) {
	t.Helper()
	var p wikiPages.Entity
	if err := conn.Unscoped().Model(&wikiPages.Entity{}).Select("path", "namespace", "content_hash").Where("id = ?", id).Scan(&p).Error; err != nil {
		t.Fatalf("scan page id %d: %v", id, err)
	}
	return p.Path, p.Namespace, p.ContentHash
}

// TestRevertWikiNamespaceSlugsMovesSlugPathBackToDirName slug 已生效的存量
// 页面（path 首段 = slug，namespace 列 = slug）：按 source_path（仓库真实
// 相对路径）迁回目录名，namespace 列同步；content_hash 清空强制下次同步重渲染。
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
	if path, ns, hash := scanSlugRevertPage(t, conn, 1); path != "同济新手教程/index" || ns != "同济新手教程" || hash != "" {
		t.Fatalf("page 1 after revert: path=%q namespace=%q hash=%q, want 同济新手教程/index/同济新手教程/empty", path, ns, hash)
	}
	if path, ns, hash := scanSlugRevertPage(t, conn, 2); path != "同济新手教程/academics/课程" || ns != "同济新手教程" || hash != "" {
		t.Fatalf("page 2 after revert: path=%q namespace=%q hash=%q", path, ns, hash)
	}
}

// TestRevertWikiNamespaceSlugsSwappedSlugs review P1：两个命名空间互为 slug
// （目录 a 的 slug=b、目录 b 的 slug=a）时，逐行更新会因目标 path 仍被对方
// 占用而违反 uniq_wiki_page_path。两阶段迁移（临时路径 → 最终路径）必须成功。
func TestRevertWikiNamespaceSlugsSwappedSlugs(t *testing.T) {
	conn := openSlugRevertDB(t)
	// 目录 a 曾声明 slug=b；目录 b 曾声明 slug=a。同相对页存在：
	// 页面 1 path=b/index、source_path=a/index（目标 a/index 被页面 2 占用）
	// 页面 2 path=a/index、source_path=b/index（目标 b/index 被页面 1 占用）
	insertSlugRevertPage(t, conn, 1, "b/index", "a/index", "b")
	insertSlugRevertPage(t, conn, 2, "a/index", "b/index", "a")

	result := RevertWikiNamespaceSlugsWithDB(conn)
	if result.Failed != 0 {
		t.Fatalf("RevertWikiNamespaceSlugsWithDB() failed = %d last=%s", result.Failed, result.LastFailed)
	}
	if result.Migrated != 2 {
		t.Fatalf("migrated = %d, want 2", result.Migrated)
	}
	if path, ns, _ := scanSlugRevertPage(t, conn, 1); path != "a/index" || ns != "a" {
		t.Fatalf("page 1 after revert: path=%q namespace=%q, want a/index/a", path, ns)
	}
	if path, ns, _ := scanSlugRevertPage(t, conn, 2); path != "b/index" || ns != "b" {
		t.Fatalf("page 2 after revert: path=%q namespace=%q, want b/index/b", path, ns)
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
	if path, ns, hash := scanSlugRevertPage(t, conn, 1); path != "guide/start" || ns != "guide" || hash != "hash-guide/start" {
		t.Fatalf("page 1 changed unexpectedly: path=%q namespace=%q hash=%q", path, ns, hash)
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

// TestRevertWikiNamespaceSlugsDropsLegacySlugSchema review P2：AutoMigrate
// 不删除从模型消失的字段，存量库升级后 wiki_namespaces.slug 列与
// uniq_wiki_namespace_slug 索引会残留；迁移必须显式删除。
func TestRevertWikiNamespaceSlugsDropsLegacySlugSchema(t *testing.T) {
	conn := openSlugRevertDB(t)
	// 模拟存量库：手动重建 slug 列与唯一索引（新模型已无该字段，AutoMigrate 不会建）。
	slug := "freshman-guide"
	if err := conn.Exec("ALTER TABLE wiki_namespaces ADD COLUMN slug varchar(64)").Error; err != nil {
		t.Fatalf("add legacy slug column: %v", err)
	}
	if err := conn.Exec("CREATE UNIQUE INDEX uniq_wiki_namespace_slug ON wiki_namespaces (slug)").Error; err != nil {
		t.Fatalf("create legacy slug index: %v", err)
	}
	if err := conn.Create(&wikiNamespaces.Entity{Name: "同济新手教程"}).Error; err != nil {
		t.Fatalf("insert namespace: %v", err)
	}
	if err := conn.Exec("UPDATE wiki_namespaces SET slug = ? WHERE name = ?", slug, "同济新手教程").Error; err != nil {
		t.Fatalf("set legacy slug value: %v", err)
	}

	result := RevertWikiNamespaceSlugsWithDB(conn)
	if result.Failed != 0 {
		t.Fatalf("RevertWikiNamespaceSlugsWithDB() failed = %d last=%s", result.Failed, result.LastFailed)
	}
	if !result.SlugIndexDropped || !result.SlugColumnDropped {
		t.Fatalf("legacy schema drop flags = index:%v column:%v, want true/true", result.SlugIndexDropped, result.SlugColumnDropped)
	}
	if conn.Migrator().HasColumn("wiki_namespaces", "slug") {
		t.Fatal("slug column still exists after migration")
	}
	if conn.Migrator().HasIndex("wiki_namespaces", "uniq_wiki_namespace_slug") {
		t.Fatal("slug unique index still exists after migration")
	}
}

// TestRevertWikiNamespaceSlugsEmptySourcePathUsesLegacySlugMap review Blocker：
// source_path 为空的 slug 行（v23 前已软删、未被 source_path 回填覆盖）不能
// 跳过——drop slug 列前用遗留 slug→name 映射推导真实仓库路径，保留 slug 后
// 的相对后缀；否则映射丢失后文件重新出现时无法收养原页面/topic。
func TestRevertWikiNamespaceSlugsEmptySourcePathUsesLegacySlugMap(t *testing.T) {
	conn := openSlugRevertDB(t)
	// 模拟存量库：遗留 slug 列 + 映射（slug=freshman-guide → name=同济新手教程）。
	if err := conn.Exec("ALTER TABLE wiki_namespaces ADD COLUMN slug varchar(64)").Error; err != nil {
		t.Fatalf("add legacy slug column: %v", err)
	}
	if err := conn.Create(&wikiNamespaces.Entity{Name: "同济新手教程"}).Error; err != nil {
		t.Fatalf("insert namespace: %v", err)
	}
	if err := conn.Exec("UPDATE wiki_namespaces SET slug = ? WHERE name = ?", "freshman-guide", "同济新手教程").Error; err != nil {
		t.Fatalf("set legacy slug value: %v", err)
	}
	// 页面：path 首段 = slug、namespace = slug，但 source_path 为空（v23 回填未覆盖的软删行）。
	insertSlugRevertPage(t, conn, 1, "freshman-guide/index", "", "freshman-guide")
	insertSlugRevertPage(t, conn, 2, "freshman-guide/academics/课程", "", "freshman-guide")

	result := RevertWikiNamespaceSlugsWithDB(conn)
	if result.Failed != 0 {
		t.Fatalf("RevertWikiNamespaceSlugsWithDB() failed = %d last=%s", result.Failed, result.LastFailed)
	}
	if result.Migrated != 2 {
		t.Fatalf("migrated = %d, want 2", result.Migrated)
	}
	if path, ns, hash := scanSlugRevertPage(t, conn, 1); path != "同济新手教程/index" || ns != "同济新手教程" || hash != "" {
		t.Fatalf("page 1 after revert: path=%q namespace=%q hash=%q, want 同济新手教程/index/同济新手教程/empty", path, ns, hash)
	}
	if path, ns, _ := scanSlugRevertPage(t, conn, 2); path != "同济新手教程/academics/课程" || ns != "同济新手教程" {
		t.Fatalf("page 2 after revert: path=%q namespace=%q, want 同济新手教程/academics/课程/同济新手教程", path, ns)
	}
}

// TestRevertWikiNamespaceSlugsTempPrefixAvoidsRealPath review Should fix：
// "__wiki_slug_revert__/N" 未被路径规则保留，仓库可合法存在同名页面路径；
// 阶段 1 临时路径必须避开现有 path，否则撞 uniq_wiki_page_path 卡死迁移。
func TestRevertWikiNamespaceSlugsTempPrefixAvoidsRealPath(t *testing.T) {
	conn := openSlugRevertDB(t)
	// 真实页面恰好占用默认临时前缀路径。
	insertSlugRevertPage(t, conn, 1, "__wiki_slug_revert__/2", "", "other")
	// 待迁移页面：source_path 推导（路径本身不同）。
	insertSlugRevertPage(t, conn, 2, "freshman-guide/index", "同济新手教程/index", "freshman-guide")

	result := RevertWikiNamespaceSlugsWithDB(conn)
	if result.Failed != 0 {
		t.Fatalf("RevertWikiNamespaceSlugsWithDB() failed = %d last=%s", result.Failed, result.LastFailed)
	}
	if result.Migrated != 1 {
		t.Fatalf("migrated = %d, want 1 (page 2 only; page 1 is real, untouched)", result.Migrated)
	}
	if path, ns, _ := scanSlugRevertPage(t, conn, 2); path != "同济新手教程/index" || ns != "同济新手教程" {
		t.Fatalf("page 2 after revert: path=%q namespace=%q, want 同济新手教程/index/同济新手教程", path, ns)
	}
	// 真实页面必须原样保留。
	if path, ns, hash := scanSlugRevertPage(t, conn, 1); path != "__wiki_slug_revert__/2" || ns != "other" || hash != "hash-__wiki_slug_revert__/2" {
		t.Fatalf("real page 1 changed: path=%q namespace=%q hash=%q", path, ns, hash)
	}
}
