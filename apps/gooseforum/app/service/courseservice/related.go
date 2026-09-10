package courseservice

import (
	"cmp"
	"slices"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/course"
)

// RelatedListLimit 每个相关课程列表的条目上限（同教师其他课 / 同课程其他教师各前 5）。
const RelatedListLimit = 5

// RelatedCourseItem 相关课程条目（course 维度，可跳转课程详情页）。
// (code, teacher) 复合身份模型下同课号不同教师是独立课程卡：同教师其他课与
// 同课程其他教师两个区块都返回该结构；TeacherName 为卡片身份教师（无教师时省略）。
type RelatedCourseItem struct {
	Id          uint64   `json:"id"`
	PrimaryCode string   `json:"primaryCode"`
	Name        string   `json:"name"`
	Department  string   `json:"department"`
	TeacherName string   `json:"teacherName,omitempty"`
	Instructors []string `json:"instructors,omitempty"`
	RatingAvg   float64  `json:"ratingAvg"`
	RatingCount int      `json:"ratingCount"`
	ReviewCount int      `json:"reviewCount"`
}

// RelationItem 沿革区块条目：本卡相关的已确认沿革关系（原名标注 + 关系类型）。
// Direction 表示方向：to（旧卡 → 本卡，本卡为当前卡）/ from（本卡 → 新卡，本卡为历史卡）。
type RelationItem struct {
	RelationId   uint64 `json:"relationId"`
	FromCourseId uint64 `json:"fromCourseId"`
	FromName     string `json:"fromName"`
	ToCourseId   uint64 `json:"toCourseId"`
	ToName       string `json:"toName"`
	RelationType string `json:"relationType"`
	Status       string `json:"status"`
	Direction    string `json:"direction"`
}

// CourseRelated 课程详情页相关课程区块数据。
// Lineage 为本卡已确认的沿革关系（approve/merged；原名标注与跳转旧卡）。
type CourseRelated struct {
	TeacherOtherCourses     []RelatedCourseItem `json:"teacherOtherCourses"`
	SameCourseOtherTeachers []RelatedCourseItem `json:"sameCourseOtherTeachers"`
	Lineage                 []RelationItem      `json:"lineage"`
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
		SameCourseOtherTeachers: []RelatedCourseItem{},
		Lineage:                 []RelationItem{},
	}
	// 沿革区块：本卡相关的已确认关系（approved/merged），含方向与原名。
	result.Lineage = buildLineageItems(entity)
	instructorIds, err := course.ListInstructorIDsByCourse(courseId)
	if err != nil {
		return CourseRelated{}, err
	}
	otherIds, err := course.ListOtherCourseIDsByInstructors(instructorIds, courseId)
	if err != nil {
		return CourseRelated{}, err
	}
	// 同课号卡（sameCourseOtherTeachers 区块负责）不重复出现在同教师区块。
	if len(otherIds) > 0 {
		byID := course.GetMapByIds(otherIds)
		distinct := otherIds[:0]
		for _, id := range otherIds {
			if c, ok := byID[id]; ok && c.PrimaryCode != entity.PrimaryCode {
				distinct = append(distinct, id)
			}
		}
		if len(distinct) > 0 {
			result.TeacherOtherCourses, err = buildRelatedTeacherCourses(distinct)
			if err != nil {
				return CourseRelated{}, err
			}
		}
	}
	result.SameCourseOtherTeachers, err = buildSameCourseOtherTeachers(entity)
	if err != nil {
		return CourseRelated{}, err
	}
	return result, nil
}

// buildLineageItems 返回本卡已确认的沿革关系条目（原名标注 + 方向）：
//   - direction=to：关系指向本卡（本卡为当前卡，from 为历史旧卡）；
//   - direction=from：本卡指向新卡（本卡为历史卡，合并后隐藏但仍可被沿革区块引用）。
//
// 状态仅 approved/merged（pending/ignored 不展示）；原名标注用 relations 快照 + from 卡当前名。
func buildLineageItems(entity course.Entity) []RelationItem {
	items := []RelationItem{}
	// 指向本卡（当前卡视角）。
	if rels, err := course.ListRelationsByToCourse(entity.Id, []string{
		string(course.RelationStatusApproved),
		string(course.RelationStatusMerged),
	}); err == nil {
		var fromIds []uint64
		for _, r := range rels {
			fromIds = append(fromIds, r.FromCourseId)
		}
		nameByID := make(map[uint64]string)
		for _, c := range course.GetMapByIds(fromIds) {
			nameByID[c.Id] = c.Name
		}
		for _, r := range rels {
			items = append(items, RelationItem{
				RelationId:   r.Id,
				FromCourseId: r.FromCourseId,
				FromName:     nameByID[r.FromCourseId],
				ToCourseId:   r.ToCourseId,
				ToName:       entity.Name,
				RelationType: r.RelationType,
				Status:       r.Status,
				Direction:    "to",
			})
		}
	}
	// 本卡出发（历史卡视角）。
	if rels, err := course.ListRelationsByFromCourse(entity.Id); err == nil {
		var toIds []uint64
		for _, r := range rels {
			toIds = append(toIds, r.ToCourseId)
		}
		nameByID := make(map[uint64]string)
		for _, c := range course.GetMapByIds(toIds) {
			nameByID[c.Id] = c.Name
		}
		for _, r := range rels {
			if r.Status != string(course.RelationStatusApproved) && r.Status != string(course.RelationStatusMerged) {
				continue
			}
			items = append(items, RelationItem{
				RelationId:   r.Id,
				FromCourseId: r.FromCourseId,
				FromName:     entity.Name,
				ToCourseId:   r.ToCourseId,
				ToName:       nameByID[r.ToCourseId],
				RelationType: r.RelationType,
				Status:       r.Status,
				Direction:    "from",
			})
		}
	}
	return items
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
	// 卡片身份教师：按 teacher_id 批量解析姓名（无教师卡保持空）。
	teacherIds := make([]uint64, 0, len(courseIds))
	for _, id := range courseIds {
		if c, ok := courseById[id]; ok && c.TeacherId != 0 {
			teacherIds = append(teacherIds, c.TeacherId)
		}
	}
	teacherNameByID := make(map[uint64]string)
	if len(teacherIds) > 0 {
		teachers, err := course.ListInstructorsByIDs(teacherIds)
		if err != nil {
			return nil, err
		}
		for _, t := range teachers {
			teacherNameByID[t.Id] = t.Name
		}
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
			TeacherName: teacherNameByID[c.TeacherId],
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

// sortRelatedItems 排序：review_count 降序 → 平均分降序 → id 降序（末级 id 唯一，结果确定）。
func sortRelatedItems(items []RelatedCourseItem) {
	slices.SortFunc(items, func(a, b RelatedCourseItem) int {
		if a.ReviewCount != b.ReviewCount {
			return cmp.Compare(b.ReviewCount, a.ReviewCount)
		}
		if a.RatingAvg != b.RatingAvg {
			if a.RatingAvg > b.RatingAvg {
				return -1
			}
			if a.RatingAvg < b.RatingAvg {
				return 1
			}
			return 0
		}
		return cmp.Compare(b.Id, a.Id)
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

// buildSameCourseOtherTeachers 同课程（同一 primary_code）其他教师卡。
// (code, teacher) 复合身份模型下同课号每行 = 一张独立课程卡（含无教师行），
// 该区块直接返回同课号的其他可见课程行，按统计排序取前 RelatedListLimit 条。
func buildSameCourseOtherTeachers(entity course.Entity) ([]RelatedCourseItem, error) {
	others, err := course.ListOtherCoursesByPrimaryCode(entity.PrimaryCode, entity.Id)
	if err != nil {
		return nil, err
	}
	if len(others) == 0 {
		return []RelatedCourseItem{}, nil
	}
	ids := make([]uint64, 0, len(others))
	teacherIds := make([]uint64, 0, len(others))
	for _, o := range others {
		ids = append(ids, o.Id)
		if o.TeacherId != 0 {
			teacherIds = append(teacherIds, o.TeacherId)
		}
	}
	// 卡片身份教师：按 teacher_id 批量解析姓名（无教师卡保持空）。
	teacherNameByID := make(map[uint64]string)
	if len(teacherIds) > 0 {
		teachers, err := course.ListInstructorsByIDs(teacherIds)
		if err != nil {
			return nil, err
		}
		for _, t := range teachers {
			teacherNameByID[t.Id] = t.Name
		}
	}
	stats, err := course.GetCourseStatsMap(ids)
	if err != nil {
		return nil, err
	}
	items := make([]RelatedCourseItem, 0, len(others))
	for _, o := range others {
		st := stats[o.Id]
		items = append(items, RelatedCourseItem{
			Id:          o.Id,
			PrimaryCode: o.PrimaryCode,
			Name:        o.Name,
			Department:  o.Department,
			TeacherName: teacherNameByID[o.TeacherId],
			RatingAvg:   ratingAvgFromStats(st.RatingSum, st.RatingCount),
			RatingCount: st.RatingCount,
			ReviewCount: st.ReviewCount,
		})
	}
	sortRelatedItems(items)
	if len(items) > RelatedListLimit {
		items = items[:RelatedListLimit]
	}
	return items, nil
}
