package course

import (
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
)

const hotInstructorLimitDefault = 8

// HotInstructorRow 热门教师聚合查询结果（按执教课程的评分聚合，内部行）。
type HotInstructorRow struct {
	Id           uint64
	Name         string
	Department   string
	ReviewCount  int64
	RatingSum    int64
	RatingCount  int64
}

// ListHotInstructors 按评分提取热门教师：统计该教师名下可见开课实例关联课程的
// 课程级评价（course_review_stats），按均分降序 + 评论数降序取前 limit 名。
// 仅统计有评分的教师（rating_count>0）；其余被 GROUP BY 过滤（聚合本身不参与 WHERE，
// 用 HAVING 过滤以保证先聚合后过滤）。
func ListHotInstructors(limit int) ([]HotInstructorRow, error) {
	if limit <= 0 {
		limit = hotInstructorLimitDefault
	}
	if limit > 50 {
		limit = 50
	}
	var rows []HotInstructorRow
	err := dbconnect.Connect().
		Table(instructorTableName + " ci").
		Select(
			"ci.id, ci.name, ci.department, "+
				"SUM(s.rating_sum) AS rating_sum, SUM(s.rating_count) AS rating_count, SUM(s.review_count) AS review_count",
		).
		Joins("JOIN "+offeringInstructorTableName+" oi ON oi.instructor_id = ci.id").
		Joins("JOIN "+offeringTableName+" o ON o.id = oi.offering_id AND o.deleted_at IS NULL AND o.status = ?", OfferingStatusVisible).
		Joins("JOIN "+tableName+" c ON c.id = o.course_id AND c.deleted_at IS NULL AND c.status = ?", StatusVisible).
		Joins("JOIN "+courseStatsTableName+" s ON s.course_id = c.id AND s.deleted_at IS NULL").
		Where("ci.deleted_at IS NULL").
		Group("ci.id, ci.name, ci.department").
		Having("SUM(s.rating_count) > 0").
		Order("SUM(s.rating_sum) * 1.0 / NULLIF(SUM(s.rating_count), 0) DESC, SUM(s.review_count) DESC, ci.id ASC").
		Limit(limit).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}
