package routes

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/api"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/middleware"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/category"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/postUserAction"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/rolePermissionRs"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topicCategoryIndex"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/permission"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// 管理后台内容贡献统计路由（issue #1145）：中间件链与 route4api.go 的生产注册一致
// （JWTAuthCheck + CheckWritableAccount + CheckPermission(Admin)）。
const (
	statsSummaryPath = "/api/admin/stats/category-summary"
	statsTrendPath   = "/api/admin/stats/category-trend"
	statsRankingPath = "/api/admin/stats/user-ranking"
	statsDetailPath  = "/api/admin/stats/user-detail"
)

var statsRouteDay = time.Date(2026, 4, 7, 0, 0, 0, 0, time.Local)

func setupAdminStatsTest(t *testing.T) (*gorm.DB, *gin.Engine) {
	t.Helper()
	conn, router := setupHTTPContractTest(t)
	if err := conn.AutoMigrate(&postUserAction.Entity{}, &rolePermissionRs.Entity{}); err != nil {
		t.Fatalf("migrate admin stats tables: %v", err)
	}
	// 统计是区间聚合读：必须清空同进程其他用例留下的内容行，否则合计无法断言。
	for _, table := range []string{"topics", "posts", "topic_category_index", "post_user_action", "category", "users"} {
		if err := conn.Exec("DELETE FROM " + table).Error; err != nil {
			t.Fatalf("clear %s: %v", table, err)
		}
	}
	adminAPI := router.Group("/api/admin",
		middleware.JWTAuthCheck,
		middleware.CheckWritableAccount,
		middleware.CheckPermission(permission.Admin),
	)
	adminAPI.POST("/stats/category-summary", UpButterReq(api.AdminStatsCategorySummary))
	adminAPI.POST("/stats/category-trend", UpButterReq(api.AdminStatsCategoryTrend))
	adminAPI.POST("/stats/user-ranking", UpButterReq(api.AdminStatsUserRanking))
	adminAPI.POST("/stats/user-detail", UpButterReq(api.AdminStatsUserDetail))
	return conn, router
}

// serveAdminStats 以 Admin 身份调用统计路由，返回解码后的响应信封。
func serveAdminStats(t *testing.T, conn *gorm.DB, router *gin.Engine, path string, body string) contractEnvelope {
	t.Helper()
	admin := createContractAdmin(t, conn)
	recorder := serveJSON(router, path, body, contractSessionToken(t, admin))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", recorder.Code, recorder.Body.String())
	}
	return decodeContractEnvelope(t, recorder)
}

func decodeStatsResult(t *testing.T, envelope contractEnvelope) map[string]any {
	t.Helper()
	if envelope.Code != 0 {
		t.Fatalf("code = %d, want 0: %s", envelope.Code, envelope.MessageCode)
	}
	var result map[string]any
	if err := json.Unmarshal(envelope.Result, &result); err != nil {
		t.Fatalf("decode stats result %q: %v", envelope.Result, err)
	}
	return result
}

func statsFixtureDate(dayOffset int) string {
	return statsRouteDay.AddDate(0, 0, dayOffset).Format("2006-01-02")
}

func seedStatsRouteAdminData(t *testing.T, conn *gorm.DB) {
	t.Helper()
	if err := conn.Create(&category.Entity{Id: 1, Name: "综合", Sort: 1}).Error; err != nil {
		t.Fatalf("create category: %v", err)
	}
	if err := conn.Create(&category.Entity{Id: 2, Name: "学习", Sort: 2}).Error; err != nil {
		t.Fatalf("create category: %v", err)
	}
	createUser := func(id uint64, actorType int8) {
		if err := conn.Create(&users.EntityComplete{
			Id: id, Username: "stats-route-" + string(rune('a'+id)), Nickname: "统计路由用户", ActorType: actorType,
		}).Error; err != nil {
			t.Fatalf("create user %d: %v", id, err)
		}
	}
	createUser(1, users.ActorTypeHuman)
	createUser(2, users.ActorTypeHuman)
	createUser(10, users.ActorTypeBot)

	at := func(hour int) time.Time {
		return statsRouteDay.Add(time.Duration(hour) * time.Hour)
	}
	createTopic := func(id, userId uint64, likeCount uint64) {
		published := at(10)
		if err := conn.Create(&topics.Entity{
			Id: id, Title: "统计路由主题", UserId: userId, Status: 1, LikeCount: likeCount,
			VisibilityStatus: topics.VisibilityActive, RetentionStatus: topics.RetentionNormal,
			FirstPostId: id + 700, PostCount: 1, PostSeq: 1,
			CreatedAt: published, UpdatedAt: published, FirstPublicAt: &published,
		}).Error; err != nil {
			t.Fatalf("create topic %d: %v", id, err)
		}
		if err := conn.Create(&topicCategoryIndex.Entity{TopicId: id, CategoryId: 1, Effective: 1}).Error; err != nil {
			t.Fatalf("index topic %d: %v", id, err)
		}
		if err := conn.Create(&posts.Entity{
			Id: id + 700, TopicId: id, PostNo: 1, UserId: userId, Content: "首楼",
			VisibilityStatus: posts.VisibilityActive, RetentionStatus: posts.RetentionNormal,
			CreatedAt: published, UpdatedAt: published, FirstPublicAt: &published,
		}).Error; err != nil {
			t.Fatalf("create first post %d: %v", id+700, err)
		}
	}
	createTopic(100, 1, 5)
	createTopic(101, 10, 1)

	repliedAt := at(11)
	if err := conn.Create(&posts.Entity{
		Id: 200, TopicId: 100, PostNo: 2, UserId: 2, Content: "回复",
		VisibilityStatus: posts.VisibilityActive, RetentionStatus: posts.RetentionNormal,
		CreatedAt: repliedAt, UpdatedAt: repliedAt, FirstPublicAt: &repliedAt,
	}).Error; err != nil {
		t.Fatalf("create reply: %v", err)
	}
	if err := conn.Create(&postUserAction.Entity{PostId: 200, UserId: 1, LikedAt: &repliedAt}).Error; err != nil {
		t.Fatalf("like reply: %v", err)
	}
}

// TestAdminStatsRoutesGuardScenarios 三条守卫场景与生产中间件链一致：
// 未登录 401 / 冻结账号 403 / 无 Admin 权限 403。
func TestAdminStatsRoutesGuardScenarios(t *testing.T) {
	for _, path := range []string{statsSummaryPath, statsTrendPath, statsRankingPath, statsDetailPath} {
		t.Run(path, func(t *testing.T) {
			t.Run("missing session returns 401", func(t *testing.T) {
				_, router := setupAdminStatsTest(t)
				assertInteractionUnauthenticated(t, router, path, `{}`, "auth-required.json")
			})
			t.Run("frozen account returns 403", func(t *testing.T) {
				conn, router := setupAdminStatsTest(t)
				assertInteractionForbidden(t, conn, router, path, `{}`, "account-frozen.json")
			})
			t.Run("user without Admin permission returns 403", func(t *testing.T) {
				conn, router := setupAdminStatsTest(t)
				assertAdminOpsPermissionDenied(t, conn, router, path, "admin-agent-create-permission-denied.json")
			})
		})
	}
}

// TestAdminStatsCategorySummaryHTTP 断言板块汇总的 JSON 形状（冻结契约 §2.1）。
func TestAdminStatsCategorySummaryHTTP(t *testing.T) {
	conn, router := setupAdminStatsTest(t)
	seedStatsRouteAdminData(t, conn)

	envelope := serveAdminStats(t, conn, router, statsSummaryPath, `{"startDate":"`+statsFixtureDate(0)+`","endDate":"`+statsFixtureDate(0)+`"}`)
	result := decodeStatsResult(t, envelope)

	categories, ok := result["categories"].([]any)
	if !ok || len(categories) != 2 {
		t.Fatalf("categories = %v, want two entries", result["categories"])
	}
	first := categories[0].(map[string]any)
	if first["categoryId"].(float64) != 1 || first["name"].(string) != "综合" {
		t.Fatalf("category[0] = %v", first)
	}
	assertStatsNumber(t, first, "topicCount", 2)
	assertStatsNumber(t, first, "replyCount", 1)
	assertStatsNumber(t, first, "contributorCount", 2)
	assertStatsNumber(t, first, "agentContributorCount", 1)
	second := categories[1].(map[string]any)
	if second["categoryId"].(float64) != 2 {
		t.Fatalf("category[1] = %v, want the empty category listed as 0", second)
	}
	assertStatsNumber(t, second, "topicCount", 0)
	assertStatsNumber(t, second, "replyCount", 0)

	totals := result["totals"].(map[string]any)
	assertStatsNumber(t, totals, "topicCount", 2)
	assertStatsNumber(t, totals, "replyCount", 1)
	assertStatsNumber(t, totals, "contributorCount", 2)
	assertStatsNumber(t, totals, "agentContributorCount", 1)

	coverage := result["coverage"].(map[string]any)
	if coverage["timezone"].(string) == "" {
		t.Fatalf("coverage.timezone empty: %v", coverage)
	}
	assertStatsNumber(t, coverage, "maxRangeDays", 366)
	if _, ok := coverage["estimated"].(bool); !ok {
		t.Fatalf("coverage.estimated = %v, want a boolean", coverage["estimated"])
	}
}

// TestAdminStatsCategoryTrendHTTP 断言板块趋势逐日序列与估算标记（§2.2）。
func TestAdminStatsCategoryTrendHTTP(t *testing.T) {
	conn, router := setupAdminStatsTest(t)
	seedStatsRouteAdminData(t, conn)

	body := `{"startDate":"` + statsFixtureDate(0) + `","endDate":"` + statsFixtureDate(1) + `","categoryId":1}`
	envelope := serveAdminStats(t, conn, router, statsTrendPath, body)
	result := decodeStatsResult(t, envelope)

	if result["categoryId"].(float64) != 1 {
		t.Fatalf("categoryId = %v", result["categoryId"])
	}
	days, ok := result["days"].([]any)
	if !ok || len(days) != 2 {
		t.Fatalf("days = %v, want two entries", result["days"])
	}
	first := days[0].(map[string]any)
	if first["date"].(string) != statsFixtureDate(0) {
		t.Fatalf("days[0].date = %v", first["date"])
	}
	assertStatsNumber(t, first, "topicCount", 2)
	assertStatsNumber(t, first, "replyCount", 1)
	if first["estimated"].(bool) {
		t.Fatalf("days[0].estimated = true, want false for exact first_public_at")
	}
	second := days[1].(map[string]any)
	assertStatsNumber(t, second, "topicCount", 0)
	assertStatsNumber(t, second, "replyCount", 0)
	if second["estimated"].(bool) {
		t.Fatalf("days[1].estimated = true, want false on a quiet day")
	}
}

// TestAdminStatsUserRankingHTTP 断言排行条目、分页元数据与真人/Agent 分列（§2.3）。
func TestAdminStatsUserRankingHTTP(t *testing.T) {
	conn, router := setupAdminStatsTest(t)
	seedStatsRouteAdminData(t, conn)

	body := `{"startDate":"` + statsFixtureDate(0) + `","endDate":"` + statsFixtureDate(0) + `","sortBy":"likes","pageSize":10}`
	envelope := serveAdminStats(t, conn, router, statsRankingPath, body)
	result := decodeStatsResult(t, envelope)

	assertStatsNumber(t, result, "page", 1)
	assertStatsNumber(t, result, "pageSize", 10)
	assertStatsNumber(t, result, "total", 2)
	items, ok := result["items"].([]any)
	if !ok || len(items) != 2 {
		t.Fatalf("items = %v, want two human accounts", result["items"])
	}
	first := items[0].(map[string]any)
	if first["actorType"].(float64) != 0 {
		t.Fatalf("actorType = %v, want 0 for humans", first["actorType"])
	}
	// 获赞排序：账号 1 的主题有 5 个赞，账号 2 的回复有 1 个赞。
	assertStatsNumber(t, first, "topicCount", 1)
	assertStatsNumber(t, first, "replyCount", 0)
	assertStatsNumber(t, first, "likeCount", 5)

	agentEnvelope := serveAdminStats(t, conn, router, statsRankingPath,
		`{"startDate":"`+statsFixtureDate(0)+`","endDate":"`+statsFixtureDate(0)+`","actorType":1}`)
	agentResult := decodeStatsResult(t, agentEnvelope)
	assertStatsNumber(t, agentResult, "total", 1)
	agentItem := agentResult["items"].([]any)[0].(map[string]any)
	if agentItem["actorType"].(float64) != 1 {
		t.Fatalf("agent actorType = %v, want 1", agentItem["actorType"])
	}
}

// TestAdminStatsUserDetailHTTP 断言单用户明细形状与用户解析失败的错误码（§2.4）。
func TestAdminStatsUserDetailHTTP(t *testing.T) {
	conn, router := setupAdminStatsTest(t)
	seedStatsRouteAdminData(t, conn)

	body := `{"startDate":"` + statsFixtureDate(0) + `","endDate":"` + statsFixtureDate(0) + `","userId":2}`
	envelope := serveAdminStats(t, conn, router, statsDetailPath, body)
	result := decodeStatsResult(t, envelope)

	user := result["user"].(map[string]any)
	if user["userId"].(float64) != 2 || user["username"].(string) == "" {
		t.Fatalf("user = %v", user)
	}
	summary := result["summary"].(map[string]any)
	assertStatsNumber(t, summary, "topicCount", 0)
	assertStatsNumber(t, summary, "replyCount", 1)
	assertStatsNumber(t, summary, "likeCount", 1)
	days, ok := result["days"].([]any)
	if !ok || len(days) != 1 {
		t.Fatalf("days = %v", result["days"])
	}
	assertStatsNumber(t, days[0].(map[string]any), "replyCount", 1)

	notFound := serveAdminStats(t, conn, router, statsDetailPath,
		`{"startDate":"`+statsFixtureDate(0)+`","endDate":"`+statsFixtureDate(0)+`","username":"missing-user"}`)
	if notFound.Code != 1 || notFound.MessageCode != "user.notFound" {
		t.Fatalf("unknown user = code %d / %s, want 1 / user.notFound", notFound.Code, notFound.MessageCode)
	}
}

// TestAdminStatsValidationHTTP 非法参数被明确拒绝：HTTP 200 + code 1 +
// common.request.invalidParams + params.field（口径 R9）。
func TestAdminStatsValidationHTTP(t *testing.T) {
	conn, router := setupAdminStatsTest(t)
	seedStatsRouteAdminData(t, conn)

	cases := []struct {
		name  string
		path  string
		body  string
		field string
		extra map[string]any
	}{
		{name: "bad date", path: statsSummaryPath, body: `{"startDate":"2026-4-7","endDate":"2026-04-07"}`, field: "startDate"},
		{name: "inverted range", path: statsSummaryPath, body: `{"startDate":"2026-04-08","endDate":"2026-04-07"}`, field: "endDate"},
		// 超限必须随错误回传上限值，界面据此展示「最多 366 天」。
		{name: "range too long", path: statsSummaryPath, body: `{"startDate":"2026-01-01","endDate":"2027-01-02"}`, field: "endDate", extra: map[string]any{"maxRangeDays": float64(366)}},
		{name: "missing category", path: statsTrendPath, body: `{"categoryId":0}`, field: "categoryId"},
		{name: "unknown category", path: statsTrendPath, body: `{"categoryId":999}`, field: "categoryId"},
		{name: "bad sort key", path: statsRankingPath, body: `{"sortBy":"views"}`, field: "sortBy"},
		{name: "bad page size", path: statsRankingPath, body: `{"pageSize":101}`, field: "pageSize"},
		// 显式 0 低于 schema 的 minimum，必须拒绝而不是静默回落到默认 20。
		{name: "explicit zero page size", path: statsRankingPath, body: `{"pageSize":0}`, field: "pageSize"},
		{name: "explicit zero page", path: statsRankingPath, body: `{"page":0}`, field: "page"},
		// 极大页码会让 (page-1)*pageSize 溢出：必须按参数错误拒绝，而不是变成查询失败。
		{name: "page over limit", path: statsRankingPath, body: `{"page":99999999,"pageSize":100}`, field: "page"},
		{name: "bad actor type", path: statsRankingPath, body: `{"actorType":7}`, field: "actorType"},
		{name: "missing identity", path: statsDetailPath, body: `{}`, field: "userId"},
		{name: "both identities", path: statsDetailPath, body: `{"userId":1,"username":"stats-route-b"}`, field: "username"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			envelope := serveAdminStats(t, conn, router, tc.path, tc.body)
			if envelope.Code != 1 || envelope.MessageCode != "common.request.invalidParams" {
				t.Fatalf("envelope = code %d / %s, want 1 / common.request.invalidParams", envelope.Code, envelope.MessageCode)
			}
			if field, _ := envelope.Params["field"].(string); field != tc.field {
				t.Fatalf("params.field = %v, want %s", envelope.Params["field"], tc.field)
			}
			for name, want := range tc.extra {
				if got := envelope.Params[name]; got != want {
					t.Fatalf("params.%s = %v, want %v", name, got, want)
				}
			}
			if tc.extra == nil && len(envelope.Params) != 1 {
				t.Fatalf("params = %v, want only the field param", envelope.Params)
			}
		})
	}
}

func assertStatsNumber(t *testing.T, source map[string]any, key string, want float64) {
	t.Helper()
	got, ok := source[key].(float64)
	if !ok || got != want {
		t.Fatalf("%s = %v, want %v", key, source[key], want)
	}
}
