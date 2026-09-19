package calendaradjustment

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func example() Rules {
	return Rules{
		Holidays: []Holiday{{"中秋节", "2026-09-25", "2026-09-27"}, {"国庆节", "2026-10-01", "2026-10-07"}},
		Moves:    []Move{{"国庆补课", "2026-10-06", "2026-09-20"}, {"国庆补课", "2026-10-07", "2026-10-10"}},
	}
}
func TestRulesApplyOriginalTeachingDate(t *testing.T) {
	r := example()
	if err := r.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		from, to string
		keep     bool
	}{
		{"2026-10-06", "2026-09-20", true}, {"2026-10-07", "2026-10-10", true},
		{"2026-09-20", "", false}, {"2026-10-10", "", false}, {"2026-09-25", "", false},
		{"2026-09-27", "", false}, {"2026-10-01", "", false}, {"2026-09-28", "2026-09-28", true},
		{"2027-09-25", "2027-09-25", true},
	} {
		d, _ := time.Parse(time.DateOnly, tc.from)
		actual, _, keep := r.Destination(d)
		if keep != tc.keep || (keep && actual.Format(time.DateOnly) != tc.to) {
			t.Fatalf("%s -> %s keep=%v", tc.from, actual, keep)
		}
	}
}
func TestRulesRejectContradictions(t *testing.T) {
	for _, scenario := range []string{"date", "overlap", "target-holiday", "duplicate-source", "duplicate-target", "cycle", "chain", "no-date", "name"} {
		t.Run(scenario, func(t *testing.T) {
			r := example()
			switch scenario {
			case "date":
				r.Holidays[0].StartDate = "2026-02-30"
			case "overlap":
				r.Holidays = append(r.Holidays, r.Holidays[0])
			case "target-holiday":
				r.Moves[0].ToDate = "2026-09-25"
			case "duplicate-source":
				r.Moves[1].FromDate = r.Moves[0].FromDate
			case "duplicate-target":
				r.Moves[1].ToDate = r.Moves[0].ToDate
			case "cycle":
				r.Holidays = []Holiday{}
				r.Moves[1] = Move{"cycle", r.Moves[0].ToDate, r.Moves[0].FromDate}
			case "chain":
				r.Moves[1].FromDate = r.Moves[0].ToDate
			case "no-date":
				r.Moves[0].FromDate = "10-06"
			case "name":
				r.Holidays[0].Name = ""
			}
			if !errors.Is(r.Validate(), ErrInvalid) {
				t.Fatal("accepted invalid rules")
			}
		})
	}
}
func TestDraftStrictJSON(t *testing.T) {
	good := `{"rules":{"holidays":[],"moves":[]},"warnings":["通知没有明确教学对应关系，请手动补充"]}`
	draft, err := decodeDraft(good)
	if err != nil || len(draft.Warnings) != 1 {
		t.Fatal(err)
	}
	for _, bad := range []string{"```json\n" + good + "\n```", good + good, strings.Replace(good, `"warnings"`, `"execute"`, 1), `{"rules":{"holidays":[],"moves":[]}}`} {
		if _, err := decodeDraft(bad); !errors.Is(err, ErrAIOutput) {
			t.Fatal("accepted invalid AI response")
		}
	}
}
