package middleware

import (
	"log/slog"
	"time"

	"github.com/leancodebox/GooseForum/app/bundles/preferences"

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
	if preferences.GetBool("log.logIp", false) {
		fields = append(fields, "ip", c.ClientIP())
	}

	slog.Info("access", fields...)
}
