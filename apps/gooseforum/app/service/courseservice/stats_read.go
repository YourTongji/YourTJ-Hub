package courseservice

import (
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/course"
)

// CourseStatsBrief 课程评价摘要（PK P13 course-review-brief 复用；
// 在 B1 统计投影落地前按需读取，避免直接跨域访问 course 表）。
type CourseStatsBrief struct {
	CourseId    uint64
	PrimaryCode string
	Name        string
	RatingAvg   *float64 // 无有效 rating 时为 nil
	ReviewCount int
}

// GetCourseStatsByPrimaryCodes 按主课号批量查询课程评价摘要。
// 只返回存在且可见的课程；未匹配或隐藏的课号不在结果中。
func GetCourseStatsByPrimaryCodes(codes []string) (map[string]CourseStatsBrief, error) {
	result := make(map[string]CourseStatsBrief)
	if len(codes) == 0 {
		return result, nil
	}
	entities, err := course.ListCoursesByPrimaryCodes(codes)
	if err != nil {
		return nil, err
	}
	ids := make([]uint64, 0, len(entities))
	for _, e := range entities {
		ids = append(ids, e.Id)
	}
	stats := make(map[uint64]course.CourseStatsEntity)
	if len(ids) > 0 {
		list, err := course.ListCourseStatsByIDs(ids)
		if err != nil {
			return nil, err
		}
		for i := range list {
			stats[list[i].CourseId] = list[i]
		}
	}
	for _, e := range entities {
		brief := CourseStatsBrief{
			CourseId:    e.Id,
			PrimaryCode: e.PrimaryCode,
			Name:        e.Name,
			ReviewCount: 0,
		}
		if s, ok := stats[e.Id]; ok {
			brief.ReviewCount = s.ReviewCount
			if s.RatingCount > 0 {
				avg := float64(s.RatingSum) / float64(s.RatingCount)
				brief.RatingAvg = &avg
			}
		}
		result[e.PrimaryCode] = brief
	}
	return result, nil
}
