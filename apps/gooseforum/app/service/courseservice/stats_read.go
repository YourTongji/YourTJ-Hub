package courseservice

import (
	"sort"
	"strings"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/course"
)

// CourseStatsBrief 课程评价摘要（PK P13 course-review-brief 复用；
// 在 B1 统计投影落地前按需读取，避免直接跨域访问 course 表）。
type CourseStatsBrief struct {
	CourseId    uint64
	PrimaryCode string
	Name        string
	TeacherName string   // 卡片身份教师名（无教师时为空串）
	RatingAvg   *float64 // 无有效 rating 时为 nil
	ReviewCount int
}

// GetCourseStatsByPrimaryCodes 按主课号批量查询课程评价摘要（旧签名，保持兼容）。
// 只返回存在且可见的课程；未匹配或隐藏的课号不在结果中。
// 复合身份模型下同 code 多教师会产生多行——调用方需要按教师精确归因时
// 请使用 GetCourseStatsByPrimaryCodeTeacher。
func GetCourseStatsByPrimaryCodes(codes []string) (map[string]CourseStatsBrief, error) {
	result := make(map[string]CourseStatsBrief)
	if len(codes) == 0 {
		return result, nil
	}
	entities, err := course.ListCoursesByPrimaryCodes(codes)
	if err != nil {
		return nil, err
	}
	briefs := attachCourseStats(entities)
	for _, b := range briefs {
		result[b.PrimaryCode] = b // 同 code 多卡时后者覆盖（旧语义；新代码走 teacher 感知路径）
	}
	return result, nil
}

// GetCourseStatsByPrimaryCodeTeacher 按 (主课号, 教师名) 查询课程评价摘要
// （PK P13 course-review-brief 用）。复合身份模型下同 code 不同教师是独立卡：
//   - teacherName 非空时优先精确匹配卡片身份教师（归一化比较）；命中返回该教师卡；
//   - 未命中或 teacherName 为空时返回该 code 全部可见卡（id 升序，调用方取首条兜底），
//     兼容旧数据（卡片尚未回填 teacher_id）与无教师课程。
func GetCourseStatsByPrimaryCodeTeacher(code, teacherName string) ([]CourseStatsBrief, error) {
	if strings.TrimSpace(code) == "" {
		return []CourseStatsBrief{}, nil
	}
	entities, err := course.ListCoursesByPrimaryCodes([]string{code})
	if err != nil {
		return nil, err
	}
	briefs := attachCourseStats(entities)
	target := Normalize(teacherName)
	if target != "" {
		matched := make([]CourseStatsBrief, 0, len(briefs))
		for _, b := range briefs {
			if Normalize(b.TeacherName) == target {
				matched = append(matched, b)
			}
		}
		if len(matched) > 0 {
			return matched, nil
		}
	}
	return briefs, nil
}

// attachCourseStats 给课程实体附加评价统计与身份教师名，按 id 升序返回。
func attachCourseStats(entities []course.Entity) []CourseStatsBrief {
	ids := make([]uint64, 0, len(entities))
	for _, e := range entities {
		ids = append(ids, e.Id)
	}
	stats := make(map[uint64]course.CourseStatsEntity)
	if len(ids) > 0 {
		stats = course.ListCourseStatsByIDs(ids)
	}
	teacherIds := make([]uint64, 0, len(entities))
	for _, e := range entities {
		if e.TeacherId != 0 {
			teacherIds = append(teacherIds, e.TeacherId)
		}
	}
	teacherNameByID := make(map[uint64]string)
	if len(teacherIds) > 0 {
		if teachers, err := course.ListInstructorsByIDs(teacherIds); err == nil {
			for _, t := range teachers {
				teacherNameByID[t.Id] = t.Name
			}
		}
	}
	out := make([]CourseStatsBrief, 0, len(entities))
	for _, e := range entities {
		brief := CourseStatsBrief{
			CourseId:    e.Id,
			PrimaryCode: e.PrimaryCode,
			Name:        e.Name,
			TeacherName: teacherNameByID[e.TeacherId],
			ReviewCount: 0,
		}
		if s, ok := stats[e.Id]; ok {
			brief.ReviewCount = s.ReviewCount
			if s.RatingCount > 0 {
				avg := float64(s.RatingSum) / float64(s.RatingCount)
				brief.RatingAvg = &avg
			}
		}
		out = append(out, brief)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CourseId < out[j].CourseId })
	return out
}
