package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
)

// Health 健康检查端点,供部署冒烟与负载均衡探测使用。
// 返回 200 表示服务可用且主库连接正常,否则返回 503。
func Health(c *gin.Context) {
	db := dbconnect.Connect()
	if db == nil {
		c.JSON(http.StatusServiceUnavailable, component.DataMap{
			"status": "unhealthy",
			"db":     "unavailable",
		})
		return
	}
	sqlDB, err := db.DB()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, component.DataMap{
			"status": "unhealthy",
			"db":     "error",
		})
		return
	}
	if err := sqlDB.Ping(); err != nil {
		c.JSON(http.StatusServiceUnavailable, component.DataMap{
			"status": "unhealthy",
			"db":     "error",
		})
		return
	}
	c.JSON(http.StatusOK, component.DataMap{
		"status": "ok",
		"db":     "ok",
	})
}
