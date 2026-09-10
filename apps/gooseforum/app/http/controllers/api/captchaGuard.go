package api

import (
	"log/slog"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/captchaOpt"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/ratelimit"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
)

// checkCaptchaForRequest 校验验证码答案。
// required=false 时直接放行（开关关闭）。
// 返回值 ok 表示校验通过；needCaptcha 表示需要验证码但请求未携带（前端应弹出验证码）。
// 同时执行提交耗时检测：从签发到提交短于 minSubmitSeconds 视为机器特征。
// c 可能为 nil（单元测试直接调用 controller），此时跳过 IP/用户日志字段。
func checkCaptchaForRequest(c *gin.Context, captchaId, captchaCode string, required bool, minSubmitSeconds int, action string) (ok bool, needCaptcha bool) {
	if !required {
		return true, false
	}
	var ip string
	var userId uint64
	if c != nil {
		ip = c.ClientIP()
		userId = c.GetUint64("userId")
	}
	if captchaId == "" || captchaCode == "" {
		// 触发验证码（前端应弹出）：记录结构化日志，count 为窗口内成功发帖数。
		count := ratelimit.Default().Count(captchaCountKey(action, userId))
		slog.Warn("captcha_triggered", "action", action, "ip", ip, "userId", userId, "count", count)
		return false, true
	}
	if captchaOpt.SubmittedTooFast(captchaId, minSubmitSeconds) {
		slog.Warn("captcha_submit_too_fast", "action", action, "ip", ip, "userId", userId)
		return false, false
	}
	return captchaOpt.VerifyCaptcha(captchaId, captchaCode), false
}

// clientIPOf 返回请求客户端 IP；c 为 nil（单测直调）时返回空串。
func clientIPOf(c *gin.Context) string {
	if c == nil {
		return ""
	}
	return c.ClientIP()
}

// minSubmitSecondsFor 返回当前限流配置中的验证码提交耗时下限。
func minSubmitSecondsFor() int {
	return hotdataserve.GetRateLimitConfigCache().MinSubmitSeconds
}

// 新用户验证码触发：成功发帖的计数窗口（24 小时，与每日上限一致）。
const captchaCountWindow = 24 * time.Hour

// captchaCountKey 返回新用户发帖计数键，动作维度独立。
func captchaCountKey(action string, userId uint64) string {
	return "captcha:" + action + ":user:" + strconv.FormatUint(userId, 10)
}

// newUserCaptchaRequired 判定当前请求是否要求验证码：
// 用户注册未满 newUserCaptchaDays 天（<=0 表示所有用户），且窗口内成功发帖数
// 已达到 newUserCaptchaAfterPosts（<=0 关闭该能力）。
func newUserCaptchaRequired(userCreatedAt time.Time, userId uint64, action string, afterPosts, days int) bool {
	if afterPosts <= 0 {
		return false
	}
	if days > 0 && time.Since(userCreatedAt) >= time.Duration(days)*24*time.Hour {
		return false
	}
	return ratelimit.Default().Count(captchaCountKey(action, userId)) >= afterPosts
}

// recordSuccessfulWrite 记录一次成功发帖（供新用户验证码触发计数）。
func recordSuccessfulWrite(userId uint64, action string) {
	ratelimit.Default().Increment(captchaCountKey(action, userId), captchaCountWindow)
}
