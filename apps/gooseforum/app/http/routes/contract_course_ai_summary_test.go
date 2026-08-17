package routes

import (
	"net/http"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/forum"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/course"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// setupCourseSummaryContractTest 迁移 AI 总结契约测试表并开启功能开关。
// 独立 setup（不复用 setupCourseContractTest）避免清空 page_config 全表影响其他契约测试。
func setupCourseSummaryContractTest(t *testing.T) (*gorm.DB, *gin.Engine) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	conn := dbconnect.Connect()
	models := []any{
		&course.Entity{},
		&course.TermEntity{},
		&course.OfferingEntity{},
		&course.ReviewEntity{},
		&course.CourseAiSummaryEntity{},
		&pageConfig.Entity{},
	}
	if err := conn.AutoMigrate(models...); err != nil {
		t.Fatalf("migrate course summary contract tables: %v", err)
	}
	for _, m := range models {
		if err := conn.Unscoped().Where("1 = 1").Delete(m).Error; err != nil {
			t.Fatalf("clean course summary contract table: %v", err)
		}
	}
	// 仅清理并重写 AI 总结开关（不动其它 page_config 行）。
	if err := conn.Where("page_type = ?", pageConfig.AiSummarySettings).Delete(&pageConfig.Entity{}).Error; err != nil {
		t.Fatalf("clean ai summary config: %v", err)
	}
	upsertAiSummaryContractConfig(t, conn, `{"enabled":true,"globalPerMinute":100}`)

	router := gin.New()
	router.GET("/api/forum/courses/:courseId/summary", UpUriQueryReq(forum.GetCourseSummary))
	return conn, router
}

// upsertAiSummaryContractConfig 写 AI 总结开关配置（复用 page_type 唯一行）并清缓存。
func upsertAiSummaryContractConfig(t *testing.T, conn *gorm.DB, configJSON string) {
	t.Helper()
	entity := pageConfig.GetByPageType(pageConfig.AiSummarySettings)
	entity.PageType = pageConfig.AiSummarySettings
	entity.Config = configJSON
	if err := conn.Save(&entity).Error; err != nil {
		t.Fatalf("save ai summary config: %v", err)
	}
	hotdataserve.ClearAiSummarySettingsConfigCache()
}

// seedCourseSummaryContractCourse 写入与 course-summary fixtures 一致的课程 42 + offering 901。
func seedCourseSummaryContractCourse(t *testing.T, conn *gorm.DB) {
	t.Helper()
	if err := conn.Create(&course.Entity{
		Id:             42,
		PrimaryCode:    "100001",
		Name:           "高等数学(A)上",
		Department:     "数学科学学院",
		NormalizedName: "高等数学a上",
		Status:         course.StatusVisible,
	}).Error; err != nil {
		t.Fatalf("create contract course: %v", err)
	}
	if err := conn.Create(&course.TermEntity{Id: 101, Code: "2025-2026-1", Name: "2025-2026 第一学期", Status: 0}).Error; err != nil {
		t.Fatalf("create contract term: %v", err)
	}
	if err := conn.Create(&course.OfferingEntity{Id: 901, CourseId: 42, TermId: 101, Status: course.OfferingStatusVisible}).Error; err != nil {
		t.Fatalf("create contract offering: %v", err)
	}
}

// seedCourseSummaryContractCache 预置与 course-summary-cached.json 完全一致的 DB 缓存行。
// generatedAt 用固定时区时间，保证 Format(RFC3339) 与 fixture 逐字符一致。
func seedCourseSummaryContractCache(t *testing.T, conn *gorm.DB) {
	t.Helper()
	cst := time.FixedZone("CST", 8*3600)
	if err := conn.Create(&course.CourseAiSummaryEntity{
		CourseId: 42,
		SummaryJson: `{"consensus":"recommend","keywords":["给分好","作业多","有收获"],` +
			`"pros":["老师讲得清楚","给分宽松","课程内容实用"],"cons":["作业量较大","点名频繁"],` +
			`"representativeReviews":[{"excerpt":"老师讲得很好，作业虽然多但有收获。","sentiment":"positive"},` +
			`{"excerpt":"内容比较难，需要花时间消化。","sentiment":"neutral"}]}`,
		Model:         "qwen3.6-flash-2026-04-16",
		PromptVersion: "v1",
		GeneratedAt:   time.Date(2026, 8, 13, 10, 0, 0, 0, cst),
	}).Error; err != nil {
		t.Fatalf("seed ai summary cache: %v", err)
	}
}

// TestCourseSummaryCachedHTTPContract DB 缓存命中 → 200 + course-summary-cached.json。
func TestCourseSummaryCachedHTTPContract(t *testing.T) {
	conn, router := setupCourseSummaryContractTest(t)
	seedCourseSummaryContractCourse(t, conn)
	seedCourseSummaryContractCache(t, conn)

	rec := serveCourseGet(router, "/api/forum/courses/42/summary")
	if rec.Code != http.StatusOK {
		t.Fatalf("course summary cached status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	assertFixtureEnvelope(t, decodeContractEnvelope(t, rec), contractFixture(t, "course-summary-cached.json"))
}

// TestCourseSummaryInsufficientDataHTTPContract 无评价 → 200 + insufficient_data 占位。
func TestCourseSummaryInsufficientDataHTTPContract(t *testing.T) {
	conn, router := setupCourseSummaryContractTest(t)
	seedCourseSummaryContractCourse(t, conn)

	rec := serveCourseGet(router, "/api/forum/courses/42/summary")
	if rec.Code != http.StatusOK {
		t.Fatalf("course summary insufficient status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	assertFixtureEnvelope(t, decodeContractEnvelope(t, rec), contractFixture(t, "course-summary-insufficient-data.json"))
}

// TestCourseSummaryDisabledHTTPContract 功能关闭 → 200 + disabled。
func TestCourseSummaryDisabledHTTPContract(t *testing.T) {
	conn, router := setupCourseSummaryContractTest(t)
	seedCourseSummaryContractCourse(t, conn)
	upsertAiSummaryContractConfig(t, conn, `{"enabled":false,"globalPerMinute":5}`)

	rec := serveCourseGet(router, "/api/forum/courses/42/summary")
	if rec.Code != http.StatusOK {
		t.Fatalf("course summary disabled status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	assertFixtureEnvelope(t, decodeContractEnvelope(t, rec), contractFixture(t, "course-summary-disabled.json"))
}

// TestCourseSummaryNotFoundHTTPContract 课程不存在 → 404。
func TestCourseSummaryNotFoundHTTPContract(t *testing.T) {
	_, router := setupCourseSummaryContractTest(t)

	rec := serveCourseGet(router, "/api/forum/courses/99999/summary")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("course summary not found status = %d, want 404: %s", rec.Code, rec.Body.String())
	}
	assertFixtureEnvelope(t, decodeContractEnvelope(t, rec), contractFixture(t, "course-not-found.json"))
}

// TestCourseSummaryMalformedIDHTTPContract 非法 courseId → 400 parseFailed。
func TestCourseSummaryMalformedIDHTTPContract(t *testing.T) {
	_, router := setupCourseSummaryContractTest(t)

	rec := serveCourseGet(router, "/api/forum/courses/not-a-number/summary")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("course summary malformed id status = %d, want 400: %s", rec.Code, rec.Body.String())
	}
	assertFixtureEnvelope(t, decodeContractEnvelope(t, rec), contractFixture(t, "parse-failed.json"))
}
