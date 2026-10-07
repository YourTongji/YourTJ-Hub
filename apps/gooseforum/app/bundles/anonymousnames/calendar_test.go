package anonymousnames

import (
	"testing"
	"time"
)

func TestShanghaiCalendarBoundaries(t *testing.T) {
	before := time.Date(2026, 10, 6, 15, 59, 59, 0, time.UTC)
	day, next := Day(before)
	if day != "2026-10-06" || !next.Equal(before.Add(time.Second)) {
		t.Fatalf("%s %v", day, next)
	}
	day, _ = Day(next)
	if day != "2026-10-07" {
		t.Fatal(day)
	}
	for _, test := range []struct{ year, month, day, wantYear, wantMonth, wantDay int }{
		{2024, 2, 29, 2025, 2, 28}, {2023, 2, 28, 2024, 2, 28},
		{2026, 10, 6, 2027, 10, 6}, {2026, 12, 31, 2027, 12, 31},
	} {
		selected := time.Date(test.year, time.Month(test.month), test.day, 23, 42, 13, 124, Shanghai)
		got := Anniversary(selected)
		want := time.Date(test.wantYear, time.Month(test.wantMonth), test.wantDay, 23, 42, 13, 124, Shanghai)
		if !got.Equal(want) {
			t.Fatalf("%v -> %v, want %v", selected, got, want)
		}
	}
}
