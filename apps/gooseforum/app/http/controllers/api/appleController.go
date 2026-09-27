package api

import (
	"errors"
	"net/http"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/jwtopt"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/appleauthservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/oidcservice"
	"github.com/gin-gonic/gin"
)

type AppleCredentialRequest struct {
	AuthorizationCode string `json:"authorizationCode"`
	IdentityToken     string `json:"identityToken"`
	Nonce             string `json:"nonce"`
}

func appleCredential(c *gin.Context) (AppleCredentialRequest, bool) {
	var req AppleCredentialRequest
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 24*1024)
	err := c.ShouldBindJSON(&req)
	if err != nil || len(req.AuthorizationCode) == 0 || len(req.AuthorizationCode) > 4096 || len(req.IdentityToken) == 0 || len(req.IdentityToken) > 16384 || len(req.Nonce) < 32 || len(req.Nonce) > 128 {
		c.JSON(http.StatusBadRequest, component.FailDataCode(component.MessageRequestInvalidParams, nil))
		return req, false
	}
	return req, true
}

// AppleExchange consumes a native Apple code and issues a numeric forum session
// only for an explicitly bound, available local account.
func AppleExchange(c *gin.Context) {
	req, ok := appleCredential(c)
	if !ok {
		return
	}
	user, err := appleauthservice.Exchange(c.Request.Context(), req.AuthorizationCode, req.IdentityToken, req.Nonce, 0)
	if err != nil {
		appleAuthFailure(c, err, false)
		return
	}
	token, err := oidcservice.IssueForumSessionToken(user.Id, c.Request.UserAgent(), c.ClientIP())
	if err != nil {
		c.JSON(http.StatusInternalServerError, component.FailDataCode(component.MessageOAuthTokenFailed, nil))
		return
	}
	jwtopt.TokenSetting(c, token)
	c.JSON(http.StatusOK, component.SuccessData(map[string]string{"token": token}))
}

// AppleBind runs behind CSRF, session, writable-account and rate-limit guards.
func AppleBind(c *gin.Context) {
	owner := c.GetUint64("userId")
	if owner == 0 {
		c.JSON(http.StatusUnauthorized, component.FailDataCode(component.MessageOAuthProcessFailed, nil))
		return
	}
	req, ok := appleCredential(c)
	if !ok {
		return
	}
	_, err := appleauthservice.Exchange(c.Request.Context(), req.AuthorizationCode, req.IdentityToken, req.Nonce, owner)
	if err != nil {
		appleAuthFailure(c, err, true)
		return
	}
	c.JSON(http.StatusOK, component.SuccessData(true))
}

func appleAuthFailure(c *gin.Context, err error, binding bool) {
	status, code := http.StatusServiceUnavailable, component.MessageAppleUnavailable
	switch {
	case errors.Is(err, appleauthservice.ErrNoBinding):
		status, code = http.StatusConflict, component.MessageAppleBindingRequired
	case errors.Is(err, appleauthservice.ErrAlreadyBound):
		status, code = http.StatusConflict, component.MessageAppleAlreadyBound
	case errors.Is(err, appleauthservice.ErrInvalidCredential):
		status, code = http.StatusUnauthorized, component.MessageOAuthProcessFailed
		if binding {
			status = http.StatusBadRequest
		}
	case errors.Is(err, appleauthservice.ErrAccountUnavailable):
		status, code = http.StatusForbidden, component.MessageOAuthAccountFrozen
	}
	// Never return or log provider tokens, database parameters or Apple responses.
	c.JSON(status, component.FailDataCode(code, nil))
}
