package migration

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiNamespaceEditors"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiPageRevisions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiPages"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/datamigration"
	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// legacyWikiPageEntity 旧版 wiki_pages 模型（迁移 v21 前）：无 content_hash /
// last_commit_sha / last_commit_at / rendered_html / toc / contributors_json 列。
// 用它 AutoMigrate 建出与真实存量库一致的旧表（列类型/索引均由旧版 GORM 生成）。
type legacyWikiPageEntity struct {
	Id                  uint64         `gorm:"primaryKey;column:id;autoIncrement;not null;" json:"id"`
	TopicId             uint64         `gorm:"column:topic_id;not null;default:0;uniqueIndex:uniq_wiki_page_topic,priority:1;" json:"topicId"`
	Namespace           string         `gorm:"column:namespace;type:varchar(64);not null;default:'';index:idx_wiki_page_namespace,priority:1;" json:"namespace"`
	Path                string         `gorm:"column:path;type:varchar(255);not null;default:'';uniqueIndex:uniq_wiki_page_path,priority:1;" json:"path"`
	ParentId            uint64         `gorm:"column:parent_id;not null;default:0;" json:"parentId"`
	SortOrder           int            `gorm:"column:sort_order;type:int;not null;default:0;" json:"sortOrder"`
	PublishedRevisionNo int            `gorm:"column:published_revision_no;type:int;not null;default:0;" json:"publishedRevisionNo"`
	DeletedAt           gorm.DeletedAt `gorm:"column:deleted_at;" json:"-"`
	CreatedAt           time.Time      `gorm:"column:created_at;autoCreateTime;<-:create;" json:"createdAt"`
	UpdatedAt           time.Time      `gorm:"column:updated_at;autoUpdateTime;" json:"updatedAt"`
}

func (legacyWikiPageEntity) TableName() string { return "wiki_pages" }

// exerciseWikiProjectionUpgrade 在给定连接上执行完整升级路径并断言结果：
// 旧表（wiki_pages 无新列 + revisions + editors 有存量数据）→ upgradeWikiPagesProjectionColumns
// （SQLite 显式 ADD COLUMN 保留数据；PG no-op 交给 AutoMigrate）→ AutoMigrate 全表
// → MigrateWikiProjectionV21（回填投影/溯源 + editors 种子迁移并保留兼容表）。
func exerciseWikiProjectionUpgrade(t *testing.T, db *gorm.DB) {
	t.Helper()
	// 1) 旧 schema + 存量数据。
	if err := db.AutoMigrate(&legacyWikiPageEntity{}, &wikiPageRevisions.Entity{}, &wikiNamespaceEditors.Entity{}); err != nil {
		t.Fatalf("migrate legacy wiki schema: %v", err)
	}
	page1 := legacyWikiPageEntity{TopicId: 1, Namespace: "school", Path: "school/intro", SortOrder: 1, PublishedRevisionNo: 2}
	if err := db.Create(&page1).Error; err != nil {
		t.Fatalf("seed page1: %v", err)
	}
	// v19 未跑过的页面（published_revision_no=0）：v21 需补回填指针。
	page2 := legacyWikiPageEntity{TopicId: 2, Namespace: "school", Path: "school/history", SortOrder: 2}
	if err := db.Create(&page2).Error; err != nil {
		t.Fatalf("seed page2: %v", err)
	}
	revisions := []wikiPageRevisions.Entity{
		{PageId: page1.Id, RevisionNo: 1, Title: "简介", Content: "旧内容", RenderedHTML: "<p>旧内容</p>", Toc: "[]", Status: wikiPageRevisions.StatusApproved, EditorId: 100},
		{PageId: page1.Id, RevisionNo: 2, Title: "简介v2", Content: "---\ntitle: 简介v2\n---\n\n# 新内容", RenderedHTML: "<h1>新内容</h1>", Toc: `[{"level":1,"id":"new-content","text":"新内容"}]`, Status: wikiPageRevisions.StatusApproved, EditorId: 100},
		{PageId: page2.Id, RevisionNo: 1, Title: "历史", Content: "历史内容", RenderedHTML: "<p>历史内容</p>", Toc: "[]", Status: wikiPageRevisions.StatusApproved, EditorId: 200},
	}
	for _, rev := range revisions {
		if err := db.Create(&rev).Error; err != nil {
			t.Fatalf("seed revision: %v", err)
		}
	}
	if err := db.Create(&wikiNamespaceEditors.Entity{Namespace: "school", UserId: 100, AddedBy: 1}).Error; err != nil {
		t.Fatalf("seed namespace editor: %v", err)
	}

	// 2) 升级：显式加列（SQLite）+ AutoMigrate 全表。
	if err := upgradeWikiPagesProjectionColumns(db); err != nil {
		t.Fatalf("upgrade wiki pages columns: %v", err)
	}
	if err := db.AutoMigrate(SchemaModels()...); err != nil {
		t.Fatalf("automigrate full schema: %v", err)
	}
	// 前置：AutoMigrate 不删表，v21 数据迁移前旧表必须仍在（回填数据源）。
	if !db.Migrator().HasTable("wiki_page_revisions") || !db.Migrator().HasTable("wiki_namespace_editors") {
		t.Fatal("precondition failed: legacy tables must still exist before v21 data migration")
	}

	// 3) v21 数据迁移：回填 + 种子；旧表留给 #262 前的兼容 API。
	result := datamigration.MigrateWikiProjectionV21WithDB(db)
	if result.Failed > 0 {
		t.Fatalf("wiki projection v21 migration failed: %s", result.LastFailed)
	}

	// 4) 断言。
	assertWikiProjectionSchema(t, db)
	assertWikiProjectionBackfill(t, db, page1.Id, page2.Id)
}

// assertWikiProjectionSchema 断言新列/新表存在、兼容期旧表仍保留。
func assertWikiProjectionSchema(t *testing.T, db *gorm.DB) {
	t.Helper()
	for _, col := range []string{"content_hash", "last_commit_sha", "last_commit_at", "rendered_html", "toc", "contributors_json"} {
		if !db.Migrator().HasColumn(&wikiPages.Entity{}, col) {
			t.Errorf("wiki_pages.%s column missing after upgrade", col)
		}
	}
	if !db.Migrator().HasTable("wiki_sync_runs") {
		t.Error("wiki_sync_runs table missing after upgrade")
	}
	if !db.Migrator().HasTable("wiki_page_revisions") {
		t.Error("wiki_page_revisions must remain until #262 switches all consumers")
	}
	if !db.Migrator().HasTable("wiki_namespace_editors") {
		t.Error("wiki_namespace_editors must remain until #262 switches all consumers")
	}
}

// assertWikiProjectionBackfill 断言回填正确：rendered_html/toc 取最新 approved 修订、
// content_hash 为最新 markdown 的 sha256、<19 页面指针补回填、editors 种子入 contributors_json。
func assertWikiProjectionBackfill(t *testing.T, db *gorm.DB, page1ID, page2ID uint64) {
	t.Helper()
	var page1 wikiPages.Entity
	if err := db.First(&page1, "id = ?", page1ID).Error; err != nil {
		t.Fatalf("load page1: %v", err)
	}
	if page1.RenderedHTML != "<h1>新内容</h1>" {
		t.Errorf("page1 rendered_html = %q, want latest approved revision", page1.RenderedHTML)
	}
	if !strings.Contains(page1.Toc, "new-content") {
		t.Errorf("page1 toc = %q, want backfilled from latest approved revision", page1.Toc)
	}
	if want := sha256HexForTest("\n# 新内容"); page1.ContentHash != want {
		t.Errorf("page1 content_hash = %q, want %q", page1.ContentHash, want)
	}
	if page1.LastCommitAt == nil {
		t.Error("page1 last_commit_at nil, want backfilled from latest approved revision")
	}
	if page1.PublishedRevisionNo != 2 {
		t.Errorf("page1 published_revision_no = %d, want 2 (already set)", page1.PublishedRevisionNo)
	}
	var page2 wikiPages.Entity
	if err := db.First(&page2, "id = ?", page2ID).Error; err != nil {
		t.Fatalf("load page2: %v", err)
	}
	if page2.PublishedRevisionNo != 1 {
		t.Errorf("page2 published_revision_no = %d, want 1 (backfilled for <19 instance)", page2.PublishedRevisionNo)
	}
	var contributors string
	if err := db.Table("wiki_pages").Select("contributors_json").Where("id = ?", page1ID).Scan(&contributors).Error; err != nil {
		t.Fatalf("load contributors_json: %v", err)
	}
	if !strings.Contains(contributors, `"userId":100`) {
		t.Errorf("page1 contributors_json = %q, want seed migrated from wiki_namespace_editors", contributors)
	}
	// 同一 namespace 下的 page2 也应带种子贡献者。
	var contributors2 string
	if err := db.Table("wiki_pages").Select("contributors_json").Where("id = ?", page2ID).Scan(&contributors2).Error; err != nil {
		t.Fatalf("load page2 contributors_json: %v", err)
	}
	if !strings.Contains(contributors2, `"userId":100`) {
		t.Errorf("page2 contributors_json = %q, want same namespace seed", contributors2)
	}
}

func sha256HexForTest(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// TestWikiProjectionUpgradeFromLegacySchema 迁移 v21（issue #259）升级路径（SQLite）：
// 存量库无新列 + 两旧表有数据 → 显式加列保留数据 → v21 回填投影/溯源 → 种子迁移 editors
// → 保留两张兼容表。核心断言：存量数据不丢、回填取最新 approved 修订、旧读写链仍可用。
func TestWikiProjectionUpgradeFromLegacySchema(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	exerciseWikiProjectionUpgrade(t, db)
}

// TestWikiProjectionV21NoopOnFreshDB 全新库路径：revision/editors 表都不存在时，
// 迁移必须 no-op 成功（Failed=0），不报错、不推进失败（review LOW #10d）。
func TestWikiProjectionV21NoopOnFreshDB(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	result := datamigration.MigrateWikiProjectionV21WithDB(db)
	if result.Failed > 0 {
		t.Fatalf("fresh-db migration failed: %s", result.LastFailed)
	}
}

// TestWikiProjectionV21PreservesLegacyTables verifies the compatibility bridge:
// #262 has not switched every reader/writer yet, so v21 must not drop either table.
func TestWikiProjectionV21PreservesLegacyTables(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&wikiNamespaceEditors.Entity{}, &wikiPageRevisions.Entity{}); err != nil {
		t.Fatalf("migrate legacy tables: %v", err)
	}
	if err := db.Create(&wikiNamespaceEditors.Entity{Namespace: "school", UserId: 100, AddedBy: 1}).Error; err != nil {
		t.Fatalf("seed editor: %v", err)
	}
	result := datamigration.MigrateWikiProjectionV21WithDB(db)
	if result.Failed > 0 {
		t.Fatalf("migration failed: %s", result.LastFailed)
	}
	if !db.Migrator().HasTable("wiki_page_revisions") || !db.Migrator().HasTable("wiki_namespace_editors") {
		t.Error("v21 must preserve both legacy tables until #262")
	}
}

// TestSchemaWikiProjectionUpgradeOnPostgreSQL 同上，PostgreSQL 版（PG 走 AutoMigrate ALTER
// COLUMN 加列，不重建表不丢数据）。依赖 YOURTJ_TEST_PG_URL，未设置时跳过。
func TestSchemaWikiProjectionUpgradeOnPostgreSQL(t *testing.T) {
	dsn := os.Getenv("YOURTJ_TEST_PG_URL")
	if dsn == "" {
		t.Skip("YOURTJ_TEST_PG_URL not set; skipping PostgreSQL wiki projection upgrade test")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{TranslateError: true, Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	// 清理共享测试库（与 migration_pg_test 共用）。
	if err := db.Exec(`DROP SCHEMA public CASCADE; CREATE SCHEMA public;`).Error; err != nil {
		t.Fatalf("reset schema: %v", err)
	}
	exerciseWikiProjectionUpgrade(t, db)
}
