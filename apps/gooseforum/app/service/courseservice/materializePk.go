package courseservice

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
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

// pkCourseAgg 按 (courseCode, identityKey) 聚合的一系统课程（物化输入）。
// (code, teacher) 复合身份模型下，同一 courseCode 的不同教师是独立课程卡，
// 必须按身份教师拆分聚合，不能只按 courseCode 合并后取 TeacherNames[0]
// （否则会漏卡，且选中教师依赖查询顺序）。身份键 = teacher_code（身份主锚），
// 缺失时回退自然键 (normalized_name, department)（review Should）。
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
	// termCache 物化单次运行内的 calendarId → term_id 解析缓存：整学期数万教学班
	// 共用同一 calendar，避免每班重复 GetCalendarByIDTx + getOrCreateTermTx
	// （review Should：逐班 3×N 查询、长事务持锁）。term_id=0 表示无学期码（合法），
	// 同样缓存避免重复查询。
	termCache := map[uint64]uint64{}
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
			offeringCourseId, err := resolveOfferingCourseIdTx(tx, courseEntity.Id)
			if err != nil {
				return err
			}
			if offeringCourseId != courseEntity.Id {
				if err := searchservice.EnqueueCourseSearchTask(tx, offeringCourseId); err != nil {
					return fmt.Errorf("materialize: enqueue search for merged target %d: %w", offeringCourseId, err)
				}
			}
			for _, offering := range agg.ClassOfferings {
				if err := upsertPkOfferingTx(tx, offeringCourseId, offering, instructorCache, termCache, report); err != nil {
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
// 聚合键含院系消歧：无工号身份教师按 (归一姓名, 院系) 分组，跨院系同名教师
// 不会被并入同组（review Should）；offering 的 Faculty 取该班自身院系。
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
		// 该班自身院系（offering 级 Faculty 用本班院系，不继承组内其它班写入值）。
		dept := strings.TrimSpace(d.Faculty)
		if v, ok := facultyI18n[d.Faculty]; ok && v != "" {
			dept = v
		}
		var identity string
		var identityKey string
		if len(classTeachers) > 0 {
			identity = classTeachers[0].Name // 教学班首位教师 = 该班身份教师
			identityKey = teacherIdentityKey(classTeachers[0], dept)
		}
		key := code + "\x00" + identityKey
		agg, ok := byKey[key]
		if !ok {
			agg = &pkCourseAgg{CourseCode: code, IdentityTeacher: identity, Department: dept}
			order = append(order, key)
			byKey[key] = agg
		}
		// 院系：首个非空值生效（details 按 id ASC，跨班院系不一致时确定性取首班）。
		if agg.Department == "" && dept != "" {
			agg.Department = dept
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
			Faculty:         dept,
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

// teacherIdentityKey 身份教师在课程聚合键中的片段：与 pkTeacherCacheKey 同语义——
// teacher_code（身份主锚）非空时仅用 code；否则回退自然键 (normalized_name, department)。
// 无教师（姓名为空）返回空串（无教师班并入 (code, "") 组，避免 (code, teacher_id=0)
// 唯一索引下拆出重复卡）。
func teacherIdentityKey(ref pkTeacherRef, department string) string {
	if ref.Code != "" {
		return ref.Code
	}
	if Normalize(ref.Name) == "" {
		return ""
	}
	return Normalize(ref.Name) + "\x00" + department
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
		identity := pkTeacherRef{Name: agg.IdentityTeacher}
		for _, ref := range agg.TeacherRefs {
			if ref.Name == agg.IdentityTeacher {
				identity.Code = ref.Code
				break
			}
		}
		teacherId = instructorCache[pkTeacherCacheKey(identity, agg.Department)]
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

// resolveOfferingCourseIdTx 解析 offering 写入用的 courseId：课程卡已被确认合并
// （hidden 且存在指向可见卡的 status=merged EQUIVALENT/RENAMED_FROM 关系）时，
// offering 应写入合并目标卡而非 hidden 旧卡（否则物化会逆转已确认的合并）；
// 无 merged 关系或目标不可见则维持现状。只影响 offering 写入，不改课程卡 upsert。
// 课程状态以 DB 行为准（upsert 返回的 entity.Status 是写入意图，非库内状态）。
func resolveOfferingCourseIdTx(tx *gorm.DB, courseId uint64) (uint64, error) {
	entity := course.GetCourseByIdTx(tx, courseId)
	if entity.Id == 0 || entity.Status != course.StatusHidden {
		return courseId, nil
	}
	target, err := course.GetMergedTargetByFromCourseTx(tx, courseId)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, fmt.Errorf("materialize: lookup merged target of course %d: %w", courseId, err)
		}
		return courseId, nil
	}
	targetEntity := course.GetCourseByIdTx(tx, target)
	if targetEntity.Id == 0 || targetEntity.Status != course.StatusVisible {
		return courseId, nil
	}
	return target, nil
}

// pkTeacherCacheKey 教师引用的缓存/去重键：teacher_code（身份主锚）非空时用 code
// 主键，否则回退 (normalized_name, department) 自然键。
func pkTeacherCacheKey(ref pkTeacherRef, department string) string {
	if ref.Code != "" {
		return "code\x00" + ref.Code
	}
	return Normalize(ref.Name) + "\x00" + department
}

// upsertPkInstructorsTx 逐个对齐教师引用（按 pkTeacherCacheKey 缓存避免重复查询），
// 缓存命中即跳过；查询/创建逻辑在 upsertPkInstructorTx。
func upsertPkInstructorsTx(tx *gorm.DB, refs []pkTeacherRef, department string, cache map[string]uint64, report *MaterializeReport) error {
	for _, ref := range refs {
		key := pkTeacherCacheKey(ref, department)
		if _, ok := cache[key]; ok {
			continue
		}
		id, err := upsertPkInstructorTx(tx, ref, department, report)
		if err != nil {
			return err
		}
		cache[key] = id
	}
	return nil
}

// upsertPkInstructorTx 对齐单个教师：code 非空时按 teacher_code（身份主锚）查找；
// miss 时回退自然键——自然键行 code 为空则回填工号复用（历史导入数据），code 已被
// 其它工号占用则视为不同教师新建；code 为空走自然键，缺则新建。
func upsertPkInstructorTx(tx *gorm.DB, ref pkTeacherRef, department string, report *MaterializeReport) (uint64, error) {
	norm := Normalize(ref.Name)
	if ref.Code != "" {
		entity, err := course.FindInstructorByCodeTx(tx, ref.Code)
		switch {
		case err == nil:
			return entity.Id, nil
		case !errors.Is(err, gorm.ErrRecordNotFound):
			return 0, fmt.Errorf("materialize: lookup instructor by code %s: %w", ref.Code, err)
		}
		existing, nameErr := course.FindInstructorByNameDeptTx(tx, norm, department)
		switch {
		case nameErr == nil && existing.TeacherCode == "":
			// 历史数据无 code 的教师：回填工号（身份主锚，物化链权威），不新建。
			if err := tx.Model(&course.InstructorEntity{}).Where("id = ?", existing.Id).
				Update("teacher_code", ref.Code).Error; err != nil {
				return 0, fmt.Errorf("materialize: backfill instructor teacher_code %s: %w", ref.Name, err)
			}
			return existing.Id, nil
		case nameErr == nil:
			// 同名同院系已被其它工号占用：不同教师，按 code 新建独立行。
		case !errors.Is(nameErr, gorm.ErrRecordNotFound):
			return 0, fmt.Errorf("materialize: lookup instructor %s: %w", ref.Name, nameErr)
		}
		return createPkInstructorTx(tx, ref, department, report)
	}
	entity, err := course.FindInstructorByNameDeptTx(tx, norm, department)
	switch {
	case err == nil:
		return entity.Id, nil
	case errors.Is(err, gorm.ErrRecordNotFound):
		return createPkInstructorTx(tx, ref, department, report)
	default:
		return 0, fmt.Errorf("materialize: lookup instructor %s: %w", ref.Name, err)
	}
}

// createPkInstructorTx 按引用创建教师行（含拼音/首字母），并计入 InstructorsInserted。
func createPkInstructorTx(tx *gorm.DB, ref pkTeacherRef, department string, report *MaterializeReport) (uint64, error) {
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
		return 0, fmt.Errorf("materialize: create instructor %s: %w", ref.Name, err)
	}
	report.InstructorsInserted++
	return created.Id, nil
}

// resolveOfferingTermIdTx 解析 offering 的学期 id：calendarId<=0 直接返回 0；
// 命中 termCache（calendarId → term_id，0 = 无学期码）直接复用；未命中时查
// calendar 并 getOrCreateTermTx（失败显式上抛，不静默写 term_id=0），结果回填
// 缓存——物化单次运行内同一 calendar 只解析一次（review Should：逐班重复查询在
// 整学期数万教学班量级下拉长事务持锁时间）。
func resolveOfferingTermIdTx(tx *gorm.DB, calendarId uint64, termCache map[uint64]uint64) (uint64, error) {
	if calendarId == 0 {
		return 0, nil
	}
	if termId, ok := termCache[calendarId]; ok {
		return termId, nil
	}
	cal, err := pk.GetCalendarByIDTx(tx, calendarId)
	if err != nil {
		// calendar 行缺失/查询失败视为数据不一致：显式报错，不静默写 term_id=0。
		return 0, fmt.Errorf("materialize: lookup calendar %d: %w", calendarId, err)
	}
	termId := uint64(0)
	// calendar 存在但无学期码（i18n 为空）：合法"无学期码"情形，保持 term_id=0。
	// 学期码先规范化（"2026-2027学年第1学期" → "2026-2027-1"）：course_term.code
	// 是标准码，直接拿一系统中文学期名创建会生成垃圾学期行并改写存量 offering 的 term_id。
	// 仅标准码形（YYYY-YYYY-N）允许建行：无法识别的标记（如 "2024-2025学年短学期"）
	// 保持 term_id=0（与"无学期码"同语义）并记日志，杜绝垃圾学期行（review LOW）。
	if i18n := strings.TrimSpace(cal.CalendarIdI18n); i18n != "" {
		code := course.NormalizeTermLabel(i18n)
		if !course.IsCanonicalTermCode(code) {
			slog.Warn("materialize: 学期标记无法规范化为标准码，保持 term_id=0（不建 course_term 行）",
				"calendar_id", calendarId, "calendar_id_i18n", i18n, "normalized", code)
		} else {
			term, err := getOrCreateTermTx(tx, code)
			if err != nil {
				return 0, fmt.Errorf("materialize: resolve term %q: %w", code, err)
			}
			termId = term.Id
		}
	}
	termCache[calendarId] = termId
	return termId, nil
}

// upsertPkOfferingTx 按 teaching_class_id 幂等 upsert offering（物化链是 offering 权威写入源）。
// term 由 calendarId → calendar_id_i18n → course_term 映射（termCache 缓存，见
// resolveOfferingTermIdTx）；不写 status（防止复活管理员隐藏的 offering）。教师 = 该班
// 全量教师（offering_instructor 全量替换，复用 importer 的 replaceOfferingInstructorsTx）。
func upsertPkOfferingTx(tx *gorm.DB, courseId uint64, offering *pkOfferingAgg, instructorCache map[string]uint64, termCache map[uint64]uint64, report *MaterializeReport) error {
	// 先对齐该班教师（填充 instructorCache），再解析教师本地 id。
	if err := upsertPkInstructorsTx(tx, offering.TeacherRefs, offering.Faculty, instructorCache, report); err != nil {
		return err
	}
	instructorIDs := make([]uint64, 0, len(offering.TeacherRefs))
	for _, ref := range offering.TeacherRefs {
		id, ok := instructorCache[pkTeacherCacheKey(ref, offering.Faculty)]
		if !ok {
			return fmt.Errorf("materialize: offering instructor %s not aligned", ref.Name)
		}
		instructorIDs = append(instructorIDs, id)
	}
	// term 映射：calendarId → calendar_id_i18n → course_term.code → term id。
	// 解析结果按 calendarId 缓存（termCache），错误显式上抛。
	termId, err := resolveOfferingTermIdTx(tx, offering.CalendarId, termCache)
	if err != nil {
		return err
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
