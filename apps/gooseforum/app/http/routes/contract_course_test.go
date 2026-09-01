package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
	"time"

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
		&course.ReviewEntity{},
		&course.HelpfulEntity{},
		&course.CourseStatsEntity{},
		&course.OfferingStatsEntity{},
		&course.CourseAiSummaryEntity{},
		&course.RelationEntity{},
	); err != nil {
		t.Fatalf("migrate course contract tables: %v", err)
	}
	// 清空课程域表，保证 fixture 断言确定性。
	for _, model := range []any{
		&course.HelpfulEntity{},
		&course.ReviewEntity{},
		&course.SourceRefEntity{},
		&course.ImportRunEntity{},
		&course.OfferingInstructorEntity{},
		&course.OfferingEntity{},
		&course.InstructorEntity{},
		&course.TermEntity{},
		&course.AliasEntity{},
		&course.CourseStatsEntity{},
		&course.OfferingStatsEntity{},
		&course.CourseAiSummaryEntity{},
		&course.RelationEntity{},
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
	router.GET("/api/forum/courses/:courseId/summary", UpUriQueryReq(forum.GetCourseSummary))
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
		Id:        901,
		CourseId:  entity.Id,
		TermId:    term.Id,
		Campus:    "四平路校区",
		Faculty:   "数学科学学院",
		ClassCode: "10000101",
		ClassName: "01班",
		Status:    course.OfferingStatusVisible,
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
	assertFixtureEnvelope(t, decodeContractEnvelope(t, rec), contractFixture(t, "parse-failed.json"))
}

// seedCourseRelatedData 写入与 course-related-success fixture 一致的相关课程数据：
// 课程 42（100001 高等数学，offering 901 由张三+李四授课）的同教师其他课为课程 43
// 线性代数（张三）；同课号其他教师卡为课程 47（100001 高等数学，身份教师李四）——
// (code, teacher) 复合身份模型下同课号不同教师是独立课程卡。
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
	// 同课号其他教师卡：课程 47 与 42 同 code 100001，身份教师李四（202）。
	if err := conn.Create(&course.Entity{
		Id:             47,
		PrimaryCode:    "100001",
		Name:           "高等数学(A)上",
		Department:     "数学科学学院",
		CreditX10:      50,
		NormalizedName: "高等数学a上",
		NamePinyin:     "gaodengshuxueashang",
		NameInitials:   "gdsxas",
		TeacherId:      202, // 李四
		Status:         course.StatusVisible,
	}).Error; err != nil {
		t.Fatalf("create same-code course 47: %v", err)
	}
	// 更早学期：offering 902 使用 term 102（code 小于 term 101，排序在 901 之后）。
	if err := conn.Create(&course.TermEntity{Id: 102, Code: "2024-2025-1", Name: "2024-2025 第一学期", Status: 0}).Error; err != nil {
		t.Fatalf("create term 102: %v", err)
	}
	for _, offering := range []*course.OfferingEntity{
		{Id: 902, CourseId: 42, TermId: 102, Campus: "四平路校区", Faculty: "数学科学学院", Status: course.OfferingStatusVisible},
		{Id: 903, CourseId: 43, TermId: 101, Campus: "四平路校区", Faculty: "数学科学学院", Status: course.OfferingStatusVisible},
		{Id: 904, CourseId: 47, TermId: 101, Campus: "四平路校区", Faculty: "数学科学学院", Status: course.OfferingStatusVisible},
	} {
		if err := conn.Create(offering).Error; err != nil {
			t.Fatalf("create offering %d: %v", offering.Id, err)
		}
	}
	for _, link := range []*course.OfferingInstructorEntity{
		{OfferingId: 902, InstructorId: 202, Role: "lecturer"}, // 李四
		{OfferingId: 903, InstructorId: 201, Role: "lecturer"}, // 张三
		{OfferingId: 904, InstructorId: 202, Role: "lecturer"}, // 李四
	} {
		if err := conn.Create(link).Error; err != nil {
			t.Fatalf("create offering instructor link: %v", err)
		}
	}
	for _, st := range []*course.CourseStatsEntity{
		{CourseId: 42, RatingCount: 2, RatingSum: 9, ReviewCount: 2},
		{CourseId: 43, RatingCount: 2, RatingSum: 9, ReviewCount: 2},
		{CourseId: 47, RatingCount: 1, RatingSum: 4, ReviewCount: 1},
	} {
		if err := conn.Create(st).Error; err != nil {
			t.Fatalf("create course stats: %v", err)
		}
	}
	for _, st := range []*course.OfferingStatsEntity{
		{OfferingId: 901, RatingCount: 1, RatingSum: 5, ReviewCount: 1},
		{OfferingId: 902, RatingCount: 1, RatingSum: 4, ReviewCount: 1},
		{OfferingId: 903, RatingCount: 2, RatingSum: 9, ReviewCount: 2},
		{OfferingId: 904, RatingCount: 1, RatingSum: 4, ReviewCount: 1},
	} {
		if err := conn.Create(st).Error; err != nil {
			t.Fatalf("create offering stats: %v", err)
		}
	}
	// 沿革数据：旧卡 41（已合并隐藏）→ 当前卡 42，relation 11 merged。
	// buildLineageItems 语义：direction=to（本卡为当前卡），fromName = 旧卡名，toName = 本卡名。
	if err := conn.Create(&course.Entity{
		Id:             41,
		PrimaryCode:    "099999", // 2025 改制前旧课程码；(code, teacher) 唯一索引下与 42 的 100001 不冲突
		Name:           "高等数学(A)上",
		Department:     "数学科学学院",
		CreditX10:      50,
		NormalizedName: "高等数学a上",
		NamePinyin:     "gaodengshuxueashang",
		NameInitials:   "gdsxas",
		Status:         course.StatusHidden, // 合并后旧卡隐藏，名称仍可解析（GetMapByIds 无状态过滤）
	}).Error; err != nil {
		t.Fatalf("create lineage old card 41: %v", err)
	}
	if err := conn.Create(&course.RelationEntity{
		Id:           11,
		FromCourseId: 41,
		ToCourseId:   42,
		RelationType: string(course.RelationEquivalent),
		Source:       course.RelationSourceRule,
		Status:       string(course.RelationStatusMerged),
		EvidenceJson: `{"originalEvidence":"rule","merge":{"movedOfferingIds":[]}}`,
	}).Error; err != nil {
		t.Fatalf("create lineage relation 11: %v", err)
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
	assertFixtureEnvelope(t, decodeContractEnvelope(t, rec), contractFixture(t, "parse-failed.json"))
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
		Id          uint64   `json:"id"`
		RatingAvg   *float64 `json:"ratingAvg,omitempty"`
		ReviewCount int      `json:"reviewCount"`
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

// TestCourseStatsSecurityFindings 验证双审修复（security F1/F2/F4）：
//  1. deleted_at 软删行不计入 ratingDistribution（与 stats 投影口径一致）；
//  2. 隐藏 offering 的评价不计入课程级 ratingAvg/reviewCount/distribution；
//  3. 无评价课程 ratingDistribution 省略（F4 指针语义）。
func TestCourseStatsSecurityFindings(t *testing.T) {
	conn, router := setupCourseContractTest(t)
	seedCourseContractData(t, conn)
	if err := conn.AutoMigrate(&course.ReviewEntity{}, &course.CourseStatsEntity{}, &course.OfferingStatsEntity{}); err != nil {
		t.Fatalf("migrate review/stats: %v", err)
	}
	conn.Unscoped().Where("1 = 1").Delete(&course.ReviewEntity{})
	conn.Unscoped().Where("1 = 1").Delete(&course.CourseStatsEntity{})
	conn.Unscoped().Where("1 = 1").Delete(&course.OfferingStatsEntity{})

	// 3 条可见评价 + 1 条软删评价（deleted_at 置位）
	r5, r4, r3 := 5, 4, 3
	seedCourseReview(t, conn, 2001, 901, 501, &r5, "五星", false, "", course.ReviewStatusVisible)
	seedCourseReview(t, conn, 2002, 901, 502, &r4, "四星", false, "", course.ReviewStatusVisible)
	seedCourseReview(t, conn, 2003, 901, 503, &r3, "三星", false, "", course.ReviewStatusVisible)
	// 软删行：status=visible 但 deleted_at 非空（模拟清理作业/数据迁移置位）
	if err := conn.Unscoped().Model(&course.ReviewEntity{}).Where("id = ?", 2003).
		Update("deleted_at", time.Now()).Error; err != nil {
		t.Fatalf("soft-delete review 2003: %v", err)
	}
	if err := course.RebuildAllCourseStats(); err != nil {
		t.Fatalf("rebuild stats: %v", err)
	}

	// 详情：ratingAvg=(5+4)/2=4.5、reviewCount=2（软删行不计）、distribution=[0,0,0,1,1]
	rec := serveCourseGet(router, "/api/forum/courses/42")
	var detailEnvelope struct {
		Result map[string]any `json:"result"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &detailEnvelope)
	detail := detailEnvelope.Result
	if got := detail["ratingAvg"]; got != 4.5 {
		t.Fatalf("ratingAvg = %#v, want 4.5 (soft-deleted row excluded)", got)
	}
	if got := detail["reviewCount"]; got != float64(2) {
		t.Fatalf("reviewCount = %#v, want 2 (soft-deleted row excluded)", got)
	}
	dist, ok := detail["ratingDistribution"].([]any)
	if !ok {
		t.Fatalf("ratingDistribution missing: %#v", detail["ratingDistribution"])
	}
	if dist[4] != float64(1) || dist[3] != float64(1) || dist[0] != float64(0) {
		t.Fatalf("ratingDistribution = %v, want [0,0,0,1,1] (soft-deleted row excluded)", dist)
	}

	// F4：无评价课程 distribution 省略
	var clean struct {
		Result map[string]any `json:"result"`
	}
	// 造一个无评价课程
	if err := conn.Create(&course.Entity{Id: 43, PrimaryCode: "100002", Name: "无评价课", Department: "数学", CreditX10: 20, NormalizedName: "无评价课", Status: course.StatusVisible}).Error; err != nil {
		t.Fatalf("create empty course: %v", err)
	}
	rec = serveCourseGet(router, "/api/forum/courses/43")
	_ = json.Unmarshal(rec.Body.Bytes(), &clean)
	if _, present := clean.Result["ratingDistribution"]; present {
		t.Fatalf("empty course should omit ratingDistribution, got %#v", clean.Result["ratingDistribution"])
	}
	if _, present := clean.Result["ratingAvg"]; present {
		t.Fatalf("empty course should omit ratingAvg, got %#v", clean.Result["ratingAvg"])
	}

	// F2：隐藏 offering 的评价不计入课程级聚合
	// 造第二个 offering（offering 902，同 course 42），给 1 条评价，然后隐藏 offering
	if err := conn.Create(&course.OfferingEntity{Id: 902, CourseId: 42, TermId: 101, Status: course.OfferingStatusVisible}).Error; err != nil {
		t.Fatalf("create offering 902: %v", err)
	}
	seedCourseReview(t, conn, 2004, 902, 504, &r5, "隐藏开课评价", false, "", course.ReviewStatusVisible)
	if err := conn.Model(&course.OfferingEntity{}).Where("id = ?", 902).Update("status", course.OfferingStatusHidden).Error; err != nil {
		t.Fatalf("hide offering 902: %v", err)
	}
	if err := course.RebuildAllCourseStats(); err != nil {
		t.Fatalf("rebuild stats after hide offering: %v", err)
	}
	rec = serveCourseGet(router, "/api/forum/courses/42")
	_ = json.Unmarshal(rec.Body.Bytes(), &detailEnvelope)
	detail = detailEnvelope.Result
	if got := detail["reviewCount"]; got != float64(2) {
		t.Fatalf("reviewCount with hidden offering = %#v, want 2 (hidden offering excluded)", got)
	}
	if got := detail["ratingAvg"]; got != 4.5 {
		t.Fatalf("ratingAvg with hidden offering = %#v, want 4.5", got)
	}
}
