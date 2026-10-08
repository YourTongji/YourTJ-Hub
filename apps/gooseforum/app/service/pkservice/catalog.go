package pkservice

import (
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/localcache"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pk"
)

var (
	calendarsCache = &localcache.Cache[[]CalendarItem]{MaxEntries: 1}
	campusesCache  = &localcache.Cache[[]CampusItem]{MaxEntries: 1}
	facultiesCache = &localcache.Cache[[]FacultyItem]{MaxEntries: 1}
)

const catalogCacheTTL = 10 * time.Minute

// InvalidateCatalogCache clears in-memory caches for PK calendars, campuses, and faculties.
func InvalidateCatalogCache() {
	calendarsCache.Clear()
	campusesCache.Clear()
	facultiesCache.Clear()
}

// cloneSlice returns a shallow copy of items. Safe for structs composed solely of value types (e.g. CampusItem, FacultyItem).
func cloneSlice[T any](items []T) []T {
	if items == nil {
		return nil
	}
	out := make([]T, len(items))
	copy(out, items)
	return out
}

func cloneStringPtr(s *string) *string {
	if s == nil {
		return nil
	}
	cp := *s
	return &cp
}

// cloneCalendars returns a deep copy of items, allocating fresh string pointers
// for StartDate and EndDate so caller mutations cannot taint the cached entries.
func cloneCalendars(items []CalendarItem) []CalendarItem {
	if items == nil {
		return nil
	}
	out := make([]CalendarItem, len(items))
	for i, item := range items {
		out[i] = CalendarItem{
			CalendarId:   item.CalendarId,
			CalendarName: item.CalendarName,
			StartDate:    cloneStringPtr(item.StartDate),
			EndDate:      cloneStringPtr(item.EndDate),
		}
	}
	return out
}

// CalendarItem P1 calendars 输出项。
type CalendarItem struct {
	CalendarId   uint64 `json:"calendarId"`
	CalendarName string `json:"calendarName"`
	// 学期起止日期（纯日期 "YYYY-MM-DD"，可空）：部署 config [pk.semester_dates] 维护、
	// course-pk-sync 写入；未配置输出 null。排课器用于「当前周次」定位与日期条展示。
	StartDate *string `json:"startDate"`
	EndDate   *string `json:"endDate"`
}

// formatPkDate 输出纯日期 "YYYY-MM-DD"；nil 保持 nil（未配置学期日期）。
func formatPkDate(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format("2006-01-02")
	return &s
}

// ListCalendars P1：最近 8 个学期（calendarId 倒序）。
func ListCalendars() ([]CalendarItem, error) {
	items, err := calendarsCache.GetOrLoadE("all", func() ([]CalendarItem, error) {
		entities, err := pk.ListCalendars(8)
		if err != nil {
			return nil, err
		}
		items := make([]CalendarItem, 0, len(entities))
		for _, e := range entities {
			items = append(items, CalendarItem{
				CalendarId:   e.CalendarId,
				CalendarName: e.CalendarIdI18n,
				StartDate:    formatPkDate(e.StartDate),
				EndDate:      formatPkDate(e.EndDate),
			})
		}
		return items, nil
	}, catalogCacheTTL)
	if err != nil {
		return nil, err
	}
	return cloneCalendars(items), nil
}

// CampusItem P2 校区输出项。
type CampusItem struct {
	CampusId   string `json:"campusId"`
	CampusName string `json:"campusName"`
}

// ListCampuses P2：校区列表。
func ListCampuses() ([]CampusItem, error) {
	items, err := campusesCache.GetOrLoadE("all", func() ([]CampusItem, error) {
		entities, err := pk.ListCampuses()
		if err != nil {
			return nil, err
		}
		items := make([]CampusItem, 0, len(entities))
		for _, e := range entities {
			items = append(items, CampusItem{CampusId: e.Campus, CampusName: e.CampusI18n})
		}
		return items, nil
	}, catalogCacheTTL)
	if err != nil {
		return nil, err
	}
	return cloneSlice(items), nil
}

// FacultyItem P2 院系输出项。
type FacultyItem struct {
	FacultyId   string `json:"facultyId"`
	FacultyName string `json:"facultyName"`
}

// ListFaculties P2：院系列表。
func ListFaculties() ([]FacultyItem, error) {
	items, err := facultiesCache.GetOrLoadE("all", func() ([]FacultyItem, error) {
		entities, err := pk.ListFaculties()
		if err != nil {
			return nil, err
		}
		items := make([]FacultyItem, 0, len(entities))
		for _, e := range entities {
			items = append(items, FacultyItem{FacultyId: e.Faculty, FacultyName: e.FacultyI18n})
		}
		return items, nil
	}, catalogCacheTTL)
	if err != nil {
		return nil, err
	}
	return cloneSlice(items), nil
}
