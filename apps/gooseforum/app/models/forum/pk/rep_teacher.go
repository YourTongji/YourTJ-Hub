package pk

import (
	"errors"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/queryopt"
	"gorm.io/gorm"
)

// TeacherArrangeRow teacher 表 JOIN coursedetail 的行，用于懒构建 timeslots。
type TeacherArrangeRow struct {
	TeachingClassId uint64
	CalendarId      int
	TeacherCode     string
	TeacherName     string
	ArrangeInfoText string
}

// CourseAggRow 课程聚合查询的明细行（未聚合）。Go 侧按 course_code 合并，
// 避免 GROUP_CONCAT / STRING_AGG 的方言差异，保证 SQLite/MySQL/PG 三方言兼容。
type CourseAggRow struct {
	CourseCode      string
	CourseName      string
	FacultyI18n     string
	CourseLabelName string
	CampusI18n      string
	Credit          float64
}

// ListTeacherArrangeRows 返回全部教师的安排文本及其所属学期（teacher_timeslots 重建用）。
func ListTeacherArrangeRows() ([]TeacherArrangeRow, error) {
	var rows []TeacherArrangeRow
	err := teacherBuilder().
		Select("pk_teacher.teaching_class_id, pk_teacher.teacher_code, pk_teacher.teacher_name, pk_teacher.arrange_info_text, pk_course_detail.calendar_id").
		Joins("JOIN pk_course_detail ON pk_course_detail.id = pk_teacher.teaching_class_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// ClearTeacherTimeslots 清空时间片投影表。
func ClearTeacherTimeslots() error {
	return teacherTimeslotBuilder().Where("1 = 1").Delete(&TeacherTimeslotEntity{}).Error
}

// UpsertTeacherTimeslots 批量写入时间片投影（先清后写由调用方负责，这里仅插入）。
func UpsertTeacherTimeslots(entities []TeacherTimeslotEntity) error {
	if len(entities) == 0 {
		return nil
	}
	return teacherTimeslotBuilder().CreateInBatches(entities, 500).Error
}

// ListTimeslotCoursesBySlot 按时间片查询占用该时段的课程明细（courses-by-time 主路径）。
// optionalLabels 为通识/选修标签白名单；nature 为空或命中白名单的课程才返回。
func ListTimeslotCoursesBySlot(calendarId, day int, sections []int, optionalLabels []string) ([]CourseAggRow, error) {
	if len(sections) == 0 {
		return nil, nil
	}
	var rows []CourseAggRow
	b := courseDetailBuilder().
		Select(
			`pk_course_detail.course_code, pk_course_detail.course_name,
			 pk_course_detail.credit, f.faculty_i18n, n.course_label_name, ca.campus_i18n`).
		Joins("JOIN pk_teacher_timeslot ts ON ts.teaching_class_id = pk_course_detail.id").
		Joins("LEFT JOIN pk_faculty f ON f.faculty = pk_course_detail.faculty AND f.calendar_id = pk_course_detail.calendar_id").
		Joins("LEFT JOIN pk_campus ca ON ca.campus = pk_course_detail.campus AND ca.calendar_id = pk_course_detail.calendar_id").
		Joins("LEFT JOIN pk_course_nature n ON n.course_label_id = pk_course_detail.course_label_id AND n.calendar_id = pk_course_detail.calendar_id").
		Where("pk_course_detail.calendar_id = ?", calendarId).
		Where("ts.calendar_id = ?", calendarId).
		Where("ts.occupy_day = ?", day).
		Where("ts.occupy_section IN ?", sections)
	if len(optionalLabels) > 0 {
		b = b.Where("(n.course_label_name IS NULL OR n.course_label_name IN ?)", optionalLabels)
	}
	err := b.Order("pk_course_detail.course_code ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// ListTimeslotCoursesByLike 降级查询：timeslots 未就绪时回退 arrange_info_text LIKE。
// likePatterns 为形如 "%星期一1-2%" 的模式列表。
func ListTimeslotCoursesByLike(calendarId int, likePatterns []string, optionalLabels []string) ([]CourseAggRow, error) {
	if len(likePatterns) == 0 {
		return nil, nil
	}
	var rows []CourseAggRow
	b := courseDetailBuilder().
		Select(
			`pk_course_detail.course_code, pk_course_detail.course_name,
			 pk_course_detail.credit, f.faculty_i18n, n.course_label_name, ca.campus_i18n`).
		Joins("JOIN pk_teacher ON pk_teacher.teaching_class_id = pk_course_detail.id").
		Joins("LEFT JOIN pk_faculty f ON f.faculty = pk_course_detail.faculty AND f.calendar_id = pk_course_detail.calendar_id").
		Joins("LEFT JOIN pk_campus ca ON ca.campus = pk_course_detail.campus AND ca.calendar_id = pk_course_detail.calendar_id").
		Joins("LEFT JOIN pk_course_nature n ON n.course_label_id = pk_course_detail.course_label_id AND n.calendar_id = pk_course_detail.calendar_id").
		Where("pk_course_detail.calendar_id = ?", calendarId)
	orLike := ""
	for range likePatterns {
		if orLike != "" {
			orLike += " OR "
		}
		orLike += "pk_teacher.arrange_info_text LIKE ?"
	}
	// 变参展开：GORM 对普通 ? 不展开 slice，必须把每个 pattern 作为独立参数。
	args := make([]interface{}, len(likePatterns))
	for i, p := range likePatterns {
		args[i] = p
	}
	b = b.Where("("+orLike+")", args...)
	if len(optionalLabels) > 0 {
		b = b.Where("(n.course_label_name IS NULL OR n.course_label_name IN ?)", optionalLabels)
	}
	err := b.Order("pk_course_detail.course_code ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// ListTeachersByClassIds 按教学班 id 批量查教师（避免 N+1，按 IN 分块）。
func ListTeachersByClassIds(classIds []uint64) ([]TeacherEntity, error) {
	valid := make([]uint64, 0, len(classIds))
	seen := make(map[uint64]struct{}, len(classIds))
	for _, id := range classIds {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		valid = append(valid, id)
	}
	if len(valid) == 0 {
		return nil, nil
	}
	var entities []TeacherEntity
	for _, part := range chunkUint64(valid, MAX_SQL_VARS) {
		var batch []TeacherEntity
		if err := teacherBuilder().Where(queryopt.In("teaching_class_id", part)).
			Order("teaching_class_id ASC, id ASC").
			Find(&batch).Error; err != nil {
			return nil, err
		}
		entities = append(entities, batch...)
	}
	return entities, nil
}

// GetSetting 读取 PK 模块键值。
func GetSetting(key string) (SettingEntity, error) {
	var entity SettingEntity
	err := settingBuilder().Where(queryopt.Eq("key", key)).First(&entity).Error
	return entity, err
}

// SetSetting 写入 PK 模块键值（存在则覆盖）。
func SetSetting(key, value string) error {
	entity := SettingEntity{Key: key, Value: value}
	return settingBuilder().Where(queryopt.Eq("key", key)).
		Assign(entity).FirstOrCreate(&entity).Error
}

// GetLatestFetchTime 返回最近一次同步时间（Unix 秒），无记录返回 0。
func GetLatestFetchTime() (int64, error) {
	var entity FetchLogEntity
	err := fetchLogBuilder().Order("fetch_time DESC, id DESC").First(&entity).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, nil
		}
		return 0, err
	}
	return entity.FetchTime, nil
}

func chunkUint64(arr []uint64, size int) [][]uint64 {
	var out [][]uint64
	for i := 0; i < len(arr); i += size {
		end := i + size
		if end > len(arr) {
			end = len(arr)
		}
		out = append(out, arr[i:end])
	}
	return out
}
