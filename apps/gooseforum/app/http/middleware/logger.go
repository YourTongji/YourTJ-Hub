package middleware

import (
	"log/slog"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/networkAccessLog"

	"github.com/gin-gonic/gin"
)

func AccessLog(c *gin.Context) {
	if !preferences.GetBool("server.accessLog", false) {
		c.Next()
		return
	}

	startTime := time.Now()
	path := c.Request.URL.Path
	raw := c.Request.URL.RawQuery

	c.Next()

	latency := time.Since(startTime)
	statusCode := c.Writer.Status()

	if raw != "" {
		path = path + "?" + raw
	}

	fields := []any{
		"status", statusCode,
		"latency", latency.String(),
		"latency_ms", latency.Milliseconds(),
		"method", c.Request.Method,
		"path", path,
		"route", c.FullPath(),
	}
	// IP 记录由 [log] logIp 开关控制，默认关闭（隐私最小化）
	clientIP := ""
	if preferences.GetBool("log.logIp", false) {
		clientIP = c.ClientIP()
		fields = append(fields, "ip", clientIP)
	}

	slog.Info("access", fields...)

	// 合规网络访问日志表：失败只告警，不阻断请求。
	if err := networkAccessLog.Record(networkAccessLog.Entity{
		Method:    c.Request.Method,
		Path:      path,
		Route:     c.FullPath(),
		Status:    statusCode,
		UserId:    c.GetUint64("userId"),
		ClientIP:  clientIP,
		LatencyMs: latency.Milliseconds(),
	}); err != nil {
		slog.Warn("network access log record failed", "err", err)
	}
}
