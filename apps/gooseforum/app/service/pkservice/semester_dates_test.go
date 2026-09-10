package pkservice

import (
	"testing"
	"time"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pk"
)

// withSemesterDates 测试注入学期日期表，结束后恢复。
func withSemesterDates(t *testing.T, dates map[string]semesterDateRange) {
	t.Helper()
	original := loadSemesterDates
	loadSemesterDates = func() map[string]semesterDateRange { return dates }
	t.Cleanup(func() { loadSemesterDates = original })
}

func TestParseSemesterDatesValidConfig(t *testing.T) {
	raw := map[string]any{
		"2025-2026-1": map[string]any{"start": "2025-09-08", "end": "2026-01-18"},
		"2025-2026-2": map[string]string{"start": "2026-03-02", "end": "2026-07-11"},
	}
	got := parseSemesterDates(raw)
	if len(got) != 2 {
		t.Fatalf("parsed = %d entries, want 2", len(got))
	}
	first := got["2025-2026-1"]
	if first.Start == nil || first.End == nil {
		t.Fatalf("2025-2026-1 dates = %+v, want both set", first)
	}
	if first.Start.Format(semesterDateLayout) != "2025-09-08" {
		t.Errorf("start = %s, want 2025-09-08", first.Start.Format(semesterDateLayout))
	}
	if first.End.Format(semesterDateLayout) != "2026-01-18" {
		t.Errorf("end = %s, want 2026-01-18", first.End.Format(semesterDateLayout))
	}
	second := got["2025-2026-2"]
	if second.Start == nil || second.Start.Format(semesterDateLayout) != "2026-03-02" {
		t.Errorf("2025-2026-2 start = %+v, want 2026-03-02", second.Start)
	}
}

func TestParseSemesterDatesSkipsInvalid(t *testing.T) {
	raw := map[string]any{
		"bad-date":   map[string]any{"start": "not-a-date", "end": ""},
		"half-valid": map[string]any{"start": "2025-09-08", "end": "oops"},
		"wrong-type": "2025-09-08",
		"":           map[string]any{"start": "2025-09-08"},
	}
	got := parseSemesterDates(raw)
	// half-valid 保留合法端点 start，非法端点置 nil。
	if _, ok := got["bad-date"]; ok {
		t.Errorf("bad-date should be skipped entirely")
	}
	if _, ok := got["wrong-type"]; ok {
		t.Errorf("wrong-type should be skipped")
	}
	if _, ok := got[""]; ok {
		t.Errorf("empty key should be skipped")
	}
	half, ok := got["half-valid"]
	if !ok || half.Start == nil || half.End != nil {
		t.Errorf("half-valid = %+v (ok=%v), want start set and end nil", half, ok)
	}
}

func TestParseSemesterDatesNonMapRaw(t *testing.T) {
	for _, raw := range []any{nil, "string", 42, []any{"2025-2026-1"}} {
		if got := parseSemesterDates(raw); len(got) != 0 {
			t.Errorf("parseSemesterDates(%v) = %d entries, want 0", raw, len(got))
		}
	}
}

func TestParseSemesterDateCases(t *testing.T) {
	if got := parseSemesterDate("  "); got != nil {
		t.Errorf("blank date = %v, want nil", got)
	}
	if got := parseSemesterDate("2025/09/08"); got != nil {
		t.Errorf("wrong layout = %v, want nil", got)
	}
	got := parseSemesterDate(" 2025-09-08 ")
	if got == nil || got.Format(semesterDateLayout) != "2025-09-08" {
		t.Errorf("trimmed date = %v, want 2025-09-08", got)
	}
}

// TestWriteBatchTxFillsSemesterDates config 命中 calendar_id_i18n 时同步写入起止日期。
func TestWriteBatchTxFillsSemesterDates(t *testing.T) {
	migratePkTables(t)
	start := time.Date(2025, 9, 8, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 1, 18, 0, 0, 0, 0, time.UTC)
	withSemesterDates(t, map[string]semesterDateRange{
		"2025-2026-1": {Start: &start, End: &end},
	})

	if _, err := writeBatchTx(121, []CourseRaw{richCourse()}); err != nil {
		t.Fatalf("writeBatchTx: %v", err)
	}
	var calendar pk.CalendarEntity
	if err := db.Connect().Where("calendar_id = ?", 121).First(&calendar).Error; err != nil {
		t.Fatalf("read calendar: %v", err)
	}
	if calendar.StartDate == nil || calendar.StartDate.Format(semesterDateLayout) != "2025-09-08" {
		t.Errorf("start_date = %+v, want 2025-09-08", calendar.StartDate)
	}
	if calendar.EndDate == nil || calendar.EndDate.Format(semesterDateLayout) != "2026-01-18" {
		t.Errorf("end_date = %+v, want 2026-01-18", calendar.EndDate)
	}
}

// TestWriteBatchTxLeavesDatesNullWithoutConfig 未配置时起止日期保持 NULL（向后兼容）。
func TestWriteBatchTxLeavesDatesNullWithoutConfig(t *testing.T) {
	migratePkTables(t)
	withSemesterDates(t, nil)

	if _, err := writeBatchTx(121, []CourseRaw{richCourse()}); err != nil {
		t.Fatalf("writeBatchTx: %v", err)
	}
	var calendar pk.CalendarEntity
	if err := db.Connect().Where("calendar_id = ?", 121).First(&calendar).Error; err != nil {
		t.Fatalf("read calendar: %v", err)
	}
	if calendar.StartDate != nil || calendar.EndDate != nil {
		t.Errorf("dates = (%+v, %+v), want both nil", calendar.StartDate, calendar.EndDate)
	}
}

// TestListCalendarsOutputsSemesterDates P1 输出纯日期字符串，未配置输出 null。
func TestListCalendarsOutputsSemesterDates(t *testing.T) {
	migratePkTables(t)
	start := time.Date(2025, 9, 8, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 1, 18, 0, 0, 0, 0, time.UTC)
	if err := db.Connect().Create(&[]pk.CalendarEntity{
		{CalendarId: 121, CalendarIdI18n: "2025-2026-1", StartDate: &start, EndDate: &end},
		{CalendarId: 120, CalendarIdI18n: "2024-2025-2"},
	}).Error; err != nil {
		t.Fatalf("seed calendars: %v", err)
	}

	items, err := ListCalendars()
	if err != nil {
		t.Fatalf("ListCalendars: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("items = %d, want 2", len(items))
	}
	// calendarId 倒序：121 在前。
	first := items[0]
	if first.CalendarId != 121 {
		t.Fatalf("first calendarId = %d, want 121", first.CalendarId)
	}
	if first.StartDate == nil || *first.StartDate != "2025-09-08" {
		t.Errorf("startDate = %v, want \"2025-09-08\"", first.StartDate)
	}
	if first.EndDate == nil || *first.EndDate != "2026-01-18" {
		t.Errorf("endDate = %v, want \"2026-01-18\"", first.EndDate)
	}
	second := items[1]
	if second.StartDate != nil || second.EndDate != nil {
		t.Errorf("unconfigured dates = (%v, %v), want both nil", second.StartDate, second.EndDate)
	}
}

// TestFormatPkDate 验证日期格式化边界。
func TestFormatPkDate(t *testing.T) {
	if got := formatPkDate(nil); got != nil {
		t.Errorf("formatPkDate(nil) = %v, want nil", got)
	}
	in := time.Date(2026, 3, 2, 15, 30, 0, 0, time.UTC)
	got := formatPkDate(&in)
	if got == nil || *got != "2026-03-02" {
		t.Errorf("formatPkDate = %v, want \"2026-03-02\"", got)
	}
}
