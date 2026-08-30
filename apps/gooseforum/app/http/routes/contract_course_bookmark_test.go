package routes

import (
	"net/http"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/forum"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/middleware"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/course"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// setupCourseBookmarkContractTest 迁移课程收藏契约测试表并挂载与生产一致的路由链
// （forumLoginApi：JWTAuthCheck + CheckWritableAccount + RateLimitCourseBookmark）。
func setupCourseBookmarkContractTest(t *testing.T) (*gorm.DB, *gin.Engine) {
	t.Helper()
	conn, router := setupHTTPContractTest(t)
	if err := conn.AutoMigrate(
		&course.Entity{},
		&course.CourseUserActionEntity{},
	); err != nil {
		t.Fatalf("migrate course bookmark contract tables: %v", err)
	}
	for _, m := range []any{
		&course.CourseUserActionEntity{},
		&course.Entity{},
	} {
		if err := conn.Unscoped().Where("1 = 1").Delete(m).Error; err != nil {
			t.Fatalf("clean course bookmark tables: %v", err)
		}
	}
	forumAPI := router.Group("/api/forum")
	forumLoginAPI := forumAPI.Use(middleware.JWTAuthCheck)
	forumLoginAPI.POST("courses/bookmark", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitCourseBookmark), UpButterReq(forum.BookmarkCourse))
	return conn, router
}

func seedBookmarkCourse(t *testing.T, conn *gorm.DB) {
	t.Helper()
	if err := conn.Create(&course.Entity{
		Id:             42,
		PrimaryCode:    "100001",
		Name:           "高等数学(A)上",
		Department:     "数学科学学院",
		NormalizedName: "高等数学a上",
		Status:         course.StatusVisible,
	}).Error; err != nil {
		t.Fatalf("create bookmark course: %v", err)
	}
}

// TestCourseBookmarkAddHTTPContract 收藏动作 1 → 200 result=true，并落库。
func TestCourseBookmarkAddHTTPContract(t *testing.T) {
	conn, router := setupCourseBookmarkContractTest(t)
	seedBookmarkCourse(t, conn)
	user := createHTTPContractUser(t, conn, contractTestID())
	token := contractSessionToken(t, user)

	rec := serveJSON(router, "/api/forum/courses/bookmark", `{"courseId":42,"action":1}`, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("course bookmark add status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	assertFixtureEnvelope(t, decodeContractEnvelope(t, rec), contractFixture(t, "result-true.json"))

	state := course.GetCourseUserAction(user.Id, 42)
	if state.Id == 0 || state.BookmarkedAt == nil {
		t.Fatalf("expected course 42 bookmarked for user %d", user.Id)
	}
}

// TestCourseBookmarkRemoveHTTPContract 收藏后取消（action 2）幂等落库清除。
func TestCourseBookmarkRemoveHTTPContract(t *testing.T) {
	conn, router := setupCourseBookmarkContractTest(t)
	seedBookmarkCourse(t, conn)
	user := createHTTPContractUser(t, conn, contractTestID())
	token := contractSessionToken(t, user)

	if !course.SetCourseBookmarked(user.Id, 42, true) {
		t.Fatal("seed bookmark failed")
	}
	rec := serveJSON(router, "/api/forum/courses/bookmark", `{"courseId":42,"action":2}`, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("course bookmark remove status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	assertFixtureEnvelope(t, decodeContractEnvelope(t, rec), contractFixture(t, "result-true.json"))
	state := course.GetCourseUserAction(user.Id, 42)
	if state.BookmarkedAt != nil {
		t.Fatalf("expected course 42 unbookmarked for user %d", user.Id)
	}
}

// TestCourseBookmarkNotFoundHTTPContract 收藏隐藏/不存在课程 → 404。
func TestCourseBookmarkNotFoundHTTPContract(t *testing.T) {
	conn, router := setupCourseBookmarkContractTest(t)
	seedBookmarkCourse(t, conn)
	user := createHTTPContractUser(t, conn, contractTestID())
	token := contractSessionToken(t, user)

	rec := serveJSON(router, "/api/forum/courses/bookmark", `{"courseId":99999,"action":1}`, token)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("course bookmark not found status = %d, want 404: %s", rec.Code, rec.Body.String())
	}
	assertFixtureEnvelope(t, decodeContractEnvelope(t, rec), contractFixture(t, "course-bookmark-not-found.json"))
}

// TestCourseBookmarkUnauthenticatedHTTPContract 未登录 → 401。
func TestCourseBookmarkUnauthenticatedHTTPContract(t *testing.T) {
	conn, router := setupCourseBookmarkContractTest(t)
	seedBookmarkCourse(t, conn)

	rec := serveJSON(router, "/api/forum/courses/bookmark", `{"courseId":42,"action":1}`, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("course bookmark unauthenticated status = %d, want 401: %s", rec.Code, rec.Body.String())
	}
	assertFixtureEnvelope(t, decodeContractEnvelope(t, rec), contractFixture(t, "auth-required.json"))
}
