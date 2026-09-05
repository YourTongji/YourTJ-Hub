package httputil

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSetLongPublic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/asset", func(c *gin.Context) {
		SetLongPublic(c)
		c.Status(http.StatusNoContent)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/asset", nil)
	router.ServeHTTP(recorder, request)

	if got := recorder.Header().Get("Cache-Control"); got != LongPublicCacheControl {
		t.Fatalf("Cache-Control = %q, want long public cache", got)
	}
}

// TestDeferCacheHeader 钉住状态感知缓存头决策：2xx 才带成功缓存头，其余状态
// 显式 no-store——静态挂载的缺失文件会经 NoRoute 在同一请求内返回 404，
// 绝不能被钉进长缓存（部署回滚窗口内的缺失 chunk 尤其致命）。
func TestDeferCacheHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		DeferCacheHeader(c, ImmutableCacheControl)
		c.Next()
	})
	router.GET("/ok", func(c *gin.Context) { c.Status(http.StatusOK) })
	router.GET("/gone", func(c *gin.Context) { c.Status(http.StatusNotFound) })
	// 隐式 200：不显式 WriteHeader，首个 Write 时按默认状态 200 决策。
	router.GET("/stream", func(c *gin.Context) {
		_, _ = c.Writer.WriteString("payload")
	})

	cases := []struct {
		path string
		want string
	}{
		{"/ok", ImmutableCacheControl},
		{"/gone", NoStoreCacheControl},
		{"/stream", ImmutableCacheControl},
	}
	for _, tc := range cases {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, tc.path, nil))
		if got := recorder.Header().Get("Cache-Control"); got != tc.want {
			t.Fatalf("GET %s Cache-Control = %q, want %q", tc.path, got, tc.want)
		}
	}
}
