package migration

import (
	"os"
	"testing"

	"github.com/leancodebox/GooseForum/app/models/forum/topics"
	"github.com/leancodebox/GooseForum/app/models/forum/userOAuth"
	"github.com/leancodebox/GooseForum/app/models/forum/users"
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

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}

	// 清空 schema 保证幂等，然后全量迁移
	if err := db.Exec(`DROP SCHEMA public CASCADE; CREATE SCHEMA public;`).Error; err != nil {
		t.Fatalf("reset schema: %v", err)
	}
	if err := db.AutoMigrate(SchemaModels()...); err != nil {
		t.Fatalf("AutoMigrate on postgres failed: %v", err)
	}

	// 关键新表必须存在（issue #8 回归点）
	for _, table := range []string{
		"user_sessions",
		"user_totp",
		"user_totp_recovery_codes",
		"user_totp_challenges",
		"user_o_auth",
		"oidc_auth_requests",
		"oidc_access_tokens",
		"users",
	} {
		if !db.Migrator().HasTable(table) {
			t.Errorf("table %q missing after postgres migration", table)
		}
	}
}

// TestSchemaUpgradeCreatesNewTablesOnPostgreSQL 模拟存量实例升级场景：
// 旧库（如 main 的 v0.0.6）没有 issue #8 新增的表，部署新二进制后
// AutoMigrate 必须自动补齐这些表。这是 main 未来自动部署不出问题的直接保障。
func TestSchemaUpgradeCreatesNewTablesOnPostgreSQL(t *testing.T) {
	dsn := os.Getenv("YOURTJ_TEST_PG_URL")
	if dsn == "" {
		t.Skip("YOURTJ_TEST_PG_URL not set; skipping PostgreSQL migration test")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}

	// 先只迁移旧版已有的部分表，模拟 issue #8 之前的 main 库
	if err := db.Exec(`DROP SCHEMA public CASCADE; CREATE SCHEMA public;`).Error; err != nil {
		t.Fatalf("reset schema: %v", err)
	}
	legacy := []any{&users.EntityComplete{}, &userOAuth.Entity{}, &topics.Entity{}}
	if err := db.AutoMigrate(legacy...); err != nil {
		t.Fatalf("AutoMigrate legacy subset failed: %v", err)
	}
	if db.Migrator().HasTable("user_sessions") {
		t.Fatal("precondition failed: legacy schema should not have user_sessions")
	}

	// 部署新二进制：全量 AutoMigrate 应补齐新表且不破坏旧表
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
	} {
		if !db.Migrator().HasTable(table) {
			t.Errorf("table %q missing after upgrade migration", table)
		}
	}
}
