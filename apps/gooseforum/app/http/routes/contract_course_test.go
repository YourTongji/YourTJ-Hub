package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/forum"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/course"
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
// 课程 42 另有仅李四授课的 offering 902；同教师课程 43 线性代数由张三开课。
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
	for _, offering := range []*course.OfferingEntity{
		{Id: 902, CourseId: 42, TermId: 101, Campus: "四平路校区", Faculty: "数学科学学院", Status: course.OfferingStatusVisible},
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
