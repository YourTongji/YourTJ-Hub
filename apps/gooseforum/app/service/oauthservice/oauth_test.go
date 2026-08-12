package oauthservice

import (
	"strings"
	"testing"
	"time"

	db "github.com/leancodebox/GooseForum/app/bundles/connect/dbconnect"
	"github.com/leancodebox/GooseForum/app/models/forum/userOAuth"
	"github.com/leancodebox/GooseForum/app/models/forum/users"
	"github.com/leancodebox/GooseForum/app/service/datamigration"
	"github.com/markbates/goth"
	"gorm.io/gorm"
)

// legacyOAuthSchema 是 Issue #131 之前的旧 user_o_auth 表结构，含 5 个明文凭据列。
const legacyOAuthSchema = `CREATE TABLE user_o_auth (
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
)`

// setupOAuthTestDB 构造 Issue #131 升级后的测试环境：先建含凭据列的旧表，
// 再执行 v15 迁移得到新表形态，并准备好 users 表。这样登录/绑定流程跑在
// 真实升级路径上，而非 AutoMigrate 出的全新结构。
func setupOAuthTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	conn := db.Connect()
	if err := conn.Exec(`DROP TABLE IF EXISTS user_o_auth`).Error; err != nil {
		t.Fatalf("drop user_o_auth: %v", err)
	}
	if err := conn.Exec(legacyOAuthSchema).Error; err != nil {
		t.Fatalf("create legacy user_o_auth: %v", err)
	}
	if err := conn.AutoMigrate(&users.EntityComplete{}); err != nil {
		t.Fatalf("migrate users: %v", err)
	}
	result := datamigration.DropUserOAuthTokenColumnsWithDB(conn)
	if result.Failed > 0 {
		t.Fatalf("migrate user_o_auth: %+v", result)
	}
	conn.Unscoped().Where("1 = 1").Delete(&users.EntityComplete{})
	return conn
}

// oauthTableColumns 读取 user_o_auth 的实际列名（PRAGMA），与列名硬编码解耦。
func oauthTableColumns(t *testing.T, conn *gorm.DB) []string {
	t.Helper()
	rows, err := conn.Raw("PRAGMA table_info(user_o_auth)").Rows()
	if err != nil {
		t.Fatalf("PRAGMA table_info(user_o_auth): %v", err)
	}
	defer rows.Close()
	var cols []string
	for rows.Next() {
		var (
			cid       int
			name      string
			typ       string
			notnull   int
			dfltValue any
			pk        int
		)
		if err := rows.Scan(&cid, &name, &typ, &notnull, &dfltValue, &pk); err != nil {
			t.Fatalf("scan column: %v", err)
		}
		cols = append(cols, name)
	}
	return cols
}

// assertNoCredentialColumns 断言 user_o_auth 不含任何凭据语义列。用子串匹配
// 保留列名之外的敏感词（token/secret/credential/scope/raw），能发现任何新列名
// 下重新引入的凭据持久化，而非只盯历史列名。
func assertNoCredentialColumns(t *testing.T, conn *gorm.DB) {
	t.Helper()
	for _, name := range oauthTableColumns(t, conn) {
		for _, keyword := range []string{"token", "secret", "credential", "scope", "raw"} {
			if strings.Contains(name, keyword) {
				t.Fatalf("user_o_auth 仍含凭据列 %q", name)
			}
		}
	}
}

// tokenizedGothUser 构造携带明文凭据的 goth 用户：若 token 仍被落库，断言会失败。
func tokenizedGothUser(provider, uid string) goth.User {
	return goth.User{
		Provider:          provider,
		UserID:            uid,
		AccessToken:       "secret-access-token",
		AccessTokenSecret: "oauth1-secret",
		RefreshToken:      "secret-refresh-token",
		ExpiresAt:         time.Now().Add(24 * time.Hour),
	}
}

// TestCreateOAuthRecordDoesNotPersistTokens 验证首次注册写路径（Issue #131）：
// 创建 OAuth 绑定只落库身份关联，绝不持久化任何第三方凭据。
func TestCreateOAuthRecordDoesNotPersistTokens(t *testing.T) {
	conn := setupOAuthTestDB(t)

	userInfo := parseOAuthUserInfo(tokenizedGothUser(ProviderGitHub, "uid-42"))
	if err := createOAuthRecord(42, userInfo); err != nil {
		t.Fatalf("createOAuthRecord() error = %v", err)
	}

	assertNoCredentialColumns(t, conn)

	entity := userOAuth.GetByProviderAndUID(ProviderGitHub, "uid-42")
	if entity == nil {
		t.Fatal("OAuth binding was not created")
	}
	if entity.UserId != 42 || entity.Provider != ProviderGitHub || entity.ProviderUid != "uid-42" {
		t.Fatalf("binding = %+v, want user 42 github uid-42", entity)
	}
}

// TestProcessOAuthCallbackExistingUserDoesNotRewriteTokens 验证重复登录既有用户
// 的分支：不再把 goth token 重新写入绑定记录。
func TestProcessOAuthCallbackExistingUserDoesNotRewriteTokens(t *testing.T) {
	conn := setupOAuthTestDB(t)
	user := users.MakeUser("re-login", "password", "relogin@example.com")
	if err := users.Create(user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := userOAuth.Create(&userOAuth.Entity{UserId: user.Id, Provider: ProviderGitHub, ProviderUid: "uid-7"}); err != nil {
		t.Fatalf("seed binding: %v", err)
	}

	got, err := ProcessOAuthCallback(tokenizedGothUser(ProviderGitHub, "uid-7"))
	if err != nil {
		t.Fatalf("ProcessOAuthCallback() error = %v", err)
	}
	if got.Id != user.Id {
		t.Fatalf("returned user id = %d, want %d", got.Id, user.Id)
	}

	assertNoCredentialColumns(t, conn)
	var count int64
	conn.Model(&userOAuth.Entity{}).Count(&count)
	if count != 1 {
		t.Fatalf("binding count = %d, want 1", count)
	}
}

// TestProcessOAuthBindAlreadyBoundDoesNotRewriteTokens 验证已绑定用户的重复绑定
// 分支同样不写凭据。
func TestProcessOAuthBindAlreadyBoundDoesNotRewriteTokens(t *testing.T) {
	conn := setupOAuthTestDB(t)
	user := users.MakeUser("re-bind", "password", "rebind@example.com")
	if err := users.Create(user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := userOAuth.Create(&userOAuth.Entity{UserId: user.Id, Provider: ProviderGitHub, ProviderUid: "uid-9"}); err != nil {
		t.Fatalf("seed binding: %v", err)
	}

	if err := ProcessOAuthBind(user.Id, tokenizedGothUser(ProviderGitHub, "uid-9")); err != nil {
		t.Fatalf("ProcessOAuthBind() error = %v", err)
	}

	assertNoCredentialColumns(t, conn)
	var count int64
	conn.Model(&userOAuth.Entity{}).Count(&count)
	if count != 1 {
		t.Fatalf("binding count = %d, want 1", count)
	}
}

// TestUnbindOAuthRemovesBinding 验证解绑仍正常删除绑定行。
func TestUnbindOAuthRemovesBinding(t *testing.T) {
	setupOAuthTestDB(t)
	user := users.MakeUser("unbind", "password", "unbind@example.com")
	if err := users.Create(user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := userOAuth.Create(&userOAuth.Entity{UserId: user.Id, Provider: ProviderGitHub, ProviderUid: "uid-11"}); err != nil {
		t.Fatalf("seed binding: %v", err)
	}

	if err := UnbindOAuth(user.Id, ProviderGitHub); err != nil {
		t.Fatalf("UnbindOAuth() error = %v", err)
	}
	if entity := userOAuth.GetByProviderAndUID(ProviderGitHub, "uid-11"); entity != nil {
		t.Fatalf("binding still exists after unbind: %+v", entity)
	}
}
