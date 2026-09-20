package campusservice

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/campus"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/calendaradjustment"
)

func todayRules() calendaradjustment.Rules {
	return calendaradjustment.Rules{Holidays: []calendaradjustment.Holiday{{Name: "国庆", StartDate: "2026-10-01", EndDate: "2026-10-07"}}, Moves: []calendaradjustment.Move{{Name: "国庆补课", FromDate: "2026-10-06", ToDate: "2026-09-20"}}}
}
func TestTodayUsesSourceWeekAndReplacesTargetClasses(t *testing.T) {
	cal, _ := calendarFixture()
	table := Dataset{Events: []Event{
		{Name: "原周日", Day: 7, Start: 1, End: 2, Weeks: []int{1}},
		{Name: "第四周周二", Day: 2, Start: 3, End: 4, Weeks: []int{4}},
		{Name: "第一周周二", Day: 2, Start: 1, End: 2, Weeks: []int{1}},
	}}
	for _, tc := range []struct{ now, kind, want string }{
		{"2026-09-19T17:00:00Z", "makeup", "第四周周二"}, // already Sunday in Shanghai
		{"2026-10-06T00:00:00Z", "holiday", ""},
		{"2026-10-01T00:00:00Z", "holiday", ""},
		{"2026-10-07T00:00:00Z", "holiday", ""},
		{"2026-09-15T00:00:00Z", "none", "第一周周二"},
		{"2027-01-18T00:00:00Z", "none", ""},
	} {
		t.Run(tc.now, func(t *testing.T) {
			now, _ := time.Parse(time.RFC3339, tc.now)
			result, err := buildToday(cal, table, now, todayRules())
			if err != nil {
				t.Fatal(err)
			}
			if result.TeachingDay.Kind != tc.kind {
				t.Fatalf("kind=%s", result.TeachingDay.Kind)
			}
			if tc.want == "" {
				if len(result.Events) != 0 || result.Status != "empty" {
					t.Fatal("unexpected classes")
				}
				return
			}
			if len(result.Events) != 1 || result.Events[0].Name != tc.want {
				t.Fatalf("wrong classes: %+v", result.Events)
			}
			// Daily display agrees with the existing forward export mapping.
			source, _ := time.Parse(time.DateOnly, result.TeachingDay.SourceDate)
			actual, _, keep := todayRules().Destination(source)
			if !keep || actual.Format(time.DateOnly) != result.TeachingDay.Date {
				t.Fatal("daily schedule differs from export")
			}
		})
	}
}
func TestTodayMovedAwayCrossYearAndIncompleteData(t *testing.T) {
	cal, table := calendarFixture()
	rules := calendaradjustment.Empty()
	rules.Moves = []calendaradjustment.Move{{Name: "跨年补课", FromDate: "2026-09-14", ToDate: "2027-01-02"}}
	now, _ := time.Parse(time.DateOnly, "2027-01-02")
	result, err := buildToday(cal, table, now, rules)
	if err != nil || len(result.Events) != 1 || result.TeachingDay.SourceDate != "2026-09-14" {
		t.Fatalf("cross-year makeup: %v", err)
	}
	now, _ = time.Parse(time.DateOnly, "2026-09-14")
	result, err = buildToday(cal, table, now, rules)
	if err != nil || len(result.Events) != 0 || result.TeachingDay.Kind != "moved" {
		t.Fatalf("source still scheduled: %v", err)
	}
	for _, scenario := range []string{"calendar", "source", "weeks", "upstream", "rules"} {
		t.Run(scenario, func(t *testing.T) {
			cal, table := calendarFixture()
			rules := todayRules()
			date, _ := time.Parse(time.DateOnly, "2026-09-20")
			switch scenario {
			case "calendar":
				cal.Metrics = nil
			case "source":
				rules.Moves[0].FromDate = "2025-10-06"
			case "weeks":
				table.Events[0].Weeks = nil
			case "upstream":
				table.Status = "unavailable"
			case "rules":
				rules.Moves[0].ToDate = "2026-10-01"
			}
			if _, err := buildToday(cal, table, date, rules); err == nil {
				t.Fatal("incomplete data silently displayed as no classes")
			}
		})
	}
}
func TestTodayUsesPrivateRefreshAndBindingFence(t *testing.T) {
	s, p := setup(t)
	provider := &calendarProvider{fakeProvider: p}
	s.provider = provider
	if _, err := s.Today(context.Background(), 1, todayRules()); err == nil {
		t.Fatal("unbound read allowed")
	}
	bind(t, s, 1)
	provider.failOnce = true
	if _, err := s.Today(context.Background(), 1, todayRules()); err != nil || p.refreshes.Load() != 1 {
		t.Fatalf("refresh read: %v", err)
	}
	provider.beforeTable = func() {
		b, _ := s.store.Get(1)
		if err := s.Unbind(context.Background(), 1, b.Revision); err != nil {
			t.Fatal(err)
		}
	}
	result, err := s.Today(context.Background(), 1, todayRules())
	if !errors.Is(err, campus.ErrChanged) || result.TeachingDay != nil {
		t.Fatal("late identity data escaped fence")
	}
}
