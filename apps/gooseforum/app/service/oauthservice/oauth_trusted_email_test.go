package oauthservice

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userOAuth"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
	"github.com/markbates/goth"
)

// setSecurityConfigForTest 注入 security 配置并清缓存，返回还原函数。
func setSecurityConfigForTest(t *testing.T, config pageConfig.SecurityAndRegistration) {
	t.Helper()
	conn := db.Connect()
	if err := conn.AutoMigrate(&pageConfig.Entity{}); err != nil {
		t.Fatalf("migrate page_config: %v", err)
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

// stubGitHubEmailFetch 注入 verified 邮箱接缝（fetchGitHubVerifiedEmailFn）并
// 在测试结束后还原。不经回环 HTTP：受限环境（禁止回环 TCP）里 httptest
// server 不可达，接缝注入使测试在沙箱与 CI 下行为一致且意图确定。
func stubGitHubEmailFetch(t *testing.T, fetch func(token string) string) {
	t.Helper()
	oldFn := fetchGitHubVerifiedEmailFn
	fetchGitHubVerifiedEmailFn = fetch
	t.Cleanup(func() { fetchGitHubVerifiedEmailFn = oldFn })
}

// verifiedGithubUser 构造带 verified 邮箱的 GitHub goth 用户（接缝返回该邮箱，
// 与生产 fetchGitHubVerifiedEmail 同样做小写归一）。
func verifiedGithubUser(t *testing.T, uid, login, email string) goth.User {
	t.Helper()
	verified := strings.ToLower(email)
	stubGitHubEmailFetch(t, func(string) string { return verified })

	return goth.User{
		Provider:    ProviderGitHub,
		UserID:      uid,
		NickName:    login,
		AccessToken: "test-token",
	}
}

// unverifiedGithubUser 构造无 verified 邮箱的 GitHub 用户（接缝返回空）。
func unverifiedGithubUser(t *testing.T, uid, login string) goth.User {
	t.Helper()
	stubGitHubEmailFetch(t, func(string) string { return "" })

	return goth.User{
		Provider:    ProviderGitHub,
		UserID:      uid,
		NickName:    login,
		AccessToken: "test-token",
	}
}

// prepareNoSignupExpectations 迁移并清空 task_queue，使「无邮件入队」断言可靠。
func prepareNoSignupExpectations(t *testing.T) {
	t.Helper()
	conn := db.Connect()
	if err := conn.AutoMigrate(&taskQueue.Entity{}); err != nil {
		t.Fatalf("migrate task_queue: %v", err)
	}
	conn.Unscoped().Where("1 = 1").Delete(&taskQueue.Entity{})
}

// assertNoLocalAccountCreated（issue #531 验收核心）：回调后不产生 users 行、
// user_oauth 行、邮件队列任务。UserSignUpEvent 只在建号成功后发布且发布代码已
// 随建号路径整体移除，无建号即无事件（另有全仓引用检查兜底）。
func assertNoLocalAccountCreated(t *testing.T) {
	t.Helper()
	conn := db.Connect()
	var userCount int64
	if err := conn.Model(&users.EntityComplete{}).Count(&userCount).Error; err != nil {
		t.Fatalf("count users: %v", err)
	}
	if userCount != 0 {
		t.Fatalf("user rows = %d, want 0 (OAuth callback must not create accounts)", userCount)
	}
	var bindingCount int64
	if err := conn.Model(&userOAuth.Entity{}).Count(&bindingCount).Error; err != nil {
		t.Fatalf("count user_oauth: %v", err)
	}
	if bindingCount != 0 {
		t.Fatalf("user_oauth rows = %d, want 0", bindingCount)
	}
	var taskCount int64
	if err := conn.Model(&taskQueue.Entity{}).Count(&taskCount).Error; err != nil {
		t.Fatalf("count task_queue: %v", err)
	}
	if taskCount != 0 {
		t.Fatalf("task_queue rows = %d, want 0 (no activation email may be enqueued)", taskCount)
	}
}

// TestProcessOAuthCallbackNoLocalAccountTrustedEmail（issue #531）：
// 信任域名 verified 邮箱但站内无同邮箱账号 → 返回 ErrOAuthNoLocalAccount。
// 旧版在此路径直接建号（tongji.edu.cn 一键建号），现注册单点收敛到
// /api/register（allowedDomains 白名单门禁），OAuth 侧不建号、不发激活邮件。
func TestProcessOAuthCallbackNoLocalAccountTrustedEmail(t *testing.T) {
	setupOAuthTestDB(t)
	setSecurityConfigForTest(t, pageConfig.SecurityAndRegistration{
		AllowedDomains: []string{"tongji.edu.cn"},
	})
	prepareNoSignupExpectations(t)

	_, err := ProcessOAuthCallback(verifiedGithubUser(t, "uid-new-trusted", "alice", "alice@tongji.edu.cn"))
	if !errors.Is(err, ErrOAuthNoLocalAccount) {
		t.Fatalf("ProcessOAuthCallback() error = %v, want ErrOAuthNoLocalAccount", err)
	}
	assertNoLocalAccountCreated(t)
}

// TestProcessOAuthCallbackNoLocalAccountOutsideDomain（issue #531 验收矩阵）：
// allowedDomains 非空 × verified 邮箱未命中白名单 × EnableEmailVerification
// 开/关——两种配置下纯新号一律拒绝建号。旧版关闭开关时直接激活（白名单绕过）、
// 开启时向站外邮箱补发激活邮件后仍可激活，两条旁路均已消除。
func TestProcessOAuthCallbackNoLocalAccountOutsideDomain(t *testing.T) {
	for _, enableVerification := range []bool{false, true} {
		t.Run(fmt.Sprintf("verification=%v", enableVerification), func(t *testing.T) {
			setupOAuthTestDB(t)
			setSecurityConfigForTest(t, pageConfig.SecurityAndRegistration{
				EnableEmailVerification: enableVerification,
				AllowedDomains:          []string{"tongji.edu.cn"},
			})
			prepareNoSignupExpectations(t)

			_, err := ProcessOAuthCallback(verifiedGithubUser(t, "uid-outside", "bob", "bob@gmail.com"))
			if !errors.Is(err, ErrOAuthNoLocalAccount) {
				t.Fatalf("ProcessOAuthCallback() error = %v, want ErrOAuthNoLocalAccount", err)
			}
			assertNoLocalAccountCreated(t)
		})
	}
}

// TestProcessOAuthCallbackNoLocalAccountNoVerifiedEmail（issue #531 验收矩阵）：
// 无 verified 邮箱的 GitHub 身份 × 邮箱验证开关开/关 → 一律拒绝建号。
func TestProcessOAuthCallbackNoLocalAccountNoVerifiedEmail(t *testing.T) {
	for _, enableVerification := range []bool{false, true} {
		t.Run(fmt.Sprintf("verification=%v", enableVerification), func(t *testing.T) {
			setupOAuthTestDB(t)
			setSecurityConfigForTest(t, pageConfig.SecurityAndRegistration{
				EnableEmailVerification: enableVerification,
				AllowedDomains:          []string{"tongji.edu.cn"},
			})
			prepareNoSignupExpectations(t)

			_, err := ProcessOAuthCallback(unverifiedGithubUser(t, "uid-noemail", "noemail"))
			if !errors.Is(err, ErrOAuthNoLocalAccount) {
				t.Fatalf("ProcessOAuthCallback() error = %v, want ErrOAuthNoLocalAccount", err)
			}
			assertNoLocalAccountCreated(t)
		})
	}
}

// TestProcessOAuthCallbackNoLocalAccountGoogleNewUser Google 侧新号口径与 GitHub
// 一致：verified_email=true 的站外邮箱在白名单下同样不建号。
func TestProcessOAuthCallbackNoLocalAccountGoogleNewUser(t *testing.T) {
	setupOAuthTestDB(t)
	setSecurityConfigForTest(t, pageConfig.SecurityAndRegistration{
		EnableEmailVerification: false,
		AllowedDomains:          []string{"tongji.edu.cn"},
	})
	prepareNoSignupExpectations(t)

	_, err := ProcessOAuthCallback(goth.User{
		Provider: ProviderGoogle,
		UserID:   "google-new",
		NickName: "gnew",
		Email:    "gnew@gmail.com",
		RawData:  map[string]any{"verified_email": true},
	})
	if !errors.Is(err, ErrOAuthNoLocalAccount) {
		t.Fatalf("ProcessOAuthCallback() error = %v, want ErrOAuthNoLocalAccount", err)
	}
	assertNoLocalAccountCreated(t)
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

// TestBindOAuthByTrustedEmailPropagatesDatabaseError 数据库查询失败时不得降级为注册。
func TestBindOAuthByTrustedEmailPropagatesDatabaseError(t *testing.T) {
	conn := setupOAuthTestDB(t)
	setSecurityConfigForTest(t, pageConfig.SecurityAndRegistration{
		AllowedDomains: []string{"tongji.edu.cn"},
	})

	if bound, err := bindOAuthByTrustedEmail(OAuthUserInfo{
		VerifiedEmail: "missing@tongji.edu.cn",
		EmailVerified: true,
	}); bound != nil || err != nil {
		t.Fatalf("bindOAuthByTrustedEmail() for missing account = (%v, %v), want (nil, nil)", bound, err)
	}

	if err := conn.Exec(`ALTER TABLE users RENAME TO users_lookup_error`).Error; err != nil {
		t.Fatalf("rename users table: %v", err)
	}
	t.Cleanup(func() {
		if err := conn.Exec(`ALTER TABLE users_lookup_error RENAME TO users`).Error; err != nil {
			t.Errorf("restore users table: %v", err)
		}
	})

	_, err := bindOAuthByTrustedEmail(OAuthUserInfo{
		VerifiedEmail: "broken@tongji.edu.cn",
		EmailVerified: true,
	})
	if err == nil {
		t.Fatal("bindOAuthByTrustedEmail() error = nil, want database error")
	}
}

// TestBindOAuthByTrustedEmailSkipsUnmatchedDomainUnbound（issue #531 改写）：
// 同邮箱账号存在但域名未命中 → 不绑定；整个身份无本地账号 → 返回
// ErrOAuthNoLocalAccount（旧版在此降级为「另建一个同邮箱账号」，已删除）。
func TestBindOAuthByTrustedEmailSkipsUnmatchedDomainUnbound(t *testing.T) {
	setupOAuthTestDB(t)
	setSecurityConfigForTest(t, pageConfig.SecurityAndRegistration{
		AllowedDomains: []string{"tongji.edu.cn"},
	})

	existing := users.MakeUser("prof-zhang", "password", "zhang@tongji.edu.cn")
	if err := users.Create(existing); err != nil {
		t.Fatalf("create existing user: %v", err)
	}

	_, err := ProcessOAuthCallback(verifiedGithubUser(t, "uid-nobind", "someone", "someone@outlook.com"))
	if !errors.Is(err, ErrOAuthNoLocalAccount) {
		t.Fatalf("ProcessOAuthCallback() error = %v, want ErrOAuthNoLocalAccount", err)
	}
	if userOAuth.GetByProviderAndUID(ProviderGitHub, "uid-nobind") != nil {
		t.Fatal("unmatched-domain identity must not be bound to the existing account")
	}
	if users.ExistEmail("someone@outlook.com") {
		t.Fatal("duplicate-email account was created for unmatched domain (issue #531)")
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

// TestParseGitHubEmailsPrefersVerifiedPrimary 验证 /user/emails 响应解析：
// 取 verified+primary，无 primary 时取任一 verified。纯函数直测，无网络依赖
// （原 httptest 回环实现在禁止回环 TCP 的受限环境不可达，行为口径不变）。
func TestParseGitHubEmailsPrefersVerifiedPrimary(t *testing.T) {
	cases := []struct {
		name string
		list []gitHubEmailEntry
		want string
	}{
		{
			name: "verified primary wins",
			list: []gitHubEmailEntry{
				{Email: "secondary@tongji.edu.cn", Primary: false, Verified: true},
				{Email: "Primary@Tongji.edu.cn", Primary: true, Verified: true},
				{Email: "unverified@example.com", Primary: false, Verified: false},
			},
			want: "primary@tongji.edu.cn",
		},
		{
			name: "falls back to any verified",
			list: []gitHubEmailEntry{{Email: "only@tongji.edu.cn", Primary: false, Verified: true}},
			want: "only@tongji.edu.cn",
		},
		{
			name: "unverified ignored",
			list: []gitHubEmailEntry{{Email: "x@example.com", Primary: true, Verified: false}},
			want: "",
		},
		{
			name: "empty list",
			list: nil,
			want: "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := parseGitHubEmails(tc.list); got != tc.want {
				t.Fatalf("parseGitHubEmails() = %q, want %q", got, tc.want)
			}
		})
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
