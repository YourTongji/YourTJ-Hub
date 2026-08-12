package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/forum"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/course"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/courseservice"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func setupCourseContractTest(t *testing.T) (*gorm.DB, *gin.Engine) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(
		&course.Entity{},
		&course.AliasEntity{},
		&course.TermEntity{},
		&course.OfferingEntity{},
		&course.InstructorEntity{},
		&course.OfferingInstructorEntity{},
		&course.ImportRunEntity{},
		&course.SourceRefEntity{},
		&course.CourseStatsEntity{},
		&course.OfferingStatsEntity{},
	); err != nil {
		t.Fatalf("migrate course contract tables: %v", err)
	}
	// 清空课程域表，保证 fixture 断言确定性。
	for _, model := range []any{
		&course.SourceRefEntity{},
		&course.ImportRunEntity{},
		&course.OfferingInstructorEntity{},
		&course.OfferingEntity{},
		&course.InstructorEntity{},
		&course.TermEntity{},
		&course.AliasEntity{},
		&course.CourseStatsEntity{},
		&course.OfferingStatsEntity{},
		&course.Entity{},
	} {
		if err := conn.Unscoped().Where("1 = 1").Delete(model).Error; err != nil {
			t.Fatalf("clean course table: %v", err)
		}
	}

	router := gin.New()
	router.GET("/api/forum/courses", UpQueryReq(forum.CourseListJSON))
	router.GET("/api/forum/courses/:courseId", UpUriQueryReq(forum.CourseDetailJSON))
	router.GET("/api/forum/courses/:courseId/related", UpUriQueryReq(forum.CourseRelatedJSON))
	return conn, router
}

// seedCourseContractData 写入与 course-list-success/course-detail-success fixture 一致的数据。
func seedCourseContractData(t *testing.T, conn *gorm.DB) {
	t.Helper()
	entity := &course.Entity{
		Id:             42,
		PrimaryCode:    "100001",
		Name:           "高等数学(A)上",
		Department:     "数学科学学院",
		CreditX10:      50,
		NormalizedName: "高等数学a上",
		NamePinyin:     "gaodengshuxueashang",
		NameInitials:   "gdsxas",
		Status:         course.StatusVisible,
	}
	if err := conn.Create(entity).Error; err != nil {
		t.Fatalf("create course: %v", err)
	}
	if err := conn.Create(&course.AliasEntity{
		CourseId:        entity.Id,
		Kind:            course.AliasKindName,
		Value:           "高数",
		NormalizedValue: "高数",
		Source:          "fixture",
	}).Error; err != nil {
		t.Fatalf("create alias: %v", err)
	}
	term := &course.TermEntity{Id: 101, Code: "2025-2026-1", Name: "2025-2026 第一学期", Status: 0}
	if err := conn.Create(term).Error; err != nil {
		t.Fatalf("create term: %v", err)
	}
	zhang := &course.InstructorEntity{Id: 201, Name: "张三", NormalizedName: "张三", Department: "数学科学学院"}
	li := &course.InstructorEntity{Id: 202, Name: "李四", NormalizedName: "李四", Department: "数学科学学院"}
	if err := conn.Create(zhang).Error; err != nil {
		t.Fatalf("create instructor 张三: %v", err)
	}
	if err := conn.Create(li).Error; err != nil {
		t.Fatalf("create instructor 李四: %v", err)
	}
	offering := &course.OfferingEntity{
		Id:       901,
		CourseId: entity.Id,
		TermId:   term.Id,
		Campus:   "四平路校区",
		Faculty:  "数学科学学院",
		Status:   course.OfferingStatusVisible,
	}
	if err := conn.Create(offering).Error; err != nil {
		t.Fatalf("create offering: %v", err)
	}
	for _, instructor := range []*course.InstructorEntity{zhang, li} {
		if err := conn.Create(&course.OfferingInstructorEntity{
			OfferingId:   offering.Id,
			InstructorId: instructor.Id,
			Role:         "lecturer",
		}).Error; err != nil {
			t.Fatalf("create offering instructor link: %v", err)
		}
	}
}

func serveCourseGet(router http.Handler, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func TestCourseListHTTPContract(t *testing.T) {
	conn, router := setupCourseContractTest(t)
	seedCourseContractData(t, conn)

	rec := serveCourseGet(router, "/api/forum/courses?page=1&size=20")
	if rec.Code != http.StatusOK {
		t.Fatalf("course list status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	assertFixtureEnvelope(t, decodeContractEnvelope(t, rec), contractFixture(t, "course-list-success.json"))
}

func TestCourseDetailHTTPContract(t *testing.T) {
	conn, router := setupCourseContractTest(t)
	seedCourseContractData(t, conn)

	rec := serveCourseGet(router, "/api/forum/courses/42")
	if rec.Code != http.StatusOK {
		t.Fatalf("course detail status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	assertFixtureEnvelope(t, decodeContractEnvelope(t, rec), contractFixture(t, "course-detail-success.json"))
}

func TestCourseDetailNotFoundHTTPContract(t *testing.T) {
	conn, router := setupCourseContractTest(t)
	seedCourseContractData(t, conn)

	rec := serveCourseGet(router, "/api/forum/courses/99999")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("course detail status = %d, want 404: %s", rec.Code, rec.Body.String())
	}
	assertFixtureEnvelope(t, decodeContractEnvelope(t, rec), contractFixture(t, "course-not-found.json"))
}

func TestCourseDetailMalformedIDHTTPContract(t *testing.T) {
	conn, router := setupCourseContractTest(t)
	seedCourseContractData(t, conn)

	rec := serveCourseGet(router, "/api/forum/courses/not-a-number")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("course detail malformed id status = %d, want 400: %s", rec.Code, rec.Body.String())
	}
	assertFixtureEnvelope(t, decodeContractEnvelope(t, rec), contractFixture(t, "course-parse-failed.json"))
}

// seedCourseRelatedData 写入与 course-related-success fixture 一致的相关课程数据：
// 课程 42 另有仅李四授课、且位于更早学期（102）的 offering 902，用以覆盖"最近学期开课"
// 作为 primary 的 term 最近性选择；同教师课程 43 线性代数由张三开课。
func seedCourseRelatedData(t *testing.T, conn *gorm.DB) {
	t.Helper()
	linear := &course.Entity{
		Id:             43,
		PrimaryCode:    "100002",
		Name:           "线性代数",
		Department:     "数学科学学院",
		CreditX10:      40,
		NormalizedName: "线性代数",
		NamePinyin:     "xianxingdaishu",
		NameInitials:   "xxds",
		Status:         course.StatusVisible,
	}
	if err := conn.Create(linear).Error; err != nil {
		t.Fatalf("create course 线性代数: %v", err)
	}
	// 更早学期：offering 902 使用 term 102（code 小于 term 101，排序在 901 之后）。
	if err := conn.Create(&course.TermEntity{Id: 102, Code: "2024-2025-1", Name: "2024-2025 第一学期", Status: 0}).Error; err != nil {
		t.Fatalf("create term 102: %v", err)
	}
	for _, offering := range []*course.OfferingEntity{
		{Id: 902, CourseId: 42, TermId: 102, Campus: "四平路校区", Faculty: "数学科学学院", Status: course.OfferingStatusVisible},
		{Id: 903, CourseId: 43, TermId: 101, Campus: "四平路校区", Faculty: "数学科学学院", Status: course.OfferingStatusVisible},
	} {
		if err := conn.Create(offering).Error; err != nil {
			t.Fatalf("create offering %d: %v", offering.Id, err)
		}
	}
	for _, link := range []*course.OfferingInstructorEntity{
		{OfferingId: 902, InstructorId: 202, Role: "lecturer"}, // 李四
		{OfferingId: 903, InstructorId: 201, Role: "lecturer"}, // 张三
	} {
		if err := conn.Create(link).Error; err != nil {
			t.Fatalf("create offering instructor link: %v", err)
		}
	}
	for _, st := range []*course.CourseStatsEntity{
		{CourseId: 42, RatingCount: 2, RatingSum: 9, ReviewCount: 2},
		{CourseId: 43, RatingCount: 2, RatingSum: 9, ReviewCount: 2},
	} {
		if err := conn.Create(st).Error; err != nil {
			t.Fatalf("create course stats: %v", err)
		}
	}
	for _, st := range []*course.OfferingStatsEntity{
		{OfferingId: 901, RatingCount: 1, RatingSum: 5, ReviewCount: 1},
		{OfferingId: 902, RatingCount: 1, RatingSum: 4, ReviewCount: 1},
		{OfferingId: 903, RatingCount: 2, RatingSum: 9, ReviewCount: 2},
	} {
		if err := conn.Create(st).Error; err != nil {
			t.Fatalf("create offering stats: %v", err)
		}
	}
}

func TestCourseRelatedHTTPContract(t *testing.T) {
	conn, router := setupCourseContractTest(t)
	seedCourseContractData(t, conn)
	seedCourseRelatedData(t, conn)

	rec := serveCourseGet(router, "/api/forum/courses/42/related")
	if rec.Code != http.StatusOK {
		t.Fatalf("course related status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	assertFixtureEnvelope(t, decodeContractEnvelope(t, rec), contractFixture(t, "course-related-success.json"))
}

func TestCourseRelatedNotFoundHTTPContract(t *testing.T) {
	conn, router := setupCourseContractTest(t)
	seedCourseContractData(t, conn)

	rec := serveCourseGet(router, "/api/forum/courses/99999/related")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("course related not found status = %d, want 404: %s", rec.Code, rec.Body.String())
	}
	assertFixtureEnvelope(t, decodeContractEnvelope(t, rec), contractFixture(t, "course-not-found.json"))
}

func TestCourseRelatedMalformedIDHTTPContract(t *testing.T) {
	conn, router := setupCourseContractTest(t)
	seedCourseContractData(t, conn)

	rec := serveCourseGet(router, "/api/forum/courses/not-a-number/related")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("course related malformed id status = %d, want 400: %s", rec.Code, rec.Body.String())
	}
	assertFixtureEnvelope(t, decodeContractEnvelope(t, rec), contractFixture(t, "course-parse-failed.json"))
}

func TestCourseRelatedEmptyHTTPContract(t *testing.T) {
	// 仅 seed 基础课程（无同教师其他课、无其他教师开课）：两个列表均为空数组。
	conn, router := setupCourseContractTest(t)
	seedCourseContractData(t, conn)

	rec := serveCourseGet(router, "/api/forum/courses/42/related")
	if rec.Code != http.StatusOK {
		t.Fatalf("course related empty status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	assertFixtureEnvelope(t, decodeContractEnvelope(t, rec), contractFixture(t, "course-related-empty.json"))
}

func TestCourseRelatedZeroIDHTTPContract(t *testing.T) {
	conn, router := setupCourseContractTest(t)
	seedCourseContractData(t, conn)

	rec := serveCourseGet(router, "/api/forum/courses/0/related")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("course related zero id status = %d, want 400: %s", rec.Code, rec.Body.String())
	}
	assertFixtureEnvelope(t, decodeContractEnvelope(t, rec), contractFixture(t, "course-related-invalid-params.json"))
}

func TestCourseRelatedHiddenHTTPContract(t *testing.T) {
	conn, router := setupCourseContractTest(t)
	seedCourseContractData(t, conn)
	hidden := &course.Entity{
		Id:             44,
		PrimaryCode:    "100044",
		Name:           "隐藏课程",
		Department:     "数学科学学院",
		NormalizedName: "隐藏课程",
		Status:         course.StatusHidden,
	}
	if err := conn.Create(hidden).Error; err != nil {
		t.Fatalf("create hidden course: %v", err)
	}

	rec := serveCourseGet(router, "/api/forum/courses/44/related")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("course related hidden status = %d, want 404: %s", rec.Code, rec.Body.String())
	}
	assertFixtureEnvelope(t, decodeContractEnvelope(t, rec), contractFixture(t, "course-not-found.json"))

// TestCourseStatsProjectionHTTPContract 验证 B1 统计投影对外暴露（issue #173 验收）：
//  1. 3 条评价（5/4/NULL）→ ratingAvg=4.5、reviewCount=3（NULL 不计均分但计评论数）；
//  2. ratingDistribution 各档与可见 course_review 行一致；
//  3. 管理员隐藏一条 → 统计即时排除（与 SetReviewVisibility delta 同事务）。
func TestCourseStatsProjectionHTTPContract(t *testing.T) {
	conn, router := setupCourseContractTest(t)
	seedCourseContractData(t, conn)

	// 迁移评价与统计投影表
	if err := conn.AutoMigrate(&course.ReviewEntity{}, &course.CourseStatsEntity{}, &course.OfferingStatsEntity{}); err != nil {
		t.Fatalf("migrate review/stats tables: %v", err)
	}
	conn.Unscoped().Where("1 = 1").Delete(&course.ReviewEntity{})
	conn.Unscoped().Where("1 = 1").Delete(&course.CourseStatsEntity{})
	conn.Unscoped().Where("1 = 1").Delete(&course.OfferingStatsEntity{})

	// 3 条评价：rating 5、4、NULL（legacy 无评分）
	r5, r4 := 5, 4
	seedCourseReview(t, conn, 1001, 901, 1, &r5, "五星", false, "contract", course.ReviewStatusVisible)
	seedCourseReview(t, conn, 1002, 901, 2, &r4, "四星", false, "contract", course.ReviewStatusVisible)
	seedCourseReview(t, conn, 1003, 901, 3, nil, "无评分", false, "contract", course.ReviewStatusVisible)

	// 同步统计投影（生产路径由 Upsert 维护；测试直接调用重建）
	if err := course.RebuildAllCourseStats(); err != nil {
		t.Fatalf("rebuild stats: %v", err)
	}

	// --- 验收 1 + 2：目录/详情带均分/计数/分布 ---
	rec := serveCourseGet(router, "/api/forum/courses?page=1&size=20")
	if rec.Code != http.StatusOK {
		t.Fatalf("course list status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	var listEnvelope struct {
		Result struct {
			List []map[string]any `json:"list"`
		} `json:"result"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &listEnvelope); err != nil {
		t.Fatalf("decode course list: %v", err)
	}
	if len(listEnvelope.Result.List) != 1 {
		t.Fatalf("course list length = %d, want 1", len(listEnvelope.Result.List))
	}
	item := listEnvelope.Result.List[0]
	if got := item["ratingAvg"]; got != 4.5 {
		t.Fatalf("list ratingAvg = %#v, want 4.5", got)
	}
	if got := item["reviewCount"]; got != float64(3) {
		t.Fatalf("list reviewCount = %#v, want 3", got)
	}

	rec = serveCourseGet(router, "/api/forum/courses/42")
	if rec.Code != http.StatusOK {
		t.Fatalf("course detail status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	var detailEnvelope struct {
		Result map[string]any `json:"result"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &detailEnvelope); err != nil {
		t.Fatalf("decode course detail: %v", err)
	}
	detail := detailEnvelope.Result
	if got := detail["ratingAvg"]; got != 4.5 {
		t.Fatalf("detail ratingAvg = %#v, want 4.5", got)
	}
	if got := detail["reviewCount"]; got != float64(3) {
		t.Fatalf("detail reviewCount = %#v, want 3", got)
	}
	dist, ok := detail["ratingDistribution"].([]any)
	if !ok || len(dist) != 5 {
		t.Fatalf("detail ratingDistribution = %#v, want [5]int array", detail["ratingDistribution"])
	}
	if dist[4] != float64(1) || dist[3] != float64(1) || dist[0] != float64(0) {
		t.Fatalf("detail ratingDistribution = %v, want [0,0,0,1,1]", dist)
	}
	// offering 级统计（详情开课列表）
	offerings, _ := detail["offerings"].([]any)
	if len(offerings) != 1 {
		t.Fatalf("detail offerings = %#v, want 1", offerings)
	}
	off := offerings[0].(map[string]any)
	if got := off["ratingAvg"]; got != 4.5 {
		t.Fatalf("offering ratingAvg = %#v, want 4.5", got)
	}
	if got := off["reviewCount"]; got != float64(3) {
		t.Fatalf("offering reviewCount = %#v, want 3", got)
	}

	// --- 验收 3：隐藏一条评价 → 统计即时排除（SetReviewVisibility delta 同事务） ---
	if err := courseservice.SetReviewVisibility(1003, true); err != nil { // 隐藏 NULL 评分评价
		t.Fatalf("SetReviewVisibility: %v", err)
	}
	rec = serveCourseGet(router, "/api/forum/courses/42")
	if rec.Code != http.StatusOK {
		t.Fatalf("course detail after hide status = %d: %s", rec.Code, rec.Body.String())
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &detailEnvelope); err != nil {
		t.Fatalf("decode course detail after hide: %v", err)
	}
	detail = detailEnvelope.Result
	if got := detail["reviewCount"]; got != float64(2) {
		t.Fatalf("reviewCount after hide = %#v, want 2", got)
	}
	if got := detail["ratingAvg"]; got != 4.5 {
		t.Fatalf("ratingAvg after hide (5+4) = %#v, want 4.5", got)
	}
	dist, _ = detail["ratingDistribution"].([]any)
	if dist[4] != float64(1) || dist[3] != float64(1) {
		t.Fatalf("ratingDistribution after hide = %v, want [0,0,0,1,1]", dist)
	}

	// 隐藏 5 星评价 → 均分变 4.0、分布只剩 4 星
	if err := courseservice.SetReviewVisibility(1001, true); err != nil {
		t.Fatalf("SetReviewVisibility(1001): %v", err)
	}
	rec = serveCourseGet(router, "/api/forum/courses/42")
	_ = json.Unmarshal(rec.Body.Bytes(), &detailEnvelope)
	detail = detailEnvelope.Result
	if got := detail["ratingAvg"]; got != float64(4) {
		t.Fatalf("ratingAvg after hide 5-star = %#v, want 4", got)
	}
	if got := detail["reviewCount"]; got != float64(1) {
		t.Fatalf("reviewCount after hide 5-star = %#v, want 1", got)
	}
}
