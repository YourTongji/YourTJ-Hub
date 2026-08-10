package api

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/leancodebox/GooseForum/app/bundles/jwtopt"
	"github.com/leancodebox/GooseForum/app/bundles/validate"
	"github.com/leancodebox/GooseForum/app/http/controllers/component"
	"github.com/leancodebox/GooseForum/app/models/forum/users"
	"github.com/leancodebox/GooseForum/app/service/oidcservice"
)

// OidcExchangeRequest is the JSON body for the mobile OIDC exchange endpoint.
// The mobile client obtained an authorization code from the forum built-in
// OIDC provider via AppAuth (PKCE S256) and exchanges it for a forum JWT.
type OidcExchangeRequest struct {
	Code         string `json:"code" validate:"required"`
	CodeVerifier string `json:"codeVerifier" validate:"required"`
	Nonce        string `json:"nonce" validate:"required"`
	RedirectURI  string `json:"redirectUri" validate:"required"`
}

// OidcExchange signs the mobile client in after it obtained an authorization
// code from the forum built-in OIDC provider via AppAuth. The code is bound
// to the forum user at authorize time and redeemed atomically (single-use,
// PKCE S256, nonce verified) through the same path as the token endpoint; on
// success a forum JWT session is created. No MatchOrCreateUser and no
// user_o_auth rows are involved.
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
		case errors.Is(err, oidcservice.ErrOIDCDisabled):
			c.JSON(http.StatusForbidden, component.FailDataCode(component.MessageOidcStartFailed, nil))
		case errors.Is(err, oidcservice.ErrInvalidGrant):
			c.JSON(http.StatusUnauthorized, component.FailDataCode(component.MessageOAuthProcessFailed, nil))
		default:
			c.JSON(http.StatusUnauthorized, component.FailDataCode(component.MessageOAuthProcessFailed, nil))
		}
		return
	}

	user, err := users.Get(result.Sub)
	if err != nil || user.Id == 0 {
		c.JSON(http.StatusUnauthorized, component.FailDataCode(component.MessageOAuthProcessFailed, nil))
		return
	}
	if user.IsFrozen == users.StatusFrozen {
		c.JSON(http.StatusForbidden, component.FailDataCode(component.MessageOAuthAccountFrozen, nil))
		return
	}

	token, err := oidcservice.IssueForumSessionToken(user.Id, "yourtj-mobile/1.0", c.ClientIP())
	if err != nil {
		slog.Error("OIDC create session failed", "userId", user.Id, "error", err)
		c.JSON(http.StatusInternalServerError, component.FailDataCode(component.MessageOAuthTokenFailed, nil))
		return
	}

	jwtopt.TokenSetting(c, token)
	c.JSON(http.StatusOK, component.SuccessData(map[string]string{
		"token": token,
	}))
}
