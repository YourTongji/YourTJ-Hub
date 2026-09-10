package pkservice

import (
	"fmt"
	"strings"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pk"
	"gorm.io/gorm"
)

// writeBatchTxInner 将一批一系统教学班原始行转换为 PK 域实体并批量 upsert。
// 与上游 upsertCourseList 对齐：calendar/查找表去重 upsert，专业先 upsert 再回填 id。
func writeBatchTxInner(tx *gorm.DB, calendarId uint64, list []CourseRaw) (int, error) {
	return writeBatchTxInnerForAudience(tx, pk.AudienceUndergraduate, calendarId, list)
}

// writeBatchTxInnerForAudience 将指定受众的一批上游教学班写入统一 PK 表。
// 研究生数字 ID 使用独立命名空间；关联字段始终使用同一 scoped id，避免本科
// 与研究生的同一上游 ID 互相覆盖。
func writeBatchTxInnerForAudience(tx *gorm.DB, audience pk.Audience, calendarId uint64, list []CourseRaw) (int, error) {
	if len(list) == 0 {
		return 0, nil
	}
	if !audience.Valid() {
		return 0, fmt.Errorf("无效的一系统数据来源 %q", audience)
	}
	if !pk.ValidExternalID(calendarId) {
		return 0, fmt.Errorf("calendar ID exceeds the supported numeric range")
	}
	for _, row := range list {
		if id, ok := courseClassID(row); ok && !pk.ValidExternalID(id) {
			return 0, fmt.Errorf("class ID exceeds the supported numeric range")
		}
		for _, teacher := range row.TeacherList {
			if id, ok := teacherID(teacher); ok && !pk.ValidExternalID(id) {
				return 0, fmt.Errorf("teacher ID exceeds the supported numeric range")
			}
		}
	}
	scopedCalendarId := pk.ScopeID(audience, calendarId)

	// 元数据列（issue #185）：本批次所有行共享同一 schema 版本与同步时间。
	now := time.Now()

	// calendar 只需写一次（取首行 i18n）；学期起止日期从部署 config [pk.semester_dates]
	// 按 calendar_id_i18n 命中填充（一系统数据不含日期，已核实），未配置保持 NULL。
	calendarI18n := strings.TrimSpace(list[0].CalendarIdI18n)
	dateRange := loadSemesterDates()[calendarI18n]
	calendarRow := pk.CalendarEntity{CalendarId: scopedCalendarId, Audience: string(audience), ExternalId: calendarId, CalendarIdI18n: calendarI18n, SchemaVersion: pk.PKDataSchemaVersion, SyncedAt: &now}
	if dateRange.Start != nil || dateRange.End != nil {
		calendarRow.StartDate, calendarRow.EndDate = dateRange.Start, dateRange.End
	}
	if err := pk.UpsertCalendarsTx(tx, []pk.CalendarEntity{calendarRow}); err != nil {
		return 0, err
	}

	seenLang := map[string]bool{}
	seenNature := map[uint64]bool{}
	seenAssessment := map[string]bool{}
	seenCampus := map[string]bool{}
	seenFaculty := map[string]bool{}
	majorIDCache := map[string]uint64{}

	var langs []pk.LanguageEntity
	var natures []pk.CourseNatureEntity
	var naturesByCal []pk.CourseNatureByCalendarEntity
	var assessments []pk.AssessmentEntity
	var campuses []pk.CampusEntity
	var faculties []pk.FacultyEntity
	var majors []pk.MajorEntity
	var majorNamesByClass = map[uint64][]string{}
	var courseDetails []pk.CourseDetailEntity
	var teachers []pk.TeacherEntity

	for _, course := range list {
		teachingClassId, ok := courseClassID(course)
		if !ok {
			continue
		}

		lang := strings.TrimSpace(course.TeachingLanguage)
		if lang != "" && !seenLang[lang] {
			seenLang[lang] = true
			langs = append(langs, pk.LanguageEntity{
				Audience: string(audience), TeachingLanguage: lang, TeachingLanguageI18n: strings.TrimSpace(course.TeachingLanguageI18n), CalendarId: scopedCalendarId, SchemaVersion: pk.PKDataSchemaVersion, SyncedAt: &now,
			})
		}
		if course.CourseLabelId != nil {
			cid := *course.CourseLabelId
			if !seenNature[cid] {
				seenNature[cid] = true
				labelName := strings.TrimSpace(course.CourseLabelName)
				natures = append(natures, pk.CourseNatureEntity{Audience: string(audience), CourseLabelId: cid, CourseLabelName: labelName, CalendarId: scopedCalendarId, SchemaVersion: pk.PKDataSchemaVersion, SyncedAt: &now})
				naturesByCal = append(naturesByCal, pk.CourseNatureByCalendarEntity{Audience: string(audience), CalendarId: scopedCalendarId, CourseLabelId: cid, CourseLabelName: labelName, SchemaVersion: pk.PKDataSchemaVersion, SyncedAt: &now})
			}
		}
		mode := strings.TrimSpace(course.AssessmentMode)
		if mode != "" && !seenAssessment[mode] {
			seenAssessment[mode] = true
			assessments = append(assessments, pk.AssessmentEntity{
				Audience: string(audience), AssessmentMode: mode, AssessmentModeI18n: strings.TrimSpace(course.AssessmentModeI18n), CalendarId: scopedCalendarId, SchemaVersion: pk.PKDataSchemaVersion, SyncedAt: &now,
			})
		}
		campus := strings.TrimSpace(course.Campus)
		if campus != "" && !seenCampus[campus] {
			seenCampus[campus] = true
			campuses = append(campuses, pk.CampusEntity{Audience: string(audience), Campus: campus, CampusI18n: strings.TrimSpace(course.CampusI18n), CalendarId: scopedCalendarId, SchemaVersion: pk.PKDataSchemaVersion, SyncedAt: &now})
		}
		faculty := strings.TrimSpace(course.Faculty)
		if faculty != "" && !seenFaculty[faculty] {
			seenFaculty[faculty] = true
			faculties = append(faculties, pk.FacultyEntity{Audience: string(audience), Faculty: faculty, FacultyI18n: strings.TrimSpace(course.FacultyI18n), CalendarId: scopedCalendarId, SchemaVersion: pk.PKDataSchemaVersion, SyncedAt: &now})
		}

		var classMajorNames []string
		for _, rawMajor := range course.MajorList {
			majorName := strings.TrimSpace(rawMajor)
			if majorName == "" {
				continue
			}
			classMajorNames = append(classMajorNames, majorName)
			if _, seen := majorIDCache[majorName]; seen {
				continue
			}
			grade, code, name := parseMajorString(majorName)
			majors = append(majors, pk.MajorEntity{Audience: string(audience), Code: code, Grade: grade, Name: name, CalendarId: scopedCalendarId, SchemaVersion: pk.PKDataSchemaVersion, SyncedAt: &now})
			majorIDCache[majorName] = 0 // 占位，upsert 后回填
		}
		if len(classMajorNames) > 0 {
			majorNamesByClass[pk.ScopeID(audience, teachingClassId)] = classMajorNames
		}

		newCourseCode, newCode := computeNewCode(course.NewCourseCode, course.Code, course.CourseCode)
		var credit *float64
		if course.Credits != nil {
			credit = course.Credits
		} else {
			credit = course.Credit
		}
		scopedClassId := pk.ScopeID(audience, teachingClassId)
		courseDetails = append(courseDetails, pk.CourseDetailEntity{
			Id:               scopedClassId,
			Audience:         string(audience),
			ExternalId:       teachingClassId,
			Code:             strings.TrimSpace(course.Code),
			Name:             strings.TrimSpace(course.Name),
			CourseLabelId:    course.CourseLabelId,
			AssessmentMode:   strings.TrimSpace(course.AssessmentMode),
			Period:           course.Period,
			WeekHour:         course.WeekHour,
			Campus:           strings.TrimSpace(course.Campus),
			Number:           course.Number,
			ElcNumber:        course.ElcNumber,
			StartWeek:        course.StartWeek,
			EndWeek:          course.EndWeek,
			CourseCode:       strings.TrimSpace(course.CourseCode),
			CourseName:       strings.TrimSpace(course.CourseName),
			Credit:           credit,
			TeachingLanguage: strings.TrimSpace(course.TeachingLanguage),
			Faculty:          strings.TrimSpace(course.Faculty),
			CalendarId:       scopedCalendarId,
			NewCourseCode:    newCourseCode,
			NewCode:          newCode,
			SchemaVersion:    pk.PKDataSchemaVersion,
			SyncedAt:         &now,
		})

		arrangeInfo := strings.TrimSpace(course.ArrangeInfo)
		for _, t := range course.TeacherList {
			teacherID, ok := teacherID(t)
			if !ok {
				continue
			}
			teacherName := strings.TrimSpace(t.TeacherName)
			teachers = append(teachers, pk.TeacherEntity{
				Id:              pk.ScopeID(audience, teacherID),
				Audience:        string(audience),
				ExternalId:      teacherID,
				TeachingClassId: scopedClassId,
				TeacherCode:     strings.TrimSpace(t.TeacherCode),
				TeacherName:     teacherName,
				ArrangeInfoText: extractTeacherArrangeInfo(arrangeInfo, teacherName),
				SchemaVersion:   pk.PKDataSchemaVersion,
				SyncedAt:        &now,
			})
		}
	}

	// 先写查找表，再 upsert 专业以生成 id，最后写课程/教师并回填专业关联。
	if err := pk.UpsertLanguagesTx(tx, langs); err != nil {
		return 0, err
	}
	if err := pk.UpsertCourseNaturesTx(tx, natures); err != nil {
		return 0, err
	}
	if err := pk.UpsertCourseNatureByCalendarTx(tx, naturesByCal); err != nil {
		return 0, err
	}
	if err := pk.UpsertAssessmentsTx(tx, assessments); err != nil {
		return 0, err
	}
	if err := pk.UpsertCampusesTx(tx, campuses); err != nil {
		return 0, err
	}
	if err := pk.UpsertFacultiesTx(tx, faculties); err != nil {
		return 0, err
	}
	if err := pk.UpsertMajorsTx(tx, majors); err != nil {
		return 0, err
	}
	if err := pk.UpsertCourseDetailsTx(tx, courseDetails); err != nil {
		return 0, err
	}
	if err := pk.UpsertTeachersTx(tx, teachers); err != nil {
		return 0, err
	}

	var majorCourses []pk.MajorCourseEntity
	for name, id := range majorIDCache {
		if id == 0 {
			got, err := pk.GetMajorIdByAudienceNameTx(tx, audience, name)
			if err != nil || got == 0 {
				continue
			}
			majorIDCache[name] = got
		}
	}
	for classID, names := range majorNamesByClass {
		for _, name := range names {
			if mid, ok := majorIDCache[name]; ok && mid != 0 {
				majorCourses = append(majorCourses, pk.MajorCourseEntity{Audience: string(audience), MajorId: mid, CourseId: classID, SchemaVersion: pk.PKDataSchemaVersion, SyncedAt: &now})
			}
		}
	}
	if err := pk.UpsertMajorCoursesTx(tx, majorCourses); err != nil {
		return 0, err
	}

	return len(courseDetails), nil
}

// courseClassID 返回教学班 id（course.id），为空则跳过该行。
func courseClassID(course CourseRaw) (uint64, bool) {
	if course.Id == nil {
		return 0, false
	}
	return *course.Id, true
}

// teacherID 返回教师 id（teacherList[].id），为空则跳过。
func teacherID(t TeacherRaw) (uint64, bool) {
	if t.Id == nil {
		return 0, false
	}
	return *t.Id, true
}
