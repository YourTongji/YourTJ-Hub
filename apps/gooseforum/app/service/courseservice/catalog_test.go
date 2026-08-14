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
}

// setupCatalogTest 迁移并清空目录筛选相关表（共享全局连接，与 courseservice 其它测试一致）。
func setupCatalogTest(t *testing.T) *gorm.DB {
	t.Helper()
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(catalogTestModels...); err != nil {
		t.Fatalf("migrate catalog tables: %v", err)
	}
	for _, model := range catalogTestModels {
		if err := conn.Unscoped().Where("1 = 1").Delete(model).Error; err != nil {
			t.Fatalf("clean catalog table: %v", err)
		}
	}
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
