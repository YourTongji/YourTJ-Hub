package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/leancodebox/GooseForum/app/bundles/connect/dbconnect"
	"github.com/leancodebox/GooseForum/app/http/controllers/forum"
	"github.com/leancodebox/GooseForum/app/models/forum/course"
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
		&course.Entity{},
	} {
		if err := conn.Unscoped().Where("1 = 1").Delete(model).Error; err != nil {
			t.Fatalf("clean course table: %v", err)
		}
	}

	router := gin.New()
	router.GET("/api/forum/courses", UpQueryReq(forum.CourseListJSON))
	router.GET("/api/forum/courses/:courseId", UpUriQueryReq(forum.CourseDetailJSON))
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
