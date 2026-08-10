package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	jwt "github.com/leancodebox/GooseForum/app/bundles/jwtopt"
	"github.com/leancodebox/GooseForum/app/http/controllers/component"
	"github.com/leancodebox/GooseForum/app/models/forum/users"
	"github.com/leancodebox/GooseForum/app/service/totpservice"
	"github.com/leancodebox/GooseForum/app/service/userservice"
)

// TOTPChallengeAuth 校验两步验证 challenge token：仅接受
// purpose=totp_challenge 且未过期的短时 token，并把 userId 写入上下文。
// 不检查会话表——challenge token 从不创建会话记录。
func TOTPChallengeAuth(c *gin.Context) {
	token := jwt.GetGinAccessToken(c)
	if token == "" {
		c.JSON(http.StatusUnauthorized, component.FailDataCode(component.MessageAuthRequired, nil))
		c.Abort()
		return
	}
	claims, _, err := jwt.VerifyTokenWithFreshClaims(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, component.FailDataCode(component.MessageAuthRequired, nil))
		c.Abort()
		return
	}
	if claims.UserId == 0 || claims.Purpose != jwt.PurposeTotpChallenge {
		c.JSON(http.StatusUnauthorized, component.FailDataCode(component.MessageAuthRequired, nil))
		c.Abort()
		return
	}
	// 改密或退出所有设备（TokenVersion++）后，旧 challenge token 立即失效。
	user, ok := userservice.GetUserInfo(claims.UserId)
	if !ok || user.TokenVersion != claims.TokenVersion {
		c.JSON(http.StatusUnauthorized, component.FailDataCode(component.MessageAuthRequired, nil))
		c.Abort()
		return
	}
	// 机器人（Agent）账号不参与人类两步验证流程。
	if user.ActorType == users.ActorTypeBot {
		c.JSON(http.StatusUnauthorized, component.FailDataCode(component.MessageAuthRequired, nil))
		c.Abort()
		return
	} else if user.IsFrozen == users.StatusFrozen {
		// 冻结用户不允许登录（与 Login 第一阶段的冻结检查一致）：challenge
		// 签发后被冻结的账号在提交 TOTP 时在此被拒绝，不消费 challenge、
		// 不创建会话。
		c.JSON(http.StatusForbidden, component.FailDataCode(component.MessageAuthAccountFrozen, nil))
		c.Abort()
		return
	}
	exp, err := claims.GetExpirationTime()
	if err != nil || exp == nil || time.Now().After(exp.Time) {
		c.JSON(http.StatusUnauthorized, component.FailDataCode(component.MessageAuthRequired, nil))
		c.Abort()
		return
	}
	if !totpservice.ChallengeValid(claims.UserId, claims.Jti) {
		c.JSON(http.StatusUnauthorized, component.FailDataCode(component.MessageAuthRequired, nil))
		c.Abort()
		return
	}
	c.Set("currentJti", claims.Jti)
	c.Set("userId", claims.UserId)
	c.Next()
}
