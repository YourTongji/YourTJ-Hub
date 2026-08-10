package courseservice

import (
	"errors"

	"github.com/leancodebox/GooseForum/app/models/forum/course"
)

// CourseSummary 课程列表卡片（Slice A 只读目录；评价统计在 Slice C 追加）。
type CourseSummary struct {
	Id          uint64   `json:"id"`
	PrimaryCode string   `json:"primaryCode"`
	Name        string   `json:"name"`
	Department  string   `json:"department"`
	CreditX10   int      `json:"creditX10"`
	Aliases     []string `json:"aliases,omitempty"`
	Instructors []string `json:"instructors,omitempty"`
	RecentTerms []string `json:"recentTerms,omitempty"`
}

// OfferingSummary 开课实例摘要（详情页）。
type OfferingSummary struct {
	Id          uint64   `json:"id"`
	TermCode    string   `json:"termCode"`
	TermName    string   `json:"termName,omitempty"`
	Campus      string   `json:"campus,omitempty"`
	Faculty     string   `json:"faculty,omitempty"`
	Instructors []string `json:"instructors,omitempty"`
}

// CourseDetail 课程详情页数据。
type CourseDetail struct {
	Id          uint64            `json:"id"`
	PrimaryCode string            `json:"primaryCode"`
	Name        string            `json:"name"`
	Department  string            `json:"department"`
	CreditX10   int               `json:"creditX10"`
	Aliases     []string          `json:"aliases,omitempty"`
	Offerings   []OfferingSummary `json:"offerings,omitempty"`
}

// CatalogQuery 目录筛选条件。
type CatalogQuery struct {
	Keyword    string
	Department string
	TermCode   string
	Campus     string
	Page       int
	Size       int
}

// CatalogPage 目录分页结果。
type CatalogPage struct {
	List    []CourseSummary `json:"list"`
	Page    int             `json:"page"`
	Size    int             `json:"size"`
	Total   int64           `json:"total"`
	HasNext bool            `json:"hasNext"`
}

// ListCatalog 返回课程目录分页（canonical course 一页）。
func ListCatalog(q CatalogQuery) (CatalogPage, error) {
	page := q.Page
	if page <= 0 {
		page = 1
	}
	size := q.Size
	if size <= 0 {
		size = 20
	}
	if size > 50 {
		size = 50
	}
	entities, total, err := course.ListCourses(course.ListCourseQuery{
		Keyword:    Normalize(q.Keyword),
		Department: q.Department,
		TermCode:   q.TermCode,
		Campus:     q.Campus,
		Page:       page,
		Size:       size,
	})
	if err != nil {
		return CatalogPage{}, err
	}
	if len(entities) == 0 {
		return CatalogPage{List: []CourseSummary{}, Page: page, Size: size, Total: total, HasNext: false}, nil
	}
	summaries, err := buildSummaries(entities)
	if err != nil {
		return CatalogPage{}, err
	}
	return CatalogPage{List: summaries, Page: page, Size: size, Total: total, HasNext: int64(page)*int64(size) < total}, nil
}

// GetCourseDetail 返回课程详情；课程不存在或已隐藏时返回错误。
func GetCourseDetail(id uint64) (CourseDetail, error) {
	entity := course.GetCourse(id)
	if entity.Id == 0 || entity.Status != course.StatusVisible {
		return CourseDetail{}, errors.New("course not found")
	}
	detail := CourseDetail{
		Id:          entity.Id,
		PrimaryCode: entity.PrimaryCode,
		Name:        entity.Name,
		Department:  entity.Department,
		CreditX10:   entity.CreditX10,
		Aliases:     []string{},
		Offerings:   []OfferingSummary{},
	}
	aliases, err := course.ListAliasesByCourse(entity.Id)
	if err != nil {
		return CourseDetail{}, err
	}
	for _, a := range aliases {
		detail.Aliases = append(detail.Aliases, a.Value)
	}
	offerings, err := course.ListOfferingsByCourse(entity.Id)
	if err != nil {
		return CourseDetail{}, err
	}
	if len(offerings) == 0 {
		return detail, nil
	}
	offeringIds := make([]uint64, 0, len(offerings))
	for _, o := range offerings {
		offeringIds = append(offeringIds, o.Id)
	}
	links, err := course.ListOfferingInstructorLinks(offeringIds)
	if err != nil {
		return CourseDetail{}, err
	}
	instructors, err := course.ListInstructorsByOfferings(offeringIds)
	if err != nil {
		return CourseDetail{}, err
	}
	instructorByID := make(map[uint64]string, len(instructors))
	for _, ins := range instructors {
		instructorByID[ins.Id] = ins.Name
	}
	termIds := make([]uint64, 0, len(offerings))
	for _, o := range offerings {
		termIds = append(termIds, o.TermId)
	}
	terms, err := course.ListTermsByIDs(termIds)
	if err != nil {
		return CourseDetail{}, err
	}
	termByID := make(map[uint64]course.TermEntity, len(terms))
	for _, t := range terms {
		termByID[t.Id] = t
	}
	instructorsByOffering := make(map[uint64][]string)
	for _, link := range links {
		if name, ok := instructorByID[link.InstructorId]; ok {
			instructorsByOffering[link.OfferingId] = append(instructorsByOffering[link.OfferingId], name)
		}
	}
	for _, o := range offerings {
		os := OfferingSummary{
			Id:          o.Id,
			Campus:      o.Campus,
			Faculty:     o.Faculty,
			Instructors: instructorsByOffering[o.Id],
		}
		if t, ok := termByID[o.TermId]; ok {
			os.TermCode = t.Code
			os.TermName = t.Name
		}
		detail.Offerings = append(detail.Offerings, os)
	}
	return detail, nil
}

func buildSummaries(entities []course.Entity) ([]CourseSummary, error) {
	courseIds := make([]uint64, 0, len(entities))
	for _, e := range entities {
		courseIds = append(courseIds, e.Id)
	}
	aliases, err := course.ListAliasesByCourses(courseIds)
	if err != nil {
		return nil, err
	}
	aliasesByCourse := make(map[uint64][]string)
	for _, a := range aliases {
		aliasesByCourse[a.CourseId] = append(aliasesByCourse[a.CourseId], a.Value)
	}
	offerings, err := course.ListOfferingsByCourses(courseIds)
	if err != nil {
		return nil, err
	}
	offeringIds := make([]uint64, 0, len(offerings))
	offeringsByCourse := make(map[uint64][]uint64)
	for _, o := range offerings {
		offeringIds = append(offeringIds, o.Id)
		offeringsByCourse[o.CourseId] = append(offeringsByCourse[o.CourseId], o.Id)
	}
	links, err := course.ListOfferingInstructorLinks(offeringIds)
	if err != nil {
		return nil, err
	}
	instructors, err := course.ListInstructorsByOfferings(offeringIds)
	if err != nil {
		return nil, err
	}
	instructorByID := make(map[uint64]string, len(instructors))
	for _, ins := range instructors {
		instructorByID[ins.Id] = ins.Name
	}
	instructorsByOffering := make(map[uint64][]string)
	for _, link := range links {
		if name, ok := instructorByID[link.InstructorId]; ok {
			instructorsByOffering[link.OfferingId] = append(instructorsByOffering[link.OfferingId], name)
		}
	}
	termIds := make([]uint64, 0, len(offerings))
	for _, o := range offerings {
		termIds = append(termIds, o.TermId)
	}
	terms, err := course.ListTermsByIDs(termIds)
	if err != nil {
		return nil, err
	}
	offeringByID := make(map[uint64]course.OfferingEntity, len(offerings))
	termByID := make(map[uint64]course.TermEntity, len(terms))
	for _, o := range offerings {
		offeringByID[o.Id] = o
	}
	for _, t := range terms {
		termByID[t.Id] = t
	}
	summaries := make([]CourseSummary, 0, len(entities))
	for _, e := range entities {
		s := CourseSummary{
			Id:          e.Id,
			PrimaryCode: e.PrimaryCode,
			Name:        e.Name,
			Department:  e.Department,
			CreditX10:   e.CreditX10,
			Aliases:     aliasesByCourse[e.Id],
		}
		seen := make(map[string]struct{})
		seenTerms := make(map[string]struct{})
		for _, oid := range offeringsByCourse[e.Id] {
			for _, name := range instructorsByOffering[oid] {
				if _, ok := seen[name]; !ok {
					seen[name] = struct{}{}
					s.Instructors = append(s.Instructors, name)
				}
			}
			if o, ok := offeringByID[oid]; ok {
				if t, ok := termByID[o.TermId]; ok {
					if _, ok := seenTerms[t.Code]; !ok {
						seenTerms[t.Code] = struct{}{}
						s.RecentTerms = append(s.RecentTerms, t.Code)
					}
				}
			}
		}
		summaries = append(summaries, s)
	}
	return summaries, nil
}
