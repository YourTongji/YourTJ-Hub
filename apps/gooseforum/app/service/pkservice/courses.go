package pkservice

import (
	"sort"
	"strconv"
	"strings"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pk"
)

// TeacherRefItem 教师引用（code + name，JSON 输出用）。
type TeacherRefItem struct {
	TeacherCode string `json:"teacherCode"`
	TeacherName string `json:"teacherName"`
}

// CourseClassItem 一个教学班（P5/P8/P12）。
type CourseClassItem struct {
	Code             string            `json:"code"`
	Teachers         []TeacherRefItem  `json:"teachers"`
	Campus           string            `json:"campus"`
	TeachingLanguage string            `json:"teachingLanguage"`
	ArrangementInfo  []ArrangementInfo `json:"arrangementInfo"`
	IsExclusive      bool              `json:"isExclusive"`
}

// CourseByMajorItem P5 courses-by-major 输出项。
type CourseByMajorItem struct {
	CourseCode   string            `json:"courseCode"`
	CourseName   string            `json:"courseName"`
	Faculty      string            `json:"faculty"`
	FacultyI18n  string            `json:"facultyI18n"`
	Credit       float64           `json:"credit"`
	Grade        int               `json:"grade"`
	CourseNature []string          `json:"courseNature"`
	Courses      []CourseClassItem `json:"courses"`
}

// FindCoursesByMajor P5：按专业查计划内课程（含更早年级），教学班按 classCode 合并。
func FindCoursesByMajor(grade int, code string, calendarId int) ([]CourseByMajorItem, error) {
	targetMajorId, err := pk.GetTargetMajorId(code, grade, calendarId)
	if err != nil {
		return nil, err
	}
	exclusiveSet := map[uint64]struct{}{}
	if targetMajorId != 0 {
		ids, err := pk.ListMajorCourseIds(targetMajorId)
		if err != nil {
			return nil, err
		}
		for _, id := range ids {
			exclusiveSet[id] = struct{}{}
		}
	}

	// 第一查询：该专业的计划内课程（含更早年级），用于确定课程分组与性质。
	rows, err := pk.ListMajorCourseRows(calendarId, code, grade)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return []CourseByMajorItem{}, nil
	}

	groupKeys := []string{}
	groupMeta := map[string]struct {
		CourseCode string
		Grade      int
	}{}
	courseCodeToGroupKeys := map[string][]string{}
	groupToNature := map[string][]string{}
	groupExclusive := map[string]map[uint64]struct{}{}
	for _, row := range rows {
		courseCode := normalizeText(row.CourseCode)
		if courseCode == "" {
			continue
		}
		rowGrade := row.Grade
		if rowGrade == 0 {
			rowGrade = grade
		}
		groupKey := courseCode + "__" + strconv.Itoa(rowGrade)
		if _, ok := groupMeta[groupKey]; !ok {
			groupKeys = append(groupKeys, groupKey)
			groupMeta[groupKey] = struct {
				CourseCode string
				Grade      int
			}{courseCode, rowGrade}
		}
		courseCodeToGroupKeys[courseCode] = append(courseCodeToGroupKeys[courseCode], groupKey)
		if label := normalizeText(row.CourseLabelName); label != "" {
			groupToNature[groupKey] = append(groupToNature[groupKey], label)
		}
		if _, ok := exclusiveSet[row.Id]; ok {
			if groupExclusive[groupKey] == nil {
				groupExclusive[groupKey] = map[uint64]struct{}{}
			}
			groupExclusive[groupKey][row.Id] = struct{}{}
		}
	}

	allCourseCodes := make([]string, 0, len(groupMeta))
	for _, key := range groupKeys {
		allCourseCodes = append(allCourseCodes, groupMeta[key].CourseCode)
	}
	allCourseCodes = uniqueText(allCourseCodes)

	// 第二查询：这些 courseCode 的全部教学班（不限专业）。
	allRows, err := pk.ListAllCourseDetailRowsByCodes(calendarId, allCourseCodes)
	if err != nil {
		return nil, err
	}
	rowsByCourseCode := map[string][]pk.MajorCourseRow{}
	for _, row := range allRows {
		courseCode := normalizeText(row.CourseCode)
		if courseCode == "" {
			continue
		}
		rowsByCourseCode[courseCode] = append(rowsByCourseCode[courseCode], row)
		// 上游第二查询会把该 courseCode 全部教学班的性质并入课程组；
		// 只从专业范围取性质会漏掉其他专业班级带上的标签。
		if label := normalizeText(row.CourseLabelName); label != "" {
			for _, key := range courseCodeToGroupKeys[courseCode] {
				groupToNature[key] = append(groupToNature[key], label)
			}
		}
	}

	teachersByClass, err := loadTeacherRows(allRows)
	if err != nil {
		return nil, err
	}

	output := make([]CourseByMajorItem, 0, len(groupKeys))
	for _, groupKey := range groupKeys {
		meta := groupMeta[groupKey]
		courseRows := rowsByCourseCode[meta.CourseCode]
		if len(courseRows) == 0 {
			continue
		}
		first := courseRows[0]
		item := CourseByMajorItem{
			CourseCode:   meta.CourseCode,
			CourseName:   first.CourseName,
			Faculty:      first.FacultyI18n,
			FacultyI18n:  first.FacultyI18n,
			Credit:       first.Credit,
			Grade:        meta.Grade,
			CourseNature: uniqueText(groupToNature[groupKey]),
			Courses:      buildClassItems(courseRows, teachersByClass, groupExclusive[groupKey]),
		}
		sort.SliceStable(item.Courses, func(i, j int) bool {
			return strings.Compare(item.Courses[i].Code, item.Courses[j].Code) < 0
		})
		output = append(output, item)
	}
	return output, nil
}

// CourseDetailBriefItem P8/P12 教学班明细输出项（isExclusive 仅 P12 的 major 课程带，指针显式区分 true/false）。
type CourseDetailBriefItem struct {
	Code             string            `json:"code"`
	Teachers         []TeacherRefItem  `json:"teachers"`
	Campus           string            `json:"campus"`
	TeachingLanguage string            `json:"teachingLanguage"`
	ArrangementInfo  []ArrangementInfo `json:"arrangementInfo"`
	IsExclusive      *bool             `json:"isExclusive,omitempty"`
}

// FindCourseDetailsByCodes P8：批量按 courseCode 查课程详情字典。
func FindCourseDetailsByCodes(calendarId int, courseCodes []string) (map[string][]CourseDetailBriefItem, error) {
	codes := normalizeCodes(courseCodes)
	out := make(map[string][]CourseDetailBriefItem, len(codes))
	for _, code := range codes {
		out[code] = []CourseDetailBriefItem{}
	}
	if len(codes) == 0 {
		return out, nil
	}
	rows, err := pk.ListCourseDetailRowsByCodes(calendarId, codes)
	if err != nil {
		return nil, err
	}
	teachersByClass, err := loadTeacherRows(rows)
	if err != nil {
		return nil, err
	}
	byCourseCode := map[string][]pk.CourseDetailRow{}
	for _, row := range rows {
		cc := normalizeText(row.CourseCode)
		if cc == "" {
			continue
		}
		byCourseCode[cc] = append(byCourseCode[cc], row)
	}
	for code, courseRows := range byCourseCode {
		items := make([]CourseDetailBriefItem, 0, len(courseRows))
		for _, row := range courseRows {
			teachers := teachersByClass[row.Id]
			items = append(items, CourseDetailBriefItem{
				Code:             row.Code,
				Teachers:         teacherRefs(teachers),
				Campus:           row.CampusI18n,
				TeachingLanguage: row.TeachingLanguageI18n,
				ArrangementInfo:  mergeArrangementInfo(teachers),
			})
		}
		out[code] = items
	}
	return out, nil
}

// SearchCourseParams P9 高级检索入参。
type SearchCourseParams struct {
	CalendarId  int
	CourseName  string
	CourseCode  string
	TeacherCode string
	TeacherName string
	Campus      string
	Faculty     string
}

// SearchCourseItem P9 搜索结果项。
type SearchCourseItem struct {
	CourseCode   string   `json:"courseCode"`
	CourseName   string   `json:"courseName"`
	Faculty      string   `json:"faculty"`
	FacultyI18n  string   `json:"facultyI18n"`
	CourseNature []string `json:"courseNature"`
	Campus       []string `json:"campus"`
	CampusList   []string `json:"campus_list"`
	Credit       float64  `json:"credit"`
}

// SearchResult P9 输出。
type SearchResult struct {
	Courses   []SearchCourseItem `json:"courses"`
	SizeLimit int                `json:"sizeLimit"`
}

// SearchCourses P9：高级检索，LIMIT 100，按 courseCode 聚合（多教师/多校区去重）。
func SearchCourses(p SearchCourseParams) (SearchResult, error) {
	rows, err := pk.SearchCourseRows(pk.CourseSearchQuery{
		CalendarId:  p.CalendarId,
		CourseName:  normalizeText(p.CourseName),
		CourseCode:  normalizeText(p.CourseCode),
		TeacherCode: normalizeText(p.TeacherCode),
		TeacherName: normalizeText(p.TeacherName),
		Campus:      normalizeText(p.Campus),
		Faculty:     normalizeText(p.Faculty),
		SizeLimit:   100,
	})
	if err != nil {
		return SearchResult{}, err
	}
	return SearchResult{Courses: aggregateSearchCourses(rows), SizeLimit: 100}, nil
}

// MajorInfo P12 已选专业信息（用于 isExclusive）。
type MajorInfo struct {
	Grade int    `json:"grade"`
	Code  string `json:"code"`
}

// SyncCourseInfoParams P12 增量刷新已选课程入参。
type SyncCourseInfoParams struct {
	CalendarId       int
	MajorCourseCodes []string
	OtherCourseCodes []string
	MajorInfo        *MajorInfo
}

// SyncCourseInfo P12：刷新已选课程详情（isExclusive 仅 major 课程带标记）。
func SyncCourseInfo(p SyncCourseInfoParams) (map[string][]CourseDetailBriefItem, error) {
	allCodes := append([]string{}, p.MajorCourseCodes...)
	allCodes = append(allCodes, p.OtherCourseCodes...)
	codes := normalizeCodes(allCodes)
	out := make(map[string][]CourseDetailBriefItem, len(codes))
	for _, code := range codes {
		out[code] = []CourseDetailBriefItem{}
	}
	if len(codes) == 0 {
		return out, nil
	}
	rows, err := pk.ListCourseDetailRowsByCodes(p.CalendarId, codes)
	if err != nil {
		return nil, err
	}
	teachersByClass, err := loadTeacherRows(rows)
	if err != nil {
		return nil, err
	}

	// 目标专业集合：majorInfo（grade+code）→ 该专业 courseId 集合。
	// 用 grade <= 找最近专业（对齐上游 getLatestCourseInfo），不限学期。
	exclusiveSet := map[uint64]struct{}{}
	if p.MajorInfo != nil && p.MajorInfo.Code != "" && p.MajorInfo.Grade > 0 {
		majorId, err := pk.GetNearestMajorId(p.MajorInfo.Code, p.MajorInfo.Grade)
		if err != nil {
			return nil, err
		}
		if majorId != 0 {
			ids, err := pk.ListMajorCourseIds(majorId)
			if err != nil {
				return nil, err
			}
			for _, id := range ids {
				exclusiveSet[id] = struct{}{}
			}
		}
	}
	majorSet := make(map[string]struct{}, len(p.MajorCourseCodes))
	for _, code := range p.MajorCourseCodes {
		majorSet[normalizeText(code)] = struct{}{}
	}

	byCourseCode := map[string][]pk.CourseDetailRow{}
	for _, row := range rows {
		cc := normalizeText(row.CourseCode)
		if cc == "" {
			continue
		}
		byCourseCode[cc] = append(byCourseCode[cc], row)
	}
	for code, courseRows := range byCourseCode {
		_, isMajor := majorSet[code]
		items := make([]CourseDetailBriefItem, 0, len(courseRows))
		for _, row := range courseRows {
			teachers := teachersByClass[row.Id]
			item := CourseDetailBriefItem{
				Code:             row.Code,
				Teachers:         teacherRefs(teachers),
				Campus:           row.CampusI18n,
				TeachingLanguage: row.TeachingLanguageI18n,
				ArrangementInfo:  mergeArrangementInfo(teachers),
			}
			if isMajor {
				_, exclusive := exclusiveSet[row.Id]
				excl := exclusive
				item.IsExclusive = &excl
			}
			items = append(items, item)
		}
		out[code] = items
	}
	return out, nil
}

// ---- 共享辅助 ----

func normalizeCodes(codes []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(codes))
	for _, code := range codes {
		c := normalizeText(code)
		if c == "" {
			continue
		}
		if _, ok := seen[c]; ok {
			continue
		}
		seen[c] = struct{}{}
		out = append(out, c)
	}
	return out
}

// loadTeacherRows 从明细行集合提取教学班 id 并批量查教师（保留 arrangeInfoText）。
func loadTeacherRows(rows any) (map[uint64][]teacherRow, error) {
	ids := []uint64{}
	switch v := rows.(type) {
	case []pk.MajorCourseRow:
		for _, row := range v {
			ids = append(ids, row.Id)
		}
	case []pk.CourseDetailRow:
		for _, row := range v {
			ids = append(ids, row.Id)
		}
	}
	teachers, err := pk.ListTeachersByClassIds(ids)
	if err != nil {
		return nil, err
	}
	out := map[uint64][]teacherRow{}
	for _, t := range teachers {
		out[t.TeachingClassId] = append(out[t.TeachingClassId], teacherRow{
			TeachingClassId: t.TeachingClassId,
			TeacherCode:     t.TeacherCode,
			TeacherName:     t.TeacherName,
			ArrangeInfoText: t.ArrangeInfoText,
		})
	}
	return out, nil
}

func teacherRefs(rows []teacherRow) []TeacherRefItem {
	out := make([]TeacherRefItem, 0, len(rows))
	for _, r := range rows {
		out = append(out, TeacherRefItem{TeacherCode: r.TeacherCode, TeacherName: r.TeacherName})
	}
	return out
}

// buildClassItems 把某课程的教学班明细合并为 CourseClassItem（按 classCode 去重，合并安排）。
// 同一 classCode 跨校区/专业出现时，Teachers 取首个班级，arrangementInfo 合并全部班级的
// 教师安排文本（含首个班级的 schedule），与上游 courseMap 合并语义一致。
func buildClassItems(courseRows []pk.MajorCourseRow, teachersByClass map[uint64][]teacherRow, exclusive map[uint64]struct{}) []CourseClassItem {
	courseMap := map[string]*CourseClassItem{}
	rowAccum := map[string][]teacherRow{}
	order := []string{}
	for _, row := range courseRows {
		classCode := normalizeText(row.Code)
		if classCode == "" {
			continue
		}
		teachers := teachersByClass[row.Id]
		rowAccum[classCode] = append(rowAccum[classCode], teachers...)
		if _, ok := courseMap[classCode]; ok {
			continue
		}
		_, excl := exclusive[row.Id]
		courseMap[classCode] = &CourseClassItem{
			Code:             classCode,
			Teachers:         teacherRefs(teachers),
			Campus:           row.CampusI18n,
			TeachingLanguage: row.TeachingLanguageI18n,
			ArrangementInfo:  mergeArrangementInfo(teachers),
			IsExclusive:      excl,
		}
		order = append(order, classCode)
	}
	items := make([]CourseClassItem, 0, len(order))
	for _, classCode := range order {
		item := courseMap[classCode]
		if rows := rowAccum[classCode]; len(rows) > 0 {
			item.ArrangementInfo = mergeArrangementInfo(rows)
		}
		items = append(items, *item)
	}
	return items
}

// aggregateSearchCourses 把检索明细行按 courseCode 聚合（credit 取 MAX、nature/campus 去重）。
func aggregateSearchCourses(rows []pk.CourseAggRow) []SearchCourseItem {
	byCode := map[string]*SearchCourseItem{}
	order := []string{}
	for _, row := range rows {
		cc := normalizeText(row.CourseCode)
		if cc == "" {
			continue
		}
		item, ok := byCode[cc]
		if !ok {
			item = &SearchCourseItem{
				CourseCode:  cc,
				CourseName:  row.CourseName,
				Faculty:     row.FacultyI18n,
				FacultyI18n: row.FacultyI18n,
				Credit:      row.Credit,
			}
			byCode[cc] = item
			order = append(order, cc)
		}
		if row.Credit > item.Credit {
			item.Credit = row.Credit
		}
		if label := normalizeText(row.CourseLabelName); label != "" {
			item.CourseNature = uniqueText(append(item.CourseNature, label))
		}
		if campus := normalizeText(row.CampusI18n); campus != "" {
			item.Campus = uniqueText(append(item.Campus, campus))
		}
	}
	items := make([]SearchCourseItem, 0, len(order))
	for _, cc := range order {
		it := byCode[cc]
		it.CampusList = it.Campus
		items = append(items, *it)
	}
	return items
}
