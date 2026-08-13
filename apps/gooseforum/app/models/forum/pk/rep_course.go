package pk

import "gorm.io/gorm"

// MajorCourseRow P5 courses-by-major 的明细行（含年级与 i18n 列）。
type MajorCourseRow struct {
	Id                   uint64
	Code                 string
	CourseCode           string
	CourseName           string
	CourseLabelId        int
	Credit               float64
	Campus               string
	Faculty              string
	TeachingLanguage     string
	CalendarId           int
	Grade                int
	FacultyI18n          string
	CampusI18n           string
	CourseLabelName      string
	TeachingLanguageI18n string
}

// ListMajorCourseRows 查询某专业（code）计划内教学班明细，包含更早年级（grade <= 当前）。
func ListMajorCourseRows(calendarId int, code string, grade int) ([]MajorCourseRow, error) {
	var rows []MajorCourseRow
	err := courseDetailBuilder().
		Select(
			`pk_course_detail.id, pk_course_detail.code, pk_course_detail.course_code,
			 pk_course_detail.course_name, pk_course_detail.course_label_id,
			 pk_course_detail.credit, pk_course_detail.campus, pk_course_detail.faculty,
			 pk_course_detail.teaching_language, pk_course_detail.calendar_id,
			 pk_major.grade, f.faculty_i18n, ca.campus_i18n,
			 n.course_label_name, l.teaching_language_i18n`).
		Joins("JOIN pk_major_course mac ON mac.course_id = pk_course_detail.id").
		Joins("JOIN pk_major ON pk_major.id = mac.major_id").
		Joins("LEFT JOIN pk_faculty f ON f.faculty = pk_course_detail.faculty").
		Joins("LEFT JOIN pk_campus ca ON ca.campus = pk_course_detail.campus").
		Joins("LEFT JOIN pk_course_nature n ON n.course_label_id = pk_course_detail.course_label_id AND n.calendar_id = pk_course_detail.calendar_id").
		Joins("LEFT JOIN pk_language l ON l.teaching_language = pk_course_detail.teaching_language").
		Where("pk_course_detail.calendar_id = ?", calendarId).
		Where("pk_major.code = ?", code).
		Where("pk_major.grade <= ?", grade).
		Order("pk_course_detail.course_code ASC, pk_course_detail.code ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// CourseDetailRow P8/P12/P13 课程明细行（含 i18n 列）。
type CourseDetailRow struct {
	Id                   uint64
	Code                 string
	CourseCode           string
	CourseName           string
	CourseLabelId        int
	Credit               float64
	Campus               string
	TeachingLanguage     string
	Faculty              string
	CalendarId           int
	CampusI18n           string
	TeachingLanguageI18n string
	NewCourseCode        string
	NewCode              string
}

// ListCourseDetailRowsByCodes 按 courseCode 批量查教学班明细（P8/P12）。
func ListCourseDetailRowsByCodes(calendarId int, codes []string) ([]CourseDetailRow, error) {
	unique := make([]string, 0, len(codes))
	seen := make(map[string]struct{}, len(codes))
	for _, code := range codes {
		code = trimSpace(code)
		if code == "" {
			continue
		}
		if _, ok := seen[code]; ok {
			continue
		}
		seen[code] = struct{}{}
		unique = append(unique, code)
	}
	if len(unique) == 0 {
		return nil, nil
	}
	var rows []CourseDetailRow
	for _, part := range chunkString(unique, MAX_SQL_VARS) {
		var batch []CourseDetailRow
		if err := courseDetailBuilder().
			Select(
				`pk_course_detail.id, pk_course_detail.code, pk_course_detail.course_code,
				 pk_course_detail.course_name, pk_course_detail.course_label_id,
				 pk_course_detail.credit, pk_course_detail.campus,
				 pk_course_detail.teaching_language, pk_course_detail.faculty,
				 pk_course_detail.calendar_id, ca.campus_i18n, l.teaching_language_i18n`).
			Joins("LEFT JOIN pk_campus ca ON ca.campus = pk_course_detail.campus").
			Joins("LEFT JOIN pk_language l ON l.teaching_language = pk_course_detail.teaching_language").
			Where("pk_course_detail.calendar_id = ?", calendarId).
			Where("pk_course_detail.course_code IN ?", part).
			Order("pk_course_detail.course_code ASC, pk_course_detail.code ASC").
			Scan(&batch).Error; err != nil {
			return nil, err
		}
		rows = append(rows, batch...)
	}
	return rows, nil
}

// ListAllCourseDetailRowsByCodes 按 courseCode 查该课程的全部教学班（含 i18n 列），
// 不限定专业（P5 第二查询：一个 courseCode 的所有班级都展示）。
func ListAllCourseDetailRowsByCodes(calendarId int, codes []string) ([]MajorCourseRow, error) {
	unique := make([]string, 0, len(codes))
	seen := make(map[string]struct{}, len(codes))
	for _, code := range codes {
		code = trimSpace(code)
		if code == "" {
			continue
		}
		if _, ok := seen[code]; ok {
			continue
		}
		seen[code] = struct{}{}
		unique = append(unique, code)
	}
	if len(unique) == 0 {
		return nil, nil
	}
	var rows []MajorCourseRow
	for _, part := range chunkString(unique, MAX_SQL_VARS) {
		var batch []MajorCourseRow
		if err := courseDetailBuilder().
			Select(
				`pk_course_detail.id, pk_course_detail.code, pk_course_detail.course_code,
				 pk_course_detail.course_name, pk_course_detail.course_label_id,
				 pk_course_detail.credit, pk_course_detail.campus, pk_course_detail.faculty,
				 pk_course_detail.teaching_language, pk_course_detail.calendar_id,
				 f.faculty_i18n, ca.campus_i18n, n.course_label_name, l.teaching_language_i18n`).
			Joins("LEFT JOIN pk_faculty f ON f.faculty = pk_course_detail.faculty").
			Joins("LEFT JOIN pk_campus ca ON ca.campus = pk_course_detail.campus").
			Joins("LEFT JOIN pk_course_nature n ON n.course_label_id = pk_course_detail.course_label_id AND n.calendar_id = pk_course_detail.calendar_id").
			Joins("LEFT JOIN pk_language l ON l.teaching_language = pk_course_detail.teaching_language").
			Where("pk_course_detail.calendar_id = ?", calendarId).
			Where("pk_course_detail.course_code IN ?", part).
			Order("pk_course_detail.course_code ASC, pk_course_detail.code ASC").
			Scan(&batch).Error; err != nil {
			return nil, err
		}
		rows = append(rows, batch...)
	}
	return rows, nil
}

// FindCourseDetailByCodeAnyCalendar 不限学期按 courseCode 查最近的一条教学班
// （P13 course-review-brief 不带 calendarId，取任意学期最近记录）。
func FindCourseDetailByCodeAnyCalendar(code string) (CourseDetailRow, error) {
	code = trimSpace(code)
	var row CourseDetailRow
	err := courseDetailBuilder().
		Select(
			`pk_course_detail.id, pk_course_detail.code, pk_course_detail.course_code,
			 pk_course_detail.course_name, pk_course_detail.course_label_id,
			 pk_course_detail.credit, pk_course_detail.campus,
			 pk_course_detail.teaching_language, pk_course_detail.faculty,
			 pk_course_detail.calendar_id, pk_course_detail.new_course_code,
			 pk_course_detail.new_code`).
		Where("pk_course_detail.course_code = ?", code).
		Order("pk_course_detail.calendar_id DESC, pk_course_detail.id ASC").
		Limit(1).
		Scan(&row).Error
	if err != nil {
		return row, err
	}
	if row.Id == 0 {
		return row, gorm.ErrRecordNotFound
	}
	return row, nil
}

// CourseSearchQuery P9 course-search 过滤条件。
type CourseSearchQuery struct {
	CalendarId  int
	CourseName  string
	CourseCode  string
	TeacherCode string
	TeacherName string
	Campus      string
	Faculty     string
	SizeLimit   int
}

// searchCourseFilter 应用 P9 高级检索过滤条件，返回带过滤的查询（JOIN 教师行会重复，只用于过滤/DISTINCT）。
func searchCourseFilter(q CourseSearchQuery) *gorm.DB {
	b := courseDetailBuilder().
		Joins("LEFT JOIN pk_faculty f ON f.faculty = pk_course_detail.faculty").
		Joins("LEFT JOIN pk_campus ca ON ca.campus = pk_course_detail.campus").
		Joins("LEFT JOIN pk_course_nature n ON n.course_label_id = pk_course_detail.course_label_id AND n.calendar_id = pk_course_detail.calendar_id").
		Joins("LEFT JOIN pk_teacher ON pk_teacher.teaching_class_id = pk_course_detail.id").
		Where("pk_course_detail.calendar_id = ?", q.CalendarId)
	if q.CourseName != "" {
		b = b.Where("pk_course_detail.course_name LIKE ?", "%"+q.CourseName+"%")
	}
	if q.CourseCode != "" {
		b = b.Where("(pk_course_detail.course_code = ? OR pk_course_detail.code = ?)", q.CourseCode, q.CourseCode)
	}
	if q.Campus != "" {
		b = b.Where("(pk_course_detail.campus = ? OR ca.campus_i18n = ?)", q.Campus, q.Campus)
	}
	if q.Faculty != "" {
		b = b.Where("(pk_course_detail.faculty = ? OR f.faculty_i18n = ?)", q.Faculty, q.Faculty)
	}
	if q.TeacherCode != "" {
		b = b.Where("pk_teacher.teacher_code = ?", q.TeacherCode)
	}
	if q.TeacherName != "" {
		b = b.Where("pk_teacher.teacher_name = ?", q.TeacherName)
	}
	return b
}

// SearchCourseCodes 返回满足过滤的 DISTINCT course_code（LIMIT 先于聚合，保证最多返回 sizeLimit 个课程）。
func SearchCourseCodes(q CourseSearchQuery) ([]string, error) {
	if q.SizeLimit <= 0 {
		q.SizeLimit = 100
	}
	var codes []string
	err := searchCourseFilter(q).
		Distinct("pk_course_detail.course_code").
		Where("pk_course_detail.course_code <> ''").
		Order("pk_course_detail.course_code ASC").
		Limit(q.SizeLimit).
		Pluck("pk_course_detail.course_code", &codes).Error
	if err != nil {
		return nil, err
	}
	return codes, nil
}

// SearchCourseRows 高级检索（P9）：先取 DISTINCT 课程码（LIMIT 100），再按这些码查明细，
// 返回明细行供 Go 侧按 courseCode 聚合（避免 JOIN 教师产生多行撑爆 LIMIT）。
func SearchCourseRows(q CourseSearchQuery) ([]CourseAggRow, error) {
	codes, err := SearchCourseCodes(q)
	if err != nil {
		return nil, err
	}
	if len(codes) == 0 {
		return nil, nil
	}
	rows, err := ListAllCourseDetailRowsByCodes(q.CalendarId, codes)
	if err != nil {
		return nil, err
	}
	agg := make([]CourseAggRow, 0, len(rows))
	for _, r := range rows {
		agg = append(agg, CourseAggRow{
			CourseCode:      r.CourseCode,
			CourseName:      r.CourseName,
			Credit:          r.Credit,
			FacultyI18n:     r.FacultyI18n,
			CourseLabelName: r.CourseLabelName,
			CampusI18n:      r.CampusI18n,
		})
	}
	return agg, nil
}

func trimSpace(v string) string {
	start := 0
	for start < len(v) && (v[start] == ' ' || v[start] == '\t' || v[start] == '\n' || v[start] == '\r') {
		start++
	}
	end := len(v)
	for end > start && (v[end-1] == ' ' || v[end-1] == '\t' || v[end-1] == '\n' || v[end-1] == '\r') {
		end--
	}
	return v[start:end]
}

func chunkString(arr []string, size int) [][]string {
	var out [][]string
	for i := 0; i < len(arr); i += size {
		end := i + size
		if end > len(arr) {
			end = len(arr)
		}
		out = append(out, arr[i:end])
	}
	return out
}
