package middleware

import (
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/eventbus"
	jwt "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/jwtopt"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/authsessionservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/eventhandlers"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/sessionservice"
	"github.com/gin-gonic/gin"
)

const SkipUpdateUserActivity = "SkipUpdateUserActivity"

func JWTAuthCheck(c *gin.Context) {
	userId := JWTAuthGetUserId(c)
	if userId == 0 {
		abortGuardFailure(c, http.StatusUnauthorized, component.FailDataCode(component.MessageAuthRequired, nil))
		c.Abort()
		return
	}
	c.Set("userId", userId)
	c.Next()
	if !c.GetBool(SkipUpdateUserActivity) {
		eventbus.Publish(eventbus.DetachedContext(c.Request.Context()), &eventhandlers.UserLastActiveUpdatedEvent{
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
		eventbus.Publish(eventbus.DetachedContext(c.Request.Context()), &eventhandlers.UserLastActiveUpdatedEvent{
			UserId:     userId,
			ActiveTime: time.Now(),
		})
	}
}

func JWTAuthGetUserId(c *gin.Context) uint64 {
	token := jwt.GetGinAccessToken(c)
	res, ok := authsessionservice.ValidateSession(token)
	if !ok {
		return 0
	}
	if res.Refreshed {
		persisted := true
		if res.NeedsPersist && res.Jti != "" {
			var err error
			persisted, err = sessionservice.TouchExpiry(res.Jti, res.ExpiresAt)
			if err != nil {
				slog.Warn("Failed to persist sliding session expiry", "userId", res.UserID, "jti", res.Jti, "error", err)
			}
		}
		if persisted {
			maxAge := time.Until(res.ExpiresAt)
			if maxAge < time.Second {
				maxAge = time.Second
			}
			jwt.TokenSettingWithMaxAge(c, res.Token, maxAge)
		}
	}
	c.Set("currentJti", res.Jti)
	return res.UserID
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
