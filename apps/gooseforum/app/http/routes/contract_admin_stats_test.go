package routes

import (
	"encoding/json"
	"reflect"
	"regexp"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/category"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/postUserAction"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topicCategoryIndex"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"gorm.io/gorm"
)

// 管理后台内容贡献统计（issue #1145）的契约测试：以固定场景逐字段比对
// packages/api-contract/fixtures/admin-stats-*.json。中间件守卫场景（未登录 401 /
// 冻结账号 403 / 无 Admin 权限 403）已由 admin_stats_api_test.go 的
// TestAdminStatsRoutesGuardScenarios 覆盖同一组路由，此处不重复。

const (
	statsContractAlice = 1024 // 真人：2 个主题 + 1 条回复
	statsContractBob   = 2049 // 真人：1 条回复（被点赞 1 次）
	statsContractBot   = 4097 // Agent：1 条回复
	statsContractRange = `"startDate":"2020-01-01","endDate":"2020-01-03"`
)

// statsContractDay 是契约场景的基准日（本地日 00:00），fixture 的三天序列由它推算。
var statsContractDay = time.Date(2020, 1, 1, 0, 0, 0, 0, time.Local)

// statsTimezoneFormat 是契约冻结的 coverage.timezone 格式："<缩写> (UTC±HH:MM)"。
var statsTimezoneFormat = regexp.MustCompile(`^\S.* \(UTC[+-]\d{2}:\d{2}\)$`)

// seedAdminStatsContractData 造出与 fixtures/admin-stats-*-success.json 完全对应的场景：
// 板块 1 有 2020-01-01 与 2020-01-03 两个主题、01-02 两条回复、01-03 一条 Agent 回复；
// 板块 2 无内容（契约要求无数据板块仍列出且计 0）。所有行都有精确的 first_public_at，
// 因此 estimated 恒为 false。
func seedAdminStatsContractData(t *testing.T, conn *gorm.DB) {
	t.Helper()
	for _, entity := range []category.Entity{
		{Id: 1, Name: "校园生活", Sort: 1},
		{Id: 2, Name: "课程交流", Sort: 2},
	} {
		if err := conn.Create(&entity).Error; err != nil {
			t.Fatalf("create category %d: %v", entity.Id, err)
		}
	}
	for _, account := range []struct {
		id        uint64
		username  string
		nickname  string
		actorType int8
	}{
		{statsContractAlice, "alice", "小艾", users.ActorTypeHuman},
		{statsContractBob, "bob", "小波", users.ActorTypeHuman},
		{statsContractBot, "campus-helper", "校园助手", users.ActorTypeBot},
	} {
		if err := conn.Create(&users.EntityComplete{
			Id: account.id, Username: account.username, Nickname: account.nickname, ActorType: account.actorType,
		}).Error; err != nil {
			t.Fatalf("create user %d: %v", account.id, err)
		}
	}

	at := func(dayOffset, hour int) time.Time {
		return statsContractDay.AddDate(0, 0, dayOffset).Add(time.Duration(hour) * time.Hour)
	}
	createPost := func(id, topicID, postNo, author uint64, published time.Time) {
		if err := conn.Create(&posts.Entity{
			Id: id, TopicId: topicID, PostNo: postNo, UserId: author, Content: "契约统计内容",
			VisibilityStatus: posts.VisibilityActive, RetentionStatus: posts.RetentionNormal,
			CreatedAt: published, UpdatedAt: published, FirstPublicAt: &published,
		}).Error; err != nil {
			t.Fatalf("create post %d: %v", id, err)
		}
	}
	createTopic := func(id, author uint64, published time.Time, likeCount uint64) {
		if err := conn.Create(&topics.Entity{
			Id: id, Title: "契约统计主题", UserId: author, Status: 1, LikeCount: likeCount,
			VisibilityStatus: topics.VisibilityActive, RetentionStatus: topics.RetentionNormal,
			FirstPostId: id + 900, PostCount: 1, PostSeq: 1,
			CreatedAt: published, UpdatedAt: published, FirstPublicAt: &published,
		}).Error; err != nil {
			t.Fatalf("create topic %d: %v", id, err)
		}
		if err := conn.Create(&topicCategoryIndex.Entity{TopicId: id, CategoryId: 1, Effective: 1}).Error; err != nil {
			t.Fatalf("index topic %d: %v", id, err)
		}
		// 首楼（post_no = 1）与主题同日公开，按口径 R1 不计入回复。
		createPost(id+900, id, 1, author, published)
	}

	createTopic(1101, statsContractAlice, at(0, 10), 4)
	createTopic(1102, statsContractAlice, at(2, 15), 1)
	createPost(2101, 1101, 2, statsContractAlice, at(1, 9))
	createPost(2102, 1101, 3, statsContractBob, at(1, 14))
	createPost(2103, 1101, 4, statsContractBot, at(2, 9))

	// 回复获赞：只有 liked_at 非空的行计入（取消点赞置空，口径 R5）。
	likedAt := at(1, 16)
	if err := conn.Create(&postUserAction.Entity{PostId: 2102, UserId: statsContractAlice, LikedAt: &likedAt}).Error; err != nil {
		t.Fatalf("like reply 2102: %v", err)
	}
	cleanupAdminStatsContractData(t, conn)
}

// cleanupAdminStatsContractData 硬删除本用例造的行。users/topics/posts 都是软删模型，
// 用 GORM Delete 只会写 deleted_at、留下唯一键残留，同进程后续用例再建同 id 账号或
// 同主题楼层就会 duplicated key，因此统一走原生 DELETE。
func cleanupAdminStatsContractData(t *testing.T, conn *gorm.DB) {
	t.Helper()
	t.Cleanup(func() {
		statements := []struct {
			query string
			args  []any
		}{
			{"DELETE FROM post_user_action WHERE post_id IN (?, ?, ?)", []any{2101, 2102, 2103}},
			{"DELETE FROM posts WHERE topic_id IN (?, ?)", []any{1101, 1102}},
			{"DELETE FROM topic_category_index WHERE topic_id IN (?, ?)", []any{1101, 1102}},
			{"DELETE FROM topics WHERE id IN (?, ?)", []any{1101, 1102}},
			{"DELETE FROM users WHERE id IN (?, ?, ?)", []any{statsContractAlice, statsContractBob, statsContractBot}},
			{"DELETE FROM category WHERE id IN (?, ?)", []any{1, 2}},
		}
		for _, statement := range statements {
			if err := conn.Exec(statement.query, statement.args...).Error; err != nil {
				t.Errorf("clean up %q: %v", statement.query, err)
			}
		}
	})
}

// assertAdminStatsFixture 按 fixture 逐字段比对响应信封（code / messageCode / result / params）。
func assertAdminStatsFixture(t *testing.T, actual contractEnvelope, fixtureName string) {
	t.Helper()
	fixture := contractFixture(t, fixtureName)
	if actual.Code != fixture.Code {
		t.Fatalf("code = %d, want fixture code %d", actual.Code, fixture.Code)
	}
	if fixture.MessageCode != "" && actual.MessageCode != fixture.MessageCode {
		t.Fatalf("messageCode = %q, want fixture messageCode %q", actual.MessageCode, fixture.MessageCode)
	}
	var actualResult map[string]any
	if err := json.Unmarshal(actual.Result, &actualResult); err != nil {
		t.Fatalf("decode result %q: %v", actual.Result, err)
	}
	var fixtureResult map[string]any
	if err := json.Unmarshal(fixture.Result, &fixtureResult); err != nil {
		t.Fatalf("decode fixture %s result %q: %v", fixtureName, fixture.Result, err)
	}
	normalizeStatsTimezone(t, actualResult, fixtureResult)
	if !reflect.DeepEqual(actualResult, fixtureResult) {
		t.Fatalf("result = %#v, want fixture %s result %#v", actualResult, fixtureName, fixtureResult)
	}
	assertFixtureParams(t, actual.Params, fixture.Params)
}

// normalizeStatsTimezone 校验实际与 fixture 的 coverage.timezone 都符合冻结格式，
// 再把实际值替换为 fixture 示例值。timezone 是运行时派生的服务器本地时区，跨机器
// 必然与 fixture 的示例值不同——沿用 assertFixtureParams 对 retryAfterSeconds 的
// 既有一致性例外先例（contract_http_test.go）：该字段只断言格式，其余字段仍逐字比对。
func normalizeStatsTimezone(t *testing.T, actual, fixture map[string]any) {
	t.Helper()
	fixtureCoverage, ok := fixture["coverage"].(map[string]any)
	if !ok {
		if _, present := actual["coverage"]; present {
			t.Fatalf("actual result carries coverage %v but the fixture has none", actual["coverage"])
		}
		return
	}
	actualCoverage, ok := actual["coverage"].(map[string]any)
	if !ok {
		t.Fatalf("coverage missing from actual result %#v", actual)
	}
	actualTimezone, _ := actualCoverage["timezone"].(string)
	fixtureTimezone, _ := fixtureCoverage["timezone"].(string)
	if !statsTimezoneFormat.MatchString(actualTimezone) {
		t.Fatalf("coverage.timezone = %q, want the frozen %q format", actualTimezone, statsTimezoneFormat)
	}
	if !statsTimezoneFormat.MatchString(fixtureTimezone) {
		t.Fatalf("fixture coverage.timezone = %q is not the frozen %q format", fixtureTimezone, statsTimezoneFormat)
	}
	actualCoverage["timezone"] = fixtureTimezone
}

func TestAdminStatsHTTPContract(t *testing.T) {
	t.Run("category summary returns the seeded per-category totals", func(t *testing.T) {
		conn, router := setupAdminStatsTest(t)
		seedAdminStatsContractData(t, conn)
		assertAdminStatsFixture(t,
			serveAdminStats(t, conn, router, statsSummaryPath, `{`+statsContractRange+`}`),
			"admin-stats-category-summary-success.json")
	})

	t.Run("category trend zero-fills days without contribution", func(t *testing.T) {
		conn, router := setupAdminStatsTest(t)
		seedAdminStatsContractData(t, conn)
		assertAdminStatsFixture(t,
			serveAdminStats(t, conn, router, statsTrendPath, `{`+statsContractRange+`,"categoryId":1}`),
			"admin-stats-category-trend-success.json")
	})

	t.Run("user ranking pages the seeded human contributors", func(t *testing.T) {
		conn, router := setupAdminStatsTest(t)
		seedAdminStatsContractData(t, conn)
		assertAdminStatsFixture(t,
			serveAdminStats(t, conn, router, statsRankingPath, `{`+statsContractRange+`}`),
			"admin-stats-user-ranking-success.json")
	})

	t.Run("user detail returns identity, summary and the daily series", func(t *testing.T) {
		conn, router := setupAdminStatsTest(t)
		seedAdminStatsContractData(t, conn)
		assertAdminStatsFixture(t,
			serveAdminStats(t, conn, router, statsDetailPath, `{`+statsContractRange+`,"userId":1024}`),
			"admin-stats-user-detail-success.json")
	})

	t.Run("category summary rejects an inverted range", func(t *testing.T) {
		conn, router := setupAdminStatsTest(t)
		assertAdminStatsFixture(t,
			serveAdminStats(t, conn, router, statsSummaryPath, `{"startDate":"2020-01-03","endDate":"2020-01-01"}`),
			"admin-stats-category-summary-invalid-params.json")
	})

	t.Run("category trend rejects a missing category", func(t *testing.T) {
		conn, router := setupAdminStatsTest(t)
		assertAdminStatsFixture(t,
			serveAdminStats(t, conn, router, statsTrendPath, `{"startDate":"2020-01-01","endDate":"2020-01-03"}`),
			"admin-stats-category-trend-invalid-params.json")
	})

	t.Run("user ranking rejects a page size above the cap", func(t *testing.T) {
		conn, router := setupAdminStatsTest(t)
		assertAdminStatsFixture(t,
			serveAdminStats(t, conn, router, statsRankingPath, `{"pageSize":101}`),
			"admin-stats-user-ranking-invalid-params.json")
	})

	t.Run("user detail rejects a malformed date", func(t *testing.T) {
		conn, router := setupAdminStatsTest(t)
		seedAdminStatsContractData(t, conn)
		// 账号存在但结束日期非法：校验在解析账号之后，仍须报 invalidParams 而不是 user.notFound。
		assertAdminStatsFixture(t,
			serveAdminStats(t, conn, router, statsDetailPath, `{"startDate":"2020-01-01","endDate":"2020-1-3","userId":1024}`),
			"admin-stats-user-detail-invalid-params.json")
	})

	t.Run("user detail reports a closed or unknown account", func(t *testing.T) {
		conn, router := setupAdminStatsTest(t)
		seedAdminStatsContractData(t, conn)
		assertAdminStatsFixture(t,
			serveAdminStats(t, conn, router, statsDetailPath, `{"userId":999999}`),
			"admin-stats-user-detail-not-found.json")
	})

	// fixture admin-stats-fetch-failed.json 的 admin.stats.fetchFailed 分支需要让聚合
	// 查询真的失败（例如删表），会污染同进程共享的测试库，故只由契约链的 ajv 校验
	// （packages/api-contract/scripts/validate-fixtures.mjs），不在本文件内触发。既有
	// adminTrafficOverview 同样只为成功分支写运行时断言。
}

// 未登录 / 冻结账号 / 无 Admin 权限三条守卫场景由
// admin_stats_api_test.go#TestAdminStatsRoutesGuardScenarios 断言（同一组路由、
// 同一组 fixture：auth-required.json / account-frozen.json /
// admin-agent-create-permission-denied.json），本文件不再重复。
