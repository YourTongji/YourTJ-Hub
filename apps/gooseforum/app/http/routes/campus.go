package routes

import (
	"context"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/api"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/middleware"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/campusservice"
	"github.com/gin-gonic/gin"
)

func campusRoutes(app *gin.Engine) {
	timeout := func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 8*time.Second)
		defer cancel()
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
	app.GET("/api/campus/tongji/callback", middleware.CampusCallbackPrivacy, timeout,
		func(c *gin.Context) {
			if !campusservice.IsLoginState(c.Query("state")) {
				middleware.JWTAuthCheck(c)
			}
		},
		func(c *gin.Context) {
			if !campusservice.IsLoginState(c.Query("state")) {
				middleware.CheckWritableAccount(c)
			}
		},
		func(c *gin.Context) {
			action := middleware.RateLimitCampusAuthorize
			if campusservice.IsLoginState(c.Query("state")) {
				action = middleware.RateLimitLogin
			}
			middleware.RateLimit(action)(c)
		}, api.CampusCallback)
	g := app.Group("/api/campus", middleware.CampusCallbackPrivacy, middleware.CSRFProtection, middleware.JWTAuthCheck, timeout)
	g.GET("/status", api.CampusStatus)
	g.GET("/data/:dataset", middleware.RateLimit(middleware.RateLimitCampusRead), api.CampusDataset)
	g.GET("/messages/:messageId", middleware.RateLimit(middleware.RateLimitCampusRead), api.CampusMessage)
	g.GET("/calendar-rules", middleware.RateLimit(middleware.RateLimitCampusRead), api.CampusCalendarRules)
	g.GET("/calendar-export", middleware.RateLimit(middleware.RateLimitCampusRead), api.CampusCalendarExport)
	g.POST("/tongji/start", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitCampusAuthorize), api.CampusStart)
	g.POST("/tongji/confirm", middleware.CheckWritableAccount, api.CampusConfirm)
	g.POST("/tongji/unbind", middleware.CheckWritableAccountAllowPendingActivation, api.CampusUnbind)
}
