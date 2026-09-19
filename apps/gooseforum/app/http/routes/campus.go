package routes

import (
	"context"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/api"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/middleware"
	"github.com/gin-gonic/gin"
)

func campusRoutes(app *gin.Engine) {
	g := app.Group("/api/campus", middleware.CSRFProtection, middleware.JWTAuthCheck, func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 8*time.Second)
		defer cancel()
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
	g.GET("/status", api.CampusStatus)
	g.GET("/data/:dataset", middleware.RateLimit(middleware.RateLimitCourseCatalog), api.CampusDataset)
	g.GET("/messages/:messageId", middleware.RateLimit(middleware.RateLimitCourseCatalog), api.CampusMessage)
	g.POST("/tongji/start", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitLogin), api.CampusStart)
	g.GET("/tongji/callback", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitLogin), api.CampusCallback)
	g.POST("/tongji/confirm", middleware.CheckWritableAccount, api.CampusConfirm)
	g.POST("/tongji/unbind", middleware.CheckWritableAccountAllowPendingActivation, api.CampusUnbind)
}
