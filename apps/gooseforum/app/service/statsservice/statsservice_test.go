package statsservice

import (
	"context"
	"fmt"
	"strings"
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

// 统计口径测试使用固定的本地日，便于断言「服务器本地日」边界。
var statsDay = time.Date(2026, 3, 10, 0, 0, 0, 0, time.Local)

const (
	firstPostIdOffset = uint64(1000)
	categoryMain      = uint64(1)
	categoryStudy     = uint64(2)
	categoryEmpty     = uint64(3)
)

func statsAt(dayOffset, hour, minute, second int) time.Time {
	return statsDay.AddDate(0, 0, dayOffset).
		Add(time.Duration(hour)*time.Hour + time.Duration(minute)*time.Minute + time.Duration(second)*time.Second)
}

func statsDate(dayOffset int) string {
	return statsDay.AddDate(0, 0, dayOffset).Format("2006-01-02")
}

// setupStatsTestDB 准备内存库并清空统计相关表，避免同进程其他用例的残留数据串扰。
func setupStatsTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(
		&users.EntityComplete{},
		&category.Entity{},
		&topics.Entity{},
		&posts.Entity{},
		&topicCategoryIndex.Entity{},
		&postUserAction.Entity{},
	); err != nil {
		t.Fatalf("migrate stats tables: %v", err)
	}
	for _, table := range []string{"topics", "posts", "topic_category_index", "post_user_action", "category", "users"} {
		if err := conn.Exec("DELETE FROM " + table).Error; err != nil {
			t.Fatalf("clear %s: %v", table, err)
		}
	}
	return conn
}

func seedHuman(t *testing.T, conn *gorm.DB, id uint64) {
	t.Helper()
	seedUser(t, conn, id, users.ActorTypeHuman, false)
}

func seedAgent(t *testing.T, conn *gorm.DB, id uint64) {
	t.Helper()
	seedUser(t, conn, id, users.ActorTypeBot, false)
}

func seedUser(t *testing.T, conn *gorm.DB, id uint64, actorType int8, closed bool) {
	t.Helper()
	entity := users.EntityComplete{
		Id:          id,
		Username:    fmt.Sprintf("stats-user-%d", id),
		Nickname:    fmt.Sprintf("统计用户%d", id),
		IsActivated: users.ActivationSuccess,
		ActorType:   actorType,
	}
	if closed {
		entity.DeletedAt = gorm.DeletedAt{Time: statsAt(0, 0, 0, 0), Valid: true}
	}
	if err := conn.Create(&entity).Error; err != nil {
		t.Fatalf("create user %d: %v", id, err)
	}
}

func seedCategory(t *testing.T, conn *gorm.DB, id uint64, name string, sort int) {
	t.Helper()
	if err := conn.Create(&category.Entity{Id: id, Name: name, Sort: sort}).Error; err != nil {
		t.Fatalf("create category %d: %v", id, err)
	}
}

type statsTopicSpec struct {
	id            uint64
	userId        uint64
	categories    []uint64
	createdAt     time.Time
	firstPublicAt *time.Time
	estimated     bool
	likeCount     uint64
	personaUid    string
	topicType     int8
	processStatus int8
	visibility    string
	deleted       bool
	draft         bool
}

// seedTopic 建主题、板块归属与首楼（post_no=1）：首楼在任何回复统计里都必须被排除。
func seedTopic(t *testing.T, conn *gorm.DB, spec statsTopicSpec) {
	t.Helper()
	visibility := spec.visibility
	if visibility == "" {
		visibility = topics.VisibilityActive
	}
	status := int8(1) // 未发布草稿（status=0）不算公开内容
	if spec.draft {
		status = 0
	}
	entity := topics.Entity{
		Id:                   spec.id,
		Title:                fmt.Sprintf("主题 %d", spec.id),
		UserId:               spec.userId,
		Status:               status,
		ProcessStatus:        spec.processStatus,
		TopicType:            spec.topicType,
		VisibilityStatus:     visibility,
		RetentionStatus:      topics.RetentionNormal,
		FirstPostId:          spec.id + firstPostIdOffset,
		PostCount:            1,
		PostSeq:              1,
		LikeCount:            spec.likeCount,
		PersonaUID:           spec.personaUid,
		CreatedAt:            spec.createdAt,
		UpdatedAt:            spec.createdAt,
		FirstPublicAt:        spec.firstPublicAt,
		FirstPublicEstimated: spec.estimated,
	}
	if spec.deleted {
		entity.DeletedAt = gorm.DeletedAt{Time: spec.createdAt.Add(time.Hour), Valid: true}
	}
	if err := conn.Create(&entity).Error; err != nil {
		t.Fatalf("create topic %d: %v", spec.id, err)
	}
	for _, categoryId := range spec.categories {
		if err := conn.Create(&topicCategoryIndex.Entity{
			TopicId: spec.id, CategoryId: categoryId, Effective: 1,
		}).Error; err != nil {
			t.Fatalf("index topic %d in category %d: %v", spec.id, categoryId, err)
		}
	}
	seedPost(t, conn, statsPostSpec{
		id:            spec.id + firstPostIdOffset,
		topicId:       spec.id,
		userId:        spec.userId,
		postNo:        1,
		createdAt:     spec.createdAt,
		firstPublicAt: spec.firstPublicAt,
	})
}

type statsPostSpec struct {
	id            uint64
	topicId       uint64
	userId        uint64
	postNo        uint64
	createdAt     time.Time
	firstPublicAt *time.Time
	estimated     bool
	anonymous     bool
	personaUid    string
	visibility    string
	processStatus int8
	deleted       bool
}

func seedPost(t *testing.T, conn *gorm.DB, spec statsPostSpec) {
	t.Helper()
	visibility := spec.visibility
	if visibility == "" {
		visibility = posts.VisibilityActive
	}
	entity := posts.Entity{
		Id:                   spec.id,
		TopicId:              spec.topicId,
		PostNo:               spec.postNo,
		UserId:               spec.userId,
		Content:              fmt.Sprintf("楼层 %d", spec.id),
		ProcessStatus:        spec.processStatus,
		VisibilityStatus:     visibility,
		RetentionStatus:      posts.RetentionNormal,
		IsAnonymous:          spec.anonymous,
		PersonaUID:           spec.personaUid,
		CreatedAt:            spec.createdAt,
		UpdatedAt:            spec.createdAt,
		FirstPublicAt:        spec.firstPublicAt,
		FirstPublicEstimated: spec.estimated,
	}
	if spec.deleted {
		entity.DeletedAt = gorm.DeletedAt{Time: spec.createdAt.Add(time.Hour), Valid: true}
	}
	if err := conn.Create(&entity).Error; err != nil {
		t.Fatalf("create post %d: %v", spec.id, err)
	}
}

// seedPostLike 写入点赞行：likedAt 为 nil 表示已取消点赞（行保留、列置空）。
func seedPostLike(t *testing.T, conn *gorm.DB, postId, userId uint64, likedAt *time.Time) {
	t.Helper()
	if err := conn.Create(&postUserAction.Entity{PostId: postId, UserId: userId, LikedAt: likedAt}).Error; err != nil {
		t.Fatalf("like post %d by %d: %v", postId, userId, err)
	}
}

func timePtr(value time.Time) *time.Time { return &value }

func findCategory(t *testing.T, result CategorySummaryResult, categoryId uint64) CategorySummaryItem {
	t.Helper()
	for _, item := range result.Categories {
		if item.CategoryId == categoryId {
			return item
		}
	}
	t.Fatalf("category %d missing from summary", categoryId)
	return CategorySummaryItem{}
}

func assertCategoryCounts(t *testing.T, item CategorySummaryItem, topics, replies, contributors, agents int64) {
	t.Helper()
	if item.TopicCount != topics || item.ReplyCount != replies || item.ContributorCount != contributors || item.AgentContributorCount != agents {
		t.Fatalf("category %d = topics %d replies %d contributors %d agents %d, want %d/%d/%d/%d",
			item.CategoryId, item.TopicCount, item.ReplyCount, item.ContributorCount, item.AgentContributorCount,
			topics, replies, contributors, agents)
	}
}

func seedCategorySummaryFixtures(t *testing.T, conn *gorm.DB) {
	t.Helper()
	seedCategory(t, conn, categoryMain, "综合", 1)
	seedCategory(t, conn, categoryStudy, "学习", 2)
	seedCategory(t, conn, categoryEmpty, "空板块", 3)
	seedHuman(t, conn, 1)
	seedHuman(t, conn, 2)
	seedHuman(t, conn, 3)
	seedAgent(t, conn, 10)

	// 跨两个板块的主题：主题量在每个所属板块各计一次，全站合计只计一次。
	seedTopic(t, conn, statsTopicSpec{
		id: 100, userId: 1, categories: []uint64{categoryMain, categoryStudy},
		createdAt: statsAt(0, 10, 0, 0), firstPublicAt: timePtr(statsAt(0, 10, 0, 0)),
	})
	seedTopic(t, conn, statsTopicSpec{
		id: 101, userId: 2, categories: []uint64{categoryMain},
		createdAt: statsAt(1, 12, 0, 0), firstPublicAt: timePtr(statsAt(1, 12, 0, 0)),
	})
	seedTopic(t, conn, statsTopicSpec{
		id: 102, userId: 10, categories: []uint64{categoryMain},
		createdAt: statsAt(1, 13, 0, 0), firstPublicAt: timePtr(statsAt(1, 13, 0, 0)),
	})
	// 首楼不算回复：只有 post_no > 1 的楼层计入。
	seedPost(t, conn, statsPostSpec{
		id: 200, topicId: 100, userId: 2, postNo: 2,
		createdAt: statsAt(0, 11, 0, 0), firstPublicAt: timePtr(statsAt(0, 11, 0, 0)),
	})
	seedPost(t, conn, statsPostSpec{
		id: 201, topicId: 100, userId: 3, postNo: 3,
		createdAt: statsAt(1, 9, 0, 0), firstPublicAt: timePtr(statsAt(1, 9, 0, 0)),
	})
	seedPost(t, conn, statsPostSpec{
		id: 202, topicId: 101, userId: 1, postNo: 2,
		createdAt: statsAt(2, 10, 0, 0), firstPublicAt: timePtr(statsAt(2, 10, 0, 0)),
	})
}

// TestCategorySummaryCountsAndContributorDedup 覆盖 AC-2：板块计数、跨板块重复归属、
// 首楼不计回复、贡献人按账号去重、真人/Agent 分列。
func TestStatsCategorySummaryCountsAndContributorDedup(t *testing.T) {
	conn := setupStatsTestDB(t)
	seedCategorySummaryFixtures(t, conn)

	result, err := CategorySummary(context.Background(), statsDate(0), statsDate(2))
	if err != nil {
		t.Fatalf("category summary: %v", err)
	}
	if len(result.Categories) != 3 {
		t.Fatalf("categories = %d, want 3 (empty categories must still be listed)", len(result.Categories))
	}
	// 板块1：3 主题；回复 3 条；贡献人 {1,2,10}∪{2,3,1} 去重后真人 3、Agent 1。
	assertCategoryCounts(t, findCategory(t, result, categoryMain), 3, 3, 3, 1)
	// 板块2：只有跨板块主题 100 与其 2 条回复。
	assertCategoryCounts(t, findCategory(t, result, categoryStudy), 1, 2, 3, 0)
	assertCategoryCounts(t, findCategory(t, result, categoryEmpty), 0, 0, 0, 0)
	if name := findCategory(t, result, categoryEmpty).Name; name != "空板块" {
		t.Fatalf("empty category name = %q", name)
	}
	// 全站合计按内容主键去重：跨板块主题只计一次。
	if result.Totals.TopicCount != 3 || result.Totals.ReplyCount != 3 {
		t.Fatalf("totals = %+v, want 3 topics / 3 replies", result.Totals)
	}
	if result.Totals.ContributorCount != 3 || result.Totals.AgentContributorCount != 1 {
		t.Fatalf("totals contributors = %+v, want 3 humans / 1 agent", result.Totals)
	}
	if result.Coverage.MaxRangeDays != MaxRangeDays {
		t.Fatalf("coverage = %+v", result.Coverage)
	}
	if !strings.Contains(result.Coverage.Timezone, " (UTC") || !strings.HasSuffix(result.Coverage.Timezone, ")") {
		t.Fatalf("coverage.timezone = %q, want the '<abbr> (UTC±HH:MM)' shape", result.Coverage.Timezone)
	}
	if result.Coverage.Estimated {
		t.Fatalf("coverage.estimated = true, want false for exact first_public_at data")
	}
}

// TestCategorySummaryExcludesAnonymousAndClosedAccounts 覆盖 AC-5：匿名行与已注销账号
// 不计入贡献人数，但其已公开内容仍计入内容量与全站合计。
func TestStatsCategorySummaryExcludesAnonymousAndClosedAccounts(t *testing.T) {
	conn := setupStatsTestDB(t)
	seedCategory(t, conn, categoryMain, "综合", 1)
	seedHuman(t, conn, 1)
	seedHuman(t, conn, 12)
	seedUser(t, conn, 11, users.ActorTypeHuman, true)

	seedTopic(t, conn, statsTopicSpec{
		id: 100, userId: 1, categories: []uint64{categoryMain},
		createdAt: statsAt(0, 10, 0, 0), firstPublicAt: timePtr(statsAt(0, 10, 0, 0)),
	})
	// 已注销账号的主题：内容仍公开，计入内容量；作者不进入贡献人。
	seedTopic(t, conn, statsTopicSpec{
		id: 101, userId: 11, categories: []uint64{categoryMain},
		createdAt: statsAt(0, 11, 0, 0), firstPublicAt: timePtr(statsAt(0, 11, 0, 0)),
	})
	// 匿名（persona）主题：同样只计内容，不归因到任何账号。
	seedTopic(t, conn, statsTopicSpec{
		id: 102, userId: 1, categories: []uint64{categoryMain}, personaUid: "persona-1",
		createdAt: statsAt(0, 12, 0, 0), firstPublicAt: timePtr(statsAt(0, 12, 0, 0)),
	})
	seedPost(t, conn, statsPostSpec{
		id: 200, topicId: 100, userId: 11, postNo: 2,
		createdAt: statsAt(0, 13, 0, 0), firstPublicAt: timePtr(statsAt(0, 13, 0, 0)),
	})
	seedPost(t, conn, statsPostSpec{
		id: 201, topicId: 100, userId: 12, postNo: 3, anonymous: true,
		createdAt: statsAt(0, 14, 0, 0), firstPublicAt: timePtr(statsAt(0, 14, 0, 0)),
	})
	seedPost(t, conn, statsPostSpec{
		id: 202, topicId: 100, userId: 12, postNo: 4, personaUid: "persona-2",
		createdAt: statsAt(0, 15, 0, 0), firstPublicAt: timePtr(statsAt(0, 15, 0, 0)),
	})

	result, err := CategorySummary(context.Background(), statsDate(0), statsDate(0))
	if err != nil {
		t.Fatalf("category summary: %v", err)
	}
	item := findCategory(t, result, categoryMain)
	assertCategoryCounts(t, item, 3, 3, 1, 0)
	if result.Totals.TopicCount != 3 || result.Totals.ReplyCount != 3 {
		t.Fatalf("totals = %+v, want anonymous and closed-author content still counted", result.Totals)
	}
	if result.Totals.ContributorCount != 1 {
		t.Fatalf("contributors = %d, want only the active human author", result.Totals.ContributorCount)
	}
}

// TestCategorySummaryContentScopeExcludesUnpublished 覆盖 AC-4：删除、封禁、
// 待审、wiki 页面与工作区外内容都不进入统计。
func TestStatsCategorySummaryContentScopeExcludesUnpublished(t *testing.T) {
	conn := setupStatsTestDB(t)
	seedCategory(t, conn, categoryMain, "综合", 1)
	seedHuman(t, conn, 1)

	seedTopic(t, conn, statsTopicSpec{
		id: 100, userId: 1, categories: []uint64{categoryMain},
		createdAt: statsAt(0, 10, 0, 0), firstPublicAt: timePtr(statsAt(0, 10, 0, 0)),
	})
	seedTopic(t, conn, statsTopicSpec{
		id: 101, userId: 1, categories: []uint64{categoryMain}, deleted: true,
		createdAt: statsAt(0, 11, 0, 0), firstPublicAt: timePtr(statsAt(0, 11, 0, 0)),
	})
	seedTopic(t, conn, statsTopicSpec{
		id: 102, userId: 1, categories: []uint64{categoryMain}, visibility: topics.VisibilityUserDeleted,
		createdAt: statsAt(0, 11, 0, 0), firstPublicAt: timePtr(statsAt(0, 11, 0, 0)),
	})
	seedTopic(t, conn, statsTopicSpec{
		id: 103, userId: 1, categories: []uint64{categoryMain}, processStatus: topics.ProcessStatusBlocked,
		createdAt: statsAt(0, 12, 0, 0), firstPublicAt: timePtr(statsAt(0, 12, 0, 0)),
	})
	seedTopic(t, conn, statsTopicSpec{
		id: 104, userId: 1, categories: []uint64{categoryMain}, topicType: topics.TopicTypeWiki,
		createdAt: statsAt(0, 13, 0, 0), firstPublicAt: timePtr(statsAt(0, 13, 0, 0)),
	})
	// 未发布草稿（status=0）：即使被回填过估算的首次公开时间也不算公开内容。
	seedTopic(t, conn, statsTopicSpec{
		id: 105, userId: 1, categories: []uint64{categoryMain}, draft: true,
		createdAt: statsAt(0, 13, 30, 0),
	})

	// 回复侧：删除/封禁楼层与已删除主题下的楼层都不计入。
	seedPost(t, conn, statsPostSpec{
		id: 200, topicId: 100, userId: 1, postNo: 2, deleted: true,
		createdAt: statsAt(0, 14, 0, 0), firstPublicAt: timePtr(statsAt(0, 14, 0, 0)),
	})
	seedPost(t, conn, statsPostSpec{
		id: 201, topicId: 100, userId: 1, postNo: 3, processStatus: posts.ProcessStatusBlocked,
		createdAt: statsAt(0, 15, 0, 0), firstPublicAt: timePtr(statsAt(0, 15, 0, 0)),
	})
	seedPost(t, conn, statsPostSpec{
		id: 202, topicId: 102, userId: 1, postNo: 2,
		createdAt: statsAt(0, 16, 0, 0), firstPublicAt: timePtr(statsAt(0, 16, 0, 0)),
	})
	seedPost(t, conn, statsPostSpec{
		id: 203, topicId: 100, userId: 1, postNo: 4,
		createdAt: statsAt(0, 17, 0, 0), firstPublicAt: timePtr(statsAt(0, 17, 0, 0)),
	})

	result, err := CategorySummary(context.Background(), statsDate(0), statsDate(0))
	if err != nil {
		t.Fatalf("category summary: %v", err)
	}
	assertCategoryCounts(t, findCategory(t, result, categoryMain), 1, 1, 1, 0)
}

// TestCategoryTrendDaysAndEstimated 覆盖 AC-7 与 AC-4 的午夜边界：
// 区间含两端按本地日切片；缺失日为 0 且 estimated=false；估算来源日 estimated=true。
func TestStatsCategoryTrendDaysAndEstimated(t *testing.T) {
	conn := setupStatsTestDB(t)
	seedCategory(t, conn, categoryMain, "综合", 1)
	seedHuman(t, conn, 1)
	seedHuman(t, conn, 2)

	seedTopic(t, conn, statsTopicSpec{ // 当日 23:59:59，仍属 day0
		id: 100, userId: 1, categories: []uint64{categoryMain},
		createdAt: statsAt(0, 23, 59, 59), firstPublicAt: timePtr(statsAt(0, 23, 59, 59)),
	})
	seedTopic(t, conn, statsTopicSpec{ // day1 00:00:00，属 day1
		id: 101, userId: 1, categories: []uint64{categoryMain},
		createdAt: statsAt(1, 0, 0, 0), firstPublicAt: timePtr(statsAt(1, 0, 0, 0)),
	})
	seedTopic(t, conn, statsTopicSpec{ // 首次公开时间缺失：估算来源，落在 day1
		id: 102, userId: 1, categories: []uint64{categoryMain},
		createdAt: statsAt(1, 8, 0, 0),
	})
	seedTopic(t, conn, statsTopicSpec{ // day(-1) 23:59:59：区间之外
		id: 103, userId: 1, categories: []uint64{categoryMain},
		createdAt: statsAt(-1, 23, 59, 59), firstPublicAt: timePtr(statsAt(-1, 23, 59, 59)),
	})
	seedTopic(t, conn, statsTopicSpec{ // 区间右界（day3 00:00）之后：左闭右开被排除
		id: 104, userId: 1, categories: []uint64{categoryMain},
		createdAt: statsAt(3, 0, 0, 0), firstPublicAt: timePtr(statsAt(3, 0, 0, 0)),
	})
	seedPost(t, conn, statsPostSpec{ // day2 的回复
		id: 200, topicId: 100, userId: 2, postNo: 2,
		createdAt: statsAt(2, 10, 0, 0), firstPublicAt: timePtr(statsAt(2, 10, 0, 0)),
	})
	seedPost(t, conn, statsPostSpec{ // 延迟审核：created_at 在 day0，首次公开在 day2
		id: 201, topicId: 100, userId: 2, postNo: 3,
		createdAt: statsAt(0, 10, 0, 0), firstPublicAt: timePtr(statsAt(2, 12, 0, 0)),
	})

	result, err := CategoryTrend(context.Background(), categoryMain, statsDate(0), statsDate(2))
	if err != nil {
		t.Fatalf("category trend: %v", err)
	}
	if result.CategoryId != categoryMain {
		t.Fatalf("categoryId = %d", result.CategoryId)
	}
	if len(result.Days) != 3 {
		t.Fatalf("days = %d, want 3 (%+v)", len(result.Days), result.Days)
	}
	assertTrendDay(t, result.Days[0], statsDate(0), 1, 0, false)
	assertTrendDay(t, result.Days[1], statsDate(1), 2, 0, true)
	assertTrendDay(t, result.Days[2], statsDate(2), 0, 2, false)
	if !result.Coverage.Estimated {
		t.Fatalf("coverage.estimated = false, want true when the range contains estimated attribution")
	}

	// 缺省区间（近 7 天到当天）固定为 8 天，且不含 2026-03 的造数。
	defaulted, err := CategoryTrend(context.Background(), categoryMain, "", "")
	if err != nil {
		t.Fatalf("default trend: %v", err)
	}
	if len(defaulted.Days) != defaultRangeDays+1 {
		t.Fatalf("default days = %d, want %d", len(defaulted.Days), defaultRangeDays+1)
	}
	for _, day := range defaulted.Days {
		if day.TopicCount != 0 || day.ReplyCount != 0 {
			t.Fatalf("default range day %s = %+v, want empty", day.Date, day)
		}
	}
}

func assertTrendDay(t *testing.T, day TrendDay, date string, topics, replies int64, estimated bool) {
	t.Helper()
	if day.Date != date || day.TopicCount != topics || day.ReplyCount != replies || day.Estimated != estimated {
		t.Fatalf("day = %+v, want date %s topics %d replies %d estimated %t", day, date, topics, replies, estimated)
	}
}

// TestCategoryTrendFollowsCurrentOwnershipAndVisibility 覆盖 AC-4：
// 主题移动后按当前板块归属统计，删除后归零、恢复后重新计入。
func TestStatsCategoryTrendFollowsCurrentOwnershipAndVisibility(t *testing.T) {
	conn := setupStatsTestDB(t)
	seedCategory(t, conn, categoryMain, "综合", 1)
	seedCategory(t, conn, categoryStudy, "学习", 2)
	seedHuman(t, conn, 1)
	seedTopic(t, conn, statsTopicSpec{
		id: 100, userId: 1, categories: []uint64{categoryMain},
		createdAt: statsAt(0, 10, 0, 0), firstPublicAt: timePtr(statsAt(0, 10, 0, 0)),
	})

	assertTrendTopics(t, categoryMain, statsDate(0), statsDate(0), 1)
	assertTrendTopics(t, categoryStudy, statsDate(0), statsDate(0), 0)

	if err := conn.Model(&topicCategoryIndex.Entity{}).
		Where("topic_id = ? AND category_id = ?", 100, categoryMain).
		Update("effective", 0).Error; err != nil {
		t.Fatalf("move topic out of category: %v", err)
	}
	if err := conn.Create(&topicCategoryIndex.Entity{TopicId: 100, CategoryId: categoryStudy, Effective: 1}).Error; err != nil {
		t.Fatalf("move topic into category: %v", err)
	}
	assertTrendTopics(t, categoryMain, statsDate(0), statsDate(0), 0)
	assertTrendTopics(t, categoryStudy, statsDate(0), statsDate(0), 1)

	if err := conn.Model(&topics.Entity{}).Where("id = ?", 100).
		Update("visibility_status", topics.VisibilityUserDeleted).Error; err != nil {
		t.Fatalf("hide topic: %v", err)
	}
	assertTrendTopics(t, categoryStudy, statsDate(0), statsDate(0), 0)

	if err := conn.Model(&topics.Entity{}).Where("id = ?", 100).
		Update("visibility_status", topics.VisibilityActive).Error; err != nil {
		t.Fatalf("restore topic: %v", err)
	}
	assertTrendTopics(t, categoryStudy, statsDate(0), statsDate(0), 1)
}

func assertTrendTopics(t *testing.T, categoryId uint64, start, end string, want int64) {
	t.Helper()
	result, err := CategoryTrend(context.Background(), categoryId, start, end)
	if err != nil {
		t.Fatalf("category trend: %v", err)
	}
	if len(result.Days) != 1 || result.Days[0].TopicCount != want {
		t.Fatalf("category %d topics = %+v, want %d", categoryId, result.Days, want)
	}
}

// TestStatsCategorySummaryIgnoresEdits 覆盖 AC-4 的「编辑不重复新增」：编辑只改行内容，
// 不新增内容行，也不改变首次公开时间归属。
func TestStatsCategorySummaryIgnoresEdits(t *testing.T) {
	conn := setupStatsTestDB(t)
	seedCategory(t, conn, categoryMain, "综合", 1)
	seedHuman(t, conn, 1)
	seedHuman(t, conn, 2)
	seedTopic(t, conn, statsTopicSpec{
		id: 100, userId: 1, categories: []uint64{categoryMain},
		createdAt: statsAt(0, 10, 0, 0), firstPublicAt: timePtr(statsAt(0, 10, 0, 0)),
	})
	seedPost(t, conn, statsPostSpec{
		id: 200, topicId: 100, userId: 2, postNo: 2,
		createdAt: statsAt(0, 11, 0, 0), firstPublicAt: timePtr(statsAt(0, 11, 0, 0)),
	})

	before, err := CategorySummary(context.Background(), statsDate(0), statsDate(0))
	if err != nil {
		t.Fatalf("category summary before edit: %v", err)
	}
	editedAt := statsAt(1, 9, 0, 0)
	if err := conn.Model(&topics.Entity{}).Where("id = ?", 100).
		Updates(map[string]any{"title": "改过的标题", "updated_at": editedAt}).Error; err != nil {
		t.Fatalf("edit topic: %v", err)
	}
	if err := conn.Model(&posts.Entity{}).Where("id = ?", 200).
		Updates(map[string]any{"content": "改过的回复", "updated_at": editedAt, "last_edited_at": editedAt}).Error; err != nil {
		t.Fatalf("edit reply: %v", err)
	}

	after, err := CategorySummary(context.Background(), statsDate(0), statsDate(0))
	if err != nil {
		t.Fatalf("category summary after edit: %v", err)
	}
	if after.Totals != before.Totals {
		t.Fatalf("totals after edit = %+v, want unchanged %+v", after.Totals, before.Totals)
	}
	assertCategoryCounts(t, findCategory(t, after, categoryMain), 1, 1, 2, 0)
}

// TestStatsCategorySummaryOrdersByActivity 覆盖契约与产品文档冻结的板块排序：
// 主题数降序、同值按板块 ID 升序（零贡献板块沉底），且与板块自身的 sort 权重无关。
func TestStatsCategorySummaryOrdersByActivity(t *testing.T) {
	conn := setupStatsTestDB(t)
	// sort 权重与 id 顺序相反：若按 category.All() 原序返回，顺序会是 3、2、1。
	seedCategory(t, conn, categoryMain, "综合", 3)
	seedCategory(t, conn, categoryStudy, "学习", 2)
	seedCategory(t, conn, categoryEmpty, "空板块", 1)
	seedHuman(t, conn, 1)

	seedTopic(t, conn, statsTopicSpec{
		id: 100, userId: 1, categories: []uint64{categoryMain},
		createdAt: statsAt(0, 10, 0, 0), firstPublicAt: timePtr(statsAt(0, 10, 0, 0)),
	})
	seedTopic(t, conn, statsTopicSpec{
		id: 101, userId: 1, categories: []uint64{categoryStudy},
		createdAt: statsAt(0, 11, 0, 0), firstPublicAt: timePtr(statsAt(0, 11, 0, 0)),
	})
	seedTopic(t, conn, statsTopicSpec{
		id: 102, userId: 1, categories: []uint64{categoryEmpty},
		createdAt: statsAt(0, 12, 0, 0), firstPublicAt: timePtr(statsAt(0, 12, 0, 0)),
	})
	seedTopic(t, conn, statsTopicSpec{
		id: 103, userId: 1, categories: []uint64{categoryEmpty},
		createdAt: statsAt(0, 13, 0, 0), firstPublicAt: timePtr(statsAt(0, 13, 0, 0)),
	})

	result, err := CategorySummary(context.Background(), statsDate(0), statsDate(0))
	if err != nil {
		t.Fatalf("category summary: %v", err)
	}
	// 空板块（id=3）有 2 个主题排首位；综合(1) 与学习(2) 各 1 个主题，按 id 升序。
	wantOrder := []uint64{categoryEmpty, categoryMain, categoryStudy}
	for i, want := range wantOrder {
		if result.Categories[i].CategoryId != want {
			t.Fatalf("categories[%d].categoryId = %d, want %d (order %+v)",
				i, result.Categories[i].CategoryId, want, result.Categories)
		}
	}
	if result.Categories[0].TopicCount != 2 || result.Categories[1].TopicCount != 1 {
		t.Fatalf("categories = %+v, want 2/1/1 topics", result.Categories)
	}
}

// TestStatsContentScopeFollowsFirstPostVisibility 覆盖 AC-4：首楼不再可见（进入删除生命周期
// 或封禁）后主题与其回复一起退出统计，恢复后重新计入。
func TestStatsContentScopeFollowsFirstPostVisibility(t *testing.T) {
	conn := setupStatsTestDB(t)
	seedCategory(t, conn, categoryMain, "综合", 1)
	seedHuman(t, conn, 1)
	seedHuman(t, conn, 2)
	seedTopic(t, conn, statsTopicSpec{
		id: 100, userId: 1, categories: []uint64{categoryMain},
		createdAt: statsAt(0, 10, 0, 0), firstPublicAt: timePtr(statsAt(0, 10, 0, 0)),
	})
	seedPost(t, conn, statsPostSpec{
		id: 200, topicId: 100, userId: 2, postNo: 2,
		createdAt: statsAt(0, 11, 0, 0), firstPublicAt: timePtr(statsAt(0, 11, 0, 0)),
	})
	assertCategorySummaryCounts(t, 1, 1, 2)

	firstPostID := uint64(100) + firstPostIdOffset
	updateFirstPost := func(fields map[string]any) {
		t.Helper()
		if err := conn.Model(&posts.Entity{}).Where("id = ?", firstPostID).Updates(fields).Error; err != nil {
			t.Fatalf("update first post: %v", err)
		}
	}

	updateFirstPost(map[string]any{"visibility_status": posts.VisibilityUserDeleted})
	assertCategorySummaryCounts(t, 0, 0, 0)

	updateFirstPost(map[string]any{"visibility_status": posts.VisibilityActive, "process_status": posts.ProcessStatusBlocked})
	assertCategorySummaryCounts(t, 0, 0, 0)

	updateFirstPost(map[string]any{"process_status": posts.ProcessStatusNormal})
	assertCategorySummaryCounts(t, 1, 1, 2)
}

func assertCategorySummaryCounts(t *testing.T, topicsCount, repliesCount, contributors int64) {
	t.Helper()
	result, err := CategorySummary(context.Background(), statsDate(0), statsDate(0))
	if err != nil {
		t.Fatalf("category summary: %v", err)
	}
	assertCategoryCounts(t, findCategory(t, result, categoryMain), topicsCount, repliesCount, contributors, 0)
	if result.Totals.TopicCount != topicsCount || result.Totals.ReplyCount != repliesCount {
		t.Fatalf("totals = %+v, want %d topics / %d replies", result.Totals, topicsCount, repliesCount)
	}
}
