package datamigration

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// legacyUserOAuthCredentialColumns 是 Issue #131 之前 user_o_auth 表上的
// 明文凭据列。生产主库为 SQLite（deploy/config.toml.example），该建表语句
// 模拟升级前已含凭据列的旧库。
func TestDropUserOAuthTokenColumnsRemovesLegacyCredentialColumns(t *testing.T) {
	conn, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := conn.Exec(`CREATE TABLE user_o_auth (
		id integer primary key autoincrement,
		user_id integer not null default 0,
		provider varchar(32) default '0',
		provider_uid varchar(255) not null default '',
		access_token varchar(1024) not null default '',
		refresh_token varchar(1024) not null default '',
		token_expiry datetime,
		scopes text,
		raw_user_data text,
		created_at datetime,
		updated_at datetime
	)`).Error; err != nil {
		t.Fatalf("create table: %v", err)
	}
	if err := conn.Exec(
		`INSERT INTO user_o_auth (user_id, provider, provider_uid, access_token, refresh_token) VALUES (1, 'github', 'uid-1', 'secret-access', 'secret-refresh')`,
	).Error; err != nil {
		t.Fatalf("insert rows: %v", err)
	}

	result := DropUserOAuthTokenColumnsWithDB(conn)
	if result.Failed > 0 {
		t.Fatalf("migration failed: %+v", result)
	}
	for _, col := range []string{"access_token", "refresh_token", "token_expiry", "scopes", "raw_user_data"} {
		if conn.Migrator().HasColumn("user_o_auth", col) {
			t.Fatalf("column %s still exists after migration", col)
		}
	}
	if len(result.Dropped) != 5 {
		t.Fatalf("Dropped = %v, want 5 columns", result.Dropped)
	}

	var rows []struct {
		UserID      uint64 `gorm:"column:user_id"`
		Provider    string
		ProviderUid string `gorm:"column:provider_uid"`
	}
	if err := conn.Table("user_o_auth").Find(&rows).Error; err != nil {
		t.Fatalf("read rows: %v", err)
	}
	if len(rows) != 1 || rows[0].Provider != "github" || rows[0].ProviderUid != "uid-1" {
		t.Fatalf("preserved binding data = %+v, want github/uid-1", rows)
	}

	// 已迁移的旧库再次运行迁移必须是 no-op（幂等）。
	second := DropUserOAuthTokenColumnsWithDB(conn)
	if second.Failed > 0 || len(second.Dropped) != 0 {
		t.Fatalf("second migration run = %+v, want no-op", second)
	}
}

// 迁移的目标表不存在时（如全新安装）必须无副作用地早退。
func TestDropUserOAuthTokenColumnsSkipsMissingTable(t *testing.T) {
	conn, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	result := DropUserOAuthTokenColumnsWithDB(conn)
	if result.Failed > 0 || len(result.Dropped) != 0 {
		t.Fatalf("missing-table migration result = %+v, want no-op", result)
	}
}

// 任一列 DROP 失败时，迁移必须停下并记录 LastFailed，不得继续尝试后续列
// （上层据此不推进迁移版本，服务下次启动重试）。
func TestDropUserOAuthTokenColumnsStopsOnDropFailure(t *testing.T) {
	conn, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	// access_token 被一个部分索引（WHERE 子句）引用，SQLite 会拒绝 DROP 该列。
	if err := conn.Exec(`CREATE TABLE user_o_auth (
		id integer primary key autoincrement,
		user_id integer not null default 0,
		provider varchar(32) default '0',
		provider_uid varchar(255) not null default '',
		access_token varchar(1024) not null default '',
		refresh_token varchar(1024) not null default ''
	)`).Error; err != nil {
		t.Fatalf("create table: %v", err)
	}
	if err := conn.Exec(`CREATE INDEX idx_oauth_partial ON user_o_auth(user_id) WHERE access_token <> ''`).Error; err != nil {
		t.Fatalf("create partial index: %v", err)
	}

	result := DropUserOAuthTokenColumnsWithDB(conn)
	if result.Failed == 0 {
		t.Fatalf("migration should fail when DROP blocked by partial index, got %+v", result)
	}
	if result.LastFailed == "" {
		t.Fatal("LastFailed should be populated on failure")
	}
	if !conn.Migrator().HasColumn("user_o_auth", "refresh_token") {
		t.Fatal("refresh_token was touched after failure; migration must stop at first failed column")
	}
}

// 全新库（从未包含凭据列）上迁移必须是无副作用的 no-op。
func TestDropUserOAuthTokenColumnsSkipsFreshSchema(t *testing.T) {
	conn, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := conn.Exec(`CREATE TABLE user_o_auth (
		id integer primary key autoincrement,
		user_id integer not null default 0,
		provider varchar(32) default '0',
		provider_uid varchar(255) not null default '',
		created_at datetime,
		updated_at datetime
	)`).Error; err != nil {
		t.Fatalf("create table: %v", err)
	}

	result := DropUserOAuthTokenColumnsWithDB(conn)
	if result.Failed > 0 || len(result.Dropped) != 0 {
		t.Fatalf("fresh schema migration result = %+v, want no-op", result)
	}
}
