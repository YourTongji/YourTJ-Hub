package pkservice

import (
	"errors"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/course"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pk"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/courseservice"
	"gorm.io/gorm"
)

// ReviewBrief P13 course-review-brief 输出项：排课器弹窗展示的课程评价摘要。
// CourseId 为 Hub 课程目录主键（/courses/:courseId 详情页跳转用）；未匹配课评目录时为 0。
// Classes 为该课程各教学班的 offering 级课评摘要（class_code 匹配，无评价记录时为空数组）。
type ReviewBrief struct {
	CourseId    uint64             `json:"courseId"`
	CourseCode  string             `json:"courseCode"`
	CourseName  string             `json:"courseName"`
	TeacherName string             `json:"teacherName"`
	RatingAvg   *float64           `json:"ratingAvg"`
	ReviewCount int                `json:"reviewCount"`
	Classes     []ReviewBriefClass `json:"classes"`
}

// ReviewBriefClass P13 教学班级课评摘要项：按 Hub offering（class_code 匹配）聚合。
// OfferingId 供跳转 /courses/:courseId?offeringId=:offeringId 聚焦该班评价。
type ReviewBriefClass struct {
	ClassCode   string   `json:"classCode"`
	OfferingId  uint64   `json:"offeringId"`
	Teachers    []string `json:"teachers"`
	RatingAvg   *float64 `json:"ratingAvg"`
	ReviewCount int      `json:"reviewCount"`
}

func FindCourseReviewBrief(courseCode, teacherName string, calendarId uint64) (ReviewBrief, error) {
	brief := ReviewBrief{
		CourseCode:  normalizeText(courseCode),
		TeacherName: normalizeText(teacherName),
		Classes:     []ReviewBriefClass{},
	}

	row, err := pk.FindCourseDetailByCodeAnyCalendar(brief.CourseCode)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return brief, err
	}
	candidateCodes := []string{}
	if err == nil {
		brief.CourseName = row.CourseName
		if newCode := normalizeText(row.NewCourseCode); newCode != "" {
			candidateCodes = append(candidateCodes, newCode)
		}
	}
	candidateCodes = append(candidateCodes, brief.CourseCode)

	// 复合身份模型（issue #326）：同 code 不同教师是独立课程卡。
	// teacherName 非空时按 (code, teacher) 精确归因；未命中（旧数据未回填
	// teacher_id / 无教师课程 / 教师名不匹配）退回该 code 首卡（id 升序），
	// 保证排课器弹窗至少拿到一份课程评价摘要。
	for _, code := range candidateCodes {
		briefs, err := courseservice.GetCourseStatsByPrimaryCodeTeacher(code, brief.TeacherName)
		if err != nil {
			return brief, err
		}
		if len(briefs) == 0 {
			continue
		}
		if brief.CourseName == "" {
			brief.CourseName = briefs[0].Name
		}
		chosen := briefs[0]
		if brief.TeacherName != "" {
			for _, b := range briefs {
				if courseservice.Normalize(b.TeacherName) == courseservice.Normalize(brief.TeacherName) {
					chosen = b
					break
				}
			}
		}
		brief.CourseId = chosen.CourseId
		brief.RatingAvg = chosen.RatingAvg
		brief.ReviewCount = chosen.ReviewCount
		break
	}

	if err := fillClassBriefs(&brief, calendarId); err != nil {
		return brief, err
	}
	return brief, nil
}

// fillClassBriefs 填充教学班级课评摘要：PK 教学班课号 → offering.class_code 匹配，
// 聚合 offering 级统计与教师；无匹配（旧数据包班号为空）时保持空数组。
// calendarId > 0 时班级课号只在指定学期内匹配，并进一步把 offering 限定到该学期
// （calendar_id_i18n ↔ course_term.code 映射；跨学期班号复用不串学期）。
func fillClassBriefs(brief *ReviewBrief, calendarId uint64) error {
	classCodes, err := pk.ListClassCodesByCourseCode(brief.CourseCode, calendarId)
	if err != nil {
		return err
	}
	// calendarId → calendar_id_i18n（如 "2025-2026-1"）→ course_term.id：
	// 同一班号跨学期都有 offering 时，只返回所选学期的那个（review P3）。
	termId, ok := resolveTermIdForCalendar(calendarId)
	if !ok {
		// calendarId > 0 而日历/term 映射缺失（该学期未物化到课程目录）时保持空 classes：
		// 契约承诺「限定该学期内匹配」，不得退化为全学期查询（review CHANGES_REQUESTED）。
		return nil
	}
	offerings, err := course.ListVisibleOfferingsByClassCodes(classCodes, termId)
	if err != nil {
		return err
	}
	offeringIds := make([]uint64, 0, len(offerings))
	for _, o := range offerings {
		offeringIds = append(offeringIds, o.Id)
	}
	stats := course.ListOfferingStatsByIDs(offeringIds)
	teachers, err := classTeacherNames(offeringIds)
	if err != nil {
		return err
	}
	for _, o := range offerings {
		// 无教师关联的 offering 序列化为空数组而非 null（契约要求 teachers: array）。
		teacherNames := teachers[o.Id]
		if teacherNames == nil {
			teacherNames = []string{}
		}
		item := ReviewBriefClass{
			ClassCode:  o.ClassCode,
			OfferingId: o.Id,
			Teachers:   teacherNames,
		}
		if s, ok := stats[o.Id]; ok {
			item.ReviewCount = s.ReviewCount
			if s.RatingCount > 0 {
				avg := float64(s.RatingSum) / float64(s.RatingCount)
				item.RatingAvg = &avg
			}
		}
		brief.Classes = append(brief.Classes, item)
	}
	return nil
}

// resolveTermIdForCalendar 将 PK 学期（calendarId）映射到课程目录学期（course_term.id）：
// calendarId <= 0 时直接返回 (0, true)（不限定学期）；calendarId > 0 时要求
// calendar 存在、calendar_id_i18n 非空且能在 course_term 中找到对应学期，
// 任一环节缺失返回 (0, false)——调用方须保持空 classes，不得退化为全学期查询。
func resolveTermIdForCalendar(calendarId uint64) (uint64, bool) {
	if calendarId == 0 {
		return 0, true
	}
	cal, err := pk.GetCalendarByID(calendarId)
	if err != nil || cal.CalendarIdI18n == "" {
		return 0, false
	}
	term, err := course.GetTermByCode(cal.CalendarIdI18n)
	if err != nil {
		return 0, false
	}
	return term.Id, true
}

// classTeacherNames 批量返回 offering → 教师姓名列表（按关联顺序）。
func classTeacherNames(offeringIds []uint64) (map[uint64][]string, error) {
	result := make(map[uint64][]string, len(offeringIds))
	if len(offeringIds) == 0 {
		return result, nil
	}
	instructors, err := course.ListInstructorsByOfferings(offeringIds)
	if err != nil {
		return nil, err
	}
	links, err := course.ListOfferingInstructorLinks(offeringIds)
	if err != nil {
		return nil, err
	}
	nameById := make(map[uint64]string, len(instructors))
	for _, ins := range instructors {
		nameById[ins.Id] = ins.Name
	}
	for _, link := range links {
		if name, ok := nameById[link.InstructorId]; ok {
			result[link.OfferingId] = append(result[link.OfferingId], name)
		}
	}
	return result, nil
}
