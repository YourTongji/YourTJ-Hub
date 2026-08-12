package forum

import (
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
}

func TestScheduleHTMLReturnsNoJSContent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/schedule", Schedule)

	req := httptest.NewRequest(http.MethodGet, "/schedule", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	body := recorder.Body.String()
	if !strings.Contains(body, `id="goose-app"`) {
		t.Fatalf("expected app mount point in HTML: %s", body)
	}
	if !strings.Contains(body, `id="goose-payload"`) {
		t.Fatalf("expected initial payload in HTML: %s", body)
	}
}
