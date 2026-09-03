package oidcservice

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// 响应缺 Content-Type 且有 body 时按首块字节嗅探补头（与 net/http 默认
// 行为一致），使 /api/oauth 表面与是否挂入 gzip 路由组解耦。
func TestDefaultContentTypeWriterFillsMissingHTMLType(t *testing.T) {
	recorder := httptest.NewRecorder()
	handler := withDefaultContentType(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 模拟 zitadel/oidc AuthResponseFormPost：先 WriteHeader 再写 HTML，
		// 全程不声明 Content-Type。
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("<!DOCTYPE html>\n<html><body>form post</body></html>"))
	}))
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/authorize", nil))

	if got := recorder.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Fatalf("Content-Type = %q, want text/html; charset=utf-8", got)
	}
	if !strings.Contains(recorder.Body.String(), "form post") {
		t.Fatalf("body truncated: %q", recorder.Body.String())
	}
}

func TestDefaultContentTypeWriterKeepsExplicitType(t *testing.T) {
	recorder := httptest.NewRecorder()
	handler := withDefaultContentType(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"iss":"x"}`))
	}))
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/.well-known/openid-configuration", nil))

	if got := recorder.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json untouched", got)
	}
}

func TestDefaultContentTypeWriterRedirectUntouched(t *testing.T) {
	recorder := httptest.NewRecorder()
	handler := withDefaultContentType(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "/login")
		w.WriteHeader(http.StatusFound)
	}))
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/authorize", nil))

	if recorder.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302", recorder.Code)
	}
	if got := recorder.Header().Get("Content-Type"); got != "" {
		t.Fatalf("Content-Type = %q, want empty for bodyless redirect", got)
	}
}
