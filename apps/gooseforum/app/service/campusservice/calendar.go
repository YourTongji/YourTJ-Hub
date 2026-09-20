package campusservice

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/defaultconfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/calendaradjustment"
)

var ErrCalendarIncomplete = errors.New("campus calendar dates or course times incomplete")
var ErrCalendarEmpty = errors.New("campus calendar has no scheduled classes")

// CalendarExport is an explicit user-owned snapshot, never a public subscription.
type CalendarExport struct {
	Filename   string `json:"filename"`
	Content    string `json:"content"`
	EventCount int    `json:"eventCount"`
}

func (s *Service) ExportCalendar(ctx context.Context, id uint64, times []pageConfig.ScheduleSectionTime, adjustments ...calendaradjustment.Rules) (CalendarExport, error) {
	// Both upstream reads share credentials, refresh retries and a single binding
	// fence. A replacement/unlink cannot mix a calendar with another identity.
	value, err := s.readPrivate(ctx, id, func(c Credentials) (any, error) {
		calendar, e := s.provider.Data(ctx, "calendar", c.Access)
		if e != nil {
			return nil, e
		}
		timetable, e := s.provider.Data(ctx, "timetable", c.Access)
		if e != nil {
			return nil, e
		}
		namespace := fingerprint(s.config.IdentityKey, fmt.Sprintf("calendar\x00%d\x00%s", id, c.StudentID))
		return buildCalendar(normalize("calendar", calendar), normalize("timetable", timetable), times, namespace, time.Now(), adjustments...)
	})
	if err != nil {
		return CalendarExport{}, err
	}
	return value.(CalendarExport), nil
}

func calendarMetric(d Dataset, label string) string {
	for _, m := range d.Metrics {
		if m.Label == label {
			return m.Value
		}
	}
	return ""
}

// Matches the campus grid: 11 sections with configured overrides, or the
// historical 12-section defaults when a course occupies section 12.
func calendarSectionTimes(events []Event, overrides []pageConfig.ScheduleSectionTime) []pageConfig.ScheduleSectionTime {
	times := defaultconfig.GetDefaultScheduleSettingsConfig().SectionTimes
	if slices.ContainsFunc(events, func(e Event) bool { return e.End == 12 }) {
		for i := 8; i < len(times); i++ {
			times[i].Section++
		}
		return slices.Insert(times, 8, pageConfig.ScheduleSectionTime{Section: 9, Start: "17:10", End: "17:55"})
	}
	for _, override := range overrides {
		if override.Section >= 1 && override.Section <= len(times) {
			times[override.Section-1] = override
		}
	}
	return times
}

// calendarBounds is shared by daily display and calendar export; neither guesses
// the teaching week from an incomplete calendar.
func calendarBounds(calendar Dataset) (time.Time, time.Time, int, error) {
	zone := time.FixedZone("Asia/Shanghai", 8*3600)
	begin, err := time.ParseInLocation(time.DateOnly, calendarMetric(calendar, "学期开始"), zone)
	end, endErr := time.ParseInLocation(time.DateOnly, calendarMetric(calendar, "学期结束"), zone)
	weeks, weekErr := strconv.Atoi(calendarMetric(calendar, "学期周数"))
	// Do not guess which teaching week a non-Monday term boundary belongs to.
	if err != nil || endErr != nil || weekErr != nil || begin.Weekday() != time.Monday || end.Before(begin) || weeks < 1 || weeks > 53 || end.Sub(begin) >= 371*24*time.Hour {
		return time.Time{}, time.Time{}, 0, ErrCalendarIncomplete
	}
	return begin, end, weeks, nil
}

func buildCalendar(calendar, timetable Dataset, overrides []pageConfig.ScheduleSectionTime, namespace string, now time.Time, adjustments ...calendaradjustment.Rules) (CalendarExport, error) {
	rules := calendaradjustment.Empty()
	if len(adjustments) > 0 {
		rules = adjustments[0]
	}
	if err := rules.Validate(); err != nil {
		return CalendarExport{}, err
	}
	begin, end, weeks, err := calendarBounds(calendar)
	if err != nil {
		return CalendarExport{}, err
	}
	// A source outside the known term cannot replace a day within it: no source
	// timetable is available, so refuse instead of silently dropping that day.
	for _, m := range rules.Moves {
		if (m.FromDate < begin.Format(time.DateOnly) || m.FromDate > end.Format(time.DateOnly)) && m.ToDate >= begin.Format(time.DateOnly) && m.ToDate <= end.Format(time.DateOnly) {
			return CalendarExport{}, ErrCalendarIncomplete
		}
	}
	times := calendarSectionTimes(timetable.Events, overrides)
	lines := []string{"BEGIN:VCALENDAR", "VERSION:2.0", "PRODID:-//YourTJ//Campus Calendar//ZH", "CALSCALE:GREGORIAN", "X-WR-CALNAME:" + calendarText("YourTJ · "+calendarMetric(calendar, "当前学期"))}
	seen := map[string]bool{}
	for _, course := range timetable.Events {
		if strings.TrimSpace(course.Name) == "" || course.Day < 1 || course.Day > 7 || course.Start < 1 || course.End < course.Start || course.End > len(times) || len(course.Weeks) == 0 {
			return CalendarExport{}, ErrCalendarIncomplete
		}
		startTime, startErr := time.Parse("15:04", times[course.Start-1].Start)
		endTime, finishErr := time.Parse("15:04", times[course.End-1].End)
		if startErr != nil || finishErr != nil || !endTime.After(startTime) {
			return CalendarExport{}, ErrCalendarIncomplete
		}
		for _, week := range course.Weeks {
			if week < 1 || week > weeks {
				return CalendarExport{}, ErrCalendarIncomplete
			}
			day := begin.AddDate(0, 0, (week-1)*7+course.Day-1)
			if day.After(end) {
				return CalendarExport{}, ErrCalendarIncomplete
			}
			originalDay := day
			actual, adjustmentName, keep := rules.Destination(day)
			if !keep {
				continue
			}
			day = actual
			start := day.Add(time.Duration(startTime.Hour()*60+startTime.Minute()) * time.Minute)
			finish := day.Add(time.Duration(endTime.Hour()*60+endTime.Minute()) * time.Minute)
			// Stable, opaque IDs deduplicate exact repeated source slots and do not
			// contain student IDs, forum IDs or credentials.
			originalStart := originalDay.Add(time.Duration(startTime.Hour()*60+startTime.Minute()) * time.Minute)
			originalFinish := originalDay.Add(time.Duration(endTime.Hour()*60+endTime.Minute()) * time.Minute)
			identity, marshalErr := json.Marshal([]string{namespace, begin.Format(time.DateOnly), course.Name, course.Teacher, course.Room, course.Campus, originalStart.Format(time.RFC3339), originalFinish.Format(time.RFC3339)})
			if marshalErr != nil {
				return CalendarExport{}, fmt.Errorf("encode calendar event identity: %w", marshalErr)
			}
			uid := fmt.Sprintf("%x@campus.yourtj", sha256.Sum256(identity))
			if seen[uid] {
				continue
			}
			seen[uid] = true
			if len(seen) > 10000 {
				return CalendarExport{}, ErrCalendarIncomplete
			}
			location := strings.TrimSpace(course.Campus + " " + course.Room)
			description := fmt.Sprintf("%s\n第 %d 周 · 第 %d–%d 节", calendarMetric(calendar, "当前学期"), week, course.Start, course.End)
			if adjustmentName != "" {
				description += fmt.Sprintf("\n%s：原 %s 的课程调整至 %s", adjustmentName, originalDay.Format(time.DateOnly), day.Format(time.DateOnly))
			}
			if course.Teacher != "" {
				description += "\n教师：" + course.Teacher
			}
			lines = append(lines, "BEGIN:VEVENT", "UID:"+uid, "DTSTAMP:"+now.UTC().Format("20060102T150405Z"), "DTSTART:"+start.UTC().Format("20060102T150405Z"), "DTEND:"+finish.UTC().Format("20060102T150405Z"), "SUMMARY:"+calendarText(course.Name), "LOCATION:"+calendarText(location), "DESCRIPTION:"+calendarText(description), "CLASS:PRIVATE", "END:VEVENT")
		}
	}
	if len(seen) == 0 {
		return CalendarExport{}, ErrCalendarEmpty
	}
	lines = append(lines, "END:VCALENDAR")
	var content strings.Builder
	for _, line := range lines {
		// RFC 5545 folding is measured in UTF-8 octets, never split a rune.
		column := 0
		for _, r := range line {
			text := string(r)
			if column+len(text) > 75 {
				content.WriteString("\r\n ")
				column = 1
			}
			content.WriteString(text)
			column += len(text)
		}
		content.WriteString("\r\n")
	}
	return CalendarExport{Filename: "yourtj-courses-" + begin.Format(time.DateOnly) + ".ics", Content: content.String(), EventCount: len(seen)}, nil
}

func calendarText(value string) string {
	value = strings.ReplaceAll(strings.ReplaceAll(value, "\r\n", "\n"), "\r", "\n")
	value = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && r != '\n' {
			return -1
		}
		return r
	}, value)
	return strings.NewReplacer("\\", "\\\\", "\n", "\\n", ";", "\\;", ",", "\\,").Replace(value)
}
