package courseservice

import (
	"context"
	"testing"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/course"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pk"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
)

func migrateMaterializeTables(t *testing.T) {
	t.Helper()
	models := []any{
		&course.Entity{},
		&course.AliasEntity{},
		&course.InstructorEntity{},
		&taskQueue.Entity{},
		&pk.CourseDetailEntity{},
		&pk.TeacherEntity{},
		&pk.FacultyEntity{},
		&pk.CalendarEntity{},
		&pk.LanguageEntity{},
		&pk.CourseNatureEntity{},
		&pk.CourseNatureByCalendarEntity{},
		&pk.AssessmentEntity{},
		&pk.CampusEntity{},
		&pk.MajorEntity{},
		&pk.MajorCourseEntity{},
		&pk.TeacherTimeslotEntity{},
		&pk.FetchLogEntity{},
	}
	conn := db.Connect()
	if err := conn.AutoMigrate(models...); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	for _, m := range models {
		if err := conn.Unscoped().Where("1 = 1").Delete(m).Error; err != nil {
			t.Fatalf("clean: %v", err)
		}
	}
}

func seedPkForMaterialize(t *testing.T) {
	t.Helper()
	conn := db.Connect()
	credit := 5.0
	if err := conn.Create(&pk.CourseDetailEntity{
		Id: 1, CalendarId: 1, CourseCode: "A001", CourseName: "高等数学(A)上", Code: "A00101",
		NewCourseCode: "A001N", NewCode: "A001N01", Credit: &credit, Faculty: "数学科学学院",
	}).Error; err != nil {
		t.Fatalf("seed course detail: %v", err)
	}
	if err := conn.Create(&pk.TeacherEntity{Id: 100, TeachingClassId: 1, TeacherCode: "T001", TeacherName: "张三"}).Error; err != nil {
		t.Fatalf("seed teacher: %v", err)
	}
	if err := conn.Create(&pk.FacultyEntity{Faculty: "数学科学学院", FacultyI18n: "数学科学学院"}).Error; err != nil {
		t.Fatalf("seed faculty: %v", err)
	}
}

func TestMaterializeFromPkCreatesCatalog(t *testing.T) {
	migrateMaterializeTables(t)
	seedPkForMaterialize(t)

	report, err := MaterializeFromPk(context.Background(), []uint64{1})
	if err != nil {
		t.Fatalf("materialize: %v", err)
	}
	if report.CoursesInserted != 1 {
		t.Errorf("coursesInserted = %d, want 1", report.CoursesInserted)
	}
	if report.InstructorsInserted != 1 {
		t.Errorf("instructorsInserted = %d, want 1", report.InstructorsInserted)
	}

	conn := db.Connect()
	var courses []course.Entity
	if err := conn.Where("primary_code = ?", "A001").Find(&courses).Error; err != nil {
		t.Fatalf("find course: %v", err)
	}
	if len(courses) != 1 {
		t.Fatalf("courses = %d, want 1", len(courses))
	}
	if courses[0].CreditX10 != 50 {
		t.Errorf("credit_x10 = %d, want 50", courses[0].CreditX10)
	}
	if courses[0].Department != "数学科学学院" {
		t.Errorf("department = %q", courses[0].Department)
	}

	var aliases []course.AliasEntity
	if err := conn.Where("source = ?", materializePkSource).Find(&aliases).Error; err != nil {
		t.Fatalf("find aliases: %v", err)
	}
	// 期望 courseCode/code/newCourseCode/newCode 四条 code 别名。
	if len(aliases) != 4 {
		t.Errorf("aliases = %d, want 4: %+v", len(aliases), aliases)
	}
	var instructors []course.InstructorEntity
	if err := conn.Where("name = ?", "张三").Find(&instructors).Error; err != nil {
		t.Fatalf("find instructor: %v", err)
	}
	if len(instructors) != 1 {
		t.Errorf("instructors = %d, want 1", len(instructors))
	}
}

func TestMaterializeFromPkIdempotent(t *testing.T) {
	migrateMaterializeTables(t)
	seedPkForMaterialize(t)

	first, err := MaterializeFromPk(context.Background(), []uint64{1})
	if err != nil {
		t.Fatalf("first materialize: %v", err)
	}
	second, err := MaterializeFromPk(context.Background(), []uint64{1})
	if err != nil {
		t.Fatalf("second materialize: %v", err)
	}
	if second.CoursesInserted != 0 {
		t.Errorf("second coursesInserted = %d, want 0", second.CoursesInserted)
	}
	if second.InstructorsInserted != 0 {
		t.Errorf("second instructorsInserted = %d, want 0", second.InstructorsInserted)
	}
	conn := db.Connect()
	var courseCount int64
	if err := conn.Model(&course.Entity{}).Where("primary_code = ?", "A001").Count(&courseCount).Error; err != nil {
		t.Fatalf("count courses: %v", err)
	}
	if courseCount != 1 {
		t.Errorf("courses after 2nd run = %d, want 1", courseCount)
	}
	_ = first
}
