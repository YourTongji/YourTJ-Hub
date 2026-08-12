package oauthservice

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pointsRecord"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userOAuth"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userPoints"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userStatistics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
	"github.com/markbates/goth"
)

// setSecurityConfigForTest 注入 security 配置并清缓存，返回还原函数。
func setSecurityConfigForTest(t *testing.T, config pageConfig.SecurityAndRegistration) {
	t.Helper()
	conn := db.Connect()
	// CreateUser 依赖 user_points / user_statistics / points_record，迁移避免缺表。
	if err := conn.AutoMigrate(
		&pageConfig.Entity{},
		&userPoints.Entity{},
		&pointsRecord.Entity{},
		&userStatistics.Entity{},
	); err != nil {
		t.Fatalf("migrate config/points tables: %v", err)
	}
	encoded, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("encode security config: %v", err)
	}
	entity := pageConfig.Entity{PageType: pageConfig.SecuritySettings, Config: string(encoded)}
	if err := conn.Where("page_type = ?", pageConfig.SecuritySettings).Assign(entity).FirstOrCreate(&entity).Error; err != nil {
		t.Fatalf("save security config: %v", err)
	}
	hotdataserve.ClearSecuritySettingsConfigCache()
	t.Cleanup(hotdataserve.ClearSecuritySettingsConfigCache)
}

// verifiedGithubUser 构造带 verified 邮箱的 GitHub goth 用户（token 触发 API 调用）。
func verifiedGithubUser(t *testing.T, uid, login, email string) goth.User {
	t.Helper()
	// 用 mock server 返回 verified primary 邮箱，避免真实网络调用。
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{"email": email, "primary": true, "verified": true},
		})
	}))
	t.Cleanup(server.Close)
	oldURL := gitHubEmailAPIURL
	gitHubEmailAPIURL = server.URL
	t.Cleanup(func() { gitHubEmailAPIURL = oldURL })

	return goth.User{
		Provider:    ProviderGitHub,
		UserID:      uid,
		NickName:    login,
		AccessToken: "test-token",
	}
}

// unverifiedGithubUser 构造无 verified 邮箱的 GitHub 用户（API 返回空列表）。
func unverifiedGithubUser(t *testing.T, uid, login string) goth.User {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[]`))
	}))
	t.Cleanup(server.Close)
	oldURL := gitHubEmailAPIURL
	gitHubEmailAPIURL = server.URL
	t.Cleanup(func() { gitHubEmailAPIURL = oldURL })

	return goth.User{
		Provider:    ProviderGitHub,
		UserID:      uid,
		NickName:    login,
		AccessToken: "test-token",
	}
}

// TestCreateUserFromOAuthTrustedDomainAutoActivates 域名命中信任列表 → 直接激活。
func TestCreateUserFromOAuthTrustedDomainAutoActivates(t *testing.T) {
	setupOAuthTestDB(t)
	setSecurityConfigForTest(t, pageConfig.SecurityAndRegistration{
		AllowedDomains: []string{"tongji.edu.cn"},
	})

	user, err := createUserFromOAuth(OAuthUserInfo{
		ID: "1", Login: "trusted-user", Provider: ProviderGitHub,
		VerifiedEmail: "alice@tongji.edu.cn", EmailVerified: true,
	})
	if err != nil {
		t.Fatalf("createUserFromOAuth() error = %v", err)
	}
	if user.IsActivated != users.ActivationSuccess {
		t.Fatalf("trusted domain user activation = %v, want %v", user.IsActivated, users.ActivationSuccess)
	}
	if user.Email != "alice@tongji.edu.cn" {
		t.Fatalf("stored email = %q, want alice@tongji.edu.cn", user.Email)
	}
}

// TestCreateUserFromOAuthEmptyDomainsTrustsAll 空 allowedDomains = 全信任 → 直接激活。
func TestCreateUserFromOAuthEmptyDomainsTrustsAll(t *testing.T) {
	setupOAuthTestDB(t)
	setSecurityConfigForTest(t, pageConfig.SecurityAndRegistration{
		EnableEmailVerification: true,
		AllowedDomains:          []string{},
	})

	user, err := createUserFromOAuth(OAuthUserInfo{
		ID: "1", Login: "any-domain", Provider: ProviderGitHub,
		VerifiedEmail: "alice@example.com", EmailVerified: true,
	})
	if err != nil {
		t.Fatalf("createUserFromOAuth() error = %v", err)
	}
	if user.IsActivated != users.ActivationSuccess {
		t.Fatalf("empty domains should trust all, activation = %v, want %v", user.IsActivated, users.ActivationSuccess)
	}
}

// TestCreateUserFromOAuthUnmatchedDomainFollowsSwitch 未命中域名跟随开关：
// 开关开 → ActivationPending；开关关（默认）→ 免验证激活。
func TestCreateUserFromOAuthUnmatchedDomainFollowsSwitch(t *testing.T) {
	setupOAuthTestDB(t)
	setSecurityConfigForTest(t, pageConfig.SecurityAndRegistration{
		EnableEmailVerification: true,
		AllowedDomains:          []string{"tongji.edu.cn"},
	})

	// 开关开 + 未命中 → pending
	user, err := createUserFromOAuth(OAuthUserInfo{
		ID: "1", Login: "outside-user", Provider: ProviderGitHub,
		VerifiedEmail: "bob@gmail.com", EmailVerified: true,
	})
	if err != nil {
		t.Fatalf("createUserFromOAuth() error = %v", err)
	}
	if user.IsActivated != users.ActivationPending {
		t.Fatalf("unmatched domain with verification on: activation = %v, want pending", user.IsActivated)
	}

	// 开关关（默认）→ 免验证（不回归）
	setSecurityConfigForTest(t, pageConfig.SecurityAndRegistration{
		EnableEmailVerification: false,
		AllowedDomains:          []string{"tongji.edu.cn"},
	})
	user2, err := createUserFromOAuth(OAuthUserInfo{
		ID: "2", Login: "outside-user2", Provider: ProviderGitHub,
		VerifiedEmail: "bob@gmail.com", EmailVerified: true,
	})
	if err != nil {
		t.Fatalf("createUserFromOAuth() error = %v", err)
	}
	if user2.IsActivated != users.ActivationSuccess {
		t.Fatalf("unmatched domain with verification off should activate: %v", user2.IsActivated)
	}
}

// TestCreateUserFromOAuthNoEmailFollowsSwitch 无 verified 邮箱保持免验证激活
// （PR #167 review, blocking：无 verified 邮箱时无激活邮件可发，进入 pending
// 将形成无恢复路径的死账号；保持旧行为免验证）。
func TestCreateUserFromOAuthNoEmailFollowsSwitch(t *testing.T) {
	setupOAuthTestDB(t)

	// 开关关（默认）→ 免验证（不回归）
	setSecurityConfigForTest(t, pageConfig.SecurityAndRegistration{
		EnableEmailVerification: false,
		AllowedDomains:          []string{"tongji.edu.cn"},
	})
	user, err := createUserFromOAuth(OAuthUserInfo{
		ID: "1", Login: "no-email", Provider: ProviderGitHub,
		EmailVerified: false,
	})
	if err != nil {
		t.Fatalf("createUserFromOAuth() error = %v", err)
	}
	if user.IsActivated != users.ActivationSuccess {
		t.Fatalf("no email with verification off should activate: %v", user.IsActivated)
	}

	// 开关开 + 无 verified 邮箱 → 仍免验证激活（不死账号）
	setSecurityConfigForTest(t, pageConfig.SecurityAndRegistration{
		EnableEmailVerification: true,
		AllowedDomains:          []string{"tongji.edu.cn"},
	})
	user2, err := createUserFromOAuth(OAuthUserInfo{
		ID: "2", Login: "no-email2", Provider: ProviderGitHub,
		EmailVerified: false,
	})
	if err != nil {
		t.Fatalf("createUserFromOAuth() error = %v", err)
	}
	if user2.IsActivated != users.ActivationSuccess {
		t.Fatalf("no verified email with verification on should still activate (no dead account): %v", user2.IsActivated)
	}

	// 无 verified 邮箱时不降级存 goth 公开邮箱（PR #167 review, medium：
	// 未验证邮箱经 OIDC 会被推导为 email_verified=true，造成信任越界）。
	user3, err := createUserFromOAuth(OAuthUserInfo{
		ID: "3", Login: "public-only", Provider: ProviderGitHub,
		Email:         "public@example.com", // goth 公开邮箱，未验证
		EmailVerified: false,
	})
	if err != nil {
		t.Fatalf("createUserFromOAuth() error = %v", err)
	}
	if user3.Email != "" {
		t.Fatalf("unverified goth public email should NOT be stored, got %q", user3.Email)
	}
}

// TestBindOAuthByTrustedEmailBindsExisting 信任域名内 verified 邮箱已有账号 → 直接绑定。
func TestBindOAuthByTrustedEmailBindsExisting(t *testing.T) {
	conn := setupOAuthTestDB(t)
	setSecurityConfigForTest(t, pageConfig.SecurityAndRegistration{
		AllowedDomains: []string{"tongji.edu.cn"},
	})

	existing := users.MakeUser("prof-zhang", "password", "zhang@tongji.edu.cn")
	if err := users.Create(existing); err != nil {
		t.Fatalf("create existing user: %v", err)
	}

	user, err := ProcessOAuthCallback(verifiedGithubUser(t, "uid-bind", "prof-zhang", "zhang@tongji.edu.cn"))
	if err != nil {
		t.Fatalf("ProcessOAuthCallback() error = %v", err)
	}
	if user.Id != existing.Id {
		t.Fatalf("bound user id = %d, want %d", user.Id, existing.Id)
	}
	// OAuth 关联已创建
	entity := userOAuth.GetByProviderAndUID(ProviderGitHub, "uid-bind")
	if entity == nil || entity.UserId != existing.Id {
		t.Fatalf("oauth binding = %+v, want user %d", entity, existing.Id)
	}
	// 未创建新账号
	var count int64
	conn.Model(&users.EntityComplete{}).Count(&count)
	if count != 1 {
		t.Fatalf("user count = %d, want 1 (no duplicate registration)", count)
	}
}

// TestBindOAuthByTrustedEmailSkipsUnmatchedDomain 信任域名外 verified 邮箱 → 不绑定，走注册。
func TestBindOAuthByTrustedEmailSkipsUnmatchedDomain(t *testing.T) {
	setupOAuthTestDB(t)
	setSecurityConfigForTest(t, pageConfig.SecurityAndRegistration{
		AllowedDomains: []string{"tongji.edu.cn"},
	})

	existing := users.MakeUser("prof-zhang", "password", "zhang@tongji.edu.cn")
	if err := users.Create(existing); err != nil {
		t.Fatalf("create existing user: %v", err)
	}

	// 同邮箱但域名未命中 → 不绑定
	user, err := ProcessOAuthCallback(verifiedGithubUser(t, "uid-nobind", "someone", "someone@outlook.com"))
	if err != nil {
		t.Fatalf("ProcessOAuthCallback() error = %v", err)
	}
	if user.Id == existing.Id {
		t.Fatalf("unmatched domain should not bind existing user")
	}
	if userOAuth.GetByProviderAndUID(ProviderGitHub, "uid-nobind") == nil {
		t.Fatal("new user oauth binding missing")
	}
}

// TestBindOAuthByTrustedEmailRejectsFrozen 冻结账号即使 verified 邮箱命中也不能绑定。
func TestBindOAuthByTrustedEmailRejectsFrozen(t *testing.T) {
	setupOAuthTestDB(t)
	setSecurityConfigForTest(t, pageConfig.SecurityAndRegistration{
		AllowedDomains: []string{"tongji.edu.cn"},
	})

	existing := users.MakeUser("frozen-prof", "password", "frozen@tongji.edu.cn")
	if err := users.Create(existing); err != nil {
		t.Fatalf("create user: %v", err)
	}
	existing.IsFrozen = users.StatusFrozen
	if err := users.Save(existing); err != nil {
		t.Fatalf("freeze user: %v", err)
	}

	_, err := ProcessOAuthCallback(verifiedGithubUser(t, "uid-frozen", "frozen-prof", "frozen@tongji.edu.cn"))
	if err != ErrAccountFrozen {
		t.Fatalf("ProcessOAuthCallback() error = %v, want ErrAccountFrozen", err)
	}
}

// TestFetchGitHubVerifiedEmailParsing 验证 GitHub API 响应解析：取 verified+primary，
// 无 primary 时取任一 verified。
func TestFetchGitHubVerifiedEmailParsing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{"email": "secondary@tongji.edu.cn", "primary": false, "verified": true},
			{"email": "primary@tongji.edu.cn", "primary": true, "verified": true},
			{"email": "unverified@example.com", "primary": false, "verified": false},
		})
	}))
	defer server.Close()
	oldURL := gitHubEmailAPIURL
	gitHubEmailAPIURL = server.URL
	defer func() { gitHubEmailAPIURL = oldURL }()

	if got := fetchGitHubVerifiedEmail("token"); got != "primary@tongji.edu.cn" {
		t.Fatalf("fetchGitHubVerifiedEmail() = %q, want primary@tongji.edu.cn", got)
	}

	// 无 primary：取任一 verified
	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{"email": "only@tongji.edu.cn", "primary": false, "verified": true},
		})
	}))
	defer server2.Close()
	gitHubEmailAPIURL = server2.URL
	if got := fetchGitHubVerifiedEmail("token"); got != "only@tongji.edu.cn" {
		t.Fatalf("fetchGitHubVerifiedEmail() fallback = %q, want only@tongji.edu.cn", got)
	}
}

// TestProcessOAuthCallbackNoAccessToken 无 token（非 GitHub provider）降级旧行为：免验证。
func TestProcessOAuthCallbackNoAccessToken(t *testing.T) {
	setupOAuthTestDB(t)
	setSecurityConfigForTest(t, pageConfig.SecurityAndRegistration{
		EnableEmailVerification: true,
		AllowedDomains:          []string{"tongji.edu.cn"},
	})

	// 无 token、无 verified 邮箱的 GitHub 用户：开关开 → pending（不回归：开关关则激活）
	setSecurityConfigForTest(t, pageConfig.SecurityAndRegistration{
		EnableEmailVerification: false,
		AllowedDomains:          []string{"tongji.edu.cn"},
	})
	user, err := ProcessOAuthCallback(goth.User{
		Provider: ProviderGitHub,
		UserID:   "uid-notoken",
		NickName: "notoken",
	})
	if err != nil {
		t.Fatalf("ProcessOAuthCallback() error = %v", err)
	}
	if user.IsActivated != users.ActivationSuccess {
		t.Fatalf("no token + verification off should activate, got %v", user.IsActivated)
	}
	if user.Email != "" {
		t.Fatalf("no verified email should not store email, got %q", user.Email)
	}
}

// TestEmailInTrustedDomainsCaseInsensitive 域名匹配大小写不敏感（PR #167 review）：
// 与注册白名单 ValidateEmailDomain 语义统一（均 EqualFold）。
func TestEmailInTrustedDomainsCaseInsensitive(t *testing.T) {
	setSecurityConfigForTest(t, pageConfig.SecurityAndRegistration{
		AllowedDomains: []string{"Tongji.Edu.Cn"},
	})

	if !emailInTrustedDomains("alice@tongji.edu.cn") {
		t.Fatal("lowercase domain should match mixed-case allowlist entry")
	}
	if !emailInTrustedDomains("alice@TONGJI.EDU.CN") {
		t.Fatal("uppercase domain should match mixed-case allowlist entry")
	}
	if emailInTrustedDomains("alice@tongji.edu.cn.evil.com") {
		t.Fatal("subdomain suffix must not match (no suffix relaxation)")
	}
	if emailInTrustedDomains("not-an-email") {
		t.Fatal("malformed email must not match")
	}
}
