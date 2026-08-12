package migration

import (
	"errors"
	"os"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pointsRecord"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userOAuth"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// TestSchemaMigratesOnPostgreSQL 验证全部主库模型能在 PostgreSQL 上完成 AutoMigrate。
//
// 回归背景（issue #8）：user_sessions / user_totp / user_totp_recovery_codes 曾硬编码
// MySQL 专用类型（bigint unsigned / datetime / tinyint），在 PostgreSQL 上建表失败，
// 而迁移错误只记日志不退出，服务带着残缺 schema 启动，登录/注册接口运行期才报错。
//
// 通过环境变量 YOURTJ_TEST_PG_URL 提供 PostgreSQL DSN；未设置时跳过。
// 测试使用独立数据库，会重建 public schema，不会触碰业务数据。
func TestSchemaMigratesOnPostgreSQL(t *testing.T) {
	dsn := os.Getenv("YOURTJ_TEST_PG_URL")
	if dsn == "" {
		t.Skip("YOURTJ_TEST_PG_URL not set; skipping PostgreSQL migration test")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}

	// 清空 schema 保证幂等，然后全量迁移
	if err := db.Exec(`DROP SCHEMA public CASCADE; CREATE SCHEMA public;`).Error; err != nil {
		t.Fatalf("reset schema: %v", err)
	}
	if err := validateUniqueUsernames(db); err != nil {
		t.Fatalf("username preflight on fresh postgres failed: %v", err)
	}
	if err := db.AutoMigrate(SchemaModels()...); err != nil {
		t.Fatalf("AutoMigrate on postgres failed: %v", err)
	}

	// 关键新表必须存在（issue #8 回归点；agents 为本仓库 Agent 模型新增表）
	for _, table := range []string{
		"user_sessions",
		"user_totp",
		"user_totp_recovery_codes",
		"user_totp_challenges",
		"user_o_auth",
		"oidc_auth_requests",
		"oidc_access_tokens",
		"users",
		"agents",
		// Issue #186：一系统同步管线（course-pk-sync）新增 PK 域表。
		"pk_calendar",
		"pk_language",
		"pk_course_nature",
		"pk_course_nature_by_calendar",
		"pk_assessment",
		"pk_campus",
		"pk_faculty",
		"pk_major",
		"pk_major_course",
		"pk_course_detail",
		"pk_teacher",
		"pk_teacher_timeslot",
		"pk_fetch_log",
	} {
		if !db.Migrator().HasTable(table) {
			t.Errorf("table %q missing after postgres migration", table)
		}
	}
	// Issue #83：users.email_changed_at 列必须存在（邮箱变更冷静期依赖此列）。
	if !db.Migrator().HasColumn(&users.EntityComplete{}, "email_changed_at") {
		t.Error("users.email_changed_at column missing after postgres migration")
	}
	if !db.Migrator().HasColumn(&users.EntityComplete{}, "actor_type") {
		t.Error("users.actor_type column missing after postgres migration")
	}
	assertPointsSourceKeySchema(t, db)
}

// TestSchemaUpgradeCreatesNewTablesOnPostgreSQL 模拟存量实例升级场景：
// 旧库（如 main 的 v0.0.6）没有 issue #8 新增的表，部署新二进制后
// AutoMigrate 必须自动补齐这些表。这是 main 未来自动部署不出问题的直接保障。
func TestSchemaUpgradeCreatesNewTablesOnPostgreSQL(t *testing.T) {
	dsn := os.Getenv("YOURTJ_TEST_PG_URL")
	if dsn == "" {
		t.Skip("YOURTJ_TEST_PG_URL not set; skipping PostgreSQL migration test")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}

	// 先只迁移旧版已有的部分表，模拟 issue #8 之前的 main 库
	if err := db.Exec(`DROP SCHEMA public CASCADE; CREATE SCHEMA public;`).Error; err != nil {
		t.Fatalf("reset schema: %v", err)
	}
	legacy := []any{&users.EntityComplete{}, &userOAuth.Entity{}, &topics.Entity{}, &pointsRecord.Entity{}}
	if err := db.AutoMigrate(legacy...); err != nil {
		t.Fatalf("AutoMigrate legacy subset failed: %v", err)
	}
	// The current model includes actor_type, but a true legacy users table did
	// not. Remove it before the upgrade so this test exercises column addition.
	if err := db.Migrator().DropColumn(&users.EntityComplete{}, "actor_type"); err != nil {
		t.Fatalf("drop legacy-missing users.actor_type: %v", err)
	}
	if err := db.Migrator().DropColumn(&users.EntityComplete{}, "email_changed_at"); err != nil {
		t.Fatalf("drop legacy-missing users.email_changed_at: %v", err)
	}
	// The current model also includes the username unique index. Remove it so
	// the upgrade phase proves AutoMigrate creates the constraint for legacy DBs.
	if err := db.Migrator().DropIndex(&users.EntityComplete{}, "uniq_users_username"); err != nil {
		t.Fatalf("drop legacy-missing username index: %v", err)
	}
	if err := db.Migrator().DropColumn(&pointsRecord.Entity{}, "source_key"); err != nil {
		t.Fatalf("drop legacy-missing points_record.source_key: %v", err)
	}
	if err := db.Exec("INSERT INTO points_record (user_id, action, points_change, created_at) VALUES (?, ?, ?, NOW())", 1, "init", 100).Error; err != nil {
		t.Fatalf("insert legacy points record: %v", err)
	}
	if db.Migrator().HasColumn(&users.EntityComplete{}, "actor_type") {
		t.Fatal("precondition failed: legacy users table should not have actor_type")
	}
	if db.Migrator().HasColumn(&users.EntityComplete{}, "email_changed_at") {
		t.Fatal("precondition failed: legacy users table should not have email_changed_at")
	}
	if db.Migrator().HasIndex(&users.EntityComplete{}, "uniq_users_username") {
		t.Fatal("precondition failed: legacy users table should not have username unique index")
	}
	if db.Migrator().HasColumn(&pointsRecord.Entity{}, "source_key") {
		t.Fatal("precondition failed: legacy points_record table should not have source_key")
	}
	if db.Migrator().HasTable("user_sessions") {
		t.Fatal("precondition failed: legacy schema should not have user_sessions")
	}

	// 部署新二进制：全量 AutoMigrate 应补齐新表且不破坏旧表
	if err := validateUniqueUsernames(db); err != nil {
		t.Fatalf("username preflight on postgres upgrade failed: %v", err)
	}
	if err := db.AutoMigrate(SchemaModels()...); err != nil {
		t.Fatalf("upgrade AutoMigrate on postgres failed: %v", err)
	}
	for _, table := range []string{
		"user_sessions",
		"user_totp",
		"user_totp_recovery_codes",
		"user_totp_challenges",
		"user_o_auth",
		"oidc_auth_requests",
		"oidc_access_tokens",
		"users",
		"topics",
		"agents",
		// Issue #186：升级存量实例时同样需补齐 PK 域表。
		"pk_calendar",
		"pk_course_detail",
		"pk_teacher",
		"pk_teacher_timeslot",
		"pk_fetch_log",
	} {
		if !db.Migrator().HasTable(table) {
			t.Errorf("table %q missing after upgrade migration", table)
		}
	}
	if !db.Migrator().HasColumn(&users.EntityComplete{}, "actor_type") {
		t.Error("users.actor_type column missing after upgrade migration")
	}
	if !db.Migrator().HasColumn(&users.EntityComplete{}, "email_changed_at") {
		t.Error("users.email_changed_at column missing after upgrade migration")
	}
	if !db.Migrator().HasIndex(&users.EntityComplete{}, "uniq_users_username") {
		t.Error("users username unique index missing after upgrade migration")
	}
	assertPointsSourceKeySchema(t, db)
	var legacyPointsCount int64
	if err := db.Model(&pointsRecord.Entity{}).Where("action = ? AND points_change = ?", "init", 100).Count(&legacyPointsCount).Error; err != nil {
		t.Fatalf("count legacy points records after upgrade: %v", err)
	}
	if legacyPointsCount != 1 {
		t.Errorf("legacy points record count after upgrade = %d, want 1", legacyPointsCount)
	}
}

func assertPointsSourceKeySchema(t *testing.T, db *gorm.DB) {
	t.Helper()
	if !db.Migrator().HasColumn(&pointsRecord.Entity{}, "source_key") {
		t.Error("points_record.source_key column missing after postgres migration")
	}
	indexes, err := db.Migrator().GetIndexes(&pointsRecord.Entity{})
	if err != nil {
		t.Fatalf("list points_record indexes: %v", err)
	}
	uniqueSourceKey := false
	for _, index := range indexes {
		if index.Name() != "idx_points_record_source_key" {
			continue
		}
		unique, ok := index.Unique()
		uniqueSourceKey = ok && unique
		break
	}
	if !uniqueSourceKey {
		t.Error("points_record.source_key unique index missing after postgres migration")
	}

	if err := db.Create(&pointsRecord.Entity{UserId: 9001, Action: "legacy", PointsChange: 1}).Error; err != nil {
		t.Fatalf("insert first NULL source_key record: %v", err)
	}
	if err := db.Create(&pointsRecord.Entity{UserId: 9002, Action: "legacy", PointsChange: 1}).Error; err != nil {
		t.Fatalf("insert second NULL source_key record: %v", err)
	}
	key := "schema-test:unique"
	if err := db.Create(&pointsRecord.Entity{UserId: 9003, Action: "test", SourceKey: &key}).Error; err != nil {
		t.Fatalf("insert unique source_key record: %v", err)
	}
	if err := db.Create(&pointsRecord.Entity{UserId: 9004, Action: "test", SourceKey: &key}).Error; !errors.Is(err, gorm.ErrDuplicatedKey) {
		t.Fatalf("duplicate source_key error = %v, want gorm.ErrDuplicatedKey", err)
	}
}
