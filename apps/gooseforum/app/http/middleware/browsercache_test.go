package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/gin-gonic/gin"
)

func TestBrowserCacheProduction(t *testing.T) {
	oldEnv := preferences.GetString("app.env", "production")
	t.Cleanup(func() {
		preferences.Set("app.env", oldEnv)
	})

	preferences.Set("app.env", "production")
	recorder := requestWithMiddleware(BrowserCache, http.MethodGet)

	if got := recorder.Header().Get("Cache-Control"); got != "public, max-age=18144000" {
		t.Fatalf("Cache-Control = %q, want long public cache", got)
	}
}

func TestBrowserCacheLocal(t *testing.T) {
	oldEnv := preferences.GetString("app.env", "production")
	t.Cleanup(func() {
		preferences.Set("app.env", oldEnv)
	})

	preferences.Set("app.env", "local")
	recorder := requestWithMiddleware(BrowserCache, http.MethodGet)

	if got := recorder.Header().Get("Cache-Control"); got != "" {
		t.Fatalf("Cache-Control = %q, want empty local cache header", got)
	}
}

func requestWithMiddleware(middleware gin.HandlerFunc, method string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware)
	router.Handle(method, "/", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(method, "/", nil)
	router.ServeHTTP(recorder, request)
	return recorder
}

func TestAssetsCacheProduction(t *testing.T) {
	oldEnv := preferences.GetString("app.env", "production")
	t.Cleanup(func() {
		preferences.Set("app.env", oldEnv)
	})

	preferences.Set("app.env", "production")
	recorder := requestWithMiddleware(AssetsCache, http.MethodGet)

	if got := recorder.Header().Get("Cache-Control"); got != "public, max-age=31536000, immutable" {
		t.Fatalf("Cache-Control = %q, want immutable long cache", got)
	}
}

func TestAssetsCacheLocal(t *testing.T) {
	oldEnv := preferences.GetString("app.env", "production")
	t.Cleanup(func() {
		preferences.Set("app.env", oldEnv)
	})

	preferences.Set("app.env", "local")
	recorder := requestWithMiddleware(AssetsCache, http.MethodGet)

	if got := recorder.Header().Get("Cache-Control"); got != "" {
		t.Fatalf("Cache-Control = %q, want empty local cache header", got)
	}
}

func TestAssetsCacheSkipsErrorResponses(t *testing.T) {
	oldEnv := preferences.GetString("app.env", "production")
	t.Cleanup(func() {
		preferences.Set("app.env", oldEnv)
	})

	preferences.Set("app.env", "production")
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(AssetsCache)
	router.GET("/ok", func(c *gin.Context) { c.Status(http.StatusOK) })
	router.GET("/gone", func(c *gin.Context) { c.Status(http.StatusNotFound) })

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/ok", nil))
	if got := recorder.Header().Get("Cache-Control"); got != "public, max-age=31536000, immutable" {
		t.Fatalf("GET /ok Cache-Control = %q, want immutable long cache", got)
	}

	// 缺失 chunk 的 404 绝不能被钉进缓存（部署回滚窗口内浏览器/共享缓存
	// 会把失败响应按 immutable 缓存一年）：错误响应必须显式 no-store。
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/gone", nil))
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("GET /gone status = %d, want 404", recorder.Code)
	}
	if got := recorder.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("GET /gone Cache-Control = %q, want no-store", got)
	}
}

func TestBrowserCacheSkipsErrorResponses(t *testing.T) {
	oldEnv := preferences.GetString("app.env", "production")
	t.Cleanup(func() {
		preferences.Set("app.env", oldEnv)
	})

	preferences.Set("app.env", "production")
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(BrowserCache)
	router.GET("/ok", func(c *gin.Context) { c.Status(http.StatusOK) })
	router.GET("/gone", func(c *gin.Context) { c.Status(http.StatusNotFound) })

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/ok", nil))
	if got := recorder.Header().Get("Cache-Control"); got != "public, max-age=18144000" {
		t.Fatalf("GET /ok Cache-Control = %q, want long public cache", got)
	}

	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/gone", nil))
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("GET /gone status = %d, want 404", recorder.Code)
	}
	if got := recorder.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("GET /gone Cache-Control = %q, want no-store", got)
	}
}
