package courseservice

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/course"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pk"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/searchservice"
	"gorm.io/gorm"
)

// MaterializeReport 一系统课程物化到课程目录的报表。
type MaterializeReport struct {
	CoursesInserted     int `json:"coursesInserted"`
	CoursesUpdated      int `json:"coursesUpdated"`
	InstructorsInserted int `json:"instructorsInserted"`
	AliasesInserted     int `json:"aliasesInserted"`
	AliasesSkipped      int `json:"aliasesSkipped"`
}

// materializePkSourceAlias 物化写别名用的来源标记。
const materializePkSource = "onesystem"

// pkCourseAgg 按 courseCode 聚合的一系统课程（物化输入）。
type pkCourseAgg struct {
	CourseCode   string
	Name         string
	Credit       float64
	Department   string
	TeacherNames []string
	Aliases      []string
}

// MaterializeFromPk 将指定学期的一系统（PK）课程物化到课程目录：缺教师按名创建、缺课程按
// courseCode 创建、别名（courseCode/code/newCourseCode/newCode）映射到课程行，并对新增/更新
// 课程入队搜索同步。幂等且保守：不复活管理员隐藏课程、不抢占已被其它课程占用的别名。
//
// 边界规则：课程域 owner（本包）读取 PK 域数据（只读），写入本域表。
func MaterializeFromPk(ctx context.Context, calendarIds []uint64) (*MaterializeReport, error) {
	report := &MaterializeReport{}
	if len(calendarIds) == 0 {
		return report, nil
	}

	aggs, err := aggregatePkCourses(calendarIds)
	if err != nil {
		return nil, err
	}
	if len(aggs) == 0 {
		return report, nil
	}

	conn := db.Connect()
	err = conn.Transaction(func(tx *gorm.DB) error {
		instructorCache := map[string]uint64{}
		for _, agg := range aggs {
			courseEntity, _, err := upsertPkCourseTx(tx, agg, instructorCache, report)
			if err != nil {
				return err
			}
			if err := searchservice.EnqueueCourseSearchTask(tx, courseEntity.Id); err != nil {
				return fmt.Errorf("materialize: enqueue search for course %d: %w", courseEntity.Id, err)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return report, nil
}

// aggregatePkCourses 读取指定学期的一系统教学班并按 courseCode 聚合（名称取非空最优、学分取最大、
// 院系取 facultyI18n、教师/别名去重收集）。
func aggregatePkCourses(calendarIds []uint64) ([]*pkCourseAgg, error) {
	var details []pk.CourseDetailEntity
	var allTeachers []pk.TeacherEntity
	for _, cid := range calendarIds {
		rows, err := pk.ListCourseDetailsByCalendar(cid)
		if err != nil {
			return nil, err
		}
		details = append(details, rows...)
	}
	{
		var classIds []uint64
		for _, d := range details {
			classIds = append(classIds, d.Id)
		}
		if len(classIds) > 0 {
			teachers, err := pk.ListTeachersByClassIds(classIds)
			if err != nil {
				return nil, err
			}
			allTeachers = teachers
		}
	}
	faculties, err := pk.ListFacultiesTx(db.Connect())
	if err != nil {
		return nil, err
	}
	facultyI18n := map[string]string{}
	for _, f := range faculties {
		facultyI18n[f.Faculty] = f.FacultyI18n
	}
	teachersByClass := map[uint64][]string{}
	for _, t := range allTeachers {
		if t.TeacherName == "" {
			continue
		}
		teachersByClass[t.TeachingClassId] = appendUnique(teachersByClass[t.TeachingClassId], t.TeacherName)
	}

	order := make([]string, 0, len(details))
	byCode := map[string]*pkCourseAgg{}
	for _, d := range details {
		code := strings.TrimSpace(d.CourseCode)
		if code == "" {
			continue
		}
		agg, ok := byCode[code]
		if !ok {
			agg = &pkCourseAgg{CourseCode: code}
			order = append(order, code)
			byCode[code] = agg
		}
		// 名称：course_name 优先，其次 name，最后 courseCode
		name := strings.TrimSpace(d.CourseName)
		if name == "" {
			name = strings.TrimSpace(d.Name)
		}
		if name == "" {
			name = code
		}
		if agg.Name == "" || name != code {
			agg.Name = name
		}
		if d.Credit != nil && *d.Credit > agg.Credit {
			agg.Credit = *d.Credit
		}
		if dept, ok := facultyI18n[d.Faculty]; ok && dept != "" {
			agg.Department = dept
		} else if agg.Department == "" {
			agg.Department = strings.TrimSpace(d.Faculty)
		}
		agg.TeacherNames = appendUnique(agg.TeacherNames, teachersByClass[d.Id]...)
		for _, v := range []string{d.CourseCode, d.Code, d.NewCourseCode, d.NewCode} {
			v = strings.TrimSpace(v)
			if v != "" {
				agg.Aliases = appendUnique(agg.Aliases, v)
			}
		}
	}

	out := make([]*pkCourseAgg, 0, len(order))
	for _, code := range order {
		out = append(out, byCode[code])
	}
	return out, nil
}

// upsertPkCourseTx 在事务内 upsert 一门一系统课程（含教师/别名），返回课程实体与是否新建。
func upsertPkCourseTx(tx *gorm.DB, agg *pkCourseAgg, instructorCache map[string]uint64, report *MaterializeReport) (*course.Entity, bool, error) {
	pinyin, initials := searchservice.PinyinFields(agg.Name)
	entity := course.Entity{
		PrimaryCode:    agg.CourseCode,
		Name:           agg.Name,
		Department:     agg.Department,
		CreditX10:      int(math.Round(agg.Credit * 10)),
		NormalizedName: Normalize(agg.Name),
		NamePinyin:     pinyin,
		NameInitials:   initials,
		Status:         course.StatusVisible,
	}

	existing, err := course.GetCourseByPrimaryCodeTx(tx, agg.CourseCode)
	inserted := false
	switch {
	case err == nil:
		if err := tx.Model(&course.Entity{}).Where("id = ?", existing.Id).Updates(map[string]any{
			"name":            entity.Name,
			"department":      entity.Department,
			"credit_x10":      entity.CreditX10,
			"normalized_name": entity.NormalizedName,
			"name_pinyin":     entity.NamePinyin,
			"name_initials":   entity.NameInitials,
			// 不写 status：避免复活管理员隐藏课程。
		}).Error; err != nil {
			return nil, false, fmt.Errorf("materialize: update course %s: %w", agg.CourseCode, err)
		}
		entity.Id = existing.Id
		report.CoursesUpdated++
	case errors.Is(err, gorm.ErrRecordNotFound):
		if err := tx.Model(&course.Entity{}).Create(&entity).Error; err != nil {
			return nil, false, fmt.Errorf("materialize: create course %s: %w", agg.CourseCode, err)
		}
		inserted = true
		report.CoursesInserted++
	default:
		return nil, false, fmt.Errorf("materialize: lookup course %s: %w", agg.CourseCode, err)
	}

	if err := upsertPkInstructorsTx(tx, agg.TeacherNames, agg.Department, instructorCache, report); err != nil {
		return nil, false, err
	}
	if err := upsertPkAliasesTx(tx, &entity, agg.Aliases, report); err != nil {
		return nil, false, err
	}
	return &entity, inserted, nil
}

// upsertPkInstructorsTx 按 (normalized_name, department) 缺教师则创建，norm|dept 缓存避免重复查询。
func upsertPkInstructorsTx(tx *gorm.DB, names []string, department string, cache map[string]uint64, report *MaterializeReport) error {
	for _, name := range names {
		norm := Normalize(name)
		key := norm + "\x00" + department
		if _, ok := cache[key]; ok {
			continue
		}
		entity, err := course.FindInstructorByNameDeptTx(tx, norm, department)
		switch {
		case err == nil:
			cache[key] = entity.Id
		case errors.Is(err, gorm.ErrRecordNotFound):
			pinyin, initials := searchservice.PinyinFields(name)
			created := course.InstructorEntity{
				Name:           name,
				NormalizedName: Normalize(name),
				NamePinyin:     pinyin,
				NameInitials:   initials,
				Department:     department,
				Status:         0,
			}
			if err := tx.Model(&course.InstructorEntity{}).Create(&created).Error; err != nil {
				return fmt.Errorf("materialize: create instructor %s: %w", name, err)
			}
			cache[key] = created.Id
			report.InstructorsInserted++
		default:
			return fmt.Errorf("materialize: lookup instructor %s: %w", name, err)
		}
	}
	return nil
}

// upsertPkAliasesTx 为课程插入 code 类别名；若 (kind, normalized_value) 已被其它课程占用则跳过。
func upsertPkAliasesTx(tx *gorm.DB, entity *course.Entity, aliases []string, report *MaterializeReport) error {
	for _, alias := range aliases {
		norm := Normalize(alias)
		if norm == "" {
			continue
		}
		existing, err := course.GetAliasByNormalizedValueTx(tx, course.AliasKindCode, norm)
		switch {
		case err == nil:
			if existing.CourseId != entity.Id {
				report.AliasesSkipped++
				continue
			}
			// 同课程已有该别名：若此前被软删（唯一索引占位）则恢复。
			if existing.DeletedAt.Valid {
				if err := tx.Unscoped().Model(&course.AliasEntity{}).Where("id = ?", existing.Id).Updates(map[string]any{
					"value":      alias,
					"source":     materializePkSource,
					"deleted_at": nil,
				}).Error; err != nil {
					return fmt.Errorf("materialize: restore alias %q: %w", alias, err)
				}
			}
		case errors.Is(err, gorm.ErrRecordNotFound):
			if err := tx.Model(&course.AliasEntity{}).Create(&course.AliasEntity{
				CourseId:        entity.Id,
				Kind:            course.AliasKindCode,
				Value:           alias,
				NormalizedValue: norm,
				Source:          materializePkSource,
			}).Error; err != nil {
				return fmt.Errorf("materialize: create alias %q: %w", alias, err)
			}
			report.AliasesInserted++
		default:
			return fmt.Errorf("materialize: lookup alias %q: %w", alias, err)
		}
	}
	return nil
}

// appendUnique 追加元素去重。
func appendUnique(slice []string, values ...string) []string {
	seen := map[string]bool{}
	for _, v := range slice {
		seen[v] = true
	}
	for _, v := range values {
		if !seen[v] {
			seen[v] = true
			slice = append(slice, v)
		}
	}
	return slice
}
