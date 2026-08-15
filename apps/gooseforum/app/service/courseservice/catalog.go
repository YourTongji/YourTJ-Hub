package courseservice

import (
	"errors"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/course"
)

// ErrCourseNotFound 课程不存在或已隐藏（与存储层错误区分，控制器映射为 404）。
var ErrCourseNotFound = errors.New("course not found")

// CourseSummary 课程列表卡片（B1：携带评分聚合）。
type CourseSummary struct {
	Id          uint64   `json:"id"`
	PrimaryCode string   `json:"primaryCode"`
	Name        string   `json:"name"`
	Department  string   `json:"department"`
	CreditX10   int      `json:"creditX10"`
	Aliases     []string `json:"aliases,omitempty"`
	Instructors []string `json:"instructors,omitempty"`
	RecentTerms []string `json:"recentTerms,omitempty"`
	// RatingAvg 非 NULL rating 均分（legacy 0→NULL 不计）；无评分时 null。
	RatingAvg   *float64 `json:"ratingAvg,omitempty"`
	ReviewCount int      `json:"reviewCount,omitempty"`
}

// OfferingSummary 开课实例摘要（详情页；B1 携带 offering 级评分聚合）。
type OfferingSummary struct {
	Id          uint64   `json:"id"`
	TermCode    string   `json:"termCode"`
	TermName    string   `json:"termName,omitempty"`
	Campus      string   `json:"campus,omitempty"`
	Faculty     string   `json:"faculty,omitempty"`
	Instructors []string `json:"instructors,omitempty"`
	RatingAvg   *float64 `json:"ratingAvg,omitempty"`
	ReviewCount int      `json:"reviewCount,omitempty"`
}

// CourseDetail 课程详情页数据（B1：携带评分聚合与分布）。
type CourseDetail struct {
	Id          uint64            `json:"id"`
	PrimaryCode string            `json:"primaryCode"`
	Name        string            `json:"name"`
	Department  string            `json:"department"`
	CreditX10   int               `json:"creditX10"`
	Aliases     []string          `json:"aliases,omitempty"`
	Offerings   []OfferingSummary `json:"offerings,omitempty"`
	RatingAvg   *float64          `json:"ratingAvg,omitempty"`
	ReviewCount int               `json:"reviewCount,omitempty"`
	// RatingDistribution 1-5 星各档可见评价计数（index 0 = 1 星）；
	// 无评价时省略（security F4：定长数组 omitempty 无效，改用指针）。
	RatingDistribution *course.RatingDistribution `json:"ratingDistribution,omitempty"`
}

// ratingAvgPtrFromStats 由统计投影计算均分：无评分（ratingCount==0）时返回 nil。
// legacy 0→NULL 转换后 rating 恒 1-5，rating_sum/rating_count 均为实际评分数。
// 命名区别于 related.go 的 float64 版 ratingAvgFromStats（#198 相关课程排序用，
// 无评分返回 0；本函数用于目录/详情/搜索展示，无评分返回 nil 以省略字段）。
func ratingAvgPtrFromStats(ratingCount, ratingSum int) *float64 {
	if ratingCount <= 0 {
		return nil
	}
	avg := float64(ratingSum) / float64(ratingCount)
	return &avg
}

// TermOption 目录页学期筛选项：value 为筛选入参（course_term.code），label 为展示名称。
type TermOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// CatalogQuery 目录筛选条件。
type CatalogQuery struct {
	Keyword    string
	Department string
	TermCode   string
	Campus     string
	Instructor string
	HasReview  bool
	SortBy     string
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

// ListCatalog 返回课程目录分页（canonical course 一页，B1 携带评分聚合）。
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
		Instructor: q.Instructor,
		HasReview:  q.HasReview,
		SortBy:     q.SortBy,
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

// ListDepartments 返回课程目录可筛选的院系列表（去重、按字典序）。
func ListDepartments() ([]string, error) {
	return course.ListDistinctDepartments()
}

// ListTerms 返回课程目录可筛选的学期列表（可见课程关联、去重、按时间倒序），
// 与详情页开课列表的学期排序一致（starts_on 倒序，回退 code 字典序）。
// label 优先学期名称，回退 code。
func ListTerms() ([]TermOption, error) {
	terms, err := course.ListDistinctTerms()
	if err != nil {
		return nil, err
	}
	options := make([]TermOption, 0, len(terms))
	for _, t := range terms {
		label := t.Name
		if label == "" {
			label = t.Code
		}
		options = append(options, TermOption{Value: t.Code, Label: label})
	}
	return options, nil
}

// ListCampuses 返回课程目录可筛选的校区列表（可见课程关联、去重、非空、按字典序），
// 取 course_offering.campus 原始值，与筛选值域完全一致。
func ListCampuses() ([]string, error) {
	return course.ListDistinctCampuses()
}

// GetCourseDetail 返回课程详情；课程不存在或已隐藏时返回 ErrCourseNotFound。
func GetCourseDetail(id uint64) (CourseDetail, error) {
	entity := course.GetCourse(id)
	if entity.Id == 0 || entity.Status != course.StatusVisible {
		return CourseDetail{}, ErrCourseNotFound
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
	// 课程级统计在 offerings 为空（如全部隐藏）时也填充（security F6）。
	courseStatsMap := course.ListCourseStatsByIDs([]uint64{entity.Id})
	if stats, ok := courseStatsMap[entity.Id]; ok {
		detail.RatingAvg = ratingAvgPtrFromStats(stats.RatingCount, stats.RatingSum)
		detail.ReviewCount = stats.ReviewCount
	}
	if dist := course.GetRatingDistributionsByCourseIds([]uint64{entity.Id})[entity.Id]; dist != (course.RatingDistribution{}) {
		d := dist
		detail.RatingDistribution = &d
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
	// B1：offering 级统计（详情开课列表展示）
	offeringStats := course.ListOfferingStatsByIDs(offeringIds)
	for _, o := range offerings {
		os := OfferingSummary{
			Id:          o.Id,
			Campus:      o.Campus,
			Faculty:     o.Faculty,
			Instructors: instructorsByOffering[o.Id],
		}
		if s, ok := offeringStats[o.Id]; ok {
			os.RatingAvg = ratingAvgPtrFromStats(s.RatingCount, s.RatingSum)
			os.ReviewCount = s.ReviewCount
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
	// B1：课程级统计投影（目录列表展示均分与评论数，N+1 防护）。
	courseStats := course.ListCourseStatsByIDs(courseIds)
	for _, e := range entities {
		s := CourseSummary{
			Id:          e.Id,
			PrimaryCode: e.PrimaryCode,
			Name:        e.Name,
			Department:  e.Department,
			CreditX10:   e.CreditX10,
			Aliases:     aliasesByCourse[e.Id],
		}
		if stats, ok := courseStats[e.Id]; ok {
			s.RatingAvg = ratingAvgPtrFromStats(stats.RatingCount, stats.RatingSum)
			s.ReviewCount = stats.ReviewCount
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
