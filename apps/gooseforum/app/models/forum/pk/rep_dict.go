package pk

// ListCalendars 返回最近 limit 个学期（calendarId 倒序）。
func ListCalendars(limit int) ([]CalendarEntity, error) {
	if limit <= 0 {
		limit = 8
	}
	var entities []CalendarEntity
	err := calendarBuilder().Order("calendar_id DESC").Limit(limit).Find(&entities).Error
	return entities, err
}

// ListCampuses 返回全部校区字典（按 code 排序）。
func ListCampuses() ([]CampusEntity, error) {
	var entities []CampusEntity
	err := campusBuilder().Order("campus ASC").Find(&entities).Error
	return entities, err
}

// ListFaculties 返回全部院系列表（按 code 排序）。
func ListFaculties() ([]FacultyEntity, error) {
	var entities []FacultyEntity
	err := facultyBuilder().Order("faculty ASC").Find(&entities).Error
	return entities, err
}

// ListGradesByCalendar 返回某学期计划内课程的年级（按年级倒序去重）。
func ListGradesByCalendar(calendarId int) ([]int, error) {
	var grades []int
	err := majorBuilder().
		Select("DISTINCT pk_major.grade").
		Joins("JOIN pk_major_course mac ON mac.major_id = pk_major.id").
		Joins("JOIN pk_course_detail cd ON cd.id = mac.course_id").
		Where("cd.calendar_id = ?", calendarId).
		Where("pk_major.grade > 0").
		Order("pk_major.grade DESC").
		Pluck("pk_major.grade", &grades).Error
	if err != nil {
		return nil, err
	}
	return grades, nil
}

// MajorOption 年级 → 专业候选（{code, name}）。
type MajorOption struct {
	Code string
	Name string
}

// ListMajorsByGrade 返回某年级的专业列表；hasCalendar 时限定该学期有计划内课程。
func ListMajorsByGrade(grade, calendarId int, hasCalendar bool) ([]MajorOption, error) {
	b := majorBuilder().Select("DISTINCT pk_major.code, pk_major.name")
	if hasCalendar {
		b = b.
			Joins("JOIN pk_major_course mac ON mac.major_id = pk_major.id").
			Joins("JOIN pk_course_detail cd ON cd.id = mac.course_id").
			Where("cd.calendar_id = ?", calendarId)
	}
	var options []MajorOption
	err := b.Where("pk_major.grade = ?", grade).
		Order("pk_major.code ASC").
		Scan(&options).Error
	if err != nil {
		return nil, err
	}
	return options, nil
}

// GetTargetMajorId 返回 (code, grade) 对应且在该学期有计划内课程的专业 id。
func GetTargetMajorId(code string, grade, calendarId int) (uint64, error) {
	var id uint64
	err := majorBuilder().
		Select("pk_major.id").
		Joins("JOIN pk_major_course mac ON mac.major_id = pk_major.id").
		Joins("JOIN pk_course_detail cd ON cd.id = mac.course_id").
		Where("pk_major.code = ?", code).
		Where("pk_major.grade = ?", grade).
		Where("cd.calendar_id = ?", calendarId).
		Order("pk_major.id DESC").
		Limit(1).
		Scan(&id).Error
	if err != nil {
		return 0, err
	}
	return id, nil
}

// GetNearestMajorId 返回 code 匹配且 grade <= 给定年级的最近专业 id（P12 isExclusive 用，
// 对齐上游 getLatestCourseInfo 的 `grade <= ? ORDER BY grade DESC LIMIT 1` 语义，不限学期）。
func GetNearestMajorId(code string, grade int) (uint64, error) {
	var id uint64
	err := majorBuilder().
		Select("id").
		Where("code = ?", code).
		Where("grade <= ?", grade).
		Order("grade DESC, id DESC").
		Limit(1).
		Scan(&id).Error
	if err != nil {
		return 0, err
	}
	return id, nil
}

// ListMajorCourseIds 返回某专业的教学班 id 集合（isExclusive 判定用）。
func ListMajorCourseIds(majorId uint64) ([]uint64, error) {
	if majorId == 0 {
		return nil, nil
	}
	var ids []uint64
	err := majorCourseBuilder().
		Select("course_id").
		Where("major_id = ?", majorId).
		Pluck("course_id", &ids).Error
	if err != nil {
		return nil, err
	}
	return ids, nil
}

// CourseNatureOption 课程性质候选（P6）。
type CourseNatureOption struct {
	CourseLabelId   int
	CourseLabelName string
}

// ListOptionalTypesByCalendar 返回某学期的通识/选修课程性质（coursenature_by_calendar）。
// labels 为通识/选修标签白名单。无匹配返回空切片；查询错误必须向上传播（不能静默当空数据）。
func ListOptionalTypesByCalendar(calendarId int, labels []string) ([]CourseNatureOption, error) {
	if len(labels) == 0 {
		return []CourseNatureOption{}, nil
	}
	var options []CourseNatureOption
	err := courseNatureBuilder().
		Select("DISTINCT pk_course_nature.course_label_id, pk_course_nature.course_label_name").
		Joins("JOIN pk_course_detail cd ON cd.course_label_id = pk_course_nature.course_label_id AND cd.calendar_id = pk_course_nature.calendar_id").
		Where("pk_course_nature.calendar_id = ?", calendarId).
		Where("pk_course_nature.course_label_name IN ?", labels).
		Order("pk_course_nature.course_label_id DESC").
		Scan(&options).Error
	if err != nil {
		return nil, err
	}
	return options, nil
}

// ListCourseNatureRowsByLabelIds 按课程性质 id 查课程明细（P7，Go 侧按 labelName 合并）。
// 按 MAX_SQL_VARS 分块，避免 SQLite/MySQL 单查询变量上限。
func ListCourseNatureRowsByLabelIds(calendarId int, ids []int) ([]CourseNatureRow, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var rows []CourseNatureRow
	for _, part := range chunkInt(ids, MAX_SQL_VARS) {
		var batch []CourseNatureRow
		if err := courseDetailBuilder().
			Select(
				`pk_course_detail.course_code, pk_course_detail.course_name,
				 pk_course_detail.course_label_id, pk_course_detail.credit,
				 f.faculty_i18n, n.course_label_name, ca.campus_i18n`).
			Joins("LEFT JOIN pk_course_nature n ON n.course_label_id = pk_course_detail.course_label_id AND n.calendar_id = pk_course_detail.calendar_id").
			Joins("LEFT JOIN pk_faculty f ON f.faculty = pk_course_detail.faculty AND f.calendar_id = pk_course_detail.calendar_id").
			Joins("LEFT JOIN pk_campus ca ON ca.campus = pk_course_detail.campus AND ca.calendar_id = pk_course_detail.calendar_id").
			Where("pk_course_detail.calendar_id = ?", calendarId).
			Where("pk_course_detail.course_label_id IN ?", part).
			Order("pk_course_detail.course_label_id DESC, pk_course_detail.course_code ASC").
			Scan(&batch).Error; err != nil {
			return nil, err
		}
		rows = append(rows, batch...)
	}
	return rows, nil
}

// CourseNatureRow P7 查询明细行。
type CourseNatureRow struct {
	CourseCode      string
	CourseName      string
	CourseLabelId   int
	CourseLabelName string
	FacultyI18n     string
	CampusI18n      string
	Credit          float64
}

func chunkInt(arr []int, size int) [][]int {
	var out [][]int
	for i := 0; i < len(arr); i += size {
		end := i + size
		if end > len(arr) {
			end = len(arr)
		}
		out = append(out, arr[i:end])
	}
	return out
}
