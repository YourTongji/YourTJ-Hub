package course

import (
	"github.com/leancodebox/GooseForum/app/bundles/queryopt"
	"gorm.io/gorm"
)

// ---- Course ----

// GetCourse 按主键读取课程（含 soft-delete 过滤）。
func GetCourse(id uint64) (entity Entity) {
	courseBuilder().First(&entity, id)
	return
}

// GetCourseByPrimaryCode 按主课号精确查找（含 soft-delete 过滤）。
func GetCourseByPrimaryCode(code string) (entity Entity, err error) {
	return GetCourseByPrimaryCodeTx(courseBuilder(), code)
}

// GetCourseByPrimaryCodeTx 事务内按主课号精确查找，能看到同一事务内未提交的写入。
func GetCourseByPrimaryCodeTx(tx *gorm.DB, code string) (entity Entity, err error) {
	err = tx.Where(queryopt.Eq("primary_code", code)).First(&entity).Error
	return
}

// ListCourseQuery 课程目录筛选条件。
type ListCourseQuery struct {
	Keyword    string // 名称/课号/别名/教师（归一化前缀或包含）
	Department string // 院系精确
	TermCode   string // 学期（通过 offering 关联）
	Campus     string // 校区（通过 offering 关联）
	Page       int
	Size       int
}

// ListCourses 返回课程列表（canonical course 一页），并返回总条数。
// 排序固定为 id 倒序（新课程优先），保证分页稳定。
func ListCourses(q ListCourseQuery) (entities []Entity, total int64, err error) {
	b := courseBuilder().Where(queryopt.Eq("status", StatusVisible))
	if q.Department != "" {
		b = b.Where(queryopt.Eq("department", q.Department))
	}
	if q.Keyword != "" {
		kw := "%" + q.Keyword + "%"
		b = b.Where(
			`(normalized_name LIKE ? OR primary_code LIKE ? OR name_pinyin LIKE ? OR name_initials LIKE ?
OR EXISTS (SELECT 1 FROM course_alias WHERE course_alias.course_id = course.id AND course_alias.deleted_at IS NULL AND (course_alias.value LIKE ? OR course_alias.normalized_value LIKE ?))
OR EXISTS (
	SELECT 1 FROM course_offering
	JOIN course_offering_instructor ON course_offering_instructor.offering_id = course_offering.id
	JOIN course_instructor ON course_instructor.id = course_offering_instructor.instructor_id AND course_instructor.deleted_at IS NULL
	WHERE course_offering.course_id = course.id AND course_offering.deleted_at IS NULL
	  AND (course_instructor.name LIKE ? OR course_instructor.normalized_name LIKE ? OR course_instructor.name_pinyin LIKE ? OR course_instructor.name_initials LIKE ?)
))`,
			kw, kw, kw, kw, kw, kw, kw, kw, kw, kw,
		)
	}
	if q.TermCode != "" || q.Campus != "" {
		ob := offeringBuilder()
		if q.TermCode != "" {
			ob = ob.Where("term_id IN (SELECT id FROM course_term WHERE code = ?)", q.TermCode)
		}
		if q.Campus != "" {
			ob = ob.Where(queryopt.Eq("campus", q.Campus))
		}
		sub := ob.Select("course_id")
		b = b.Where("id IN (?)", sub)
	}
	if err = b.Count(&total).Error; err != nil {
		return
	}
	if q.Size <= 0 {
		q.Size = 20
	}
	if q.Size > 50 {
		q.Size = 50
	}
	if q.Page <= 0 {
		q.Page = 1
	}
	err = b.Order("id DESC").Offset((q.Page - 1) * q.Size).Limit(q.Size).Find(&entities).Error
	return
}

// ListAllCourses 全量遍历课程（重建搜索索引/统计用），按 id 升序 keyset 分页。
func ListAllCourses(limit, offset int) (entities []Entity, err error) {
	if limit <= 0 {
		return []Entity{}, nil
	}
	err = courseBuilder().Order("id ASC").Offset(offset).Limit(limit).Find(&entities).Error
	return
}

// GetMapByIds 批量按 ID 查询课程，返回 id -> entity 映射（搜索 hydration 用）。
func GetMapByIds(ids []uint64) map[uint64]*Entity {
	result := make(map[uint64]*Entity, len(ids))
	if len(ids) == 0 {
		return result
	}
	var entities []Entity
	if err := courseBuilder().Where(queryopt.In("id", ids)).Find(&entities).Error; err != nil {
		return result
	}
	for i := range entities {
		result[entities[i].Id] = &entities[i]
	}
	return result
}

// ---- Alias ----

// GetAliasByNormalizedValue 按 (kind, normalized_value) 查找别名（跨课程冲突检测）。
func GetAliasByNormalizedValue(kind, value string) (entity AliasEntity, err error) {
	return GetAliasByNormalizedValueTx(aliasBuilder(), kind, value)
}

// GetAliasByNormalizedValueTx 事务内按 (kind, normalized_value) 查找别名。
func GetAliasByNormalizedValueTx(tx *gorm.DB, kind, value string) (entity AliasEntity, err error) {
	err = tx.
		Where(queryopt.Eq("kind", kind)).
		Where(queryopt.Eq("normalized_value", value)).
		First(&entity).Error
	return
}

func ListAliasesByCourse(courseId uint64) (entities []AliasEntity, err error) {
	err = aliasBuilder().Where(queryopt.Eq("course_id", courseId)).Order("id ASC").Find(&entities).Error
	return
}

// ListAliasesByCourses 批量返回多门课程的别名（避免列表页 N+1）。
func ListAliasesByCourses(courseIds []uint64) (entities []AliasEntity, err error) {
	if len(courseIds) == 0 {
		return []AliasEntity{}, nil
	}
	err = aliasBuilder().
		Where(queryopt.In("course_id", courseIds)).
		Order("course_id ASC, id ASC").
		Find(&entities).Error
	return
}

// ---- Term ----

// GetTermByCode 按学期代码精确查找。
func GetTermByCode(code string) (entity TermEntity, err error) {
	return GetTermByCodeTx(termBuilder(), code)
}

// GetTermByCodeTx 事务内按学期代码精确查找，能看到同一事务内未提交的写入。
func GetTermByCodeTx(tx *gorm.DB, code string) (entity TermEntity, err error) {
	err = tx.Where(queryopt.Eq("code", code)).First(&entity).Error
	return
}

// ListTermsByIDs 批量返回学期（详情页 offering → term 名称）。
func ListTermsByIDs(ids []uint64) (entities []TermEntity, err error) {
	if len(ids) == 0 {
		return []TermEntity{}, nil
	}
	err = termBuilder().Where(queryopt.In("id", ids)).Find(&entities).Error
	return
}

// ---- Offering ----

// GetOffering 按 ID 读取开课实例（含软删除过滤）。
func GetOffering(id uint64) (entity OfferingEntity, err error) {
	err = offeringBuilder().Where("id = ?", id).First(&entity).Error
	return
}

// ListOfferingsByCourse 单课程的可见开课实例，按学期时间序（starts_on，回退 code）倒序。
func ListOfferingsByCourse(courseId uint64) (entities []OfferingEntity, err error) {
	err = offeringBuilder().
		Joins("LEFT JOIN course_term ON course_term.id = course_offering.term_id AND course_term.deleted_at IS NULL").
		Where(queryopt.Eq("course_offering.course_id", courseId)).
		Where(queryopt.Eq("course_offering.status", OfferingStatusVisible)).
		Order("COALESCE(CAST(course_term.starts_on AS TEXT), course_term.code) DESC, course_offering.id ASC").
		Find(&entities).Error
	return
}

// ListOfferingsByCourses 批量返回多门课程的开课实例（列表页避免 N+1）。
// 排序通过 term 的 starts_on（未设置时回退 code 字典序）保证学期时间序，不依赖自增 id。
func ListOfferingsByCourses(courseIds []uint64) (entities []OfferingEntity, err error) {
	if len(courseIds) == 0 {
		return []OfferingEntity{}, nil
	}
	err = offeringBuilder().
		Joins("LEFT JOIN course_term ON course_term.id = course_offering.term_id AND course_term.deleted_at IS NULL").
		Where(queryopt.In("course_offering.course_id", courseIds)).
		Where(queryopt.Eq("course_offering.status", OfferingStatusVisible)).
		Order("course_offering.course_id ASC, COALESCE(CAST(course_term.starts_on AS TEXT), course_term.code) DESC, course_offering.id ASC").
		Find(&entities).Error
	return
}

// ---- Instructor ----

// FindInstructorByNameDept 按 (normalized_name, department) 自然键查找教师。
func FindInstructorByNameDept(name, department string) (entity InstructorEntity, err error) {
	return FindInstructorByNameDeptTx(instructorBuilder(), name, department)
}

// FindInstructorByNameDeptTx 事务内按 (normalized_name, department) 自然键查找教师。
func FindInstructorByNameDeptTx(tx *gorm.DB, name, department string) (entity InstructorEntity, err error) {
	err = tx.
		Where(queryopt.Eq("normalized_name", name)).
		Where(queryopt.Eq("department", department)).
		First(&entity).Error
	return
}

// ListInstructorsByOfferings 批量返回多个开课实例的教师（详情/列表页避免 N+1）。
func ListInstructorsByOfferings(offeringIds []uint64) (entities []InstructorEntity, err error) {
	if len(offeringIds) == 0 {
		return []InstructorEntity{}, nil
	}
	err = instructorBuilder().
		Joins("JOIN course_offering_instructor ON course_offering_instructor.instructor_id = course_instructor.id").
		Where(queryopt.In("course_offering_instructor.offering_id", offeringIds)).
		Order("course_offering_instructor.offering_id ASC, course_instructor.id ASC").
		Find(&entities).Error
	return
}

// ---- OfferingInstructor ----

// ListOfferingInstructorLinks 批量返回多个开课实例的教师关联（用于按 offering 分组）。
func ListOfferingInstructorLinks(offeringIds []uint64) (entities []OfferingInstructorEntity, err error) {
	if len(offeringIds) == 0 {
		return []OfferingInstructorEntity{}, nil
	}
	err = offeringInstructorBuilder().
		Where(queryopt.In("offering_id", offeringIds)).
		Order("offering_id ASC, instructor_id ASC").
		Find(&entities).Error
	return
}

// ---- ImportRun ----

func CreateImportRun(entity *ImportRunEntity) error {
	return importRunBuilder().Create(entity).Error
}

func GetImportRunByManifestHash(hash string) (entity ImportRunEntity, err error) {
	err = importRunBuilder().Where(queryopt.Eq("manifest_hash", hash)).First(&entity).Error
	return
}

func SaveImportRun(entity *ImportRunEntity) error {
	return importRunBuilder().Save(entity).Error
}
