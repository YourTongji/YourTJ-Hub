package middleware

import (
	"log/slog"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/ratelimit"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/permission"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/userservice"
	"github.com/gin-gonic/gin"
)

// 限流动作标识，与管理面板 rateLimitSettings 的 actions.action 对应。
const (
	RateLimitRegister       = "register"
	RateLimitLogin          = "login"
	RateLimitOIDCAuthorize  = "oidc.authorize"
	RateLimitOIDCToken      = "oidc.token"
	RateLimitForgotPassword = "forgot-password"
	RateLimitResetPassword  = "reset-password"
	RateLimitEmailChange    = "email.change"
	RateLimitPasswordChange = "password.change"
	// RateLimitTotpSetup/Enable/Disable 限流 TOTP 账户管理中的凭据校验入口：
	// setup 校验账户密码、enable/disable 校验 6 位验证码（disable 也接受密码），
	// 未限流时会话窃取者可无限暴力破解，配额对齐 password.change。
	RateLimitTotpSetup     = "totp.setup"
	RateLimitTotpEnable    = "totp.enable"
	RateLimitTotpDisable   = "totp.disable"
	RateLimitTopicWrite    = "topic.write"
	RateLimitTopicStatus   = "topic.status"
	RateLimitPostCreate    = "post.create"
	RateLimitPostDelete    = "post.delete"
	RateLimitMessageSend   = "message.send"
	RateLimitUpload        = "upload"
	RateLimitInteract      = "interact"
	RateLimitLLMSIndex     = "llms.index"
	RateLimitLLMSFull      = "llms.full"
	RateLimitLLMSTopic     = "llms.topic"
	RateLimitMCPAuth       = "mcp.auth"
	RateLimitCourseCatalog = "course.catalog"
	RateLimitReviewWrite   = "course.review.write"
	RateLimitReviewHelpful = "course.review.helpful"
	RateLimitReviewReport  = "course.review.report"
	RateLimitReviewReveal  = "course.review.reveal"
	// RateLimitReviewModerate 课评审核操作（隐藏/恢复、举报队列）：60s 窗口
	// per-IP 60 / per-User 30（issue #176 B4）。比写接口宽松（审核是低频
	// 操作但需批量处理举报），同时防止单账号刷审核接口。
	RateLimitReviewModerate = "course.review.moderate"
	// RateLimitCourseSummary 课程 AI 总结端点（B7, issue #181）：
	// 读缓存免费，生成动作另有 service 内全局/单课限流，此处仅防脚本高频打端点。
	RateLimitCourseSummary = "course.summary"
)

// 配置（开关/配额/窗口）每次请求动态读取，管理面板保存后即时生效。
func RateLimit(action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		cfg := hotdataserve.GetRateLimitConfigCache()
		if !cfg.Enabled {
			c.Next()
			return
		}
		rule := findRateLimitRule(cfg.Actions, action)
		if rule == nil || (rule.LimitPerIp <= 0 && rule.LimitPerUser <= 0) {
			c.Next()
			return
		}

		userId := c.GetUint64("userId")
		if cfg.SkipAdmin && userId != 0 {
			if roleId, ok := userservice.GetUserRoleId(userId); ok && permission.CheckRole(roleId, permission.Admin) {
				c.Next()
				return
			}
		}

		ip := c.ClientIP()
		window := time.Duration(rule.WindowSeconds) * time.Second
		store := ratelimit.Default()

		var retryAfter time.Duration
		var count, limit int
		limited := false
		if rule.LimitPerIp > 0 {
			key := action + ":ip:" + ip
			ok, retry, cnt := store.Allow(key, rule.LimitPerIp, window)
			if !ok {
				limited, retryAfter, count, limit = true, retry, cnt, rule.LimitPerIp
			}
		}
		if !limited && userId != 0 && rule.LimitPerUser > 0 {
			key := action + ":user:" + strconv.FormatUint(userId, 10)
			ok, retry, cnt := store.Allow(key, rule.LimitPerUser, window)
			if !ok {
				limited, retryAfter, count, limit = true, retry, cnt, rule.LimitPerUser
			}
		}

		if limited {
			seconds := int(math.Ceil(retryAfter.Seconds()))
			if seconds < 1 {
				seconds = 1
			}
			slog.Warn("rate_limit_hit",
				"action", action,
				"ip", ip,
				"userId", userId,
				"count", count,
				"limit", limit,
				"window", rule.WindowSeconds,
			)
			c.Header("Retry-After", strconv.Itoa(seconds))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, component.FailDataCode(
				component.MessageRateLimited,
				component.MessageParams{
					"action":            action,
					"retryAfterSeconds": seconds,
				},
			))
			return
		}
		c.Next()
	}
}

func findRateLimitRule(rules []pageConfig.RateLimitRule, action string) *pageConfig.RateLimitRule {
	for i := range rules {
		if rules[i].Action == action {
			return &rules[i]
		}
	}
	return nil
}
