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

// TestCourseRateLimitActionsWired 验证课程域各限流动作同时满足两件事：
// 声明了中间件动作常量，且在默认 ratelimit.json 中注册了正配额。
// 二者缺其一，middleware.RateLimit 会在 findRateLimitRule 返回 nil 时静默放行，
// 端点看起来挂了限流、实际完全不限流（PR #338 的 course.bookmark /
// course.review.dislike 即漏了后半段）。存量部署依赖
// mergeDefaultRateLimitActions 补齐，而它只能补齐默认表中已存在的动作。
func TestCourseRateLimitActionsWired(t *testing.T) {
	actions := map[string]string{
		middleware.RateLimitCourseCatalog:  "RateLimitCourseCatalog",
		middleware.RateLimitReviewWrite:    "RateLimitReviewWrite",
		middleware.RateLimitReviewHelpful:  "RateLimitReviewHelpful",
		middleware.RateLimitReviewDislike:  "RateLimitReviewDislike",
		middleware.RateLimitReviewReport:   "RateLimitReviewReport",
		middleware.RateLimitReviewReveal:   "RateLimitReviewReveal",
		middleware.RateLimitReviewModerate: "RateLimitReviewModerate",
		middleware.RateLimitCourseSummary:  "RateLimitCourseSummary",
		middleware.RateLimitCourseBookmark: "RateLimitCourseBookmark",
	}
	defaults := defaultconfig.GetDefaultRateLimitConfig()
	quotas := make(map[string]struct {
		LimitPerIp   int
		LimitPerUser int
	}, len(defaults.Actions))
	for _, rule := range defaults.Actions {
		quotas[rule.Action] = struct {
			LimitPerIp   int
			LimitPerUser int
		}{rule.LimitPerIp, rule.LimitPerUser}
	}
	for action, constant := range actions {
		quota, ok := quotas[action]
		if !ok {
			t.Errorf("%s (%q) missing from default rate-limit config", constant, action)
			continue
		}
		if quota.LimitPerIp <= 0 && quota.LimitPerUser <= 0 {
			t.Errorf("%s (%q) default rule has no positive quota: %+v", constant, action, quota)
		}
	}
}
