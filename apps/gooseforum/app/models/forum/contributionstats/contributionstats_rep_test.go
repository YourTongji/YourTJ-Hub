package contributionstats

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/category"
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
	); err != nil {
		t.Fatalf("migrate contribution stats tables: %v", err)
	}
	for _, table := range []string{"topics", "posts", "topic_category_index", "users", "category"} {
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
	if err := conn.Create(&users.EntityComplete{
		Id: id, Username: fmt.Sprintf("stats-repo-%d", id), ActorType: users.ActorTypeHuman,
	}).Error; err != nil {
		t.Fatalf("create user %d: %v", id, err)
	}
}

// seedStatsTopic 造一个公开主题（含首楼）并挂到给定板块。
func seedStatsTopic(t *testing.T, conn *gorm.DB, id, authorId uint64, categories ...uint64) {
	t.Helper()
	published := statsRepoDay.Add(10 * time.Hour)
	firstPostId := id + 800
	if err := conn.Create(&topics.Entity{
		Id: id, Title: fmt.Sprintf("主题 %d", id), UserId: authorId, Status: 1,
		VisibilityStatus: topics.VisibilityActive, RetentionStatus: topics.RetentionNormal,
		FirstPostId: firstPostId, PostCount: 1, PostSeq: 1,
		CreatedAt: published, UpdatedAt: published, FirstPublicAt: &published,
	}).Error; err != nil {
		t.Fatalf("create topic %d: %v", id, err)
	}
	if err := conn.Create(&posts.Entity{
		Id: firstPostId, TopicId: id, PostNo: 1, UserId: authorId, Content: "首楼",
		VisibilityStatus: posts.VisibilityActive, RetentionStatus: posts.RetentionNormal,
		CreatedAt: published, UpdatedAt: published, FirstPublicAt: &published,
	}).Error; err != nil {
		t.Fatalf("create first post %d: %v", firstPostId, err)
	}
	for _, categoryId := range categories {
		if err := conn.Create(&topicCategoryIndex.Entity{TopicId: id, CategoryId: categoryId, Effective: 1}).Error; err != nil {
			t.Fatalf("index topic %d in category %d: %v", id, categoryId, err)
		}
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
