package api

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/leancodebox/GooseForum/app/bundles/jwtopt"
	"github.com/leancodebox/GooseForum/app/bundles/validate"
	"github.com/leancodebox/GooseForum/app/http/controllers/component"
	"github.com/leancodebox/GooseForum/app/http/controllers/forum"
	"github.com/leancodebox/GooseForum/app/models/forum/userOAuth"
	"github.com/leancodebox/GooseForum/app/models/forum/users"
	"github.com/leancodebox/GooseForum/app/models/hotdataserve"
	"github.com/leancodebox/GooseForum/app/service/oidcservice"
	"github.com/leancodebox/GooseForum/app/service/sessionservice"
	"github.com/leancodebox/GooseForum/app/service/userservice"
)

// OidcLogin starts the OIDC (Casdoor) authorization flow and redirects the
// browser to the provider. It works both for login and binding; the callback
// decides based on the current login state.
func OidcLogin(c *gin.Context) {
	authURL, err := oidcservice.StartLogin(c)
	if err != nil {
		slog.Error("OIDC login start failed", "error", err)
		forum.RenderInternalOAuthErrorPage(c, component.MessageOidcStartFailed)
		return
	}
	c.Redirect(http.StatusFound, authURL)
}

// OidcCallback verifies the OIDC callback and either binds the identity to the
// logged-in user or signs the matched/created user in.
func OidcCallback(c *gin.Context) {
	result, err := oidcservice.HandleCallback(c)
	if err != nil {
		slog.Error("OIDC callback failed", "error", err)
		if errors.Is(err, oidcservice.ErrNonNumericSub) {
			forum.RenderOAuthErrorPage(c, http.StatusForbidden, component.MessageOAuthNumericSubRequired)
			return
		}
		forum.RenderInternalOAuthErrorPage(c, component.MessageOidcCallbackFailed)
		return
	}

	subStr := strconv.FormatUint(result.Sub, 10)
	currentUserId := c.GetUint64("userId")

	if currentUserId > 0 {
		// 绑定模式：已登录用户绑定 Casdoor 身份
		if userInfo, ok := userservice.GetUserInfo(currentUserId); !ok || userInfo.IsFrozen == users.StatusFrozen {
			forum.RenderOAuthErrorPage(c, http.StatusForbidden, component.MessagePermissionUserFrozen)
			return
		}

		existing := userOAuth.GetByProviderAndUID(oidcservice.ProviderCasdoor, subStr)
		if existing != nil {
			if existing.UserId != currentUserId {
				forum.RenderOAuthErrorPage(c, http.StatusForbidden, component.MessageOidcBindConflict)
				return
			}
			// 已绑定当前用户，幂等处理
			c.Redirect(http.StatusFound, "/settings?tab=binding")
			return
		}

		// 与 goth 路径一致：每个用户每个 provider 只允许一条绑定，
		// 防止同一用户绑定多个 Casdoor 身份（UI 不可见但每个都能登录）。
		if userOAuth.GetByUserIDAndProvider(currentUserId, oidcservice.ProviderCasdoor) != nil {
			forum.RenderOAuthErrorPage(c, http.StatusForbidden, component.MessageOidcBindConflict)
			return
		}

		if err := userOAuth.Create(&userOAuth.Entity{
			UserId:      currentUserId,
			Provider:    oidcservice.ProviderCasdoor,
			ProviderUid: subStr,
		}); err != nil {
			slog.Error("OIDC bind failed", "userId", currentUserId, "error", err)
			forum.RenderInternalOAuthErrorPage(c, component.MessageOidcBindFailed)
			return
		}
		c.Redirect(http.StatusFound, "/settings?tab=binding")
		return
	}
	// 登录模式：匹配已有绑定或创建新用户。
	// 站点关闭注册时，仅允许已存在 Casdoor 绑定的用户登录，禁止自助建号。
	securityConfig := hotdataserve.GetSecuritySettingsConfigCache()
	if !securityConfig.EnableSignup && userOAuth.GetByProviderAndUID(oidcservice.ProviderCasdoor, subStr) == nil {
		forum.RenderOAuthErrorPage(c, http.StatusForbidden, component.MessageAuthSignupDisabled)
		return
	}
	user, err := oidcservice.MatchOrCreateUser(result)
	if err != nil {
		slog.Error("OIDC match or create user failed", "error", err)
		forum.RenderInternalOAuthErrorPage(c, component.MessageOAuthProcessFailed)
		return
	}
	if user.IsFrozen == users.StatusFrozen {
		forum.RenderOAuthErrorPage(c, http.StatusForbidden, component.MessageOAuthAccountFrozen)
		return
	}

	token, jti, err := jwtopt.CreateSessionToken(user.Id, user.TokenVersion)
	if err != nil {
		slog.Error("Generate JWT token failed", "error", err)
		forum.RenderInternalOAuthErrorPage(c, component.MessageOAuthTokenFailed)
		return
	}
	if err = sessionservice.Create(user.Id, jti, c.Request.UserAgent(), c.ClientIP()); err != nil {
		slog.Error("OIDC create session failed", "userId", user.Id, "error", err)
		forum.RenderInternalOAuthErrorPage(c, component.MessageOAuthProcessFailed)
		return
	}

	jwtopt.TokenSetting(c, token)
	c.Redirect(http.StatusFound, "/")
}

// OidcExchangeRequest is the JSON body for the mobile OIDC exchange endpoint.
type OidcExchangeRequest struct {
	Code         string `json:"code" validate:"required"`
	CodeVerifier string `json:"codeVerifier" validate:"required"`
	Nonce        string `json:"nonce" validate:"required"`
	RedirectURI  string `json:"redirectUri" validate:"required"`
}

// OidcExchange signs the mobile client in after it obtained an authorization
// code from Casdoor via AppAuth. The PKCE verifier is generated and held by
// the client, so no server-side OIDC session is involved. The redirect URI
// must match the configured mobile allowlist.
func OidcExchange(c *gin.Context) {
	var req OidcExchangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, component.FailDataCode(component.MessageRequestInvalidFormat, nil))
		return
	}
	if err := validate.Valid(req); err != nil {
		c.JSON(http.StatusBadRequest, component.FailDataCode(component.MessageRequestInvalidParams, nil))
		return
	}

	result, err := oidcservice.ExchangeCode(req.Code, req.CodeVerifier, req.Nonce, req.RedirectURI)
	if err != nil {
		slog.Error("OIDC exchange failed", "error", err)
		switch {
		case errors.Is(err, oidcservice.ErrInvalidMobileRedirectURI):
			c.JSON(http.StatusForbidden, component.FailDataCode(component.MessageOidcCallbackFailed, nil))
		case errors.Is(err, oidcservice.ErrOIDCNotConfigured):
			c.JSON(http.StatusForbidden, component.FailDataCode(component.MessageOidcStartFailed, nil))
		case errors.Is(err, oidcservice.ErrNonNumericSub):
			c.JSON(http.StatusForbidden, component.FailDataCode(component.MessageOAuthNumericSubRequired, nil))
		default:
			c.JSON(http.StatusUnauthorized, component.FailDataCode(component.MessageOAuthProcessFailed, nil))
		}
		return
	}

	subStr := strconv.FormatUint(result.Sub, 10)
	// 登录模式：移动端不做绑定，绑定走 web。
	securityConfig := hotdataserve.GetSecuritySettingsConfigCache()
	if !securityConfig.EnableSignup && userOAuth.GetByProviderAndUID(oidcservice.ProviderCasdoor, subStr) == nil {
		c.JSON(http.StatusForbidden, component.FailDataCode(component.MessageAuthSignupDisabled, nil))
		return
	}
	user, err := oidcservice.MatchOrCreateUser(result)
	if err != nil {
		slog.Error("OIDC match or create user failed", "userId", result.Sub, "error", err)
		c.JSON(http.StatusInternalServerError, component.FailDataCode(component.MessageOAuthProcessFailed, nil))
		return
	}
	if user.IsFrozen == users.StatusFrozen {
		c.JSON(http.StatusForbidden, component.FailDataCode(component.MessageOAuthAccountFrozen, nil))
		return
	}

	token, jti, err := jwtopt.CreateSessionToken(user.Id, user.TokenVersion)
	if err != nil {
		slog.Error("Generate JWT token failed", "error", err)
		c.JSON(http.StatusInternalServerError, component.FailDataCode(component.MessageOAuthTokenFailed, nil))
		return
	}
	if err = sessionservice.Create(user.Id, jti, "yourtj-mobile/1.0", c.ClientIP()); err != nil {
		slog.Error("OIDC create session failed", "userId", user.Id, "error", err)
		c.JSON(http.StatusInternalServerError, component.FailDataCode(component.MessageOAuthProcessFailed, nil))
		return
	}

	jwtopt.TokenSetting(c, token)
	c.JSON(http.StatusOK, component.SuccessData(map[string]string{
		"token": token,
	}))
}
