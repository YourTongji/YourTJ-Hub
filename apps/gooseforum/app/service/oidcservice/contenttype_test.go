package oidcservice

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// eagerHeaderSnapshotWriter 模拟真实 net/http 的响应头快照语义：handler
// 触碰过 Header 后调用 WriteHeader 的瞬间，响应头即被克隆提交，其后的修改
// 不再生效。httptest.ResponseRecorder 的 Header() 始终返回同一 map，会掩盖
// 这类时序问题（#441 review 指出），因此用本 writer 做时序验证。
type eagerHeaderSnapshotWriter struct {
	header http.Header
	sent   http.Header
	code   int
	body   strings.Builder
}

func newEagerHeaderSnapshotWriter() *eagerHeaderSnapshotWriter {
	return &eagerHeaderSnapshotWriter{header: http.Header{}}
}

func (w *eagerHeaderSnapshotWriter) Header() http.Header { return w.header }

func (w *eagerHeaderSnapshotWriter) WriteHeader(code int) {
	if w.sent != nil {
		return
	}
	w.code = code
	w.sent = w.header.Clone()
}

func (w *eagerHeaderSnapshotWriter) Write(p []byte) (int, error) {
	if w.sent == nil {
		w.WriteHeader(http.StatusOK)
	}
	return w.body.Write(p)
}

// 带 body 的 HTML 响应必须自带 Content-Type：即便 handler 按zitadel/oidc
// AuthResponseFormPost 的顺序先 WriteHeader(200) 再写模板（且此前设置过
// Cache-Control），默认头也已在进入下游前就位，经快照提交后依然存在。
func TestAuthorizeDefaultContentTypeSurvivesHeaderSnapshot(t *testing.T) {
	handler := withDefaultHTMLContentType(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("<!DOCTYPE html>\n<html><body>form post</body></html>"))
	}))
	eager := newEagerHeaderSnapshotWriter()
	handler.ServeHTTP(eager, httptest.NewRequest(http.MethodGet, "/authorize", nil))

	if got := eager.sent.Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Fatalf("committed Content-Type = %q, want text/html; charset=utf-8", got)
	}
	if got := eager.sent.Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", got)
	}
	if !strings.Contains(eager.body.String(), "form post") {
		t.Fatalf("body truncated: %q", eager.body.String())
	}
}

// 下游显式声明的 Content-Type 覆盖默认值。
func TestAuthorizeDefaultContentTypeDefersToExplicit(t *testing.T) {
	recorder := httptest.NewRecorder()
	handler := withDefaultHTMLContentType(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"iss":"x"}`))
	}))
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/authorize", nil))

	if got := recorder.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json untouched", got)
	}
}

// 未登录 302 分支携带默认头：与 net/http http.Redirect 对 GET 重定向显式
// 写 text/html 的标准行为一致，无 body 时客户端不渲染该头。
func TestAuthorizeDefaultContentTypeOnRedirect(t *testing.T) {
	recorder := httptest.NewRecorder()
	handler := withDefaultHTMLContentType(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "/login")
		w.WriteHeader(http.StatusFound)
	}))
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/authorize", nil))

	if recorder.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302", recorder.Code)
	}
	if got := recorder.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Fatalf("Content-Type = %q, want text/html; charset=utf-8", got)
	}
}
