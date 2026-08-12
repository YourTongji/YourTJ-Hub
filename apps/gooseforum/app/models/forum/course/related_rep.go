package course

import (
	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/queryopt"
)

// ---- Related（相关课程：同教师其他课 / 同课程其他教师）----

// ListInstructorIDsByCourse 返回课程可见开课实例的全部教师 ID（去重）。
// 相关课程计算用：同教师其他课 = 与该课程共享任一教师的其他可见课程（团队授课按教师成员匹配）。
func ListInstructorIDsByCourse(courseId uint64) ([]uint64, error) {
	var ids []uint64
	err := db.Connect().Table(offeringTableName+" o").
		Select("DISTINCT oi.instructor_id").
		Joins("JOIN "+offeringInstructorTableName+" oi ON oi.offering_id = o.id").
		Where("o.course_id = ?", courseId).
		Where("o.deleted_at IS NULL").
		Where("o.status = ?", OfferingStatusVisible).
		Scan(&ids).Error
	return ids, err
}

// ListOtherCourseIDsByInstructors 返回与任一给定教师共享可见开课、且非指定课程的其他可见课程 ID。
// 仅返回 ID，排序由 service 按课程统计（review_count/评分）决定；DISTINCT 去重团队授课导致的重复。
func ListOtherCourseIDsByInstructors(instructorIds []uint64, excludeCourseId uint64) ([]uint64, error) {
	if len(instructorIds) == 0 {
		return nil, nil
	}
	var ids []uint64
	err := db.Connect().Table(tableName+" c").
		Select("DISTINCT c.id").
		Joins("JOIN "+offeringTableName+" o ON o.course_id = c.id AND o.deleted_at IS NULL").
		Joins("JOIN "+offeringInstructorTableName+" oi ON oi.offering_id = o.id").
		Where("c.id <> ?", excludeCourseId).
		Where("c.deleted_at IS NULL").
		Where("c.status = ?", StatusVisible).
		Where("o.status = ?", OfferingStatusVisible).
		Where("oi.instructor_id IN ?", instructorIds).
		Scan(&ids).Error
	return ids, err
}

// GetCourseStatsMap 批量返回课程级评价统计（id -> stats）。
func GetCourseStatsMap(courseIds []uint64) map[uint64]CourseStatsEntity {
	result := make(map[uint64]CourseStatsEntity, len(courseIds))
	if len(courseIds) == 0 {
		return result
	}
	var entities []CourseStatsEntity
	courseStatsBuilder().
		Where(queryopt.In("course_id", courseIds)).
		Find(&entities)
	for i := range entities {
		result[entities[i].CourseId] = entities[i]
	}
	return result
}

// GetOfferingStatsMap 批量返回 offering 级评价统计（id -> stats）。
func GetOfferingStatsMap(offeringIds []uint64) map[uint64]OfferingStatsEntity {
	result := make(map[uint64]OfferingStatsEntity, len(offeringIds))
	if len(offeringIds) == 0 {
		return result
	}
	var entities []OfferingStatsEntity
	offeringStatsBuilder().
		Where(queryopt.In("offering_id", offeringIds)).
		Find(&entities)
	for i := range entities {
		result[entities[i].OfferingId] = entities[i]
	}
	return result
}
