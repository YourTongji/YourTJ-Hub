package pkservice

import (
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pk"
)

// CalendarItem P1 calendars 输出项。
type CalendarItem struct {
	CalendarId   int    `json:"calendarId"`
	CalendarName string `json:"calendarName"`
}

// ListCalendars P1：最近 8 个学期（calendarId 倒序）。
func ListCalendars() ([]CalendarItem, error) {
	entities, err := pk.ListCalendars(8)
	if err != nil {
		return nil, err
	}
	items := make([]CalendarItem, 0, len(entities))
	for _, e := range entities {
		items = append(items, CalendarItem{CalendarId: e.CalendarId, CalendarName: e.CalendarIdI18n})
	}
	return items, nil
}

// CampusItem P2 校区输出项。
type CampusItem struct {
	CampusId   string `json:"campusId"`
	CampusName string `json:"campusName"`
}

// ListCampuses P2：校区列表。
func ListCampuses() ([]CampusItem, error) {
	entities, err := pk.ListCampuses()
	if err != nil {
		return nil, err
	}
	items := make([]CampusItem, 0, len(entities))
	for _, e := range entities {
		items = append(items, CampusItem{CampusId: e.Campus, CampusName: e.CampusI18n})
	}
	return items, nil
}

// FacultyItem P2 院系输出项。
type FacultyItem struct {
	FacultyId   string `json:"facultyId"`
	FacultyName string `json:"facultyName"`
}

// ListFaculties P2：院系列表。
func ListFaculties() ([]FacultyItem, error) {
	entities, err := pk.ListFaculties()
	if err != nil {
		return nil, err
	}
	items := make([]FacultyItem, 0, len(entities))
	for _, e := range entities {
		items = append(items, FacultyItem{FacultyId: e.Faculty, FacultyName: e.FacultyI18n})
	}
	return items, nil
}
