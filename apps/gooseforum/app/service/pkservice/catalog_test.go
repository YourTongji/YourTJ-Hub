package pkservice

import (
	"testing"
	"time"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pk"
)

func TestCatalogCachingAndInvalidation(t *testing.T) {
	migratePkTables(t)

	// 1. Calendars caching & invalidation
	start := time.Date(2025, 9, 8, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 1, 18, 0, 0, 0, 0, time.UTC)
	conn := db.Connect()
	if err := conn.Create(&[]pk.CalendarEntity{
		{CalendarId: 101, CalendarIdI18n: "2025-2026-1", StartDate: &start, EndDate: &end},
	}).Error; err != nil {
		t.Fatalf("seed calendar: %v", err)
	}

	cals, err := ListCalendars()
	if err != nil {
		t.Fatalf("ListCalendars: %v", err)
	}
	if len(cals) != 1 {
		t.Fatalf("expected 1 calendar, got %d", len(cals))
	}

	// Defensive copy verification: mutating returned slice or date pointers does not taint cache
	cals[0].CalendarName = "mutated"
	*cals[0].StartDate = "2099-01-01"
	*cals[0].EndDate = "2099-02-01"
	cals2, err := ListCalendars()
	if err != nil {
		t.Fatalf("ListCalendars: %v", err)
	}
	if cals2[0].CalendarName != "2025-2026-1" {
		t.Fatalf("cached calendar was mutated in place: %q", cals2[0].CalendarName)
	}
	if cals2[0].StartDate == nil || *cals2[0].StartDate != "2025-09-08" {
		t.Fatalf("cached calendar StartDate was mutated in place: %v", cals2[0].StartDate)
	}
	if cals2[0].EndDate == nil || *cals2[0].EndDate != "2026-01-18" {
		t.Fatalf("cached calendar EndDate was mutated in place: %v", cals2[0].EndDate)
	}

	// Insert without invalidating -> cache hit should return old count
	if err := conn.Create(&pk.CalendarEntity{
		CalendarId: 102, CalendarIdI18n: "2025-2026-2",
	}).Error; err != nil {
		t.Fatalf("seed calendar 2: %v", err)
	}
	calsCached, err := ListCalendars()
	if err != nil {
		t.Fatalf("ListCalendars: %v", err)
	}
	if len(calsCached) != 1 {
		t.Fatalf("expected cached 1 calendar, got %d", len(calsCached))
	}

	// Invalidate -> fresh read from DB
	InvalidateCatalogCache()
	calsFresh, err := ListCalendars()
	if err != nil {
		t.Fatalf("ListCalendars: %v", err)
	}
	if len(calsFresh) != 2 {
		t.Fatalf("expected 2 calendars after invalidation, got %d", len(calsFresh))
	}

	// 2. Campuses caching & invalidation
	if err := conn.Create(&pk.CampusEntity{
		Campus: "Siping", CampusI18n: "四平路校区",
	}).Error; err != nil {
		t.Fatalf("seed campus: %v", err)
	}
	campuses, err := ListCampuses()
	if err != nil {
		t.Fatalf("ListCampuses: %v", err)
	}
	if len(campuses) != 1 {
		t.Fatalf("expected 1 campus, got %d", len(campuses))
	}

	campuses[0].CampusName = "mutated"
	campuses2, err := ListCampuses()
	if err != nil {
		t.Fatalf("ListCampuses: %v", err)
	}
	if campuses2[0].CampusName != "四平路校区" {
		t.Fatalf("cached campus was mutated in place: %q", campuses2[0].CampusName)
	}

	if err := conn.Create(&pk.CampusEntity{
		Campus: "Jiading", CampusI18n: "嘉定校区",
	}).Error; err != nil {
		t.Fatalf("seed campus 2: %v", err)
	}
	campusesCached, err := ListCampuses()
	if err != nil {
		t.Fatalf("ListCampuses: %v", err)
	}
	if len(campusesCached) != 1 {
		t.Fatalf("expected cached 1 campus, got %d", len(campusesCached))
	}

	InvalidateCatalogCache()
	campusesFresh, err := ListCampuses()
	if err != nil {
		t.Fatalf("ListCampuses: %v", err)
	}
	if len(campusesFresh) != 2 {
		t.Fatalf("expected 2 campuses after invalidation, got %d", len(campusesFresh))
	}

	// 3. Faculties caching & invalidation
	if err := conn.Create(&pk.FacultyEntity{
		Faculty: "CS", FacultyI18n: "计算机科学与技术学院",
	}).Error; err != nil {
		t.Fatalf("seed faculty: %v", err)
	}
	faculties, err := ListFaculties()
	if err != nil {
		t.Fatalf("ListFaculties: %v", err)
	}
	if len(faculties) != 1 {
		t.Fatalf("expected 1 faculty, got %d", len(faculties))
	}

	faculties[0].FacultyName = "mutated"
	faculties2, err := ListFaculties()
	if err != nil {
		t.Fatalf("ListFaculties: %v", err)
	}
	if faculties2[0].FacultyName != "计算机科学与技术学院" {
		t.Fatalf("cached faculty was mutated in place: %q", faculties2[0].FacultyName)
	}

	if err := conn.Create(&pk.FacultyEntity{
		Faculty: "SE", FacultyI18n: "软件学院",
	}).Error; err != nil {
		t.Fatalf("seed faculty 2: %v", err)
	}
	facultiesCached, err := ListFaculties()
	if err != nil {
		t.Fatalf("ListFaculties: %v", err)
	}
	if len(facultiesCached) != 1 {
		t.Fatalf("expected cached 1 faculty, got %d", len(facultiesCached))
	}

	InvalidateCatalogCache()
	facultiesFresh, err := ListFaculties()
	if err != nil {
		t.Fatalf("ListFaculties: %v", err)
	}
	if len(facultiesFresh) != 2 {
		t.Fatalf("expected 2 faculties after invalidation, got %d", len(facultiesFresh))
	}

	// 4. CoursesByMajor caching & invalidation
	if err := conn.Create(&pk.MajorEntity{
		Id: 9000, Code: "03074", Grade: ptr(2025), Name: "2025(03074 测试专业)", CalendarId: 99999,
	}).Error; err != nil {
		t.Fatalf("seed major: %v", err)
	}
	if err := conn.Create(&pk.CourseDetailEntity{
		Id: 900001, Code: "CS10101", CourseCode: "CS101", CourseName: "计算机基础",
		CalendarId: 99999, Campus: "Siping", Faculty: "CS",
	}).Error; err != nil {
		t.Fatalf("seed course detail: %v", err)
	}
	if err := conn.Create(&pk.MajorCourseEntity{
		MajorId: 9000, CourseId: 900001,
	}).Error; err != nil {
		t.Fatalf("seed major course: %v", err)
	}
	day := 1
	room := "四平路校区 A101"
	tname := "张老师(T001)"
	if err := conn.Create(&pk.TeacherEntity{
		Id: 1, TeachingClassId: 900001, TeacherCode: "T001", TeacherName: "张老师",
		ArrangeInfoText: "张老师(T001) 星期一1-2节[1-16周] 四平路校区 A101",
	}).Error; err != nil {
		t.Fatalf("seed teacher: %v", err)
	}

	courses, err := FindCoursesByMajor(2025, "03074", 99999)
	if err != nil {
		t.Fatalf("FindCoursesByMajor: %v", err)
	}
	if len(courses) != 1 || len(courses[0].Courses) != 1 {
		t.Fatalf("expected 1 course with 1 class, got %+v", courses)
	}

	// Defensive copy verification: mutate returned structs, slices, and pointers
	courses[0].CourseName = "mutated"
	courses[0].CourseNature = append(courses[0].CourseNature, "mutated")
	courses[0].Courses[0].Code = "mutated"
	if len(courses[0].Courses[0].ArrangementInfo) > 0 {
		*courses[0].Courses[0].ArrangementInfo[0].OccupyDay = 99
		*courses[0].Courses[0].ArrangementInfo[0].OccupyRoom = "mutated"
		*courses[0].Courses[0].ArrangementInfo[0].TeacherAndCode = "mutated"
		courses[0].Courses[0].ArrangementInfo[0].OccupyTime = append(courses[0].Courses[0].ArrangementInfo[0].OccupyTime, 999)
	}

	courses2, err := FindCoursesByMajor(2025, "03074", 99999)
	if err != nil {
		t.Fatalf("FindCoursesByMajor cached: %v", err)
	}
	if courses2[0].CourseName != "计算机基础" {
		t.Fatalf("cached course CourseName mutated: %q", courses2[0].CourseName)
	}
	if courses2[0].Courses[0].Code != "CS10101" {
		t.Fatalf("cached class Code mutated: %q", courses2[0].Courses[0].Code)
	}
	if len(courses2[0].Courses[0].ArrangementInfo) > 0 {
		if *courses2[0].Courses[0].ArrangementInfo[0].OccupyDay != day {
			t.Fatalf("cached OccupyDay mutated: %d", *courses2[0].Courses[0].ArrangementInfo[0].OccupyDay)
		}
		if *courses2[0].Courses[0].ArrangementInfo[0].OccupyRoom != room {
			t.Fatalf("cached OccupyRoom mutated: %s", *courses2[0].Courses[0].ArrangementInfo[0].OccupyRoom)
		}
		if *courses2[0].Courses[0].ArrangementInfo[0].TeacherAndCode != tname {
			t.Fatalf("cached TeacherAndCode mutated: %s", *courses2[0].Courses[0].ArrangementInfo[0].TeacherAndCode)
		}
	}

	// Insert second course detail without invalidating -> cache hit should return old 1 course
	if err := conn.Create(&pk.CourseDetailEntity{
		Id: 900002, Code: "CS10201", CourseCode: "CS102", CourseName: "算法分析",
		CalendarId: 99999, Campus: "Siping", Faculty: "CS",
	}).Error; err != nil {
		t.Fatalf("seed course detail 2: %v", err)
	}
	if err := conn.Create(&pk.MajorCourseEntity{
		MajorId: 9000, CourseId: 900002,
	}).Error; err != nil {
		t.Fatalf("seed major course 2: %v", err)
	}

	coursesCached, err := FindCoursesByMajor(2025, "03074", 99999)
	if err != nil {
		t.Fatalf("FindCoursesByMajor cached: %v", err)
	}
	if len(coursesCached) != 1 {
		t.Fatalf("expected cached 1 course, got %d", len(coursesCached))
	}

	// Invalidate -> fresh read from DB returns 2 courses
	InvalidateCatalogCache()
	coursesFresh, err := FindCoursesByMajor(2025, "03074", 99999)
	if err != nil {
		t.Fatalf("FindCoursesByMajor fresh: %v", err)
	}
	if len(coursesFresh) != 2 {
		t.Fatalf("expected 2 courses after invalidation, got %d", len(coursesFresh))
	}
}

func TestSyncWriteAndDeleteInvalidatesCatalogCache(t *testing.T) {
	migratePkTables(t)

	// 1. Prime caches while DB is empty
	cals0, err := ListCalendars()
	if err != nil {
		t.Fatalf("ListCalendars: %v", err)
	}
	if len(cals0) != 0 {
		t.Fatalf("expected 0 calendars, got %d", len(cals0))
	}

	campuses0, err := ListCampuses()
	if err != nil {
		t.Fatalf("ListCampuses: %v", err)
	}
	if len(campuses0) != 0 {
		t.Fatalf("expected 0 campuses, got %d", len(campuses0))
	}

	faculties0, err := ListFaculties()
	if err != nil {
		t.Fatalf("ListFaculties: %v", err)
	}
	if len(faculties0) != 0 {
		t.Fatalf("expected 0 faculties, got %d", len(faculties0))
	}

	// 2. writeBatchTx without manual InvalidateCatalogCache(): must invalidate caches
	const calendarID = uint64(121)
	if _, err := writeBatchTx(calendarID, []CourseRaw{richCourse()}); err != nil {
		t.Fatalf("writeBatchTx: %v", err)
	}

	calsAfterWrite, err := ListCalendars()
	if err != nil {
		t.Fatalf("ListCalendars after write: %v", err)
	}
	if len(calsAfterWrite) != 1 || calsAfterWrite[0].CalendarName != "2025-2026-1" {
		t.Fatalf("expected writeBatchTx to invalidate and return 1 calendar, got %+v", calsAfterWrite)
	}

	campusesAfterWrite, err := ListCampuses()
	if err != nil {
		t.Fatalf("ListCampuses after write: %v", err)
	}
	if len(campusesAfterWrite) != 1 || campusesAfterWrite[0].CampusName != "四平路校区" {
		t.Fatalf("expected writeBatchTx to invalidate and return 1 campus, got %+v", campusesAfterWrite)
	}

	facultiesAfterWrite, err := ListFaculties()
	if err != nil {
		t.Fatalf("ListFaculties after write: %v", err)
	}
	if len(facultiesAfterWrite) != 1 || facultiesAfterWrite[0].FacultyName != "数学科学学院" {
		t.Fatalf("expected writeBatchTx to invalidate and return 1 faculty, got %+v", facultiesAfterWrite)
	}

	// Prime coursesByMajorCache with data from writeBatchTx
	coursesAfterWrite, err := FindCoursesByMajor(2025, "03074", int(calendarID))
	if err != nil {
		t.Fatalf("FindCoursesByMajor after write: %v", err)
	}
	if len(coursesAfterWrite) != 1 {
		t.Fatalf("expected 1 course after write, got %d", len(coursesAfterWrite))
	}

	// 3. deleteCalendarData without manual InvalidateCatalogCache(): must invalidate caches
	if err := deleteCalendarData(nil, calendarID); err != nil {
		t.Fatalf("deleteCalendarData: %v", err)
	}

	calsAfterDelete, err := ListCalendars()
	if err != nil {
		t.Fatalf("ListCalendars after delete: %v", err)
	}
	if len(calsAfterDelete) != 0 {
		t.Fatalf("expected deleteCalendarData to invalidate and return 0 calendars, got %d", len(calsAfterDelete))
	}

	coursesAfterDelete, err := FindCoursesByMajor(2025, "03074", int(calendarID))
	if err != nil {
		t.Fatalf("FindCoursesByMajor after delete: %v", err)
	}
	if len(coursesAfterDelete) != 0 {
		t.Fatalf("expected deleteCalendarData to invalidate and return 0 courses, got %d", len(coursesAfterDelete))
	}
}

