package middleware

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/eventbus"
	jwt "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/jwtopt"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/authsessionservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/eventhandlers"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/sessionservice"
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
	userID, jti, newToken, ok := authsessionservice.ValidateToken(token)
	if !ok {
		return 0
	}
	if token != newToken {
		jwt.TokenSetting(c, newToken)
		// Keep the session record expiry aligned with the refreshed token.
		if jti != "" {
			if claims, _, err := jwt.VerifyTokenWithFreshClaims(newToken); err == nil {
				if exp, err := claims.GetExpirationTime(); err == nil {
					sessionservice.TouchExpiry(jti, exp.Time)
				}
			}
		}
	}
	c.Set("currentJti", jti)
	return userID
}

func NoUpdateUserActivity(c *gin.Context) {
	c.Set(SkipUpdateUserActivity, true)
	c.Next()
}

func CheckLogin(c *gin.Context) {
	userId := c.GetUint64("userId")
	if userId == 0 {
		// 获取当前请求的完整URL作为重定向参数，并用 QueryEscape 正确编码，
		// 避免 redirect 值中的 & / ? 等字符破坏 login 页面的查询串。
		redirectURL := c.Request.URL.String()
		c.Redirect(http.StatusFound, "/login?redirect="+url.QueryEscape(redirectURL))
		c.Abort()
		return
	}
	c.Next()
}
