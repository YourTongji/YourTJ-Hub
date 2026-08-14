package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/forum"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/middleware"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/course"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/optRecord"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/rolePermissionRs"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/permission"
	"gorm.io/gorm"
)

// setupCourseManageContractTest 注册课程管理端点并迁移 course 域 + 权限/审计表。
func setupCourseManageContractTest(t *testing.T) (*gorm.DB, *gin.Engine) {
	t.Helper()
	conn, _ := setupHTTPContractTest(t)
	if err := conn.AutoMigrate(
		&course.Entity{},
		&course.TermEntity{},
		&course.OfferingEntity{},
		&course.ReviewEntity{},
		&course.HelpfulEntity{},
		&course.CourseStatsEntity{},
		&course.OfferingStatsEntity{},
		&course.AliasEntity{},
		&course.InstructorEntity{},
		&course.OfferingInstructorEntity{},
		&course.SourceRefEntity{},
		&course.CourseAiSummaryEntity{},
		&rolePermissionRs.Entity{},
		&optRecord.Entity{},
		&taskQueue.Entity{},
	); err != nil {
		t.Fatalf("migrate course manage tables: %v", err)
	}
	for _, model := range []any{
		&course.HelpfulEntity{},
		&course.ReviewEntity{},
		&course.OfferingInstructorEntity{},
		&course.InstructorEntity{},
		&course.OfferingEntity{},
		&course.TermEntity{},
		&course.AliasEntity{},
		&course.SourceRefEntity{},
		&course.CourseStatsEntity{},
		&course.OfferingStatsEntity{},
		&course.Entity{},
		&rolePermissionRs.Entity{},
		&taskQueue.Entity{},
	} {
		if err := conn.Unscoped().Where("1 = 1").Delete(model).Error; err != nil {
			t.Fatalf("clean course manage tables: %v", err)
		}
	}

	router := gin.New()
	forumLoginAPI := router.Group("/api/forum").Use(middleware.JWTAuthCheck)
	// 与 route4api.go 一致：页面路由带 CheckLogin 中间件，测试用 JWTAuthCheck 等价替代。
	forumLoginAPI.GET("moderation/course-reviews", forum.CourseReviewModeration)
	forumLoginAPI.GET("moderation/courses", forum.CourseManagement)
	forumLoginAPI.POST("moderation/course-list", middleware.NoUpdateUserActivity, UpButterReq(forum.AdminCourseList))
	forumLoginAPI.POST("moderation/course-create", middleware.CheckWritableAccount, UpButterReq(forum.AdminCourseCreate))
	forumLoginAPI.POST("moderation/course-update", middleware.CheckWritableAccount, UpButterReq(forum.AdminCourseUpdate))
	forumLoginAPI.POST("moderation/course-delete", middleware.CheckWritableAccount, UpButterReq(forum.AdminCourseDelete))
	forumLoginAPI.POST("moderation/course-review-list", middleware.NoUpdateUserActivity, UpButterReq(forum.AdminReviewList))
	forumLoginAPI.POST("moderation/course-review-edit", middleware.CheckWritableAccount, UpButterReq(forum.AdminReviewUpdate))
	forumLoginAPI.POST("moderation/course-review-delete", middleware.CheckWritableAccount, UpButterReq(forum.AdminReviewDelete))
	return conn, router
}

// TestCourseManagePermissionDenied 非 CourseManager/Admin 访问管理端点返回 permission.denied（语义统一 403）。
func TestCourseManagePermissionDenied(t *testing.T) {
	conn, router := setupCourseManageContractTest(t)
	user := createHTTPContractUser(t, conn, contractTestID())
	token := contractSessionToken(t, user)

	recorder := serveAuthSecurityJSON(router, http.MethodPost, "/api/forum/moderation/course-list", `{"keyword":"","page":1,"pageSize":20}`, token)
	envelope := decodeContractEnvelope(t, recorder)
	if envelope.Code != 1 || envelope.MessageCode != "permission.denied" {
		t.Fatalf("expected permission.denied, got code=%d messageCode=%q", envelope.Code, envelope.MessageCode)
	}
}

// TestCourseModerationPagesSidebarActiveKey 回归 #237：课评审核与课程管理 SSR 页面
// 下发的 layout.sidebar.activeKey 必须与前端 AppShell.vue 侧边栏菜单 key 一致
// （courseReviews / courseManage），否则高亮会错位到「课程」（courses）。
func TestCourseModerationPagesSidebarActiveKey(t *testing.T) {
	conn, router := setupCourseManageContractTest(t)
	manager := createHTTPContractUser(t, conn, contractTestID())
	grantContractPermission(t, conn, manager.Id, permission.CourseManager)
	token := contractSessionToken(t, manager)

	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "course reviews", path: "/api/forum/moderation/course-reviews", want: "courseReviews"},
		{name: "course manage", path: "/api/forum/moderation/courses", want: "courseManage"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("X-Goose-Page", "true")
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, req)
			if recorder.Code != http.StatusOK {
				t.Fatalf("GET %s status = %d, want 200: %s", tt.path, recorder.Code, recorder.Body.String())
			}
			var payload struct {
				Layout struct {
					Sidebar struct {
						ActiveKey string `json:"activeKey"`
					} `json:"sidebar"`
				} `json:"layout"`
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
				t.Fatalf("decode page payload %q: %v", recorder.Body.String(), err)
			}
			if payload.Layout.Sidebar.ActiveKey != tt.want {
				t.Fatalf("GET %s layout.sidebar.activeKey = %q, want %q（与 AppShell.vue 菜单 key 一致）",
					tt.path, payload.Layout.Sidebar.ActiveKey, tt.want)
			}
		})
	}
}

// TestCourseManageListAndCreate CourseManager 能列出并新增课程（含审计日志）。
func TestCourseManageListAndCreate(t *testing.T) {
	conn, router := setupCourseManageContractTest(t)
	manager := createHTTPContractUser(t, conn, contractTestID())
	grantContractPermission(t, conn, manager.Id, permission.CourseManager)
	token := contractSessionToken(t, manager)

	// 初始为空列表
	recorder := serveAuthSecurityJSON(router, http.MethodPost, "/api/forum/moderation/course-list", `{"keyword":"","page":1,"pageSize":20}`, token)
	envelope := decodeContractEnvelope(t, recorder)
	if envelope.Code != 0 {
		t.Fatalf("list courses code = %d, want 0", envelope.Code)
	}

	// 新增课程
	recorder = serveAuthSecurityJSON(router, http.MethodPost, "/api/forum/moderation/course-create", `{"primaryCode":"CS101","name":"数据结构","department":"计算机","creditX10":30}`, token)
	envelope = decodeContractEnvelope(t, recorder)
	if envelope.Code != 0 {
		t.Fatalf("create course code = %d, messageCode=%q", envelope.Code, envelope.MessageCode)
	}
	var created struct {
		Id uint64 `json:"id"`
	}
	if err := json.Unmarshal(envelope.Result, &created); err != nil || created.Id == 0 {
		t.Fatalf("decode created course %q: %v", envelope.Result, err)
	}

	// 列表能看到新增课程
	recorder = serveAuthSecurityJSON(router, http.MethodPost, "/api/forum/moderation/course-list", `{"keyword":"数据结构","page":1,"pageSize":20}`, token)
	envelope = decodeContractEnvelope(t, recorder)
	var list struct {
		Total int64 `json:"total"`
	}
	if err := json.Unmarshal(envelope.Result, &list); err != nil || list.Total != 1 {
		t.Fatalf("list total = %+v, want 1 (%v)", list, err)
	}

	// 审计日志写入（opt_record 有一条 course.created）
	var optCount int64
	if err := conn.Model(&optRecord.Entity{}).Where("opt_user_id = ?", manager.Id).Count(&optCount).Error; err != nil {
		t.Fatalf("count opt records: %v", err)
	}
	if optCount == 0 {
		t.Fatal("expected course.created audit log")
	}
}

// TestCourseManageReviewWriteEndpoints 管理端评价编辑/删除端点的成功与 404 路径
// （含 courseManageErrorResponse 的 review.notFound 语义映射）。
func TestCourseManageReviewWriteEndpoints(t *testing.T) {
	conn, router := setupCourseManageContractTest(t)
	manager := createHTTPContractUser(t, conn, contractTestID())
	grantContractPermission(t, conn, manager.Id, permission.CourseManager)
	token := contractSessionToken(t, manager)

	// 准备一个可见评价
	c := course.Entity{PrimaryCode: "CS101", Name: "数据结构", Status: course.StatusVisible}
	if err := conn.Create(&c).Error; err != nil {
		t.Fatalf("create course: %v", err)
	}
	term := course.TermEntity{Code: "2025-2026-1", Name: "t", Status: 0}
	if err := conn.Create(&term).Error; err != nil {
		t.Fatalf("create term: %v", err)
	}
	offering := course.OfferingEntity{CourseId: c.Id, TermId: term.Id, Status: course.OfferingStatusVisible}
	if err := conn.Create(&offering).Error; err != nil {
		t.Fatalf("create offering: %v", err)
	}
	rating := 3
	review := course.ReviewEntity{OfferingId: offering.Id, AuthorUserId: uint64Ptr(1001), Rating: &rating, Content: "内容", Status: course.ReviewStatusVisible}
	if err := conn.Create(&review).Error; err != nil {
		t.Fatalf("create review: %v", err)
	}

	// 编辑评分成功
	editBody, _ := json.Marshal(map[string]any{"reviewId": review.Id, "rating": 5})
	recorder := serveAuthSecurityJSON(router, http.MethodPost, "/api/forum/moderation/course-review-edit", string(editBody), token)
	if recorder.Code != http.StatusOK {
		t.Fatalf("edit review status = %d, want 200", recorder.Code)
	}
	if env := decodeContractEnvelope(t, recorder); env.Code != 0 {
		t.Fatalf("edit review code = %d, messageCode=%q", env.Code, env.MessageCode)
	}

	// 编辑不存在的评价 → 404 review.notFound
	recorder = serveAuthSecurityJSON(router, http.MethodPost, "/api/forum/moderation/course-review-edit", `{"reviewId":999999,"rating":5}`, token)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("edit missing review status = %d, want 404", recorder.Code)
	}
	if env := decodeContractEnvelope(t, recorder); env.MessageCode != "review.notFound" {
		t.Fatalf("edit missing review messageCode = %q, want review.notFound", env.MessageCode)
	}

	// 删除不存在的评价 → 404 review.notFound
	recorder = serveAuthSecurityJSON(router, http.MethodPost, "/api/forum/moderation/course-review-delete", `{"reviewId":999999}`, token)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("delete missing review status = %d, want 404", recorder.Code)
	}
	if env := decodeContractEnvelope(t, recorder); env.MessageCode != "review.notFound" {
		t.Fatalf("delete missing review messageCode = %q, want review.notFound", env.MessageCode)
	}

	// 删除现有评价成功
	deleteBody, _ := json.Marshal(map[string]any{"reviewId": review.Id})
	recorder = serveAuthSecurityJSON(router, http.MethodPost, "/api/forum/moderation/course-review-delete", string(deleteBody), token)
	if recorder.Code != http.StatusOK {
		t.Fatalf("delete review status = %d, want 200", recorder.Code)
	}
	if env := decodeContractEnvelope(t, recorder); env.Code != 0 {
		t.Fatalf("delete review code = %d, messageCode=%q", env.Code, env.MessageCode)
	}

	// 审计日志：编辑与删除各写一条（review.updated / review.deleted）。
	for _, code := range []string{"review.updated", "review.deleted"} {
		var cnt int64
		if err := conn.Model(&optRecord.Entity{}).Where("opt_user_id = ? AND opt_info LIKE ?", manager.Id, "%"+code+"%").Count(&cnt).Error; err != nil {
			t.Fatalf("count %s audit: %v", code, err)
		}
		if cnt == 0 {
			t.Fatalf("expected %s audit log", code)
		}
	}
}

// TestCourseManageReviewWritePermissionDenied 非 CourseManager 访问评价写端点返回 permission.denied。
func TestCourseManageReviewWritePermissionDenied(t *testing.T) {
	conn, router := setupCourseManageContractTest(t)
	user := createHTTPContractUser(t, conn, contractTestID())
	token := contractSessionToken(t, user)

	for _, path := range []string{
		"/api/forum/moderation/course-review-edit",
		"/api/forum/moderation/course-review-delete",
	} {
		body := `{"reviewId":1}`
		if path == "/api/forum/moderation/course-review-edit" {
			body = `{"reviewId":1,"rating":5}`
		}
		recorder := serveAuthSecurityJSON(router, http.MethodPost, path, body, token)
		envelope := decodeContractEnvelope(t, recorder)
		if envelope.Code != 1 || envelope.MessageCode != "permission.denied" {
			t.Fatalf("%s expected permission.denied, got code=%d messageCode=%q", path, envelope.Code, envelope.MessageCode)
		}
	}
}
