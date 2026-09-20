// Package calendaradjustment owns administrator-confirmed teaching-date overrides.
// It processes public notices only; personal timetables are never sent to an LLM.
package calendaradjustment

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
)

const settingsKey = "campusCalendarAdjustments"

var ErrConflict = errors.New("calendar rules changed")
var ErrInvalid = errors.New("invalid calendar rules")

type Holiday struct {
	Name      string `json:"name"`
	StartDate string `json:"startDate"`
	EndDate   string `json:"endDate"`
}
type Move struct {
	Name     string `json:"name"`
	FromDate string `json:"fromDate"`
	ToDate   string `json:"toDate"`
}
type Rules struct {
	Holidays []Holiday `json:"holidays"`
	Moves    []Move    `json:"moves"`
}
type Settings struct {
	Revision string `json:"revision"`
	Rules    Rules  `json:"rules"`
}

func Empty() Rules               { return Rules{Holidays: []Holiday{}, Moves: []Move{}} }
func revision(raw string) string { return fmt.Sprintf("%x", sha256.Sum256([]byte(raw))) }
func Read(ctx context.Context) (Settings, error) {
	raw, err := pageConfig.ReadConfig(ctx, settingsKey)
	if err != nil {
		return Settings{}, err
	}
	rules := Empty()
	if raw != "" {
		if err = json.Unmarshal([]byte(raw), &rules); err != nil {
			return Settings{}, ErrInvalid
		}
	}
	if err = rules.Validate(); err != nil {
		return Settings{}, err
	}
	return Settings{Revision: revision(raw), Rules: rules}, nil
}
func Save(ctx context.Context, settings Settings) (Settings, error) {
	if err := settings.Rules.Validate(); err != nil {
		return Settings{}, err
	}
	raw, err := pageConfig.ReadConfig(ctx, settingsKey)
	if err != nil {
		return Settings{}, err
	}
	if settings.Revision != revision(raw) {
		return Settings{}, ErrConflict
	}
	next, err := json.Marshal(settings.Rules)
	if err != nil {
		return Settings{}, err
	}
	if raw != string(next) {
		ok, err := pageConfig.CompareAndSwapConfig(ctx, settingsKey, raw, string(next))
		if err != nil {
			return Settings{}, err
		}
		if !ok {
			return Settings{}, ErrConflict
		}
	}
	return Settings{Revision: revision(string(next)), Rules: settings.Rules}, nil
}
func date(value string) (time.Time, error) {
	t, err := time.Parse(time.DateOnly, value)
	if err != nil || len(value) != 10 || t.Year() < 2000 || t.Year() > 2100 {
		return time.Time{}, ErrInvalid
	}
	return t, nil
}
func invalid(message string) error { return fmt.Errorf("%w: %s", ErrInvalid, message) }
func validName(name string) bool {
	return strings.TrimSpace(name) != "" && utf8.RuneCountInString(name) <= 80
}
func (r Rules) Validate() error {
	if r.Holidays == nil || r.Moves == nil || len(r.Holidays) > 60 || len(r.Moves) > 120 {
		return invalid("请提供放假和补课列表，最多 60 段假期、120 次补课")
	}
	days := map[string]bool{}
	for _, h := range r.Holidays {
		start, e1 := date(h.StartDate)
		end, e2 := date(h.EndDate)
		if !validName(h.Name) || e1 != nil || e2 != nil || end.Before(start) || end.Sub(start) > 365*24*time.Hour {
			return invalid("假期名称或日期范围无效（日期需含年份，范围不超过一年）")
		}
		for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
			key := d.Format(time.DateOnly)
			if days[key] {
				return invalid("放假区间不能重叠")
			}
			days[key] = true
		}
	}
	from, to := map[string]bool{}, map[string]bool{}
	for _, m := range r.Moves {
		_, e1 := date(m.FromDate)
		_, e2 := date(m.ToDate)
		if !validName(m.Name) || e1 != nil || e2 != nil || m.FromDate == m.ToDate {
			return invalid("补课需填写名称及两个不同的完整日期")
		}
		if from[m.FromDate] || to[m.ToDate] {
			return invalid("同一原教学日或补课日不能重复")
		}
		if days[m.ToDate] {
			return invalid("实际补课日不能同时设为放假")
		}
		from[m.FromDate], to[m.ToDate] = true, true
	}
	for d := range from {
		if to[d] {
			return invalid("不支持连锁或循环调课，请直接填写原教学日到最终上课日")
		}
	}
	return nil
}

// Destination applies one non-recursive mapping to ORIGINAL occurrences. A makeup
// day replaces that day's native timetable; a moved source wins over its holiday.
func (r Rules) Destination(original time.Time) (time.Time, string, bool) {
	key := original.Format(time.DateOnly)
	for _, m := range r.Moves {
		if m.FromDate == key {
			d, _ := time.ParseInLocation(time.DateOnly, m.ToDate, original.Location())
			return d, m.Name, true
		}
	}
	for _, m := range r.Moves {
		if m.ToDate == key {
			return time.Time{}, "", false
		}
	}
	for _, h := range r.Holidays {
		if key >= h.StartDate && key <= h.EndDate {
			return time.Time{}, "", false
		}
	}
	return original, "", true
}

// TeachingDate is the inverse of Destination for a date displayed in a daily
// schedule. The target replaces its native classes; never apply mappings twice.
func (r Rules) TeachingDate(actual time.Time) (time.Time, string, string) {
	key := actual.Format(time.DateOnly)
	for _, m := range r.Moves {
		if m.ToDate == key {
			source, _ := time.ParseInLocation(time.DateOnly, m.FromDate, actual.Location())
			return source, m.Name, "makeup"
		}
	}
	for _, h := range r.Holidays {
		if key >= h.StartDate && key <= h.EndDate {
			return time.Time{}, h.Name, "holiday"
		}
	}
	for _, m := range r.Moves {
		if m.FromDate == key {
			return time.Time{}, m.Name, "moved"
		}
	}
	return actual, "", "none"
}
