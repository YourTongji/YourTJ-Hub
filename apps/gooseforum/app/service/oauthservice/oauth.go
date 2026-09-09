package oauthservice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/sessionstore"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userOAuth"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/github"
	"github.com/markbates/goth/providers/google"
	"github.com/samber/lo"
	"gorm.io/gorm"
)

const (
	ProviderGitHub   = "github"
	ProviderGoogle   = "google"
	ProviderFacebook = "facebook"
	ProviderTwitter  = "twitter"
)

// ErrAccountFrozen 表示账号被冻结，禁止通过 OAuth 重新获取论坛会话。
// controller 依据该 sentinel error 渲染 403 冻结错误页（与 OIDC exchange 的冻结语义一致）。
var ErrAccountFrozen = errors.New("账号已冻结，禁止 OAuth 登录")

// ErrOAuthNoLocalAccount 表示 OAuth 身份在站内没有对应账号（无既有绑定，
// verified 邮箱也无既有账号可绑定）。OAuth 回调不再承担建号（issue #531，
// 注册统一走 /api/register 单点白名单门禁），controller 层据此 302 跳转注册页。
var ErrOAuthNoLocalAccount = errors.New("OAuth 身份无对应本地账号")

type oauthCredentials struct {
	clientID     string
	clientSecret string
}

var (
	// oauthProviderMu 与 goth provider registry 的替换及请求查找保持同步。
	oauthProviderMu sync.RWMutex
	// startupOAuthCredentials 固定进程启动时读取的 OAuth 凭据；运行时只刷新回调地址。
	startupGitHubCredentials oauthCredentials
	startupGoogleCredentials oauthCredentials
	// googleProvider 保存当前已注册的 Google provider。
	googleProvider atomic.Pointer[google.Provider]
)

// InitOAuth configures available OAuth providers.
func InitOAuth() {
	oauthProviderMu.Lock()
	defer oauthProviderMu.Unlock()

	startupGitHubCredentials = oauthCredentials{
		clientID:     strings.TrimSpace(preferences.GetString("github.client_id", "")),
		clientSecret: strings.TrimSpace(preferences.GetString("github.client_secret", "")),
	}
	startupGoogleCredentials = oauthCredentials{
		clientID:     strings.TrimSpace(preferences.GetString("google.client_id", "")),
		clientSecret: strings.TrimSpace(preferences.GetString("google.client_secret", "")),
	}
	initOAuthProvidersLocked()
}

// RefreshOAuthProviders rebuilds providers with startup credentials and the
// current site callback URL. It deliberately does not reread client secrets.
func RefreshOAuthProviders() {
	oauthProviderMu.Lock()
	defer oauthProviderMu.Unlock()
	initOAuthProvidersLocked()
}

func initOAuthProvidersLocked() {
	gothic.Store = sessionstore.GetSession()

	var providers []goth.Provider
	var configuredGoogleProvider *google.Provider

	if provider := initGitHubProviderWithCredentials(startupGitHubCredentials); provider != nil {
		providers = append(providers, provider)
	}

	if provider := initGoogleProviderWithCredentials(startupGoogleCredentials); provider != nil {
		providers = append(providers, provider)
		configuredGoogleProvider = provider
	}

	// SiteUrl 可在管理后台修改；重建 provider 前先移除旧实例，避免继续使用过期回调地址。
	goth.ClearProviders()
	if len(providers) > 0 {
		goth.UseProviders(providers...)
		slog.Info("OAuth提供商初始化完成", "count", len(providers))
	} else {
		slog.Warn("未配置任何OAuth提供商")
	}
	googleProvider.Store(configuredGoogleProvider)
}

// BeginOAuthAuthHandler serializes provider lookup with runtime provider refresh.
func BeginOAuthAuthHandler(w http.ResponseWriter, r *http.Request) {
	oauthProviderMu.RLock()
	defer oauthProviderMu.RUnlock()
	if err := prepareOIDCResume(w, r); err != nil {
		http.Error(w, "Invalid OAuth continuation", http.StatusBadRequest)
		return
	}
	gothic.BeginAuthHandler(w, r)
}

// CompleteOAuthUserAuth serializes provider lookup and user fetching with
// runtime provider refresh. It returns the verified upstream user and the
// validated local continuation (empty for ordinary Web login/binding).
func CompleteOAuthUserAuth(w http.ResponseWriter, r *http.Request) (goth.User, string, error) {
	oauthProviderMu.RLock()
	defer oauthProviderMu.RUnlock()
	target, err := consumeOIDCResume(w, r)
	if err != nil {
		return goth.User{}, "", err
	}
	user, err := gothic.CompleteUserAuth(w, r)
	if err != nil {
		return goth.User{}, "", err
	}
	return user, target, nil
}

func initGitHubProviderWithCredentials(credentials oauthCredentials) goth.Provider {
	callbackURL := strings.TrimRight(strings.TrimSpace(hotdataserve.GetSiteSettingsConfigCache().SiteUrl), "/") + "/api/auth/github/callback"
	clientID := strings.TrimSpace(credentials.clientID)
	clientSecret := strings.TrimSpace(credentials.clientSecret)
	if clientID == "" || clientSecret == "" {
		slog.Warn("GitHub OAuth配置缺失，跳过初始化")
		return nil
	}

	slog.Info("GitHub OAuth提供商初始化完成")
	// user:email scope：允许调用 GET /user/emails 获取 verified primary 邮箱
	// （issue #155：只信 GitHub 已验证邮箱，公开邮箱字段不区分 verified 不可依赖）。
	return github.New(clientID, clientSecret, callbackURL, "user:email")
}

// IsGoogleOAuthConfigured reports whether Google OAuth has all values required
// to construct an absolute callback URL. It does not imply provider registration.
func IsGoogleOAuthConfigured() bool {
	clientID, clientSecret, callbackURL := googleOAuthConfig()
	return clientID != "" && clientSecret != "" && callbackURL != ""
}

// IsGoogleOAuthReady reports whether the running process has registered Google
// with its startup credentials and the current callback URL.
// RefreshOAuthProviders must be called after a runtime siteUrl change.
func IsGoogleOAuthReady() bool {
	oauthProviderMu.RLock()
	defer oauthProviderMu.RUnlock()

	callbackURL := googleOAuthCallbackURL()
	if callbackURL == "" {
		return false
	}

	provider := googleProvider.Load()
	return provider != nil &&
		provider.CallbackURL == callbackURL
}

func googleOAuthConfig() (clientID, clientSecret, callbackURL string) {
	clientID = strings.TrimSpace(preferences.GetString("google.client_id", ""))
	clientSecret = strings.TrimSpace(preferences.GetString("google.client_secret", ""))
	return clientID, clientSecret, googleOAuthCallbackURL()
}

func googleOAuthCallbackURL() string {
	siteURL := strings.TrimRight(strings.TrimSpace(hotdataserve.GetSiteSettingsConfigCache().SiteUrl), "/")
	parsedURL, err := url.Parse(siteURL)
	if err != nil || parsedURL.Host == "" || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") || parsedURL.RawQuery != "" || parsedURL.Fragment != "" {
		return ""
	}
	return siteURL + "/api/auth/google/callback"
}

// initGoogleProvider returns a Google provider when configured.
func initGoogleProvider() *google.Provider {
	return initGoogleProviderWithCredentials(oauthCredentials{
		clientID:     preferences.GetString("google.client_id", ""),
		clientSecret: preferences.GetString("google.client_secret", ""),
	})
}

func initGoogleProviderWithCredentials(credentials oauthCredentials) *google.Provider {
	clientID := strings.TrimSpace(credentials.clientID)
	clientSecret := strings.TrimSpace(credentials.clientSecret)
	callbackURL := googleOAuthCallbackURL()
	if clientID == "" || clientSecret == "" || callbackURL == "" {
		slog.Warn("Google OAuth配置缺失，跳过初始化")
		return nil
	}

	slog.Info("Google OAuth提供商初始化完成")
	return google.New(clientID, clientSecret, callbackURL, "openid", "email", "profile")
}

// OAuthUserInfo is the normalized user data from an OAuth provider.
type OAuthUserInfo struct {
	ID        string `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
	Bio       string `json:"bio"`
	Blog      string `json:"blog"`
	Location  string `json:"location"`
	Provider  string `json:"provider"`

	// VerifiedEmail 是 provider 已确认真实的邮箱（GitHub verified 邮箱或 Google
	// verified_email=true 的邮箱）。
	// EmailVerified 标记该邮箱来自可信来源（GitHub /user/emails 或 Google OIDC）。
	// issue #155：只信 verified 邮箱作为绑定/激活依据，goth 的 Email 字段
	// 可能是未验证的公开邮箱，不可直接用于信任决策。
	VerifiedEmail string `json:"verifiedEmail,omitempty"`
	EmailVerified bool   `json:"emailVerified,omitempty"`
}

// ProcessOAuthCallback 只处理「既有绑定登录」或「verified 邮箱绑定既有账号」，
// 不再创建新账号（issue #531）：注册入口统一收敛到 /api/register，注册域名
// 白名单（allowedDomains）由 ValidateEmailDomain 在注册单点把关，OAuth 侧
// 不再存在「未命中白名单仍建号 / 向站外域名补发激活邮件」的旁路。
// 无对应本地账号时返回 ErrOAuthNoLocalAccount，由 controller 跳转注册页。
func ProcessOAuthCallback(gothUser goth.User) (*users.EntityComplete, error) {
	userInfo := parseOAuthUserInfo(gothUser)

	existingOAuth := userOAuth.GetByProviderAndUID(userInfo.Provider, userInfo.ID)
	if existingOAuth != nil {
		user, err := users.Get(existingOAuth.UserId)
		if err != nil {
			return nil, fmt.Errorf("获取用户信息失败: %w", err)
		}
		// 冻结账号禁止通过 OAuth（goth）重新获取论坛会话。
		if user.IsFrozen == users.StatusFrozen {
			return nil, ErrAccountFrozen
		}
		// 机器人（Agent）账号禁止通过 OAuth（goth）登录。
		if user.IsBot() {
			return nil, fmt.Errorf("机器人账号不允许 OAuth 登录")
		}
		return &user, nil
	}

	// 绑定路径（issue #155）：verified 邮箱命中信任域名且该邮箱已有账号时，
	// 直接建立 OAuth 关联并登录该账号，不重复注册。
	if bound, err := bindOAuthByTrustedEmail(userInfo); bound != nil || err != nil {
		return bound, err
	}

	return nil, ErrOAuthNoLocalAccount
}

// bindOAuthByTrustedEmail 尝试把 OAuth 身份绑定到 verified 邮箱命中的已有账号。
// 仅当 email 命中信任域名（allowedDomains 空 = 全信任）且存在同邮箱账号时绑定；
// 返回 (nil, nil) 表示无账号可绑，走正常注册路径。
func bindOAuthByTrustedEmail(userInfo OAuthUserInfo) (*users.EntityComplete, error) {
	if !userInfo.EmailVerified || userInfo.VerifiedEmail == "" {
		return nil, nil
	}
	if !emailInTrustedDomains(userInfo.VerifiedEmail) {
		return nil, nil
	}

	user, err := users.GetByEmail(userInfo.VerifiedEmail)
	if errors.Is(err, gorm.ErrRecordNotFound) || (err == nil && user.Id == 0) {
		return nil, nil // 无同邮箱账号，走注册
	}
	if err != nil {
		return nil, fmt.Errorf("查询可信邮箱账号失败: %w", err)
	}

	// 与既有 OAuth 绑定路径一致的账号状态检查（issue #130 冻结语义保留）。
	if user.IsFrozen == users.StatusFrozen {
		return nil, ErrAccountFrozen
	}
	if user.IsBot() {
		return nil, fmt.Errorf("机器人账号不允许 OAuth 登录")
	}

	if err := createOAuthRecord(user.Id, userInfo); err != nil {
		return nil, err
	}
	slog.Info("OAuth 绑定到已存在账号（verified 邮箱命中信任域名）",
		"userId", user.Id, "provider", userInfo.Provider, "email", userInfo.VerifiedEmail)
	return &user, nil
}

// emailInTrustedDomains 判断邮箱域名是否命中信任域名列表。
// allowedDomains 为空 = 全信任（默认配置，行为与现状无条件信任一致）。
// 域名精确匹配（大小写不敏感），不做子域名放宽。
func emailInTrustedDomains(email string) bool {
	securityConfig := hotdataserve.GetSecuritySettingsConfigCache()
	if len(securityConfig.AllowedDomains) == 0 {
		return true
	}
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return false
	}
	domain := strings.ToLower(parts[1])
	for _, allowed := range securityConfig.AllowedDomains {
		if strings.EqualFold(strings.TrimSpace(allowed), domain) {
			return true
		}
	}
	return false
}

// parseOAuthUserInfo normalizes provider-specific user data.
// 对 GitHub 额外获取 verified primary 邮箱（issue #155），Google 则只信任
// userinfo 中 verified_email=true 的邮箱。
func parseOAuthUserInfo(gothUser goth.User) OAuthUserInfo {
	userInfo := OAuthUserInfo{
		ID:        gothUser.UserID,
		Login:     gothUser.NickName,
		Name:      gothUser.Name,
		Email:     gothUser.Email,
		AvatarURL: gothUser.AvatarURL,
		Provider:  gothUser.Provider,
	}

	if gothUser.RawData != nil {
		if bio, ok := gothUser.RawData["bio"].(string); ok {
			userInfo.Bio = bio
		}
		if blog, ok := gothUser.RawData["blog"].(string); ok {
			userInfo.Blog = blog
		}
		if location, ok := gothUser.RawData["location"].(string); ok {
			userInfo.Location = location
		}
		if login, ok := gothUser.RawData["login"].(string); ok && login != "" {
			userInfo.Login = login
		}
	}

	// GitHub：通过 /user/emails 获取 verified 邮箱。goth 的 Email 字段可能来自
	// 公开 profile（不保证 verified），不能直接作为信任依据。
	if userInfo.Provider == ProviderGitHub {
		if verified := fetchGitHubVerifiedEmailFn(gothUser.AccessToken); verified != "" {
			userInfo.VerifiedEmail = verified
			userInfo.EmailVerified = true
		}
	}

	// Google goth provider 的实际字段是 verified_email；同时兼容 OIDC 代理常见的
	// email_verified 别名及字符串/数字形式。只有断言为 true 且邮箱非空时，才进入
	// 可信邮箱绑定/激活路径。
	if userInfo.Provider == ProviderGoogle {
		verifiedFlag, ok := gothUser.RawData["verified_email"]
		if !ok {
			verifiedFlag, ok = gothUser.RawData["email_verified"]
		}
		if ok && oauthFlagIsTrue(verifiedFlag) {
			if verified := strings.ToLower(strings.TrimSpace(userInfo.Email)); verified != "" {
				userInfo.VerifiedEmail = verified
				userInfo.EmailVerified = true
			}
		}
	}

	return userInfo
}

// oauthFlagIsTrue accepts the boolean representation emitted by standard JSON
// decoding and the string/number variants used by some OIDC proxies and mocks.
func oauthFlagIsTrue(value any) bool {
	switch value := value.(type) {
	case bool:
		return value
	case string:
		normalized := strings.ToLower(strings.TrimSpace(value))
		return normalized == "true" || normalized == "1"
	case json.Number:
		return value == "1" || value == "1.0"
	case float64:
		return value == 1
	case float32:
		return value == 1
	case int:
		return value == 1
	case int64:
		return value == 1
	case uint:
		return value == 1
	case uint64:
		return value == 1
	default:
		return false
	}
}

// gitHubEmailAPIURL 为 GitHub 邮箱列表 API（var 便于测试覆盖）。
var gitHubEmailAPIURL = "https://api.github.com/user/emails"

// fetchGitHubVerifiedEmailFn 是 verified 邮箱拉取接缝（var 便于测试注入）：
// 生产路径走 fetchGitHubVerifiedEmail 的真实 HTTP 拉取；受限环境（禁止回环
// TCP，httptest server 不可达）的测试可直接注入内存实现。
var fetchGitHubVerifiedEmailFn = fetchGitHubVerifiedEmail

// parseGitHubEmails 从 /user/emails 响应中解析 verified 邮箱（纯函数）：
// 优先 verified && primary；无 primary 时退而取任一 verified 邮箱。
func parseGitHubEmails(list []gitHubEmailEntry) string {
	for _, e := range list {
		if e.Verified && e.Primary && e.Email != "" {
			return strings.ToLower(e.Email)
		}
	}
	for _, e := range list {
		if e.Verified && e.Email != "" {
			return strings.ToLower(e.Email)
		}
	}
	return ""
}

// gitHubEmailEntry 是 GET /user/emails 的返回项。
type gitHubEmailEntry struct {
	Email    string `json:"email"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}

// fetchGitHubVerifiedEmail 调用 GitHub API 获取用户邮箱列表，
// 返回 verified && primary 的邮箱；无 primary 时退而取任一 verified 邮箱。
// 失败（网络/401/无 verified 邮箱）返回空字符串，调用方按无邮箱处理。
func fetchGitHubVerifiedEmail(accessToken string) string {
	if accessToken == "" {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, gitHubEmailAPIURL, nil)
	if err != nil {
		slog.Warn("构造 GitHub 邮箱请求失败", "err", err)
		return ""
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		slog.Warn("获取 GitHub 邮箱列表失败", "err", err)
		return ""
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		slog.Warn("获取 GitHub 邮箱列表返回非 200", "status", resp.StatusCode)
		return ""
	}

	var list []gitHubEmailEntry
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		slog.Warn("解析 GitHub 邮箱列表失败", "err", err)
		return ""
	}
	return parseGitHubEmails(list)
}

// createOAuthRecord stores a provider account binding. Only the identity
// linkage (user/provider/provider_uid) is persisted; third-party OAuth tokens
// are never written to the database (Issue #131).
// 同 (user_id, provider) 已有绑定时不重复创建（幂等，PR #167 review minor）。
func createOAuthRecord(userID uint64, userInfo OAuthUserInfo) error {
	if existing := userOAuth.GetByUserIDAndProvider(userID, userInfo.Provider); existing != nil {
		return nil
	}
	oauthEntity := &userOAuth.Entity{
		UserId:      userID,
		Provider:    userInfo.Provider,
		ProviderUid: userInfo.ID,
	}
	return userOAuth.Create(oauthEntity)
}

// UnbindOAuth removes one OAuth binding after safety checks.
func UnbindOAuth(userID uint64, provider string) error {
	oauthEntity := userOAuth.GetByUserIDAndProvider(userID, provider)
	if oauthEntity == nil {
		return errors.New("OAuth绑定不存在")
	}

	if err := checkUnbindSafety(userID, provider); err != nil {
		return err
	}

	return userOAuth.Delete(oauthEntity.Id)
}

// checkUnbindSafety ensures the user keeps at least one login method.
func checkUnbindSafety(userID uint64, providerToUnbind string) error {
	user, err := users.Get(userID)
	if err != nil {
		return fmt.Errorf("获取用户信息失败: %w", err)
	}

	hasEmail := user.Email != ""

	bindings := GetUserOAuthBindings(userID)

	remainingBindings := lo.CountBy(lo.Keys(bindings), func(p string) bool {
		return p != providerToUnbind
	})

	if !hasEmail && remainingBindings == 0 {
		return errors.New("解绑失败：您必须至少保留一种登录方式（邮箱或其他OAuth绑定）")
	}

	return nil
}

// ProcessOAuthBind binds a provider account to an existing user.
func ProcessOAuthBind(userID uint64, gothUser goth.User) error {
	userInfo := parseOAuthUserInfo(gothUser)

	// 机器人（Agent）账号禁止绑定任何 OAuth 身份。
	if err := rejectBotUser(userID); err != nil {
		return err
	}

	existingOAuth := userOAuth.GetByProviderAndUID(userInfo.Provider, userInfo.ID)
	if existingOAuth != nil {
		if existingOAuth.UserId != userID {
			return errors.New("该OAuth账户已被其他用户绑定")
		}
		return nil
	}

	existingUserOAuth := userOAuth.GetByUserIDAndProvider(userID, userInfo.Provider)
	if existingUserOAuth != nil {
		return errors.New("您已绑定该平台账户")
	}

	return createOAuthRecord(userID, userInfo)
}

// rejectBotUser returns an error when the user is a bot (agent) persona.
func rejectBotUser(userID uint64) error {
	user, err := users.Get(userID)
	if err != nil {
		return fmt.Errorf("获取用户信息失败: %w", err)
	}
	if user.Id == 0 {
		return errors.New("获取用户信息失败")
	}
	if user.IsBot() {
		return errors.New("机器人账号不允许该操作")
	}
	return nil
}

// GetUserOAuthBindings returns active OAuth bindings keyed by provider.
func GetUserOAuthBindings(userID uint64) map[string]*userOAuth.Entity {
	providers := []string{ProviderGitHub, ProviderGoogle}
	return lo.PickBy(lo.Associate(providers, func(p string) (string, *userOAuth.Entity) {
		return p, userOAuth.GetByUserIDAndProvider(userID, p)
	}), func(_ string, v *userOAuth.Entity) bool {
		return v != nil
	})
}

// HasOAuthBinding reports whether the user can authenticate through an external provider.
func HasOAuthBinding(userID uint64) bool {
	return len(GetUserOAuthBindings(userID)) > 0
}
