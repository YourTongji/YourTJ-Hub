package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/jsonopt"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/ratelimit"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/middleware"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
	"github.com/gin-gonic/gin"
)

func TestHomeMarkdownSharesLLMSIndexRateLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(&pageConfig.Entity{}, &topics.Entity{}, &posts.Entity{}); err != nil {
		t.Fatalf("migrate home markdown tables: %v", err)
	}
	restoreLLMSRouteSettings(t, conn)
	persistLLMSRouteSettings(t, conn, pageConfig.LLMSConfig{Enabled: true})
	setLLMSIndexRateLimit(t, 1)

	router := gin.New()
	router.GET("/", serveHome)

	first := requestHome(router, "text/markdown")
	if first.Code != http.StatusOK {
		t.Fatalf("first markdown status = %d, want 200: %s", first.Code, first.Body.String())
	}
	second := requestHome(router, "text/markdown")
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("second markdown status = %d, want 429", second.Code)
	}
	html := requestHome(router, "")
	if html.Code != http.StatusOK {
		t.Fatalf("HTML status = %d, want 200 after the markdown quota is spent", html.Code)
	}

	persistLLMSRouteSettings(t, conn, pageConfig.LLMSConfig{})
	disabled := requestHome(router, "text/markdown")
	if disabled.Code != http.StatusOK {
		t.Fatalf("disabled-index markdown status = %d, want 200 without the index quota", disabled.Code)
	}
}

func requestHome(router http.Handler, accept string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "http://localhost/", nil)
	if accept != "" {
		req.Header.Set("Accept", accept)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func setLLMSIndexRateLimit(t *testing.T, limitPerIp int) {
	t.Helper()
	conn := dbconnect.Connect()
	var prev pageConfig.Entity
	hasPrev := conn.Where("page_type = ?", pageConfig.RateLimitSettings).First(&prev).Error == nil
	cfg := pageConfig.RateLimitConfig{
		Enabled: true,
		Actions: []pageConfig.RateLimitRule{
			{Action: middleware.RateLimitLLMSIndex, WindowSeconds: 60, LimitPerIp: limitPerIp},
		},
	}
	conn.Where("page_type = ?", pageConfig.RateLimitSettings).Delete(&pageConfig.Entity{})
	if err := conn.Create(&pageConfig.Entity{PageType: pageConfig.RateLimitSettings, Config: jsonopt.Encode(cfg)}).Error; err != nil {
		t.Fatalf("write llms.index rate limit: %v", err)
	}
	hotdataserve.ClearRateLimitConfigCache()
	ratelimit.Default().ResetAll()
	t.Cleanup(func() {
		conn.Where("page_type = ?", pageConfig.RateLimitSettings).Delete(&pageConfig.Entity{})
		if hasPrev {
			conn.Create(&prev)
		}
		hotdataserve.ClearRateLimitConfigCache()
		ratelimit.Default().ResetAll()
	})
}
