package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/ratelimit"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
	"github.com/gin-gonic/gin"
)

// withUser 在进入被测中间件前注入 userId（模拟已登录请求）。
func withUser(userId uint64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("userId", userId)
		c.Next()
	}
}

func rateLimitRecorder(middlewares ...gin.HandlerFunc) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middlewares...)
	router.POST("/", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/", nil)
	router.ServeHTTP(recorder, request)
	return recorder
}

func TestRateLimitAllowsWithinLimit(t *testing.T) {
	ratelimit.Default().ResetAll()
	for i := 0; i < 3; i++ {
		recorder := rateLimitRecorder(RateLimit(RateLimitTopicWrite))
		if recorder.Code != http.StatusOK {
			t.Fatalf("attempt %d status = %d, want 200", i+1, recorder.Code)
		}
	}
}

func TestRateLimitReturns429WithRetryAfter(t *testing.T) {
	ratelimit.Default().ResetAll()
	// topic.write 默认 limitPerIp=5，第 6 次应 429。
	for i := 0; i < 5; i++ {
		rateLimitRecorder(RateLimit(RateLimitTopicWrite))
	}
	recorder := rateLimitRecorder(RateLimit(RateLimitTopicWrite))
	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429", recorder.Code)
	}
	if retry := recorder.Header().Get("Retry-After"); retry == "" {
		t.Fatal("Retry-After header missing on 429")
	}
	var body component.ResultStruct
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.MessageCode != component.MessageRateLimited {
		t.Fatalf("messageCode = %q, want %q", body.MessageCode, component.MessageRateLimited)
	}
}

func TestRateLimitUserDimension(t *testing.T) {
	ratelimit.Default().ResetAll()
	// topic.write 默认 limitPerUser=1，同用户第 2 次应 429（即使 IP 不同）。
	first := rateLimitRecorder(withUser(1001), RateLimit(RateLimitTopicWrite))
	if first.Code != http.StatusOK {
		t.Fatalf("first status = %d, want 200", first.Code)
	}
	second := rateLimitRecorder(withUser(1001), RateLimit(RateLimitTopicWrite))
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("second status = %d, want 429", second.Code)
	}
	// 不同用户不受影响。
	other := rateLimitRecorder(withUser(1002), RateLimit(RateLimitTopicWrite))
	if other.Code != http.StatusOK {
		t.Fatalf("other user status = %d, want 200", other.Code)
	}
}

func TestRateLimitUnknownActionBypass(t *testing.T) {
	ratelimit.Default().ResetAll()
	recorder := rateLimitRecorder(RateLimit("no-such-action"))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 for unknown action", recorder.Code)
	}
}

func TestRateLimitLLMSFullReturns429(t *testing.T) {
	// 清空计数与配置缓存：保证读取默认 ratelimit.json（llms.full 生效），
	// 并防御同包其他测试（如 rateLimit_hotreload_test）残留的自定义 RateLimitSettings 行。
	ratelimit.Default().ResetAll()
	hotdataserve.ClearRateLimitConfigCache()
	t.Cleanup(func() {
		ratelimit.Default().ResetAll()
		hotdataserve.ClearRateLimitConfigCache()
	})

	// 从配置读取配额，避免硬编码 10 与默认配置脱节。
	limit := rateLimitQuotaFor(RateLimitLLMSFull)
	if limit <= 0 {
		t.Fatalf("llms.full limitPerIp = %d, want > 0", limit)
	}
	for i := 0; i < limit; i++ {
		recorder := rateLimitRecorder(RateLimit(RateLimitLLMSFull))
		if recorder.Code != http.StatusOK {
			t.Fatalf("attempt %d status = %d, want 200", i+1, recorder.Code)
		}
	}
	recorder := rateLimitRecorder(RateLimit(RateLimitLLMSFull))
	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429", recorder.Code)
	}
	if retry := recorder.Header().Get("Retry-After"); retry == "" {
		t.Fatal("Retry-After header missing on 429")
	}
}

func TestWriteActionsReturn429(t *testing.T) {
	for _, action := range []string{RateLimitTopicStatus, RateLimitPostDelete, RateLimitPostUpdate} {
		t.Run(action, func(t *testing.T) {
			ratelimit.Default().ResetAll()
			hotdataserve.ClearRateLimitConfigCache()
			t.Cleanup(func() {
				ratelimit.Default().ResetAll()
				hotdataserve.ClearRateLimitConfigCache()
			})
			limit := rateLimitQuotaFor(action)
			if limit <= 0 {
				t.Fatalf("%s limitPerIp = %d, want > 0", action, limit)
			}
			for range limit {
				if recorder := rateLimitRecorder(RateLimit(action)); recorder.Code != http.StatusOK {
					t.Fatalf("request within %s limit returned %d", action, recorder.Code)
				}
			}
			if recorder := rateLimitRecorder(RateLimit(action)); recorder.Code != http.StatusTooManyRequests {
				t.Fatalf("request beyond %s limit returned %d, want 429", action, recorder.Code)
			}
		})
	}
}

func rateLimitQuotaFor(action string) int {
	cfg := hotdataserve.GetRateLimitConfigCache()
	if rule, ok := cfg.RuleForAction(action); ok {
		return rule.LimitPerIp
	}
	return 0
}

// TestRateLimitCourseSummaryCheckSeparateQuota check 预检（?check=true）走独立
// course.summary.check 配额：即使 course.summary 的 per-User/per-IP 配额已耗尽，
// check 请求仍放行（review P2：浏览 N 门课程页的挂载预检不得耗尽生成配额）。
func TestRateLimitCourseSummaryCheckSeparateQuota(t *testing.T) {
	ratelimit.Default().ResetAll()
	hotdataserve.ClearRateLimitConfigCache()
	t.Cleanup(func() {
		ratelimit.Default().ResetAll()
		hotdataserve.ClearRateLimitConfigCache()
	})

	userLimit := rateLimitUserQuotaFor(RateLimitCourseSummary)
	if userLimit <= 0 {
		t.Fatalf("course.summary limitPerUser = %d, want > 0", userLimit)
	}
	// 以同一用户身份耗尽 course.summary 的 per-User 生成配额。
	for i := 0; i < userLimit; i++ {
		if recorder := rateLimitRecorder(withUser(1001), RateLimitCourseSummaryAware()); recorder.Code != http.StatusOK {
			t.Fatalf("generate attempt %d status = %d, want 200", i+1, recorder.Code)
		}
	}
	// 生成端点已 429（per-User 维度）。
	if recorder := rateLimitRecorder(withUser(1001), RateLimitCourseSummaryAware()); recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("generate beyond user limit status = %d, want 429", recorder.Code)
	}
	// check 预检（?check=1）走独立 course.summary.check 配额，仍应放行。
	// 注意 checkQuery 必须排在 RateLimitCourseSummaryAware 之前，让中间件读到 query。
	checkRecorder := rateLimitRecorder(withUser(1001), checkQuery("1"), RateLimitCourseSummaryAware())
	if checkRecorder.Code != http.StatusOK {
		t.Fatalf("check after generate quota exhausted status = %d, want 200", checkRecorder.Code)
	}
	// 反向验证：check 请求消耗的是独立配额，不消耗生成配额——
	// 生成配额仍满（同一用户再次生成依旧 429）。
	if recorder := rateLimitRecorder(withUser(1001), RateLimitCourseSummaryAware()); recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("generate after check status = %d, want still 429 (check must not consume generate quota)", recorder.Code)
	}
}

func rateLimitUserQuotaFor(action string) int {
	cfg := hotdataserve.GetRateLimitConfigCache()
	if rule, ok := cfg.RuleForAction(action); ok {
		return rule.LimitPerUser
	}
	return 0
}

// checkQuery 给 GET 请求附加 ?check=1（rateLimitRecorder 固定 POST /，这里用
// 一个读 query 的中间件模拟真实路由的 check 分流）。
func checkQuery(value string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request = c.Request.Clone(c.Request.Context())
		c.Request.URL.RawQuery = "check=" + value
		c.Next()
	}
}
