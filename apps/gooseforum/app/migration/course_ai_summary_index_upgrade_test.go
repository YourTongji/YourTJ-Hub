package migration

import (
	"os"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/course"
	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// legacyCourseAiSummaryEntity 旧版 course_ai_summary 模型（#342 合入时形态）：
// status 列带 index tag，AutoMigrate 会在存量库上建出 idx_course_ai_summary_status。
// 用它 AutoMigrate 建出与真实存量库一致的旧表，验证 #343 升级步骤能删掉冗余索引。
type legacyCourseAiSummaryEntity struct {
	CourseId      uint64    `gorm:"column:course_id;primaryKey;not null;" json:"courseId"`
	SummaryJson   string    `gorm:"column:summary_json;type:text;not null;default:'';" json:"summaryJson"`
	Model         string    `gorm:"column:model;type:varchar(128);not null;default:'';" json:"model"`
	PromptVersion string    `gorm:"column:prompt_version;type:varchar(64);not null;default:'';" json:"promptVersion"`
	GeneratedAt   time.Time `gorm:"column:generated_at;not null;" json:"generatedAt"`
	Status        string    `gorm:"column:status;type:varchar(16);not null;default:'generated';index;comment:summary row status;" json:"status"`
}

func (legacyCourseAiSummaryEntity) TableName() string { return "course_ai_summary" }

// exerciseCourseAiSummaryStatusIndexUpgrade 在给定连接上执行完整升级路径并断言结果：
// 旧表（status 带 index）→ upgradeCourseAiSummaryStatusIndex（删索引）→
// AutoMigrate 新模型（无 index tag，不重建）→ 再次执行升级步骤幂等。
func exerciseCourseAiSummaryStatusIndexUpgrade(t *testing.T, db *gorm.DB) {
	t.Helper()
	// 1. 旧形态建表（status 带 index），与 #342 合入后的存量库一致。
	if err := db.AutoMigrate(&legacyCourseAiSummaryEntity{}); err != nil {
		t.Fatalf("migrate legacy schema: %v", err)
	}
	if !db.Migrator().HasIndex(&course.CourseAiSummaryEntity{}, "idx_course_ai_summary_status") {
		t.Fatal("precondition failed: legacy schema should have idx_course_ai_summary_status")
	}
	// 2. 升级步骤：显式删除冗余索引。
	if err := upgradeCourseAiSummaryStatusIndex(db); err != nil {
		t.Fatalf("upgrade status index: %v", err)
	}
	if db.Migrator().HasIndex(&course.CourseAiSummaryEntity{}, "idx_course_ai_summary_status") {
		t.Fatal("idx_course_ai_summary_status still present after upgrade")
	}
	// 3. AutoMigrate 新模型：模型已无 index tag，索引不得被重建。
	if err := db.AutoMigrate(&course.CourseAiSummaryEntity{}); err != nil {
		t.Fatalf("automigrate: %v", err)
	}
	if db.Migrator().HasIndex(&course.CourseAiSummaryEntity{}, "idx_course_ai_summary_status") {
		t.Fatal("idx_course_ai_summary_status recreated after automigrate")
	}
	// 4. 幂等：存量库每次启动都会执行 upgrade 步骤，重复执行不得报错。
	if err := upgradeCourseAiSummaryStatusIndex(db); err != nil {
		t.Fatalf("upgrade status index second run: %v", err)
	}
}

// TestCourseAiSummaryStatusIndexUpgradeFromLegacySchema #343 review：
// 存量库（#342 已建 idx_course_ai_summary_status）升级必须显式删索引——
// GORM AutoMigrate 按索引名判重，模型移除 index tag 不会自动删除既有索引。
// SQLite 版。
func TestCourseAiSummaryStatusIndexUpgradeFromLegacySchema(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	exerciseCourseAiSummaryStatusIndexUpgrade(t, db)
}

// TestCourseAiSummaryStatusIndexUpgradeFromLegacySchemaOnPostgreSQL 同上，
// PostgreSQL 版（bot review 点名要求验证 PG 升级路径）。
// 依赖 YOURTJ_TEST_PG_URL，未设置时跳过。
func TestCourseAiSummaryStatusIndexUpgradeFromLegacySchemaOnPostgreSQL(t *testing.T) {
	dsn := os.Getenv("YOURTJ_TEST_PG_URL")
	if dsn == "" {
		t.Skip("YOURTJ_TEST_PG_URL not set; skipping PostgreSQL course_ai_summary index upgrade test")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	// 清理可能残留的表（与 migration_pg_test 共享同一测试库）。
	if err := db.Migrator().DropTable(&course.CourseAiSummaryEntity{}); err != nil {
		t.Fatalf("drop leftover table: %v", err)
	}
	exerciseCourseAiSummaryStatusIndexUpgrade(t, db)
}
