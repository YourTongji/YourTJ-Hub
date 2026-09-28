package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/ratelimit"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
	"github.com/gin-gonic/gin"
)

func TestChatForwardChargesSharedSendQuotaAndRestoresBody(t *testing.T) {
	ratelimit.Default().ResetAll()
	hotdataserve.ClearRateLimitConfigCache()
	t.Cleanup(func() { ratelimit.Default().ResetAll(); hotdataserve.ClearRateLimitConfigCache() })
	quota := rateLimitQuotaFor(RateLimitMessageSend)
	if quota < 3 {
		t.Fatalf("expected normal chat quota >=3, got %d", quota)
	}
	router := gin.New()
	body := `{"mode":"individual","messageIds":[1,2,3]}`
	router.POST("/forward", RateLimitChatForward(), func(c *gin.Context) {
		raw, _ := io.ReadAll(c.Request.Body)
		if string(raw) != body {
			t.Error("request body not restored")
		}
		c.Status(200)
	})
	router.POST("/send", RateLimit(RateLimitMessageSend), func(c *gin.Context) { c.Status(200) })
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/forward", strings.NewReader(body)))
	if rec.Code != 200 {
		t.Fatalf("forward status %d", rec.Code)
	}
	for range quota - 3 {
		rec = httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/send", nil))
		if rec.Code != 200 {
			t.Fatal("premature throttle")
		}
	}
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/send", nil))
	if rec.Code != 429 {
		t.Fatalf("forward bypassed ordinary send quota: %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/forward", strings.NewReader(strings.Repeat("x", 8193))))
	if rec.Code != 400 {
		t.Fatalf("oversized forward body status %d", rec.Code)
	}
}
