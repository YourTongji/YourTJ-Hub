package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
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
}

// seedCourseFilterData 写入高级筛选测试数据（不参与 course-list-success 基线，仅配合
// 新筛选参数的 HTTP 契约测试）：
//   - 课程 45 大学物理，李四授课，CourseStatsEntity RatingCount 3 / RatingSum 12 / ReviewCount 5（平均分 4.0）
//   - 课程 46 无机化学，张三授课，CourseStatsEntity RatingCount 1 / RatingSum 5  / ReviewCount 2（平均分 5.0）
//
// 基线课程 42 高等数学无 CourseStatsEntity 行（三字段零值），用于验证 onlyWithReviews 的
// 排除与 sortBy=rating 的"无评分垫底"排序。
func seedCourseFilterData(t *testing.T, conn *gorm.DB) {
	t.Helper()
	for _, c := range []*course.Entity{
		{
			Id:             45,
			PrimaryCode:    "200001",
			Name:           "大学物理",
			Department:     "物理科学与工程学院",
			CreditX10:      30,
			NormalizedName: "大学物理",
			NamePinyin:     "daxuewuli",
			NameInitials:   "dxwl",
			Status:         course.StatusVisible,
		},
		{
			Id:             46,
			PrimaryCode:    "200002",
			Name:           "无机化学",
			Department:     "化学科学与工程学院",
			CreditX10:      40,
			NormalizedName: "无机化学",
			NamePinyin:     "wujihuaxue",
			NameInitials:   "wjhx",
			Status:         course.StatusVisible,
		},
	} {
		if err := conn.Create(c).Error; err != nil {
			t.Fatalf("create course %d: %v", c.Id, err)
		}
	}
	// offering 904 由李四(202)授课程 45，offering 905 由张三(201)授课程 46，复用 term 101。
	for _, offering := range []*course.OfferingEntity{
		{Id: 904, CourseId: 45, TermId: 101, Campus: "四平路校区", Faculty: "物理科学与工程学院", Status: course.OfferingStatusVisible},
		{Id: 905, CourseId: 46, TermId: 101, Campus: "四平路校区", Faculty: "化学科学与工程学院", Status: course.OfferingStatusVisible},
	} {
		if err := conn.Create(offering).Error; err != nil {
			t.Fatalf("create offering %d: %v", offering.Id, err)
		}
	}
	for _, link := range []*course.OfferingInstructorEntity{
		{OfferingId: 904, InstructorId: 202, Role: "lecturer"}, // 李四
		{OfferingId: 905, InstructorId: 201, Role: "lecturer"}, // 张三
	} {
		if err := conn.Create(link).Error; err != nil {
			t.Fatalf("create offering instructor link: %v", err)
		}
	}
	for _, st := range []*course.CourseStatsEntity{
		{CourseId: 45, RatingCount: 3, RatingSum: 12, ReviewCount: 5},
		{CourseId: 46, RatingCount: 1, RatingSum: 5, ReviewCount: 2},
	} {
		if err := conn.Create(st).Error; err != nil {
			t.Fatalf("create course stats: %v", err)
		}
	}
}

// courseListContract 是 CourseListResponse.result 的黑盒 JSON 视图，仅按契约字段解析，
// 不引用后端 Go 结构体。
type courseListContract struct {
	List []struct {
		Id          uint64  `json:"id"`
		RatingAvg   float64 `json:"ratingAvg"`
		ReviewCount int     `json:"reviewCount"`
	} `json:"list"`
}

func decodeCourseList(t *testing.T, rec *httptest.ResponseRecorder) courseListContract {
	t.Helper()
	var list courseListContract
	if err := json.Unmarshal(decodeContractEnvelope(t, rec).Result, &list); err != nil {
		t.Fatalf("decode course list %q: %v", rec.Body.String(), err)
	}
	return list
}

func courseListIDs(list courseListContract) []uint64 {
	ids := make([]uint64, 0, len(list.List))
	for _, item := range list.List {
		ids = append(ids, item.Id)
	}
	return ids
}

func assertCourseIDSet(t *testing.T, list courseListContract, want ...uint64) {
	t.Helper()
	got := make(map[uint64]bool, len(list.List))
	for _, item := range list.List {
		got[item.Id] = true
	}
	if len(got) != len(want) {
		t.Fatalf("course id set = %v, want %v", got, want)
	}
	for _, id := range want {
		if !got[id] {
			t.Fatalf("course id set = %v, missing %d", got, id)
		}
	}
}

func TestCourseListOnlyWithReviewsHTTPContract(t *testing.T) {
	conn, router := setupCourseContractTest(t)
	seedCourseContractData(t, conn)
	seedCourseFilterData(t, conn)

	rec := serveCourseGet(router, "/api/forum/courses?onlyWithReviews=1")
	if rec.Code != http.StatusOK {
		t.Fatalf("course list onlyWithReviews status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	list := decodeCourseList(t, rec)
	// 基线课程 42 无 CourseStatsEntity 行（无评价）→ 被排除；45/46 有评价 → 保留。
	assertCourseIDSet(t, list, 45, 46)
	for _, item := range list.List {
		if item.ReviewCount == 0 {
			t.Fatalf("onlyWithReviews=1 返回了无评价课程: %+v", item)
		}
	}
}

func TestCourseListInstructorHTTPContract(t *testing.T) {
	conn, router := setupCourseContractTest(t)
	seedCourseContractData(t, conn)
	seedCourseFilterData(t, conn)

	rec := serveCourseGet(router, "/api/forum/courses?instructor="+url.QueryEscape("张三"))
	if rec.Code != http.StatusOK {
		t.Fatalf("course list instructor status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	list := decodeCourseList(t, rec)
	// 42 高等数学（offering 901）与 46 无机化学（offering 905）均由张三授课；45 大学物理由李四授课被排除。
	assertCourseIDSet(t, list, 42, 46)
}

func TestCourseListSortByRatingHTTPContract(t *testing.T) {
	conn, router := setupCourseContractTest(t)
	seedCourseContractData(t, conn)
	seedCourseFilterData(t, conn)

	rec := serveCourseGet(router, "/api/forum/courses?sortBy=rating")
	if rec.Code != http.StatusOK {
		t.Fatalf("course list sortBy status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	list := decodeCourseList(t, rec)
	// 46 平均分 5.0 → 45 平均分 4.0 → 42 无评分垫底；同分时 id 降序兜底。
	want := []uint64{46, 45, 42}
	if got := courseListIDs(list); !reflect.DeepEqual(got, want) {
		t.Fatalf("sortBy=rating ids = %v, want %v", got, want)
	}
}
