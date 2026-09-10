package forum

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
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
	// 节次作息表：SSR 注入 props.sectionTimes，未保存配置时回默认 11 节作息（现行 11 节制）。
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
	if len(payload.Props.SectionTimes) != 11 {
		t.Fatalf("expected 11 default section times, got %d: %s", len(payload.Props.SectionTimes), body)
	}
	if first := payload.Props.SectionTimes[0]; first.Section != 1 || first.Start != "08:00" || first.End != "08:45" {
		t.Fatalf("expected default first section 1 08:00-08:45, got %#v", first)
	}
	if evening := payload.Props.SectionTimes[8]; evening.Section != 9 || evening.Start != "18:30" || evening.End != "19:15" {
		t.Fatalf("expected default evening section 9 18:30-19:15, got %#v", evening)
	}
	if last := payload.Props.SectionTimes[10]; last.Section != 11 || last.Start != "20:10" || last.End != "20:55" {
		t.Fatalf("expected default last section 11 20:10-20:55, got %#v", last)
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

// 存量旧 12 节制配置（PR #496 之前保存的默认表，第 9 节 17:10-17:55）在读取时
// 必须归一为现行 11 节语义：SSR 注入的 sectionTimes 晚间应为 18:30/19:20/20:10，
// 且管理端回显正确值，保存后存储自愈（review P1，issue #478）。
func TestSchedulePageRequestNormalizesLegacyStoredSectionTimes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	legacy := `{"sectionTimes":[` +
		`{"section":1,"start":"08:00","end":"08:45"},{"section":2,"start":"08:50","end":"09:35"},` +
		`{"section":3,"start":"10:00","end":"10:45"},{"section":4,"start":"10:50","end":"11:35"},` +
		`{"section":5,"start":"13:30","end":"14:15"},{"section":6,"start":"14:20","end":"15:05"},` +
		`{"section":7,"start":"15:30","end":"16:15"},{"section":8,"start":"16:20","end":"17:05"},` +
		`{"section":9,"start":"17:10","end":"17:55"},{"section":10,"start":"18:30","end":"19:15"},` +
		`{"section":11,"start":"19:20","end":"20:05"},{"section":12,"start":"20:10","end":"20:55"}]}`
	conn := db.Connect()
	if err := conn.AutoMigrate(&pageConfig.Entity{}); err != nil {
		t.Fatalf("migrate page_config table: %v", err)
	}
	if err := conn.Where("page_type = ?", pageConfig.ScheduleSettings).Delete(&pageConfig.Entity{}).Error; err != nil {
		t.Fatalf("clear schedule settings row: %v", err)
	}
	if err := conn.Create(&pageConfig.Entity{PageType: pageConfig.ScheduleSettings, Config: legacy}).Error; err != nil {
		t.Fatalf("seed legacy schedule settings row: %v", err)
	}
	t.Cleanup(func() {
		conn.Where("page_type = ?", pageConfig.ScheduleSettings).Delete(&pageConfig.Entity{})
	})

	router := gin.New()
	router.GET("/schedule", Schedule)
	req := httptest.NewRequest(http.MethodGet, "/schedule", nil)
	req.Header.Set("X-Goose-Page", "true")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}

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
	times := payload.Props.SectionTimes
	if len(times) != 11 {
		t.Fatalf("expected 11 normalized section times, got %d: %s", len(times), recorder.Body.String())
	}
	want := map[int][2]string{8: {"16:20", "17:05"}, 9: {"18:30", "19:15"}, 10: {"19:20", "20:05"}, 11: {"20:10", "20:55"}}
	bySection := map[int][2]string{}
	for _, item := range times {
		bySection[item.Section] = [2]string{item.Start, item.End}
		if item.Start == "17:10" {
			t.Fatalf("legacy 17:10 row leaked into SSR props: %#v", times)
		}
	}
	for section, expect := range want {
		if bySection[section] != expect {
			t.Fatalf("section %d = %v, want %v (all: %#v)", section, bySection[section], expect, times)
		}
	}
}
