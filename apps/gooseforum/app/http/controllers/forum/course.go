package forum

import (
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/leancodebox/GooseForum/app/bundles/i18n"
	"github.com/leancodebox/GooseForum/app/http/controllers/component"
	"github.com/leancodebox/GooseForum/app/service/courseservice"
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
		renderNotFoundWithMessage(c, component.MessagePageNotFound)
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
type CourseListReq struct {
	Keyword    string `form:"keyword"`
	Department string `form:"department"`
	TermCode   string `form:"term"`
	Campus     string `form:"campus"`
	Page       int    `form:"page"`
	Size       int    `form:"size"`
}

// CourseListJSON 课程目录 JSON API（公开只读，供前端异步加载与后续移动端使用）。
func CourseListJSON(req component.BetterRequest[CourseListReq]) component.Response {
	pageData, err := courseservice.ListCatalog(courseservice.CatalogQuery{
		Keyword:    strings.TrimSpace(req.Params.Keyword),
		Department: strings.TrimSpace(req.Params.Department),
		TermCode:   strings.TrimSpace(req.Params.TermCode),
		Campus:     strings.TrimSpace(req.Params.Campus),
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
		return component.BuildResponse(http.StatusNotFound,
			component.FailDataCode(component.MessagePageNotFound, nil))
	}
	return component.SuccessResponse(detail)
}

// buildCourseCatalogProps 构建课程目录 SSR props；分页与分页回显以 service 归一化后的结果为准。
func buildCourseCatalogProps(c *gin.Context, page, size int) CourseCatalogProps {
	keyword := strings.TrimSpace(c.Query("keyword"))
	department := strings.TrimSpace(c.Query("department"))
	termCode := strings.TrimSpace(c.Query("term"))
	campus := strings.TrimSpace(c.Query("campus"))
	pageData, err := courseservice.ListCatalog(courseservice.CatalogQuery{
		Keyword:    keyword,
		Department: department,
		TermCode:   termCode,
		Campus:     campus,
		Page:       page,
		Size:       size,
	})
	if err != nil {
		slog.Error("course_catalog_list_failed", "error", err)
		pageData = courseservice.CatalogPage{List: []courseservice.CourseSummary{}, Page: page, Size: size}
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
			Page:       pageData.Page,
			Size:       pageData.Size,
		},
		Courses: pageData.List,
		Pagination: PaginationPayload{
			Page:     pageData.Page,
			NextPage: nextPage,
			HasNext:  pageData.HasNext,
			NextURL:  buildCourseListURL(c, nextPage),
		},
	}
}

func buildCourseDetailProps(detail courseservice.CourseDetail) CourseDetailProps {
	return CourseDetailProps{Course: detail}
}

func buildCourseListURL(c *gin.Context, page int) string {
	if page <= 0 {
		return ""
	}
	params := url.Values{}
	params.Set("page", strconv.Itoa(page))
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
	return "/courses?" + params.Encode()
}

func buildCourseMeta(c *gin.Context) PageMeta {
	lang := requestLang(c)
	return PageMeta{
		Title:       pageTitle(i18n.T(lang, "course")),
		Description: i18n.T(lang, "meta.courseDesc"),
		Canonical:   component.GetBaseUri(c) + c.Request.URL.Path,
	}
}
