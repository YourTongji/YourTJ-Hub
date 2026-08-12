package forum

import "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/courseservice"

// CourseCatalogProps 课程目录页 props（对应 course.index）。
type CourseCatalogProps struct {
	Query      CourseCatalogQueryPayload     `json:"query"`
	Courses    []courseservice.CourseSummary `json:"courses"`
	Pagination PaginationPayload             `json:"pagination"`
}

// CourseCatalogQueryPayload 课程目录页查询条件回显。
type CourseCatalogQueryPayload struct {
	Keyword    string `json:"keyword,omitempty"`
	Department string `json:"department,omitempty"`
	TermCode   string `json:"term,omitempty"`
	Campus     string `json:"campus,omitempty"`
	Page       int    `json:"page"`
	Size       int    `json:"size"`
}

// CourseDetailProps 课程详情页 props（对应 course.detail）。
type CourseDetailProps struct {
	Course courseservice.CourseDetail `json:"course"`
}
