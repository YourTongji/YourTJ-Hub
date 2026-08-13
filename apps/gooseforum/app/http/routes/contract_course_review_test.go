package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
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
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/courseservice"
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
	forumLoginAPI.POST("moderation/course-review-status", middleware.CheckWritableAccount, UpButterReq(forum.ModerationCourseReviewStatus))
	forumLoginAPI.POST("moderation/course-review-reports", middleware.NoUpdateUserActivity, UpButterReq(forum.ModerationCourseReviewReportList))
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

// decodeReviewPageList 解析评价列表分页响应的 list 数组（B2 分页对象结构）。
func decodeReviewPageList(t *testing.T, response contractEnvelope) []map[string]any {
	t.Helper()
	var page struct {
		List []map[string]any `json:"list"`
	}
	if err := json.Unmarshal(response.Result, &page); err != nil {
		t.Fatalf("decode review page list %q: %v", response.Result, err)
	}
	return page.List
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
	var page struct {
		List  []map[string]any `json:"list"`
		Total float64          `json:"total"`
	}
	if err := json.Unmarshal(response.Result, &page); err != nil {
		t.Fatalf("decode course review list result %q: %v", response.Result, err)
	}
	if page.Total != 3 {
		t.Fatalf("course review list total = %v, want 3", page.Total)
	}
	items := page.List
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
	var fixturePage struct {
		List []map[string]any `json:"list"`
	}
	if err := json.Unmarshal(fixture.Result, &fixturePage); err != nil {
		t.Fatalf("decode course review list fixture %q: %v", fixture.Result, err)
	}
	fixtureItems := fixturePage.List
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

// TestCourseReviewOfferingStatsHTTPContract 验证 spec-reviewer N1（PR #195）：
// reviews?offeringId= 响应的 offeringRatingAvg / offeringReviewCount 统计字段有契约断言锁定。
//  1. offering 901：3 条评价（5 星 + 4 星 + NULL rating）→ offeringRatingAvg=4.5
//     （非 NULL 均分）、offeringReviewCount=3（含 NULL 行，与 reviewCount 语义一致）；
//  2. offering 902：仅 1 条 NULL 评分评价 → offeringRatingAvg 省略（omitempty），
//     offeringReviewCount=1（无评分评价仍计入评论数）。
func TestCourseReviewOfferingStatsHTTPContract(t *testing.T) {
	conn, router := setupCourseReviewContractTest(t)
	seedCourseReviewCatalog(t, conn, 901)
	// 追加第二个 offering（seedCourseReviewCatalog 固定 course/term 主键，不能重复调用）
	if err := conn.Create(&course.OfferingEntity{
		Id: 902, CourseId: 42, TermId: 101, Status: course.OfferingStatusVisible,
	}).Error; err != nil {
		t.Fatalf("create contract offering 902: %v", err)
	}

	// offering 901：rating 5、4、NULL（legacy 无评分）
	seedCourseReview(t, conn, 11, 901, 1, intPtr(5), "五星", false, "contract", course.ReviewStatusVisible)
	seedCourseReview(t, conn, 12, 901, 2, intPtr(4), "四星", false, "contract", course.ReviewStatusVisible)
	seedCourseReview(t, conn, 13, 901, 3, nil, "无评分", false, "contract", course.ReviewStatusVisible)
	// offering 902：仅 1 条 NULL 评分评价
	seedCourseReview(t, conn, 21, 902, 4, nil, "无评分评价", false, "contract", course.ReviewStatusVisible)

	// 同步统计投影（生产路径由 Upsert 维护；测试直接调用重建）
	if err := course.RebuildAllCourseStats(); err != nil {
		t.Fatalf("rebuild stats: %v", err)
	}

	// --- 场景 1：有评分 offering → avg 正确、count 含 NULL 行 ---
	rec := serveAuthSecurityJSON(router, http.MethodGet, "/api/forum/courses/42/reviews?offeringId=901", "", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("review list status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	response := decodeContractEnvelope(t, rec)
	if response.Code != 0 {
		t.Fatalf("review list code = %d, want 0: %s", rec.Code, rec.Body.String())
	}
	items := decodeReviewPageList(t, response)
	if len(items) != 3 {
		t.Fatalf("review list length = %d, want 3", len(items))
	}
	for _, item := range items {
		if got := item["offeringRatingAvg"]; got != 4.5 {
			t.Fatalf("item %v offeringRatingAvg = %#v, want 4.5 (非 NULL 均分)", item["id"], got)
		}
		if got := item["offeringReviewCount"]; got != float64(3) {
			t.Fatalf("item %v offeringReviewCount = %#v, want 3 (含 NULL 行)", item["id"], got)
		}
	}

	// --- 场景 2：无评分评价的 offering → avg 省略（omitempty）、count 仍计数 ---
	rec = serveAuthSecurityJSON(router, http.MethodGet, "/api/forum/courses/42/reviews?offeringId=902", "", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("no-rating review list status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	response = decodeContractEnvelope(t, rec)
	items = decodeReviewPageList(t, response)
	if len(items) != 1 {
		t.Fatalf("no-rating review list length = %d, want 1", len(items))
	}
	if _, ok := items[0]["offeringRatingAvg"]; ok {
		t.Fatalf("item %v offeringRatingAvg present, want omitted (RatingCount=0 omitempty); full item: %#v", items[0]["id"], items[0])
	}
	if got := items[0]["offeringReviewCount"]; got != float64(1) {
		t.Fatalf("item %v offeringReviewCount = %#v, want 1 (无评分评价仍计入)", items[0]["id"], got)
	}
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
		items := decodeReviewPageList(t, decodeContractEnvelope(t, list))
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
		items := decodeReviewPageList(t, decodeContractEnvelope(t, rec))
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
		items := decodeReviewPageList(t, decodeContractEnvelope(t, list))
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
		hiddenItems := decodeReviewPageList(t, decodeContractEnvelope(t, list))
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
		shownItems := decodeReviewPageList(t, decodeContractEnvelope(t, listed))
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
	items := decodeReviewPageList(t, decodeContractEnvelope(t, rec))
	if len(items) != 1 {
		t.Fatalf("expected 1 review, got %d", len(items))
	}
	if items[0]["helpfulCount"] != float64(7) {
		t.Fatalf("helpfulCount = %#v, want 7 (legacy count exposed)", items[0]["helpfulCount"])
	}
}

// TestCourseReviewPaginationHTTPContract 验证 B2 cursor 分页（issue #174 验收）：
//  1. 250 条评价 pageSize=20 → 13 页，无重复无遗漏（集合比对一致）；
//  2. 非法 cursor / pageSize>50 → 400 + common.request.invalidParams；
//  3. 删除一条评价（隔离窗口）后续页 cursor 稳定不跳页。
func TestCourseReviewPaginationHTTPContract(t *testing.T) {
	conn, router := setupCourseReviewContractTest(t)
	seedCourseReviewCatalog(t, conn, 901)

	// 清空并插入 250 条评价（offering 901；id 1..250，按 id DESC 排序）
	// 注意：必须硬删（Unscoped），软删行会与显式主键 Create 冲突。
	conn.Unscoped().Where("1 = 1").Delete(&course.ReviewEntity{})
	// authorID 用唯一值（(offering_id, author_user_id) 唯一约束）
	for i := uint64(1); i <= 250; i++ {
		seedCourseReview(t, conn, i, 901, i, nil, fmt.Sprintf("评价 %d", i), true, course.ReviewSourceLegacyImport, course.ReviewStatusVisible)
	}

	// --- 验收 1：pageSize=20 共 13 页，无重复无遗漏 ---
	seen := map[float64]bool{}
	pageCount := 0
	cursor := ""
	expectTotal := float64(250)
	for {
		path := fmt.Sprintf("/api/forum/courses/42/reviews?pageSize=20&cursor=%s", url.QueryEscape(cursor))
		rec := serveAuthSecurityJSON(router, http.MethodGet, path, "", "")
		if rec.Code != http.StatusOK {
			t.Fatalf("page %d status = %d, want 200: %s", pageCount+1, rec.Code, rec.Body.String())
		}
		response := decodeContractEnvelope(t, rec)
		var page struct {
			List       []map[string]any `json:"list"`
			NextCursor string           `json:"nextCursor"`
			Total      float64          `json:"total"`
		}
		if err := json.Unmarshal(response.Result, &page); err != nil {
			t.Fatalf("decode page %d: %v", pageCount+1, err)
		}
		if page.Total != expectTotal {
			t.Fatalf("page %d total = %v, want %v", pageCount+1, page.Total, expectTotal)
		}
		if len(page.List) > 20 {
			t.Fatalf("page %d length = %d, want <= 20", pageCount+1, len(page.List))
		}
		for _, item := range page.List {
			id := item["id"].(float64)
			if seen[id] {
				t.Fatalf("duplicate review id %v on page %d", id, pageCount+1)
			}
			seen[id] = true
		}
		pageCount++
		if page.NextCursor == "" {
			break
		}
		cursor = page.NextCursor
	}
	if pageCount != 13 {
		t.Fatalf("page count = %d, want 13", pageCount)
	}
	if len(seen) != 250 {
		t.Fatalf("collected %d unique reviews, want 250", len(seen))
	}

	// --- 验收 2：非法 cursor / pageSize 超限 → 400 ---
	rec := serveAuthSecurityJSON(router, http.MethodGet, "/api/forum/courses/42/reviews?cursor=not-a-cursor", "", "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid cursor status = %d, want 400: %s", rec.Code, rec.Body.String())
	}
	assertFixtureEnvelope(t, decodeContractEnvelope(t, rec), contractFixture(t, "course-review-invalid-params.json"))

	rec = serveAuthSecurityJSON(router, http.MethodGet, "/api/forum/courses/42/reviews?pageSize=999", "", "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("oversized pageSize status = %d, want 400: %s", rec.Code, rec.Body.String())
	}
	assertFixtureEnvelope(t, decodeContractEnvelope(t, rec), contractFixture(t, "course-review-invalid-params.json"))

	// --- 验收 3：删除一条（隔离窗口）后翻页不跳页 ---
	// 第一页取前 20 条（id 250..231），删除其中一条（id 240），
	// 从第一页 cursor 继续翻页：后续集合 = 全集 - 已见 - 被删。
	rec = serveAuthSecurityJSON(router, http.MethodGet, "/api/forum/courses/42/reviews?pageSize=20", "", "")
	response := decodeContractEnvelope(t, rec)
	var page struct {
		List       []map[string]any `json:"list"`
		NextCursor string           `json:"nextCursor"`
	}
	_ = json.Unmarshal(response.Result, &page)
	if len(page.List) != 20 {
		t.Fatalf("first page length = %d, want 20", len(page.List))
	}
	firstPageIDs := map[float64]bool{}
	for _, item := range page.List {
		firstPageIDs[item["id"].(float64)] = true
	}
	// 删除 id=240（隔离窗口内；用生产 DeleteReview 路径——status=deleted 软删，
	// 与 gorm 直接软删（deleted_at）语义不同；spec S1）。
	if err := courseservice.DeleteReview(240, 240); err != nil {
		t.Fatalf("delete review 240: %v", err)
	}
	// 从第一页 cursor 继续：只应看到 id < 231 的剩余评价，无跳页无重复
	rec = serveAuthSecurityJSON(router, http.MethodGet,
		"/api/forum/courses/42/reviews?pageSize=20&cursor="+url.QueryEscape(page.NextCursor), "", "")
	response = decodeContractEnvelope(t, rec)
	var page2 struct {
		List       []map[string]any `json:"list"`
		NextCursor string           `json:"nextCursor"`
		Total      float64          `json:"total"`
	}
	_ = json.Unmarshal(response.Result, &page2)
	// total 同步递减（生产删除路径扣减；spec S1：断言 total 口径一致）
	if page2.Total != 249 {
		t.Fatalf("total after delete = %v, want 249 (deleted review excluded)", page2.Total)
	}
	seenAfter := map[float64]bool{}
	for _, item := range page2.List {
		id := item["id"].(float64)
		if firstPageIDs[id] {
			t.Fatalf("review %v reappeared after cursor (skip/jump)", id)
		}
		if seenAfter[id] {
			t.Fatalf("duplicate review %v in page 2", id)
		}
		seenAfter[id] = true
	}
	if len(page2.List) != 20 {
		t.Fatalf("page 2 length = %d, want 20 (no skipped page due to deletion)", len(page2.List))
	}
	for _, item := range page2.List {
		if item["id"].(float64) == 240 {
			t.Fatalf("deleted review 240 should not appear")
		}
	}
}

// TestCourseReviewPaginationHiddenOfferingExcluded 验证 PR #201 security F1：
// 隐藏 offering 后，其评价不出现在课程级分页的 list 与 total 中。
func TestCourseReviewPaginationHiddenOfferingExcluded(t *testing.T) {
	conn, router := setupCourseReviewContractTest(t)
	seedCourseReviewCatalog(t, conn, 901)
	// 第二个 offering（course 42）
	if err := conn.Create(&course.OfferingEntity{Id: 903, CourseId: 42, TermId: 101, Status: course.OfferingStatusVisible}).Error; err != nil {
		t.Fatalf("create offering 903: %v", err)
	}
	conn.Unscoped().Where("1 = 1").Delete(&course.ReviewEntity{})
	seedCourseReview(t, conn, 1, 901, 1, nil, "可见", true, "", course.ReviewStatusVisible)
	seedCourseReview(t, conn, 2, 903, 2, nil, "隐藏开课", true, "", course.ReviewStatusVisible)

	// 初始：2 条（两个 offering 都可见）
	rec := serveAuthSecurityJSON(router, http.MethodGet, "/api/forum/courses/42/reviews?pageSize=50", "", "")
	response := decodeContractEnvelope(t, rec)
	var page struct {
		List  []map[string]any `json:"list"`
		Total float64          `json:"total"`
	}
	_ = json.Unmarshal(response.Result, &page)
	if page.Total != 2 || len(page.List) != 2 {
		t.Fatalf("initial total=%v len=%d, want 2/2", page.Total, len(page.List))
	}

	// 隐藏 offering 903 → 其评价（id=2）从 list + total 消失
	if err := conn.Model(&course.OfferingEntity{}).Where("id = ?", 903).Update("status", course.OfferingStatusHidden).Error; err != nil {
		t.Fatalf("hide offering 903: %v", err)
	}
	rec = serveAuthSecurityJSON(router, http.MethodGet, "/api/forum/courses/42/reviews?pageSize=50", "", "")
	response = decodeContractEnvelope(t, rec)
	_ = json.Unmarshal(response.Result, &page)
	if page.Total != 1 {
		t.Fatalf("total after hide = %v, want 1 (hidden offering excluded)", page.Total)
	}
	if len(page.List) != 1 || page.List[0]["id"] != float64(1) {
		t.Fatalf("list after hide = %#v, want only review 1", page.List)
	}

	// offering 过滤路径仍可读隐藏 offering 前的可见评价？不——offering 过滤需 offering 可见（404）
	rec = serveAuthSecurityJSON(router, http.MethodGet, "/api/forum/courses/42/reviews?offeringId=903", "", "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("hidden offering scoped list status = %d, want 404", rec.Code)
	}
}

// TestCourseReviewPaginationMultiOfferingCursor 验证 PR #201 spec S2：
// 2+ offering 复合游标 (offering_id DESC, id DESC) 交错翻页——
// 元组比较逻辑在跨 offering 场景下无重复无遗漏。
func TestCourseReviewPaginationMultiOfferingCursor(t *testing.T) {
	conn, router := setupCourseReviewContractTest(t)
	seedCourseReviewCatalog(t, conn, 901)
	// 第二个 offering 902（course 42）
	if err := conn.Create(&course.OfferingEntity{Id: 902, CourseId: 42, TermId: 101, Status: course.OfferingStatusVisible}).Error; err != nil {
		t.Fatalf("create offering 902: %v", err)
	}
	conn.Unscoped().Where("1 = 1").Delete(&course.ReviewEntity{})
	// offering 901: id 1..15（新→旧），offering 902: id 101..115
	// 交错后的全局序（offering_id DESC, id DESC）：
	// 902(115..101) → 901(15..1)
	for i := uint64(1); i <= 15; i++ {
		seedCourseReview(t, conn, i, 901, i, nil, fmt.Sprintf("a%d", i), true, "", course.ReviewStatusVisible)
		seedCourseReview(t, conn, 100+i, 902, 200+i, nil, fmt.Sprintf("b%d", i), true, "", course.ReviewStatusVisible)
	}

	// pageSize=10 翻页：全局序 902:115..101, 901:15..1 → 30 条 / 10 = 3 页
	seen := map[float64]bool{}
	cursor := ""
	pages := 0
	for {
		path := fmt.Sprintf("/api/forum/courses/42/reviews?pageSize=10&cursor=%s", url.QueryEscape(cursor))
		rec := serveAuthSecurityJSON(router, http.MethodGet, path, "", "")
		if rec.Code != http.StatusOK {
			t.Fatalf("page %d status = %d: %s", pages+1, rec.Code, rec.Body.String())
		}
		response := decodeContractEnvelope(t, rec)
		var page struct {
			List       []map[string]any `json:"list"`
			NextCursor string           `json:"nextCursor"`
			Total      float64          `json:"total"`
		}
		_ = json.Unmarshal(response.Result, &page)
		if page.Total != 30 {
			t.Fatalf("page %d total = %v, want 30", pages+1, page.Total)
		}
		for _, item := range page.List {
			id := item["id"].(float64)
			if seen[id] {
				t.Fatalf("duplicate review %v across pages", id)
			}
			seen[id] = true
		}
		pages++
		if page.NextCursor == "" {
			break
		}
		cursor = page.NextCursor
	}
	if pages != 3 {
		t.Fatalf("pages = %d, want 3", pages)
	}
	if len(seen) != 30 {
		t.Fatalf("collected %d unique reviews, want 30 (no skip/dup across offerings)", len(seen))
	}
	// 验证全局序首元素为 902 的最大 id（115）
	rec := serveAuthSecurityJSON(router, http.MethodGet, "/api/forum/courses/42/reviews?pageSize=10", "", "")
	response := decodeContractEnvelope(t, rec)
	var first struct {
		List []map[string]any `json:"list"`
	}
	_ = json.Unmarshal(response.Result, &first)
	if len(first.List) == 0 || first.List[0]["id"] != float64(115) {
		t.Fatalf("first item = %#v, want id 115 (offering 902 newest first)", first.List[0])
	}
}

// TestCourseReviewPaginationOfferingStats 验证跨 PR 接线（#195 字段 → #201 分页路径）：
// reviews?offeringId= 分页响应的 ReviewPayload 携带 offeringRatingAvg/
// offeringReviewCount 且值正确（数据来自 offering_review_stats 投影）。
func TestCourseReviewPaginationOfferingStats(t *testing.T) {
	conn, router := setupCourseReviewContractTest(t)
	seedCourseReviewCatalog(t, conn, 901)
	conn.Unscoped().Where("1 = 1").Delete(&course.ReviewEntity{})
	conn.Unscoped().Where("1 = 1").Delete(&course.OfferingStatsEntity{})

	// 3 条评价（5/4/NULL）+ offering 统计投影（2 评分 sum=9 → avg=4.5, reviewCount=3）
	r5, r4 := 5, 4
	seedCourseReview(t, conn, 1, 901, 1, &r5, "五星", true, "", course.ReviewStatusVisible)
	seedCourseReview(t, conn, 2, 901, 2, &r4, "四星", true, "", course.ReviewStatusVisible)
	seedCourseReview(t, conn, 3, 901, 3, nil, "无评分", true, "", course.ReviewStatusVisible)
	if err := conn.Create(&course.OfferingStatsEntity{OfferingId: 901, RatingCount: 2, RatingSum: 9, ReviewCount: 3}).Error; err != nil {
		t.Fatalf("create offering stats: %v", err)
	}

	rec := serveAuthSecurityJSON(router, http.MethodGet, "/api/forum/courses/42/reviews?offeringId=901&pageSize=50", "", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	response := decodeContractEnvelope(t, rec)
	var page struct {
		List []map[string]any `json:"list"`
	}
	_ = json.Unmarshal(response.Result, &page)
	if len(page.List) != 3 {
		t.Fatalf("list length = %d, want 3", len(page.List))
	}
	// 每条 payload 都带 offering 级统计（同 offering 901）
	for i, item := range page.List {
		if got := item["offeringRatingAvg"]; got != 4.5 {
			t.Fatalf("item %d offeringRatingAvg = %#v, want 4.5", i, got)
		}
		if got := item["offeringReviewCount"]; got != float64(3) {
			t.Fatalf("item %d offeringReviewCount = %#v, want 3", i, got)
		}
	}
	// 无 offeringId 过滤（course 级）→ 不填充 offering 统计
	rec = serveAuthSecurityJSON(router, http.MethodGet, "/api/forum/courses/42/reviews?pageSize=50", "", "")
	response = decodeContractEnvelope(t, rec)
	var coursePage struct {
		List []map[string]any `json:"list"`
	}
	_ = json.Unmarshal(response.Result, &coursePage)
	if len(coursePage.List) != 3 {
		t.Fatalf("course list length = %d, want 3", len(coursePage.List))
	}
	for i, item := range coursePage.List {
		if _, present := item["offeringRatingAvg"]; present {
			t.Fatalf("course-level item %d should omit offeringRatingAvg, got %#v", i, item["offeringRatingAvg"])
		}
	}
}
