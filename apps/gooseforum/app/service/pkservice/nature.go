package pkservice

import (
	"cmp"
	"slices"
	"sort"
	"strconv"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pk"
)

// OptionalTypeItem P6 通识/选修类型输出项。
type OptionalTypeItem struct {
	CourseLabelId   int    `json:"courseLabelId"`
	CourseLabelName string `json:"courseLabelName"`
}

// FindOptionalTypes P6：某学期通识/选修课程性质。
func FindOptionalTypes(calendarId int) ([]OptionalTypeItem, error) {
	options, err := pk.ListOptionalTypesByCalendar(calendarId, OPTIONAL_LABEL_NAMES)
	if err != nil {
		return nil, err
	}
	items := make([]OptionalTypeItem, 0, len(options))
	for _, o := range options {
		items = append(items, OptionalTypeItem{CourseLabelId: o.CourseLabelId, CourseLabelName: o.CourseLabelName})
	}
	return items, nil
}

// NatureCourseItem P7 性质下的一门课程。
type NatureCourseItem struct {
	Campus          []string `json:"campus"`
	CourseCode      string   `json:"courseCode"`
	CourseName      string   `json:"courseName"`
	Faculty         string   `json:"faculty"`
	FacultyI18n     string   `json:"facultyI18n"`
	Credit          float64  `json:"credit"`
	CourseLabelName string   `json:"courseLabelName"`
	CrossDiscipline bool     `json:"crossDiscipline"`
}

// CourseByNatureItem P7 courses-by-nature 输出项（按 labelName 合并）。
type CourseByNatureItem struct {
	CourseLabelId   int                `json:"courseLabelId"`
	CourseLabelIds  []int              `json:"courseLabelIds"`
	CourseLabelName string             `json:"courseLabelName"`
	CrossDiscipline bool               `json:"crossDiscipline"`
	Courses         []NatureCourseItem `json:"courses"`
}

// FindCoursesByNature P7：按课程性质 id 查课程，按 labelName 合并去重。
func FindCoursesByNature(calendarId int, ids []int) ([]CourseByNatureItem, error) {
	if len(ids) == 0 {
		return []CourseByNatureItem{}, nil
	}
	rows, err := pk.ListCourseNatureRowsByLabelIds(calendarId, ids)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return []CourseByNatureItem{}, nil
	}

	// 按 labelName 分组，合并 courseLabelIds；课程按 (labelName, courseCode, faculty, credit) 去重。
	// 同一门课出现在多个性质分组时，每个分组都保留（对齐上游每 labelName 独立 courseMap）。
	byLabel := map[string]*CourseByNatureItem{}
	courseIndex := map[string]map[string]*NatureCourseItem{}
	labelOrder := []string{}
	for _, row := range rows {
		labelName := normalizeText(row.CourseLabelName)
		if labelName == "" {
			continue
		}
		item, ok := byLabel[labelName]
		if !ok {
			item = &CourseByNatureItem{
				CourseLabelId:   row.CourseLabelId,
				CourseLabelIds:  []int{row.CourseLabelId},
				CourseLabelName: labelName,
				CrossDiscipline: isCrossDisciplineLabel(labelName),
				Courses:         []NatureCourseItem{},
			}
			byLabel[labelName] = item
			courseIndex[labelName] = map[string]*NatureCourseItem{}
			labelOrder = append(labelOrder, labelName)
		} else {
			item.CourseLabelIds = append(item.CourseLabelIds, row.CourseLabelId)
		}
		key := courseKey(row.CourseCode, row.FacultyI18n, row.Credit)
		if course, ok := courseIndex[labelName][key]; ok {
			course.Campus = uniqueText(append(course.Campus, row.CampusI18n))
			course.CrossDiscipline = course.CrossDiscipline || isCrossDisciplineLabel(labelName)
			continue
		}
		course := NatureCourseItem{
			Campus:          nonEmptySlice(row.CampusI18n),
			CourseCode:      row.CourseCode,
			CourseName:      row.CourseName,
			Faculty:         row.FacultyI18n,
			FacultyI18n:     row.FacultyI18n,
			Credit:          row.Credit,
			CourseLabelName: labelName,
			CrossDiscipline: isCrossDisciplineLabel(labelName),
		}
		item.Courses = append(item.Courses, course)
		courseIndex[labelName][key] = &item.Courses[len(item.Courses)-1]
	}

	// 排序：labelName 分组按 courseLabelId 倒序；组内课程按 courseCode。
	output := make([]CourseByNatureItem, 0, len(labelOrder))
	for _, labelName := range labelOrder {
		item := byLabel[labelName]
		item.CourseLabelIds = uniqueIntsDesc(item.CourseLabelIds)
		item.CourseLabelId = maxInt(item.CourseLabelIds)
		slices.SortStableFunc(item.Courses, func(a, b NatureCourseItem) int {
			return cmp.Compare(a.CourseCode, b.CourseCode)
		})
		output = append(output, *item)
	}
	slices.SortStableFunc(output, func(a, b CourseByNatureItem) int {
		return cmp.Compare(b.CourseLabelId, a.CourseLabelId)
	})
	return output, nil
}

func courseKey(courseCode, faculty string, credit float64) string {
	return courseCode + "__" + faculty + "__" + formatFloat(credit)
}

func formatFloat(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}

func nonEmptySlice(v string) []string {
	if v == "" {
		return []string{}
	}
	return []string{v}
}

func uniqueIntsDesc(values []int) []int {
	seen := map[int]struct{}{}
	out := []int{}
	for _, v := range values {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(out)))
	return out
}

func maxInt(values []int) int {
	max := 0
	for _, v := range values {
		if v > max {
			max = v
		}
	}
	return max
}
