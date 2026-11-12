package controllers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRobotsTxtContentSignalKeepsExistingRules(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/robots.txt", RenderRobotsTxt)

	req := httptest.NewRequest(http.MethodGet, "http://localhost/robots.txt", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/plain") {
		t.Fatalf("content-type = %q, want text/plain", got)
	}
	body := rec.Body.String()
	for _, line := range []string{
		"User-agent: *",
		"Content-Signal: search=yes, ai-input=yes, ai-train=no",
		"Allow: /",
		"Sitemap: http://localhost/sitemap.xml",
		"Sitemap: http://localhost/rss.xml",
		"Disallow: /api/",
		"Disallow: /admin",
		"Disallow: /login",
		"Disallow: /publish",
	} {
		if !strings.Contains(body, line) {
			t.Fatalf("robots.txt missing %q:\n%s", line, body)
		}
	}
}
