package api

import (
	"context"
	"strings"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/eventbus"
	jwt "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/jwtopt"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/logincrypto"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/vo"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/emailactivationservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/eventhandlers"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/moderationservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/sessionservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/totpservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/userservice"

	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/validate"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
)

func Logout(c *gin.Context) {
	token := jwt.GetGinAccessToken(c)
	if claims, _, err := jwt.VerifyTokenWithFreshClaims(token); err == nil && claims.Jti != "" {
		if err := sessionservice.RevokeByJti(claims.UserId, claims.Jti); err != nil {
			slog.Error("Logout revoke session failed", "userId", claims.UserId, "jti", claims.Jti, "error", err)
			jwt.TokenClean(c)
			c.JSON(http.StatusOK, component.FailDataCode(component.MessageSessionRevokeFailed, nil))
			return
		}
	}
	jwt.TokenClean(c)
	c.JSON(http.StatusOK, component.SuccessData("logout"))
}

// Register 注册
func Register(c *gin.Context) {
	var r vo.RegReq
	if err := c.ShouldBindJSON(&r); err != nil {
		c.JSON(200, component.FailDataCode(component.MessageRequestInvalidFormat, nil))
		return
	}
	if err := validate.Valid(r); err != nil {
		c.JSON(200, component.FailDataCode(component.MessageRequestInvalidParams, nil))
		return
	}

	securityConfig := hotdataserve.GetSecuritySettingsConfigCache()

	if !securityConfig.EnableSignup {
		c.JSON(200, component.FailDataCode(component.MessageAuthSignupDisabled, nil))
		return
	}

	// 蜜罐字段：正常用户不可见，填了即机器，静默拒绝（返回成功但不创建账号）。
	if strings.TrimSpace(r.Website) != "" {
		slog.Warn("honeypot_hit", "action", "register", "ip", c.ClientIP(), "userId", uint64(0))
		c.JSON(http.StatusOK, component.SuccessDataCode("登录成功", component.MessageAuthLoginSuccess, nil))
		return
	}

	r.Username = strings.TrimSpace(r.Username)
	r.Email = strings.TrimSpace(strings.ToLower(r.Email))

	if err := component.ValidateEmailDomain(r.Email); err != nil {
		c.JSON(200, component.FailDataError(err))
		return
	}

	if !component.ValidateUsername(r.Username) {
		c.JSON(200, component.FailDataCode(component.MessageAuthUsernameInvalid, nil))
		return
	}

	// 保留/禁用用户名检查
	if _, err := moderationservice.CheckUsernameAllowed(r.Username); err != nil {
		c.JSON(200, component.FailDataError(err))
		return
	}

	if err := component.ValidatePassword(r.Password, 6); err != nil {
		c.JSON(200, component.FailDataError(err))
		return
	}

	if ok, needCaptcha := checkCaptchaForRequest(c, r.CaptchaId, r.CaptchaCode, securityConfig.CaptchaRequired, minSubmitSecondsFor(), "register"); !ok {
		if needCaptcha {
			c.JSON(200, component.FailDataCode(component.MessageCaptchaRequired, component.MessageParams{"action": "register"}))
		} else {
			c.JSON(200, component.FailDataCode(component.MessageAuthCaptchaInvalid, nil))
		}
		return
	}

	// 账号枚举防护（CWE-208）：用户名/邮箱已占用时返回与其他注册失败一致的
	// auth.register.failed 错误体，不再区分 auth.username.exists / auth.email.exists，
	// 消除"具体哪个字段被占用"的子 oracle（邮箱注册状态属 PII 级身份关联信息）。
	// 两次存在性查询无条件执行，查询次数不随账号状态变化，消除查询次数侧信道。
	// 注意：注册协议本身（新建账号并自动登录成功 vs 失败）仍固有地区分邮箱是否
	// 已注册；彻底消除该残余信号需改为异步邮箱验证流程，属产品决策（issue #124 验收项 1）。
	usernameExists := users.ExistUsername(r.Username)
	emailExists := users.ExistEmail(r.Email)
	if usernameExists || emailExists {
		c.JSON(200, component.FailDataCode(component.MessageAuthRegisterFailed, nil))
		return
	}

	userEntity, err := userservice.CreateUser(r.Username, r.Password, r.Email, true, r.Locale)
	if userEntity == nil || err != nil {
		slog.Error("注册创建用户失败", "username", r.Username, "email", r.Email, "error", err)
		c.JSON(200, component.FailDataCode(component.MessageAuthRegisterFailed, nil))
		return
	}

	slog.Debug("注册用户创建成功", "userId", userEntity.Id, "username", userEntity.Username, "email", userEntity.Email, "enableEmailVerification", securityConfig.EnableEmailVerification)
	if err = emailactivationservice.SendActivationEmail(userEntity); err != nil {
		slog.Error("添加邮件任务到队列失败", "userId", userEntity.Id, "email", userEntity.Email, "error", err)
	} else {
		slog.Debug("注册激活邮件任务已提交", "userId", userEntity.Id, "email", userEntity.Email, "enableEmailVerification", securityConfig.EnableEmailVerification)
	}

	eventbus.Publish(context.Background(), &eventhandlers.UserSignUpEvent{
		UserId:   userEntity.Id,
		Username: userEntity.Username,
	})

	if userEntity.Id == 1 {
		WriteTopic(component.BetterRequest[WriteTopicReq]{
			Params: WriteTopicReq{
				Content:     userservice.GetWelcomeTopicContent(),
				Title:       "Hi With GooseForum",
				CategoryId:  []uint64{1},
				TopicStatus: 1,
			},
			UserId: userEntity.Id,
		})
	}

	token, jti, err := jwt.CreateSessionToken(userEntity.Id, userEntity.TokenVersion)
	if err != nil {
		c.JSON(200, component.FailDataCode(component.MessageAuthRegisterRetryLogin, nil))
		return
	}
	if err = sessionservice.Create(userEntity.Id, jti, c.Request.UserAgent(), c.ClientIP()); err != nil {
		slog.Error("注册创建会话失败", "userId", userEntity.Id, "error", err)
		c.JSON(200, component.FailDataCode(component.MessageAuthRegisterRetryLogin, nil))
		return
	}
	jwt.TokenSetting(c, token)

	if securityConfig.EnableEmailVerification {
		c.JSON(http.StatusOK, component.SuccessDataCode(
			"注册成功，请前往邮箱验证您的账号",
			component.MessageAuthRegisterEmailVerify,

			nil))
		return
	}

	c.JSON(http.StatusOK, component.SuccessDataCode("登录成功", component.MessageAuthLoginSuccess, nil))
}

type LoginReq struct {
	Username          string `json:"username" validate:"required"` // 可以是用户名或邮箱
	EncryptedPassword string `json:"encryptedPassword" validate:"required"`
	CaptchaId         string `json:"captchaId"`
	CaptchaCode       string `json:"captchaCode"`
}

func LoginPublicKey(c *gin.Context) {
	c.JSON(http.StatusOK, component.SuccessData(map[string]any{
		"publicKey": logincrypto.PublicKeyPEM(),
		"serverTs":  time.Now().UnixMilli(),
		"algorithm": "RSA-OAEP-256",
	}))
}

// Login 处理登录请求
func Login(c *gin.Context) {
	var req LoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(200, component.FailDataCode(component.MessageRequestInvalidFormat, nil))
		return
	}

	if err := validate.Valid(req); err != nil {
		c.JSON(200, component.FailDataCode(component.MessageRequestInvalidParams, nil))
		return
	}

	username := strings.TrimSpace(req.Username)
	captchaId := req.CaptchaId
	captchaCode := req.CaptchaCode

	if username == "" {
		c.JSON(200, component.FailDataCode(component.MessageRequestInvalidParams, nil))
		return
	}

	password, err := logincrypto.DecryptPassword(req.EncryptedPassword)
	if err != nil {
		slog.Info("登录密码解密失败", "username", username, "error", err)
		c.JSON(200, component.FailDataCode(component.MessageAuthLoginInvalidRequest, nil))
		return
	}

	if len(password) < 6 {
		c.JSON(200, component.FailDataCode(component.MessageAuthPasswordInvalidFormat, nil))
		return
	}

	if ok, needCaptcha := checkCaptchaForRequest(c, captchaId, captchaCode, hotdataserve.GetSecuritySettingsConfigCache().CaptchaRequired, minSubmitSecondsFor(), "login"); !ok {
		if needCaptcha {
			c.JSON(200, component.FailDataCode(component.MessageCaptchaRequired, component.MessageParams{"action": "login"}))
		} else {
			c.JSON(200, component.FailDataCode(component.MessageAuthCaptchaInvalid, nil))
		}
		return
	}

	userEntity, err := users.Verify(username, password)
	if err != nil {
		slog.Info("登录失败", "username", username, "error", err)
		c.JSON(200, component.FailDataCode(component.MessageAuthInvalidCredentials, nil))
		return
	}

	securityConfig := hotdataserve.GetSecuritySettingsConfigCache()
	if securityConfig.EnableEmailVerification && userEntity.IsActivated == users.ActivationPending {
		c.JSON(200, component.FailDataCode(component.MessageAuthEmailUnverified, nil))
		return
	}

	// 封禁用户不允许登录（与 OIDC/goth 路径的冻结检查一致）。
	if userEntity.IsFrozen == users.StatusFrozen {
		c.JSON(200, component.FailDataCode(component.MessageAuthAccountFrozen, nil))
		return
	}
	if totpservice.IsEnabled(userEntity.Id) {
		challengeToken, challengeJti, err := jwt.CreateChallengeTokenWithJti(userEntity.Id, userEntity.TokenVersion, jwt.PurposeTotpChallenge, 5*time.Minute)
		if err != nil {
			slog.Error("生成两步验证 challenge token 失败", "userId", userEntity.Id, "error", err)
			c.JSON(200, component.FailDataCode(component.MessageAuthLoginFailed, nil))
			return
		}
		if err := totpservice.SaveChallenge(userEntity.Id, challengeJti, 5*time.Minute); err != nil {
			slog.Error("Save TOTP challenge failed", "userId", userEntity.Id, "error", err)
			c.JSON(200, component.FailDataCode(component.MessageAuthLoginFailed, nil))
			return
		}
		jwt.TokenSettingWithMaxAge(c, challengeToken, 5*time.Minute)
		c.JSON(http.StatusOK, component.SuccessDataCode(
			map[string]any{
				"twoFactorRequired": true,
				"message":           "请输入两步验证码",
			},
			component.MessageAuthTotpRequired,
			nil,
		))
		return
	}

	token, jti, err := jwt.CreateSessionToken(userEntity.Id, userEntity.TokenVersion)
	if err != nil {
		slog.Error("生成 token 失败", "userId", userEntity.Id, "error", err)
		c.JSON(200, component.FailDataCode(component.MessageAuthLoginFailed, nil))
		return
	}
	if err = sessionservice.Create(userEntity.Id, jti, c.Request.UserAgent(), c.ClientIP()); err != nil {
		slog.Error("创建会话失败", "userId", userEntity.Id, "error", err)
		c.JSON(200, component.FailDataCode(component.MessageAuthLoginFailed, nil))
		return
	}

	jwt.TokenSetting(c, token)
	c.JSON(http.StatusOK, component.SuccessDataCode("登录成功", component.MessageAuthLoginSuccess, nil))
}
