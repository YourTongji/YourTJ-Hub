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

	if got := recorder.Header().Get("Cache-Control"); got != "public, max-age=18144000" {
		t.Fatalf("Cache-Control = %q, want long public cache", got)
	}
}

func TestSetImmutable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/asset", func(c *gin.Context) {
		SetImmutable(c)
		c.Status(http.StatusNoContent)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/asset", nil)
	router.ServeHTTP(recorder, request)

	if got := recorder.Header().Get("Cache-Control"); got != "public, max-age=31536000, immutable" {
		t.Fatalf("Cache-Control = %q, want immutable long cache", got)
	}
}
