package routes

import (
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/middleware"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/defaultconfig"
)

// TestResetPasswordRateLimitWired 验证 issue #106 point 2 的限流兜底已就位：
// 中间件动作常量与默认 ratelimit.json 配置同时包含 reset-password 动作，
// 这样存量部署经 mergeDefaultRateLimitActions 自动获得该项默认配额。
func TestResetPasswordRateLimitWired(t *testing.T) {
	if middleware.RateLimitResetPassword != "reset-password" {
		t.Fatalf("RateLimitResetPassword = %q, want %q", middleware.RateLimitResetPassword, "reset-password")
	}
	defaults := defaultconfig.GetDefaultRateLimitConfig()
	for _, rule := range defaults.Actions {
		if rule.Action == "reset-password" {
			if rule.LimitPerIp <= 0 {
				t.Fatalf("reset-password default rule has non-positive limit: %+v", rule)
			}
			return
		}
	}
	t.Fatal("reset-password action missing from default rate-limit config")
}
