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

func TestGetCourseRelatedOtherTeachersDedup(t *testing.T) {
	conn := setupRelatedTest(t)
	zhang := createInstructor(t, conn, "张三")
	li := createInstructor(t, conn, "李四")
	// A 最近学期（2025-2026-1）由 [张三, 李四] 授课（primary）；
	// 更早学期分别由 [李四]（o2）、[李四]（o3）授课——o3 与 o2 组合相同，应被去重。
	a := createRelatedCourse(t, conn, "100030", []uint64{zhang, li})
	o2 := addRelatedOffering(t, conn, a, "2024-2025-2", []uint64{li})
	_ = addRelatedOffering(t, conn, a, "2023-2024-2", []uint64{li})

	related, err := GetCourseRelated(a)
	if err != nil {
		t.Fatalf("GetCourseRelated err = %v", err)
	}
	if len(related.SameCourseOtherTeachers) != 1 {
		t.Fatalf("sameCourseOtherTeachers len = %d, want 1 (去重后仅 o2): %#v", len(related.SameCourseOtherTeachers), related.SameCourseOtherTeachers)
	}
	item := related.SameCourseOtherTeachers[0]
	if item.OfferingId != o2 {
		t.Fatalf("offering = %d, want o2 (%d)", item.OfferingId, o2)
	}
	if len(item.Instructors) != 1 || item.Instructors[0] != "李四" {
		t.Fatalf("instructors = %#v, want [李四]", item.Instructors)
	}
}

func TestGetCourseRelatedSameCoursePrimaryOnly(t *testing.T) {
	conn := setupRelatedTest(t)
	zhang := createInstructor(t, conn, "张三")
	// A 全部学期都由同一教师组合授课：无"其他教师"，列表为空。
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
