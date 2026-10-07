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

	// Defensive copy verification: mutating returned slice does not taint cache
	cals[0].CalendarName = "mutated"
	cals2, err := ListCalendars()
	if err != nil {
		t.Fatalf("ListCalendars: %v", err)
	}
	if cals2[0].CalendarName != "2025-2026-1" {
		t.Fatalf("cached calendar was mutated in place: %q", cals2[0].CalendarName)
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
