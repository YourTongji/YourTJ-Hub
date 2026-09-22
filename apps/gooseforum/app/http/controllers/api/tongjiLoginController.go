package api

import (
	"crypto/subtle"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/algorithm"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/eventbus"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/i18n"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/jwtopt"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/setting"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/forum"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/campus"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/campusservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/eventhandlers"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/moderationservice"
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
	if result.Registration != "" {
		tongjiRegistrationCookie(c, result.Registration, 600)
		c.Redirect(http.StatusSeeOther, "/register/tongji")
		return
	}
	if err := finishTongjiSession(c, result); err != nil {
		tongjiLoginFailure(c, "failed", result.Redirect)
		return
	}
	c.Redirect(http.StatusSeeOther, result.Redirect)
}

const tongjiRegistrationCookieName = "yourtj_tongji_registration"

func tongjiRegistrationCookie(c *gin.Context, value string, age int) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(tongjiRegistrationCookieName, value, age, "/api/auth/tongji/registration", "", setting.CookieSecure(), true)
}

func registrationHeaders(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	c.Header("Referrer-Policy", "no-referrer")
}

func TongjiRegistrationStatus(c *gin.Context) {
	registrationHeaders(c)
	s, err := tongjiLoginService()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, component.FailDataCode(component.MessageAuthRegisterFailed, nil))
		return
	}
	cookie, _ := c.Cookie(tongjiRegistrationCookieName)
	status, err := s.RegistrationStatus(cookie)
	if err != nil {
		c.JSON(http.StatusGone, component.FailDataCode(component.MessageAuthRequired, nil))
		return
	}
	c.JSON(http.StatusOK, component.SuccessData(status))
}

type tongjiRegistrationRequest struct {
	Username  string `json:"username"`
	Password  string `json:"password"`
	CSRFToken string `json:"csrfToken"`
}

// tongjiRegistrationFailure maps completion errors onto the contract's statuses
// and codes. The proof pre-check already returned 410/403 for an invalid proof,
// so ErrFlow here means it expired or was consumed between the pre-check and
// the final submit; identity/email collisions return dedicated guidance because
// the requester's school identity is already verified, so naming the conflict
// creates no enumeration oracle.
func tongjiRegistrationFailure(err error) (int, component.MessageCode) {
	switch {
	case errors.Is(err, campusservice.ErrFlow):
		return http.StatusGone, component.MessageAuthRequired
	case errors.Is(err, campus.ErrIdentityUsed), errors.Is(err, users.ErrEmailOccupied):
		return http.StatusConflict, component.MessageAuthTongjiAccountExists
	case errors.Is(err, campusservice.ErrSignupDisabled):
		return http.StatusConflict, component.MessageAuthSignupDisabled
	case errors.Is(err, users.ErrSignupQuota):
		return http.StatusConflict, component.MessageAuthRegisterDailyQuota
	}
	return http.StatusConflict, component.MessageAuthRegisterFailed
}

func TongjiRegister(c *gin.Context) {
	registrationHeaders(c)
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 16<<10)
	var input tongjiRegistrationRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, component.FailDataCode(component.MessageRequestInvalidFormat, nil))
		return
	}
	s, err := tongjiLoginService()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, component.FailDataCode(component.MessageAuthRegisterFailed, nil))
		return
	}
	ticket, _ := c.Cookie(tongjiRegistrationCookieName)
	status, err := s.RegistrationStatus(ticket)
	if err != nil {
		c.JSON(http.StatusGone, component.FailDataCode(component.MessageAuthRequired, nil))
		return
	}
	if input.CSRFToken == "" || subtle.ConstantTimeCompare([]byte(input.CSRFToken), []byte(status.CSRFToken)) != 1 {
		c.JSON(http.StatusForbidden, component.FailDataCode(component.MessageAuthCsrfRejected, nil))
		return
	}
	input.Username = strings.TrimSpace(input.Username)
	if !component.ValidateUsername(input.Username) {
		c.JSON(http.StatusBadRequest, component.FailDataCode(component.MessageAuthUsernameInvalid, nil))
		return
	}
	if _, err := moderationservice.CheckUsernameAllowed(input.Username); err != nil {
		c.JSON(http.StatusBadRequest, component.FailDataError(err))
		return
	}
	if err := component.ValidatePassword(input.Password, 6); err != nil {
		c.JSON(http.StatusBadRequest, component.FailDataError(err))
		return
	}
	hash, err := algorithm.MakePassword(input.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, component.FailDataCode(component.MessageAuthRegisterFailed, nil))
		return
	}
	result, err := s.CompleteRegistration(c.Request.Context(), ticket, input.CSRFToken, campusservice.Registration{Username: input.Username, PasswordHash: hash}, hotdataserve.GetSecuritySettingsConfigCache())
	if err != nil {
		status, code := tongjiRegistrationFailure(err)
		c.JSON(status, component.FailDataCode(code, nil))
		return
	}
	tongjiRegistrationCookie(c, "", -1)
	if err := finishTongjiSession(c, result); err != nil {
		c.JSON(http.StatusInternalServerError, component.FailDataCode(component.MessageAuthRegisterFailed, nil))
		return
	}
	c.JSON(http.StatusOK, component.SuccessData(map[string]string{"redirect": result.Redirect}))
}

func finishTongjiSession(c *gin.Context, result campusservice.LoginResult) error {
	if result.Created {
		eventbus.Publish(detachedRequestContext(c), &eventhandlers.UserSignUpEvent{UserId: result.User.Id, Username: result.User.Username})
	}
	token, jti, err := jwtopt.CreateSessionToken(result.User.Id, result.User.TokenVersion)
	if err == nil {
		err = sessionservice.Create(result.User.Id, jti, c.Request.UserAgent(), c.ClientIP())
	}
	if err != nil {
		return err
	}
	jwtopt.TokenSetting(c, token)
	return nil
}
