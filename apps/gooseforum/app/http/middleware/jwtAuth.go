package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leancodebox/GooseForum/app/bundles/eventbus"
	jwt "github.com/leancodebox/GooseForum/app/bundles/jwtopt"
	"github.com/leancodebox/GooseForum/app/http/controllers/component"
	"github.com/leancodebox/GooseForum/app/models/forum/users"
	"github.com/leancodebox/GooseForum/app/service/eventhandlers"
	"github.com/leancodebox/GooseForum/app/service/sessionservice"
	"github.com/leancodebox/GooseForum/app/service/userservice"
)

const SkipUpdateUserActivity = "SkipUpdateUserActivity"

func JWTAuthCheck(c *gin.Context) {
	userId := JWTAuthGetUserId(c)
	if userId == 0 {
		c.JSON(http.StatusUnauthorized, component.FailDataCode(component.MessageAuthRequired, nil))
		c.Abort()
		return
	}
	c.Set("userId", userId)
	c.Next()
	if !c.GetBool(SkipUpdateUserActivity) {
		eventbus.Publish(context.Background(), &eventhandlers.UserLastActiveUpdatedEvent{
			UserId:     userId,
			ActiveTime: time.Now(),
		})
	}
}

func JWTAuth(c *gin.Context) {
	userId := JWTAuthGetUserId(c)
	if userId != 0 {
		c.Set("userId", userId)
	}
	c.Next()
	if userId != 0 && !c.GetBool(SkipUpdateUserActivity) {
		eventbus.Publish(context.Background(), &eventhandlers.UserLastActiveUpdatedEvent{
			UserId:     userId,
			ActiveTime: time.Now(),
		})
	}
}

func JWTAuthGetUserId(c *gin.Context) uint64 {
	token := jwt.GetGinAccessToken(c)
	if token == "" {
		return 0
	}
	claims, newToken, err := jwt.VerifyTokenWithFreshClaims(token)
	if err != nil {
		return 0
	}
	user, ok := userservice.GetUserInfo(claims.UserId)
	if !ok || user.TokenVersion != claims.TokenVersion {
		return 0
	}
	// 机器人（Agent）账号不参与人类会话：bot 行无法通过 JWT 会话中间件，
	// 也就无法创建/列出人类会话或访问任何登录态接口。
	if user.ActorType == users.ActorTypeBot {
		return 0
	}
	// Every accepted token must map to a live session record. Challenge
	// tokens (e.g. TOTP second factor) never create a session row, so they
	// are rejected here and can only be used by their dedicated middleware.
	if !sessionValid(claims.Jti, claims.UserId) {
		return 0
	}
	if token != newToken {
		jwt.TokenSetting(c, newToken)
		// Keep the session record expiry aligned with the refreshed token.
		if claims.Jti != "" {
			if exp, err := claims.GetExpirationTime(); err == nil {
				sessionservice.TouchExpiry(claims.Jti, exp.Time)
			}
		}
	}
	c.Set("currentJti", claims.Jti)
	return claims.UserId
}

// sessionValid reports whether jti maps to a non-expired session owned by userID.
func sessionValid(jti string, userID uint64) bool {
	if jti == "" {
		return false
	}
	entity := sessionservice.GetValidByJti(jti)
	return entity != nil && entity.UserId == userID
}

func NoUpdateUserActivity(c *gin.Context) {
	c.Set(SkipUpdateUserActivity, true)
	c.Next()
}

func CheckLogin(c *gin.Context) {
	userId := c.GetUint64("userId")
	if userId == 0 {
		// 获取当前请求的完整URL作为重定向参数
		redirectURL := c.Request.URL.String()
		c.Redirect(http.StatusFound, "/login?redirect="+redirectURL)
		c.Abort()
		return
	}
	c.Next()
}
