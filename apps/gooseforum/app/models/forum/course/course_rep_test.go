package course

import (
	"slices"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"gorm.io/gorm"
)

// courseRepTestModels 目录筛选测试用到的表。
var courseRepTestModels = []any{
	&Entity{},
	&OfferingEntity{},
	&InstructorEntity{},
	&OfferingInstructorEntity{},
	&CourseStatsEntity{},
}

// setupCourseRepTest 迁移并清空目录筛选相关表（共享全局连接，与 course 域其它测试一致）。
func setupCourseRepTest(t *testing.T) *gorm.DB {
	t.Helper()
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(courseRepTestModels...); err != nil {
		t.Fatalf("migrate course rep tables: %v", err)
	}
	for _, model := range courseRepTestModels {
		if err := conn.Unscoped().Where("1 = 1").Delete(model).Error; err != nil {
			t.Fatalf("clean course rep table: %v", err)
		}
	}
	return conn
}

// createCourse 创建一门可见课程，返回课程 ID。
func createCourse(t *testing.T, conn *gorm.DB, code, department string) uint64 {
	t.Helper()
	c := Entity{PrimaryCode: code, Name: "课程" + code, Department: department, Status: StatusVisible}
	if err := conn.Create(&c).Error; err != nil {
		t.Fatalf("create course: %v", err)
	}
	return c.Id
}

// linkCourseInstructor 为课程创建可见 offering 并关联教师（教师记录需已存在）。
func linkCourseInstructor(t *testing.T, conn *gorm.DB, courseId, instructorId uint64) {
	t.Helper()
	offering := OfferingEntity{CourseId: courseId, Status: OfferingStatusVisible}
	if err := conn.Create(&offering).Error; err != nil {
		t.Fatalf("create offering: %v", err)
	}
	link := OfferingInstructorEntity{OfferingId: offering.Id, InstructorId: instructorId}
	if err := conn.Create(&link).Error; err != nil {
		t.Fatalf("create offering-instructor link: %v", err)
	}
}

// createTestInstructor 创建教师（覆盖归一化名/拼音/首字母四列，用于验证 LIKE 匹配路径）。
func createTestInstructor(t *testing.T, conn *gorm.DB, name, normalized, pinyin, initials string) uint64 {
	t.Helper()
	ins := InstructorEntity{Name: name, NormalizedName: normalized, NamePinyin: pinyin, NameInitials: initials}
	if err := conn.Create(&ins).Error; err != nil {
		t.Fatalf("create instructor: %v", err)
	}
	return ins.Id
}

// setCourseStats 写入课程级评价统计。
func setCourseStats(t *testing.T, conn *gorm.DB, courseId uint64, ratingCount, ratingSum, reviewCount int) {
	t.Helper()
	st := CourseStatsEntity{CourseId: courseId, RatingCount: ratingCount, RatingSum: ratingSum, ReviewCount: reviewCount}
	if err := conn.Create(&st).Error; err != nil {
		t.Fatalf("create course stats: %v", err)
	}
}

// courseIDs 提取实体的 ID 序列，便于断言排序结果。
func courseIDs(entities []Entity) []uint64 {
	ids := make([]uint64, 0, len(entities))
	for _, e := range entities {
		ids = append(ids, e.Id)
	}
	return ids
}

// TestListCoursesHasReview HasReview 过滤：仅 review_count > 0 的课程进入结果，且 COUNT 同步收窄。
func TestListCoursesHasReview(t *testing.T) {
	conn := setupCourseRepTest(t)
	withReviews := createCourse(t, conn, "100001", "CS")
	zeroReviews := createCourse(t, conn, "100002", "CS")
	_ = createCourse(t, conn, "100003", "CS")     // 无 stats 行，HasReview 同样应排除
	setCourseStats(t, conn, withReviews, 2, 8, 3) // review_count > 0
	setCourseStats(t, conn, zeroReviews, 0, 0, 0) // review_count == 0（有行但零评价）

	t.Run("only_with_reviews", func(t *testing.T) {
		got, total, err := ListCourses(ListCourseQuery{HasReview: true, Size: 50})
		if err != nil {
			t.Fatalf("ListCourses err = %v", err)
		}
		if total != 1 || !slices.Equal(courseIDs(got), []uint64{withReviews}) {
			t.Fatalf("HasReview=true: total=%d ids=%v, want total=1 ids=[%d]", total, courseIDs(got), withReviews)
		}
	})

	t.Run("all_courses", func(t *testing.T) {
		got, total, err := ListCourses(ListCourseQuery{Size: 50})
		if err != nil {
			t.Fatalf("ListCourses err = %v", err)
		}
		if total != 3 || len(got) != 3 {
			t.Fatalf("HasReview unset: total=%d len=%d, want 3/3", total, len(got))
		}
	})
}

// TestListCoursesInstructor Instructor 过滤：%v% LIKE 命中教师四列中的任一列即可，且独立于 keyword。
func TestListCoursesInstructor(t *testing.T) {
	conn := setupCourseRepTest(t)
	zhang := createTestInstructor(t, conn, "张三", "zhangsan", "zhangsan", "zs")
	li := createTestInstructor(t, conn, "李四", "lisi", "lisi", "ls")
	cA := createCourse(t, conn, "100010", "CS")
	cB := createCourse(t, conn, "100011", "CS")
	_ = createCourse(t, conn, "100012", "CS") // 无 offering，不应被教师筛选命中
	linkCourseInstructor(t, conn, cA, zhang)
	linkCourseInstructor(t, conn, cB, li)

	cases := []struct {
		name   string
		input  string
		wantID uint64
	}{
		{"chinese_partial", "张", cA},
		{"chinese_exact", "张三", cA},
		{"normalized_name", "zhangsan", cA},
		{"pinyin_partial", "zhang", cA},
		{"initials", "zs", cA},
		{"other_teacher", "李", cB},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, total, err := ListCourses(ListCourseQuery{Instructor: tc.input, Size: 50})
			if err != nil {
				t.Fatalf("ListCourses(instructor=%q) err = %v", tc.input, err)
			}
			if total != 1 || !slices.Equal(courseIDs(got), []uint64{tc.wantID}) {
				t.Fatalf("instructor=%q: total=%d ids=%v, want total=1 ids=[%d]", tc.input, total, courseIDs(got), tc.wantID)
			}
		})
	}

	t.Run("no_match", func(t *testing.T) {
		got, total, err := ListCourses(ListCourseQuery{Instructor: "王", Size: 50})
		if err != nil {
			t.Fatalf("ListCourses err = %v", err)
		}
		if total != 0 || len(got) != 0 {
			t.Fatalf("instructor=王: total=%d len=%d, want 0/0", total, len(got))
		}
	})
}

// TestListCoursesSortByRating SortBy=rating 按平均分降序，零/无评分课程排末尾（id 倒序兜底）；
// COUNT 与排序无关（LEFT JOIN 不放大计数），默认排序仍为 id 倒序。
func TestListCoursesSortByRating(t *testing.T) {
	conn := setupCourseRepTest(t)
	// 创建顺序即 id 升序：a(avg4.0) b(avg5.0) c(avg3.0) d(无评分行) e(rating_count=0)。
	a := createCourse(t, conn, "100020", "CS")
	b := createCourse(t, conn, "100021", "CS")
	c := createCourse(t, conn, "100022", "CS")
	d := createCourse(t, conn, "100023", "CS")
	e := createCourse(t, conn, "100024", "CS")
	setCourseStats(t, conn, a, 2, 8, 2)
	setCourseStats(t, conn, b, 1, 5, 1)
	setCourseStats(t, conn, c, 1, 3, 1)
	setCourseStats(t, conn, e, 0, 0, 0)

	got, total, err := ListCourses(ListCourseQuery{SortBy: "rating", Size: 50})
	if err != nil {
		t.Fatalf("ListCourses(sortBy=rating) err = %v", err)
	}
	// 有评分者按平均分降序（b 5.0 > a 4.0 > c 3.0），零评分者排末尾按 id 降序（e > d）。
	want := []uint64{b, a, c, e, d}
	if total != 5 || !slices.Equal(courseIDs(got), want) {
		t.Fatalf("sortBy=rating: total=%d ids=%v, want total=5 ids=%v", total, courseIDs(got), want)
	}

	gotDefault, totalDefault, err := ListCourses(ListCourseQuery{Size: 50})
	if err != nil {
		t.Fatalf("ListCourses(default) err = %v", err)
	}
	wantDefault := []uint64{e, d, c, b, a}
	if totalDefault != 5 || !slices.Equal(courseIDs(gotDefault), wantDefault) {
		t.Fatalf("default sort: total=%d ids=%v, want total=5 ids=%v", totalDefault, courseIDs(gotDefault), wantDefault)
	}
}

// TestListDistinctDepartments 院系列表去重与排序：排除空值、隐藏课程与软删课程。
func TestListDistinctDepartments(t *testing.T) {
	conn := setupCourseRepTest(t)
	_ = createCourse(t, conn, "100030", "Math")
	_ = createCourse(t, conn, "100031", "CS")
	_ = createCourse(t, conn, "100032", "Physics")
	_ = createCourse(t, conn, "100033", "CS") // 重复院系应去重
	_ = createCourse(t, conn, "100034", "")   // 空院系应排除
	hidden := createCourse(t, conn, "100035", "HiddenDept")
	if err := conn.Model(&Entity{}).Where("id = ?", hidden).Update("status", StatusHidden).Error; err != nil {
		t.Fatalf("hide course: %v", err)
	}
	ghost := createCourse(t, conn, "100036", "GhostDept")
	if err := conn.Delete(&Entity{Id: ghost}).Error; err != nil {
		t.Fatalf("soft-delete course: %v", err)
	}

	got, err := ListDistinctDepartments()
	if err != nil {
		t.Fatalf("ListDistinctDepartments err = %v", err)
	}
	want := []string{"CS", "Math", "Physics"}
	if len(got) != len(want) {
		t.Fatalf("ListDistinctDepartments = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ListDistinctDepartments = %v, want %v", got, want)
		}
	}
}
