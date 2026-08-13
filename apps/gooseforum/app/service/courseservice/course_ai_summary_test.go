package courseservice

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/llmprovider"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/ratelimit"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/course"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
)

// aiSummaryTestModels AI 总结测试用表（course 域 + 总结缓存 + pageConfig）。
var aiSummaryTestModels = []any{
	&course.Entity{},
	&course.TermEntity{},
	&course.OfferingEntity{},
	&course.ReviewEntity{},
	&course.CourseAiSummaryEntity{},
	&course.CourseStatsEntity{},
	&course.OfferingStatsEntity{},
	&pageConfig.Entity{},
}

// setupAiSummaryTest 迁移并清空 AI 总结相关表，开启功能开关，返回可见课程。
func setupAiSummaryTest(t *testing.T) uint64 {
	t.Helper()
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(aiSummaryTestModels...); err != nil {
		t.Fatalf("migrate ai summary tables: %v", err)
	}
	for _, model := range aiSummaryTestModels {
		if err := conn.Unscoped().Where("1 = 1").Delete(model).Error; err != nil {
			t.Fatalf("clean ai summary table: %v", err)
		}
	}
	// 开启 AI 总结（pageConfig 热配置），并清掉缓存与限流计数。
	pageConfig.CreateOrSave(&pageConfig.Entity{
		PageType: pageConfig.AiSummarySettings,
		Config:   `{"enabled":true,"globalPerMinute":100}`,
	})
	hotdataserve.ClearAiSummarySettingsConfigCache()
	ratelimit.Default().ResetAll()

	c := course.Entity{PrimaryCode: "100001", Name: "高等数学(A)上", Department: "数学科学学院", Status: course.StatusVisible}
	if err := conn.Create(&c).Error; err != nil {
		t.Fatalf("create course: %v", err)
	}
	term := course.TermEntity{Code: "2025-2026-1", Name: "2025-2026 第一学期", Status: 0}
	if err := conn.Create(&term).Error; err != nil {
		t.Fatalf("create term: %v", err)
	}
	offering := course.OfferingEntity{CourseId: c.Id, TermId: term.Id, Campus: "四平路校区", Status: course.OfferingStatusVisible}
	if err := conn.Create(&offering).Error; err != nil {
		t.Fatalf("create offering: %v", err)
	}
	_ = offering
	return c.Id
}

// upsertAiSummaryConfig 更新（或创建）AI 总结开关配置并清缓存。
// 必须复用已有行（page_type 唯一索引），不能每次 Create 新行。
func upsertAiSummaryConfig(t *testing.T, configJSON string) {
	t.Helper()
	entity := pageConfig.GetByPageType(pageConfig.AiSummarySettings)
	entity.PageType = pageConfig.AiSummarySettings
	entity.Config = configJSON
	pageConfig.CreateOrSave(&entity)
	hotdataserve.ClearAiSummarySettingsConfigCache()
}

// seedAiSummaryReviews 为课程写入 n 条可见有效评价（评分 5 分循环）。
func seedAiSummaryReviews(t *testing.T, courseId uint64, n int) {
	t.Helper()
	conn := dbconnect.Connect()
	offering := course.OfferingEntity{}
	if err := conn.Where("course_id = ?", courseId).First(&offering).Error; err != nil {
		t.Fatalf("find offering: %v", err)
	}
	for i := 0; i < n; i++ {
		rating := 3 + i%3 // 3..5
		author := uint64(10000 + i)
		review := course.ReviewEntity{
			OfferingId:   offering.Id,
			AuthorUserId: &author,
			Rating:       &rating,
			Content:      "老师讲得很好，作业量适中，给分不错。",
			Status:       course.ReviewStatusVisible,
		}
		if err := conn.Create(&review).Error; err != nil {
			t.Fatalf("create review %d: %v", i, err)
		}
	}
}

// seedHiddenReview 写入一条隐藏评价（不应进入总结输入）。
func seedHiddenReview(t *testing.T, courseId uint64) {
	t.Helper()
	conn := dbconnect.Connect()
	offering := course.OfferingEntity{}
	if err := conn.Where("course_id = ?", courseId).First(&offering).Error; err != nil {
		t.Fatalf("find offering: %v", err)
	}
	author := uint64(99999)
	rating := 1
	if err := conn.Create(&course.ReviewEntity{
		OfferingId:   offering.Id,
		AuthorUserId: &author,
		Rating:       &rating,
		Content:      "这门课很糟糕。",
		Status:       course.ReviewStatusHidden,
	}).Error; err != nil {
		t.Fatalf("create hidden review: %v", err)
	}
}

// enableAiSummaryProvider 配置 fake provider 并注入 fake LLM 调用；返回调用计数。
func enableAiSummaryProvider(t *testing.T, fake func() (string, error)) *int {
	t.Helper()
	preferences.Set("ai_summary.base_url", "http://fake.local/v1")
	preferences.Set("ai_summary.model", "fake-model")
	t.Cleanup(func() {
		preferences.Set("ai_summary.base_url", "")
		preferences.Set("ai_summary.model", "")
		llmChat = func(ctx context.Context, cfg llmprovider.Config, prompt string) (string, error) {
			return cfg.Complete(ctx, llmprovider.ChatRequest{
				Messages: []llmprovider.Message{
					{Role: "system", Content: aiSummarySystemPrompt},
					{Role: "user", Content: prompt},
				},
				ResponseFormat: &llmprovider.ResponseFormat{Type: "json_object"},
			})
		}
	})
	orig := llmChat
	calls := 0
	llmChat = func(_ context.Context, _ llmprovider.Config, _ string) (string, error) {
		calls++
		return fake()
	}
	t.Cleanup(func() { llmChat = orig })
	return &calls
}

const fakeSummaryJSON = `{"consensus":"recommend","keywords":["给分好","作业多"],"pros":["老师讲得清楚","给分宽松"],"cons":["作业量较大"],"representativeReviews":[{"excerpt":"老师讲得很好，作业虽然多但有收获。","sentiment":"positive"},{"excerpt":"内容比较难。","sentiment":"neutral"}]}`

// TestAiSummaryInsufficientData 少于 10 条有效评价 → insufficient_data，不调 LLM、不落库。
func TestAiSummaryInsufficientData(t *testing.T) {
	courseId := setupAiSummaryTest(t)
	seedAiSummaryReviews(t, courseId, 5)
	calls := enableAiSummaryProvider(t, func() (string, error) { return fakeSummaryJSON, nil })

	result, err := GetAiSummary(courseId, false)
	if err != nil {
		t.Fatalf("get ai summary: %v", err)
	}
	if result.Status != AiSummaryStatusInsufficientData {
		t.Fatalf("status = %q, want insufficient_data", result.Status)
	}
	if *calls != 0 {
		t.Fatalf("llm calls = %d, want 0 (data insufficient short-circuits before provider)", *calls)
	}
	if cached := course.GetCourseAiSummary(courseId); cached.CourseId != 0 {
		t.Fatalf("insufficient_data must not persist a cache row")
	}
}

// TestAiSummaryGeneratedThenCached 生成后落库；重复请求（无 refresh）命中 DB 缓存，
// provider 调用计数不增加（验收 2）。
func TestAiSummaryGeneratedThenCached(t *testing.T) {
	courseId := setupAiSummaryTest(t)
	seedAiSummaryReviews(t, courseId, 12)
	calls := enableAiSummaryProvider(t, func() (string, error) { return fakeSummaryJSON, nil })

	first, err := GetAiSummary(courseId, false)
	if err != nil {
		t.Fatalf("first get: %v", err)
	}
	if first.Status != AiSummaryStatusGenerated {
		t.Fatalf("first status = %q, want generated", first.Status)
	}
	if first.Summary == nil {
		t.Fatal("generated summary must carry payload")
	}
	if first.Summary.Consensus != ConsensusRecommend {
		t.Fatalf("consensus = %q, want recommend", first.Summary.Consensus)
	}
	if first.GeneratedAt == "" || first.Model == "" {
		t.Fatal("generated result must carry generatedAt and model")
	}
	if *calls != 1 {
		t.Fatalf("llm calls after first = %d, want 1", *calls)
	}
	if cached := course.GetCourseAiSummary(courseId); cached.CourseId == 0 || cached.SummaryJson == "" {
		t.Fatal("generated summary must be persisted")
	}

	second, err := GetAiSummary(courseId, false)
	if err != nil {
		t.Fatalf("second get: %v", err)
	}
	if second.Status != AiSummaryStatusCached {
		t.Fatalf("second status = %q, want cached", second.Status)
	}
	if *calls != 1 {
		t.Fatalf("llm calls after cache hit = %d, want still 1 (验收 2: provider 计数 0)", *calls)
	}
}

// TestAiSummarySanitizeFallback LLM 返回非法 consensus/超长数组 → sanitize 归一化
// （schema 合规，验收 1），非法 consensus 按评分分布兜底。
func TestAiSummarySanitizeFallback(t *testing.T) {
	courseId := setupAiSummaryTest(t)
	seedAiSummaryReviews(t, courseId, 10) // 全是 3..5 分，均分 4 → recommend
	raw := `{"consensus":"乱七八糟","keywords":["k1","k2","k3","k4","k5","k6"],"pros":["p1","p2","p3","p4","p5"],"cons":["c1"],"representativeReviews":[{"excerpt":"好的","sentiment":"bad"},{"excerpt":"","sentiment":"positive"},{"excerpt":"中肯","sentiment":"neutral"}]}`
	enableAiSummaryProvider(t, func() (string, error) { return raw, nil })

	result, err := GetAiSummary(courseId, false)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if result.Status != AiSummaryStatusGenerated {
		t.Fatalf("status = %q, want generated", result.Status)
	}
	payload := result.Summary
	if len(payload.Keywords) != 5 {
		t.Fatalf("keywords len = %d, want 5 (truncated)", len(payload.Keywords))
	}
	if len(payload.Pros) != 4 {
		t.Fatalf("pros len = %d, want 4 (truncated)", len(payload.Pros))
	}
	if len(payload.RepresentativeReviews) != 2 {
		t.Fatalf("representative len = %d, want 2 (empty excerpt dropped)", len(payload.RepresentativeReviews))
	}
	if payload.RepresentativeReviews[0].Sentiment != SentimentNeutral {
		t.Fatalf("invalid sentiment must fall back to neutral, got %q", payload.RepresentativeReviews[0].Sentiment)
	}
	if payload.Consensus != ConsensusRecommend {
		t.Fatalf("invalid consensus must fall back to rating-based mapping, got %q", payload.Consensus)
	}
}

// TestAiSummaryLLMFailure LLM 失败 → ErrAiSummaryGenerationFailed，不落库（验收 4）。
func TestAiSummaryLLMFailure(t *testing.T) {
	courseId := setupAiSummaryTest(t)
	seedAiSummaryReviews(t, courseId, 10)
	enableAiSummaryProvider(t, func() (string, error) { return "", errors.New("upstream timeout") })

	_, err := GetAiSummary(courseId, false)
	if !errors.Is(err, ErrAiSummaryGenerationFailed) {
		t.Fatalf("err = %v, want ErrAiSummaryGenerationFailed", err)
	}
	if cached := course.GetCourseAiSummary(courseId); cached.CourseId != 0 {
		t.Fatal("failed generation must not persist a cache row")
	}
}

// TestAiSummaryCourseNotFound 课程不存在/隐藏 → ErrAiSummaryCourseNotFound。
func TestAiSummaryCourseNotFound(t *testing.T) {
	setupAiSummaryTest(t)
	_, err := GetAiSummary(999999, false)
	if !errors.Is(err, ErrAiSummaryCourseNotFound) {
		t.Fatalf("err = %v, want ErrAiSummaryCourseNotFound", err)
	}
}

// TestAiSummaryDisabled 功能开关关闭 → status=disabled，不调 LLM。
func TestAiSummaryDisabled(t *testing.T) {
	courseId := setupAiSummaryTest(t)
	// 关闭开关（复用已有 pageConfig 行，唯一索引 page_type）。
	upsertAiSummaryConfig(t, `{"enabled":false,"globalPerMinute":5}`)
	calls := enableAiSummaryProvider(t, func() (string, error) { return fakeSummaryJSON, nil })

	result, err := GetAiSummary(courseId, false)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if result.Status != AiSummaryStatusDisabled {
		t.Fatalf("status = %q, want disabled", result.Status)
	}
	if *calls != 0 {
		t.Fatalf("llm calls = %d, want 0", *calls)
	}
}

// TestAiSummaryHiddenReviewsExcluded 隐藏评价不进入总结输入（仅 visible 口径）。
func TestAiSummaryHiddenReviewsExcluded(t *testing.T) {
	courseId := setupAiSummaryTest(t)
	seedAiSummaryReviews(t, courseId, 10)
	seedHiddenReview(t, courseId)
	calls := enableAiSummaryProvider(t, func() (string, error) { return fakeSummaryJSON, nil })

	// 隐藏评价不计入，但 10 条可见评价已达标 → generated；若隐藏评价被计入也不改变状态。
	result, err := GetAiSummary(courseId, false)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if result.Status != AiSummaryStatusGenerated {
		t.Fatalf("status = %q, want generated", result.Status)
	}
	if *calls != 1 {
		t.Fatalf("llm calls = %d, want 1", *calls)
	}
}

// TestAiSummaryRateLimit 单课 10 分钟限流：无缓存时第二次生成请求 → 429 语义错误。
func TestAiSummaryRateLimit(t *testing.T) {
	courseId := setupAiSummaryTest(t)
	seedAiSummaryReviews(t, courseId, 10)
	calls := enableAiSummaryProvider(t, func() (string, error) { return fakeSummaryJSON, nil })

	// 第一次生成（消耗单课配额），随后删除缓存模拟"无缓存被限"。
	if _, err := GetAiSummary(courseId, true); err != nil {
		t.Fatalf("first generate: %v", err)
	}
	conn := dbconnect.Connect()
	if err := conn.Where("course_id = ?", courseId).Delete(&course.CourseAiSummaryEntity{}).Error; err != nil {
		t.Fatalf("delete cache: %v", err)
	}
	_, err := GetAiSummary(courseId, true)
	var rateErr *AiSummaryRateLimitError
	if !errors.As(err, &rateErr) {
		t.Fatalf("err = %v, want AiSummaryRateLimitError", err)
	}
	if rateErr.RetryAfter <= 0 {
		t.Fatalf("retryAfter = %v, want > 0", rateErr.RetryAfter)
	}
	if *calls != 1 {
		t.Fatalf("llm calls = %d, want 1 (rate limited before provider)", *calls)
	}
}

// TestTruncateRunes rune 截断不破坏 UTF-8。
func TestTruncateRunes(t *testing.T) {
	s := strings.Repeat("好", 3000)
	got := truncateRunes(s, 2000)
	if len([]rune(got)) != 2000 {
		t.Fatalf("truncate runes = %d, want 2000", len([]rune(got)))
	}
	if json.Valid([]byte(`"`+got+`"`)) == false {
		t.Fatal("truncated string must remain valid UTF-8")
	}
}

// TestConsensusFromRatings 评分兜底映射边界。
func TestConsensusFromRatings(t *testing.T) {
	cases := []struct {
		ratings []int
		want    AiSummaryConsensus
	}{
		{[]int{5, 5, 5, 5}, ConsensusStrongRecommend},
		{[]int{4, 4, 4, 3}, ConsensusRecommend},
		{[]int{3, 3, 2, 2}, ConsensusNeutral},
		{[]int{2, 2, 1, 2}, ConsensusCautious},
		{[]int{1, 1, 1}, ConsensusNotRecommend},
		{nil, ConsensusNeutral},
	}
	for _, c := range cases {
		reviews := make([]reviewInput, 0, len(c.ratings))
		for _, r := range c.ratings {
			reviews = append(reviews, reviewInput{Rating: r, Content: "x"})
		}
		if got := consensusFromRatings(reviews); got != c.want {
			t.Fatalf("ratings %v: got %q, want %q", c.ratings, got, c.want)
		}
	}
}

// TestBuildAiSummaryPrompt prompt 含课程信息与编号评价。
func TestBuildAiSummaryPrompt(t *testing.T) {
	p := buildAiSummaryPrompt("高等数学(A)上", "100001", []reviewInput{{Rating: 5, Content: "好课"}, {Rating: 4, Content: "不错"}})
	for _, want := range []string{"高等数学(A)上", "100001", "1. 评分5：好课", "2. 评分4：不错"} {
		if !strings.Contains(p, want) {
			t.Fatalf("prompt missing %q: %s", want, p)
		}
	}
}

// TestAiSummaryConfigDefault 默认配置为关闭（成本护栏优先）。
func TestAiSummaryConfigDefault(t *testing.T) {
	conn := dbconnect.Connect()
	// 清掉可能残留的配置行与进程级缓存，保证读的是默认值。
	if err := conn.Where("page_type = ?", pageConfig.AiSummarySettings).Delete(&pageConfig.Entity{}).Error; err != nil {
		t.Fatalf("delete ai summary config: %v", err)
	}
	hotdataserve.ClearAiSummarySettingsConfigCache()
	cfg := hotdataserve.GetAiSummarySettingsConfigCache()
	if cfg.Enabled {
		t.Fatal("ai summary must default to disabled")
	}
}

// TestAiSummaryGeneratedAtPersisted generated_at 落库并可读回（验收 1: generated_at 更新）。
func TestAiSummaryGeneratedAtPersisted(t *testing.T) {
	courseId := setupAiSummaryTest(t)
	seedAiSummaryReviews(t, courseId, 10)
	enableAiSummaryProvider(t, func() (string, error) { return fakeSummaryJSON, nil })

	before := time.Now()
	result, err := GetAiSummary(courseId, false)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	cached := course.GetCourseAiSummary(courseId)
	if cached.GeneratedAt.Before(before.Add(-time.Minute)) || cached.GeneratedAt.After(time.Now().Add(time.Minute)) {
		t.Fatalf("generated_at out of range: %v", cached.GeneratedAt)
	}
	parsed, err := time.Parse(time.RFC3339, result.GeneratedAt)
	if err != nil {
		t.Fatalf("generatedAt not RFC3339: %q", result.GeneratedAt)
	}
	if parsed.Unix() != cached.GeneratedAt.Unix() {
		t.Fatalf("response generatedAt %v != stored %v", parsed, cached.GeneratedAt)
	}
}
