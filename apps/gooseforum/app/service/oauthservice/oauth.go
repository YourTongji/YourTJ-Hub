package oauthservice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"path"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/randopt"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/sessionstore"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/filemodel/filedata"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/moderationLog"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/emailactivationservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/eventhandlers"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/moderationservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/userservice"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/eventbus"
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

// ErrOAuthEmailUnverified 表示 OAuth provider 未提供可用于站点验证的邮箱。
var ErrOAuthEmailUnverified = errors.New("OAuth 邮箱未验证或不可用")

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
	gothic.BeginAuthHandler(w, r)
}

// CompleteOAuthUserAuth serializes provider lookup and user fetching with
// runtime provider refresh.
func CompleteOAuthUserAuth(w http.ResponseWriter, r *http.Request) (goth.User, error) {
	oauthProviderMu.RLock()
	defer oauthProviderMu.RUnlock()
	return gothic.CompleteUserAuth(w, r)
}

// initGitHubProvider returns a GitHub provider when configured.
func initGitHubProvider() goth.Provider {
	return initGitHubProviderWithCredentials(oauthCredentials{
		clientID:     preferences.GetString("github.client_id", ""),
		clientSecret: preferences.GetString("github.client_secret", ""),
	})
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

// ProcessOAuthCallback logs in an existing OAuth user or creates a new one.
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

	newUser, err := createUserFromOAuth(userInfo)
	if err != nil {
		return nil, err
	}

	eventbus.Publish(context.Background(), &eventhandlers.UserSignUpEvent{
		UserId:   newUser.Id,
		Username: newUser.Username,
	})

	err = createOAuthRecord(newUser.Id, userInfo)
	if err != nil {
		return nil, err
	}

	return newUser, nil
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
		if verified := fetchGitHubVerifiedEmail(gothUser.AccessToken); verified != "" {
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

// maxOAuthUsernameAttempts 限制 OAuth 建号用户名退避（重名/命中名单）尝试次数。
const maxOAuthUsernameAttempts = 100

// usernameReservedOrBanned 报告候选用户名是否命中保留/禁用名单（注册入口同规则）。
func usernameReservedOrBanned(username string, cfg pageConfig.SecurityAndRegistration) bool {
	code, _ := moderationservice.CheckUsernameAllowedWithConfig(username, cfg)
	return code != ""
}

// createUserFromOAuth creates a local account from OAuth user data.
// 激活策略（issue #155）：
//   - verified 邮箱命中信任域名（allowedDomains 空 = 全信任）→ needValid=false 直接激活；
//   - verified 邮箱未命中信任域名 → EnableEmailVerification 开启时进入
//     ActivationPending 并补发激活邮件；
//   - 未获取到可用 verified 邮箱 → EnableEmailVerification 开启时拒绝 OAuth 建号，
//     关闭时保持免验证兼容行为。
func createUserFromOAuth(userInfo OAuthUserInfo) (*users.EntityComplete, error) {
	securityConfig := hotdataserve.GetSecuritySettingsConfigCache()
	email := strings.ToLower(strings.TrimSpace(userInfo.VerifiedEmail))
	hasVerifiedEmail := userInfo.EmailVerified && email != ""
	if securityConfig.EnableEmailVerification && !hasVerifiedEmail {
		// 没有可发激活邮件的可信地址时直接拒绝，不能将未验证的 OAuth 身份
		// 当作已激活账号放行，也不能创建无法恢复的 pending 账号。
		return nil, ErrOAuthEmailUnverified
	}

	username := userInfo.Login
	originalUsername := username
	counter := 1
	// 候选名必须既不与存量用户重名，也不命中保留/禁用名单（社交登录对
	// 保留/禁用名做退避建号而非拒绝登录，最小惊讶）。退避候选同样过名单，
	// 上限 100 防死循环（超限属异常，报错回滚）。
	for users.ExistUsername(username) || usernameReservedOrBanned(username, securityConfig) {
		username = fmt.Sprintf("%s_%d", originalUsername, counter)
		counter++
		if counter > maxOAuthUsernameAttempts {
			return nil, fmt.Errorf("无法为 OAuth 登录名 %q 分配可用用户名（重名或命中保留/禁用名单）", originalUsername)
		}
	}

	// 存储邮箱：只存 provider 已确认真实的邮箱（GitHub verified 邮箱或 Google
	// verified_email=true 的邮箱）。
	// 无 verified 邮箱时保持存 ""（与旧行为一致），不降级存 goth 公开邮箱——
	// 未验证邮箱若被 OIDC userinfo 推导为 verified_email=true 会造成信任越界
	// （PR #167 review, medium）。

	// 信任判定：仅 verified 邮箱命中信任域名才免验证。
	trusted := hasVerifiedEmail && emailInTrustedDomains(email)
	needValid := !trusted && securityConfig.EnableEmailVerification

	userEntity, err := userservice.CreateUser(username, randopt.RandomString(32), email, needValid)
	if err != nil {
		return nil, fmt.Errorf("创建用户失败: %w", err)
	}

	// 需要邮箱激活（开关开且未命中信任域名、且有 verified 邮箱）：
	// 补发激活邮件（OAuth 路径原本不发）。needValid=true 时 email 必非空。
	if needValid {
		if err := emailactivationservice.SendActivationEmail(userEntity); err != nil {
			slog.Warn("OAuth 用户激活邮件入队失败", "userId", userEntity.Id, "email", email, "err", err)
		} else {
			slog.Info("OAuth 用户激活邮件已入队", "userId", userEntity.Id, "email", email)
		}
	}

	if userInfo.AvatarURL != "" {
		localAvatarPath, err := downloadAndSaveAvatar(userEntity.Id, userInfo.AvatarURL)
		if err != nil {
			slog.Warn("下载头像失败，使用默认头像", "error", err, "avatarURL", userInfo.AvatarURL)
			userEntity.AvatarUrl = users.RandAvatarUrl()
		} else {
			userEntity.AvatarUrl = localAvatarPath
		}
	} else {
		userEntity.AvatarUrl = users.RandAvatarUrl()
	}

	userEntity.Nickname = username
	userEntity.Bio = userInfo.Bio
	userEntity.Website = userInfo.Blog
	// 第三方资料自由文本（GitHub bio/blog）与 EditUserInfo 同规则过内容敏感词检查：
	// 命中即清空对应字段并写审核日志（subject=user_profile）。社交登录建号不应被
	// 第三方简介文本阻断（最小惊讶），但落库资料必须与站内资料拦截口径一致，否则
	// GitHub 简介可绕过 /api/set-user-info 的敏感词拦截直接进入资料。
	if hit, word := moderationservice.CheckContentAllowed(userEntity.Bio); hit {
		moderationservice.SensitiveContentBlocked(userEntity.Id, moderationLog.SubjectUserProfile, userEntity.Id, word, oauthProfileExcerpt(userEntity.Bio))
		slog.Warn("OAuth 导入 bio 命中敏感词已清空", "provider", userInfo.Provider, "login", username, "word", word)
		userEntity.Bio = ""
	}
	if hit, word := moderationservice.CheckContentAllowed(userEntity.Website); hit {
		moderationservice.SensitiveContentBlocked(userEntity.Id, moderationLog.SubjectUserProfile, userEntity.Id, word, oauthProfileExcerpt(userEntity.Website))
		slog.Warn("OAuth 导入 blog 命中敏感词已清空", "provider", userInfo.Provider, "login", username, "word", word)
		userEntity.Website = ""
	}
	if err := userservice.SaveUser(userEntity); err != nil {
		return nil, err
	}

	return userEntity, nil
}

// oauthProfileExcerpt 截断 OAuth 资料字段为审核日志摘要（100 个 rune，rune 边界安全）。
func oauthProfileExcerpt(s string) string {
	const maxRunes = 100
	n := 0
	for i := range s {
		if n >= maxRunes {
			return s[:i]
		}
		n++
	}
	return s
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

// downloadAndSaveAvatar stores an external OAuth avatar locally.
func downloadAndSaveAvatar(userID uint64, avatarURL string) (string, error) {
	if avatarURL == "" {
		return "", nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, avatarURL, nil)
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %w", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("下载头像失败: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("下载头像失败，状态码: %d", resp.StatusCode)
	}

	const maxFileSize = 2 * 1024 * 1024
	limitedReader := io.LimitReader(resp.Body, maxFileSize+1)

	avatarData, err := io.ReadAll(limitedReader)
	if err != nil {
		return "", fmt.Errorf("读取头像数据失败: %w", err)
	}

	if len(avatarData) > maxFileSize {
		return "", errors.New("头像文件过大，最大允许2MB")
	}

	filename := "avatar"
	if urlPath := resp.Request.URL.Path; urlPath != "" {
		ext := path.Ext(urlPath)
		if ext != "" {
			filename = "avatar" + ext
		} else {
			contentType := resp.Header.Get("Content-Type")
			switch {
			case strings.Contains(contentType, "jpeg"):
				filename = "avatar.jpg"
			case strings.Contains(contentType, "png"):
				filename = "avatar.png"
			case strings.Contains(contentType, "gif"):
				filename = "avatar.gif"
			case strings.Contains(contentType, "webp"):
				filename = "avatar.webp"
			default:
				filename = "avatar.jpg"
			}
		}
	}

	fileEntity, err := filedata.SaveAvatar(userID, avatarData, filename)
	if err != nil {
		return "", fmt.Errorf("保存头像失败: %w", err)
	}

	return fileEntity.Name, nil
}
