package course

import (
	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/queryopt"
)

// ---- Related（相关课程：同教师其他课 / 同课程其他教师）----

// ListInstructorIDsByCourse 返回课程可见开课实例的全部教师 ID（去重，排除软删教师）。
// 相关课程计算用：同教师其他课 = 与该课程共享任一教师的其他可见课程（团队授课按教师成员匹配）。
func ListInstructorIDsByCourse(courseId uint64) ([]uint64, error) {
	var ids []uint64
	err := db.Connect().Table(offeringTableName+" o").
		Select("DISTINCT oi.instructor_id").
		Joins("JOIN "+offeringInstructorTableName+" oi ON oi.offering_id = o.id").
		Joins("JOIN "+instructorTableName+" ci ON ci.id = oi.instructor_id AND ci.deleted_at IS NULL").
		Where("o.course_id = ?", courseId).
		Where("o.deleted_at IS NULL").
		Where("o.status = ?", OfferingStatusVisible).
		Scan(&ids).Error
	return ids, err
}

// ListOtherCourseIDsByInstructors 返回与任一给定教师共享可见开课、且非指定课程的其他可见课程 ID。
// 仅返回 ID，排序由 service 按课程统计（review_count/评分）决定；DISTINCT 去重团队授课导致的重复。
// 关联教师同样排除软删，保证候选与详情页展示的教师名单一致。
func ListOtherCourseIDsByInstructors(instructorIds []uint64, excludeCourseId uint64) ([]uint64, error) {
	if len(instructorIds) == 0 {
		return nil, nil
	}
	var ids []uint64
	err := db.Connect().Table(tableName+" c").
		Select("DISTINCT c.id").
		Joins("JOIN "+offeringTableName+" o ON o.course_id = c.id AND o.deleted_at IS NULL").
		Joins("JOIN "+offeringInstructorTableName+" oi ON oi.offering_id = o.id").
		Joins("JOIN "+instructorTableName+" ci ON ci.id = oi.instructor_id AND ci.deleted_at IS NULL").
		Where("c.id <> ?", excludeCourseId).
		Where("c.deleted_at IS NULL").
		Where("c.status = ?", StatusVisible).
		Where("o.status = ?", OfferingStatusVisible).
		Where("oi.instructor_id IN ?", instructorIds).
		Scan(&ids).Error
	return ids, err
}

// ListOtherCoursesByPrimaryCode 返回同 primary_code 的其他可见课程行
// （(code, teacher) 复合身份模型下同一课号的不同教师卡），排除自身，按 id 升序。
// 供"同课程其他教师"相关区块使用：拆卡后该区块退化为同课号其他卡片。
func ListOtherCoursesByPrimaryCode(code string, excludeCourseId uint64) ([]Entity, error) {
	var entities []Entity
	err := courseBuilder().
		Where(queryopt.Eq("primary_code", code)).
		Where(queryopt.Ne("id", excludeCourseId)).
		Where(queryopt.Eq("status", StatusVisible)).
		Order("id ASC").
		Find(&entities).Error
	return entities, err
}

// GetCourseStatsMap 批量返回课程级评价统计（id -> stats）。
// 读取失败如实返回错误（相关课程依赖统计做排序/展示，吞错会让接口以 200 返回全零评分）。
func GetCourseStatsMap(courseIds []uint64) (map[uint64]CourseStatsEntity, error) {
	result := make(map[uint64]CourseStatsEntity, len(courseIds))
	if len(courseIds) == 0 {
		return result, nil
	}
	var entities []CourseStatsEntity
	if err := courseStatsBuilder().
		Where(queryopt.In("course_id", courseIds)).
		Find(&entities).Error; err != nil {
		return nil, err
	}
	for i := range entities {
		result[entities[i].CourseId] = entities[i]
	}
	return result, nil
}

// GetOfferingStatsMap 批量返回 offering 级评价统计（id -> stats）。
func GetOfferingStatsMap(offeringIds []uint64) (map[uint64]OfferingStatsEntity, error) {
	result := make(map[uint64]OfferingStatsEntity, len(offeringIds))
	if len(offeringIds) == 0 {
		return result, nil
	}
	var entities []OfferingStatsEntity
	if err := offeringStatsBuilder().
		Where(queryopt.In("offering_id", offeringIds)).
		Find(&entities).Error; err != nil {
		return nil, err
	}
	for i := range entities {
		result[entities[i].OfferingId] = entities[i]
	}
	return result, nil
}
