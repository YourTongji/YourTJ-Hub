package courseservice

import (
	"errors"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/course"
	"gorm.io/gorm"
)

// relatedTestModels 相关课程测试用到的 course 域表。
var relatedTestModels = []any{
	&course.Entity{},
	&course.TermEntity{},
	&course.OfferingEntity{},
	&course.InstructorEntity{},
	&course.OfferingInstructorEntity{},
	&course.CourseStatsEntity{},
	&course.OfferingStatsEntity{},
}

// setupRelatedTest 迁移并清空课程域表（共享全局连接，与现有 course 域测试一致）。
func setupRelatedTest(t *testing.T) *gorm.DB {
	t.Helper()
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(relatedTestModels...); err != nil {
		t.Fatalf("migrate related tables: %v", err)
	}
	for _, model := range relatedTestModels {
		if err := conn.Unscoped().Where("1 = 1").Delete(model).Error; err != nil {
			t.Fatalf("clean related table: %v", err)
		}
	}
	return conn
}

// createRelatedCourse 创建课程与一个开课实例（term code 固定为最大学期，作为"最近开课"）。
// 返回课程 ID。instructorIds 为空时仍创建无教师的 offering（模拟无教师关联的课程）。
func createRelatedCourse(t *testing.T, conn *gorm.DB, primaryCode string, instructorIds []uint64) uint64 {
	t.Helper()
	c := course.Entity{PrimaryCode: primaryCode, Name: "课程" + primaryCode, Department: "数学科学学院", Status: course.StatusVisible}
	if err := conn.Create(&c).Error; err != nil {
		t.Fatalf("create course: %v", err)
	}
	term := course.TermEntity{Code: "2025-2026-1", Name: "学期", Status: 0}
	if err := conn.Where("code = ?", "2025-2026-1").FirstOrCreate(&term).Error; err != nil {
		t.Fatalf("create term: %v", err)
	}
	offering := course.OfferingEntity{CourseId: c.Id, TermId: term.Id, Status: course.OfferingStatusVisible}
	if err := conn.Create(&offering).Error; err != nil {
		t.Fatalf("create offering: %v", err)
	}
	for _, id := range instructorIds {
		if err := conn.Create(&course.OfferingInstructorEntity{OfferingId: offering.Id, InstructorId: id, Role: "lecturer"}).Error; err != nil {
			t.Fatalf("create offering instructor link: %v", err)
		}
	}
	return c.Id
}

// addRelatedOffering 为已有课程追加一个指定学期的开课实例，返回 offering ID。
func addRelatedOffering(t *testing.T, conn *gorm.DB, courseId uint64, termCode string, instructorIds []uint64) uint64 {
	t.Helper()
	term := course.TermEntity{Code: termCode, Name: "学期", Status: 0}
	if err := conn.Where("code = ?", termCode).FirstOrCreate(&term).Error; err != nil {
		t.Fatalf("create term: %v", err)
	}
	offering := course.OfferingEntity{CourseId: courseId, TermId: term.Id, Status: course.OfferingStatusVisible}
	if err := conn.Create(&offering).Error; err != nil {
		t.Fatalf("create offering: %v", err)
	}
	for _, id := range instructorIds {
		if err := conn.Create(&course.OfferingInstructorEntity{OfferingId: offering.Id, InstructorId: id, Role: "lecturer"}).Error; err != nil {
			t.Fatalf("create link: %v", err)
		}
	}
	return offering.Id
}

func createInstructor(t *testing.T, conn *gorm.DB, name string) uint64 {
	t.Helper()
	ins := course.InstructorEntity{Name: name, NormalizedName: name, Department: "数学科学学院"}
	if err := conn.Create(&ins).Error; err != nil {
		t.Fatalf("create instructor: %v", err)
	}
	return ins.Id
}

func TestGetCourseRelatedNotFound(t *testing.T) {
	conn := setupRelatedTest(t)
	hidden := course.Entity{PrimaryCode: "100009", Name: "隐藏课", Department: "数学科学学院", Status: course.StatusHidden}
	if err := conn.Create(&hidden).Error; err != nil {
		t.Fatalf("create hidden course: %v", err)
	}
	if _, err := GetCourseRelated(hidden.Id); !errors.Is(err, ErrCourseNotFound) {
		t.Fatalf("hidden course err = %v, want ErrCourseNotFound", err)
	}
	if _, err := GetCourseRelated(99999999); !errors.Is(err, ErrCourseNotFound) {
		t.Fatalf("missing course err = %v, want ErrCourseNotFound", err)
	}
}

func TestGetCourseRelatedEmptyLists(t *testing.T) {
	conn := setupRelatedTest(t)
	// 无教师关联的可见课程：两个列表均为空，而非报错。
	id := createRelatedCourse(t, conn, "100010", nil)
	related, err := GetCourseRelated(id)
	if err != nil {
		t.Fatalf("GetCourseRelated err = %v", err)
	}
	if len(related.TeacherOtherCourses) != 0 || len(related.SameCourseOtherTeachers) != 0 {
		t.Fatalf("expected empty lists, got %#v", related)
	}
}

func TestGetCourseRelatedSameTeacherCourses(t *testing.T) {
	conn := setupRelatedTest(t)
	zhang := createInstructor(t, conn, "张三")
	// A 由张三授课；B、C 也由张三授课（B 评价数更高），D 由其他人授课。
	a := createRelatedCourse(t, conn, "100011", []uint64{zhang})
	b := createRelatedCourse(t, conn, "100012", []uint64{zhang})
	c := createRelatedCourse(t, conn, "100013", []uint64{zhang})
	_ = createRelatedCourse(t, conn, "100014", []uint64{createInstructor(t, conn, "王五")})
	if err := conn.Create(&course.CourseStatsEntity{CourseId: b, RatingCount: 2, RatingSum: 9, ReviewCount: 9}).Error; err != nil {
		t.Fatalf("create stats for B: %v", err)
	}
	if err := conn.Create(&course.CourseStatsEntity{CourseId: c, RatingCount: 1, RatingSum: 4, ReviewCount: 3}).Error; err != nil {
		t.Fatalf("create stats for C: %v", err)
	}

	related, err := GetCourseRelated(a)
	if err != nil {
		t.Fatalf("GetCourseRelated err = %v", err)
	}
	if len(related.TeacherOtherCourses) != 2 {
		t.Fatalf("teacherOtherCourses len = %d, want 2: %#v", len(related.TeacherOtherCourses), related.TeacherOtherCourses)
	}
	// 按 review_count 降序：B(9) 在 C(3) 前。
	if related.TeacherOtherCourses[0].Id != b || related.TeacherOtherCourses[0].ReviewCount != 9 {
		t.Fatalf("first item = %#v, want B with reviewCount 9", related.TeacherOtherCourses[0])
	}
	if related.TeacherOtherCourses[1].Id != c || related.TeacherOtherCourses[1].ReviewCount != 3 {
		t.Fatalf("second item = %#v, want C with reviewCount 3", related.TeacherOtherCourses[1])
	}
}

func TestGetCourseRelatedLimitFive(t *testing.T) {
	conn := setupRelatedTest(t)
	zhang := createInstructor(t, conn, "张三")
	a := createRelatedCourse(t, conn, "100020", []uint64{zhang})
	// 6 门同教师课程，review_count 依次为 1..6。
	for i := 1; i <= 6; i++ {
		id := createRelatedCourse(t, conn, "10002"+string(rune('0'+i)), []uint64{zhang})
		if err := conn.Create(&course.CourseStatsEntity{CourseId: id, RatingCount: 1, RatingSum: 4, ReviewCount: i}).Error; err != nil {
			t.Fatalf("create stats: %v", err)
		}
	}
	related, err := GetCourseRelated(a)
	if err != nil {
		t.Fatalf("GetCourseRelated err = %v", err)
	}
	if len(related.TeacherOtherCourses) != RelatedListLimit {
		t.Fatalf("teacherOtherCourses len = %d, want %d", len(related.TeacherOtherCourses), RelatedListLimit)
	}
	// 前 5 条应为 review_count 最高的 5 门（6..2），降序。
	for i, item := range related.TeacherOtherCourses {
		if want := 6 - i; item.ReviewCount != want {
			t.Fatalf("item[%d] reviewCount = %d, want %d", i, item.ReviewCount, want)
		}
	}
}

func TestGetCourseRelatedSameCodeOtherTeachers(t *testing.T) {
	conn := setupRelatedTest(t)
	zhang := createInstructor(t, conn, "张三")
	li := createInstructor(t, conn, "李四")
	// (code, teacher) 复合身份模型：同课号不同教师是独立课程行。
	// A（张三）与 A2（李四）同 code 100030 → 同课程其他教师 = A2 卡片。
	a := createRelatedCourse(t, conn, "100030", []uint64{zhang})
	a2 := course.Entity{PrimaryCode: "100030", TeacherId: li, Name: "课程100030", Department: "数学科学学院", Status: course.StatusVisible}
	if err := conn.Create(&a2).Error; err != nil {
		t.Fatalf("create same-code course: %v", err)
	}
	if err := conn.Create(&course.CourseStatsEntity{CourseId: a2.Id, RatingCount: 2, RatingSum: 9, ReviewCount: 7}).Error; err != nil {
		t.Fatalf("create stats for A2: %v", err)
	}

	related, err := GetCourseRelated(a)
	if err != nil {
		t.Fatalf("GetCourseRelated err = %v", err)
	}
	if len(related.SameCourseOtherTeachers) != 1 {
		t.Fatalf("sameCourseOtherTeachers len = %d, want 1: %#v", len(related.SameCourseOtherTeachers), related.SameCourseOtherTeachers)
	}
	item := related.SameCourseOtherTeachers[0]
	if item.Id != a2.Id {
		t.Fatalf("item id = %d, want A2 (%d)", item.Id, a2.Id)
	}
	if item.TeacherName != "李四" {
		t.Fatalf("item teacherName = %q, want 李四", item.TeacherName)
	}
	if item.ReviewCount != 7 {
		t.Fatalf("item reviewCount = %d, want 7", item.ReviewCount)
	}
}

func TestGetCourseRelatedSameCoursePrimaryOnly(t *testing.T) {
	conn := setupRelatedTest(t)
	zhang := createInstructor(t, conn, "张三")
	// A 课号只有一张卡（无同课号其他教师行）：列表为空。
	a := createRelatedCourse(t, conn, "100040", []uint64{zhang})
	_ = addRelatedOffering(t, conn, a, "2024-2025-2", []uint64{zhang})

	related, err := GetCourseRelated(a)
	if err != nil {
		t.Fatalf("GetCourseRelated err = %v", err)
	}
	if len(related.SameCourseOtherTeachers) != 0 {
		t.Fatalf("sameCourseOtherTeachers len = %d, want 0: %#v", len(related.SameCourseOtherTeachers), related.SameCourseOtherTeachers)
	}
}

func TestGetCourseRelatedNoOfferings(t *testing.T) {
	conn := setupRelatedTest(t)
	// 可见但没有任何开课实例的课程：两个列表均为空。
	c := course.Entity{PrimaryCode: "100070", Name: "无开课", Department: "数学科学学院", Status: course.StatusVisible}
	if err := conn.Create(&c).Error; err != nil {
		t.Fatalf("create course: %v", err)
	}
	related, err := GetCourseRelated(c.Id)
	if err != nil {
		t.Fatalf("GetCourseRelated err = %v", err)
	}
	if len(related.TeacherOtherCourses) != 0 || len(related.SameCourseOtherTeachers) != 0 {
		t.Fatalf("expected empty lists, got %#v", related)
	}
}

func TestGetCourseRelatedOtherTeachersSortAndLimit(t *testing.T) {
	conn := setupRelatedTest(t)
	zhang := createInstructor(t, conn, "张三")
	a := createRelatedCourse(t, conn, "100050", []uint64{zhang})
	// 6 个同课号不同教师卡，review_count 依次 10,8,6,4,2,1。
	type spec struct {
		name        string
		teacherID   uint64
		reviewCount int
	}
	specs := []spec{
		{"李四", createInstructor(t, conn, "李四"), 10},
		{"王五", createInstructor(t, conn, "王五"), 8},
		{"赵六", createInstructor(t, conn, "赵六"), 6},
		{"孙七", createInstructor(t, conn, "孙七"), 4},
		{"周八", createInstructor(t, conn, "周八"), 2},
		{"吴九", createInstructor(t, conn, "吴九"), 1},
	}
	for _, s := range specs {
		other := course.Entity{PrimaryCode: "100050", TeacherId: s.teacherID, Name: "课程100050", Department: "数学科学学院", Status: course.StatusVisible}
		if err := conn.Create(&other).Error; err != nil {
			t.Fatalf("create same-code course: %v", err)
		}
		if err := conn.Create(&course.CourseStatsEntity{CourseId: other.Id, RatingCount: 1, RatingSum: 5, ReviewCount: s.reviewCount}).Error; err != nil {
			t.Fatalf("create course stats: %v", err)
		}
	}
	related, err := GetCourseRelated(a)
	if err != nil {
		t.Fatalf("GetCourseRelated err = %v", err)
	}
	if len(related.SameCourseOtherTeachers) != RelatedListLimit {
		t.Fatalf("sameCourseOtherTeachers len = %d, want %d", len(related.SameCourseOtherTeachers), RelatedListLimit)
	}
	want := []string{"李四", "王五", "赵六", "孙七", "周八"}
	for i, item := range related.SameCourseOtherTeachers {
		if item.TeacherName != want[i] {
			t.Fatalf("item[%d] teacherName = %q, want %q", i, item.TeacherName, want[i])
		}
	}
}

func TestGetCourseRelatedTeacherSortTiebreaks(t *testing.T) {
	conn := setupRelatedTest(t)
	zhang := createInstructor(t, conn, "张三")
	a := createRelatedCourse(t, conn, "100060", []uint64{zhang})
	// 四门同教师课程全部 reviewCount=5，用 ratingAvg 与 id 区分两级 tie-break：
	// 100061 avg 4.5、100062 avg 5.0、100063/100064 avg 4.0（后者 id 更大）。
	specs := []struct {
		code       string
		sum, count int
	}{
		{"100061", 9, 2},
		{"100062", 5, 1},
		{"100063", 4, 1},
		{"100064", 4, 1},
	}
	for _, s := range specs {
		id := createRelatedCourse(t, conn, s.code, []uint64{zhang})
		if err := conn.Create(&course.CourseStatsEntity{CourseId: id, RatingCount: s.count, RatingSum: s.sum, ReviewCount: 5}).Error; err != nil {
			t.Fatalf("create stats: %v", err)
		}
	}
	related, err := GetCourseRelated(a)
	if err != nil {
		t.Fatalf("GetCourseRelated err = %v", err)
	}
	if len(related.TeacherOtherCourses) != 4 {
		t.Fatalf("teacherOtherCourses len = %d, want 4", len(related.TeacherOtherCourses))
	}
	wantAvg := []float64{5.0, 4.5, 4.0, 4.0}
	for i, item := range related.TeacherOtherCourses {
		if item.RatingAvg != wantAvg[i] {
			t.Fatalf("item[%d] ratingAvg = %v, want %v", i, item.RatingAvg, wantAvg[i])
		}
	}
	// 同 ratingAvg（4.0）按 id 降序：后创建的 100064 排在 100063 前。
	if related.TeacherOtherCourses[2].PrimaryCode != "100064" || related.TeacherOtherCourses[3].PrimaryCode != "100063" {
		t.Fatalf("4.0 tie-break order wrong: %s, %s", related.TeacherOtherCourses[2].PrimaryCode, related.TeacherOtherCourses[3].PrimaryCode)
	}
}
