package middleware

import (
	"log/slog"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leancodebox/GooseForum/app/bundles/ratelimit"
	"github.com/leancodebox/GooseForum/app/http/controllers/component"
	"github.com/leancodebox/GooseForum/app/models/forum/pageConfig"
	"github.com/leancodebox/GooseForum/app/models/hotdataserve"
	"github.com/leancodebox/GooseForum/app/service/permission"
	"github.com/leancodebox/GooseForum/app/service/userservice"
)

// 限流动作标识，与管理面板 rateLimitSettings 的 actions.action 对应。
const (
	RateLimitRegister       = "register"
	RateLimitLogin          = "login"
	RateLimitOIDCAuthorize  = "oidc.authorize"
	RateLimitOIDCToken      = "oidc.token"
	RateLimitForgotPassword = "forgot-password"
	RateLimitEmailChange    = "email.change"
	RateLimitPasswordChange = "password.change"
	RateLimitTopicWrite     = "topic.write"
	RateLimitTopicStatus    = "topic.status"
	RateLimitPostCreate     = "post.create"
	RateLimitPostDelete     = "post.delete"
	RateLimitMessageSend    = "message.send"
	RateLimitUpload         = "upload"
	RateLimitInteract       = "interact"
	RateLimitLLMSIndex      = "llms.index"
	RateLimitLLMSFull       = "llms.full"
	RateLimitLLMSTopic      = "llms.topic"
	RateLimitMCPAuth        = "mcp.auth"
)

// RateLimit 按动作限流：同时检查 IP 与用户双维度，任一超限返回 429。
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
