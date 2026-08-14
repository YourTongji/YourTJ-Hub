package pkservice

import (
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pk"
)

// GradesResult P3 grades 输出项。
type GradesResult struct {
	GradeList []int `json:"gradeList"`
}

// FindGradesByCalendar P3：某学期计划内课程的年级列表（倒序）。
func FindGradesByCalendar(calendarId int) (GradesResult, error) {
	grades, err := pk.ListGradesByCalendar(calendarId)
	if err != nil {
		return GradesResult{}, err
	}
	if grades == nil {
		grades = []int{}
	}
	return GradesResult{GradeList: grades}, nil
}

// MajorItem P4 majors 输出项。
type MajorItem struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

// FindMajorsByGrade P4：年级 → 专业列表；calendarId 为 0 时不限定学期。
func FindMajorsByGrade(grade, calendarId int) ([]MajorItem, error) {
	options, err := pk.ListMajorsByGrade(grade, calendarId, calendarId != 0)
	if err != nil {
		return nil, err
	}
	items := make([]MajorItem, 0, len(options))
	for _, o := range options {
		items = append(items, MajorItem{Code: o.Code, Name: o.Name})
	}
	return items, nil
}
