package campusservice

import (
	"context"
	"slices"
	"strings"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/calendaradjustment"
)

// TeachingDay describes the school-local date and the original timetable it uses.
// Empty sourceDate means no teaching on this date (holiday or moved away).
type TeachingDay struct {
	Date         string `json:"date"`
	SourceDate   string `json:"sourceDate"`
	Kind         string `json:"kind"`
	Label        string `json:"label"`
	SectionCount int    `json:"sectionCount"`
}

// Today reads calendar and timetable under one binding fence, just like export.
// It never falls back to an unadjusted schedule when published rules cannot load.
func (s *Service) Today(ctx context.Context, id uint64, rules calendaradjustment.Rules) (Dataset, error) {
	value, err := s.readPrivate(ctx, id, func(c Credentials) (any, error) {
		calendar, err := s.provider.Data(ctx, "calendar", c.Access)
		if err != nil {
			return nil, err
		}
		timetable, err := s.provider.Data(ctx, "timetable", c.Access)
		if err != nil {
			return nil, err
		}
		return buildToday(normalize("calendar", calendar), normalize("timetable", timetable), time.Now(), rules)
	})
	if err != nil {
		return Dataset{}, err
	}
	return value.(Dataset), nil
}

func buildToday(calendar, timetable Dataset, now time.Time, rules calendaradjustment.Rules) (Dataset, error) {
	if err := rules.Validate(); err != nil {
		return Dataset{}, err
	}
	begin, end, weeks, err := calendarBounds(calendar)
	if err != nil {
		return Dataset{}, err
	}
	if timetable.Status == "unavailable" {
		return Dataset{}, ErrUpstream
	}
	date, _ := time.ParseInLocation(time.DateOnly, now.In(begin.Location()).Format(time.DateOnly), begin.Location())
	source, label, kind := rules.TeachingDate(date)
	result := emptyDataset("today", "empty")
	result.TeachingDay = &TeachingDay{Date: date.Format(time.DateOnly), Kind: kind, Label: label, SectionCount: len(calendarSectionTimes(timetable.Events, nil))}
	if source.IsZero() {
		return result, nil
	}
	result.TeachingDay.SourceDate = source.Format(time.DateOnly)
	week := int(source.Sub(begin)/(7*24*time.Hour)) + 1
	if source.Before(begin) || source.After(end) || week > weeks {
		if kind == "makeup" {
			return Dataset{}, ErrCalendarIncomplete
		}
		return result, nil
	}
	day := (int(source.Weekday())+6)%7 + 1
	for _, event := range timetable.Events {
		if strings.TrimSpace(event.Name) == "" || event.Day < 1 || event.Day > 7 || event.Start < 1 || event.End < event.Start || event.End > 12 || len(event.Weeks) == 0 {
			return Dataset{}, ErrCalendarIncomplete
		}
		for _, w := range event.Weeks {
			if w < 1 || w > weeks || begin.AddDate(0, 0, (w-1)*7+event.Day-1).After(end) {
				return Dataset{}, ErrCalendarIncomplete
			}
		}
		if event.Day == day && slices.Contains(event.Weeks, week) {
			result.Events = append(result.Events, event)
		}
	}
	slices.SortStableFunc(result.Events, func(a, b Event) int { return a.Start - b.Start })
	if len(result.Events) > 0 {
		result.Status = "ready"
	}
	return result, nil
}
