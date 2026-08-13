package pkservice

import (
	"testing"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pk"
)

// richCourse 覆盖全部 lookup 维度的一系统教学班行。
func richCourse() CourseRaw {
	id := uint64(1)
	tid := uint64(100)
	labelID := uint64(9001)
	credit := 3.5
	period := 2.0
	return CourseRaw{
		Id:                   &id,
		Code:                 "A00101",
		Name:                 "高等数学(A)上",
		CourseCode:           "A001",
		CourseName:           "高等数学(A)上",
		Credits:              &credit,
		Period:               &period,
		CourseLabelId:        &labelID,
		CourseLabelName:      "通识必修",
		TeachingLanguage:     "zh",
		TeachingLanguageI18n: "中文",
		AssessmentMode:       "exam",
		AssessmentModeI18n:   "考试",
		Campus:               "campus1",
		CampusI18n:           "四平路校区",
		Faculty:              "fac1",
		FacultyI18n:          "数学科学学院",
		MajorList:            []string{"2025(03074 土木工程(国际班))"},
		NewCourseCode:        "A001N",
		ArrangeInfo:          "张三(T001) 星期一1-2节[1-16周]",
		CalendarIdI18n:       "2025-2026-1",
		TeacherList: []TeacherRaw{{
			Id: &tid, TeacherCode: "T001", TeacherName: "张三",
		}},
	}
}

func countWhere(t *testing.T, model any, query string, args ...any) int64 {
	t.Helper()
	var n int64
	if err := db.Connect().Model(model).Where(query, args...).Count(&n).Error; err != nil {
		t.Fatalf("count %s: %v", query, err)
	}
	return n
}

func TestWriteBatchTxUpsertsAllDimensions(t *testing.T) {
	migratePkTables(t)
	n, err := writeBatchTx(121, []CourseRaw{richCourse()})
	if err != nil {
		t.Fatalf("writeBatchTx: %v", err)
	}
	if n != 1 {
		t.Errorf("processed = %d, want 1", n)
	}

	if got := countWhere(t, &pk.CalendarEntity{}, "calendar_id = ?", 121); got != 1 {
		t.Errorf("calendar = %d, want 1", got)
	}
	if got := countWhere(t, &pk.LanguageEntity{}, "teaching_language = ?", "zh"); got != 1 {
		t.Errorf("language = %d, want 1", got)
	}
	if got := countWhere(t, &pk.CourseNatureEntity{}, "course_label_id = ?", 9001); got != 1 {
		t.Errorf("course_nature = %d, want 1", got)
	}
	if got := countWhere(t, &pk.CourseNatureByCalendarEntity{}, "course_label_id = ? AND calendar_id = ?", 9001, 121); got != 1 {
		t.Errorf("course_nature_by_calendar = %d, want 1", got)
	}
	if got := countWhere(t, &pk.AssessmentEntity{}, "assessment_mode = ?", "exam"); got != 1 {
		t.Errorf("assessment = %d, want 1", got)
	}
	if got := countWhere(t, &pk.CampusEntity{}, "campus = ?", "campus1"); got != 1 {
		t.Errorf("campus = %d, want 1", got)
	}
	if got := countWhere(t, &pk.FacultyEntity{}, "faculty = ?", "fac1"); got != 1 {
		t.Errorf("faculty = %d, want 1", got)
	}
	if got := countWhere(t, &pk.MajorEntity{}, "name = ?", "2025(03074 土木工程(国际班))"); got != 1 {
		t.Errorf("major = %d, want 1", got)
	}
	if got := countWhere(t, &pk.MajorCourseEntity{}, "course_id = ?", 1); got != 1 {
		t.Errorf("major_course = %d, want 1", got)
	}
	if got := countWhere(t, &pk.CourseDetailEntity{}, "id = ? AND calendar_id = ?", 1, 121); got != 1 {
		t.Errorf("course_detail = %d, want 1", got)
	}
	if got := countWhere(t, &pk.TeacherEntity{}, "id = ?", 100); got != 1 {
		t.Errorf("teacher = %d, want 1", got)
	}
}

func TestWriteBatchTxSkipsRowsWithoutIDs(t *testing.T) {
	migratePkTables(t)
	courses := []CourseRaw{{
		Code:        "X001",
		CourseName:  "无 id 的课程",
		TeacherList: []TeacherRaw{{TeacherCode: "T001", TeacherName: "无 id 的教师"}},
	}}
	n, err := writeBatchTx(121, courses)
	if err != nil {
		t.Fatalf("writeBatchTx: %v", err)
	}
	if n != 0 {
		t.Errorf("processed = %d, want 0 (id-less rows skipped)", n)
	}
	if got := countWhere(t, &pk.CourseDetailEntity{}, "1 = 1"); got != 0 {
		t.Errorf("course_detail rows = %d, want 0", got)
	}
	if got := countWhere(t, &pk.TeacherEntity{}, "1 = 1"); got != 0 {
		t.Errorf("teacher rows = %d, want 0", got)
	}
}
