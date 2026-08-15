package forum

import (
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/i18n"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/courseservice"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
)

// CourseCatalog 课程目录 SSR 页面（/courses）。
func CourseCatalog(c *gin.Context) {
	page := parsePositiveInt(c.DefaultQuery("page", "1"), 1)
	size := parsePositiveInt(c.DefaultQuery("size", "20"), 20)
	props := buildCourseCatalogProps(c, page, size)
	payload := PagePayload{
		Component: PageComponentCourse,
		Props:     props,
		Meta:      buildCourseMeta(c),
		Layout:    buildLayout(c, "courses"),
		URL:       buildPageURL(c),
		Version:   payloadVersion,
	}
	renderAppShell(c, payload)
}

// CourseDetail 课程详情 SSR 页面（/courses/:courseId）。
func CourseDetail(c *gin.Context) {
	courseId := cast.ToUint64(c.Param("courseId"))
	detail, err := courseservice.GetCourseDetail(courseId)
	if err != nil {
		if errors.Is(err, courseservice.ErrCourseNotFound) {
			renderNotFoundWithMessage(c, component.MessagePageNotFound)
			return
		}
		slog.Error("course_detail_read_failed", "courseId", courseId, "error", err)
		renderInternalError(c)
		return
	}
	payload := PagePayload{
		Component: PageComponentCourseDetail,
		Props:     buildCourseDetailProps(detail),
		Meta:      buildCourseMeta(c),
		Layout:    buildLayout(c, "courses"),
		URL:       buildPageURL(c),
		Version:   payloadVersion,
	}
	renderAppShell(c, payload)
}

// CourseListReq 课程目录 JSON API 请求参数。
// OnlyWithReviews 用 string 承接以便解析 1/true/on 等多种布尔表示（见 parseBoolLike）。
type CourseListReq struct {
	Keyword         string `form:"keyword"`
	Department      string `form:"department"`
	TermCode        string `form:"term"`
	Campus          string `form:"campus"`
	Instructor      string `form:"instructor"`
	OnlyWithReviews string `form:"onlyWithReviews"`
	SortBy          string `form:"sortBy"`
	Page            int    `form:"page"`
	Size            int    `form:"size"`
}

// CourseListJSON 课程目录 JSON API（公开只读，供前端异步加载与后续移动端使用）。
// page/size 越界值按 service 归一化处理（page>=1、1<=size<=50），
// OpenAPI 契约（listCourses page/size description）已声明该 clamp 行为。
func CourseListJSON(req component.BetterRequest[CourseListReq]) component.Response {
	pageData, err := courseservice.ListCatalog(courseservice.CatalogQuery{
		Keyword:    strings.TrimSpace(req.Params.Keyword),
		Department: strings.TrimSpace(req.Params.Department),
		TermCode:   strings.TrimSpace(req.Params.TermCode),
		Campus:     strings.TrimSpace(req.Params.Campus),
		Instructor: strings.TrimSpace(req.Params.Instructor),
		HasReview:  parseBoolLike(req.Params.OnlyWithReviews),
		SortBy:     strings.TrimSpace(req.Params.SortBy),
		Page:       req.Params.Page,
		Size:       req.Params.Size,
	})
	if err != nil {
		slog.Error("course_catalog_list_failed", "error", err)
		return component.BuildResponse(http.StatusInternalServerError,
			component.FailDataCode(component.MessageOperationFailed, nil))
	}
	return component.SuccessResponse(pageData)
}

// CourseDetailReq 课程详情 JSON API 请求参数。
type CourseDetailReq struct {
	CourseId uint64 `uri:"courseId"`
}

func CourseDetailJSON(req component.BetterRequest[CourseDetailReq]) component.Response {
	if req.Params.CourseId == 0 {
		return component.BuildResponse(http.StatusBadRequest,
			component.FailDataCode(component.MessageRequestInvalidParams, nil))
	}
	detail, err := courseservice.GetCourseDetail(req.Params.CourseId)
	if err != nil {
		if errors.Is(err, courseservice.ErrCourseNotFound) {
			return component.BuildResponse(http.StatusNotFound,
				component.FailDataCode(component.MessagePageNotFound, nil))
		}
		slog.Error("course_detail_read_failed", "courseId", req.Params.CourseId, "error", err)
		return component.BuildResponse(http.StatusInternalServerError,
			component.FailDataCode(component.MessageOperationFailed, nil))
	}
	return component.SuccessResponse(detail)
}

// CourseRelatedReq 相关课程 JSON API 请求参数。
type CourseRelatedReq struct {
	CourseId uint64 `uri:"courseId"`
}

// CourseRelatedJSON 相关课程 JSON API（公开只读）：同教师其他课 + 同课程其他教师，
// 各前 5 条带评分与评论数。课程不存在或已隐藏时返回 404（与 CourseDetailJSON 一致）。
func CourseRelatedJSON(req component.BetterRequest[CourseRelatedReq]) component.Response {
	if req.Params.CourseId == 0 {
		return component.BuildResponse(http.StatusBadRequest,
			component.FailDataCode(component.MessageRequestInvalidParams, nil))
	}
	related, err := courseservice.GetCourseRelated(req.Params.CourseId)
	if err != nil {
		if errors.Is(err, courseservice.ErrCourseNotFound) {
			return component.BuildResponse(http.StatusNotFound,
				component.FailDataCode(component.MessagePageNotFound, nil))
		}
		slog.Error("course_related_read_failed", "courseId", req.Params.CourseId, "error", err)
		return component.BuildResponse(http.StatusInternalServerError,
			component.FailDataCode(component.MessageOperationFailed, nil))
	}
	return component.SuccessResponse(related)
}

// buildCourseCatalogProps 构建课程目录 SSR props；分页与分页回显以 service 归一化后的结果为准。
func buildCourseCatalogProps(c *gin.Context, page, size int) CourseCatalogProps {
	keyword := strings.TrimSpace(c.Query("keyword"))
	department := strings.TrimSpace(c.Query("department"))
	termCode := strings.TrimSpace(c.Query("term"))
	campus := strings.TrimSpace(c.Query("campus"))
	instructor := strings.TrimSpace(c.Query("instructor"))
	onlyWithReviews := parseBoolLike(c.Query("onlyWithReviews"))
	sortBy := strings.TrimSpace(c.Query("sortBy"))
	pageData, err := courseservice.ListCatalog(courseservice.CatalogQuery{
		Keyword:    keyword,
		Department: department,
		TermCode:   termCode,
		Campus:     campus,
		Instructor: instructor,
		HasReview:  onlyWithReviews,
		SortBy:     sortBy,
		Page:       page,
		Size:       size,
	})
	if err != nil {
		slog.Error("course_catalog_list_failed", "error", err)
		pageData = courseservice.CatalogPage{List: []courseservice.CourseSummary{}, Page: page, Size: size}
	}
	departments, deptErr := courseservice.ListDepartments()
	if deptErr != nil {
		slog.Error("course_departments_list_failed", "error", deptErr)
		departments = []string{}
	}
	terms, termErr := courseservice.ListTerms()
	if termErr != nil {
		slog.Error("course_terms_list_failed", "error", termErr)
		terms = []courseservice.TermOption{}
	}
	campuses, campusErr := courseservice.ListCampuses()
	if campusErr != nil {
		slog.Error("course_campuses_list_failed", "error", campusErr)
		campuses = []string{}
	}
	nextPage := 0
	if pageData.HasNext {
		nextPage = pageData.Page + 1
	}
	return CourseCatalogProps{
		Query: CourseCatalogQueryPayload{
			Keyword:    keyword,
			Department: department,
			TermCode:   termCode,
			Campus:     campus,
			Instructor: instructor,
			HasReview:  onlyWithReviews,
			SortBy:     sortBy,
			Page:       pageData.Page,
			Size:       pageData.Size,
		},
		Courses:     pageData.List,
		Departments: departments,
		Terms:       terms,
		Campuses:    campuses,
		Pagination: PaginationPayload{
			Page:     pageData.Page,
			NextPage: nextPage,
			HasNext:  pageData.HasNext,
			NextURL:  buildCourseListURL(c, nextPage, pageData.Size),
		},
	}
}

func buildCourseDetailProps(detail courseservice.CourseDetail) CourseDetailProps {
	return CourseDetailProps{Course: detail}
}

func buildCourseListURL(c *gin.Context, page, size int) string {
	if page <= 0 {
		return ""
	}
	params := url.Values{}
	params.Set("page", strconv.Itoa(page))
	// 用 service 归一化后的 size 生成链接，避免越界值（如 size=100）被原样回显，
	// 导致下一页请求落到未归一化的分页参数上（重复条目/后续页不可达）。
	params.Set("size", strconv.Itoa(size))
	if v := strings.TrimSpace(c.Query("keyword")); v != "" {
		params.Set("keyword", v)
	}
	if v := strings.TrimSpace(c.Query("department")); v != "" {
		params.Set("department", v)
	}
	if v := strings.TrimSpace(c.Query("term")); v != "" {
		params.Set("term", v)
	}
	if v := strings.TrimSpace(c.Query("campus")); v != "" {
		params.Set("campus", v)
	}
	if v := strings.TrimSpace(c.Query("instructor")); v != "" {
		params.Set("instructor", v)
	}
	if parseBoolLike(c.Query("onlyWithReviews")) {
		params.Set("onlyWithReviews", "1")
	}
	if v := strings.TrimSpace(c.Query("sortBy")); v != "" {
		params.Set("sortBy", v)
	}
	return "/courses?" + params.Encode()
}

// parseBoolLike 解析布尔 query 参数：1/true/on（大小写不敏感）视为 true，其余视为 false。
// 前端/JSON API 可能以不同形式传 onlyWithReviews，统一在此归一化。
func parseBoolLike(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "on":
		return true
	default:
		return false
	}
}

func buildCourseMeta(c *gin.Context) PageMeta {
	lang := requestLang(c)
	return PageMeta{
		Title:       pageTitle(i18n.T(lang, "course")),
		Description: i18n.T(lang, "meta.courseDesc"),
		Canonical:   component.GetBaseUri(c) + c.Request.URL.Path,
	}
}
