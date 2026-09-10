package forum

import "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/courseservice"

// CourseCatalogProps 课程目录页 props（对应 course.index）。
type CourseCatalogProps struct {
	Query       CourseCatalogQueryPayload     `json:"query"`
	Courses     []courseservice.CourseSummary `json:"courses"`
	Departments []string                      `json:"departments"`
	// Terms 可筛选学期（value=code，label 优先学期名，按 starts_on 倒序），与 term 筛选值域一致。
	Terms []courseservice.TermOption `json:"terms"`
	// Campuses 可筛选校区（course_offering.campus 原始值，按字典序），与 campus 筛选值域一致。
	Campuses []string `json:"campuses"`
	// BookmarkedCourseIDs 当前登录用户已收藏的课程 id（issue #331 R5，表格收藏状态）；
	// 未登录/无收藏时省略或为空数组。
	BookmarkedCourseIDs []uint64          `json:"bookmarkedCourseIDs,omitempty"`
	Pagination          PaginationPayload `json:"pagination"`
}

// CourseCatalogQueryPayload 课程目录页查询条件回显（Department/TermCode/Campus/Instructor 为多值数组）。
type CourseCatalogQueryPayload struct {
	Keyword    string   `json:"keyword,omitempty"`
	Department []string `json:"department,omitempty"`
	TermCode   []string `json:"term,omitempty"`
	Campus     []string `json:"campus,omitempty"`
	Instructor []string `json:"instructor,omitempty"`
	HasReview  bool     `json:"onlyWithReviews,omitempty"`
	SortBy     string   `json:"sortBy,omitempty"`
	Page       int      `json:"page"`
	Size       int      `json:"size"`
}

// CourseDetailProps 课程详情页 props（对应 course.detail）。
type CourseDetailProps struct {
	Course courseservice.CourseDetail `json:"course"`
}
