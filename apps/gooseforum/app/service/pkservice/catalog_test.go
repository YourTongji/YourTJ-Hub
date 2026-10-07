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
}

