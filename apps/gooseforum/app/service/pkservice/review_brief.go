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

func FindCourseReviewBrief(courseCode, teacherName string) (ReviewBrief, error) {
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

	stats, err := courseservice.GetCourseStatsByPrimaryCodes(candidateCodes)
	if err != nil {
		return brief, err
	}
	for _, code := range candidateCodes {
		if s, ok := stats[code]; ok {
			if brief.CourseName == "" {
				brief.CourseName = s.Name
			}
			brief.CourseId = s.CourseId
			brief.RatingAvg = s.RatingAvg
			brief.ReviewCount = s.ReviewCount
			break
		}
	}

	if err := fillClassBriefs(&brief); err != nil {
		return brief, err
	}
	return brief, nil
}

// fillClassBriefs 填充教学班级课评摘要：PK 教学班课号 → offering.class_code 匹配，
// 聚合 offering 级统计与教师；无匹配（旧数据包班号为空）时保持空数组。
func fillClassBriefs(brief *ReviewBrief) error {
	classCodes, err := pk.ListClassCodesByCourseCode(brief.CourseCode)
	if err != nil {
		return err
	}
	offerings, err := course.ListVisibleOfferingsByClassCodes(classCodes)
	if err != nil {
		return err
	}
	if len(offerings) == 0 {
		return nil
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
		item := ReviewBriefClass{
			ClassCode:  o.ClassCode,
			OfferingId: o.Id,
			Teachers:   teachers[o.Id],
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
