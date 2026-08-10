package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/leancodebox/GooseForum/app/http/controllers"
	"github.com/leancodebox/GooseForum/app/http/middleware"
)

func RegisterByGin(ginApp *gin.Engine) {
	// 基础中间件
	ginApp.Use(middleware.Recovery())
	ginApp.Use(middleware.SiteMaintenance)
	ginApp.Use(middleware.SiteInfo)
	// 访问日志中间件
	ginApp.Use(middleware.AccessLog)

	// 健康检查(部署冒烟/负载均衡探测,匿名可访问)
	ginApp.GET("/health", controllers.Health)
	siteInfoRoute(ginApp)
	// 接口
	apiRoute(ginApp)
	// MCP（streamable HTTP，读默认可用、写 opt-in）
	mcpRoute(ginApp)
	// 文件
	fileServer(ginApp)
	// view
	viewRoute(ginApp)
	// 资源
	assertRouter(ginApp)

	ginApp.NoRoute(controllers.NotFound)

}
