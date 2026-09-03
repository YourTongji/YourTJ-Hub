package forum

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSchedulePageRequestReturnsPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/schedule", Schedule)

	req := httptest.NewRequest(http.MethodGet, "/schedule", nil)
	req.Header.Set("X-Goose-Page", "true")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	if recorder.Header().Get("Content-Type") == "" {
		t.Fatal("expected JSON content type")
	}
	if got := recorder.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("expected page payload response to avoid browser cache, got %q", got)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, `"component":"course.schedule"`) {
		t.Fatalf("expected course.schedule component in payload: %s", body)
	}
	// 排课器专属 meta：title 与 description（含站点名插值，不残留 {site} 占位符）。
	if !strings.Contains(body, `"title":"排课器`) {
		t.Fatalf("expected schedule page title in payload: %s", body)
	}
	if strings.Contains(body, "meta.scheduleDesc") || strings.Contains(body, "{site}") {
		t.Fatalf("expected interpolated meta description, got raw placeholder: %s", body)
	}
	// 节次作息表：SSR 注入 props.sectionTimes，未保存配置时回默认 12 节作息。
	var payload struct {
		Props struct {
			SectionTimes []struct {
				Section int    `json:"section"`
				Start   string `json:"start"`
				End     string `json:"end"`
			} `json:"sectionTimes"`
		} `json:"props"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode schedule payload: %v", err)
	}
	if len(payload.Props.SectionTimes) != 12 {
		t.Fatalf("expected 12 default section times, got %d: %s", len(payload.Props.SectionTimes), body)
	}
	if first := payload.Props.SectionTimes[0]; first.Section != 1 || first.Start != "08:00" || first.End != "08:45" {
		t.Fatalf("expected default first section 1 08:00-08:45, got %#v", first)
	}
	if last := payload.Props.SectionTimes[11]; last.Section != 12 || last.Start != "20:10" || last.End != "20:55" {
		t.Fatalf("expected default last section 12 20:10-20:55, got %#v", last)
	}
}

func TestScheduleHTMLReturnsNoJSContent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/schedule", Schedule)

	req := httptest.NewRequest(http.MethodGet, "/schedule", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	body := recorder.Body.String()
	if !strings.Contains(body, `id="goose-app"`) {
		t.Fatalf("expected app mount point in HTML: %s", body)
	}
	if !strings.Contains(body, `id="goose-payload"`) {
		t.Fatalf("expected initial payload in HTML: %s", body)
	}
	// HTML 文档同样必须禁用缓存：goose-payload 内嵌节次作息等管理端配置，
	// bfcache/启发式缓存会把保存后的新作息继续以旧 DOM 呈现（作息不同步根因）。
	if got := recorder.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("expected HTML document to avoid browser/bfcache, got Cache-Control %q", got)
	}
}

func TestScheduleHTMLCacheControlNoStore(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/schedule", Schedule)

	req := httptest.NewRequest(http.MethodGet, "/schedule", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	if got := recorder.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("HTML /schedule Cache-Control = %q, want no-store（内嵌配置防 bfcache 陈旧）", got)
	}
}
