package middleware

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/networkAccessLog"
	"github.com/gin-gonic/gin"
)

func TestCallbackQueriesAreExcludedFromAllAccessLogs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	previousAccessLog := preferences.GetBool("server.accessLog", false)
	t.Cleanup(func() { preferences.Set("server.accessLog", previousAccessLog) })
	preferences.Set("server.accessLog", true)
	conn := db.Connect()
	if err := conn.AutoMigrate(&networkAccessLog.Entity{}); err != nil {
		t.Fatalf("migrate access log: %v", err)
	}
	var output bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&output, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })

	router := gin.New()
	router.Use(Recovery(), AccessLog)
	router.Any("/*path", func(c *gin.Context) {
		if c.Query("panic") == "yes" {
			panic("fake callback panic")
		}
		c.Status(http.StatusNoContent)
	})

	requests := []string{
		"/api/campus/tongji/callback?code=fake-campus-code&state=fake-campus-state",
		"/api/campus/tongji/callback/?code=fake-campus-slash-code&state=fake-campus-slash-state",
		"/api/auth/future-provider/callback?code=fake-goth-code&state=fake-goth-state",
		"/api/oauth/authorize/callback?id=fake-oidc-id",
		"/login?redirect=%2Fapi%2Foauth%2Fauthorize%2Fcallback%3Fid%3Dfake-redirect-id",
	}
	callbackPaths := []string{"/api/campus/tongji/callback", "/api/campus/tongji/callback/", "/api/auth/future-provider/callback", "/api/oauth/authorize/callback", "/login"}
	if err := conn.Where("path IN ?", callbackPaths).Delete(&networkAccessLog.Entity{}).Error; err != nil {
		t.Fatalf("clear access log fixtures: %v", err)
	}
	for _, target := range requests {
		req := httptest.NewRequest(http.MethodGet, target, nil)
		normal := httptest.NewRecorder()
		router.ServeHTTP(normal, req)
		if normal.Code != http.StatusNoContent {
			t.Errorf("normal request %q status = %d, want 204", target, normal.Code)
		}
		panicReq := httptest.NewRequest(http.MethodGet, target+"&panic=yes", nil)
		panicReq.Header.Set("Referer", "https://forum.test/api/oauth/authorize/callback?id=fake-referer-id")
		panicked := httptest.NewRecorder()
		router.ServeHTTP(panicked, panicReq)
		if panicked.Code != http.StatusInternalServerError {
			t.Errorf("panic request %q status = %d, want 500", target, panicked.Code)
		}
	}
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/search?q=hello", nil))

	ordinaryPanic := httptest.NewRequest(http.MethodGet, "/search?panic=yes&q=diagnostic-panic-fixture", nil)
	ordinaryPanic.Header.Set("Referer", "https://forum.test/referrer-fixture?q=diagnostic-referer-fixture")
	panicked := httptest.NewRecorder()
	router.ServeHTTP(panicked, ordinaryPanic)
	if panicked.Code != http.StatusInternalServerError {
		t.Fatalf("ordinary panic status = %d, want 500", panicked.Code)
	}

	var rows []networkAccessLog.Entity
	if err := conn.Where("path IN ?", callbackPaths).Find(&rows).Error; err != nil {
		t.Fatalf("read access logs: %v", err)
	}
	if len(rows) != len(requests) {
		t.Fatalf("network access log rows = %d, want %d callback requests", len(rows), len(requests))
	}
	for _, secret := range []string{"fake-campus-code", "fake-campus-state", "fake-campus-slash-code", "fake-campus-slash-state", "fake-goth-code", "fake-goth-state", "fake-oidc-id", "fake-redirect-id", "fake-referer-id"} {
		if strings.Contains(output.String(), secret) {
			t.Errorf("slog output contains %q", secret)
		}
		for _, row := range rows {
			if strings.Contains(row.Path, secret) {
				t.Errorf("network access log contains %q in %q", secret, row.Path)
			}
		}
	}
	for _, row := range rows {
		if strings.Contains(row.Path, "?") {
			t.Errorf("callback query retained in network access log path %q", row.Path)
		}
	}
	if strings.Contains(output.String(), "q=hello") || strings.Contains(output.String(), "diagnostic-panic-fixture") || strings.Contains(output.String(), "diagnostic-referer-fixture") || strings.Contains(output.String(), "referrer-fixture") {
		t.Error("ordinary query retained in diagnostic output")
	}
	var ordinary networkAccessLog.Entity
	if err := conn.Where("path = ?", "/search").Last(&ordinary).Error; err != nil {
		t.Fatalf("ordinary access log missing: %v", err)
	}
	if ordinary.Path != "/search" {
		t.Errorf("ordinary path = %q, want path only", ordinary.Path)
	}
}

func TestCallbackQueryRedaction(t *testing.T) {
	for _, raw := range []string{
		"/api/campus/tongji/callback?code=fake",
		"/api/campus/tongji/callback/?code=fake",
		"/api/auth/new-provider/callback?code=fake&state=fake",
		"/api/auth/new-provider/callback/?code=fake&state=fake",
		"/api/oauth/authorize/callback?id=fake",
		"/api/oauth/authorize/callback/?id=fake",
		"/login?redirect=%2Fapi%2Foauth%2Fauthorize%2Fcallback%3Fid%3Dfake",
		// Malformed redirect still embeds an opaque continuation: fail closed.
		"/login?redirect=%2Fapi%2Foauth%2Fauthorize%2Fcallback%3Fid%3Dfake%zz",
	} {
		r := httptest.NewRequest(http.MethodGet, raw, nil)
		if !ShouldRedactQuery(r.URL) {
			t.Errorf("ShouldRedactQuery(%q) = false", raw)
		}
	}
	ordinaryReferer := "https://forum.test/search?q=hello"
	if got := logReferer(ordinaryReferer); got != "https://forum.test" {
		t.Errorf("referrer = %q, want origin only", got)
	}
}
