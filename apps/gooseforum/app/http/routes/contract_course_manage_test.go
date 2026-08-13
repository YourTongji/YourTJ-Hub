package routes

import (
	"encoding/json"
	"net/http"
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
	forumLoginAPI.POST("moderation/course-list", middleware.NoUpdateUserActivity, UpButterReq(forum.AdminCourseList))
	forumLoginAPI.POST("moderation/course-create", middleware.CheckWritableAccount, UpButterReq(forum.AdminCourseCreate))
	forumLoginAPI.POST("moderation/course-update", middleware.CheckWritableAccount, UpButterReq(forum.AdminCourseUpdate))
	forumLoginAPI.POST("moderation/course-delete", middleware.CheckWritableAccount, UpButterReq(forum.AdminCourseDelete))
	forumLoginAPI.POST("moderation/course-review-list", middleware.NoUpdateUserActivity, UpButterReq(forum.AdminReviewList))
	forumLoginAPI.POST("moderation/course-stats-rebuild", middleware.CheckWritableAccount, UpButterReq(forum.AdminCourseStatsRebuild))
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
