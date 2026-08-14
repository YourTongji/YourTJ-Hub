package pkservice

import (
	"errors"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pk"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/courseservice"
	"gorm.io/gorm"
)

// ReviewBrief P13 course-review-brief 输出项：排课器弹窗展示的课程评价摘要。
type ReviewBrief struct {
	CourseCode  string   `json:"courseCode"`
	CourseName  string   `json:"courseName"`
	TeacherName string   `json:"teacherName"`
	RatingAvg   *float64 `json:"ratingAvg"`
	ReviewCount int      `json:"reviewCount"`
}

// FindCourseReviewBrief P13：按 PK 课程码（+教师名）查课程评价摘要。
// 先取 PK 教学班数据中的课程名，再尝试匹配 Hub 课评目录（newCourseCode / courseCode →
// primary_code），未匹配时返回仅含 PK 侧课程名、统计为空的摘要。
func FindCourseReviewBrief(courseCode, teacherName string) (ReviewBrief, error) {
	brief := ReviewBrief{
		CourseCode:  normalizeText(courseCode),
		TeacherName: normalizeText(teacherName),
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
			brief.RatingAvg = s.RatingAvg
			brief.ReviewCount = s.ReviewCount
			break
		}
	}
	return brief, nil
}
