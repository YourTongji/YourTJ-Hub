package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/leancodebox/GooseForum/app/bundles/ratelimit"
	"github.com/leancodebox/GooseForum/app/http/controllers/component"
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
