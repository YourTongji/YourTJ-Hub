package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
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

// enableEmailVerificationForTest 开启全站邮箱验证（SecuritySettings 页配置）。
// OAuth 未验证邮箱拒绝路径依赖该开关；cleanup 删除配置并清缓存还原默认。
func enableEmailVerificationForTest(t *testing.T) {
	t.Helper()
	conn := db.Connect()
	if err := conn.AutoMigrate(&pageConfig.Entity{}); err != nil {
		t.Fatalf("migrate page_config: %v", err)
	}
	encoded, err := json.Marshal(pageConfig.SecurityAndRegistration{EnableEmailVerification: true})
	if err != nil {
		t.Fatalf("encode security config: %v", err)
	}
	entity := pageConfig.Entity{PageType: pageConfig.SecuritySettings, Config: string(encoded)}
	if err := conn.Where("page_type = ?", pageConfig.SecuritySettings).Assign(entity).FirstOrCreate(&entity).Error; err != nil {
		t.Fatalf("save security config: %v", err)
	}
	hotdataserve.ClearSecuritySettingsConfigCache()
	t.Cleanup(func() {
		conn.Where("page_type = ?", pageConfig.SecuritySettings).Delete(&pageConfig.Entity{})
		hotdataserve.ClearSecuritySettingsConfigCache()
	})
}

// TestOAuthCallbackLoginRejectsUnverifiedEmail 守卫 controller 拒绝分支：全站邮箱验证开启时，
// Google OAuth 未提供 verified_email=true 邮箱的注册回调必须返回 403 +
// MessageAuthEmailUnverified，不创建账号、不写 OAuth 绑定、不发会话 Cookie。
func TestOAuthCallbackLoginRejectsUnverifiedEmail(t *testing.T) {
	setupOAuthCallbackTestDB(t)
	enableEmailVerificationForTest(t)

	stubGothUser(t, goth.User{
		Provider: oauthservice.ProviderGoogle,
		UserID:   "google-uid-unverified",
		NickName: "oauthunverified",
		Email:    "attacker@gmail.com",
		RawData:  map[string]any{"verified_email": false},
	})

	recorder, c := oauthCallbackRequest(t)
	// stub 接管 goth 后 provider 仅用于 query 构造；保持请求与模拟的 Google 身份一致。
	c.Params = gin.Params{{Key: "provider", Value: oauthservice.ProviderGoogle}}
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
	if payload.Props.MessageCode != component.MessageAuthEmailUnverified {
		t.Fatalf("messageCode = %q, want %q", payload.Props.MessageCode, component.MessageAuthEmailUnverified)
	}
	if hasAccessTokenCookie(recorder) {
		t.Fatal("rejected OAuth registration must not receive an access_token cookie")
	}
	if users.ExistUsername("oauthunverified") {
		t.Fatal("unverified OAuth user was created despite email verification being enabled")
	}
	if binding := userOAuth.GetByProviderAndUID(oauthservice.ProviderGoogle, "google-uid-unverified"); binding != nil {
		t.Fatalf("unverified OAuth identity was bound despite rejection: %#v", binding)
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
