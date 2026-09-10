package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/sessionstore"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userOAuth"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userSessions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/oauthservice"
	"github.com/gin-gonic/gin"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
)

// setupOAuthCallbackTestDB 迁移 OAuth callback 路径涉及的 model。
func setupOAuthCallbackTestDB(t *testing.T) {
	t.Helper()
	conn := db.Connect()
	for _, model := range []any{
		&users.EntityComplete{},
		&userOAuth.Entity{},
		&userSessions.Entity{},
	} {
		if err := conn.AutoMigrate(model); err != nil {
			t.Fatalf("migrate %T: %v", model, err)
		}
	}
}

// stubGothUser 替换 gothic.CompleteUserAuth，模拟第三方 OAuth 回调返回指定用户。
// 全局变量替换是 goth 的标准测试钩子，测试间必须串行（本包无 t.Parallel）。
func stubGothUser(t *testing.T, user goth.User) {
	t.Helper()
	original := gothic.CompleteUserAuth
	gothic.CompleteUserAuth = func(_ http.ResponseWriter, _ *http.Request) (goth.User, error) {
		return user, nil
	}
	t.Cleanup(func() { gothic.CompleteUserAuth = original })
}

// oauthCallbackRequest 构造 GitHub callback 请求（页面模式，错误页返回 JSON）。
func oauthCallbackRequest(t *testing.T) (*httptest.ResponseRecorder, *gin.Context) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Params = gin.Params{{Key: "provider", Value: oauthservice.ProviderGitHub}}
	c.Request = httptest.NewRequest(http.MethodGet, "/api/auth/github/callback", nil)
	c.Request.Header.Set("X-Goose-Page", "true")
	return recorder, c
}

// setSecurityConfigForCallbackTest 为回调测试注入 security 配置并清缓存（issue #531
// 白名单×开关矩阵依赖 allowedDomains / EnableEmailVerification 两种组合）。
func setSecurityConfigForCallbackTest(t *testing.T, config pageConfig.SecurityAndRegistration) {
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

func countSessions(t *testing.T, userID uint64) int64 {
	t.Helper()
	var count int64
	if err := db.Connect().Model(&userSessions.Entity{}).Where("user_id = ?", userID).Count(&count).Error; err != nil {
		t.Fatalf("count sessions: %v", err)
	}
	return count
}

func hasAccessTokenCookie(recorder *httptest.ResponseRecorder) bool {
	for _, cookie := range recorder.Result().Cookies() {
		if cookie.Name == "access_token" && cookie.Value != "" {
			return true
		}
	}
	return false
}

// TestOAuthCallbackLoginRejectsFrozenUser 复现 issue #130：已绑定 GitHub 的冻结账号
// 通过 OAuth callback 登录时，必须返回 403 + MessageOAuthAccountFrozen，
// 不创建 session、不设置认证 Cookie、不重定向。
func TestOAuthCallbackLoginRejectsFrozenUser(t *testing.T) {
	setupOAuthCallbackTestDB(t)

	user := &users.EntityComplete{
		Username:    "oauthfrozen",
		Email:       "oauthfrozen@example.com",
		IsFrozen:    users.StatusFrozen,
		IsActivated: users.ActivationSuccess,
	}
	if err := users.Create(user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := userOAuth.Create(&userOAuth.Entity{
		UserId:      user.Id,
		Provider:    oauthservice.ProviderGitHub,
		ProviderUid: "gh-uid-frozen",
	}); err != nil {
		t.Fatalf("create oauth binding: %v", err)
	}

	stubGothUser(t, goth.User{
		Provider: oauthservice.ProviderGitHub,
		UserID:   "gh-uid-frozen",
		NickName: "oauthfrozen",
		Email:    "oauthfrozen@example.com",
	})

	recorder, c := oauthCallbackRequest(t)
	ProviderCallback(c)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403 (body: %s)", recorder.Code, recorder.Body.String())
	}
	var payload struct {
		Props struct {
			MessageCode component.MessageCode `json:"messageCode"`
		} `json:"props"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode error page: %v", err)
	}
	if payload.Props.MessageCode != component.MessageOAuthAccountFrozen {
		t.Fatalf("messageCode = %q, want %q", payload.Props.MessageCode, component.MessageOAuthAccountFrozen)
	}
	if hasAccessTokenCookie(recorder) {
		t.Fatal("frozen user must not receive an access_token cookie")
	}
	if count := countSessions(t, user.Id); count != 0 {
		t.Fatalf("frozen user session rows = %d, want 0", count)
	}
	if loc := recorder.Header().Get("Location"); loc != "" {
		t.Fatalf("unexpected redirect for frozen user: %q", loc)
	}
}

// TestOAuthCallbackLoginSuccessIssuesSession 守卫正常路径：非冻结用户 OAuth 登录
// 仍应创建 session 并设置认证 Cookie。
func TestOAuthCallbackLoginSuccessIssuesSession(t *testing.T) {
	setupOAuthCallbackTestDB(t)

	user := &users.EntityComplete{
		Username:    "oauthlogin",
		Email:       "oauthlogin@example.com",
		IsActivated: users.ActivationSuccess,
	}
	if err := users.Create(user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := userOAuth.Create(&userOAuth.Entity{
		UserId:      user.Id,
		Provider:    oauthservice.ProviderGitHub,
		ProviderUid: "gh-uid-login",
	}); err != nil {
		t.Fatalf("create oauth binding: %v", err)
	}

	stubGothUser(t, goth.User{
		Provider: oauthservice.ProviderGitHub,
		UserID:   "gh-uid-login",
		NickName: "oauthlogin",
		Email:    "oauthlogin@example.com",
	})

	recorder, c := oauthCallbackRequest(t)
	ProviderCallback(c)

	if recorder.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302 (body: %s)", recorder.Code, recorder.Body.String())
	}
	if loc := recorder.Header().Get("Location"); loc != "/" {
		t.Fatalf("redirect location = %q, want /", loc)
	}
	if !hasAccessTokenCookie(recorder) {
		t.Fatal("successful OAuth login must set access_token cookie")
	}
	if count := countSessions(t, user.Id); count != 1 {
		t.Fatalf("session rows = %d, want 1", count)
	}
}

// TestOAuthCallbackLoginRejectsUnverifiedEmail（issue #531 改写）：无 verified 邮箱的
// OAuth 回调不再按全站邮箱验证开关 403 拒绝，而是与「无本地账号」同口径：
// 302 跳转注册页携带 oauthNotice；不创建账号、不写 OAuth 绑定、不发会话 Cookie。
func TestOAuthCallbackLoginRejectsUnverifiedEmail(t *testing.T) {
	setupOAuthCallbackTestDB(t)

	stubGothUser(t, goth.User{
		Provider: oauthservice.ProviderGoogle,
		UserID:   "google-uid-unverified",
		NickName: "oauthunverified",
		Email:    "attacker@gmail.com",
		RawData:  map[string]any{"verified_email": false},
	})

	recorder, c := oauthCallbackRequest(t)
	c.Params = gin.Params{{Key: "provider", Value: oauthservice.ProviderGoogle}}
	ProviderCallback(c)

	if recorder.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302 (body: %s)", recorder.Code, recorder.Body.String())
	}
	if loc := recorder.Header().Get("Location"); !strings.HasPrefix(loc, "/login?register=true&oauthNotice=1") {
		t.Fatalf("redirect location = %q, want /login?register=true&oauthNotice=1 prefix", loc)
	}
	if hasAccessTokenCookie(recorder) {
		t.Fatal("rejected OAuth registration must not receive an access_token cookie")
	}
	if users.ExistUsername("oauthunverified") {
		t.Fatal("unverified OAuth user was created")
	}
	if binding := userOAuth.GetByProviderAndUID(oauthservice.ProviderGoogle, "google-uid-unverified"); binding != nil {
		t.Fatalf("unverified OAuth identity was bound: %#v", binding)
	}
}

// TestOAuthCallbackLoginWithoutLocalAccountRedirectsToRegister（issue #531 核心验收）：
// 纯新号 OAuth 回调（无既有绑定、verified 邮箱无同邮箱账号）不再建号，
// 302 跳转注册页并携带 oauthNotice；安全 redirect 透传；不创建账号、
// 不写 OAuth 绑定、不发会话 Cookie。
func TestOAuthCallbackLoginWithoutLocalAccountRedirectsToRegister(t *testing.T) {
	setupOAuthCallbackTestDB(t)

	stubGothUser(t, goth.User{
		Provider: oauthservice.ProviderGoogle,
		UserID:   "google-new",
		NickName: "brandnew",
		Email:    "brandnew@gmail.com",
		RawData:  map[string]any{"verified_email": true},
	})

	recorder, c := oauthCallbackRequest(t)
	c.Params = gin.Params{{Key: "provider", Value: oauthservice.ProviderGoogle}}
	c.Request.URL.RawQuery = "redirect=%2Ftopics%2F3"
	ProviderCallback(c)

	if recorder.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302 (body: %s)", recorder.Code, recorder.Body.String())
	}
	loc := recorder.Header().Get("Location")
	if !strings.HasPrefix(loc, "/login?register=true&oauthNotice=1") {
		t.Fatalf("redirect location = %q, want /login?register=true&oauthNotice=1 prefix", loc)
	}
	if !strings.Contains(loc, "redirect=%2Ftopics%2F3") {
		t.Fatalf("safe redirect not passed through: %q", loc)
	}
	if hasAccessTokenCookie(recorder) {
		t.Fatal("no-local-account callback must not set access_token cookie")
	}
	if users.ExistUsername("brandnew") {
		t.Fatal("OAuth callback created an account (issue #531 regression)")
	}
	if binding := userOAuth.GetByProviderAndUID(oauthservice.ProviderGoogle, "google-new"); binding != nil {
		t.Fatalf("OAuth identity was bound without a local account: %#v", binding)
	}
}

// TestOAuthCallbackLoginWithoutLocalAccountDropsUnsafeRedirect 不安全 redirect
// 参数静默丢弃，防开放重定向。
func TestOAuthCallbackLoginWithoutLocalAccountDropsUnsafeRedirect(t *testing.T) {
	setupOAuthCallbackTestDB(t)

	stubGothUser(t, goth.User{
		Provider: oauthservice.ProviderGoogle,
		UserID:   "google-new-2",
		NickName: "brandnew2",
		Email:    "brandnew2@gmail.com",
		RawData:  map[string]any{"verified_email": true},
	})

	recorder, c := oauthCallbackRequest(t)
	c.Params = gin.Params{{Key: "provider", Value: oauthservice.ProviderGoogle}}
	c.Request.URL.RawQuery = "redirect=" + url.QueryEscape("https://evil.com")
	ProviderCallback(c)

	if recorder.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302", recorder.Code)
	}
	loc := recorder.Header().Get("Location")
	if !strings.HasPrefix(loc, "/login?register=true&oauthNotice=1") {
		t.Fatalf("redirect location = %q, want /login?register=true&oauthNotice=1 prefix", loc)
	}
	if strings.Contains(loc, "redirect=") {
		t.Fatalf("unsafe redirect leaked into Location: %q", loc)
	}
}

// TestOAuthCallbackLoginWithoutLocalAccountIgnoresEmailVerificationSwitch（issue #531
// 验收矩阵）：注册白名单非空 × 全站邮箱验证开/关，站外域名新用户均不建号、
// 不发激活邮件、一律 302 跳注册页——issue 描述的两条绕过路径全部封死。
func TestOAuthCallbackLoginWithoutLocalAccountIgnoresEmailVerificationSwitch(t *testing.T) {
	for _, enableVerification := range []bool{false, true} {
		t.Run(fmt.Sprintf("verification=%v", enableVerification), func(t *testing.T) {
			setupOAuthCallbackTestDB(t)
			setSecurityConfigForCallbackTest(t, pageConfig.SecurityAndRegistration{
				EnableEmailVerification: enableVerification,
				AllowedDomains:          []string{"tongji.edu.cn"},
			})

			stubGothUser(t, goth.User{
				Provider: oauthservice.ProviderGoogle,
				UserID:   "google-outside",
				NickName: "outsider",
				Email:    "outsider@gmail.com",
				RawData:  map[string]any{"verified_email": true},
			})

			recorder, c := oauthCallbackRequest(t)
			c.Params = gin.Params{{Key: "provider", Value: oauthservice.ProviderGoogle}}
			ProviderCallback(c)

			if recorder.Code != http.StatusFound {
				t.Fatalf("verification=%v: status = %d, want 302 (body: %s)", enableVerification, recorder.Code, recorder.Body.String())
			}
			if loc := recorder.Header().Get("Location"); !strings.HasPrefix(loc, "/login?register=true&oauthNotice=1") {
				t.Fatalf("verification=%v: redirect location = %q", enableVerification, loc)
			}
			if users.ExistUsername("outsider") || users.ExistEmail("outsider@gmail.com") {
				t.Fatalf("verification=%v: account created despite allowlist (issue #531)", enableVerification)
			}
			if binding := userOAuth.GetByProviderAndUID(oauthservice.ProviderGoogle, "google-outside"); binding != nil {
				t.Fatalf("verification=%v: identity bound without local account", enableVerification)
			}
		})
	}
}

// TestOAuthCallbackLoginPendingUserIssuesSession 待激活 OAuth 用户登录口径
// （issue #427）：pending 账号 OAuth 登录同样签发会话（写权限在权限层由
// CheckWritableAccount 拦截，会话本身不授予写能力），用户借此在会话过期后
// 重新登录并继续激活恢复流程——与密码登录对 pending 用户的放行口径一致。
func TestOAuthCallbackLoginPendingUserIssuesSession(t *testing.T) {
	setupOAuthCallbackTestDB(t)

	user := &users.EntityComplete{
		Username:    "oauthpending",
		Email:       "oauthpending@example.com",
		IsActivated: users.ActivationPending,
	}
	if err := users.Create(user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := userOAuth.Create(&userOAuth.Entity{
		UserId:      user.Id,
		Provider:    oauthservice.ProviderGitHub,
		ProviderUid: "gh-uid-pending",
	}); err != nil {
		t.Fatalf("create oauth binding: %v", err)
	}

	stubGothUser(t, goth.User{
		Provider: oauthservice.ProviderGitHub,
		UserID:   "gh-uid-pending",
		NickName: "oauthpending",
		Email:    "oauthpending@example.com",
	})

	recorder, c := oauthCallbackRequest(t)
	ProviderCallback(c)

	if recorder.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302 (body: %s)", recorder.Code, recorder.Body.String())
	}
	if loc := recorder.Header().Get("Location"); loc != "/" {
		t.Fatalf("redirect location = %q, want /", loc)
	}
	if !hasAccessTokenCookie(recorder) {
		t.Fatal("pending OAuth login must set access_token cookie")
	}
	if count := countSessions(t, user.Id); count != 1 {
		t.Fatalf("pending user session rows = %d, want 1", count)
	}
}

// seedOIDCResumeCookie 按 oauthservice/oidc_resume.go 的会话契约伪造续跳
// cookie（cookie 名与 session 键为 oauthservice 侧同构测试覆盖的浏览器契约，
// controller 测试只关心 CompleteOAuthUserAuth 恢复出 continuation 的语义）。
func seedOIDCResumeCookie(t *testing.T, provider, state, target string) *http.Cookie {
	t.Helper()
	start := httptest.NewRequest(http.MethodGet, "/api/auth/"+provider+"?provider="+provider, nil)
	session, err := sessionstore.GetSession().New(start, "yourtj_oauth_resume")
	if err != nil {
		t.Fatalf("new resume session: %v", err)
	}
	session.Values["state"] = state
	session.Values["provider"] = provider
	session.Values["target"] = target
	session.Values["expires"] = time.Now().Add(10 * time.Minute).Unix()
	recorder := httptest.NewRecorder()
	if err := session.Save(start, recorder); err != nil {
		t.Fatalf("save resume session: %v", err)
	}
	return recorder.Result().Cookies()[0]
}

// TestOAuthCallbackLoginWithoutLocalAccountKeepsOIDCContinuation（PR #552 review P1）：
// 经内置 OIDC 桥接进来的纯新号，CompleteOAuthUserAuth 已从签名 cookie 恢复出
// /api/oauth/authorize/callback?id=… 续跳目标；provider 回调查询串只有 state/code，
// 注册跳转必须携带 continuation（注册完成回到桥接回调，移动端才能拿到授权码），
// 而不是读 callback 查询串的 redirect（实际流程里不存在而丢失续跳）。
func TestOAuthCallbackLoginWithoutLocalAccountKeepsOIDCContinuation(t *testing.T) {
	setupOAuthCallbackTestDB(t)

	stubGothUser(t, goth.User{
		Provider: oauthservice.ProviderGoogle,
		UserID:   "google-bridge",
		NickName: "bridgenew",
		Email:    "bridgenew@gmail.com",
		RawData:  map[string]any{"verified_email": true},
	})

	const bridgeTarget = "/api/oauth/authorize/callback?id=reg-1"
	const state = "resume-state-1"
	cookie := seedOIDCResumeCookie(t, oauthservice.ProviderGoogle, state, bridgeTarget)

	recorder, c := oauthCallbackRequest(t)
	c.Params = gin.Params{{Key: "provider", Value: oauthservice.ProviderGoogle}}
	c.Request.URL.RawQuery = "provider=google&state=" + url.QueryEscape(state)
	c.Request.AddCookie(cookie)
	ProviderCallback(c)

	if recorder.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302 (body: %s)", recorder.Code, recorder.Body.String())
	}
	loc := recorder.Header().Get("Location")
	if !strings.HasPrefix(loc, "/login?register=true&oauthNotice=1") {
		t.Fatalf("redirect location = %q, want /login?register=true&oauthNotice=1 prefix", loc)
	}
	if !strings.Contains(loc, "redirect="+url.QueryEscape(bridgeTarget)) {
		t.Fatalf("OIDC continuation lost in registration redirect: %q", loc)
	}
	if hasAccessTokenCookie(recorder) {
		t.Fatal("no-local-account callback must not set access_token cookie")
	}
	if users.ExistUsername("bridgenew") {
		t.Fatal("OAuth callback created an account")
	}
}
