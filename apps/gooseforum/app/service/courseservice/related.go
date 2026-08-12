package courseservice

import (
	"sort"
	"strconv"
	"strings"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/course"
)

// RelatedListLimit 每个相关课程列表的条目上限（同教师其他课 / 同课程其他教师各前 5）。
const RelatedListLimit = 5

// RelatedCourseItem 同教师其他课程条目（course 维度，可跳转课程详情页）。
type RelatedCourseItem struct {
	Id          uint64   `json:"id"`
	PrimaryCode string   `json:"primaryCode"`
	Name        string   `json:"name"`
	Department  string   `json:"department"`
	Instructors []string `json:"instructors,omitempty"`
	RatingAvg   float64  `json:"ratingAvg"`
	RatingCount int      `json:"ratingCount"`
	ReviewCount int      `json:"reviewCount"`
}

// RelatedTeacherOfferingItem 同课程其他教师条目（offering 维度）。
// Hub 中一门 canonical course 唯一对应一个 primary_code，"同课程其他教师"指该课程
// 与"最近学期开课教师组合"不同的其他开课实例；评分统计按 offering 聚合。
type RelatedTeacherOfferingItem struct {
	OfferingId  uint64   `json:"offeringId"`
	TermCode    string   `json:"termCode"`
	TermName    string   `json:"termName,omitempty"`
	Campus      string   `json:"campus,omitempty"`
	Instructors []string `json:"instructors,omitempty"`
	RatingAvg   float64  `json:"ratingAvg"`
	RatingCount int      `json:"ratingCount"`
	ReviewCount int      `json:"reviewCount"`
}

// CourseRelated 课程详情页相关课程区块数据。
type CourseRelated struct {
	TeacherOtherCourses     []RelatedCourseItem          `json:"teacherOtherCourses"`
	SameCourseOtherTeachers []RelatedTeacherOfferingItem `json:"sameCourseOtherTeachers"`
}

// GetCourseRelated 返回课程的"同教师其他课"与"同课程其他教师"（各前 RelatedListLimit 条）。
// 课程不存在或已隐藏时返回 ErrCourseNotFound（与 CourseDetailJSON 一致的 404 语义）。
func GetCourseRelated(courseId uint64) (CourseRelated, error) {
	entity := course.GetCourse(courseId)
	if entity.Id == 0 || entity.Status != course.StatusVisible {
		return CourseRelated{}, ErrCourseNotFound
	}
	result := CourseRelated{
		TeacherOtherCourses:     []RelatedCourseItem{},
		SameCourseOtherTeachers: []RelatedTeacherOfferingItem{},
	}
	instructorIds, err := course.ListInstructorIDsByCourse(courseId)
	if err != nil {
		return CourseRelated{}, err
	}
	otherIds, err := course.ListOtherCourseIDsByInstructors(instructorIds, courseId)
	if err != nil {
		return CourseRelated{}, err
	}
	if len(otherIds) > 0 {
		result.TeacherOtherCourses, err = buildRelatedTeacherCourses(otherIds)
		if err != nil {
			return CourseRelated{}, err
		}
	}
	result.SameCourseOtherTeachers, err = buildSameCourseOtherTeachers(courseId)
	if err != nil {
		return CourseRelated{}, err
	}
	return result, nil
}

// ratingAvgFromStats 由 rating_sum/rating_count 计算平均分（无评分时为 0）。
func ratingAvgFromStats(sum, count int) float64 {
	if count <= 0 {
		return 0
	}
	return float64(sum) / float64(count)
}

// buildRelatedTeacherCourses 将候选课程按统计排序（review_count 降序、平均分降序、id 降序）
// 取前 RelatedListLimit 条，并批量回填每门课的教师名单（去重）。
func buildRelatedTeacherCourses(courseIds []uint64) ([]RelatedCourseItem, error) {
	courseById := course.GetMapByIds(courseIds)
	stats, err := course.GetCourseStatsMap(courseIds)
	if err != nil {
		return nil, err
	}
	items := make([]RelatedCourseItem, 0, len(courseIds))
	for _, id := range courseIds {
		c, ok := courseById[id]
		if !ok {
			continue
		}
		st := stats[id]
		items = append(items, RelatedCourseItem{
			Id:          c.Id,
			PrimaryCode: c.PrimaryCode,
			Name:        c.Name,
			Department:  c.Department,
			RatingAvg:   ratingAvgFromStats(st.RatingSum, st.RatingCount),
			RatingCount: st.RatingCount,
			ReviewCount: st.ReviewCount,
		})
	}
	sortRelatedItems(items)
	if len(items) > RelatedListLimit {
		items = items[:RelatedListLimit]
	}
	ids := make([]uint64, 0, len(items))
	for _, it := range items {
		ids = append(ids, it.Id)
	}
	instructors, err := instructorsByCourses(ids)
	if err != nil {
		return nil, err
	}
	for i := range items {
		items[i].Instructors = instructors[items[i].Id]
	}
	return items, nil
}

// sortRelatedItems 稳定排序：review_count 降序 → 平均分降序 → id 降序。
func sortRelatedItems(items []RelatedCourseItem) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].ReviewCount != items[j].ReviewCount {
			return items[i].ReviewCount > items[j].ReviewCount
		}
		if items[i].RatingAvg != items[j].RatingAvg {
			return items[i].RatingAvg > items[j].RatingAvg
		}
		return items[i].Id > items[j].Id
	})
}

// instructorsByCourses 批量返回课程的去重教师名单（相关课程卡片展示用，避免 N+1）。
func instructorsByCourses(courseIds []uint64) (map[uint64][]string, error) {
	result := make(map[uint64][]string, len(courseIds))
	if len(courseIds) == 0 {
		return result, nil
	}
	offerings, err := course.ListOfferingsByCourses(courseIds)
	if err != nil {
		return nil, err
	}
	offeringIds := make([]uint64, 0, len(offerings))
	courseByOffering := make(map[uint64]uint64, len(offerings))
	for _, o := range offerings {
		offeringIds = append(offeringIds, o.Id)
		courseByOffering[o.Id] = o.CourseId
	}
	links, err := course.ListOfferingInstructorLinks(offeringIds)
	if err != nil {
		return nil, err
	}
	instructorList, err := course.ListInstructorsByOfferings(offeringIds)
	if err != nil {
		return nil, err
	}
	nameByInstructor := make(map[uint64]string, len(instructorList))
	for _, ins := range instructorList {
		nameByInstructor[ins.Id] = ins.Name
	}
	seen := make(map[uint64]map[string]struct{})
	for _, l := range links {
		cid, ok := courseByOffering[l.OfferingId]
		if !ok {
			continue
		}
		name, ok := nameByInstructor[l.InstructorId]
		if !ok {
			continue
		}
		if seen[cid] == nil {
			seen[cid] = make(map[string]struct{})
		}
		if _, dup := seen[cid][name]; dup {
			continue
		}
		seen[cid][name] = struct{}{}
		result[cid] = append(result[cid], name)
	}
	return result, nil
}

// buildSameCourseOtherTeachers 同课程（同一 canonical course）其他教师开课的 offering。
// 以最近学期开课（ListOfferingsByCourse 的 term 倒序）的教师组合为"当前教师"，
// 其余教师组合各取最近一条 offering（按组合去重），按统计排序取前 RelatedListLimit 条。
func buildSameCourseOtherTeachers(courseId uint64) ([]RelatedTeacherOfferingItem, error) {
	offerings, err := course.ListOfferingsByCourse(courseId)
	if err != nil {
		return nil, err
	}
	if len(offerings) == 0 {
		return []RelatedTeacherOfferingItem{}, nil
	}
	offeringIds := make([]uint64, 0, len(offerings))
	for _, o := range offerings {
		offeringIds = append(offeringIds, o.Id)
	}
	links, err := course.ListOfferingInstructorLinks(offeringIds)
	if err != nil {
		return nil, err
	}
	instructorList, err := course.ListInstructorsByOfferings(offeringIds)
	if err != nil {
		return nil, err
	}
	nameByInstructor := make(map[uint64]string, len(instructorList))
	for _, ins := range instructorList {
		nameByInstructor[ins.Id] = ins.Name
	}
	instructorIDsByOffering := make(map[uint64][]uint64, len(offerings))
	for _, l := range links {
		instructorIDsByOffering[l.OfferingId] = append(instructorIDsByOffering[l.OfferingId], l.InstructorId)
	}
	// 教师组合签名：排序后的教师 ID 拼接（团队授课顺序无关）。
	signatureByOffering := make(map[uint64]string, len(offerings))
	for oid, ids := range instructorIDsByOffering {
		signatureByOffering[oid] = instructorSignature(ids)
	}
	// offerings 已按 term 倒序：首条教师组合视为"当前教师"，其余组合各取最近一条。
	seen := map[string]bool{signatureByOffering[offerings[0].Id]: true}
	chosen := make([]course.OfferingEntity, 0, len(offerings))
	for _, o := range offerings {
		sig := signatureByOffering[o.Id]
		if seen[sig] {
			continue
		}
		seen[sig] = true
		chosen = append(chosen, o)
	}
	if len(chosen) == 0 {
		return []RelatedTeacherOfferingItem{}, nil
	}
	terms, err := course.ListTermsByIDs(offeringTermIDs(offerings))
	if err != nil {
		return nil, err
	}
	termByID := make(map[uint64]course.TermEntity, len(terms))
	for _, t := range terms {
		termByID[t.Id] = t
	}
	stats, err := course.GetOfferingStatsMap(offeringIds)
	if err != nil {
		return nil, err
	}
	items := make([]RelatedTeacherOfferingItem, 0, len(chosen))
	for _, o := range chosen {
		item := RelatedTeacherOfferingItem{
			OfferingId: o.Id,
			Campus:     o.Campus,
		}
		if t, ok := termByID[o.TermId]; ok {
			item.TermCode = t.Code
			item.TermName = t.Name
		}
		for _, id := range instructorIDsByOffering[o.Id] {
			if name, ok := nameByInstructor[id]; ok {
				item.Instructors = append(item.Instructors, name)
			}
		}
		st := stats[o.Id]
		item.RatingAvg = ratingAvgFromStats(st.RatingSum, st.RatingCount)
		item.RatingCount = st.RatingCount
		item.ReviewCount = st.ReviewCount
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].ReviewCount != items[j].ReviewCount {
			return items[i].ReviewCount > items[j].ReviewCount
		}
		if items[i].RatingAvg != items[j].RatingAvg {
			return items[i].RatingAvg > items[j].RatingAvg
		}
		return items[i].OfferingId > items[j].OfferingId
	})
	if len(items) > RelatedListLimit {
		items = items[:RelatedListLimit]
	}
	return items, nil
}

// instructorSignature 生成教师 ID 组合的稳定签名（排序后以逗号拼接）。
func instructorSignature(ids []uint64) string {
	if len(ids) == 0 {
		return ""
	}
	sorted := make([]uint64, len(ids))
	copy(sorted, ids)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	parts := make([]string, 0, len(sorted))
	for _, id := range sorted {
		parts = append(parts, strconv.FormatUint(id, 10))
	}
	return strings.Join(parts, ",")
}

// offeringTermIDs 收集多个 offering 的学期 ID。
func offeringTermIDs(offerings []course.OfferingEntity) []uint64 {
	ids := make([]uint64, 0, len(offerings))
	for _, o := range offerings {
		ids = append(ids, o.TermId)
	}
	return ids
}
