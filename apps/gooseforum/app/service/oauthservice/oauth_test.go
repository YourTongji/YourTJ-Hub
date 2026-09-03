package oauthservice

import (
	"net/url"
	"strings"
	"testing"
	"time"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/jsonopt"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userOAuth"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/datamigration"
	"github.com/markbates/goth"
	"gorm.io/gorm"
)

func setupGoogleOAuthProviderConfig(t *testing.T, siteURL string) {
	t.Helper()
	conn := db.Connect()
	if err := conn.AutoMigrate(&pageConfig.Entity{}); err != nil {
		t.Fatalf("migrate page_config: %v", err)
	}
	conn.Where("page_type = ?", pageConfig.SiteSettings).Delete(&pageConfig.Entity{})
	if siteURL != "" {
		pageConfig.CreateOrSave(&pageConfig.Entity{
			PageType: pageConfig.SiteSettings,
			Config:   jsonopt.Encode(pageConfig.SiteSettingsConfig{SiteUrl: siteURL}),
		})
	}
	hotdataserve.ClearSiteSettingsConfigCache()
	preferences.Set("google.client_id", "test-google-client-id")
	preferences.Set("google.client_secret", "test-google-client-secret")
	googleProvider.Store(nil)
	t.Cleanup(func() {
		preferences.Set("google.client_id", "")
		preferences.Set("google.client_secret", "")
		googleProvider.Store(nil)
		conn.Where("page_type = ?", pageConfig.SiteSettings).Delete(&pageConfig.Entity{})
		hotdataserve.ClearSiteSettingsConfigCache()
	})
}

func TestInitGoogleProviderUsesConfiguredCredentialsAndScopes(t *testing.T) {
	setupGoogleOAuthProviderConfig(t, "https://hub.example.test/")

	if !IsGoogleOAuthConfigured() {
		t.Fatal("IsGoogleOAuthConfigured() = false, want true")
	}
	InitOAuth()
	if !IsGoogleOAuthReady() {
		t.Fatal("IsGoogleOAuthReady() = false after InitOAuth(), want true")
	}
	provider := initGoogleProvider()
	if provider == nil {
		t.Fatal("initGoogleProvider() returned nil with valid configuration")
	}
	if provider.ClientKey != "test-google-client-id" || provider.Secret != "test-google-client-secret" {
		t.Fatalf("provider credentials = %q/%q, want test credentials", provider.ClientKey, provider.Secret)
	}
	if provider.CallbackURL != "https://hub.example.test/api/auth/google/callback" {
		t.Fatalf("provider callback URL = %q, want absolute site callback", provider.CallbackURL)
	}

	session, err := provider.BeginAuth("test-state")
	if err != nil {
		t.Fatalf("BeginAuth() error = %v", err)
	}
	authURL, err := session.GetAuthURL()
	if err != nil {
		t.Fatalf("GetAuthURL() error = %v", err)
	}
	parsed, err := url.Parse(authURL)
	if err != nil {
		t.Fatalf("parse auth URL: %v", err)
	}
	if got := parsed.Query().Get("scope"); got != "openid email profile" {
		t.Fatalf("Google OAuth scope = %q, want %q", got, "openid email profile")
	}
}

func TestGoogleOAuthReadyRequiresMatchingStartupRegistration(t *testing.T) {
	setupGoogleOAuthProviderConfig(t, "https://hub.example.test")

	if IsGoogleOAuthReady() {
		t.Fatal("IsGoogleOAuthReady() = true before InitOAuth(), want false")
	}
	InitOAuth()
	if !IsGoogleOAuthReady() {
		t.Fatal("IsGoogleOAuthReady() = false after InitOAuth(), want true")
	}

	preferences.Set("google.client_secret", "changed-after-startup")
	if IsGoogleOAuthReady() {
		t.Fatal("IsGoogleOAuthReady() = true after changing live credentials, want false")
	}
}

func TestParseGoogleVerifiedEmail(t *testing.T) {
	verified := parseOAuthUserInfo(goth.User{
		Provider: ProviderGoogle,
		Email:    " Alice@Example.Test ",
		RawData:  map[string]any{"email_verified": true},
	})
	if verified.VerifiedEmail != "alice@example.test" || !verified.EmailVerified {
		t.Fatalf("verified Google email = %q, emailVerified = %v; want normalized trusted email", verified.VerifiedEmail, verified.EmailVerified)
	}

	unverified := parseOAuthUserInfo(goth.User{
		Provider: ProviderGoogle,
		Email:    "unverified@example.test",
		RawData:  map[string]any{"email_verified": false},
	})
	if unverified.VerifiedEmail != "" || unverified.EmailVerified {
		t.Fatalf("unverified Google email = %q, emailVerified = %v; want no trusted email", unverified.VerifiedEmail, unverified.EmailVerified)
	}
}

func TestGoogleOAuthRequiresSiteURL(t *testing.T) {
	setupGoogleOAuthProviderConfig(t, "")

	if IsGoogleOAuthConfigured() {
		t.Fatal("IsGoogleOAuthConfigured() = true without site URL")
	}
	if provider := initGoogleProvider(); provider != nil {
		t.Fatal("initGoogleProvider() returned a provider without site URL")
	}
}

func TestGoogleOAuthRequiresAbsoluteSiteURL(t *testing.T) {
	setupGoogleOAuthProviderConfig(t, "hub.example.test")

	if IsGoogleOAuthConfigured() {
		t.Fatal("IsGoogleOAuthConfigured() = true with relative site URL")
	}
	if provider := initGoogleProvider(); provider != nil {
		t.Fatal("initGoogleProvider() returned a provider with relative site URL")
	}
}

func TestGoogleOAuthRequiresCredentials(t *testing.T) {
	setupGoogleOAuthProviderConfig(t, "https://hub.example.test")

	preferences.Set("google.client_secret", "")
	if IsGoogleOAuthConfigured() {
		t.Fatal("IsGoogleOAuthConfigured() = true without client secret")
	}
	if provider := initGoogleProvider(); provider != nil {
		t.Fatal("initGoogleProvider() returned a provider without client secret")
	}
}

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
