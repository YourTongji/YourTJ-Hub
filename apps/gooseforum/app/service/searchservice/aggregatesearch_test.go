package searchservice

import (
	"encoding/json"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/course"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/meilisearch/meilisearch-go"
)

func TestNormalizeScope(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"", "all"},
		{"all", "all"},
		{"topics", "topics"},
		{"users", "users"},
		{"categories", "categories"},
		{"courses", "courses"},
		{"invalid", "all"},
		{"Topics", "all"},
	}
	for _, tc := range cases {
		if got := NormalizeScope(tc.input); got != tc.want {
			t.Fatalf("NormalizeScope(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestScopeQueries(t *testing.T) {
	queries := scopeQueries(AggregateSearchRequest{Query: "hello", Scope: ScopeAll, Limit: 10})
	if len(queries) != 4 {
		t.Fatalf("scope all should produce 4 queries, got %d", len(queries))
	}
	if queries[0].IndexUID != TopicIndex || queries[0].Limit != 10 || queries[0].Offset != 0 {
		t.Fatalf("topics query wrong: %+v", queries[0])
	}
	if queries[1].IndexUID != UserIndex {
		t.Fatalf("users query wrong: %+v", queries[1])
	}
	if queries[2].IndexUID != CategoryIndex {
		t.Fatalf("categories query wrong: %+v", queries[2])
	}
	if queries[3].IndexUID != CourseIndex {
		t.Fatalf("courses query wrong: %+v", queries[3])
	}

	usersOnly := scopeQueries(AggregateSearchRequest{Query: "hello", Scope: ScopeUsers, Limit: 0})
	if len(usersOnly) != 1 || usersOnly[0].IndexUID != UserIndex {
		t.Fatalf("users scope should produce only users query: %+v", usersOnly)
	}
	if usersOnly[0].Limit != MaxAggregateLimit {
		t.Fatalf("limit should be capped to MaxAggregateLimit, got %d", usersOnly[0].Limit)
	}

	coursesOnly := scopeQueries(AggregateSearchRequest{Query: "hello", Scope: ScopeCourses, Limit: 0})
	if len(coursesOnly) != 1 || coursesOnly[0].IndexUID != CourseIndex {
		t.Fatalf("courses scope should produce only courses query: %+v", coursesOnly)
	}
}

// TestScopeQueriesTopicTypeFilter review B1：topics 域 filter 必须是 Meilisearch
// filter 表达式（字符串或字符串数组），不能是 {"filter": [...]} 包装对象
// （后者会被 SDK 直接序列化导致 400 失败）。
func TestScopeQueriesTopicTypeFilter(t *testing.T) {
	topicType := topics.TopicTypeForum
	queries := scopeQueries(AggregateSearchRequest{Query: "hello", Scope: ScopeTopics, Limit: 10, TopicType: &topicType})
	if len(queries) != 1 {
		t.Fatalf("topics scope should produce 1 query, got %d", len(queries))
	}
	filter := queries[0].Filter
	if filter == nil {
		t.Fatalf("topic type filter missing: %+v", queries[0])
	}
	// 断言 shape：[]string 表达式（review B1 修复后）。
	exprs, ok := filter.([]string)
	if !ok {
		t.Fatalf("filter shape = %T (%#v), want []string", filter, filter)
	}
	if len(exprs) != 1 || exprs[0] != "topicType = 0" {
		t.Fatalf("filter exprs = %#v, want [topicType = 0]", exprs)
	}
	// nil TopicType 不产生 filter。
	noFilter := scopeQueries(AggregateSearchRequest{Query: "hello", Scope: ScopeTopics, Limit: 10})
	if noFilter[0].Filter != nil {
		t.Fatalf("filter should be nil when TopicType is nil, got %#v", noFilter[0].Filter)
	}
}

func TestAggregateSearchEmptyQuery(t *testing.T) {
	resp, err := AggregateSearch(AggregateSearchRequest{Query: "   ", Scope: ScopeAll})
	if err != nil {
		t.Fatalf("empty query should not error, got %v", err)
	}
	if resp == nil || resp.Topics == nil || resp.Users == nil || resp.Categories == nil || resp.Courses == nil {
		t.Fatalf("empty query should return empty slices, got %+v", resp)
	}
}

// seedSearchableTopics 建表并插入指定可见性的话题，返回清理函数。
func seedSearchableTopics(t *testing.T, list []topics.Entity) {
	t.Helper()
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(&topics.Entity{}); err != nil {
		t.Fatalf("migrate topics: %v", err)
	}
	conn.Unscoped().Where("1 = 1").Delete(&topics.Entity{})
	for i := range list {
		if err := conn.Create(&list[i]).Error; err != nil {
			t.Fatalf("create topic %d: %v", list[i].Id, err)
		}
	}
	t.Cleanup(func() { conn.Unscoped().Where("1 = 1").Delete(&topics.Entity{}) })
}

func TestCollectScopeResultsTopics(t *testing.T) {
	seedSearchableTopics(t, []topics.Entity{
		{Id: 1, Title: "hello", Status: 1, ProcessStatus: topics.ProcessStatusNormal, VisibilityStatus: topics.VisibilityActive},
		{Id: 2, Title: "world", Status: 1, ProcessStatus: topics.ProcessStatusNormal, VisibilityStatus: topics.VisibilityActive},
	})
	resp := &AggregateSearchResponse{Topics: []SearchResult{}, Users: []UserSearchResult{}, Categories: []CategorySearchResult{}}
	searchResp := &meilisearch.SearchResponse{
		Hits: []meilisearch.Hit{
			{"id": json.RawMessage(`1`), "title": json.RawMessage(`"hello"`)},
			{"id": json.RawMessage(`2`), "title": json.RawMessage(`"world"`)},
		},
		EstimatedTotalHits: 2,
	}
	collectScopeResults(resp, TopicIndex, searchResp)
	if len(resp.Topics) != 2 {
		t.Fatalf("topics results = %d, want 2", len(resp.Topics))
	}
	if resp.Topics[0].ID != 1 || resp.Topics[1].ID != 2 {
		t.Fatalf("topics ids wrong: %+v", resp.Topics)
	}
	if resp.Total != 2 {
		t.Fatalf("total = %d, want 2", resp.Total)
	}
}

func TestCollectScopeResultsSkipsInvalidIDs(t *testing.T) {
	seedSearchableTopics(t, []topics.Entity{
		{Id: 3, Title: "good", Status: 1, ProcessStatus: topics.ProcessStatusNormal, VisibilityStatus: topics.VisibilityActive},
	})
	resp := &AggregateSearchResponse{Topics: []SearchResult{}, Users: []UserSearchResult{}, Categories: []CategorySearchResult{}}
	searchResp := &meilisearch.SearchResponse{
		Hits: []meilisearch.Hit{
			{"id": json.RawMessage(`0`), "title": json.RawMessage(`"bad"`)},
			{"id": json.RawMessage(`3`), "title": json.RawMessage(`"good"`)},
		},
		EstimatedTotalHits: 1,
	}
	collectScopeResults(resp, TopicIndex, searchResp)
	if len(resp.Topics) != 1 || resp.Topics[0].ID != 3 {
		t.Fatalf("invalid id should be skipped: %+v", resp.Topics)
	}
}

// TestCollectScopeResultsFiltersNonPublicTopics 验证聚合搜索防御层：
// 即使 Meili 索引中残留非公开话题文档（索引事件未落地窗口期），
// 结果也按 DB 当前状态过滤，绝不返回已下架/待审/封禁/删除的话题（issue #132）。
func TestCollectScopeResultsFiltersNonPublicTopics(t *testing.T) {
	seedSearchableTopics(t, []topics.Entity{
		{Id: 1, Title: "公开", Status: 1, ProcessStatus: topics.ProcessStatusNormal, VisibilityStatus: topics.VisibilityActive},
		{Id: 2, Title: "已下架", Status: 0, ProcessStatus: topics.ProcessStatusNormal, VisibilityStatus: topics.VisibilityActive},
		{Id: 3, Title: "待审", Status: 1, ProcessStatus: topics.ProcessStatusPending, VisibilityStatus: topics.VisibilityActive},
		{Id: 4, Title: "已封禁", Status: 1, ProcessStatus: topics.ProcessStatusBlocked, VisibilityStatus: topics.VisibilityActive},
		{Id: 5, Title: "用户删除", Status: 1, ProcessStatus: topics.ProcessStatusNormal, VisibilityStatus: topics.VisibilityUserDeleted},
		{Id: 6, Title: "管理删除", Status: 1, ProcessStatus: topics.ProcessStatusNormal, VisibilityStatus: topics.VisibilityModeratorRemoved},
	})
	resp := &AggregateSearchResponse{Topics: []SearchResult{}, Users: []UserSearchResult{}, Categories: []CategorySearchResult{}}
	searchResp := &meilisearch.SearchResponse{
		Hits: []meilisearch.Hit{
			{"id": json.RawMessage(`1`), "title": json.RawMessage(`"公开"`)},
			{"id": json.RawMessage(`2`), "title": json.RawMessage(`"已下架"`)},
			{"id": json.RawMessage(`3`), "title": json.RawMessage(`"待审"`)},
			{"id": json.RawMessage(`4`), "title": json.RawMessage(`"已封禁"`)},
			{"id": json.RawMessage(`5`), "title": json.RawMessage(`"用户删除"`)},
			{"id": json.RawMessage(`6`), "title": json.RawMessage(`"管理删除"`)},
			// ID 7 在 Meili 索引中但 DB 中不存在（索引幽灵）：同样应被过滤
			{"id": json.RawMessage(`7`), "title": json.RawMessage(`"索引幽灵"`)},
		},
		EstimatedTotalHits: 7,
	}
	collectScopeResults(resp, TopicIndex, searchResp)
	if len(resp.Topics) != 1 || resp.Topics[0].ID != 1 {
		t.Fatalf("only public topic should survive DB filter, got %+v", resp.Topics)
	}
}

// TestCollectScopeResultsCoursesFillsStats 验证课程搜索结果的评分聚合填充
// （spec O2）：有 stats 时填充 ratingAvg/reviewCount；无 stats 时省略。
func TestCollectScopeResultsCoursesFillsStats(t *testing.T) {
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(&course.Entity{}, &course.CourseStatsEntity{}); err != nil {
		t.Fatalf("migrate course tables: %v", err)
	}
	conn.Unscoped().Where("1 = 1").Delete(&course.CourseStatsEntity{})
	conn.Unscoped().Where("1 = 1").Delete(&course.Entity{})
	t.Cleanup(func() {
		conn.Unscoped().Where("1 = 1").Delete(&course.CourseStatsEntity{})
		conn.Unscoped().Where("1 = 1").Delete(&course.Entity{})
	})
	if err := conn.Create(&course.Entity{Id: 10, PrimaryCode: "100010", Name: "高数", Department: "数学", CreditX10: 50, Status: course.StatusVisible}).Error; err != nil {
		t.Fatalf("create course 10: %v", err)
	}
	if err := conn.Create(&course.Entity{Id: 11, PrimaryCode: "100011", Name: "线代", Department: "数学", CreditX10: 30, Status: course.StatusVisible}).Error; err != nil {
		t.Fatalf("create course 11: %v", err)
	}
	// course 10 有 stats：2 条评分 sum=9 → avg=4.5
	if err := conn.Create(&course.CourseStatsEntity{CourseId: 10, RatingCount: 2, RatingSum: 9, ReviewCount: 3}).Error; err != nil {
		t.Fatalf("create course stats: %v", err)
	}

	resp := &AggregateSearchResponse{Courses: []CourseSearchResult{}}
	searchResp := &meilisearch.SearchResponse{
		Hits: []meilisearch.Hit{
			{"id": json.RawMessage(`10`), "primaryCode": json.RawMessage(`"100010"`), "name": json.RawMessage(`"高数"`)},
			{"id": json.RawMessage(`11`), "primaryCode": json.RawMessage(`"100011"`), "name": json.RawMessage(`"线代"`)},
		},
		EstimatedTotalHits: 2,
	}
	collectScopeResults(resp, CourseIndex, searchResp)
	if len(resp.Courses) != 2 {
		t.Fatalf("courses results = %d, want 2", len(resp.Courses))
	}
	if resp.Courses[0].ID != 10 || resp.Courses[0].RatingAvg == nil || *resp.Courses[0].RatingAvg != 4.5 {
		t.Fatalf("course 10 ratingAvg = %#v, want 4.5", resp.Courses[0].RatingAvg)
	}
	if resp.Courses[0].ReviewCount != 3 {
		t.Fatalf("course 10 reviewCount = %d, want 3", resp.Courses[0].ReviewCount)
	}
	if resp.Courses[1].ID != 11 || resp.Courses[1].RatingAvg != nil || resp.Courses[1].ReviewCount != 0 {
		t.Fatalf("course 11 should omit stats, got ratingAvg=%#v reviewCount=%d", resp.Courses[1].RatingAvg, resp.Courses[1].ReviewCount)
	}
}
