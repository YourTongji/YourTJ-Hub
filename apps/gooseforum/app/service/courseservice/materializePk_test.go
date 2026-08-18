package courseservice

import (
	"context"
	"fmt"
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

// TestMaterializeFromPkInstructorNormalizedNameIdempotent 回归 #199：教师名含空格/中点/全角字符时
// 归一化前后不同（如 "John Smith"→"johnsmith"），此前第二次物化仍按原始姓名查找 normalized_name
// 导致必然 miss、重复创建教师。断言第二次 InstructorsInserted == 0。
func TestMaterializeFromPkInstructorNormalizedNameIdempotent(t *testing.T) {
	migrateMaterializeTables(t)
	conn := db.Connect()
	credit := 3.0
	if err := conn.Create(&pk.CourseDetailEntity{
		Id: 1, CalendarId: 1, CourseCode: "B101", CourseName: "数据科学导论", Code: "B10101",
		NewCourseCode: "B101N", NewCode: "B101N01", Credit: &credit, Faculty: "数学科学学院",
	}).Error; err != nil {
		t.Fatalf("seed course detail: %v", err)
	}
	// 含空格、中点、全角字符的教师名：归一化值分别 johnsmith / 爱丽丝史密斯 / john，与原始姓名不同。
	for i, raw := range []string{"John Smith", "爱丽丝·史密斯", "Ｊｏｈｎ"} {
		teacher := pk.TeacherEntity{
			Id: uint64(200 + i), TeachingClassId: 1, TeacherCode: fmt.Sprintf("T2%02d", i),
			TeacherName: raw,
		}
		if err := conn.Create(&teacher).Error; err != nil {
			t.Fatalf("seed teacher %q: %v", raw, err)
		}
	}
	if err := conn.Create(&pk.FacultyEntity{Faculty: "数学科学学院", FacultyI18n: "数学科学学院"}).Error; err != nil {
		t.Fatalf("seed faculty: %v", err)
	}

	first, err := MaterializeFromPk(context.Background(), []uint64{1})
	if err != nil {
		t.Fatalf("first materialize: %v", err)
	}
	if first.InstructorsInserted != 3 {
		t.Fatalf("first instructorsInserted = %d, want 3", first.InstructorsInserted)
	}

	second, err := MaterializeFromPk(context.Background(), []uint64{1})
	if err != nil {
		t.Fatalf("second materialize: %v", err)
	}
	if second.InstructorsInserted != 0 {
		t.Errorf("second instructorsInserted = %d, want 0（归一化姓名未命中导致重复创建）", second.InstructorsInserted)
	}

	var total int64
	if err := conn.Model(&course.InstructorEntity{}).Count(&total).Error; err != nil {
		t.Fatalf("count instructors: %v", err)
	}
	if total != 3 {
		t.Errorf("instructors after 2 runs = %d, want 3", total)
	}
}

// TestMaterializeFromPkSplitsByIdentityTeacher 回归 review：同一 courseCode 不同教师的
// 教学班必须物化为独立课程卡（(code, teacher) 复合身份），不能按 code 合并成一张卡
// 后只取 TeacherNames[0]（否则漏卡且选中教师依赖查询顺序）。
func TestMaterializeFromPkSplitsByIdentityTeacher(t *testing.T) {
	migrateMaterializeTables(t)
	conn := db.Connect()
	credit := 5.0
	if err := conn.Create(&pk.CourseDetailEntity{
		Id: 1, CalendarId: 1, CourseCode: "A001", CourseName: "高等数学(A)上", Code: "A00101",
		NewCourseCode: "A001N", NewCode: "A001N01", Credit: &credit, Faculty: "数学科学学院",
	}).Error; err != nil {
		t.Fatalf("seed course detail 1: %v", err)
	}
	if err := conn.Create(&pk.CourseDetailEntity{
		Id: 2, CalendarId: 1, CourseCode: "A001", CourseName: "高等数学(A)上", Code: "A00102",
		NewCourseCode: "A001N", NewCode: "A001N02", Credit: &credit, Faculty: "数学科学学院",
	}).Error; err != nil {
		t.Fatalf("seed course detail 2: %v", err)
	}
	if err := conn.Create(&pk.TeacherEntity{Id: 100, TeachingClassId: 1, TeacherCode: "T001", TeacherName: "张三"}).Error; err != nil {
		t.Fatalf("seed teacher 张三: %v", err)
	}
	if err := conn.Create(&pk.TeacherEntity{Id: 101, TeachingClassId: 2, TeacherCode: "T002", TeacherName: "李四"}).Error; err != nil {
		t.Fatalf("seed teacher 李四: %v", err)
	}
	if err := conn.Create(&pk.FacultyEntity{Faculty: "数学科学学院", FacultyI18n: "数学科学学院"}).Error; err != nil {
		t.Fatalf("seed faculty: %v", err)
	}

	report, err := MaterializeFromPk(context.Background(), []uint64{1})
	if err != nil {
		t.Fatalf("materialize: %v", err)
	}
	if report.CoursesInserted != 2 {
		t.Fatalf("coursesInserted = %d, want 2（同 code 双教师应拆成两张卡）", report.CoursesInserted)
	}

	var courses []course.Entity
	if err := conn.Where("primary_code = ?", "A001").Find(&courses).Error; err != nil {
		t.Fatalf("find courses: %v", err)
	}
	if len(courses) != 2 {
		t.Fatalf("courses = %d, want 2", len(courses))
	}
	distinct := map[uint64]bool{}
	for _, c := range courses {
		if c.TeacherId != 0 {
			distinct[c.TeacherId] = true
		}
	}
	if len(distinct) != 2 {
		t.Fatalf("cards must map to 2 distinct identity teachers, got %v", distinct)
	}
	var instructors []course.InstructorEntity
	if err := conn.Find(&instructors).Error; err != nil {
		t.Fatalf("find instructors: %v", err)
	}
	for _, ins := range instructors {
		if !distinct[ins.Id] {
			t.Fatalf("instructor %d not referenced by any card", ins.Id)
		}
	}
}
