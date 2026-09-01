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
	OfferingsInserted   int `json:"offeringsInserted"`
	OfferingsUpdated    int `json:"offeringsUpdated"`
}

// materializePkSourceAlias 物化写别名用的来源标记。
const materializePkSource = "onesystem"

// pkTeacherRef 教学班教师（姓名 + 工号）；teacherCode 是身份主锚。
type pkTeacherRef struct {
	Name string
	Code string
}

// pkOfferingAgg 一个教学班的 offering 物化输入（班粒度）。
// 物化链是 offering 的权威写入源：按 teaching_class_id 幂等 upsert，
// 教师 = 该班全量教师，term 由 calendarId → calendar_id_i18n → course_term 映射。
type pkOfferingAgg struct {
	TeachingClassId uint64
	CalendarId      uint64
	ClassCode       string
	ClassName       string
	Campus          string
	Faculty         string // facultyI18n（与课程卡 department 同源）
	TeacherRefs     []pkTeacherRef
}

// pkCourseAgg 按 (courseCode, identityTeacher) 聚合的一系统课程（物化输入）。
// (code, teacher) 复合身份模型下，同一 courseCode 的不同教师是独立课程卡，
// 必须按身份教师拆分聚合，不能只按 courseCode 合并后取 TeacherNames[0]
// （否则会漏卡，且选中教师依赖查询顺序）。
type pkCourseAgg struct {
	CourseCode      string
	IdentityTeacher string // 该组身份教师名（教学班首位教师）；无教师为空串
	Name            string
	Credit          float64
	Department      string
	TeacherRefs     []pkTeacherRef
	Aliases         []string
	ClassOfferings  []*pkOfferingAgg // 该组课程下全部教学班（offering 物化输入）
}

// MaterializeFromPk 将指定学期的一系统（PK）课程物化到课程目录：缺教师按名创建、缺课程按
// courseCode 创建、别名（courseCode/code/newCourseCode/newCode）映射到课程行，并按教学班
// 补写 offering（offering 权威写入源 = 本物化链）。幂等且保守：不复活管理员隐藏课程、
// 不抢占已被其它课程占用的别名、不写 offering.status（防止复活管理员隐藏的 offering）。
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
			for _, offering := range agg.ClassOfferings {
				if err := upsertPkOfferingTx(tx, courseEntity.Id, offering, instructorCache, report); err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return report, nil
}

// aggregatePkCourses 读取指定学期的一系统教学班并按 (courseCode, 身份教师) 聚合
// （名称取非空最优、学分取最大、院系取 facultyI18n、教师/别名去重收集），并保留
// 教学班粒度（ClassOfferings）供 offering 物化。
// 身份教师 = 教学班首位教师（合班课其余教师保留在 TeacherNames 供 offering 名单）。
// 无教师教学班归入 (code, "") 组（teacher_id=0 卡）。
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
	campuses, err := pk.ListCampuses()
	if err != nil {
		return nil, err
	}
	campusI18n := map[string]string{}
	for _, c := range campuses {
		campusI18n[c.Campus] = c.CampusI18n
	}
	teachersByClass := map[uint64][]pkTeacherRef{}
	for _, t := range allTeachers {
		if t.TeacherName == "" {
			continue
		}
		teachersByClass[t.TeachingClassId] = appendTeacherRef(teachersByClass[t.TeachingClassId], pkTeacherRef{
			Name: t.TeacherName,
			Code: t.TeacherCode,
		})
	}

	order := make([]string, 0, len(details))
	byKey := map[string]*pkCourseAgg{}
	for _, d := range details {
		code := strings.TrimSpace(d.CourseCode)
		if code == "" {
			continue
		}
		classTeachers := teachersByClass[d.Id]
		var identity string
		if len(classTeachers) > 0 {
			identity = classTeachers[0].Name // 教学班首位教师 = 该班身份教师
		}
		key := code + "\x00" + identity
		agg, ok := byKey[key]
		if !ok {
			agg = &pkCourseAgg{CourseCode: code, IdentityTeacher: identity}
			order = append(order, key)
			byKey[key] = agg
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
		agg.TeacherRefs = appendTeacherRefs(agg.TeacherRefs, classTeachers...)
		for _, v := range []string{d.CourseCode, d.Code, d.NewCourseCode, d.NewCode} {
			v = strings.TrimSpace(v)
			if v != "" {
				agg.Aliases = appendUnique(agg.Aliases, v)
			}
		}
		// 教学班 → offering 物化输入（保留班粒度）。
		offering := &pkOfferingAgg{
			TeachingClassId: d.Id,
			CalendarId:      d.CalendarId,
			ClassCode:       strings.TrimSpace(d.Code),
			ClassName:       strings.TrimSpace(d.Name),
			Campus:          campusI18n[d.Campus],
			Faculty:         agg.Department,
			TeacherRefs:     classTeachers,
		}
		agg.ClassOfferings = append(agg.ClassOfferings, offering)
	}

	out := make([]*pkCourseAgg, 0, len(order))
	for _, key := range order {
		out = append(out, byKey[key])
	}
	return out, nil
}

// namesOf 提取教师姓名列表。
func namesOf(refs []pkTeacherRef) []string {
	out := make([]string, 0, len(refs))
	for _, r := range refs {
		out = append(out, r.Name)
	}
	return out
}

// codesOf 提取教师工号列表。
func codesOf(refs []pkTeacherRef) []string {
	out := make([]string, 0, len(refs))
	for _, r := range refs {
		if r.Code != "" {
			out = append(out, r.Code)
		}
	}
	return out
}

// appendTeacherRefs 批量追加教师引用（按姓名去重）。
func appendTeacherRefs(slice []pkTeacherRef, refs ...pkTeacherRef) []pkTeacherRef {
	for _, ref := range refs {
		slice = appendTeacherRef(slice, ref)
	}
	return slice
}

// appendTeacherRef 追加单个教师引用（姓名+工号）去重（按姓名）。
func appendTeacherRef(slice []pkTeacherRef, ref pkTeacherRef) []pkTeacherRef {
	for _, r := range slice {
		if r.Name == ref.Name {
			return slice
		}
	}
	return append(slice, ref)
}

func upsertPkCourseTx(tx *gorm.DB, agg *pkCourseAgg, instructorCache map[string]uint64, report *MaterializeReport) (*course.Entity, bool, error) {
	// 先对齐教师（填充 instructorCache），再解析课程身份教师：
	// (code, teacher) 复合身份下课程行身份教师 = 该组 IdentityTeacher
	// （教学班首位教师；合班课其余教师保留在 offering 教师名单），
	// 无教师组（IdentityTeacher 为空）落 teacher_id=0 卡。
	if err := upsertPkInstructorsTx(tx, agg.TeacherRefs, agg.Department, instructorCache, report); err != nil {
		return nil, false, err
	}
	var teacherId uint64
	if agg.IdentityTeacher != "" {
		norm := Normalize(agg.IdentityTeacher)
		teacherId = instructorCache[norm+"\x00"+agg.Department]
	}
	pinyin, initials := searchservice.PinyinFields(agg.Name)
	entity := course.Entity{
		PrimaryCode:    agg.CourseCode,
		Name:           agg.Name,
		Department:     agg.Department,
		CreditX10:      int(math.Round(agg.Credit * 10)),
		NormalizedName: Normalize(agg.Name),
		NamePinyin:     pinyin,
		NameInitials:   initials,
		TeacherId:      teacherId,
		Status:         course.StatusVisible,
	}

	existing, err := course.GetCourseByCodeTeacherTx(tx, agg.CourseCode, teacherId)
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

	if err := upsertPkAliasesTx(tx, &entity, agg.Aliases, report); err != nil {
		return nil, false, err
	}
	return &entity, inserted, nil
}

// upsertPkInstructorsTx 按 (normalized_name, department) 缺教师则创建，并写入 teacher_code
// （身份主锚）；norm|dept 缓存避免重复查询。
func upsertPkInstructorsTx(tx *gorm.DB, refs []pkTeacherRef, department string, cache map[string]uint64, report *MaterializeReport) error {
	for _, ref := range refs {
		norm := Normalize(ref.Name)
		key := norm + "\x00" + department
		if _, ok := cache[key]; ok {
			continue
		}
		entity, err := course.FindInstructorByNameDeptTx(tx, norm, department)
		switch {
		case err == nil:
			cache[key] = entity.Id
			// 教师已存在但 teacher_code 为空：回填工号（身份主锚，物化链权威）。
			if entity.TeacherCode == "" && ref.Code != "" {
				if err := tx.Model(&course.InstructorEntity{}).Where("id = ?", entity.Id).
					Update("teacher_code", ref.Code).Error; err != nil {
					return fmt.Errorf("materialize: backfill instructor teacher_code %s: %w", ref.Name, err)
				}
			}
		case errors.Is(err, gorm.ErrRecordNotFound):
			pinyin, initials := searchservice.PinyinFields(ref.Name)
			created := course.InstructorEntity{
				Name:           ref.Name,
				NormalizedName: Normalize(ref.Name),
				NamePinyin:     pinyin,
				NameInitials:   initials,
				Department:     department,
				TeacherCode:    ref.Code,
				Status:         0,
			}
			if err := tx.Model(&course.InstructorEntity{}).Create(&created).Error; err != nil {
				return fmt.Errorf("materialize: create instructor %s: %w", ref.Name, err)
			}
			cache[key] = created.Id
			report.InstructorsInserted++
		default:
			return fmt.Errorf("materialize: lookup instructor %s: %w", ref.Name, err)
		}
	}
	return nil
}

// upsertPkOfferingTx 按 teaching_class_id 幂等 upsert offering（物化链是 offering 权威写入源）。
// term 由 calendarId → calendar_id_i18n → course_term 映射；不写 status（防止复活管理员隐藏的
// offering）。教师 = 该班全量教师（offering_instructor 全量替换，复用 importer 的
// replaceOfferingInstructorsTx）。
func upsertPkOfferingTx(tx *gorm.DB, courseId uint64, offering *pkOfferingAgg, instructorCache map[string]uint64, report *MaterializeReport) error {
	// 先对齐该班教师（填充 instructorCache），再解析教师本地 id。
	if err := upsertPkInstructorsTx(tx, offering.TeacherRefs, offering.Faculty, instructorCache, report); err != nil {
		return err
	}
	instructorIDs := make([]uint64, 0, len(offering.TeacherRefs))
	for _, ref := range offering.TeacherRefs {
		id, ok := instructorCache[Normalize(ref.Name)+"\x00"+offering.Faculty]
		if !ok {
			return fmt.Errorf("materialize: offering instructor %s not aligned", ref.Name)
		}
		instructorIDs = append(instructorIDs, id)
	}
	// term 映射：calendarId → calendar_id_i18n → course_term.code → term id。
	// 物化链是 offering 权威写入源：缺学期码时自动创建 course_term（与 importer 的
	// getOrCreateTermTx 同语义），保证物化产物可被学期筛选（ListDistinctTerms 按
	// term_id JOIN course_term，term_id=0 的开课不会出现在学期筛选里）。
	var termId uint64
	if offering.CalendarId > 0 {
		cal, err := pk.GetCalendarByIDTx(tx, offering.CalendarId)
		if err == nil && strings.TrimSpace(cal.CalendarIdI18n) != "" {
			if term, err := getOrCreateTermTx(tx, strings.TrimSpace(cal.CalendarIdI18n)); err == nil {
				termId = term.Id
			}
		}
	}

	existing, err := course.GetOfferingByTeachingClassIdTx(tx, offering.TeachingClassId)
	switch {
	case err == nil:
		updates := map[string]any{
			"course_id":  courseId,
			"term_id":    termId,
			"campus":     offering.Campus,
			"faculty":    offering.Faculty,
			"class_code": offering.ClassCode,
			"class_name": offering.ClassName,
		}
		if err := tx.Model(&course.OfferingEntity{}).Where("id = ?", existing.Id).Updates(updates).Error; err != nil {
			return fmt.Errorf("materialize: update offering %d: %w", existing.Id, err)
		}
		report.OfferingsUpdated++
		if err := replaceOfferingInstructorsTx(tx, existing.Id, instructorIDs); err != nil {
			return err
		}
		if existing.CourseId != courseId {
			if err := searchservice.EnqueueCourseSearchTask(tx, existing.CourseId); err != nil {
				return fmt.Errorf("materialize: enqueue old course search %d: %w", existing.CourseId, err)
			}
		}
		return nil
	case errors.Is(err, gorm.ErrRecordNotFound):
		entity := course.OfferingEntity{
			CourseId:        courseId,
			TermId:          termId,
			TeachingClassId: offering.TeachingClassId,
			Campus:          offering.Campus,
			Faculty:         offering.Faculty,
			ClassCode:       offering.ClassCode,
			ClassName:       offering.ClassName,
			Status:          course.OfferingStatusVisible,
		}
		if err := tx.Model(&course.OfferingEntity{}).Create(&entity).Error; err != nil {
			return fmt.Errorf("materialize: create offering %d: %w", offering.TeachingClassId, err)
		}
		report.OfferingsInserted++
		return replaceOfferingInstructorsTx(tx, entity.Id, instructorIDs)
	default:
		return fmt.Errorf("materialize: lookup offering by teaching_class_id %d: %w", offering.TeachingClassId, err)
	}
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
