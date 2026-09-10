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

// legacyImportRunEntity 旧版 course_import_run 模型（issue #183 前）：无 kind 字段，
// manifest_hash 单列唯一索引 uniq_course_import_run_manifest。用它 AutoMigrate 建出
// 与真实存量库一致的旧表（列类型/索引均由旧版 GORM 生成，手写 DDL 无法完全对齐）。
type legacyImportRunEntity struct {
	Id               uint64         `gorm:"primaryKey;column:id;autoIncrement;not null;" json:"id"`
	Source           string         `gorm:"column:source;type:varchar(64);not null;default:'';" json:"source"`
	ManifestHash     string         `gorm:"column:manifest_hash;type:varchar(64);not null;default:'';uniqueIndex:uniq_course_import_run_manifest;" json:"manifestHash"`
	Status           string         `gorm:"column:status;type:varchar(32);not null;default:'';" json:"status"`
	InsertedCount    int            `gorm:"column:inserted_count;not null;default:0;" json:"insertedCount"`
	UpdatedCount     int            `gorm:"column:updated_count;not null;default:0;" json:"updatedCount"`
	QuarantinedCount int            `gorm:"column:quarantined_count;not null;default:0;" json:"quarantinedCount"`
	ErrorCount       int            `gorm:"column:error_count;not null;default:0;" json:"errorCount"`
	StartedAt        *time.Time     `gorm:"column:started_at;" json:"startedAt"`
	FinishedAt       *time.Time     `gorm:"column:finished_at;" json:"finishedAt"`
	CreatedAt        time.Time      `gorm:"column:created_at;autoCreateTime;<-:create;" json:"createdAt"`
	UpdatedAt        time.Time      `gorm:"column:updated_at;autoUpdateTime;" json:"updatedAt"`
	DeletedAt        gorm.DeletedAt `json:"-"`
}

func (legacyImportRunEntity) TableName() string { return "course_import_run" }

// exerciseImportRunIndexUpgrade 在给定连接上执行完整升级路径并断言结果：
// 旧表（无 kind + manifest_hash 单列唯一）→ upgradeImportRunCompositeIndex
// （ADD COLUMN 保留存量数据 + 删旧索引）→ AutoMigrate 新模型（建复合唯一索引）→
// 同 hash 双 kind 可插、重复 (hash, kind) 被幂等拦截、存量行数据完整保留且 kind 回填 catalog。
func exerciseImportRunIndexUpgrade(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.AutoMigrate(&legacyImportRunEntity{}); err != nil {
		t.Fatalf("migrate legacy schema: %v", err)
	}
	legacy := legacyImportRunEntity{Source: "legacy", ManifestHash: "hash-1", Status: "completed"}
	if err := db.Create(&legacy).Error; err != nil {
		t.Fatalf("seed legacy run: %v", err)
	}

	if err := upgradeImportRunCompositeIndex(db); err != nil {
		t.Fatalf("upgrade index: %v", err)
	}
	if err := db.AutoMigrate(&course.ImportRunEntity{}); err != nil {
		t.Fatalf("automigrate: %v", err)
	}

	// 同 manifest_hash 的第二命令（reviews）必须能插入——旧单列唯一索引已删除。
	reviews := course.ImportRunEntity{Source: "legacy", ManifestHash: "hash-1", Kind: course.ImportKindReviews, Status: course.ImportStatusCompleted}
	if err := db.Create(&reviews).Error; err != nil {
		t.Fatalf("insert reviews run with same hash: %v", err)
	}
	// 重复 (hash, kind=catalog) 仍被复合唯一索引幂等拦截。
	dup := course.ImportRunEntity{Source: "legacy", ManifestHash: "hash-1", Kind: course.ImportKindCatalog, Status: course.ImportStatusRunning}
	if err := db.Create(&dup).Error; err == nil {
		t.Fatal("expected duplicate (manifest_hash, kind) insert to fail")
	}
	// 存量行：kind 回填 catalog，且 source/manifest_hash/status 完整保留
	// （升级必须不丢数据——GORM SQLite 迁移的整表重建路径有数据丢失风险，见 upgradeImportRunCompositeIndex 注释）。
	var got course.ImportRunEntity
	if err := db.First(&got, "id = ?", legacy.Id).Error; err != nil {
		t.Fatalf("load legacy run: %v", err)
	}
	if got.Kind != course.ImportKindCatalog {
		t.Fatalf("legacy run kind = %q, want catalog", got.Kind)
	}
	if got.ManifestHash != "hash-1" || got.Source != "legacy" || got.Status != course.ImportStatusCompleted {
		t.Fatalf("legacy run data lost after upgrade: %+v", got)
	}
}

// TestImportRunIndexUpgradeFromLegacySchema issue #183 升级路径（SQLite）：
// 存量库旧单列唯一索引必须被删除并重建为 (kind, manifest_hash) 复合唯一，
// 否则单包双命令导入（相同 manifest_hash）在存量库上必撞旧唯一约束。
func TestImportRunIndexUpgradeFromLegacySchema(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	exerciseImportRunIndexUpgrade(t, db)
}

// TestImportRunIndexUpgradeFromLegacySchemaOnPostgreSQL 同上，PostgreSQL 版
// （GORM 按索引名判重、AutoMigrate 不重建同名索引的行为在 PG 同样存在）。
// 依赖 YOURTJ_TEST_PG_URL，未设置时跳过。
func TestImportRunIndexUpgradeFromLegacySchemaOnPostgreSQL(t *testing.T) {
	dsn := os.Getenv("YOURTJ_TEST_PG_URL")
	if dsn == "" {
		t.Skip("YOURTJ_TEST_PG_URL not set; skipping PostgreSQL import run index upgrade test")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	// 清理可能残留的表（与 migration_pg_test 共享同一测试库）。
	if err := db.Migrator().DropTable(&course.ImportRunEntity{}); err != nil {
		t.Fatalf("drop leftover table: %v", err)
	}
	exerciseImportRunIndexUpgrade(t, db)
}
