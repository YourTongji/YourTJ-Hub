package pkcontroller

import (
	"log/slog"
	"strings"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/defaultconfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/pkservice"
)

// Null 无请求参数的占位类型。
type Null struct{}

// maxPkArrayItems 用户可控数组入参的上限：防止超大 ids/courseCodes 触发大量分块查询。
const maxPkArrayItems = 500

func internalError(err error) Response {
	slog.Error("pk request failed", "error", err)
	return Internal("服务器内部错误")
}

// ---- P1 calendars ----

func ListCalendars(req Request[Null]) Response {
	items, err := pkservice.ListCalendars()
	if err != nil {
		return internalError(err)
	}
	return Ok(items)
}

// ---- P2 campuses + faculties ----

func ListCampuses(req Request[Null]) Response {
	items, err := pkservice.ListCampuses()
	if err != nil {
		return internalError(err)
	}
	return Ok(items)
}

func ListFaculties(req Request[Null]) Response {
	items, err := pkservice.ListFaculties()
	if err != nil {
		return internalError(err)
	}
	return Ok(items)
}

// ---- P3 grades ----

type GradesReq struct {
	CalendarId int `json:"calendarId"`
}

func Grades(req Request[GradesReq]) Response {
	if req.Params.CalendarId <= 0 {
		return BadRequest("参数错误: 缺少 calendarId")
	}
	result, err := pkservice.FindGradesByCalendar(req.Params.CalendarId)
	if err != nil {
		return internalError(err)
	}
	return Ok(result)
}

// ---- P4 majors ----

type MajorsReq struct {
	Grade      int `json:"grade"`
	CalendarId int `json:"calendarId"`
}

func Majors(req Request[MajorsReq]) Response {
	if req.Params.Grade <= 0 {
		return BadRequest("参数错误: 缺少 grade")
	}
	items, err := pkservice.FindMajorsByGrade(req.Params.Grade, req.Params.CalendarId)
	if err != nil {
		return internalError(err)
	}
	return Ok(items)
}

// ---- P5 courses-by-major ----

type CoursesByMajorReq struct {
	Grade      int    `json:"grade"`
	Code       string `json:"code"`
	CalendarId int    `json:"calendarId"`
}

func CoursesByMajor(req Request[CoursesByMajorReq]) Response {
	if req.Params.Grade <= 0 || strings.TrimSpace(req.Params.Code) == "" || req.Params.CalendarId <= 0 {
		return BadRequest("参数错误")
	}
	items, err := pkservice.FindCoursesByMajor(req.Params.Grade, strings.TrimSpace(req.Params.Code), req.Params.CalendarId)
	if err != nil {
		return internalError(err)
	}
	return Ok(items)
}

// ---- P6 optional-types ----

type OptionalTypesReq struct {
	CalendarId int `json:"calendarId"`
}

func OptionalTypes(req Request[OptionalTypesReq]) Response {
	if req.Params.CalendarId <= 0 {
		return BadRequest("参数错误: 缺少 calendarId")
	}
	items, err := pkservice.FindOptionalTypes(req.Params.CalendarId)
	if err != nil {
		return internalError(err)
	}
	return Ok(items)
}

// ---- P7 courses-by-nature ----

type CoursesByNatureReq struct {
	CalendarId int   `json:"calendarId"`
	Ids        []int `json:"ids"`
}

func CoursesByNature(req Request[CoursesByNatureReq]) Response {
	if req.Params.CalendarId <= 0 || len(req.Params.Ids) == 0 {
		return BadRequest("参数错误: calendarId 或 ids 不能为空")
	}
	if len(req.Params.Ids) > maxPkArrayItems {
		return BadRequest("参数错误: ids 数量超出上限")
	}
	items, err := pkservice.FindCoursesByNature(req.Params.CalendarId, req.Params.Ids)
	if err != nil {
		return internalError(err)
	}
	return Ok(items)
}

// ---- P8 course-details ----

type CourseDetailsReq struct {
	CalendarId  int      `json:"calendarId"`
	CourseCode  string   `json:"courseCode"`
	CourseCodes []string `json:"courseCodes"`
}

func CourseDetails(req Request[CourseDetailsReq]) Response {
	if req.Params.CalendarId <= 0 {
		return BadRequest("参数错误: 缺少 calendarId")
	}
	single := strings.TrimSpace(req.Params.CourseCode)
	if single != "" {
		dict, err := pkservice.FindCourseDetailsByCodes(req.Params.CalendarId, []string{single})
		if err != nil {
			return internalError(err)
		}
		// 单个 courseCode：返回数组；未匹配返回空数组。
		list, ok := dict[single]
		if !ok || list == nil {
			list = []pkservice.CourseDetailBriefItem{}
		}
		return Ok(list)
	}
	if len(req.Params.CourseCodes) == 0 {
		return BadRequest("参数错误: 缺少 courseCode(s)")
	}
	if len(req.Params.CourseCodes) > maxPkArrayItems {
		return BadRequest("参数错误: courseCodes 数量超出上限")
	}
	dict, err := pkservice.FindCourseDetailsByCodes(req.Params.CalendarId, req.Params.CourseCodes)
	if err != nil {
		return internalError(err)
	}
	return Ok(dict)
}

// ---- P9 course-search ----

type CourseSearchReq struct {
	CalendarId  int    `json:"calendarId"`
	CourseName  string `json:"courseName"`
	CourseCode  string `json:"courseCode"`
	TeacherCode string `json:"teacherCode"`
	TeacherName string `json:"teacherName"`
	Campus      string `json:"campus"`
	Faculty     string `json:"faculty"`
}

func CourseSearch(req Request[CourseSearchReq]) Response {
	if req.Params.CalendarId <= 0 {
		return BadRequest("参数错误: 缺少 calendarId")
	}
	result, err := pkservice.SearchCourses(pkservice.SearchCourseParams{
		CalendarId:  req.Params.CalendarId,
		CourseName:  req.Params.CourseName,
		CourseCode:  req.Params.CourseCode,
		TeacherCode: req.Params.TeacherCode,
		TeacherName: req.Params.TeacherName,
		Campus:      req.Params.Campus,
		Faculty:     req.Params.Faculty,
	})
	if err != nil {
		return internalError(err)
	}
	return Ok(result)
}

// ---- P10 courses-by-time ----

type CoursesByTimeReq struct {
	CalendarId int `json:"calendarId"`
	Day        int `json:"day"`
	Section    int `json:"section"`
}

func CoursesByTime(req Request[CoursesByTimeReq]) Response {
	if req.Params.CalendarId <= 0 || req.Params.Day < 1 || req.Params.Day > 7 || req.Params.Section < 1 || req.Params.Section > 6 {
		return BadRequest("输入参数有误")
	}
	result, err := pkservice.FindCoursesByTime(req.Params.CalendarId, req.Params.Day, req.Params.Section)
	if err != nil {
		return internalError(err)
	}
	return Ok(result)
}

// ---- P11 latest-update ----

func LatestUpdate(req Request[Null]) Response {
	value, err := pkservice.GetLatestUpdate()
	if err != nil {
		return internalError(err)
	}
	if value == "" {
		return Ok(nil)
	}
	return Ok(value)
}

// ---- P12 course-info-sync ----

type CourseInfoSyncReq struct {
	CalendarId       int                  `json:"calendarId"`
	MajorCourseCodes []string             `json:"majorCourseCodes"`
	OtherCourseCodes []string             `json:"otherCourseCodes"`
	MajorInfo        *pkservice.MajorInfo `json:"majorInfo"`
}

func CourseInfoSync(req Request[CourseInfoSyncReq]) Response {
	if req.Params.CalendarId <= 0 {
		return BadRequest("参数错误: 缺少 calendarId")
	}
	if len(req.Params.MajorCourseCodes)+len(req.Params.OtherCourseCodes) > maxPkArrayItems {
		return BadRequest("参数错误: 课程码数量超出上限")
	}
	result, err := pkservice.SyncCourseInfo(pkservice.SyncCourseInfoParams{
		CalendarId:       req.Params.CalendarId,
		MajorCourseCodes: req.Params.MajorCourseCodes,
		OtherCourseCodes: req.Params.OtherCourseCodes,
		MajorInfo:        req.Params.MajorInfo,
	})
	if err != nil {
		return internalError(err)
	}
	return Ok(result)
}

// ---- P13 course-review-brief ----

type CourseReviewBriefReq struct {
	CourseCode  string `form:"courseCode"`
	TeacherName string `form:"teacherName"`
	// CalendarId 可选：限定教学班课号只在该学期内匹配（跨学期班号复用时不串学期）。
	CalendarId uint64 `form:"calendarId"`
	// TeachingClassId 可选：教学班直查（by-offering 精准定位）。提供时按
	// course_offering.teaching_class_id 精确定位 offering/课程卡，不再走
	// courseCode+teacherName 猜测；未命中回退旧路径。
	TeachingClassId uint64 `form:"teachingClassId"`
}

func CourseReviewBrief(req Request[CourseReviewBriefReq]) Response {
	if strings.TrimSpace(req.Params.CourseCode) == "" && req.Params.TeachingClassId == 0 {
		return BadRequest("参数错误: 缺少 courseCode")
	}
	brief, err := pkservice.FindCourseReviewBrief(req.Params.CourseCode, req.Params.TeacherName, req.Params.CalendarId, req.Params.TeachingClassId)
	if err != nil {
		return internalError(err)
	}
	return Ok(brief)
}

// ---- 节次作息（移动端 Route A 公开只读）----

// maxRowsDefault 是移动端排课器默认节次行数：后端返回现行 11 节制默认表
// （2025-2026 学年起白天 1-8 节 + 晚间 9/10/11 节 18:30 起）；历史 12 节制
// 学期（calendarId<120）由客户端内置历史表渲染，不消费本响应。
const maxRowsDefault = 11

// SectionTimesResult 节次作息响应 data。
type SectionTimesResult struct {
	SectionTimes   []pageConfig.ScheduleSectionTime `json:"sectionTimes"`
	MaxRowsDefault int                              `json:"maxRowsDefault"`
}

// SectionTimes 返回排课器节次作息表（公开只读，与 /schedule SSR 与
// admin GET /schedule-settings 同一数据源、同一读取口径：page_config
// ScheduleSettings 经 NormalizeStoredScheduleSettings 归一——PR #496 之前
// 保存的旧 12 节编号存量按旧编号重映射为现行 11 节，避免晚间整体错位）。
// 配置为空（未保存 / 保存了空表）时返回内置现行 11 节默认作息——与前端
// sectionTimesFor 的「11 节制 overrides 为空回退默认表」语义一致，移动端
// 可直接应用返回表无需再做合并。
func SectionTimes(req Request[Null]) Response {
	config := defaultconfig.NormalizeStoredScheduleSettings(
		pageConfig.GetConfigByPageType(pageConfig.ScheduleSettings, pageConfig.ScheduleSettingsConfig{}),
	)
	times := config.SectionTimes
	if len(times) == 0 {
		times = defaultconfig.GetDefaultScheduleSettingsConfig().SectionTimes
	}
	return Ok(SectionTimesResult{
		SectionTimes:   times,
		MaxRowsDefault: maxRowsDefault,
	})
}
