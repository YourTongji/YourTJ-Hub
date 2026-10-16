package contributionstats

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/category"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/postUserAction"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topicCategoryIndex"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"gorm.io/gorm"
)

// TestCategoryContributorCountsHonoursCategoryScope 覆盖评审 S2：指定板块时只返回该板块的
// 分组行——跨板块主题不得把其他板块的贡献人带出来（服务层当前恒传 0，本用例直接锁住仓储行为）。
func TestCategoryContributorCountsHonoursCategoryScope(t *testing.T) {
	conn := setupContributionStatsTestDB(t)
	seedStatsCategory(t, conn, 1)
	seedStatsCategory(t, conn, 2)
	seedStatsAuthor(t, conn, 1)
	seedStatsAuthor(t, conn, 2)

	// 主题 100 同时属于板块 1、2；主题 101 只属于板块 2。
	seedStatsTopic(t, conn, 100, 1, 1, 2)
	seedStatsTopic(t, conn, 101, 2, 2)

	scope := Scope{Start: statsRepoDay, End: statsRepoDay.AddDate(0, 0, 1)}

	scope.CategoryID = 1
	first := contributorCountsFor(t, scope)
	assertStatsCategoryRow(t, first, 1, 1)
	if len(first) != 1 {
		t.Fatalf("category 1 rows = %+v, want only the selected category", first)
	}

	scope.CategoryID = 2
	second := contributorCountsFor(t, scope)
	assertStatsCategoryRow(t, second, 2, 2)
	if len(second) != 1 {
		t.Fatalf("category 2 rows = %+v, want only the selected category", second)
	}

	scope.CategoryID = 0
	both := contributorCountsFor(t, scope)
	if len(both) != 2 {
		t.Fatalf("site rows = %+v, want both categories", both)
	}
	assertStatsCategoryRow(t, both, 1, 1)
	assertStatsCategoryRow(t, both, 2, 2)
}

var statsRepoDay = time.Date(2026, 6, 1, 0, 0, 0, 0, time.Local)

func contributorCountsFor(t *testing.T, scope Scope) []CategoryCount {
	t.Helper()
	rows, err := CategoryContributorCounts(context.Background(), scope, users.ActorTypeHuman)
	if err != nil {
		t.Fatalf("category contributor counts: %v", err)
	}
	return rows
}

func assertStatsCategoryRow(t *testing.T, rows []CategoryCount, categoryId, count int64) {
	t.Helper()
	for _, row := range rows {
		if row.CategoryId == uint64(categoryId) {
			if row.Count != count {
				t.Fatalf("category %d contributors = %d, want %d (%+v)", categoryId, row.Count, count, rows)
			}
			return
		}
	}
	t.Fatalf("category %d missing from %+v", categoryId, rows)
}

func setupContributionStatsTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(
		&users.EntityComplete{},
		&category.Entity{},
		&topics.Entity{},
		&posts.Entity{},
		&topicCategoryIndex.Entity{},
		// 排行查询 LEFT JOIN post_user_action（回复获赞），迁移齐全才能直接跑仓储查询。
		&postUserAction.Entity{},
	); err != nil {
		t.Fatalf("migrate contribution stats tables: %v", err)
	}
	for _, table := range []string{"topics", "posts", "topic_category_index", "users", "category", "post_user_action"} {
		if err := conn.Exec("DELETE FROM " + table).Error; err != nil {
			t.Fatalf("clear %s: %v", table, err)
		}
	}
	return conn
}

func seedStatsCategory(t *testing.T, conn *gorm.DB, id uint64) {
	t.Helper()
	if err := conn.Create(&category.Entity{Id: id, Name: fmt.Sprintf("板块 %d", id), Sort: int(id)}).Error; err != nil {
		t.Fatalf("create category %d: %v", id, err)
	}
}

func seedStatsAuthor(t *testing.T, conn *gorm.DB, id uint64) {
	t.Helper()
	seedStatsActor(t, conn, id, users.ActorTypeHuman)
}

func seedStatsBot(t *testing.T, conn *gorm.DB, id uint64) {
	t.Helper()
	seedStatsActor(t, conn, id, users.ActorTypeBot)
}

func seedStatsActor(t *testing.T, conn *gorm.DB, id uint64, actorType int8) {
	t.Helper()
	if err := conn.Create(&users.EntityComplete{
		Id: id, Username: fmt.Sprintf("stats-repo-%d", id), ActorType: actorType,
	}).Error; err != nil {
		t.Fatalf("create user %d: %v", id, err)
	}
}

// seedStatsTopic 造一个公开主题（含首楼）并挂到给定板块。
func seedStatsTopic(t *testing.T, conn *gorm.DB, id, authorId uint64, categories ...uint64) {
	t.Helper()
	published := statsRepoDay.Add(10 * time.Hour)
	seedStatsTopicWithFirstPublicAt(t, conn, id, authorId, &published, categories...)
}

// seedStatsTopicWithFirstPublicAt 造一个公开主题；firstPublicAt 为 nil 表示首次公开时间缺失，
// 该行按创建时间归属且计入 estimated（口径 R2/R8）。
func seedStatsTopicWithFirstPublicAt(t *testing.T, conn *gorm.DB, id, authorId uint64, firstPublicAt *time.Time, categories ...uint64) {
	t.Helper()
	published := statsRepoDay.Add(10 * time.Hour)
	if firstPublicAt != nil {
		published = *firstPublicAt
	}
	firstPostId := id + 800
	if err := conn.Create(&topics.Entity{
		Id: id, Title: fmt.Sprintf("主题 %d", id), UserId: authorId, Status: 1,
		VisibilityStatus: topics.VisibilityActive, RetentionStatus: topics.RetentionNormal,
		FirstPostId: firstPostId, PostCount: 1, PostSeq: 1,
		CreatedAt: published, UpdatedAt: published, FirstPublicAt: firstPublicAt,
	}).Error; err != nil {
		t.Fatalf("create topic %d: %v", id, err)
	}
	if err := conn.Create(&posts.Entity{
		Id: firstPostId, TopicId: id, PostNo: 1, UserId: authorId, Content: "首楼",
		VisibilityStatus: posts.VisibilityActive, RetentionStatus: posts.RetentionNormal,
		CreatedAt: published, UpdatedAt: published, FirstPublicAt: firstPublicAt,
	}).Error; err != nil {
		t.Fatalf("create first post %d: %v", firstPostId, err)
	}
	for _, categoryId := range categories {
		if err := conn.Create(&topicCategoryIndex.Entity{TopicId: id, CategoryId: categoryId, Effective: 1}).Error; err != nil {
			t.Fatalf("index topic %d in category %d: %v", id, categoryId, err)
		}
	}
}

// TestUserRankingTotalCarriesEstimatedFlag 覆盖评审 Should 建议：账号总数（分页依据）与
// coverage.estimated 由同一条 UNION 扫描返回，标记必须只反映「本次排行计入的行」——
// 每个账号先在自己名下的行里取最大估算标记，总数层再取最大；只有 Agent 的行是估算来源时
// 真人统计不得误报（口径 R4/R8）。
func TestUserRankingTotalCarriesEstimatedFlag(t *testing.T) {
	conn := setupContributionStatsTestDB(t)
	seedStatsAuthor(t, conn, 1)
	seedStatsBot(t, conn, 10)

	// 账号 1：两个主题（精确首发 + 首发时间缺失）——去重后仍算一个账号。
	seedStatsTopic(t, conn, 100, 1)
	seedStatsTopicWithFirstPublicAt(t, conn, 101, 1, nil)
	// 账号 10（Agent）：只有精确首发的主题。
	seedStatsTopic(t, conn, 102, 10)

	scope := Scope{Start: statsRepoDay, End: statsRepoDay.AddDate(0, 0, 1)}
	human, err := userRankingTotal(conn, scope, users.ActorTypeHuman)
	if err != nil {
		t.Fatalf("human ranking total: %v", err)
	}
	if human.Total != 1 || !human.Estimated {
		t.Fatalf("human ranking total = %+v, want one account with estimated=true", human)
	}
	agent, err := userRankingTotal(conn, scope, users.ActorTypeBot)
	if err != nil {
		t.Fatalf("agent ranking total: %v", err)
	}
	if agent.Total != 1 || agent.Estimated {
		t.Fatalf("agent ranking total = %+v, want one account with estimated=false", agent)
	}
}

// TestDeploymentTimezoneHasNoDST 把「部署时区无夏令时」这一前提变成断言（评审 Should 建议）：
// 逐日分桶的 localDayExpr 用查询时刻的时区偏移统一作用于整段区间，只有无夏令时的服务器时区
// （部署为 Asia/Shanghai）才与逐日换算一致；跨 DST 边界会静默错桶，MaxRangeDays 也会差 1。
// 在 DST 时区运行测试时这里直接失败，而不是让统计默默给出错误分桶（见
// docs/operations/deployment.md 的服务器时区约束）。
func TestDeploymentTimezoneHasNoDST(t *testing.T) {
	_, januaryOffset := time.Date(2026, 1, 15, 12, 0, 0, 0, time.Local).Zone()
	_, julyOffset := time.Date(2026, 7, 15, 12, 0, 0, 0, time.Local).Zone()
	if januaryOffset != julyOffset {
		t.Fatalf(
			"server timezone %q has daylight saving time (January %+ds vs July %+ds); "+
				"contribution statistics bucket by server-local day and require a DST-free timezone "+
				"(deployment uses Asia/Shanghai) — see docs/operations/deployment.md",
			time.Local.String(), januaryOffset, julyOffset,
		)
	}
}

// TestRankingOrderCarriesStableTieBreak 固定排行的 ORDER BY 片段：同值必须以
// user_id 升序兜底。SQLite 的 GROUP BY 恰好会按分组键排序，行为断言单独存在时无法
// 证明稳定序；这里直接锁住 SQL 片段（跨引擎的确定性保证），行为断言见 service 用例
// （SQLite）与 TestUserRankingTieBreakOnPostgreSQL（哈希聚合下顺序不保证）。
func TestRankingOrderCarriesStableTieBreak(t *testing.T) {
	cases := map[SortKey]string{
		SortByTopics:  "topic_count DESC, author_totals.user_id ASC",
		SortByReplies: "reply_count DESC, author_totals.user_id ASC",
		SortByLikes:   "like_count DESC, author_totals.user_id ASC",
	}
	for sort, want := range cases {
		if got := rankingOrder(sort); got != want {
			t.Fatalf("rankingOrder(%q) = %q, want %q", sort, got, want)
		}
	}
	// 未知取值回退主题数：仓储层不得把任意输入拼进 SQL（服务层已拒绝非法值）。
	if got := rankingOrder(SortKey("views")); got != cases[SortByTopics] {
		t.Fatalf("rankingOrder(unknown) = %q, want the topics fallback", got)
	}
}
