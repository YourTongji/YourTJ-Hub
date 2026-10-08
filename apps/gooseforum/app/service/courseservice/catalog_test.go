package courseservice

import (
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/course"
	"gorm.io/gorm"
)

// catalogTestModels 目录筛选测试用到的 course 域表（含 course_alias，buildSummaries 会查询）。
var catalogTestModels = []any{
	&course.Entity{},
	&course.AliasEntity{},
	&course.TermEntity{},
	&course.OfferingEntity{},
	&course.InstructorEntity{},
	&course.OfferingInstructorEntity{},
	&course.CourseStatsEntity{},
	&course.OfferingStatsEntity{},
	&course.ReviewEntity{},
}

// setupCatalogTest 迁移并清空目录筛选相关表（共享全局连接，与 courseservice 其它测试一致）。
func setupCatalogTest(t testing.TB) *gorm.DB {
	t.Helper()
	conn := dbconnect.Connect()
	models := append(catalogTestModels, &course.RelationEntity{})
	if err := conn.AutoMigrate(models...); err != nil {
		t.Fatalf("migrate catalog tables: %v", err)
	}
	for _, model := range models {
		if err := conn.Unscoped().Where("1 = 1").Delete(model).Error; err != nil {
			t.Fatalf("clean catalog table: %v", err)
		}
	}
	InvalidateCatalogFacetsCache()
	InvalidateAllCourseDetailCache()
	return conn
}

// createCatalogCourse 创建一门可见课程，返回课程 ID。
func createCatalogCourse(t *testing.T, conn *gorm.DB, code, department string) uint64 {
	t.Helper()
	c := course.Entity{PrimaryCode: code, Name: "课程" + code, Department: department, Status: course.StatusVisible}
	if err := conn.Create(&c).Error; err != nil {
		t.Fatalf("create course: %v", err)
	}
	return c.Id
}

// setCatalogStats 写入课程级评价统计。
func setCatalogStats(t *testing.T, conn *gorm.DB, courseId uint64, ratingCount, ratingSum, reviewCount int) {
	t.Helper()
	st := course.CourseStatsEntity{CourseId: courseId, RatingCount: ratingCount, RatingSum: ratingSum, ReviewCount: reviewCount}
	if err := conn.Create(&st).Error; err != nil {
		t.Fatalf("create course stats: %v", err)
	}
}

// TestListCatalogStatsBackfill 列表摘要的评价统计回填：
// 有评分行 → 用 GetCourseStatsMap 回填真实 ratingAvg/ratingCount/reviewCount；
// 无评分行（map 缺失）→ 三字段取零值。
func TestListCatalogStatsBackfill(t *testing.T) {
	conn := setupCatalogTest(t)
	withStats := createCatalogCourse(t, conn, "200001", "CS")
	noStats := createCatalogCourse(t, conn, "200002", "CS")
	setCatalogStats(t, conn, withStats, 2, 9, 5)

	page, err := ListCatalog(CatalogQuery{Page: 1, Size: 50})
	if err != nil {
		t.Fatalf("ListCatalog err = %v", err)
	}
	if page.Total != 2 || len(page.List) != 2 {
		t.Fatalf("ListCatalog total=%d len=%d, want 2/2", page.Total, len(page.List))
	}
	byID := make(map[uint64]CourseSummary, len(page.List))
	for _, s := range page.List {
		byID[s.Id] = s
	}
	if got := byID[withStats]; got.RatingAvg == nil || *got.RatingAvg != 4.5 || got.ReviewCount != 5 {
		t.Fatalf("withStats summary = %#v, want RatingAvg 4.5 ReviewCount 5", got)
	}
	if got := byID[noStats]; got.RatingAvg != nil || got.ReviewCount != 0 {
		t.Fatalf("noStats summary = %#v, want nil RatingAvg and zero ReviewCount", got)
	}
}

// TestListCatalogPassThrough 新筛选条件从 service 透传到 repo（HasReview 收窄、SortBy=rating 生效并回填统计）。
func TestListCatalogPassThrough(t *testing.T) {
	conn := setupCatalogTest(t)
	c1 := createCatalogCourse(t, conn, "200010", "CS")
	c2 := createCatalogCourse(t, conn, "200011", "CS")
	setCatalogStats(t, conn, c1, 1, 5, 2)
	setCatalogStats(t, conn, c2, 1, 3, 1)

	// HasReview=true 只返回有评价课程；这里两门都有评价，故 total=2。
	only, err := ListCatalog(CatalogQuery{HasReview: true, Page: 1, Size: 50})
	if err != nil {
		t.Fatalf("ListCatalog(HasReview) err = %v", err)
	}
	if only.Total != 2 || len(only.List) != 2 {
		t.Fatalf("ListCatalog(HasReview) total=%d len=%d, want 2/2", only.Total, len(only.List))
	}

	// SortBy=rating 按平均分降序：c1(5.0) 在 c2(3.0) 前，且摘要回填真实评分。
	rated, err := ListCatalog(CatalogQuery{SortBy: "rating", Page: 1, Size: 50})
	if err != nil {
		t.Fatalf("ListCatalog(sortBy=rating) err = %v", err)
	}
	if len(rated.List) != 2 || rated.List[0].Id != c1 || rated.List[1].Id != c2 {
		t.Fatalf("ListCatalog(sortBy=rating) ids = [%d,%d], want [%d,%d]",
			rated.List[0].Id, rated.List[1].Id, c1, c2)
	}
	if rated.List[0].RatingAvg == nil || *rated.List[0].RatingAvg != 5.0 || rated.List[1].RatingAvg == nil || *rated.List[1].RatingAvg != 3.0 {
		t.Fatalf("ListCatalog(sortBy=rating) avgs = [%v,%v], want [5,3]", rated.List[0].RatingAvg, rated.List[1].RatingAvg)
	}
}

// TestListDepartments service 层院系列表透传 repo：去重、排序、排除空/隐藏/软删。
func TestListDepartments(t *testing.T) {
	conn := setupCatalogTest(t)
	for i, dept := range []string{"CS", "Math", "Physics", "CS"} {
		createCatalogCourse(t, conn, "21000"+string(rune('0'+i)), dept)
	}
	empty := course.Entity{PrimaryCode: "210010", Name: "无院系", Department: "", Status: course.StatusVisible}
	if err := conn.Create(&empty).Error; err != nil {
		t.Fatalf("create empty-dept course: %v", err)
	}
	hidden := course.Entity{PrimaryCode: "210011", Name: "隐藏课", Department: "HiddenDept", Status: course.StatusHidden}
	if err := conn.Create(&hidden).Error; err != nil {
		t.Fatalf("create hidden course: %v", err)
	}
	ghost := course.Entity{PrimaryCode: "210012", Name: "软删课", Department: "GhostDept", Status: course.StatusVisible}
	if err := conn.Create(&ghost).Error; err != nil {
		t.Fatalf("create soft-delete course: %v", err)
	}
	if err := conn.Delete(&course.Entity{Id: ghost.Id}).Error; err != nil {
		t.Fatalf("soft-delete course: %v", err)
	}

	got, err := ListDepartments()
	if err != nil {
		t.Fatalf("ListDepartments err = %v", err)
	}
	want := []string{"CS", "Math", "Physics"}
	if len(got) != len(want) {
		t.Fatalf("ListDepartments = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ListDepartments = %v, want %v", got, want)
		}
	}
}

// TestCatalogFacetsCacheAndInvalidation verifies that facet listings use caching and update on invalidation.
func TestCatalogFacetsCacheAndInvalidation(t *testing.T) {
	conn := setupCatalogTest(t)
	createCatalogCourse(t, conn, "CS101", "Computer Science")

	depts, err := ListDepartments()
	if err != nil {
		t.Fatalf("ListDepartments failed: %v", err)
	}
	if len(depts) != 1 || depts[0] != "Computer Science" {
		t.Fatalf("unexpected departments: %v", depts)
	}

	// Mutating the returned slice should not corrupt cached data (defensive copy)
	depts[0] = "Tampered"
	deptsAgain, err := ListDepartments()
	if err != nil {
		t.Fatalf("ListDepartments again failed: %v", err)
	}
	if len(deptsAgain) != 1 || deptsAgain[0] != "Computer Science" {
		t.Fatalf("cached slice was mutated: %v", deptsAgain)
	}

	// Insert another course directly into DB without invalidating cache
	createCatalogCourse(t, conn, "MATH101", "Mathematics")

	// Calling ListDepartments should still return cached result
	deptsCached, err := ListDepartments()
	if err != nil {
		t.Fatalf("ListDepartments cached call failed: %v", err)
	}
	if len(deptsCached) != 1 || deptsCached[0] != "Computer Science" {
		t.Fatalf("expected cached single department, got: %v", deptsCached)
	}

	// Invalidate cache and call again; should reflect new department
	InvalidateCatalogFacetsCache()
	deptsUpdated, err := ListDepartments()
	if err != nil {
		t.Fatalf("ListDepartments updated call failed: %v", err)
	}
	if len(deptsUpdated) != 2 || deptsUpdated[0] != "Computer Science" || deptsUpdated[1] != "Mathematics" {
		t.Fatalf("expected updated departments [Computer Science, Mathematics], got: %v", deptsUpdated)
	}
}

func setupCourseDetailFixture(tb testing.TB, conn *gorm.DB) uint64 {
	tb.Helper()
	instructor := course.InstructorEntity{Name: "张三", Department: "电信学院"}
	if err := conn.Create(&instructor).Error; err != nil {
		tb.Fatalf("create instructor: %v", err)
	}

	c := course.Entity{
		PrimaryCode: "1001",
		Name:        "软件工程",
		Department:  "电信学院",
		CreditX10:   30,
		TeacherId:   instructor.Id,
		Status:      course.StatusVisible,
	}
	if err := conn.Create(&c).Error; err != nil {
		tb.Fatalf("create course: %v", err)
	}

	alias := course.AliasEntity{CourseId: c.Id, Value: "SE101"}
	if err := conn.Create(&alias).Error; err != nil {
		tb.Fatalf("create alias: %v", err)
	}

	term := course.TermEntity{Code: "2026-AUTUMN", Name: "2026年秋季学期"}
	if err := conn.Create(&term).Error; err != nil {
		tb.Fatalf("create term: %v", err)
	}

	offering := course.OfferingEntity{
		CourseId:  c.Id,
		TermId:    term.Id,
		Campus:    "嘉定校区",
		Faculty:   "电信学院",
		ClassCode: "01001",
		ClassName: "01班",
		Status:    course.StatusVisible,
	}
	if err := conn.Create(&offering).Error; err != nil {
		tb.Fatalf("create offering: %v", err)
	}

	offeringIns := course.OfferingInstructorEntity{
		OfferingId:   offering.Id,
		InstructorId: instructor.Id,
	}
	if err := conn.Create(&offeringIns).Error; err != nil {
		tb.Fatalf("create offering instructor: %v", err)
	}

	courseStats := course.CourseStatsEntity{
		CourseId:    c.Id,
		RatingCount: 10,
		RatingSum:   45,
		ReviewCount: 10,
	}
	if err := conn.Create(&courseStats).Error; err != nil {
		tb.Fatalf("create course stats: %v", err)
	}

	offeringStats := course.OfferingStatsEntity{
		OfferingId:  offering.Id,
		RatingCount: 5,
		RatingSum:   23,
		ReviewCount: 5,
	}
	if err := conn.Create(&offeringStats).Error; err != nil {
		tb.Fatalf("create offering stats: %v", err)
	}

	return c.Id
}

func BenchmarkGetCourseDetail(b *testing.B) {
	conn := setupCatalogTest(b)
	courseId := setupCourseDetailFixture(b, conn)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		detail, err := GetCourseDetail(courseId)
		if err != nil {
			b.Fatalf("GetCourseDetail failed: %v", err)
		}
		if detail.Id != courseId {
			b.Fatalf("unexpected detail id: %d", detail.Id)
		}
	}
}

func TestGetCourseDetailCacheAndIsolation(t *testing.T) {
	conn := setupCatalogTest(t)
	courseId := setupCourseDetailFixture(t, conn)

	detail1, err := GetCourseDetail(courseId)
	if err != nil {
		t.Fatalf("first GetCourseDetail failed: %v", err)
	}
	if detail1.Name != "软件工程" || len(detail1.Aliases) != 1 || detail1.Aliases[0] != "SE101" {
		t.Fatalf("unexpected initial detail: %+v", detail1)
	}
	if len(detail1.Offerings) != 1 || len(detail1.Offerings[0].Instructors) != 1 {
		t.Fatalf("unexpected offerings: %+v", detail1.Offerings)
	}

	// 1. Defensive cloning: caller mutation must not affect cached values
	detail1.Aliases[0] = "MUTATED_ALIAS"
	detail1.Offerings[0].Instructors[0] = "MUTATED_INSTRUCTOR"
	if detail1.RatingAvg != nil {
		*detail1.RatingAvg = 99.9
	}
	if detail1.RatingDistribution != nil {
		detail1.RatingDistribution[0] = 999
	}

	detail2, err := GetCourseDetail(courseId)
	if err != nil {
		t.Fatalf("second GetCourseDetail failed: %v", err)
	}
	if detail2.Aliases[0] == "MUTATED_ALIAS" {
		t.Fatalf("cached alias was mutated: %v", detail2.Aliases)
	}
	if detail2.Offerings[0].Instructors[0] == "MUTATED_INSTRUCTOR" {
		t.Fatalf("cached instructor was mutated: %v", detail2.Offerings[0].Instructors)
	}
	if detail2.RatingAvg != nil && *detail2.RatingAvg == 99.9 {
		t.Fatalf("cached rating avg was mutated: %v", *detail2.RatingAvg)
	}

	// 2. Cache hit: updating DB directly without invalidation serves cached data
	if err := conn.Model(&course.Entity{}).Where("id = ?", courseId).Update("name", "更新课程名").Error; err != nil {
		t.Fatalf("direct db update: %v", err)
	}
	detailCached, err := GetCourseDetail(courseId)
	if err != nil {
		t.Fatalf("GetCourseDetail cached failed: %v", err)
	}
	if detailCached.Name != "软件工程" {
		t.Fatalf("expected cached name 软件工程, got %s", detailCached.Name)
	}

	// 3. Single course invalidation
	InvalidateCourseDetailCache(courseId)
	detailReloaded, err := GetCourseDetail(courseId)
	if err != nil {
		t.Fatalf("GetCourseDetail reloaded failed: %v", err)
	}
	if detailReloaded.Name != "更新课程名" {
		t.Fatalf("expected reloaded name 更新课程名, got %s", detailReloaded.Name)
	}

	// 4. InvalidateAllCourseDetailCache
	if err := conn.Model(&course.Entity{}).Where("id = ?", courseId).Update("name", "再次更新课程名").Error; err != nil {
		t.Fatalf("second db update: %v", err)
	}
	InvalidateAllCourseDetailCache()
	detailAllReloaded, err := GetCourseDetail(courseId)
	if err != nil {
		t.Fatalf("GetCourseDetail all reloaded failed: %v", err)
	}
	if detailAllReloaded.Name != "再次更新课程名" {
		t.Fatalf("expected reloaded name 再次更新课程名, got %s", detailAllReloaded.Name)
	}
}
