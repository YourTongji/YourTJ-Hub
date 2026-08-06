package api

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/leancodebox/GooseForum/app/bundles/algorithm"
	"github.com/leancodebox/GooseForum/app/bundles/jwtopt"
	"github.com/leancodebox/GooseForum/app/http/controllers/component"
	"github.com/leancodebox/GooseForum/app/models/forum/users"
	"github.com/leancodebox/GooseForum/app/service/sessionservice"
	"github.com/leancodebox/GooseForum/app/service/totpservice"
	"github.com/leancodebox/GooseForum/app/service/userservice"
)

// TotpSetupReq 获取两步验证设置，需登录密码二次确认。
type TotpSetupReq struct {
	Password string `json:"password" validate:"required"`
}

// TotpEnableReq 启用两步验证。
type TotpEnableReq struct {
	Code string `json:"code" validate:"required"`
}

// TotpDisableReq 关闭两步验证，code 为 TOTP 码或登录密码。
type TotpDisableReq struct {
	Code string `json:"code" validate:"required"`
}

// TotpVerifyReq 完成两步验证登录，code 与 recoveryCode 二选一。
type TotpVerifyReq struct {
	Code         string `json:"code"`
	RecoveryCode string `json:"recoveryCode"`
}

// TotpSetup 获取两步验证密钥与 otpauth 链接。
// 需要登录密码确认：防止会话窃取者替受害者启用 2FA 并锁死其账户。
func TotpSetup(req component.BetterRequest[TotpSetupReq]) component.Response {
	user, err := users.Get(req.UserId)
	if err != nil {
		slog.Error("TOTP setup: user not found", "userId", req.UserId, "error", err)
		return component.FailResponseCode(component.MessageTotpSetupFailed, nil)
	}
	if err := algorithm.VerifyEncryptPassword(user.Password, req.Params.Password); err != nil {
		return component.FailResponseCode(component.MessageAuthInvalidCredentials, nil)
	}
	result, err := totpservice.Setup(req.UserId)
	if err != nil {
		if errors.Is(err, totpservice.ErrAlreadyEnabled) {
			return component.FailResponseCode(component.MessageTotpAlreadyEnabled, nil)
		}
		slog.Error("TOTP setup failed", "userId", req.UserId, "error", err)
		return component.FailResponseCode(component.MessageTotpSetupFailed, nil)
	}
	return component.SuccessResponseCode(
		component.DataMap{"secret": result.Secret, "otpauthUrl": result.OtpauthURL},
		component.MessageOperationSuccess,
		nil)
}

// TotpEnable 校验验证码并启用两步验证，返回一次性恢复码。
func TotpEnable(req component.BetterRequest[TotpEnableReq]) component.Response {
	codes, err := totpservice.Enable(req.UserId, req.Params.Code)
	if err != nil {
		switch {
		case errors.Is(err, totpservice.ErrAlreadyEnabled):
			return component.FailResponseCode(component.MessageTotpAlreadyEnabled, nil)
		case errors.Is(err, totpservice.ErrInvalidCode):
			return component.FailResponseCode(component.MessageTotpCodeInvalid, nil)
		case errors.Is(err, totpservice.ErrNotEnabled):
			return component.FailResponseCode(component.MessageTotpNotEnabled, nil)
		}
		slog.Error("TOTP enable failed", "userId", req.UserId, "error", err)
		return component.FailResponseCode(component.MessageTotpEnableFailed, nil)
	}
	return component.SuccessResponseCode(
		component.DataMap{"recoveryCodes": codes},
		component.MessageOperationSuccess,
		nil)
}

// TotpDisable 关闭两步验证，code 为 TOTP 码或登录密码。
func TotpDisable(req component.BetterRequest[TotpDisableReq]) component.Response {
	err := totpservice.Disable(req.UserId, req.Params.Code)
	if err != nil {
		switch {
		case errors.Is(err, totpservice.ErrNotEnabled):
			return component.FailResponseCode(component.MessageTotpNotEnabled, nil)
		case errors.Is(err, totpservice.ErrInvalidCode):
			return component.FailResponseCode(component.MessageTotpCodeInvalid, nil)
		}
		slog.Error("TOTP disable failed", "userId", req.UserId, "error", err)
		return component.FailResponseCode(component.MessageTotpDisableFailed, nil)
	}
	return component.SuccessResponseCode("两步验证已关闭", component.MessageOperationSuccess, nil)
}

// TotpVerify 校验两步验证码/恢复码并签发正式会话 token。
// 该端点挂在 TOTPChallengeAuth 中间件之后，userId 取自 challenge token。
func TotpVerify(c *gin.Context) {
	var req TotpVerifyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, component.FailDataCode(component.MessageRequestInvalidFormat, nil))
		return
	}
	userId := c.GetUint64("userId")
	if userId == 0 {
		c.JSON(http.StatusUnauthorized, component.FailDataCode(component.MessageAuthRequired, nil))
		return
	}
	code := req.Code
	if code == "" {
		code = req.RecoveryCode
	}
	if code == "" {
		c.JSON(http.StatusOK, component.FailDataCode(component.MessageTotpCodeInvalid, nil))
		return
	}
	ok, err := totpservice.Verify(userId, code)
	if err != nil || !ok {
		if errors.Is(err, totpservice.ErrRateLimited) {
			c.JSON(http.StatusOK, component.FailDataCode(component.MessageTotpRateLimited, nil))
			return
		}
		c.JSON(http.StatusOK, component.FailDataCode(component.MessageTotpCodeInvalid, nil))
		return
	}
	userInfo, ok := userservice.GetUserInfo(userId)
	if !ok {
		slog.Error("TOTP verify: user info missing", "userId", userId)
		c.JSON(http.StatusOK, component.FailDataCode(component.MessageAuthLoginFailed, nil))
		return
	}
	token, jti, err := jwtopt.CreateSessionToken(userId, userInfo.TokenVersion)
	if err != nil {
		slog.Error("TOTP verify: create session token failed", "userId", userId, "error", err)
		c.JSON(http.StatusOK, component.FailDataCode(component.MessageAuthLoginFailed, nil))
		return
	}
	if err = sessionservice.Create(userId, jti, c.Request.UserAgent(), c.ClientIP()); err != nil {
		slog.Error("TOTP verify: persist session failed", "userId", userId, "error", err)
		c.JSON(http.StatusOK, component.FailDataCode(component.MessageAuthLoginFailed, nil))
		return
	}
	jwtopt.TokenSetting(c, token)
	c.JSON(http.StatusOK, component.SuccessDataCode("登录成功", component.MessageAuthLoginSuccess, nil))
}
