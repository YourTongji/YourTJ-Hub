package migration

import (
	"fmt"
	"os"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/course"
	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// legacyCourseReviewDDL 旧版 course_review 表结构（B3 清理功能上线前）：
// author_user_id NOT NULL DEFAULT 0（非空），deleted_at 为 gorm.DeletedAt
// 语义的 datetime 可空列（软删标记），唯一索引 uniq_course_review_offering_author
// 建在 (offering_id, author_user_id)。
const legacyCourseReviewDDL = `
CREATE TABLE course_review (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	offering_id INTEGER NOT NULL DEFAULT 0,
	author_user_id INTEGER NOT NULL DEFAULT 0,
	rating INTEGER,
	content TEXT NOT NULL DEFAULT '',
	is_anonymous INTEGER NOT NULL DEFAULT 0,
	status INTEGER NOT NULL DEFAULT 0,
	legacy_helpful_count INTEGER NOT NULL DEFAULT 0,
	source TEXT NOT NULL DEFAULT '',
	created_at DATETIME,
	updated_at DATETIME,
	deleted_at DATETIME
);
CREATE UNIQUE INDEX uniq_course_review_offering_author ON course_review (offering_id, author_user_id);
CREATE INDEX idx_course_review_offering ON course_review (offering_id);
CREATE INDEX idx_course_review_author ON course_review (author_user_id);
CREATE INDEX idx_course_review_status ON course_review (status);
`

// TestCourseReviewUpgradeFromLegacySchema 验证 B3 清理功能（issue #175）对
// 存量 SQLite 库的 AutoMigrate 升级路径（security 复审 F1）：
//   - 旧 DDL（author_user_id NOT NULL + gorm.DeletedAt 语义列 + 唯一索引）
//     建表并插入数据（含 deleted 行）→ 跑 AutoMigrate
//   - 断言：数据行完整保留、唯一索引保留、author_user_id 可写 NULL
//     （新清理语义）、deleted_at 为普通列（写值不触发软删过滤）
//
// SQLite 下 GORM 对列类型/约束变化（NOT NULL→nullable）会整表重建，本测试
// 验证重建过程不丢数据、不丢索引。
func TestCourseReviewUpgradeFromLegacySchema(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:migration-course-review-%d?mode=memory&cache=shared", 0)), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	// 1) 旧 DDL 建表
	for _, stmt := range []string{
		legacyCourseReviewDDL,
	} {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("create legacy course_review: %v", err)
		}
	}

	// 2) 插入存量数据（含 deleted 行、visible 行、legacy 行）
	legacyRows := []struct {
		id, offeringID, authorID int
		status                   int
		content                  string
		deletedAt                any
	}{
		{id: 1, offeringID: 100, authorID: 1001, status: 2, content: "已删除的评价", deletedAt: "2026-01-01 00:00:00"},
		{id: 2, offeringID: 100, authorID: 1002, status: 0, content: "可见评价", deletedAt: nil},
		{id: 3, offeringID: 100, authorID: 0, status: 0, content: "legacy 导入", deletedAt: nil},
	}
	for _, row := range legacyRows {
		if err := db.Exec(
			"INSERT INTO course_review (id, offering_id, author_user_id, status, content, deleted_at) VALUES (?, ?, ?, ?, ?, ?)",
			row.id, row.offeringID, row.authorID, row.status, row.content, row.deletedAt,
		).Error; err != nil {
			t.Fatalf("insert legacy row %d: %v", row.id, err)
		}
	}

	// 3) 升级：migrateSchema 的 preflight（SQLite 旧形态手工全列重建）+
	// 全量 AutoMigrate（含 course.ReviewEntity 新形态）
	if err := upgradeCourseReviewLegacySchema(db); err != nil {
		t.Fatalf("upgradeCourseReviewLegacySchema failed: %v", err)
	}
	// 升级后 preflight 幂等（再次调用直接跳过）
	if err := upgradeCourseReviewLegacySchema(db); err != nil {
		t.Fatalf("upgrade preflight not idempotent: %v", err)
	}
	if err := db.AutoMigrate(&course.ReviewEntity{}); err != nil {
		t.Fatalf("upgrade AutoMigrate course_review failed: %v", err)
	}

	// 4) 断言：数据完整保留（含 deleted 行）
	var count int64
	if err := db.Table("course_review").Count(&count).Error; err != nil {
		t.Fatalf("count rows: %v", err)
	}
	if count != 3 {
		t.Fatalf("row count after upgrade = %d, want 3（数据完整保留）", count)
	}
	var deletedRow course.ReviewEntity
	if err := db.Table("course_review").Where("id = ?", 1).First(&deletedRow).Error; err != nil {
		t.Fatalf("deleted row lost after upgrade: %v", err)
	}
	if deletedRow.Content != "已删除的评价" || deletedRow.AuthorID() != 1001 || deletedRow.Status != course.ReviewStatusDeleted {
		t.Fatalf("deleted row altered: %+v", deletedRow)
	}

	// 5) 断言：唯一索引保留（升级后插入重复 (offering, author) 必须被拒）
	if err := db.Exec("INSERT INTO course_review (offering_id, author_user_id, content) VALUES (100, 1001, 'dup')").Error; err == nil {
		t.Fatal("unique index lost after upgrade: duplicate (offering, author) accepted")
	}

	// 6) 断言：author_user_id 可写 NULL（新清理语义：置 NULL 释放占位）
	if err := db.Table("course_review").Where("id = ?", 2).Update("author_user_id", nil).Error; err != nil {
		t.Fatalf("write NULL author_user_id after upgrade failed: %v", err)
	}
	var nullAuthor course.ReviewEntity
	if err := db.Table("course_review").Where("id = ?", 2).First(&nullAuthor).Error; err != nil {
		t.Fatalf("reload NULL-author row: %v", err)
	}
	if nullAuthor.AuthorID() != 0 {
		t.Fatalf("author_user_id NULL read-back = %d, want 0", nullAuthor.AuthorID())
	}
	// NULL 占位释放：同 offering 不同用户可新建行（NULL 不参与唯一约束）
	if err := db.Exec("INSERT INTO course_review (offering_id, author_user_id, content) VALUES (100, 1002, 'new')").Error; err != nil {
		t.Fatalf("re-create after NULL author failed (unique slot not released): %v", err)
	}

	// 7) 断言：deleted_at 是普通列（写值不触发 gorm 软删过滤——Table 查询
	// 与 Model 查询都应能读到该行）
	if err := db.Table("course_review").Where("id = ?", 1).Update("deleted_at", "2026-01-02 00:00:00").Error; err != nil {
		t.Fatalf("write deleted_at as plain column failed: %v", err)
	}
	var viaModel course.ReviewEntity
	if err := db.Model(&course.ReviewEntity{}).Where("id = ?", 1).First(&viaModel).Error; err != nil {
		t.Fatalf("model query filtered by deleted_at (soft-delete semantics leaked): %v", err)
	}
	if viaModel.Id != 1 {
		t.Fatalf("model query id = %d, want 1（deleted_at 不应触发软删过滤）", viaModel.Id)
	}
}

// TestCourseReviewUpgradeFromLegacySchemaOnPostgreSQL 验证 PG 存量库升级路径
// （security 复审 F1 的 PG 侧）：旧 DDL（author_user_id BIGINT NOT NULL）建表
// + 插数据 → AutoMigrate → 断言数据保留、唯一索引保留、DROP NOT NULL 后
// NULL 可写。
// 由 YOURTJ_TEST_PG_URL 门控；未设置时跳过。
func TestCourseReviewUpgradeFromLegacySchemaOnPostgreSQL(t *testing.T) {
	dsn := os.Getenv("YOURTJ_TEST_PG_URL")
	if dsn == "" {
		t.Skip("YOURTJ_TEST_PG_URL not set; skipping PostgreSQL migration test")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	if err := db.Exec(`DROP SCHEMA public CASCADE; CREATE SCHEMA public;`).Error; err != nil {
		t.Fatalf("reset schema: %v", err)
	}

	// 旧 DDL：author_user_id BIGINT NOT NULL + 唯一索引
	pgLegacy := `
CREATE TABLE course_review (
	id BIGSERIAL PRIMARY KEY,
	offering_id BIGINT NOT NULL DEFAULT 0,
	author_user_id BIGINT NOT NULL DEFAULT 0,
	rating INT,
	content TEXT NOT NULL DEFAULT '',
	is_anonymous BOOLEAN NOT NULL DEFAULT false,
	status SMALLINT NOT NULL DEFAULT 0,
	legacy_helpful_count INT NOT NULL DEFAULT 0,
	source TEXT NOT NULL DEFAULT '',
	created_at TIMESTAMPTZ,
	updated_at TIMESTAMPTZ,
	deleted_at TIMESTAMPTZ
);
CREATE UNIQUE INDEX uniq_course_review_offering_author ON course_review (offering_id, author_user_id);
`
	if err := db.Exec(pgLegacy).Error; err != nil {
		t.Fatalf("create legacy pg course_review: %v", err)
	}
	for id := 1; id <= 2; id++ {
		if err := db.Exec("INSERT INTO course_review (id, offering_id, author_user_id, status, content) VALUES (?, 100, ?, 0, 'pg row')", id, 1000+id).Error; err != nil {
			t.Fatalf("insert pg row %d: %v", id, err)
		}
	}

	// 升级（preflight 对 PG 直接跳过：AutoMigrate ALTER COLUMN 处理）
	if err := upgradeCourseReviewLegacySchema(db); err != nil {
		t.Fatalf("upgrade preflight on postgres failed: %v", err)
	}
	if err := db.AutoMigrate(&course.ReviewEntity{}); err != nil {
		t.Fatalf("upgrade AutoMigrate on postgres failed: %v", err)
	}
	var count int64
	if err := db.Table("course_review").Count(&count).Error; err != nil {
		t.Fatalf("count pg rows: %v", err)
	}
	if count != 2 {
		t.Fatalf("pg row count after upgrade = %d, want 2", count)
	}
	// 唯一索引保留
	if err := db.Exec("INSERT INTO course_review (offering_id, author_user_id, content) VALUES (100, 1001, 'dup')").Error; err == nil {
		t.Fatal("pg unique index lost after upgrade: duplicate accepted")
	}
	// DROP NOT NULL 后 NULL 可写（清理置 NULL 语义）
	if err := db.Table("course_review").Where("id = ?", 1).Update("author_user_id", nil).Error; err != nil {
		t.Fatalf("pg write NULL author_user_id after upgrade failed: %v", err)
	}
}
