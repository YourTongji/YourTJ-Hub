package routes

import (
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterByGin(ginApp *gin.Engine) {
	// 基础中间件
	ginApp.Use(middleware.Recovery())
	ginApp.Use(middleware.SiteMaintenance)
	// 安全响应头（注册在 SiteMaintenance 之后：维护页含内联脚本，不能被页面级 CSP 禁用）
	ginApp.Use(middleware.SecurityHeaders)
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
