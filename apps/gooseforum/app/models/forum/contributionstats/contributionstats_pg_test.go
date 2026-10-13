package contributionstats

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/category"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/postUserAction"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topicCategoryIndex"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// TestContributionStatsAggregatesOnPostgreSQL 在真实 PostgreSQL 上验证聚合 SQL 的方言兼容性：
// 本地日切片（SQLite strftime ↔ PostgreSQL to_char/make_interval）、UNION ALL 去重、
// MAX(CASE WHEN) 估算标记与分页排序。依赖 YOURTJ_TEST_PG_URL
// （与 migration 包真实 PG 迁移测试同一门控），未设置时跳过。
func TestContributionStatsAggregatesOnPostgreSQL(t *testing.T) {
	dsn := os.Getenv("YOURTJ_TEST_PG_URL")
	if dsn == "" {
		t.Skip("YOURTJ_TEST_PG_URL not set; skipping PostgreSQL contribution stats test")
	}
	conn, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	schema := fmt.Sprintf("contribution_stats_%d", time.Now().UnixNano())
	if err := conn.Exec("CREATE SCHEMA " + schema).Error; err != nil {
		t.Fatalf("create schema: %v", err)
	}
	t.Cleanup(func() { conn.Exec("DROP SCHEMA " + schema + " CASCADE") })
	scoped, err := gorm.Open(postgres.Open(dsn+" search_path="+schema), &gorm.Config{})
	if err != nil {
		t.Fatalf("connect scoped postgres: %v", err)
	}
	sqlDB, err := scoped.DB()
	if err != nil {
		t.Fatalf("postgres sql handle: %v", err)
	}
	defer func() {
		if closeErr := sqlDB.Close(); closeErr != nil {
			t.Error(closeErr)
		}
	}()
	if err := scoped.AutoMigrate(
		&users.EntityComplete{},
		&category.Entity{},
		&topics.Entity{},
		&posts.Entity{},
		&topicCategoryIndex.Entity{},
		&postUserAction.Entity{},
	); err != nil {
		t.Fatalf("migrate contribution stats tables on postgres: %v", err)
	}

	day := seedPostgresStatsFixtures(t, scoped)
	scope := Scope{Start: day, End: day.AddDate(0, 0, 3)}

	categoryTopics, err := categoryCounts(scoped, scope, KindTopic)
	if err != nil {
		t.Fatalf("category topic counts on postgres: %v", err)
	}
	assertPgCategoryCount(t, categoryTopics, 1, 4)
	assertPgCategoryCount(t, categoryTopics, 2, 1)

	categoryReplies, err := categoryCounts(scoped, scope, KindReply)
	if err != nil {
		t.Fatalf("category reply counts on postgres: %v", err)
	}
	assertPgCategoryCount(t, categoryReplies, 1, 3)
	assertPgCategoryCount(t, categoryReplies, 2, 2)

	total, err := siteContentTotal(scoped, scope, KindTopic)
	if err != nil {
		t.Fatalf("site topic total on postgres: %v", err)
	}
	if total.Count != 4 || !total.Estimated {
		t.Fatalf("site topic total = %+v, want 4 with estimated=true (one topic has no first_public_at)", total)
	}

	humans, err := categoryContributorCounts(scoped, scope, users.ActorTypeHuman)
	if err != nil {
		t.Fatalf("human contributors on postgres: %v", err)
	}
	assertPgCategoryCount(t, humans, 1, 2)
	assertPgCategoryCount(t, humans, 2, 2)
	agents, err := categoryContributorCounts(scoped, scope, users.ActorTypeBot)
	if err != nil {
		t.Fatalf("agent contributors on postgres: %v", err)
	}
	assertPgCategoryCount(t, agents, 1, 1)

	humanTotal, err := siteContributorCount(scoped, scope, users.ActorTypeHuman)
	if err != nil {
		t.Fatalf("site human contributors on postgres: %v", err)
	}
	if humanTotal != 2 {
		t.Fatalf("site human contributors = %d, want 2", humanTotal)
	}

	categoryScope := scope
	categoryScope.CategoryID = 1
	daily, err := categoryDailyCounts(scoped, categoryScope, KindTopic)
	if err != nil {
		t.Fatalf("category daily counts on postgres: %v", err)
	}
	if len(daily) != 2 {
		t.Fatalf("daily rows = %+v, want two local days", daily)
	}
	// 板块 1 含全部 4 个主题；本地 00:30 的首发与缺失首次公开时间的主题都落在第一天。
	assertPgDay(t, daily[0], day.Format("2006-01-02"), 2, true)
	assertPgDay(t, daily[1], day.AddDate(0, 0, 1).Format("2006-01-02"), 2, false)

	ranking, err := userRanking(scoped, scope, users.ActorTypeHuman, SortByTopics, 0, 20)
	if err != nil {
		t.Fatalf("user ranking on postgres: %v", err)
	}
	if len(ranking) != 2 {
		t.Fatalf("ranking rows = %+v, want two human accounts", ranking)
	}
	if ranking[0].UserId != 1 || ranking[0].TopicCount != 2 || ranking[0].LikeCount != 5 {
		t.Fatalf("ranking[0] = %+v, want user 1 with 2 topics and 5 likes", ranking[0])
	}
	// 账号 2 有 3 个楼层：200/201 计入，匿名楼层 202 不归因；点赞只算 200 上仍有效的那条。
	if ranking[1].UserId != 2 || ranking[1].ReplyCount != 2 || ranking[1].LikeCount != 1 {
		t.Fatalf("ranking[1] = %+v, want user 2 with 2 replies and 1 like (anonymous floor and cancelled like excluded)", ranking[1])
	}
	rankingTotal, err := userRankingTotal(scoped, scope, users.ActorTypeHuman)
	if err != nil {
		t.Fatalf("ranking total on postgres: %v", err)
	}
	if rankingTotal != 2 {
		t.Fatalf("ranking total = %d, want 2", rankingTotal)
	}

	detail, err := userContentTotals(scoped, scope, 2)
	if err != nil {
		t.Fatalf("user content totals on postgres: %v", err)
	}
	if detail.TopicCount != 0 || detail.ReplyCount != 2 || detail.LikeCount != 1 {
		t.Fatalf("user 2 totals = %+v, want 0/2/1", detail)
	}

	replyDays, err := userDailyCounts(scoped, scope, 2, KindReply)
	if err != nil {
		t.Fatalf("user daily counts on postgres: %v", err)
	}
	if len(replyDays) != 1 {
		t.Fatalf("user reply days = %+v, want only the reply day", replyDays)
	}
	assertPgDay(t, replyDays[0], day.AddDate(0, 0, 2).Format("2006-01-02"), 2, false)
}

func assertPgCategoryCount(t *testing.T, rows []CategoryCount, categoryId, want int64) {
	t.Helper()
	for _, row := range rows {
		if row.CategoryId == uint64(categoryId) {
			if row.Count != want {
				t.Fatalf("category %d count = %d, want %d", categoryId, row.Count, want)
			}
			return
		}
	}
	if want != 0 {
		t.Fatalf("category %d missing from %+v, want %d", categoryId, rows, want)
	}
}

func assertPgDay(t *testing.T, row DailyCount, date string, count int64, estimated bool) {
	t.Helper()
	if row.Day != date || row.Count != count || row.Estimated != estimated {
		t.Fatalf("day = %+v, want date %s count %d estimated %t", row, date, count, estimated)
	}
}

// seedPostgresStatsFixtures 造一组覆盖跨板块、匿名、已注销、Agent 与估算时间归属的数据。
func seedPostgresStatsFixtures(t *testing.T, conn *gorm.DB) time.Time {
	t.Helper()
	day := time.Date(2026, 5, 6, 0, 0, 0, 0, time.Local)
	at := func(dayOffset, hour, minute int) time.Time {
		return day.AddDate(0, 0, dayOffset).Add(time.Duration(hour)*time.Hour + time.Duration(minute)*time.Minute)
	}
	if err := conn.Create(&category.Entity{Id: 1, Name: "综合", Sort: 1}).Error; err != nil {
		t.Fatalf("create category: %v", err)
	}
	if err := conn.Create(&category.Entity{Id: 2, Name: "学习", Sort: 2}).Error; err != nil {
		t.Fatalf("create category: %v", err)
	}
	createUser := func(id uint64, actorType int8, closed bool) {
		entity := users.EntityComplete{Id: id, Username: fmt.Sprintf("pg-user-%d", id), ActorType: actorType}
		if closed {
			entity.DeletedAt = gorm.DeletedAt{Time: day, Valid: true}
		}
		if err := conn.Create(&entity).Error; err != nil {
			t.Fatalf("create user %d: %v", id, err)
		}
	}
	createUser(1, users.ActorTypeHuman, false)
	createUser(2, users.ActorTypeHuman, false)
	createUser(10, users.ActorTypeBot, false)
	createUser(11, users.ActorTypeHuman, true)

	// 本地 00:30 的首发时刻，其 UTC 日历日是前一天：用于验证按本地日而非 UTC 日切片。
	earlyMorning := at(0, 0, 30)
	createTopic := func(id, userId uint64, firstPublicAt *time.Time, likeCount uint64, categories ...uint64) {
		createdAt := earlyMorning
		if firstPublicAt != nil {
			createdAt = *firstPublicAt
		}

		topic := topics.Entity{
			Id: id, Title: fmt.Sprintf("pg topic %d", id), UserId: userId, Status: 1,
			VisibilityStatus: topics.VisibilityActive, RetentionStatus: topics.RetentionNormal,
			FirstPostId: id + 500, PostCount: 1, PostSeq: 1, LikeCount: likeCount,
			CreatedAt: createdAt, UpdatedAt: createdAt, FirstPublicAt: firstPublicAt,
		}
		if err := conn.Create(&topic).Error; err != nil {
			t.Fatalf("create topic %d: %v", id, err)
		}
		for _, categoryId := range categories {
			if err := conn.Create(&topicCategoryIndex.Entity{TopicId: id, CategoryId: categoryId, Effective: 1}).Error; err != nil {
				t.Fatalf("index topic %d: %v", id, err)
			}
		}
		if err := conn.Create(&posts.Entity{
			Id: id + 500, TopicId: id, PostNo: 1, UserId: userId, Content: "首楼",
			VisibilityStatus: posts.VisibilityActive, RetentionStatus: posts.RetentionNormal,
			CreatedAt: createdAt, UpdatedAt: createdAt, FirstPublicAt: firstPublicAt,
		}).Error; err != nil {
			t.Fatalf("create first post %d: %v", id+500, err)
		}
	}
	createTopic(100, 1, &earlyMorning, 5, 1, 2)
	botPublish := at(1, 12, 0)
	createTopic(101, 10, &botPublish, 1, 1)
	createTopic(102, 11, &botPublish, 0, 1) // 已注销作者的公开内容仍计入内容量
	// 首次公开时间缺失（101 的 created_at 落在 day1）：估算来源按 created_at 归属。
	createTopic(103, 1, nil, 0, 1)

	createPost := func(id, topicId, userId, postNo uint64, anonymous bool) {
		replyAt := at(2, 10, 0)
		post := posts.Entity{
			Id: id, TopicId: topicId, PostNo: postNo, UserId: userId, Content: "回复",
			IsAnonymous: anonymous, VisibilityStatus: posts.VisibilityActive,
			RetentionStatus: posts.RetentionNormal, CreatedAt: replyAt, UpdatedAt: replyAt,
			FirstPublicAt: &replyAt,
		}
		if err := conn.Create(&post).Error; err != nil {
			t.Fatalf("create post %d: %v", id, err)
		}
	}
	createPost(200, 100, 2, 2, false)
	createPost(201, 101, 2, 2, false)
	createPost(202, 100, 2, 3, true) // 匿名楼层：内容计入，作者不归因

	likedAt := at(2, 11, 0)
	if err := conn.Create(&postUserAction.Entity{PostId: 200, UserId: 2, LikedAt: &likedAt}).Error; err != nil {
		t.Fatalf("like post 200: %v", err)
	}
	if err := conn.Create(&postUserAction.Entity{PostId: 201, UserId: 1}).Error; err != nil {
		t.Fatalf("cancelled like post 201: %v", err)
	}
	return day
}

// TestUserRankingTieBreakOnPostgreSQL 在真实 PostgreSQL 上锁住排行的稳定序：
// 哈希聚合不保证同值行的输出顺序，造数刻意让 id 较小的账号最后写入，
// 只有带 `user_id ASC` 兜底才能得到升序结果。
func TestUserRankingTieBreakOnPostgreSQL(t *testing.T) {
	dsn := os.Getenv("YOURTJ_TEST_PG_URL")
	if dsn == "" {
		t.Skip("YOURTJ_TEST_PG_URL not set; skipping PostgreSQL contribution stats test")
	}
	conn, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	schema := fmt.Sprintf("contribution_stats_tie_%d", time.Now().UnixNano())
	if err := conn.Exec("CREATE SCHEMA " + schema).Error; err != nil {
		t.Fatalf("create schema: %v", err)
	}
	t.Cleanup(func() { conn.Exec("DROP SCHEMA " + schema + " CASCADE") })
	scoped, err := gorm.Open(postgres.Open(dsn+" search_path="+schema), &gorm.Config{})
	if err != nil {
		t.Fatalf("connect scoped postgres: %v", err)
	}
	if err := scoped.AutoMigrate(
		&users.EntityComplete{}, &topics.Entity{}, &posts.Entity{}, &postUserAction.Entity{},
	); err != nil {
		t.Fatalf("migrate on postgres: %v", err)
	}

	day := time.Date(2026, 7, 1, 0, 0, 0, 0, time.Local)
	published := day.Add(10 * time.Hour)
	// 降序写入 8 个同值账号：哈希聚合的输出顺序与写入顺序无关，8 组恰好升序的概率极低，
	// 若排行丢掉 `user_id ASC` 兜底，本用例在实践中必然报错。
	tiedIDs := []uint64{9, 8, 7, 6, 5, 4, 3, 2}
	for _, id := range tiedIDs {
		if err := scoped.Create(&users.EntityComplete{
			Id: id, Username: fmt.Sprintf("tie-%d", id), ActorType: users.ActorTypeHuman,
		}).Error; err != nil {
			t.Fatalf("create user %d: %v", id, err)
		}
		topicID := 500 + id
		if err := scoped.Create(&topics.Entity{
			Id: topicID, Title: "并列主题", UserId: id, Status: 1,
			VisibilityStatus: topics.VisibilityActive, RetentionStatus: topics.RetentionNormal,
			FirstPostId: topicID + 100, PostCount: 1, PostSeq: 1,
			CreatedAt: published, UpdatedAt: published, FirstPublicAt: &published,
		}).Error; err != nil {
			t.Fatalf("create topic %d: %v", topicID, err)
		}
		if err := scoped.Create(&posts.Entity{
			Id: topicID + 100, TopicId: topicID, PostNo: 1, UserId: id, Content: "首楼",
			VisibilityStatus: posts.VisibilityActive, RetentionStatus: posts.RetentionNormal,
			CreatedAt: published, UpdatedAt: published, FirstPublicAt: &published,
		}).Error; err != nil {
			t.Fatalf("create first post of topic %d: %v", topicID, err)
		}
	}

	scope := Scope{Start: day, End: day.AddDate(0, 0, 1)}
	rows, err := userRanking(scoped, scope, users.ActorTypeHuman, SortByTopics, 0, 10)
	if err != nil {
		t.Fatalf("ranking on postgres: %v", err)
	}
	if len(rows) != len(tiedIDs) {
		t.Fatalf("ranking rows = %+v, want %d tied accounts", rows, len(tiedIDs))
	}
	for i := range tiedIDs {
		want := tiedIDs[len(tiedIDs)-1-i] // 期望升序：2、3、…、9
		if rows[i].UserId != want || rows[i].TopicCount != 1 {
			t.Fatalf("ranking[%d] = %+v, want user %d with one topic", i, rows[i], want)
		}
	}
}
