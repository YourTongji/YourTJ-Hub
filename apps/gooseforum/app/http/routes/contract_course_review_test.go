package routes

import (
	"encoding/json"
	"net/http"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/ratelimit"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/forum"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/middleware"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/course"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/moderationLog"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/optRecord"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/reports"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/rolePermissionRs"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/permission"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// setupCourseReviewContractTest builds a router with the 10 course review endpoints
// registered exactly like route4api.go, over the shared in-memory test database.
func setupCourseReviewContractTest(t *testing.T) (*gorm.DB, *gin.Engine) {
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
		&reports.Entity{},
		&rolePermissionRs.Entity{},
		&moderationLog.Entity{},
		&optRecord.Entity{},
		&taskQueue.Entity{},
	); err != nil {
		t.Fatalf("migrate course review contract tables: %v", err)
	}
	// 清空课评域表，保证各用例 fixture 断言确定性。
	for _, model := range []any{
		&course.HelpfulEntity{},
		&course.ReviewEntity{},
		&course.OfferingEntity{},
		&course.TermEntity{},
		&course.Entity{},
		&course.CourseStatsEntity{},
		&course.OfferingStatsEntity{},
		&reports.Entity{},
	} {
		if err := conn.Unscoped().Where("1 = 1").Delete(model).Error; err != nil {
			t.Fatalf("clean course review tables: %v", err)
		}
	}

	router := gin.New()
	forumAPI := router.Group("/api/forum")
	forumAPI.GET("courses/:courseId/reviews", middleware.JWTAuth, UpUriQueryReq(forum.ListCourseReviews))
	forumLoginAPI := forumAPI.Use(middleware.JWTAuthCheck)
	forumLoginAPI.POST("course-reviews", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitReviewWrite), UpJsonReq(forum.CreateCourseReview))
	forumLoginAPI.PATCH("course-reviews/:reviewId", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitReviewWrite), UpUriJsonReq(forum.UpdateCourseReview))
	forumLoginAPI.DELETE("course-reviews/:reviewId", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitReviewWrite), UpUriReq(forum.DeleteCourseReview))
	forumLoginAPI.PUT("course-reviews/:reviewId/helpful", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitReviewHelpful), UpUriReq(forum.MarkReviewHelpful))
	forumLoginAPI.DELETE("course-reviews/:reviewId/helpful", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitReviewHelpful), UpUriReq(forum.UnmarkReviewHelpful))
	forumLoginAPI.POST("course-reviews/:reviewId/reports", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitReviewReport), UpUriJsonReq(forum.ReportCourseReview))
	forumLoginAPI.POST("moderation/course-review-status", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitReviewModerate), UpButterReq(forum.ModerationCourseReviewStatus))
	forumLoginAPI.POST("moderation/course-review-reports", middleware.NoUpdateUserActivity, middleware.RateLimit(middleware.RateLimitReviewModerate), UpButterReq(forum.ModerationCourseReviewReportList))
	forumLoginAPI.POST("moderation/course-review-reveal", middleware.CheckWritableAccount, UpButterReq(forum.ModerationCourseReviewReveal))
	return conn, router
}

// seedCourseReviewCatalog 写入 fixture 依赖的课程目录（course 42 / term 101 / 指定 offering）。
func seedCourseReviewCatalog(t *testing.T, conn *gorm.DB, offeringID uint64) {
	t.Helper()
	if err := conn.Create(&course.Entity{
		Id:             42,
		PrimaryCode:    "100001",
		Name:           "高等数学(A)上",
		Department:     "数学科学学院",
		CreditX10:      50,
		NormalizedName: "高等数学a上",
		NamePinyin:     "gaodengshuxueashang",
		NameInitials:   "gdsxas",
		Status:         course.StatusVisible,
	}).Error; err != nil {
		t.Fatalf("create contract course: %v", err)
	}
	if err := conn.Create(&course.TermEntity{Id: 101, Code: "2025-2026-1", Name: "2025-2026 第一学期", Status: 0}).Error; err != nil {
		t.Fatalf("create contract term: %v", err)
	}
	if err := conn.Create(&course.OfferingEntity{Id: offeringID, CourseId: 42, TermId: 101, Status: course.OfferingStatusVisible}).Error; err != nil {
		t.Fatalf("create contract offering: %v", err)
	}
}

// seedCourseReview 写入一条评价；显式主键与时间字段保证 fixture 断言确定性。
func seedCourseReview(t *testing.T, conn *gorm.DB, id, offeringID, authorID uint64, rating *int, content string, isAnonymous bool, source string, status int8) {
	t.Helper()
	entity := &course.ReviewEntity{
		Id:           id,
		OfferingId:   offeringID,
		AuthorUserId: authorID,
		Rating:       rating,
		Content:      content,
		IsAnonymous:  isAnonymous,
		Source:       source,
		Status:       status,
	}
	if err := conn.Create(entity).Error; err != nil {
		t.Fatalf("create contract review %d: %v", id, err)
	}
}

// grantContractPermission 为用户创建独立角色并授予权限（角色 ID 每次唯一，避免权限缓存串扰）。
func grantContractPermission(t *testing.T, conn *gorm.DB, userID uint64, perm permission.Enum) {
	t.Helper()
	roleID := contractTestID()
	if err := conn.Model(&users.EntityComplete{}).Where("id = ?", userID).Update("role_id", roleID).Error; err != nil {
		t.Fatalf("grant role %d to user %d: %v", roleID, userID, err)
	}
	if err := conn.Create(&rolePermissionRs.Entity{RoleId: roleID, PermissionId: perm.Id()}).Error; err != nil {
		t.Fatalf("grant permission %d to role %d: %v", perm.Id(), roleID, err)
	}
}

func intPtr(v int) *int { return &v }

func sortedReviewKeys(item map[string]any) string {
	keys := make([]string, 0, len(item))
	for key := range item {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return strings.Join(keys, ",")
}

// assertNoReviewIdentityKeys 递归断言评价 DTO 中不出现任何作者身份字段。
func assertNoReviewIdentityKeys(t *testing.T, value any) {
	t.Helper()
	switch v := value.(type) {
	case map[string]any:
		for key, child := range v {
			switch key {
			case "userId", "authorUserId", "username", "nickname", "avatar", "avatarUrl":
				t.Fatalf("review payload leaks identity key %q", key)
			}
			assertNoReviewIdentityKeys(t, child)
		}
	case []any:
		for _, child := range v {
			assertNoReviewIdentityKeys(t, child)
		}
	}
}

func isRFC3339(raw string) bool {
	_, err := time.Parse(time.RFC3339, raw)
	return err == nil
}

// assertReviewItemShape 校验单条评价的结构与 fixture 一致（时间戳只校验 RFC3339 可解析）。
func assertReviewItemShape(t *testing.T, actual, fixture map[string]any) {
	t.Helper()
	if got, want := sortedReviewKeys(actual), "author,content,contentHtml,createdAt,helpfulCount,id,offeringId,rating,updatedAt,viewer"; got != want {
		t.Fatalf("review keys = %s, want %s", got, want)
	}
	for _, key := range []string{"id", "offeringId", "rating", "content", "contentHtml", "helpfulCount"} {
		if !reflect.DeepEqual(actual[key], fixture[key]) {
			t.Fatalf("review %s = %#v, want fixture %#v", key, actual[key], fixture[key])
		}
	}
	for _, key := range []string{"createdAt", "updatedAt"} {
		raw, ok := actual[key].(string)
		if !ok || !isRFC3339(raw) {
			t.Fatalf("review %s = %#v, want RFC3339 string", key, actual[key])
		}
	}
	if !reflect.DeepEqual(actual["author"], fixture["author"]) {
		t.Fatalf("review author = %#v, want fixture %#v", actual["author"], fixture["author"])
	}
	if !reflect.DeepEqual(actual["viewer"], fixture["viewer"]) {
		t.Fatalf("review viewer = %#v, want fixture %#v", actual["viewer"], fixture["viewer"])
	}
}

// assertReviewWriteShape 校验写评/改评成功响应的结构并返回 result。
func assertReviewWriteShape(t *testing.T, response contractEnvelope, wantRating float64, wantContentHTML string) map[string]any {
	t.Helper()
	if response.Code != 0 {
		t.Fatalf("review write code = %d, want 0", response.Code)
	}
	var item map[string]any
	if err := json.Unmarshal(response.Result, &item); err != nil {
		t.Fatalf("decode review write result %q: %v", response.Result, err)
	}
	if got, want := sortedReviewKeys(item), "author,content,contentHtml,createdAt,helpfulCount,id,offeringId,rating,updatedAt,viewer"; got != want {
		t.Fatalf("review keys = %s, want %s", got, want)
	}
	assertNoReviewIdentityKeys(t, item)
	if item["rating"] != wantRating {
		t.Fatalf("review rating = %#v, want %v", item["rating"], wantRating)
	}
	if item["contentHtml"] != wantContentHTML {
		t.Fatalf("review contentHtml = %#v, want %q", item["contentHtml"], wantContentHTML)
	}
	for _, key := range []string{"createdAt", "updatedAt"} {
		raw, ok := item[key].(string)
		if !ok || !isRFC3339(raw) {
			t.Fatalf("review %s = %#v, want RFC3339 string", key, item[key])
		}
	}
	return item
}

func TestCourseReviewListHTTPContract(t *testing.T) {
	conn, router := setupCourseReviewContractTest(t)
	seedCourseReviewCatalog(t, conn, 901)
	bob := createHTTPContractUser(t, conn, contractTestID())
	if err := conn.Model(bob).Update("username", "bob").Error; err != nil {
		t.Fatalf("rename contract bob: %v", err)
	}
	seedCourseReview(t, conn, 3, 901, 0, nil, "历史评价", false, course.ReviewSourceLegacyImport, course.ReviewStatusVisible)
	seedCourseReview(t, conn, 2, 901, 424242, intPtr(4), "匿名评价", true, "", course.ReviewStatusVisible)
	seedCourseReview(t, conn, 1, 901, bob.Id, intPtr(5), "好课", false, "", course.ReviewStatusVisible)

	rec := serveAuthSecurityJSON(router, http.MethodGet, "/api/forum/courses/42/reviews", "", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("course review list status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	response := decodeContractEnvelope(t, rec)
	if response.Code != 0 {
		t.Fatalf("course review list code = %d, want 0: %s", response.Code, rec.Body.String())
	}
	var items []map[string]any
	if err := json.Unmarshal(response.Result, &items); err != nil {
		t.Fatalf("decode course review list result %q: %v", response.Result, err)
	}
	if len(items) != 3 {
		t.Fatalf("course review list length = %d, want 3", len(items))
	}
	for i, wantID := range []float64{3, 2, 1} {
		if items[i]["id"] != wantID {
			t.Fatalf("review[%d].id = %#v, want %v", i, items[i]["id"], wantID)
		}
		assertNoReviewIdentityKeys(t, items[i])
	}
	fixture := contractFixture(t, "course-reviews-list-success.json")
	var fixtureItems []map[string]any
	if err := json.Unmarshal(fixture.Result, &fixtureItems); err != nil {
		t.Fatalf("decode course review list fixture %q: %v", fixture.Result, err)
	}
	for i := range items {
		assertReviewItemShape(t, items[i], fixtureItems[i])
	}
}

func TestCourseReviewListMalformedCourseIDHTTPContract(t *testing.T) {
	_, router := setupCourseReviewContractTest(t)
	rec := serveAuthSecurityJSON(router, http.MethodGet, "/api/forum/courses/not-a-number/reviews", "", "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("malformed course id status = %d, want 400: %s", rec.Code, rec.Body.String())
	}
	assertFixtureEnvelope(t, decodeContractEnvelope(t, rec), contractFixture(t, "course-parse-failed.json"))
}

func TestCourseReviewCreateHTTPContract(t *testing.T) {
	conn, router := setupCourseReviewContractTest(t)
	seedCourseReviewCatalog(t, conn, 902)
	alice := createHTTPContractUser(t, conn, contractTestID())
	token := contractSessionToken(t, alice)

	t.Run("success returns the review payload without identity fields", func(t *testing.T) {
		ratelimit.Default().ResetAll()
		rec := serveAuthSecurityJSON(router, http.MethodPost, "/api/forum/course-reviews",
			`{"offeringId":902,"rating":5,"content":"好课"}`, token)
		if rec.Code != http.StatusOK {
			t.Fatalf("create review status = %d, want 200: %s", rec.Code, rec.Body.String())
		}
		item := assertReviewWriteShape(t, decodeContractEnvelope(t, rec), 5, "<p>好课</p>\n")
		author := item["author"].(map[string]any)
		if author["kind"] != "member" || author["label"] != alice.Username {
			t.Fatalf("review author = %#v, want member %q", author, alice.Username)
		}
		viewer := item["viewer"].(map[string]any)
		if viewer["canEdit"] != true || viewer["canDelete"] != true || viewer["isHelpful"] != false {
			t.Fatalf("review viewer = %#v, want own-editable", viewer)
		}
	})

	t.Run("duplicate review for the same offering returns 409", func(t *testing.T) {
		ratelimit.Default().ResetAll()
		rec := serveAuthSecurityJSON(router, http.MethodPost, "/api/forum/course-reviews",
			`{"offeringId":902,"rating":5,"content":"好课"}`, token)
		if rec.Code != http.StatusConflict {
			t.Fatalf("duplicate review status = %d, want 409: %s", rec.Code, rec.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, rec), contractFixture(t, "course-review-duplicate.json"))
	})

	t.Run("invalid rating stays a legacy HTTP 200 validation failure", func(t *testing.T) {
		ratelimit.Default().ResetAll()
		rec := serveAuthSecurityJSON(router, http.MethodPost, "/api/forum/course-reviews",
			`{"offeringId":902,"rating":0,"content":"好课"}`, token)
		if rec.Code != http.StatusOK {
			t.Fatalf("invalid rating status = %d, want 200: %s", rec.Code, rec.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, rec), contractFixture(t, "course-review-invalid-params.json"))
	})

	t.Run("unknown offering returns 404", func(t *testing.T) {
		ratelimit.Default().ResetAll()
		rec := serveAuthSecurityJSON(router, http.MethodPost, "/api/forum/course-reviews",
			`{"offeringId":99999,"rating":5,"content":"好课"}`, token)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("unknown offering status = %d, want 404: %s", rec.Code, rec.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, rec), contractFixture(t, "course-review-offering-not-found.json"))
	})

	t.Run("missing session returns 401", func(t *testing.T) {
		ratelimit.Default().ResetAll()
		rec := serveAuthSecurityJSON(router, http.MethodPost, "/api/forum/course-reviews",
			`{"offeringId":902,"rating":5,"content":"好课"}`, "")
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("unauthenticated create status = %d, want 401: %s", rec.Code, rec.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, rec), contractFixture(t, "course-review-unauthenticated.json"))
	})
}

func TestCourseReviewUpdateDeleteHTTPContract(t *testing.T) {
	conn, router := setupCourseReviewContractTest(t)
	seedCourseReviewCatalog(t, conn, 902)
	alice := createHTTPContractUser(t, conn, contractTestID())
	bob := createHTTPContractUser(t, conn, contractTestID())
	aliceToken := contractSessionToken(t, alice)
	bobToken := contractSessionToken(t, bob)
	seedCourseReview(t, conn, 300, 902, alice.Id, intPtr(5), "好课", false, "", course.ReviewStatusVisible)

	t.Run("updating someone else's review returns 403", func(t *testing.T) {
		ratelimit.Default().ResetAll()
		rec := serveAuthSecurityJSON(router, http.MethodPatch, "/api/forum/course-reviews/300", `{"rating":4}`, bobToken)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("not-owned update status = %d, want 403: %s", rec.Code, rec.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, rec), contractFixture(t, "course-review-not-owned.json"))
	})

	t.Run("author update succeeds", func(t *testing.T) {
		ratelimit.Default().ResetAll()
		rec := serveAuthSecurityJSON(router, http.MethodPatch, "/api/forum/course-reviews/300", `{"rating":4,"content":"更新后的评价"}`, aliceToken)
		if rec.Code != http.StatusOK {
			t.Fatalf("author update status = %d, want 200: %s", rec.Code, rec.Body.String())
		}
		item := assertReviewWriteShape(t, decodeContractEnvelope(t, rec), 4, "<p>更新后的评价</p>\n")
		viewer := item["viewer"].(map[string]any)
		if viewer["canEdit"] != true || viewer["canDelete"] != true {
			t.Fatalf("review viewer = %#v, want own-editable", viewer)
		}
	})

	t.Run("rating outside 1..5 returns 400", func(t *testing.T) {
		ratelimit.Default().ResetAll()
		rec := serveAuthSecurityJSON(router, http.MethodPatch, "/api/forum/course-reviews/300", `{"rating":6}`, aliceToken)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("out-of-range rating status = %d, want 400: %s", rec.Code, rec.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, rec), contractFixture(t, "course-review-rating-invalid.json"))
	})

	t.Run("PATCH omitting content preserves the existing body", func(t *testing.T) {
		ratelimit.Default().ResetAll()
		// 上一条 subtest 已把正文改为"更新后的评价"；本次只改 rating，正文必须保留。
		rec := serveAuthSecurityJSON(router, http.MethodPatch, "/api/forum/course-reviews/300", `{"rating":3}`, aliceToken)
		if rec.Code != http.StatusOK {
			t.Fatalf("rating-only PATCH status = %d, want 200: %s", rec.Code, rec.Body.String())
		}
		assertReviewWriteShape(t, decodeContractEnvelope(t, rec), 3, "<p>更新后的评价</p>\n")
	})

	t.Run("PATCH with explicit empty content clears the body", func(t *testing.T) {
		ratelimit.Default().ResetAll()
		rec := serveAuthSecurityJSON(router, http.MethodPatch, "/api/forum/course-reviews/300", `{"content":""}`, aliceToken)
		if rec.Code != http.StatusOK {
			t.Fatalf("empty-content PATCH status = %d, want 200: %s", rec.Code, rec.Body.String())
		}
		var item map[string]any
		if err := json.Unmarshal(decodeContractEnvelope(t, rec).Result, &item); err != nil {
			t.Fatalf("decode empty-content result %q: %v", decodeContractEnvelope(t, rec).Result, err)
		}
		if item["contentHtml"] != "" {
			t.Fatalf("explicit empty content must clear body, got contentHtml=%#v", item["contentHtml"])
		}
	})

	t.Run("unknown review returns 404", func(t *testing.T) {
		ratelimit.Default().ResetAll()
		rec := serveAuthSecurityJSON(router, http.MethodPatch, "/api/forum/course-reviews/99999", `{"rating":4}`, aliceToken)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("unknown review update status = %d, want 404: %s", rec.Code, rec.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, rec), contractFixture(t, "course-review-not-found.json"))
	})

	t.Run("deleting someone else's review returns 403", func(t *testing.T) {
		ratelimit.Default().ResetAll()
		rec := serveAuthSecurityJSON(router, http.MethodDelete, "/api/forum/course-reviews/300", "", bobToken)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("not-owned delete status = %d, want 403: %s", rec.Code, rec.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, rec), contractFixture(t, "course-review-not-owned.json"))
	})

	t.Run("author delete succeeds, stays idempotent, and hides the review from the public list", func(t *testing.T) {
		ratelimit.Default().ResetAll()
		rec := serveAuthSecurityJSON(router, http.MethodDelete, "/api/forum/course-reviews/300", "", aliceToken)
		if rec.Code != http.StatusOK {
			t.Fatalf("author delete status = %d, want 200: %s", rec.Code, rec.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, rec), contractFixture(t, "course-review-delete-success.json"))
		again := serveAuthSecurityJSON(router, http.MethodDelete, "/api/forum/course-reviews/300", "", aliceToken)
		if again.Code != http.StatusOK {
			t.Fatalf("idempotent delete status = %d, want 200: %s", again.Code, again.Body.String())
		}
		list := serveAuthSecurityJSON(router, http.MethodGet, "/api/forum/courses/42/reviews?offeringId=902", "", "")
		if list.Code != http.StatusOK {
			t.Fatalf("post-delete list status = %d, want 200: %s", list.Code, list.Body.String())
		}
		var items []map[string]any
		if err := json.Unmarshal(decodeContractEnvelope(t, list).Result, &items); err != nil {
			t.Fatalf("decode post-delete list result: %v", err)
		}
		if len(items) != 0 {
			t.Fatalf("deleted review still listed: %#v", items)
		}
	})
}

func TestCourseReviewHelpfulHTTPContract(t *testing.T) {
	conn, router := setupCourseReviewContractTest(t)
	seedCourseReviewCatalog(t, conn, 902)
	alice := createHTTPContractUser(t, conn, contractTestID())
	bob := createHTTPContractUser(t, conn, contractTestID())
	bobToken := contractSessionToken(t, bob)
	seedCourseReview(t, conn, 301, 902, alice.Id, intPtr(5), "好课", false, "", course.ReviewStatusVisible)

	t.Run("mark helpful succeeds and is idempotent", func(t *testing.T) {
		for attempt := 0; attempt < 2; attempt++ {
			rec := serveAuthSecurityJSON(router, http.MethodPut, "/api/forum/course-reviews/301/helpful", "", bobToken)
			if rec.Code != http.StatusOK {
				t.Fatalf("mark helpful attempt %d status = %d, want 200: %s", attempt+1, rec.Code, rec.Body.String())
			}
			assertFixtureEnvelope(t, decodeContractEnvelope(t, rec), contractFixture(t, "course-review-helpful-success.json"))
		}
	})

	t.Run("viewer sees isHelpful and the count while marked", func(t *testing.T) {
		rec := serveAuthSecurityJSON(router, http.MethodGet, "/api/forum/courses/42/reviews?offeringId=902", "", bobToken)
		if rec.Code != http.StatusOK {
			t.Fatalf("marked list status = %d, want 200: %s", rec.Code, rec.Body.String())
		}
		var items []map[string]any
		if err := json.Unmarshal(decodeContractEnvelope(t, rec).Result, &items); err != nil {
			t.Fatalf("decode marked list result: %v", err)
		}
		if len(items) != 1 || items[0]["id"] != float64(301) {
			t.Fatalf("marked list = %#v, want exactly review 301", items)
		}
		viewer := items[0]["viewer"].(map[string]any)
		if viewer["isHelpful"] != true {
			t.Fatalf("viewer.isHelpful = %#v, want true", viewer["isHelpful"])
		}
		if items[0]["helpfulCount"] != float64(1) {
			t.Fatalf("helpfulCount = %#v, want 1", items[0]["helpfulCount"])
		}
	})

	t.Run("unmark succeeds and clears the viewer state", func(t *testing.T) {
		rec := serveAuthSecurityJSON(router, http.MethodDelete, "/api/forum/course-reviews/301/helpful", "", bobToken)
		if rec.Code != http.StatusOK {
			t.Fatalf("unmark status = %d, want 200: %s", rec.Code, rec.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, rec), contractFixture(t, "course-review-unhelpful-success.json"))
		list := serveAuthSecurityJSON(router, http.MethodGet, "/api/forum/courses/42/reviews?offeringId=902", "", bobToken)
		if list.Code != http.StatusOK {
			t.Fatalf("unmarked list status = %d, want 200: %s", list.Code, list.Body.String())
		}
		var items []map[string]any
		if err := json.Unmarshal(decodeContractEnvelope(t, list).Result, &items); err != nil {
			t.Fatalf("decode unmarked list result: %v", err)
		}
		if len(items) != 1 {
			t.Fatalf("unmarked list = %#v, want exactly one review", items)
		}
		viewer := items[0]["viewer"].(map[string]any)
		if viewer["isHelpful"] != false {
			t.Fatalf("viewer.isHelpful = %#v, want false", viewer["isHelpful"])
		}
		if items[0]["helpfulCount"] != float64(0) {
			t.Fatalf("helpfulCount = %#v, want 0", items[0]["helpfulCount"])
		}
	})

	t.Run("unknown review returns 404", func(t *testing.T) {
		rec := serveAuthSecurityJSON(router, http.MethodPut, "/api/forum/course-reviews/99999/helpful", "", bobToken)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("unknown helpful status = %d, want 404: %s", rec.Code, rec.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, rec), contractFixture(t, "course-review-not-found.json"))
	})
}

func TestCourseReviewReportHTTPContract(t *testing.T) {
	conn, router := setupCourseReviewContractTest(t)
	seedCourseReviewCatalog(t, conn, 902)
	alice := createHTTPContractUser(t, conn, contractTestID())
	bob := createHTTPContractUser(t, conn, contractTestID())
	aliceToken := contractSessionToken(t, alice)
	bobToken := contractSessionToken(t, bob)
	seedCourseReview(t, conn, 302, 902, alice.Id, intPtr(5), "好课", false, "", course.ReviewStatusVisible)

	t.Run("report succeeds", func(t *testing.T) {
		rec := serveAuthSecurityJSON(router, http.MethodPost, "/api/forum/course-reviews/302/reports",
			`{"reason":"spam","note":"广告内容"}`, bobToken)
		if rec.Code != http.StatusOK {
			t.Fatalf("report status = %d, want 200: %s", rec.Code, rec.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, rec), contractFixture(t, "course-review-report-success.json"))
	})

	t.Run("duplicate open report returns report.duplicate", func(t *testing.T) {
		rec := serveAuthSecurityJSON(router, http.MethodPost, "/api/forum/course-reviews/302/reports",
			`{"reason":"spam"}`, bobToken)
		if rec.Code != http.StatusOK {
			t.Fatalf("duplicate report status = %d, want 200: %s", rec.Code, rec.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, rec), contractFixture(t, "course-review-report-duplicate.json"))
	})

	t.Run("reporting own review returns report.ownContent", func(t *testing.T) {
		rec := serveAuthSecurityJSON(router, http.MethodPost, "/api/forum/course-reviews/302/reports",
			`{"reason":"spam"}`, aliceToken)
		if rec.Code != http.StatusOK {
			t.Fatalf("own-content report status = %d, want 200: %s", rec.Code, rec.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, rec), contractFixture(t, "course-review-report-own-content.json"))
	})

	t.Run("unknown review returns report.targetInvalid", func(t *testing.T) {
		rec := serveAuthSecurityJSON(router, http.MethodPost, "/api/forum/course-reviews/99999/reports",
			`{"reason":"spam"}`, bobToken)
		if rec.Code != http.StatusOK {
			t.Fatalf("target-invalid report status = %d, want 200: %s", rec.Code, rec.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, rec), contractFixture(t, "course-review-report-target-invalid.json"))
	})
}

func TestCourseReviewModerationHTTPContract(t *testing.T) {
	conn, router := setupCourseReviewContractTest(t)
	seedCourseReviewCatalog(t, conn, 902)
	alice := createHTTPContractUser(t, conn, contractTestID())
	if err := conn.Model(alice).Update("nickname", "爱丽丝").Error; err != nil {
		t.Fatalf("set contract alice nickname: %v", err)
	}
	regular := createHTTPContractUser(t, conn, contractTestID())
	manager := createHTTPContractUser(t, conn, contractTestID())
	admin := createHTTPContractUser(t, conn, contractTestID())
	grantContractPermission(t, conn, manager.Id, permission.CourseManager)
	grantContractPermission(t, conn, admin.Id, permission.Admin)
	regularToken := contractSessionToken(t, regular)
	managerToken := contractSessionToken(t, manager)
	adminToken := contractSessionToken(t, admin)
	seedCourseReview(t, conn, 303, 902, alice.Id, intPtr(5), "历史评价", true, "", course.ReviewStatusVisible)

	t.Run("regular user is denied for status changes", func(t *testing.T) {
		rec := serveAuthSecurityJSON(router, http.MethodPost, "/api/forum/moderation/course-review-status",
			`{"reviewId":303,"action":"hide"}`, regularToken)
		if rec.Code != http.StatusOK {
			t.Fatalf("denied status change code = %d, want 200: %s", rec.Code, rec.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, rec), contractFixture(t, "course-review-permission-denied.json"))
	})

	t.Run("course manager hides and shows the review", func(t *testing.T) {
		hide := serveAuthSecurityJSON(router, http.MethodPost, "/api/forum/moderation/course-review-status",
			`{"reviewId":303,"action":"hide"}`, managerToken)
		if hide.Code != http.StatusOK {
			t.Fatalf("hide status = %d, want 200: %s", hide.Code, hide.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, hide), contractFixture(t, "course-review-moderation-status-success.json"))
		list := serveAuthSecurityJSON(router, http.MethodGet, "/api/forum/courses/42/reviews?offeringId=902", "", "")
		if list.Code != http.StatusOK {
			t.Fatalf("hidden list status = %d, want 200: %s", list.Code, list.Body.String())
		}
		var hiddenItems []map[string]any
		if err := json.Unmarshal(decodeContractEnvelope(t, list).Result, &hiddenItems); err != nil {
			t.Fatalf("decode hidden list result: %v", err)
		}
		if len(hiddenItems) != 0 {
			t.Fatalf("hidden review still listed: %#v", hiddenItems)
		}

		show := serveAuthSecurityJSON(router, http.MethodPost, "/api/forum/moderation/course-review-status",
			`{"reviewId":303,"action":"show"}`, managerToken)
		if show.Code != http.StatusOK {
			t.Fatalf("show status = %d, want 200: %s", show.Code, show.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, show), contractFixture(t, "course-review-moderation-status-success.json"))
		listed := serveAuthSecurityJSON(router, http.MethodGet, "/api/forum/courses/42/reviews?offeringId=902", "", "")
		if listed.Code != http.StatusOK {
			t.Fatalf("shown list status = %d, want 200: %s", listed.Code, listed.Body.String())
		}
		var shownItems []map[string]any
		if err := json.Unmarshal(decodeContractEnvelope(t, listed).Result, &shownItems); err != nil {
			t.Fatalf("decode shown list result: %v", err)
		}
		if len(shownItems) != 1 || shownItems[0]["id"] != float64(303) {
			t.Fatalf("shown list = %#v, want exactly review 303", shownItems)
		}
	})

	t.Run("reveal is admin-only", func(t *testing.T) {
		rec := serveAuthSecurityJSON(router, http.MethodPost, "/api/forum/moderation/course-review-reveal",
			`{"reviewId":303,"reason":"取证"}`, regularToken)
		if rec.Code != http.StatusOK {
			t.Fatalf("denied reveal code = %d, want 200: %s", rec.Code, rec.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, rec), contractFixture(t, "course-review-permission-denied.json"))
	})

	t.Run("admin reveal returns the author identity", func(t *testing.T) {
		rec := serveAuthSecurityJSON(router, http.MethodPost, "/api/forum/moderation/course-review-reveal",
			`{"reviewId":303,"reason":"取证"}`, adminToken)
		if rec.Code != http.StatusOK {
			t.Fatalf("reveal status = %d, want 200: %s", rec.Code, rec.Body.String())
		}
		response := decodeContractEnvelope(t, rec)
		if response.Code != 0 {
			t.Fatalf("reveal code = %d, want 0: %s", response.Code, rec.Body.String())
		}
		var reveal map[string]any
		if err := json.Unmarshal(response.Result, &reveal); err != nil {
			t.Fatalf("decode reveal result %q: %v", response.Result, err)
		}
		if got, want := sortedReviewKeys(reveal), "authorUserId,isAnonymous,nickname,reviewId,source,username"; got != want {
			t.Fatalf("reveal keys = %s, want %s", got, want)
		}
		if reveal["reviewId"] != float64(303) {
			t.Fatalf("reveal.reviewId = %#v, want 303", reveal["reviewId"])
		}
		if reveal["authorUserId"] != float64(alice.Id) {
			t.Fatalf("reveal.authorUserId = %#v, want %d", reveal["authorUserId"], alice.Id)
		}
		if reveal["username"] != alice.Username {
			t.Fatalf("reveal.username = %#v, want %q", reveal["username"], alice.Username)
		}
		if reveal["nickname"] != "爱丽丝" {
			t.Fatalf("reveal.nickname = %#v, want 爱丽丝", reveal["nickname"])
		}
		if reveal["isAnonymous"] != true || reveal["source"] != "" {
			t.Fatalf("reveal flags = %#v, want isAnonymous=true source=", reveal)
		}
	})

	t.Run("admin reveal of an unknown review returns review.notFound", func(t *testing.T) {
		rec := serveAuthSecurityJSON(router, http.MethodPost, "/api/forum/moderation/course-review-reveal",
			`{"reviewId":99999,"reason":"取证"}`, adminToken)
		if rec.Code != http.StatusOK {
			t.Fatalf("unknown reveal code = %d, want 200: %s", rec.Code, rec.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, rec), contractFixture(t, "course-review-not-found.json"))
	})
}

func TestCourseReviewModerationReportListHTTPContract(t *testing.T) {
	conn, router := setupCourseReviewContractTest(t)
	seedCourseReviewCatalog(t, conn, 902)
	charlie := createHTTPContractUser(t, conn, 7001)
	if err := conn.Model(charlie).Updates(map[string]any{"username": "charlie", "avatar_url": "/static/pic/1.webp"}).Error; err != nil {
		t.Fatalf("set contract charlie profile: %v", err)
	}
	manager := createHTTPContractUser(t, conn, contractTestID())
	grantContractPermission(t, conn, manager.Id, permission.CourseManager)
	managerToken := contractSessionToken(t, manager)
	regular := createHTTPContractUser(t, conn, contractTestID())
	regularToken := contractSessionToken(t, regular)
	seedCourseReview(t, conn, 303, 902, 0, nil, "历史评价", false, course.ReviewSourceLegacyImport, course.ReviewStatusVisible)
	if err := conn.Create(&reports.Entity{
		Id:         5001,
		TargetType: reports.TargetCourseReview,
		TargetId:   303,
		ReporterId: charlie.Id,
		Reason:     "spam",
		Note:       "广告内容",
		Status:     reports.StatusOpen,
		CreatedAt:  time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC),
	}).Error; err != nil {
		t.Fatalf("create contract report: %v", err)
	}

	t.Run("course manager sees the report queue", func(t *testing.T) {
		rec := serveAuthSecurityJSON(router, http.MethodPost, "/api/forum/moderation/course-review-reports",
			`{"status":"open","pageSize":10}`, managerToken)
		if rec.Code != http.StatusOK {
			t.Fatalf("report list status = %d, want 200: %s", rec.Code, rec.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, rec), contractFixture(t, "course-review-moderation-reports-success.json"))
	})

	t.Run("regular user is denied", func(t *testing.T) {
		rec := serveAuthSecurityJSON(router, http.MethodPost, "/api/forum/moderation/course-review-reports",
			`{"status":"open","pageSize":10}`, regularToken)
		if rec.Code != http.StatusOK {
			t.Fatalf("denied report list code = %d, want 200: %s", rec.Code, rec.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, rec), contractFixture(t, "course-review-permission-denied.json"))
	})
}

// TestCourseReviewLegacyHelpfulCountHTTPContract 历史导入评价的 legacy_helpful_count 计入公开
// helpfulCount（原生 helpful 为 0 时直接展示源数据计数）。
func TestCourseReviewLegacyHelpfulCountHTTPContract(t *testing.T) {
	conn, router := setupCourseReviewContractTest(t)
	seedCourseReviewCatalog(t, conn, 902)
	if err := conn.Create(&course.ReviewEntity{
		Id:                 500,
		OfferingId:         902,
		AuthorUserId:       0,
		Content:            "历史评价",
		IsAnonymous:        true,
		Status:             course.ReviewStatusVisible,
		Source:             course.ReviewSourceLegacyImport,
		LegacyHelpfulCount: 7,
	}).Error; err != nil {
		t.Fatalf("create legacy review: %v", err)
	}
	rec := serveAuthSecurityJSON(router, http.MethodGet, "/api/forum/courses/42/reviews?offeringId=902", "", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("list status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	var items []map[string]any
	if err := json.Unmarshal(decodeContractEnvelope(t, rec).Result, &items); err != nil {
		t.Fatalf("decode list result: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 review, got %d", len(items))
	}
	if items[0]["helpfulCount"] != float64(7) {
		t.Fatalf("helpfulCount = %#v, want 7 (legacy count exposed)", items[0]["helpfulCount"])
	}
}

// TestCourseReviewModerationRateLimit 验收 1（issue #176 B4）：course.review.moderate
// 限流（60s 窗口 per-User 30）——同一管理账号第 31 次审核操作返回 429 + Retry-After。
// 注意：hotdataserve 的限流配置缓存来自 ratelimit.json 默认值，测试用同一账号
// 触发 user 维度（per-IP 60 未超）。
func TestCourseReviewModerationRateLimit(t *testing.T) {
	conn, router := setupCourseReviewContractTest(t)
	seedCourseReviewCatalog(t, conn, 902)
	manager := createHTTPContractUser(t, conn, contractTestID())
	grantContractPermission(t, conn, manager.Id, permission.CourseManager)
	token := contractSessionToken(t, manager)
	seedCourseReview(t, conn, 304, 902, 1, intPtr(5), "限流目标", false, "", course.ReviewStatusVisible)

	const moderateUserLimit = 30
	for i := 0; i < moderateUserLimit; i++ {
		rec := serveAuthSecurityJSON(router, http.MethodPost, "/api/forum/moderation/course-review-status",
			`{"reviewId":304,"action":"hide"}`, token)
		if rec.Code != http.StatusOK {
			t.Fatalf("hit #%d status = %d, want 200 within limit: %s", i+1, rec.Code, rec.Body.String())
		}
		// 连续 30 次 hide（幂等操作，仅用于触发 per-User 限流计数；
		// 不交替 show，避免引入不必要的状态变化）
	}
	// 第 31 次：429 + Retry-After
	rec := serveAuthSecurityJSON(router, http.MethodPost, "/api/forum/moderation/course-review-status",
		`{"reviewId":304,"action":"hide"}`, token)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("31st status = %d, want 429: %s", rec.Code, rec.Body.String())
	}
	if retry := rec.Header().Get("Retry-After"); retry == "" || retry == "0" {
		t.Fatalf("Retry-After header = %q, want positive integer", retry)
	}
}

// TestCourseReviewListOfferingOwnership404 验收 2（issue #176 B4）：
// 跨课程 offering 与隐藏 offering 的评价列表返回 404（offering 归属校验）。
func TestCourseReviewListOfferingOwnership404(t *testing.T) {
	conn, router := setupCourseReviewContractTest(t)
	seedCourseReviewCatalog(t, conn, 902)
	// 造另一个课程 43 的 offering 903（不属于 course 42），以及隐藏的 offering 904
	if err := conn.Create(&course.Entity{
		Id: 43, PrimaryCode: "100002", Name: "线性代数", Department: "数学科学学院",
		Status: course.StatusVisible,
	}).Error; err != nil {
		t.Fatalf("create course 43: %v", err)
	}
	if err := conn.Create(&course.OfferingEntity{Id: 903, CourseId: 43, TermId: 101, Status: course.OfferingStatusVisible}).Error; err != nil {
		t.Fatalf("create cross-course offering: %v", err)
	}
	if err := conn.Create(&course.OfferingEntity{Id: 904, CourseId: 42, TermId: 101, Status: course.OfferingStatusHidden}).Error; err != nil {
		t.Fatalf("create hidden offering: %v", err)
	}

	cases := []struct {
		name string
		path string
	}{
		{"cross-course offering", "/api/forum/courses/42/reviews?offeringId=903"},
		{"hidden offering", "/api/forum/courses/42/reviews?offeringId=904"},
		{"unknown offering", "/api/forum/courses/42/reviews?offeringId=99999"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := serveAuthSecurityJSON(router, http.MethodGet, tc.path, "", "")
			if rec.Code != http.StatusNotFound {
				t.Fatalf("%s status = %d, want 404: %s", tc.name, rec.Code, rec.Body.String())
			}
			assertFixtureEnvelope(t, decodeContractEnvelope(t, rec), contractFixture(t, "course-review-offering-not-found.json"))
		})
	}
}
