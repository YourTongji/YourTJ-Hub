package courseservice

import (
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/course"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pk"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/searchservice"
	"gorm.io/gorm"
)

func existingPkIdentitiesTx(tx *gorm.DB, details []pk.CourseDetailEntity) (map[uint64]pkTeacherRef, error) {
	result := map[uint64]pkTeacherRef{}
	for start := 0; start < len(details); start += 80 {
		end := min(start+80, len(details))
		ids := make([]uint64, 0, end-start)
		for _, d := range details[start:end] {
			ids = append(ids, d.Id)
		}
		var rows []struct {
			TeachingClassId uint64
			Name            string
			Code            string
		}
		err := tx.Table("course_offering o").Select("o.teaching_class_id, i.name, i.teacher_code AS code").
			Joins("JOIN course c ON c.id=o.course_id AND c.deleted_at IS NULL").
			Joins("JOIN course_instructor i ON i.id=c.teacher_id AND i.deleted_at IS NULL").
			Where("o.teaching_class_id IN ? AND o.deleted_at IS NULL", ids).Scan(&rows).Error
		if err != nil {
			return nil, err
		}
		for _, r := range rows {
			result[r.TeachingClassId] = pkTeacherRef{Name: r.Name, Code: r.Code}
		}
	}
	return result, nil
}

// Reconcile even an already-moved offering: older runs may have left its alias
// behind. Manual aliases and class numbers still used by historical offerings
// retain ownership; class-code search also consults offerings directly.
func reconcilePkClassAliasesTx(tx *gorm.DB, to uint64, offering *pkOfferingAgg) error {
	codes := []string{}
	for _, code := range offering.ClassAliases {
		if norm := Normalize(code); norm != "" {
			codes = appendUnique(codes, norm)
		}
	}
	if len(codes) == 0 {
		return nil
	}
	var aliases []course.AliasEntity
	if err := tx.Model(&course.AliasEntity{}).Where("course_id <> ? AND kind = ? AND normalized_value IN ? AND source = ?", to, course.AliasKindCode, codes, materializePkSource).Find(&aliases).Error; err != nil {
		return err
	}
	for _, alias := range aliases {
		var remaining int64
		if err := tx.Model(&course.OfferingEntity{}).Where("course_id = ? AND LOWER(class_code) IN ?", alias.CourseId, codes).Count(&remaining).Error; err != nil {
			return err
		}
		if remaining > 0 {
			continue
		}
		if err := tx.Model(&course.AliasEntity{}).Where("id = ?", alias.Id).Update("course_id", to).Error; err != nil {
			return err
		}
		if err := searchservice.EnqueueCourseSearchTask(tx, alias.CourseId); err != nil {
			return err
		}
	}
	return nil
}

func enqueueRenamedInstructorCoursesTx(tx *gorm.DB, instructorID uint64) error {
	var ids []uint64
	if err := tx.Model(&course.Entity{}).Where(`teacher_id = ? OR id IN (
 SELECT o.course_id FROM course_offering o JOIN course_offering_instructor oi ON oi.offering_id=o.id
 WHERE oi.instructor_id = ? AND o.deleted_at IS NULL)`, instructorID, instructorID).Pluck("id", &ids).Error; err != nil {
		return err
	}
	for _, id := range ids {
		if err := searchservice.EnqueueCourseSearchTask(tx, id); err != nil {
			return err
		}
	}
	return nil
}
