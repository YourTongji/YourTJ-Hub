package middleware

import (
	"log/slog"
	"net/url"
	"strings"
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
	raw := logQuery(c.Request.URL)

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

// OAuth credentials and state must never enter application or network-access logs.
func logQuery(u *url.URL) string {
	if ShouldRedactQuery(u) {
		return ""
	}
	return u.RawQuery
}

// ShouldRedactQuery reports whether a URL's query contains authentication
// callback state and must be omitted from request logs.
func ShouldRedactQuery(u *url.URL) bool {
	if u == nil {
		return false
	}
	if isAuthenticationCallback(u.Path) || isSignedActionPage(u.Path) {
		return true
	}
	if u.Path == "/login" {
		values, err := url.ParseQuery(u.RawQuery)
		if err != nil {
			// A malformed query may embed an opaque continuation URL, and
			// ParseQuery drops unparseable pairs: fail closed.
			return true
		}
		raw := values.Get("redirect")
		if raw == "" {
			return false
		}
		// Fail closed: an unparseable redirect still embeds an opaque
		// continuation URL, so it must not enter logs either.
		redirect, err := url.Parse(raw)
		return err != nil || isAuthenticationCallback(redirect.Path) || isSignedActionPage(redirect.Path)
	}
	return false
}

func logReferer(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	if _, err := url.ParseQuery(u.RawQuery); err != nil {
		return ""
	}
	if !ShouldRedactQuery(u) {
		return raw
	}
	u.RawQuery = ""
	u.ForceQuery = false
	return u.String()
}

// isSignedActionPage 版主快捷审批确认页的查询串携带签名 token（issue #1049），
// 与认证回调同样不得进入请求日志（含未登录时 /login?redirect= 的续跳地址）。
func isSignedActionPage(path string) bool {
	return strings.TrimSuffix(path, "/") == "/moderation/action"
}

func isAuthenticationCallback(path string) bool {
	path = strings.TrimSuffix(path, "/")
	if path == "/api/campus/tongji/callback" || path == "/api/oauth/authorize/callback" {
		return true
	}
	rest := strings.TrimPrefix(path, "/api/auth/")
	if rest == path || !strings.HasSuffix(rest, "/callback") {
		return false
	}
	provider := strings.TrimSuffix(rest, "/callback")
	return provider != "" && !strings.Contains(provider, "/")
}
