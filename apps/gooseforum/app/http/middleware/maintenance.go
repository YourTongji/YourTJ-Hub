package middleware

import (
	_ "embed"
	"net/http"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"

	"github.com/gin-gonic/gin"
)

//go:embed maintenance.html
var maintenanceHTML []byte

// SiteMaintenance 中间件用于检查站点是否处于维护状态
func SiteMaintenance(c *gin.Context) {
	// 从配置文件中读取维护模式状态
	maintenance := preferences.GetBool("app.maintenance")
	if maintenance {
		// 本中间件注册于 SecurityHeaders 之前且直接 Abort，维护响应不会流经
		// SecurityHeaders，通用安全头在此自行补齐（#441 review 意见）。
		// 页面级 CSP 仍刻意不加：维护页自带内联脚本（时钟/自动刷新），
		// script-src 'self' 会禁掉它们（见 securityHeaders.go 顶部说明）。
		setUniversalSecurityHeaders(c.Writer.Header())
		c.Writer.Header().Set("Content-Type", "text/html; charset=utf-8")

		// 设置HTTP状态码为503 Service Unavailable
		c.Writer.WriteHeader(http.StatusServiceUnavailable)

		// 返回优化后的维护页面
		if _, err := c.Writer.Write(maintenanceHTML); err != nil {
			c.Abort()
			return
		}
		// 中止后续处理
		c.Abort()
		return
	}
	// 如果不在维护模式，则继续处理请求
	c.Next()

}
