package statsservice

import (
	"context"
	"errors"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"gorm.io/gorm"
)

func seedRankingFixtures(t *testing.T, conn *gorm.DB) {
	t.Helper()
	seedCategory(t, conn, categoryMain, "综合", 1)
	seedHuman(t, conn, 1)
	seedHuman(t, conn, 2)
	seedHuman(t, conn, 99) // 零贡献账号不得出现
	seedAgent(t, conn, 10)

	// 先造 id 较大的账号（2）的内容、后造 id 较少的账号（1）的内容：排行丢掉
	// `user_id ASC` 兜底时，按行自然顺序会先出现账号 2，因此「同值按 user_id 升序」
	// 的断言只有在真的带稳定序时才会通过。
	seedTopic(t, conn, statsTopicSpec{
		id: 101, userId: 2, categories: []uint64{categoryMain}, likeCount: 7,
		createdAt: statsAt(0, 11, 0, 0), firstPublicAt: timePtr(statsAt(0, 11, 0, 0)),
	})
	seedPost(t, conn, statsPostSpec{
		id: 201, topicId: 101, userId: 2, postNo: 2,
		createdAt: statsAt(0, 14, 0, 0), firstPublicAt: timePtr(statsAt(0, 14, 0, 0)),
	})
	seedTopic(t, conn, statsTopicSpec{
		id: 100, userId: 1, categories: []uint64{categoryMain}, likeCount: 5,
		createdAt: statsAt(0, 10, 0, 0), firstPublicAt: timePtr(statsAt(0, 10, 0, 0)),
	})
	seedPost(t, conn, statsPostSpec{
		id: 200, topicId: 100, userId: 1, postNo: 2,
		createdAt: statsAt(0, 13, 0, 0), firstPublicAt: timePtr(statsAt(0, 13, 0, 0)),
	})
	seedPost(t, conn, statsPostSpec{
		id: 202, topicId: 101, userId: 1, postNo: 3,
		createdAt: statsAt(0, 15, 0, 0), firstPublicAt: timePtr(statsAt(0, 15, 0, 0)),
	})
	seedTopic(t, conn, statsTopicSpec{
		id: 102, userId: 10, categories: []uint64{categoryMain}, likeCount: 1,
		createdAt: statsAt(0, 12, 0, 0), firstPublicAt: timePtr(statsAt(0, 12, 0, 0)),
	})
	// 获赞（R5）：当前仍有效的点赞才计入，取消点赞只把 liked_at 置空。
	seedPostLike(t, conn, 200, 2, timePtr(statsAt(0, 16, 0, 0)))
	seedPostLike(t, conn, 201, 1, nil)
}

// TestUserRankingOrderingPagingAndActors 覆盖 AC-3 与 AC-6：排序稳定（同值按 user_id 升序）、
// 分页与总数一致、真人/Agent 分列、零贡献账号不出现。
func TestStatsUserRankingOrderingPagingAndActors(t *testing.T) {
	conn := setupStatsTestDB(t)
	seedRankingFixtures(t, conn)

	byTopics, err := UserRanking(context.Background(), RankingQuery{StartDate: statsDate(0), EndDate: statsDate(0)})
	if err != nil {
		t.Fatalf("ranking: %v", err)
	}
	if byTopics.Total != 2 || byTopics.Page != 1 || byTopics.PageSize != DefaultPageSize {
		t.Fatalf("ranking meta = %+v, want total 2 page 1 size %d", byTopics, DefaultPageSize)
	}
	// 获赞（R5）= 本人主题的 like_count 之和 + 其回复上当前有效的点赞数：
	// 账号 1 = 5 + 1（回复 200 的点赞），账号 2 = 7 + 0（回复 201 的点赞已取消）。
	// 两个账号主题数同为 1：断言 [1, 2] 即是「同值按 user_id 升序」的稳定序证明。
	assertRanking(t, byTopics.Items, []rankingWant{
		{userId: 1, topics: 1, replies: 2, likes: 6},
		{userId: 2, topics: 1, replies: 1, likes: 7},
	})

	byReplies, err := UserRanking(context.Background(), RankingQuery{
		StartDate: statsDate(0), EndDate: statsDate(0), SortBy: "replies",
	})
	if err != nil {
		t.Fatalf("ranking by replies: %v", err)
	}
	assertRanking(t, byReplies.Items, []rankingWant{
		{userId: 1, topics: 1, replies: 2, likes: 6},
		{userId: 2, topics: 1, replies: 1, likes: 7},
	})

	byLikes, err := UserRanking(context.Background(), RankingQuery{
		StartDate: statsDate(0), EndDate: statsDate(0), SortBy: "likes",
	})
	if err != nil {
		t.Fatalf("ranking by likes: %v", err)
	}
	assertRanking(t, byLikes.Items, []rankingWant{
		{userId: 2, topics: 1, replies: 1, likes: 7},
		{userId: 1, topics: 1, replies: 2, likes: 6},
	})

	firstPage, err := UserRanking(context.Background(), RankingQuery{
		StartDate: statsDate(0), EndDate: statsDate(0), Page: intPtr(1), PageSize: intPtr(1),
	})
	if err != nil {
		t.Fatalf("ranking page 1: %v", err)
	}
	if len(firstPage.Items) != 1 || firstPage.Items[0].UserId != 1 || firstPage.Total != 2 {
		t.Fatalf("page1 = %+v", firstPage.Items)
	}
	secondPage, err := UserRanking(context.Background(), RankingQuery{
		StartDate: statsDate(0), EndDate: statsDate(0), Page: intPtr(2), PageSize: intPtr(1),
	})
	if err != nil {
		t.Fatalf("ranking page 2: %v", err)
	}
	if len(secondPage.Items) != 1 || secondPage.Items[0].UserId != 2 {
		t.Fatalf("page2 = %+v", secondPage.Items)
	}
	if firstPage.Items[0].Username == "" || firstPage.Items[0].Nickname == "" {
		t.Fatalf("ranking item identity = %+v, want username and nickname", firstPage.Items[0])
	}

	agents, err := UserRanking(context.Background(), RankingQuery{
		StartDate: statsDate(0), EndDate: statsDate(0), ActorType: int(users.ActorTypeBot),
	})
	if err != nil {
		t.Fatalf("agent ranking: %v", err)
	}
	if agents.Total != 1 || len(agents.Items) != 1 || agents.Items[0].UserId != 10 || agents.Items[0].ActorType != users.ActorTypeBot {
		t.Fatalf("agent ranking = %+v, want only the bot account", agents)
	}
}

type rankingWant struct {
	userId  uint64
	topics  int64
	replies int64
	likes   int64
}

func assertRanking(t *testing.T, items []UserRankingItem, want []rankingWant) {
	t.Helper()
	if len(items) != len(want) {
		t.Fatalf("ranking items = %d (%+v), want %d", len(items), items, len(want))
	}
	for i, expected := range want {
		item := items[i]
		if item.UserId != expected.userId || item.TopicCount != expected.topics ||
			item.ReplyCount != expected.replies || item.LikeCount != expected.likes {
			t.Fatalf("ranking[%d] = %+v, want %+v", i, item, expected)
		}
	}
}

// TestUserDetailSummarySeriesAndErrors 覆盖 AC-4/AC-5/AC-6/AC-7：
// 明细的汇总与逐日序列、匿名内容不归因、无贡献全 0、账号不存在或已注销被拒绝。
func TestStatsUserDetailSummarySeriesAndErrors(t *testing.T) {
	conn := setupStatsTestDB(t)
	seedCategory(t, conn, categoryMain, "综合", 1)
	seedHuman(t, conn, 1)
	seedHuman(t, conn, 2)
	seedUser(t, conn, 11, users.ActorTypeHuman, true)

	seedTopic(t, conn, statsTopicSpec{
		id: 100, userId: 1, categories: []uint64{categoryMain}, likeCount: 3,
		createdAt: statsAt(0, 10, 0, 0), firstPublicAt: timePtr(statsAt(0, 10, 0, 0)),
	})
	seedPost(t, conn, statsPostSpec{
		id: 200, topicId: 100, userId: 1, postNo: 2,
		createdAt: statsAt(1, 10, 0, 0), firstPublicAt: timePtr(statsAt(1, 10, 0, 0)),
	})
	seedPost(t, conn, statsPostSpec{ // 本人匿名楼层：不进入本人明细
		id: 201, topicId: 100, userId: 1, postNo: 3, anonymous: true,
		createdAt: statsAt(1, 11, 0, 0), firstPublicAt: timePtr(statsAt(1, 11, 0, 0)),
	})
	seedPost(t, conn, statsPostSpec{ // 估算来源日：首次公开时间缺失
		id: 202, topicId: 100, userId: 1, postNo: 4,
		createdAt: statsAt(2, 12, 0, 0),
	})
	seedPostLike(t, conn, 200, 2, timePtr(statsAt(1, 12, 0, 0)))
	seedPostLike(t, conn, 202, 2, nil)

	result, err := UserDetail(context.Background(), DetailQuery{
		StartDate: statsDate(0), EndDate: statsDate(2), UserId: 1,
	})
	if err != nil {
		t.Fatalf("user detail: %v", err)
	}
	if result.User.UserId != 1 || result.User.Username != "stats-user-1" || result.User.ActorType != users.ActorTypeHuman {
		t.Fatalf("user = %+v", result.User)
	}
	if result.Summary.TopicCount != 1 || result.Summary.ReplyCount != 2 || result.Summary.LikeCount != 4 {
		t.Fatalf("summary = %+v, want 1 topic / 2 replies / 4 likes", result.Summary)
	}
	if len(result.Days) != 3 {
		t.Fatalf("days = %d, want 3", len(result.Days))
	}
	assertTrendDay(t, result.Days[0], statsDate(0), 1, 0, false)
	assertTrendDay(t, result.Days[1], statsDate(1), 0, 1, false)
	assertTrendDay(t, result.Days[2], statsDate(2), 0, 1, true)

	byName, err := UserDetail(context.Background(), DetailQuery{
		StartDate: statsDate(0), EndDate: statsDate(2), Username: "stats-user-1",
	})
	if err != nil {
		t.Fatalf("user detail by username: %v", err)
	}
	if byName.User.UserId != 1 {
		t.Fatalf("username lookup = %+v", byName.User)
	}

	empty, err := UserDetail(context.Background(), DetailQuery{
		StartDate: statsDate(0), EndDate: statsDate(2), UserId: 2,
	})
	if err != nil {
		t.Fatalf("user detail without contributions: %v", err)
	}
	if empty.Summary.TopicCount != 0 || empty.Summary.ReplyCount != 0 || empty.Summary.LikeCount != 0 {
		t.Fatalf("empty summary = %+v", empty.Summary)
	}
	for _, day := range empty.Days {
		if day.TopicCount != 0 || day.ReplyCount != 0 || day.Estimated {
			t.Fatalf("empty day = %+v, want 0 with estimated=false", day)
		}
	}

	assertMessageCode(t, userDetailError(t, DetailQuery{StartDate: statsDate(0), EndDate: statsDate(2), UserId: 4242}), component.MessageUserNotFound, "")
	assertMessageCode(t, userDetailError(t, DetailQuery{StartDate: statsDate(0), EndDate: statsDate(2), UserId: 11}), component.MessageUserNotFound, "")
}

func userDetailError(t *testing.T, query DetailQuery) error {
	t.Helper()
	_, err := UserDetail(context.Background(), query)
	if err == nil {
		t.Fatal("user detail: expected an error")
	}
	return err
}

// TestValidationRejectsInvalidParams 覆盖 AC-8：非法日期、倒置区间、超 366 天、
// 非法排序/主体类型/板块/分页与缺失用户标识都被明确拒绝。
func TestStatsValidationRejectsInvalidParams(t *testing.T) {
	conn := setupStatsTestDB(t)
	seedCategory(t, conn, categoryMain, "综合", 1)
	seedHuman(t, conn, 1)

	cases := []struct {
		name  string
		field string
		run   func() error
	}{
		{name: "invalid start date", field: "startDate", run: func() error {
			return summaryError("2026-3-1", statsDate(0))
		}},
		{name: "invalid end date", field: "endDate", run: func() error {
			return summaryError(statsDate(0), "2026-03-32")
		}},
		{name: "inverted range", field: "endDate", run: func() error {
			return summaryError(statsDate(2), statsDate(0))
		}},
		{name: "range over 366 days", field: "endDate", run: func() error {
			return summaryError(statsDate(0), statsDay.AddDate(0, 0, MaxRangeDays).Format("2006-01-02"))
		}},
		{name: "unknown category", field: "categoryId", run: func() error {
			_, err := CategoryTrend(context.Background(), 999, statsDate(0), statsDate(0))
			return err
		}},
		{name: "missing category", field: "categoryId", run: func() error {
			_, err := CategoryTrend(context.Background(), 0, statsDate(0), statsDate(0))
			return err
		}},
		{name: "unknown sort key", field: "sortBy", run: func() error {
			_, err := UserRanking(context.Background(), RankingQuery{SortBy: "views"})
			return err
		}},
		{name: "unknown actor type", field: "actorType", run: func() error {
			_, err := UserRanking(context.Background(), RankingQuery{ActorType: 2})
			return err
		}},
		{name: "page size over limit", field: "pageSize", run: func() error {
			_, err := UserRanking(context.Background(), RankingQuery{PageSize: intPtr(MaxPageSize + 1)})
			return err
		}},
		{name: "explicit zero page size", field: "pageSize", run: func() error {
			_, err := UserRanking(context.Background(), RankingQuery{PageSize: intPtr(0)})
			return err
		}},
		{name: "explicit zero page", field: "page", run: func() error {
			_, err := UserRanking(context.Background(), RankingQuery{Page: intPtr(0)})
			return err
		}},
		{name: "page over limit", field: "page", run: func() error {
			_, err := UserRanking(context.Background(), RankingQuery{Page: intPtr(MaxPage + 1), PageSize: intPtr(MaxPageSize)})
			return err
		}},
		{name: "detail without identity", field: "userId", run: func() error {
			_, err := UserDetail(context.Background(), DetailQuery{})
			return err
		}},
		{name: "detail with both identities", field: "username", run: func() error {
			_, err := UserDetail(context.Background(), DetailQuery{UserId: 1, Username: "stats-user-1"})
			return err
		}},
		{name: "ranking with unknown category", field: "categoryId", run: func() error {
			_, err := UserRanking(context.Background(), RankingQuery{CategoryId: 999})
			return err
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assertMessageCode(t, tc.run(), component.MessageRequestInvalidParams, tc.field)
		})
	}

	// 366 天（含首尾）是允许的上限，必须正常返回。
	if _, err := CategorySummary(context.Background(), statsDate(0), statsDay.AddDate(0, 0, MaxRangeDays-1).Format("2006-01-02")); err != nil {
		t.Fatalf("366-day range rejected: %v", err)
	}
}

func summaryError(startDate, endDate string) error {
	_, err := CategorySummary(context.Background(), startDate, endDate)
	return err
}

func intPtr(value int) *int { return &value }

func assertMessageCode(t *testing.T, err error, want component.MessageCode, wantField string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error %s, got nil", want)
	}
	var messageErr component.MessageError
	if !errors.As(err, &messageErr) {
		t.Fatalf("error = %v, want MessageError %s", err, want)
	}
	if messageErr.Code != want {
		t.Fatalf("messageCode = %s, want %s", messageErr.Code, want)
	}
	if wantField != "" && messageErr.Params["field"] != wantField {
		t.Fatalf("params.field = %v, want %s", messageErr.Params["field"], wantField)
	}
}

// TestStatsUserRankingEstimatedFollowsCountedRows 覆盖评审 S1：coverage.estimated 只描述
// 「本次统计计入的行」——只有 Agent 的行是估算来源时，真人排行与真人明细都不得误报。
func TestStatsUserRankingEstimatedFollowsCountedRows(t *testing.T) {
	conn := setupStatsTestDB(t)
	seedCategory(t, conn, categoryMain, "综合", 1)
	seedHuman(t, conn, 1)
	seedAgent(t, conn, 10)
	seedTopic(t, conn, statsTopicSpec{ // 真人：精确首次公开时间
		id: 100, userId: 1, categories: []uint64{categoryMain},
		createdAt: statsAt(0, 10, 0, 0), firstPublicAt: timePtr(statsAt(0, 10, 0, 0)),
	})
	seedTopic(t, conn, statsTopicSpec{ // Agent：首次公开时间缺失（估算来源）
		id: 101, userId: 10, categories: []uint64{categoryMain},
		createdAt: statsAt(0, 11, 0, 0),
	})

	human, err := UserRanking(context.Background(), RankingQuery{StartDate: statsDate(0), EndDate: statsDate(0)})
	if err != nil {
		t.Fatalf("human ranking: %v", err)
	}
	if human.Coverage.Estimated {
		t.Fatalf("human ranking coverage.estimated = true, want false when only agent rows are backfilled")
	}
	agent, err := UserRanking(context.Background(), RankingQuery{
		StartDate: statsDate(0), EndDate: statsDate(0), ActorType: int(users.ActorTypeBot),
	})
	if err != nil {
		t.Fatalf("agent ranking: %v", err)
	}
	if !agent.Coverage.Estimated {
		t.Fatalf("agent ranking coverage.estimated = false, want true for a backfilled agent row")
	}

	detail, err := UserDetail(context.Background(), DetailQuery{StartDate: statsDate(0), EndDate: statsDate(0), UserId: 1})
	if err != nil {
		t.Fatalf("human detail: %v", err)
	}
	if detail.Coverage.Estimated {
		t.Fatalf("human detail coverage.estimated = true, want false for exact first_public_at")
	}
}
