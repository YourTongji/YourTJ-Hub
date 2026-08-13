package pkservice

import (
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pk"
	"gorm.io/gorm"
)

// seedDictJoinFixture 种入明细行与字典行：campus/faculty/language 字典行的 calendar_id 与
// 查询学期(99999)不同，模拟「非最后同步学期」的全局字典数据（同步管线按单键 last-write-wins，
// 同 code 只保留一行）。课程性质 pk_course_nature 按学期隔离，故按查询学期 99999 种入。
func seedDictJoinFixture(t *testing.T, conn *gorm.DB) {
	t.Helper()
	models := []any{
		&pk.CourseDetailEntity{Id: 910001, Code: "TJCS20101", CourseCode: "TJCS201", CourseName: "数据结构与算法", CourseLabelId: 1, Credit: 4, Campus: "JD", Faculty: "CS", TeachingLanguage: "ZH", CalendarId: 99999},
		// 字典行 calendar_id 与查询学期不同，但 code 命中 → 明细查询仍须解析出 i18n。
		&pk.CampusEntity{Campus: "JD", CampusI18n: "嘉定校区", CalendarId: 88888},
		&pk.FacultyEntity{Faculty: "CS", FacultyI18n: "计算机科学与技术系", CalendarId: 88888},
		&pk.LanguageEntity{TeachingLanguage: "ZH", TeachingLanguageI18n: "中文", CalendarId: 88888},
		// 课程性质按学期隔离：只有与查询学期一致的 calendar_id 才会命中。
		&pk.CourseNatureEntity{CalendarId: 99999, CourseLabelId: 1, CourseLabelName: "专业必修"},
	}
	for _, m := range models {
		if err := conn.Create(m).Error; err != nil {
			t.Fatalf("create %T: %v", m, err)
		}
	}
}

// TestCourseDetailDictJoinIgnoresCalendarId P8：campus/language 字典行 calendar_id 与查询学期
// 不同时，campus_i18n / teaching_language_i18n 仍能解析（回归：旧 JOIN 附加 calendar_id
// 等值条件导致非「最后同步学期」的查询 i18n 变空串）。
func TestCourseDetailDictJoinIgnoresCalendarId(t *testing.T) {
	conn := setupPkServiceTest(t)
	seedDictJoinFixture(t, conn)

	out, err := FindCourseDetailsByCodes(99999, []string{"TJCS201"})
	if err != nil {
		t.Fatalf("FindCourseDetailsByCodes: %v", err)
	}
	items := out["TJCS201"]
	if len(items) != 1 {
		t.Fatalf("course items = %d, want 1: %+v", len(items), items)
	}
	if items[0].Campus != "嘉定校区" {
		t.Fatalf("campus = %q, want 嘉定校区", items[0].Campus)
	}
	if items[0].TeachingLanguage != "中文" {
		t.Fatalf("teachingLanguage = %q, want 中文", items[0].TeachingLanguage)
	}
}

// TestSearchCourseDictJoinIgnoresCalendarId P9：faculty/campus 字典行 calendar_id 与查询学期
// 不同时仍能解析 i18n；同时确认课程性质仍按学期隔离（跨学期字典不串、性质不过滤掉）。
func TestSearchCourseDictJoinIgnoresCalendarId(t *testing.T) {
	conn := setupPkServiceTest(t)
	seedDictJoinFixture(t, conn)

	res, err := SearchCourses(SearchCourseParams{CalendarId: 99999})
	if err != nil {
		t.Fatalf("SearchCourses: %v", err)
	}
	if len(res.Courses) != 1 || res.Courses[0].CourseCode != "TJCS201" {
		t.Fatalf("courses = %+v, want [TJCS201]", res.Courses)
	}
	c := res.Courses[0]
	if c.Faculty != "计算机科学与技术系" || c.FacultyI18n != "计算机科学与技术系" {
		t.Fatalf("faculty i18n = %q/%q, want 计算机科学与技术系", c.Faculty, c.FacultyI18n)
	}
	if len(c.Campus) != 1 || c.Campus[0] != "嘉定校区" {
		t.Fatalf("campus = %v, want [嘉定校区]", c.Campus)
	}
	if len(c.CourseNature) != 1 || c.CourseNature[0] != "专业必修" {
		t.Fatalf("courseNature = %v, want [专业必修]", c.CourseNature)
	}
}
