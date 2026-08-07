// Package oidcservice implements the Casdoor OIDC login and binding flow
// (PKCE + state/nonce), independent from the goth-based GitHub/Google OAuth.
package oidcservice

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-gonic/gin"
	"github.com/leancodebox/GooseForum/app/bundles/algorithm"
	"github.com/leancodebox/GooseForum/app/bundles/eventbus"
	"github.com/leancodebox/GooseForum/app/bundles/preferences"
	"github.com/leancodebox/GooseForum/app/bundles/sessionstore"
	"github.com/leancodebox/GooseForum/app/models/forum/userOAuth"
	"github.com/leancodebox/GooseForum/app/models/forum/users"
	"github.com/leancodebox/GooseForum/app/models/hotdataserve"
	"github.com/leancodebox/GooseForum/app/service/eventhandlers"
	"github.com/leancodebox/GooseForum/app/service/userservice"
	"golang.org/x/oauth2"
)

const (
	// ProviderCasdoor is the provider key stored in user_o_auth rows.
	ProviderCasdoor = "casdoor"

	sessionName        = "oidc"
	sessionKeyState    = "state"
	sessionKeyNonce    = "nonce"
	sessionKeyVerifier = "verifier"

	callbackPath = "/api/auth/oidc/callback"
)

var (
	// ErrOIDCNotConfigured is returned when casdoor settings are missing.
	ErrOIDCNotConfigured = errors.New("oidc: casdoor 配置缺失")
	// ErrStateMismatch is returned when the callback state differs from the session state.
	ErrStateMismatch = errors.New("oidc: state 校验失败")
	// ErrNonNumericSub is returned when the id_token sub is not a numeric ID.
	ErrNonNumericSub = errors.New("oidc: sub 必须为数字ID")
	// ErrNonceMismatch is returned when the id_token nonce differs from the session nonce.
	ErrNonceMismatch = errors.New("oidc: nonce 校验失败")
)

// CallbackResult carries the verified OIDC identity claims.
type CallbackResult struct {
	Sub      uint64
	Username string
	Email    string
}

type oidcSettings struct {
	endpoint     string
	clientID     string
	clientSecret string
}

func loadSettings() (oidcSettings, bool) {
	settings := oidcSettings{
		endpoint:     preferences.GetString("casdoor.endpoint", ""),
		clientID:     preferences.GetString("casdoor.client_id", ""),
		clientSecret: preferences.GetString("casdoor.client_secret", ""),
	}
	if settings.endpoint == "" || settings.clientID == "" || settings.clientSecret == "" {
		return settings, false
	}
	return settings, true
}

// IsConfigured reports whether the OIDC (Casdoor) configuration is present.
func IsConfigured() bool {
	_, ok := loadSettings()
	return ok
}

// InitOIDC validates the OIDC configuration at startup. A missing configuration
// only disables the OIDC routes; other login paths are untouched.
func InitOIDC() {
	if !IsConfigured() {
		slog.Warn("OIDC(Casdoor)配置缺失，OIDC登录不可用")
		return
	}
	slog.Info("OIDC(Casdoor)配置已加载")
}

var (
	providerMu        sync.Mutex
	providerLoaded    bool
	cachedSettings    oidcSettings
	cachedProvider    *oidc.Provider
	cachedProviderErr error
)

// Provider returns a lazily-initialized OIDC provider, cached per configuration.
func Provider() (*oidc.Provider, error) {
	settings, ok := loadSettings()
	if !ok {
		return nil, ErrOIDCNotConfigured
	}
	providerMu.Lock()
	defer providerMu.Unlock()
	if providerLoaded && cachedSettings == settings {
		return cachedProvider, cachedProviderErr
	}
	cachedProvider, cachedProviderErr = oidc.NewProvider(context.Background(), settings.endpoint)
	cachedSettings = settings
	providerLoaded = true
	return cachedProvider, cachedProviderErr
}

// RedirectURL returns the OIDC callback URL derived from the site URL.
func RedirectURL() string {
	return hotdataserve.GetSiteSettingsConfigCache().SiteUrl + callbackPath
}

// OAuth2Config builds the oauth2 config for the OIDC provider.
func OAuth2Config(redirectURL string) (*oauth2.Config, error) {
	settings, ok := loadSettings()
	if !ok {
		return nil, ErrOIDCNotConfigured
	}
	provider, err := Provider()
	if err != nil {
		return nil, err
	}
	return &oauth2.Config{
		ClientID:     settings.clientID,
		ClientSecret: settings.clientSecret,
		Endpoint:     provider.Endpoint(),
		RedirectURL:  redirectURL,
		Scopes:       []string{oidc.ScopeOpenID, oidc.ScopeProfile, oidc.ScopeEmail},
	}, nil
}

// StartLogin begins the OIDC authorization flow: it stores a fresh
// state/nonce/PKCE verifier in the session and returns the authorization URL.
func StartLogin(c *gin.Context) (string, error) {
	if _, ok := loadSettings(); !ok {
		return "", ErrOIDCNotConfigured
	}

	state, err := randomHex(32)
	if err != nil {
		return "", err
	}
	nonce, err := randomHex(32)
	if err != nil {
		return "", err
	}
	verifierBytes, err := algorithm.GenerateRandomBytes(32)
	if err != nil {
		return "", err
	}
	verifier := base64.RawURLEncoding.EncodeToString(verifierBytes)
	challenge := base64.RawURLEncoding.EncodeToString(sha256Digest(verifier))

	session, err := sessionstore.GetSession().Get(c.Request, sessionName)
	if err != nil {
		return "", fmt.Errorf("读取session失败: %w", err)
	}
	session.Values[sessionKeyState] = state
	session.Values[sessionKeyNonce] = nonce
	session.Values[sessionKeyVerifier] = verifier
	if err := session.Save(c.Request, c.Writer); err != nil {
		return "", fmt.Errorf("保存session失败: %w", err)
	}

	config, err := OAuth2Config(RedirectURL())
	if err != nil {
		return "", err
	}
	return config.AuthCodeURL(state,
		oauth2.SetAuthURLParam("nonce", nonce),
		oauth2.SetAuthURLParam("code_challenge", challenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	), nil
}

// HandleCallback verifies the OIDC callback (state, code exchange, id_token,
// nonce) and returns the parsed numeric identity.
func HandleCallback(c *gin.Context) (CallbackResult, error) {
	settings, ok := loadSettings()
	if !ok {
		return CallbackResult{}, ErrOIDCNotConfigured
	}

	session, err := sessionstore.GetSession().Get(c.Request, sessionName)
	if err != nil {
		return CallbackResult{}, fmt.Errorf("读取session失败: %w", err)
	}
	sessionState, _ := session.Values[sessionKeyState].(string)
	sessionNonce, _ := session.Values[sessionKeyNonce].(string)
	verifier, _ := session.Values[sessionKeyVerifier].(string)
	if sessionState == "" || sessionNonce == "" || verifier == "" {
		return CallbackResult{}, ErrStateMismatch
	}
	if c.Query("state") != sessionState {
		return CallbackResult{}, ErrStateMismatch
	}

	provider, err := Provider()
	if err != nil {
		return CallbackResult{}, err
	}
	config, err := OAuth2Config(RedirectURL())
	if err != nil {
		return CallbackResult{}, err
	}
	token, err := config.Exchange(context.Background(), c.Query("code"), oauth2.VerifierOption(verifier))
	if err != nil {
		return CallbackResult{}, fmt.Errorf("交换token失败: %w", err)
	}
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		return CallbackResult{}, errors.New("id_token缺失")
	}

	idToken, err := provider.Verifier(&oidc.Config{ClientID: settings.clientID}).Verify(context.Background(), rawIDToken)
	if err != nil {
		return CallbackResult{}, fmt.Errorf("校验id_token失败: %w", err)
	}
	if idToken.Nonce != sessionNonce {
		return CallbackResult{}, ErrNonceMismatch
	}

	var claims struct {
		Sub               string `json:"sub"`
		PreferredUsername string `json:"preferred_username"`
		Name              string `json:"name"`
		Email             string `json:"email"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return CallbackResult{}, fmt.Errorf("解析claims失败: %w", err)
	}

	sub, err := strconv.ParseUint(claims.Sub, 10, 64)
	if err != nil {
		return CallbackResult{}, ErrNonNumericSub
	}
	// 0 is the logged-out/absent-user sentinel; never treat it as a real user.
	if sub == 0 {
		return CallbackResult{}, ErrNonNumericSub
	}
	username := claims.PreferredUsername
	if username == "" {
		username = claims.Name
	}
	return CallbackResult{Sub: sub, Username: username, Email: claims.Email}, nil
}

// MatchOrCreateUser returns the local user bound to the OIDC sub, creating one
// when no binding exists. Email is only attached when the claimed email is
// absent from the local users table; accounts are never merged across providers.
func MatchOrCreateUser(result CallbackResult) (*users.EntityComplete, error) {
	providerUID := strconv.FormatUint(result.Sub, 10)

	existing := userOAuth.GetByProviderAndUID(ProviderCasdoor, providerUID)
	if existing != nil {
		user, err := users.Get(existing.UserId)
		if err != nil {
			return nil, fmt.Errorf("获取用户信息失败: %w", err)
		}
		return &user, nil
	}

	username := sanitizeUsername(result.Username)
	if username == "" {
		username = fmt.Sprintf("user%d", result.Sub)
	}
	baseUsername := username
	for i := 1; users.ExistUsername(username); i++ {
		username = fmt.Sprintf("%s_%d", baseUsername, i)
	}

	email := ""
	if result.Email != "" && !users.ExistEmail(result.Email) {
		email = result.Email
	}

	// 随机密码仅作占位（OIDC 用户不通过密码登录），用加密随机源生成。
	randomPassword, err := randomBase64(32)
	if err != nil {
		return nil, fmt.Errorf("生成随机密码失败: %w", err)
	}
	userEntity, err := userservice.CreateUser(username, randomPassword, email, true, "")
	if err != nil {
		return nil, fmt.Errorf("创建用户失败: %w", err)
	}
	// 与密码注册路径一致，发布注册事件（统计/活动/通知）。
	eventbus.Publish(context.Background(), &eventhandlers.UserSignUpEvent{
		UserId:   userEntity.Id,
		Username: userEntity.Username,
	})
	if userEntity.IsActivated == users.ActivationPending {
		userEntity.IsActivated = users.ActivationSuccess
		if err := userservice.SaveUser(userEntity); err != nil {
			return nil, fmt.Errorf("激活用户失败: %w", err)
		}
	}

	oauthEntity := &userOAuth.Entity{
		UserId:      userEntity.Id,
		Provider:    ProviderCasdoor,
		ProviderUid: providerUID,
		TokenExpiry: time.Now().AddDate(1, 0, 0),
	}
	if err := userOAuth.Create(oauthEntity); err != nil {
		return nil, fmt.Errorf("保存OAuth绑定失败: %w", err)
	}

	return userEntity, nil
}

// sanitizeUsername keeps only characters allowed by the forum username rule
// ([a-zA-Z0-9_-], 6-32 chars). It returns "" when nothing usable remains.
func sanitizeUsername(raw string) string {
	var b strings.Builder
	for _, r := range raw {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_', r == '-':
			b.WriteRune(r)
		}
	}
	name := b.String()
	if len(name) < 6 {
		return ""
	}
	if len(name) > 32 {
		name = name[:32]
	}
	return name
}

func randomHex(size int) (string, error) {
	bytes, err := algorithm.GenerateRandomBytes(size)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func randomBase64(size int) (string, error) {
	bytes, err := algorithm.GenerateRandomBytes(size)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func sha256Digest(data string) []byte {
	sum := sha256.Sum256([]byte(data))
	return sum[:]
}
