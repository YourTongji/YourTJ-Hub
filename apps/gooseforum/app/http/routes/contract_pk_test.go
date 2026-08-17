package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/pk"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/course"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pk"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/pkservice"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// pkEnvelope PK 统一信封 {code, msg, data} 的契约测试结构。
type pkEnvelope struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

// pkModelList 契约测试迁移/清理的 PK 模型清单。
var pkModelList = []any{
	&pk.CalendarEntity{},
	&pk.CampusEntity{},
	&pk.FacultyEntity{},
	&pk.LanguageEntity{},
	&pk.AssessmentEntity{},
	&pk.CourseNatureEntity{},
	&pk.CourseNatureByCalendarEntity{},
	&pk.MajorEntity{},
	&pk.MajorCourseEntity{},
	&pk.CourseDetailEntity{},
	&pk.TeacherEntity{},
	&pk.TeacherTimeslotEntity{},
	&pk.FetchLogEntity{},
	&pk.SettingEntity{},
}

func setupPkContractTest(t *testing.T) (*gorm.DB, *gin.Engine) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	pkservice.ResetPkAuxiliaryStateForTest()
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(append(pkModelList,
		&course.Entity{},
		&course.CourseStatsEntity{},
		&course.TermEntity{},
		&course.OfferingEntity{},
		&course.InstructorEntity{},
		&course.OfferingInstructorEntity{},
		&course.OfferingStatsEntity{},
	)...); err != nil {
		t.Fatalf("migrate pk contract tables: %v", err)
	}
	cleanupPkTables(t, conn)
	t.Cleanup(func() {
		pkservice.WaitPkAuxiliaryBuildForTest()
		pkservice.ResetPkAuxiliaryStateForTest()
		cleanupPkTables(t, conn)
	})

	router := gin.New()
	pkApi := router.Group("/api/pk")
	pkApi.GET("calendars", pkNoReq(pkcontroller.ListCalendars))
	pkApi.GET("campuses", pkNoReq(pkcontroller.ListCampuses))
	pkApi.GET("faculties", pkNoReq(pkcontroller.ListFaculties))
	pkApi.POST("grades", pkJsonReq(pkcontroller.Grades))
	pkApi.POST("majors", pkJsonReq(pkcontroller.Majors))
	pkApi.POST("courses-by-major", pkJsonReq(pkcontroller.CoursesByMajor))
	pkApi.POST("optional-types", pkJsonReq(pkcontroller.OptionalTypes))
	pkApi.POST("courses-by-nature", pkJsonReq(pkcontroller.CoursesByNature))
	pkApi.POST("course-details", pkJsonReq(pkcontroller.CourseDetails))
	pkApi.POST("course-search", pkJsonReq(pkcontroller.CourseSearch))
	pkApi.POST("courses-by-time", pkJsonReq(pkcontroller.CoursesByTime))
	pkApi.GET("latest-update", pkNoReq(pkcontroller.LatestUpdate))
	pkApi.POST("course-info-sync", pkJsonReq(pkcontroller.CourseInfoSync))
	pkApi.GET("course-review-brief", pkQueryReq(pkcontroller.CourseReviewBrief))
	return conn, router
}

func cleanupPkTables(t *testing.T, conn *gorm.DB) {
	t.Helper()
	for _, model := range pkModelList {
		if err := conn.Unscoped().Where("1 = 1").Delete(model).Error; err != nil {
			t.Fatalf("clean pk table: %v", err)
		}
	}
	if err := conn.Unscoped().Where("1 = 1").Delete(&course.Entity{}).Error; err != nil {
		t.Fatalf("clean course table: %v", err)
	}
	if err := conn.Unscoped().Where("1 = 1").Delete(&course.CourseStatsEntity{}).Error; err != nil {
		t.Fatalf("clean course stats table: %v", err)
	}
	for _, model := range []any{
		&course.OfferingStatsEntity{},
		&course.OfferingInstructorEntity{},
		&course.InstructorEntity{},
		&course.OfferingEntity{},
		&course.TermEntity{},
	} {
		if err := conn.Unscoped().Where("1 = 1").Delete(model).Error; err != nil {
			t.Fatalf("clean course offering table: %v", err)
		}
	}
}

// seedPkContractData 写入与 pk-*-success fixture 一致的数据。
func seedPkContractData(t *testing.T, conn *gorm.DB) {
	t.Helper()
	finishedAt := time.Unix(1723456800, 0).UTC()
	models := []any{
		&pk.CalendarEntity{CalendarId: 99999, CalendarIdI18n: "本地测试学期"},
		&pk.CalendarEntity{CalendarId: 99998, CalendarIdI18n: "2025-2026 第一学期"},
		&pk.CampusEntity{Campus: "SP", CampusI18n: "四平路校区", CalendarId: 99999},
		&pk.CampusEntity{Campus: "JD", CampusI18n: "嘉定校区", CalendarId: 99999},
		&pk.FacultyEntity{Faculty: "CS", FacultyI18n: "计算机科学与技术系", CalendarId: 99999},
		&pk.LanguageEntity{TeachingLanguage: "ZH", TeachingLanguageI18n: "中文", CalendarId: 99999},
		&pk.CourseNatureByCalendarEntity{CalendarId: 99999, CourseLabelId: 1, CourseLabelName: "专业必修"},
		&pk.CourseNatureByCalendarEntity{CalendarId: 99999, CourseLabelId: 2, CourseLabelName: "通识选修课"},
		&pk.MajorEntity{Id: 9000, Code: "03074", Grade: intPtr(2025), Name: "2025(03074 测试专业)", CalendarId: 99999},
		&pk.MajorEntity{Id: 9001, Code: "03075", Grade: intPtr(2024), Name: "2024(03075 测试专业)", CalendarId: 99999},
		&pk.MajorCourseEntity{MajorId: 9000, CourseId: 900001},
		&pk.MajorCourseEntity{MajorId: 9000, CourseId: 900002},
		&pk.MajorCourseEntity{MajorId: 9001, CourseId: 900003},
		&pk.CourseDetailEntity{Id: 900001, Code: "TJCS10101", Name: "计算机程序设计-1班", CourseLabelId: uint64Ptr(1), AssessmentMode: "EXAM", Period: float64Ptr(48), WeekHour: float64Ptr(3), Campus: "SP", Number: intPtr(60), ElcNumber: intPtr(0), StartWeek: intPtr(1), EndWeek: intPtr(16), CourseCode: "TJCS101", CourseName: "计算机程序设计", Credit: float64Ptr(3), TeachingLanguage: "ZH", Faculty: "CS", CalendarId: 99999, NewCourseCode: "CS101", NewCode: "CS10101"},
		&pk.CourseDetailEntity{Id: 900002, Code: "TJCS10102", Name: "计算机程序设计-2班", CourseLabelId: uint64Ptr(1), AssessmentMode: "EXAM", Period: float64Ptr(48), WeekHour: float64Ptr(3), Campus: "SP", Number: intPtr(60), ElcNumber: intPtr(0), StartWeek: intPtr(1), EndWeek: intPtr(16), CourseCode: "TJCS101", CourseName: "计算机程序设计", Credit: float64Ptr(3), TeachingLanguage: "ZH", Faculty: "CS", CalendarId: 99999, NewCourseCode: "CS101", NewCode: "CS10102"},
		&pk.CourseDetailEntity{Id: 900003, Code: "TJCS20101", Name: "数据结构与算法-1班", CourseLabelId: uint64Ptr(2), AssessmentMode: "EXAM", Period: float64Ptr(64), WeekHour: float64Ptr(4), Campus: "JD", Number: intPtr(80), ElcNumber: intPtr(0), StartWeek: intPtr(1), EndWeek: intPtr(16), CourseCode: "TJCS201", CourseName: "数据结构与算法", Credit: float64Ptr(4), TeachingLanguage: "ZH", Faculty: "CS", CalendarId: 99999, NewCourseCode: "CS201", NewCode: "CS20101"},
		&pk.TeacherEntity{Id: 1, TeachingClassId: 900001, TeacherCode: "T001", TeacherName: "张伟", ArrangeInfoText: "张伟(T001) 星期一1-2节[1-16周] 四平路校区 A101\n张伟(T001) 星期三3-4节[1-16周] 四平路校区 A101"},
		&pk.TeacherEntity{Id: 2, TeachingClassId: 900002, TeacherCode: "T006", TeacherName: "李娜", ArrangeInfoText: "李娜(T006) 星期二1-2节[1-16周] 四平路校区 B202"},
		&pk.TeacherEntity{Id: 3, TeachingClassId: 900003, TeacherCode: "T001", TeacherName: "张伟", ArrangeInfoText: "张伟(T001) 星期五1-2节[1-16周] 嘉定校区 C303"},
		&pk.FetchLogEntity{Id: 1, CalendarId: 99999, Status: pk.FetchStatusCompleted, FinishedAt: &finishedAt},
		&course.Entity{Id: 1, PrimaryCode: "CS101", Name: "计算机程序设计", Department: "计算机", CreditX10: 30, NormalizedName: "计算机程序设计", Status: course.StatusVisible},
		&course.CourseStatsEntity{CourseId: 1, RatingCount: 1, RatingSum: 4, ReviewCount: 1},
		&course.TermEntity{Id: 1, Code: "2025-2026-2", Name: "2025-2026 第二学期", Status: 0},
		&course.OfferingEntity{Id: 1, CourseId: 1, TermId: 1, Campus: "四平路校区", Faculty: "计算机", ClassCode: "TJCS10101", ClassName: "计算机程序设计-1班", Status: course.OfferingStatusVisible},
		&course.OfferingEntity{Id: 2, CourseId: 1, TermId: 1, Campus: "四平路校区", Faculty: "计算机", ClassCode: "TJCS10102", ClassName: "计算机程序设计-2班", Status: course.OfferingStatusVisible},
		&course.InstructorEntity{Id: 1, Name: "张伟", NormalizedName: "张伟"},
		&course.InstructorEntity{Id: 2, Name: "李娜", NormalizedName: "李娜"},
		&course.OfferingInstructorEntity{OfferingId: 1, InstructorId: 1},
		&course.OfferingInstructorEntity{OfferingId: 2, InstructorId: 2},
		&course.OfferingStatsEntity{OfferingId: 1, RatingCount: 1, RatingSum: 4, ReviewCount: 1},
	}
	for _, m := range models {
		if err := conn.Create(m).Error; err != nil {
			t.Fatalf("create pk seed %T: %v", m, err)
		}
	}
}

// pkContractFixture 读取 PK fixture（{code,msg,data} 信封）。
func pkContractFixture(t *testing.T, filename string) pkEnvelope {
	t.Helper()
	_, testFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve pk contract fixture path")
	}
	root := filepath.Join(filepath.Dir(testFile), "..", "..", "..", "..", "..")
	path := filepath.Join(root, "packages", "api-contract", "fixtures", filename)
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read pk fixture %s: %v", filename, err)
	}
	var fixture pkEnvelope
	if err := json.Unmarshal(contents, &fixture); err != nil {
		t.Fatalf("decode pk fixture %s: %v", filename, err)
	}
	return fixture
}

func servePkJSON(router http.Handler, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func servePkGET(router http.Handler, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func decodePkEnvelope(t *testing.T, rec *httptest.ResponseRecorder) pkEnvelope {
	t.Helper()
	var envelope pkEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode pk response %q: %v", rec.Body.String(), err)
	}
	return envelope
}

func assertPkFixture(t *testing.T, actual pkEnvelope, fixture pkEnvelope) {
	t.Helper()
	if actual.Code != fixture.Code {
		t.Fatalf("code = %d, want fixture code %d", actual.Code, fixture.Code)
	}
	if actual.Msg != fixture.Msg {
		t.Fatalf("msg = %q, want fixture msg %q", actual.Msg, fixture.Msg)
	}
	assertPkData(t, actual.Data, fixture.Data)
}

func assertPkData(t *testing.T, actual, fixture json.RawMessage) {
	t.Helper()
	var actualValue any
	if err := json.Unmarshal(actual, &actualValue); err != nil {
		t.Fatalf("decode response data %q: %v", actual, err)
	}
	var fixtureValue any
	if err := json.Unmarshal(fixture, &fixtureValue); err != nil {
		t.Fatalf("decode fixture data %q: %v", fixture, err)
	}
	if !reflect.DeepEqual(actualValue, fixtureValue) {
		t.Fatalf("data = %s, want fixture data %s", actual, fixture)
	}
}

// TestPkCalendarsHTTPContract P1
func TestPkCalendarsHTTPContract(t *testing.T) {
	conn, router := setupPkContractTest(t)
	seedPkContractData(t, conn)
	rec := servePkGET(router, "/api/pk/calendars")
	if rec.Code != http.StatusOK {
		t.Fatalf("calendars status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	assertPkFixture(t, decodePkEnvelope(t, rec), pkContractFixture(t, "pk-calendars-success.json"))
}

// TestPkCampusesFacultiesHTTPContract P2
func TestPkCampusesFacultiesHTTPContract(t *testing.T) {
	conn, router := setupPkContractTest(t)
	seedPkContractData(t, conn)
	for _, tc := range []struct {
		path    string
		fixture string
	}{
		{"/api/pk/campuses", "pk-campuses-success.json"},
		{"/api/pk/faculties", "pk-faculties-success.json"},
	} {
		t.Run(tc.path, func(t *testing.T) {
			rec := servePkGET(router, tc.path)
			if rec.Code != http.StatusOK {
				t.Fatalf("%s status = %d, want 200: %s", tc.path, rec.Code, rec.Body.String())
			}
			assertPkFixture(t, decodePkEnvelope(t, rec), pkContractFixture(t, tc.fixture))
		})
	}
}

// TestPkGradesHTTPContract P3
func TestPkGradesHTTPContract(t *testing.T) {
	conn, router := setupPkContractTest(t)
	seedPkContractData(t, conn)

	rec := servePkJSON(router, "/api/pk/grades", `{"calendarId":99999}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("grades status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	assertPkFixture(t, decodePkEnvelope(t, rec), pkContractFixture(t, "pk-grades-success.json"))

	recBad := servePkJSON(router, "/api/pk/grades", `{}`)
	if recBad.Code != http.StatusBadRequest {
		t.Fatalf("grades bad-request status = %d, want 400: %s", recBad.Code, recBad.Body.String())
	}
	assertPkFixture(t, decodePkEnvelope(t, recBad), pkContractFixture(t, "pk-bad-request.json"))
}

// TestPkMajorsHTTPContract P4
func TestPkMajorsHTTPContract(t *testing.T) {
	conn, router := setupPkContractTest(t)
	seedPkContractData(t, conn)

	rec := servePkJSON(router, "/api/pk/majors", `{"grade":2025}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("majors status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	assertPkFixture(t, decodePkEnvelope(t, rec), pkContractFixture(t, "pk-majors-success.json"))

	recBad := servePkJSON(router, "/api/pk/majors", `{}`)
	if recBad.Code != http.StatusBadRequest {
		t.Fatalf("majors bad-request status = %d, want 400: %s", recBad.Code, recBad.Body.String())
	}
	assertPkFixture(t, decodePkEnvelope(t, recBad), pkContractFixture(t, "pk-majors-bad-request.json"))
}

// TestPkCoursesByMajorHTTPContract P5
func TestPkCoursesByMajorHTTPContract(t *testing.T) {
	conn, router := setupPkContractTest(t)
	seedPkContractData(t, conn)

	rec := servePkJSON(router, "/api/pk/courses-by-major", `{"grade":2025,"code":"03074","calendarId":99999}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("courses-by-major status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	assertPkFixture(t, decodePkEnvelope(t, rec), pkContractFixture(t, "pk-courses-by-major-success.json"))

	recBad := servePkJSON(router, "/api/pk/courses-by-major", `{}`)
	if recBad.Code != http.StatusBadRequest {
		t.Fatalf("courses-by-major bad-request status = %d, want 400: %s", recBad.Code, recBad.Body.String())
	}
	assertPkFixture(t, decodePkEnvelope(t, recBad), pkContractFixture(t, "pk-courses-by-major-bad-request.json"))
}

// TestPkOptionalTypesHTTPContract P6
func TestPkOptionalTypesHTTPContract(t *testing.T) {
	conn, router := setupPkContractTest(t)
	seedPkContractData(t, conn)

	rec := servePkJSON(router, "/api/pk/optional-types", `{"calendarId":99999}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("optional-types status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	assertPkFixture(t, decodePkEnvelope(t, rec), pkContractFixture(t, "pk-optional-types-success.json"))

	recBad := servePkJSON(router, "/api/pk/optional-types", `{}`)
	if recBad.Code != http.StatusBadRequest {
		t.Fatalf("optional-types bad-request status = %d, want 400: %s", recBad.Code, recBad.Body.String())
	}
	assertPkFixture(t, decodePkEnvelope(t, recBad), pkContractFixture(t, "pk-bad-request.json"))
}

// TestPkCoursesByNatureHTTPContract P7
func TestPkCoursesByNatureHTTPContract(t *testing.T) {
	conn, router := setupPkContractTest(t)
	seedPkContractData(t, conn)

	rec := servePkJSON(router, "/api/pk/courses-by-nature", `{"calendarId":99999,"ids":[1,2]}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("courses-by-nature status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	assertPkFixture(t, decodePkEnvelope(t, rec), pkContractFixture(t, "pk-courses-by-nature-success.json"))

	recBad := servePkJSON(router, "/api/pk/courses-by-nature", `{"calendarId":99999,"ids":[]}`)
	if recBad.Code != http.StatusBadRequest {
		t.Fatalf("courses-by-nature bad-request status = %d, want 400: %s", recBad.Code, recBad.Body.String())
	}
	assertPkFixture(t, decodePkEnvelope(t, recBad), pkContractFixture(t, "pk-courses-by-nature-bad-request.json"))
}

// TestPkCourseDetailsHTTPContract P8
func TestPkCourseDetailsHTTPContract(t *testing.T) {
	conn, router := setupPkContractTest(t)
	seedPkContractData(t, conn)

	t.Run("single courseCode returns array", func(t *testing.T) {
		rec := servePkJSON(router, "/api/pk/course-details", `{"calendarId":99999,"courseCode":"TJCS101"}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("course-details single status = %d, want 200: %s", rec.Code, rec.Body.String())
		}
		assertPkFixture(t, decodePkEnvelope(t, rec), pkContractFixture(t, "pk-course-details-single-success.json"))
	})

	t.Run("batch courseCodes returns dict", func(t *testing.T) {
		rec := servePkJSON(router, "/api/pk/course-details", `{"calendarId":99999,"courseCodes":["TJCS101","TJCS201"]}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("course-details batch status = %d, want 200: %s", rec.Code, rec.Body.String())
		}
		assertPkFixture(t, decodePkEnvelope(t, rec), pkContractFixture(t, "pk-course-details-batch-success.json"))
	})

	t.Run("missing courseCode returns 400", func(t *testing.T) {
		rec := servePkJSON(router, "/api/pk/course-details", `{"calendarId":99999}`)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("course-details bad-request status = %d, want 400: %s", rec.Code, rec.Body.String())
		}
		assertPkFixture(t, decodePkEnvelope(t, rec), pkContractFixture(t, "pk-course-details-bad-request.json"))
	})
}

// TestPkCourseSearchHTTPContract P9
func TestPkCourseSearchHTTPContract(t *testing.T) {
	conn, router := setupPkContractTest(t)
	seedPkContractData(t, conn)

	rec := servePkJSON(router, "/api/pk/course-search", `{"calendarId":99999}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("course-search status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	assertPkFixture(t, decodePkEnvelope(t, rec), pkContractFixture(t, "pk-course-search-success.json"))

	recBad := servePkJSON(router, "/api/pk/course-search", `{}`)
	if recBad.Code != http.StatusBadRequest {
		t.Fatalf("course-search bad-request status = %d, want 400: %s", recBad.Code, recBad.Body.String())
	}
	assertPkFixture(t, decodePkEnvelope(t, recBad), pkContractFixture(t, "pk-bad-request.json"))
}

// TestPkCoursesByTimeHTTPContract P10（未就绪 → 降级 + auxiliaryReady:false）
func TestPkCoursesByTimeHTTPContractDegraded(t *testing.T) {
	conn, router := setupPkContractTest(t)
	seedPkContractData(t, conn)

	rec := servePkJSON(router, "/api/pk/courses-by-time", `{"calendarId":99999,"day":5,"section":1}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("courses-by-time status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	env := decodePkEnvelope(t, rec)
	if env.Code != 0 {
		t.Fatalf("courses-by-time code = %d, want 0: %s", env.Code, rec.Body.String())
	}
	var result struct {
		AuxiliaryReady bool `json:"auxiliaryReady"`
	}
	if err := json.Unmarshal(env.Data, &result); err != nil {
		t.Fatalf("decode auxiliaryReady: %v", err)
	}
	if result.AuxiliaryReady {
		t.Fatalf("courses-by-time auxiliaryReady = true, want false (timeslots not seeded): %s", rec.Body.String())
	}
	assertPkFixture(t, env, pkContractFixture(t, "pk-courses-by-time-degraded.json"))
}

// TestPkLatestUpdateHTTPContract P11
func TestPkLatestUpdateHTTPContract(t *testing.T) {
	conn, router := setupPkContractTest(t)
	seedPkContractData(t, conn)

	rec := servePkGET(router, "/api/pk/latest-update")
	if rec.Code != http.StatusOK {
		t.Fatalf("latest-update status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	assertPkFixture(t, decodePkEnvelope(t, rec), pkContractFixture(t, "pk-latest-update-success.json"))
}

// TestPkCourseInfoSyncHTTPContract P12
func TestPkCourseInfoSyncHTTPContract(t *testing.T) {
	conn, router := setupPkContractTest(t)
	seedPkContractData(t, conn)

	rec := servePkJSON(router, "/api/pk/course-info-sync", `{"calendarId":99999,"majorCourseCodes":["TJCS101"],"otherCourseCodes":["TJCS201"],"majorInfo":{"grade":2025,"code":"03074"}}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("course-info-sync status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	assertPkFixture(t, decodePkEnvelope(t, rec), pkContractFixture(t, "pk-course-info-sync-success.json"))

	recBad := servePkJSON(router, "/api/pk/course-info-sync", `{}`)
	if recBad.Code != http.StatusBadRequest {
		t.Fatalf("course-info-sync bad-request status = %d, want 400: %s", recBad.Code, recBad.Body.String())
	}
	assertPkFixture(t, decodePkEnvelope(t, recBad), pkContractFixture(t, "pk-bad-request.json"))
}

// TestPkCourseReviewBriefHTTPContract P13
func TestPkCourseReviewBriefHTTPContract(t *testing.T) {
	conn, router := setupPkContractTest(t)
	seedPkContractData(t, conn)

	rec := servePkGET(router, "/api/pk/course-review-brief?courseCode=TJCS101&teacherName=张伟")
	if rec.Code != http.StatusOK {
		t.Fatalf("course-review-brief status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	assertPkFixture(t, decodePkEnvelope(t, rec), pkContractFixture(t, "pk-course-review-brief-success.json"))

	recBad := servePkGET(router, "/api/pk/course-review-brief")
	if recBad.Code != http.StatusBadRequest {
		t.Fatalf("course-review-brief bad-request status = %d, want 400: %s", recBad.Code, recBad.Body.String())
	}
	assertPkFixture(t, decodePkEnvelope(t, recBad), pkContractFixture(t, "pk-course-review-brief-bad-request.json"))
}

// float64Ptr 返回 v 的地址（dev 侧 CourseDetailEntity 可空指针字段的 fixture 用）。
func float64Ptr(v float64) *float64 { return &v }
