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
		brief.RatingAvg = chosen.RatingAvg
		brief.ReviewCount = chosen.ReviewCount
		break
	}
	return brief, nil
}
