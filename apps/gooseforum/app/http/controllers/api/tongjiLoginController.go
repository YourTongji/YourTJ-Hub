package api

import (
	"errors"
	"net/http"
	"net/url"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/eventbus"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/i18n"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/jwtopt"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/setting"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/forum"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/campus"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/campusservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/eventhandlers"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/sessionservice"
	"github.com/gin-gonic/gin"
)

// Keep provider I/O replaceable in handler-chain tests; production uses the configured singleton.
var tongjiLoginService = campusservice.Default

const tongjiLoginCookie = "yourtj_tongji_login"

func tongjiCookie(c *gin.Context, value string, age int) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(tongjiLoginCookie, value, age, "/api/campus/tongji/callback", "", setting.CookieSecure(), true)
}

func tongjiLoginFailure(c *gin.Context, reason, redirect string) {
	destination := "/login?tongjiNotice=" + url.QueryEscape(reason)
	if forum.IsSafeRedirect(redirect) {
		destination += "&redirect=" + url.QueryEscape(redirect)
	}
	c.Redirect(http.StatusSeeOther, destination)
}

func TongjiLoginStart(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	c.Header("Referrer-Policy", "no-referrer")
	s, err := tongjiLoginService()
	if err != nil {
		tongjiLoginFailure(c, "unavailable", "")
		return
	}
	redirect := c.Query("redirect")
	if !forum.IsSafeRedirect(redirect) {
		redirect = "/"
	}
	locale := c.Query("locale")
	if locale == "" {
		locale = c.GetString("locale")
	}
	address, browser, err := s.StartLogin(redirect, i18n.Normalize(locale))
	if err != nil {
		tongjiLoginFailure(c, "failed", redirect)
		return
	}
	tongjiCookie(c, browser, 600)
	c.Redirect(http.StatusFound, address)
}

// Only login-purpose state reaches this handler. A missing cookie can never
// downgrade into an authenticated binding, or authenticate another browser.
func TongjiLoginCallback(c *gin.Context) {
	browser, _ := c.Cookie(tongjiLoginCookie)
	tongjiCookie(c, "", -1)
	s, err := tongjiLoginService()
	if err != nil {
		tongjiLoginFailure(c, "unavailable", "")
		return
	}
	code := c.Query("code")
	if c.Query("error") != "" {
		code = ""
	}
	result, err := s.Login(c.Request.Context(), browser, c.Query("state"), code, hotdataserve.GetSecuritySettingsConfigCache())
	if err != nil {
		reason := "failed"
		switch {
		case errors.Is(err, users.ErrEmailOccupied), errors.Is(err, campus.ErrIdentityUsed):
			reason = "accountExists"
		case errors.Is(err, campusservice.ErrSignupDisabled), errors.Is(err, users.ErrSignupQuota):
			reason = "signupDisabled"
		case errors.Is(err, campusservice.ErrAccountUnavailable):
			reason = "accountUnavailable"
		}
		tongjiLoginFailure(c, reason, result.Redirect)
		return
	}
	if result.Created {
		eventbus.Publish(detachedRequestContext(c), &eventhandlers.UserSignUpEvent{UserId: result.User.Id, Username: result.User.Username})
	}
	token, jti, err := jwtopt.CreateSessionToken(result.User.Id, result.User.TokenVersion)
	if err == nil {
		err = sessionservice.Create(result.User.Id, jti, c.Request.UserAgent(), c.ClientIP())
	}
	if err != nil {
		tongjiLoginFailure(c, "failed", result.Redirect)
		return
	}
	jwtopt.TokenSetting(c, token)
	destination := result.Redirect
	if result.Created {
		// Preserve the server-validated continuation, including native OIDC, while
		// the new account chooses a public username and optional email password setup.
		destination = "/settings?onboarding=tongji&returnTo=" + url.QueryEscape(destination)
	}
	c.Redirect(http.StatusSeeOther, destination)
}
