package oidcservice

import (
	"context"
	"crypto/subtle"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/jwtopt"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/oidcAccessTokens"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/oidcAuthRequests"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/authsessionservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/sessionservice"
	"github.com/zitadel/oidc/v3/pkg/oidc"
	"github.com/zitadel/oidc/v3/pkg/op"
)

var (
	// ErrInvalidExchangeRequest is returned when a mobile exchange request is
	// missing the authorization code or PKCE verifier.
	ErrInvalidExchangeRequest = errors.New("oidc: 兑换请求参数缺失")
	// ErrInvalidMobileRedirectURI is returned when a mobile exchange request
	// supplies a redirect URI outside the configured allowlist.
	ErrInvalidMobileRedirectURI = errors.New("oidc: 移动端 redirect URI 不在白名单")
	// ErrInvalidGrant is returned when the authorization code cannot be
	// redeemed (unknown, used, expired, or PKCE mismatch).
	ErrInvalidGrant = errors.New("oidc: 授权码无效或已使用")
)

// MobileClientID is the default client ID for the forum mobile app.
const MobileClientID = "yourtj-mobile"

// defaultMobileRedirectURI is the single source of the mobile custom-scheme
// redirect default; it must stay in sync with the config template and the
// Flutter AppAuth redirectUri.
const defaultMobileRedirectURI = "yourtj://callback"

// isMobileRedirectAllowed reports whether redirectURI matches the configured
// mobile client allowlist (or the default when the client is unconfigured).
func isMobileRedirectAllowed(redirectURI string) bool {
	cfg, err := LoadConfig()
	if err != nil {
		return redirectURI == defaultMobileRedirectURI
	}
	for _, c := range cfg.Clients {
		if c.ID != MobileClientID {
			continue
		}
		for _, allowed := range c.RedirectURIs {
			if redirectURI == allowed {
				return true
			}
		}
		return false
	}
	return redirectURI == defaultMobileRedirectURI
}

// ExchangeResult carries the verified user identity after a successful code
// redemption.
type ExchangeResult struct {
	Sub uint64
}

// ExchangeCode redeems a local OIDC authorization code issued by the
// built-in provider (authorization code + PKCE S256, single-use) and returns
// the authenticated forum user. It drives the exact same library redemption
// path as the token endpoint (op.ValidateAccessTokenRequest): atomic
// single-use code consumption, PKCE verifier verification and redirect URI
// matching. No MatchOrCreateUser and no user_o_auth rows are involved — the
// code is bound to an existing forum user at authorize time.
func ExchangeCode(code, codeVerifier, nonce, redirectURI string) (ExchangeResult, error) {
	if code == "" || codeVerifier == "" || nonce == "" {
		return ExchangeResult{}, ErrInvalidExchangeRequest
	}
	if !isMobileRedirectAllowed(redirectURI) {
		return ExchangeResult{}, ErrInvalidMobileRedirectURI
	}
	provider, err := Provider()
	if err != nil {
		return ExchangeResult{}, err
	}

	tokenReq := &oidc.AccessTokenRequest{
		Code:         code,
		RedirectURI:  redirectURI,
		ClientID:     MobileClientID,
		CodeVerifier: codeVerifier,
	}
	authReq, _, err := op.ValidateAccessTokenRequest(context.Background(), tokenReq, provider)
	if err != nil {
		return ExchangeResult{}, ErrInvalidGrant
	}
	// The nonce is not part of the token request; the client sends it in the
	// JSON body, so verify it here against the stored authorization request.
	if authReq.GetNonce() != nonce {
		return ExchangeResult{}, ErrInvalidGrant
	}

	sub, err := strconv.ParseUint(authReq.GetSubject(), 10, 64)
	if err != nil || sub == 0 {
		return ExchangeResult{}, ErrInvalidGrant
	}
	user, err := users.Get(sub)
	if err != nil || user.Id == 0 || user.IsBot() {
		return ExchangeResult{}, ErrInvalidGrant
	}
	// The auth request row is fully consumed; remove it like the token
	// endpoint does after a successful exchange.
	_ = oidcAuthRequests.DeleteByRequestId(authReq.GetID())
	return ExchangeResult{Sub: sub}, nil
}

// CompleteLogin marks an in-flight authorization request as authenticated by
// the given forum user, provided the browser-binding cookie hash matches the
// hash recorded at authorize time. It is called by the login completion
// bridge after the user has authenticated; the request is then ready to be
// turned into a code by the /authorize/callback endpoint. The conditional
// update makes the completion single-shot.
func CompleteLogin(requestID string, userID uint64, authTime time.Time, bindingHash string) error {
	if requestID == "" || userID == 0 {
		return errors.New("oidc: 登录桥参数缺失")
	}
	entity := oidcAuthRequests.GetByRequestId(requestID)
	if entity == nil {
		return errors.New("oidc: 授权请求不存在")
	}
	user, err := users.Get(userID)
	if err != nil || user.Id == 0 || user.IsBot() {
		return errors.New("oidc: 机器人账号不允许使用人类 OIDC 会话")
	}
	if entity.ExpiresAt.Before(now()) {
		return errors.New("oidc: 授权请求已过期")
	}
	if entity.Done {
		return errors.New("oidc: 授权请求已完成")
	}
	// Browser binding: the callback must come from the same browser that
	// started the authorize request. Missing or mismatched binding is
	// rejected without touching the request row.
	if entity.BrowserBinding == "" || bindingHash == "" ||
		subtle.ConstantTimeCompare([]byte(entity.BrowserBinding), []byte(bindingHash)) != 1 {
		return errors.New("oidc: 浏览器绑定不匹配")
	}
	if affected := oidcAuthRequests.MarkDone(requestID, userID, authTime); affected != 1 {
		return errors.New("oidc: 授权请求并发完成冲突")
	}
	return nil
}

// IssueForumSessionToken creates a forum JWT session for the given user,
// persisting a session row. Used by the mobile exchange path.
func IssueForumSessionToken(userID uint64, userAgent, ip string) (string, error) {
	user, err := users.Get(userID)
	if err != nil || user.Id == 0 {
		return "", errors.New("oidc: 用户不存在")
	}
	if user.IsFrozen == users.StatusFrozen {
		return "", errors.New("oidc: 用户已冻结")
	}
	if user.IsBot() {
		return "", errors.New("oidc: 机器人账号不允许使用人类 OIDC 会话")
	}
	token, jti, err := jwtopt.CreateSessionToken(user.Id, user.TokenVersion)
	if err != nil {
		return "", err
	}
	if err := sessionservice.Create(user.Id, jti, userAgent, ip); err != nil {
		return "", err
	}
	return token, nil
}

// AuthenticateRequest resolves the forum user from the access_token cookie on
// a plain http.Request (used by the login completion bridge inside the OIDC
// provider mux). It shares the exact validation path with the HTTP middleware
// (authsessionservice.ValidateToken): token validity, token version and live
// session record are all required.
func AuthenticateRequest(r *http.Request) (uint64, error) {
	var token string
	if cookie, err := r.Cookie("access_token"); err == nil {
		token = cookie.Value
	}
	if token == "" {
		if auth := r.Header.Get("Authorization"); len(auth) > 7 && auth[:7] == "Bearer " {
			token = auth[7:]
		}
	}
	if token == "" {
		return 0, errors.New("oidc: 未登录")
	}
	userID, _, _, ok := authsessionservice.ValidateToken(token)
	if !ok {
		return 0, errors.New("oidc: 登录态无效")
	}
	return userID, nil
}

// CleanupExpired removes expired auth request and access token rows. It is
// cheap housekeeping called at serve startup (and can be scheduled by a cron
// job); the auth paths themselves reject expired rows independently.
func CleanupExpired() {
	nowTime := now()
	oidcAuthRequests.DeleteExpired(nowTime)
	oidcAccessTokens.DeleteExpired(nowTime)
}
