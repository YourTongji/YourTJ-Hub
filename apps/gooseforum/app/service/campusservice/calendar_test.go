package campusservice

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/campus"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
)

func calendarFixture() (Dataset, Dataset) {
	return Dataset{Metrics: []Metric{{Label: "学期开始", Value: "2026-09-14"}, {Label: "学期结束", Value: "2027-01-17"}, {Label: "学期周数", Value: "18"}, {Label: "当前学期", Value: "秋季学期"}}}, Dataset{Events: []Event{{Name: "高等数学", Teacher: "示例教师", Room: "A101", Campus: "四平路", Day: 1, Start: 1, End: 2, Weeks: []int{1, 3, 3}}}}
}

func TestCalendarExactWeeksTimesAndStableUID(t *testing.T) {
	cal, table := calendarFixture()
	first, err := buildCalendar(cal, table, nil, "opaque-owner", time.Now())
	if err != nil || first.EventCount != 2 {
		t.Fatalf("expected two odd-week occurrences: count=%d error=%v", first.EventCount, err)
	}
	for _, expected := range []string{"DTSTART:20260914T000000Z", "DTEND:20260914T013500Z", "DTSTART:20260928T000000Z", "LOCATION:四平路 A101", "CLASS:PRIVATE"} {
		if !strings.Contains(first.Content, expected) {
			t.Errorf("missing %s", expected)
		}
	}
	if strings.Contains(first.Content, "20260921") || strings.Contains(first.Content, "opaque-owner") {
		t.Fatal("invented an even week or leaked owner identifier")
	}
	// Ordering and timestamp changes cannot create different event IDs.
	table.Events[0].Weeks = []int{3, 1}
	second, _ := buildCalendar(cal, table, nil, "opaque-owner", time.Now().Add(time.Hour))
	for _, line := range strings.Split(first.Content, "\r\n") {
		if strings.HasPrefix(line, "UID:") && !strings.Contains(second.Content, line) {
			t.Fatal("unstable event UID")
		}
	}
}

func TestCalendarSectionsAndWeekdayBoundaries(t *testing.T) {
	cal, table := calendarFixture()
	table.Events[0].Day = 7
	table.Events[0].Weeks = []int{18}
	table.Events[0].Start, table.Events[0].End = 9, 11
	result, err := buildCalendar(cal, table, []pageConfig.ScheduleSectionTime{{Section: 9, Start: "18:40", End: "19:25"}}, "owner", time.Now())
	if err != nil || !strings.Contains(result.Content, "DTSTART:20270117T104000Z") || !strings.Contains(result.Content, "DTEND:20270117T125500Z") {
		t.Fatalf("last Sunday/configured evening incorrect: %v", err)
	}
	table.Events[0].End = 12
	result, err = buildCalendar(cal, table, []pageConfig.ScheduleSectionTime{{Section: 9, Start: "18:40", End: "19:25"}}, "owner", time.Now())
	if err != nil || !strings.Contains(result.Content, "DTSTART:20270117T091000Z") || !strings.Contains(result.Content, "DTEND:20270117T125500Z") {
		t.Fatalf("historical section numbering incorrect: %v", err)
	}
}

func TestCalendarEscapingAndUTF8Folding(t *testing.T) {
	cal, table := calendarFixture()
	table.Events[0].Name = strings.Repeat("中文🙂", 40) + ",;\\\r\nBEGIN:VEVENT\x00"
	result, err := buildCalendar(cal, table, nil, "owner", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(result.Content, "\r\n") {
		if len(line) > 75 || !utf8.ValidString(line) {
			t.Fatal("invalid UTF-8 line folding")
		}
	}
	unfolded := strings.ReplaceAll(result.Content, "\r\n ", "")
	if !strings.Contains(unfolded, `\,\;\\\nBEGIN:VEVENT`) || strings.ContainsRune(unfolded, 0) || strings.Count(unfolded, "\r\nBEGIN:VEVENT\r\n") != 2 {
		t.Fatal("unescaped calendar control text")
	}
}

func TestCalendarRejectsUnknownOrInvalidDatesAndSlots(t *testing.T) {
	for _, scenario := range []string{"missing-date", "non-monday", "bad-weeks", "missing-weeks", "outside-term", "bad-day", "bad-section", "bad-time", "empty"} {
		t.Run(scenario, func(t *testing.T) {
			cal, table := calendarFixture()
			var overrides []pageConfig.ScheduleSectionTime
			want := ErrCalendarIncomplete
			switch scenario {
			case "missing-date":
				cal.Metrics[0].Value = ""
			case "non-monday":
				cal.Metrics[0].Value = "2026-09-13"
			case "bad-weeks":
				table.Events[0].Weeks = []int{0, 19}
			case "missing-weeks":
				table.Events[0].Weeks = nil
			case "outside-term":
				cal.Metrics[1].Value = "2026-09-20"
			case "bad-day":
				table.Events[0].Day = 8
			case "bad-section":
				table.Events[0].Start = 0
			case "bad-time":
				overrides = []pageConfig.ScheduleSectionTime{{Section: 2, Start: "08:50", End: "07:00"}}
			case "empty":
				table.Events = nil
				want = ErrCalendarEmpty
			}
			result, err := buildCalendar(cal, table, overrides, "owner", time.Now())
			if !errors.Is(err, want) || result.Content != "" {
				t.Fatalf("expected no file and %v, got %v", want, err)
			}
		})
	}
}

type calendarProvider struct {
	*fakeProvider
	beforeTable func()
	failOnce    bool
}

func (p *calendarProvider) Data(_ context.Context, key, _ string) (any, error) {
	if key == "calendar" {
		return map[string]any{"name": "秋季", "schoolCalendar": map[string]any{"beginDay": int64(1789315200000), "endDay": int64(1800115200000), "weekNum": 18}}, nil
	}
	if p.beforeTable != nil {
		p.beforeTable()
	}
	if p.failOnce {
		p.failOnce = false
		return nil, ErrAuthorization
	}
	return []any{map[string]any{"courseName": "测试课程", "timeTableList": []any{map[string]any{"dayOfWeek": 1, "timeStart": 1, "timeEnd": 2, "weeks": []any{1, 3}}}}}, nil
}

func TestCalendarExportUsesPrivateRefreshAndBindingFence(t *testing.T) {
	s, p := setup(t)
	provider := &calendarProvider{fakeProvider: p}
	s.provider = provider
	if _, err := s.ExportCalendar(context.Background(), 1, nil); err == nil {
		t.Fatal("unbound export allowed")
	}
	bind(t, s, 1)
	provider.failOnce = true
	result, err := s.ExportCalendar(context.Background(), 1, nil)
	if err != nil || result.EventCount != 2 || p.refreshes.Load() != 1 {
		t.Fatalf("refresh export: events=%d err=%v refreshes=%d", result.EventCount, err, p.refreshes.Load())
	}
	provider.beforeTable = func() {
		binding, _ := s.store.Get(1)
		if err := s.Unbind(context.Background(), 1, binding.Revision); err != nil {
			t.Fatal(err)
		}
	}
	result, err = s.ExportCalendar(context.Background(), 1, nil)
	if !errors.Is(err, campus.ErrChanged) || result.Content != "" {
		t.Fatal("unlinked identity exported private calendar")
	}
}
