package routes

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leancodebox/GooseForum/app/bundles/connect/dbconnect"
	"github.com/leancodebox/GooseForum/app/bundles/ratelimit"
	"github.com/leancodebox/GooseForum/app/models/defaultconfig"
	"github.com/leancodebox/GooseForum/app/models/forum/pageConfig"
	"github.com/leancodebox/GooseForum/app/models/forum/posts"
	"github.com/leancodebox/GooseForum/app/models/forum/topics"
	"github.com/leancodebox/GooseForum/app/models/hotdataserve"
	"github.com/leancodebox/GooseForum/app/service/llmsservice"
	"gorm.io/gorm"
)

func TestLLMSRoutesRespectFeatureGatesAndContentTypes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// 清空限流计数与配置缓存，确保本测试使用默认 ratelimit.json（llms.* 配额宽松），
	// 不受其他测试持久化 RateLimitSettings 的影响，也避免共享 IP 窗口残留计数误触发 429。
	ratelimit.Default().ResetAll()
	hotdataserve.ClearRateLimitConfigCache()
	t.Cleanup(func() {
		ratelimit.Default().ResetAll()
		hotdataserve.ClearRateLimitConfigCache()
	})
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(&pageConfig.Entity{}, &topics.Entity{}, &posts.Entity{}); err != nil {
		t.Fatalf("migrate llms route tables: %v", err)
	}
	restoreLLMSRouteSettings(t, conn)

	base := uint64(time.Now().UnixNano()%1_000_000_000) + 8_100_000_000
	topicID := base + 1
	postID := base + 2
	t.Cleanup(func() {
		conn.Unscoped().Delete(&posts.Entity{}, postID)
		conn.Unscoped().Delete(&topics.Entity{}, topicID)
		llmsservice.ClearCache()
	})
	now := time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)
	if err := conn.Create(&posts.Entity{Id: postID, TopicId: topicID, PostNo: 1, Content: "route body", CreatedAt: now}).Error; err != nil {
		t.Fatalf("create route post: %v", err)
	}
	if err := conn.Create(&topics.Entity{Id: topicID, Title: "Route topic", FirstPostId: postID, Status: 1, ProcessStatus: topics.ProcessStatusNormal, Excerpt: "route excerpt", CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatalf("create route topic: %v", err)
	}

	router := gin.New()
	siteInfoRoute(router)

	persistLLMSRouteSettings(t, conn, pageConfig.LLMSConfig{})
	for _, path := range []string{"/llms.txt", "/llms-full.txt", fmt.Sprintf("/p/posts/%d.md", topicID)} {
		response := performLLMSRouteRequest(router, path)
		if response.Code != http.StatusNotFound {
			t.Fatalf("GET %s status=%d, want 404 while disabled", path, response.Code)
		}
	}

	persistLLMSRouteSettings(t, conn, pageConfig.LLMSConfig{Enabled: true, FullText: true, Files: true})
	cases := []struct {
		path        string
		contentType string
		body        string
	}{
		{path: "/llms.txt", contentType: "text/plain; charset=utf-8", body: "Route topic"},
		{path: "/llms-full.txt", contentType: "text/plain; charset=utf-8", body: "route body"},
		{path: fmt.Sprintf("/p/posts/%d.md", topicID), contentType: "text/markdown; charset=utf-8", body: "# Route topic"},
	}
	for _, testCase := range cases {
		response := performLLMSRouteRequest(router, testCase.path)
		if response.Code != http.StatusOK {
			t.Fatalf("GET %s status=%d body=%q, want 200", testCase.path, response.Code, response.Body.String())
		}
		if got := response.Header().Get("Content-Type"); got != testCase.contentType {
			t.Fatalf("GET %s content-type=%q, want %q", testCase.path, got, testCase.contentType)
		}
		if got := response.Header().Get("Cache-Control"); got != "public, max-age=10" {
			t.Fatalf("GET %s cache-control=%q", testCase.path, got)
		}
		if got := response.Header().Get("Vary"); got != "Host" {
			t.Fatalf("GET %s vary=%q, want Host", testCase.path, got)
		}
		if got := response.Header().Get("X-Content-Type-Options"); got != "nosniff" {
			t.Fatalf("GET %s x-content-type-options=%q, want nosniff", testCase.path, got)
		}
		if !strings.Contains(response.Body.String(), testCase.body) {
			t.Fatalf("GET %s body=%q, want fragment %q", testCase.path, response.Body.String(), testCase.body)
		}
	}

	missing := performLLMSRouteRequest(router, "/p/posts/999999999.md")
	if missing.Code != http.StatusNotFound {
		t.Fatalf("missing markdown route status=%d, want 404", missing.Code)
	}
}

func performLLMSRouteRequest(router http.Handler, path string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "http://localhost"+path, nil)
	router.ServeHTTP(recorder, request)
	return recorder
}

func restoreLLMSRouteSettings(t *testing.T, conn *gorm.DB) {
	t.Helper()
	var previous pageConfig.Entity
	result := conn.Where("page_type = ?", pageConfig.PostingSettings).First(&previous)
	hadPrevious := result.Error == nil
	if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		t.Fatalf("read existing posting settings: %v", result.Error)
	}
	t.Cleanup(func() {
		if hadPrevious {
			if err := conn.Save(&previous).Error; err != nil {
				t.Errorf("restore posting settings: %v", err)
			}
		} else if err := conn.Where("page_type = ?", pageConfig.PostingSettings).Delete(&pageConfig.Entity{}).Error; err != nil {
			t.Errorf("delete posting settings fixture: %v", err)
		}
		hotdataserve.ClearPostingSettingsConfigCache()
		llmsservice.ClearCache()
	})
}

func persistLLMSRouteSettings(t *testing.T, conn *gorm.DB, llms pageConfig.LLMSConfig) {
	t.Helper()
	config := defaultconfig.GetDefaultPostingSettingsConfig()
	config.LLMS = llms
	encoded, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("encode posting settings: %v", err)
	}
	entity := pageConfig.Entity{PageType: pageConfig.PostingSettings, Config: string(encoded)}
	if err := conn.Where("page_type = ?", pageConfig.PostingSettings).Assign(entity).FirstOrCreate(&entity).Error; err != nil {
		t.Fatalf("save posting settings: %v", err)
	}
	hotdataserve.ClearPostingSettingsConfigCache()
	llmsservice.ClearCache()
}
