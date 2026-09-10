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
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/securestore"
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

// TestAiSummaryInsufficientData 无有效评价（少于阈值 1 条）→ insufficient_data，不调 LLM，
// 且落库 insufficient 标记（下次请求直接命中，不重复评估、不消耗限流）。
func TestAiSummaryInsufficientData(t *testing.T) {
	courseId := setupAiSummaryTest(t)
	// 0 条有效评价：少于阈值 1 → insufficient_data。
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
	// 落库 insufficient 标记（无 summary_json）。
	cached := course.GetCourseAiSummary(courseId)
	if cached.CourseId == 0 {
		t.Fatal("insufficient_data must persist a row (status=insufficient)")
	}
	if cached.Status != course.AiSummaryRowStatusInsufficient {
		t.Fatalf("row status = %q, want %q", cached.Status, course.AiSummaryRowStatusInsufficient)
	}
	if cached.SummaryJson != "" {
		t.Fatal("insufficient row must not carry summary_json")
	}

	// 再次请求（无 refresh）：直接命中 insufficient 标记，不重复评估、不调 LLM。
	second, err := GetAiSummary(courseId, false)
	if err != nil {
		t.Fatalf("second get: %v", err)
	}
	if second.Status != AiSummaryStatusInsufficientData {
		t.Fatalf("second status = %q, want insufficient_data", second.Status)
	}
	if *calls != 0 {
		t.Fatalf("llm calls after insufficient short-circuit = %d, want 0", *calls)
	}
}

// TestAiSummaryInsufficientDoesNotConsumeQuota 数据不足不消耗单课/全局限流名额：
// insufficient 后立即删除标记并补足评价再生成，不应被 429 拦截。
func TestAiSummaryInsufficientDoesNotConsumeQuota(t *testing.T) {
	courseId := setupAiSummaryTest(t)
	// 0 条有效评价 → 落 insufficient 标记。
	enableAiSummaryProvider(t, func() (string, error) { return fakeSummaryJSON, nil })

	if _, err := GetAiSummary(courseId, false); err != nil {
		t.Fatalf("first insufficient: %v", err)
	}
	// 补足评价到 1 条（作者用另一偏移避免撞唯一键）。不删除 insufficient 标记——
	// 自愈语义：GetAiSummary 以当前可见有正文评价数为准重新判定，足够则忽略
	// 过期标记继续生成（生成成功时 Upsert 覆盖为 generated）。
	conn := dbconnect.Connect()
	offering := course.OfferingEntity{}
	if err := conn.Where("course_id = ?", courseId).First(&offering).Error; err != nil {
		t.Fatalf("find offering: %v", err)
	}
	rating := 4
	author := uint64(20000)
	if err := conn.Create(&course.ReviewEntity{
		OfferingId:   offering.Id,
		AuthorUserId: &author,
		Rating:       &rating,
		Content:      "补充评价，内容充实有细节。",
		Status:       course.ReviewStatusVisible,
	}).Error; err != nil {
		t.Fatalf("create extra review: %v", err)
	}
	// 立即生成：若 insufficient 消耗过单课名额会被 429 拦截。
	result, err := GetAiSummary(courseId, false)
	if err != nil {
		t.Fatalf("generate after insufficient must not be rate limited: %v", err)
	}
	if result.Status != AiSummaryStatusGenerated {
		t.Fatalf("status = %q, want generated", result.Status)
	}
}

// TestAiSummaryInsufficientSelfHeals insufficient 标记自愈：标记落库后评价增加
// （写路径漏失效/阈值下调），GetAiSummary 以当前可见有正文评价数为准重新判定，
// 足够则忽略标记继续生成，不再永久返回 insufficient_data。
func TestAiSummaryInsufficientSelfHeals(t *testing.T) {
	courseId := setupAiSummaryTest(t)
	calls := enableAiSummaryProvider(t, func() (string, error) { return fakeSummaryJSON, nil })

	// 0 条评价 → 落 insufficient 标记。
	if _, err := GetAiSummary(courseId, false); err != nil {
		t.Fatalf("first insufficient: %v", err)
	}
	if cached := course.GetCourseAiSummary(courseId); cached.Status != course.AiSummaryRowStatusInsufficient {
		t.Fatalf("row status = %q, want insufficient", cached.Status)
	}
	// 评价补充到 1 条（足够），不删除标记（模拟写路径漏失效/阈值下调）。
	seedAiSummaryReviews(t, courseId, 1)
	// 不 refresh：自愈判定后继续生成。
	result, err := GetAiSummary(courseId, false)
	if err != nil {
		t.Fatalf("get after reviews added: %v", err)
	}
	if result.Status != AiSummaryStatusGenerated {
		t.Fatalf("status = %q, want generated (self-healed)", result.Status)
	}
	if *calls != 1 {
		t.Fatalf("llm calls = %d, want 1", *calls)
	}
	if cached := course.GetCourseAiSummary(courseId); cached.Status != course.AiSummaryRowStatusGenerated {
		t.Fatalf("row status after self-heal = %q, want generated", cached.Status)
	}
}

// TestCheckAiSummaryInsufficientSelfHeals check 预检自愈：insufficient 标记过期
// （评价已足够）时返回 none（而非 insufficient_data），前端展开即可触发生成。
func TestCheckAiSummaryInsufficientSelfHeals(t *testing.T) {
	courseId := setupAiSummaryTest(t)
	enableAiSummaryProvider(t, func() (string, error) { return fakeSummaryJSON, nil })

	if _, err := GetAiSummary(courseId, false); err != nil {
		t.Fatalf("first insufficient: %v", err)
	}
	if got, err := CheckAiSummary(courseId); err != nil || got.Status != AiSummaryStatusInsufficientData {
		t.Fatalf("check before reviews = %q err=%v, want insufficient_data", got.Status, err)
	}
	seedAiSummaryReviews(t, courseId, 1)
	got, err := CheckAiSummary(courseId)
	if err != nil {
		t.Fatalf("check after reviews: %v", err)
	}
	if got.Status != AiSummaryStatusNone {
		t.Fatalf("check after reviews = %q, want none (self-healed)", got.Status)
	}
}

// TestAiSummaryRefreshBypassesInsufficient refresh=true 跳过 insufficient 标记
// 强制重新评估（前端手动刷新语义）。
func TestAiSummaryRefreshBypassesInsufficient(t *testing.T) {
	courseId := setupAiSummaryTest(t)
	seedAiSummaryReviews(t, courseId, 12)
	calls := enableAiSummaryProvider(t, func() (string, error) { return fakeSummaryJSON, nil })

	// 先落 insufficient 标记（模拟历史上评估不足）。
	if err := upsertAiSummaryInsufficient(courseId); err != nil {
		t.Fatalf("upsert insufficient: %v", err)
	}
	result, err := GetAiSummary(courseId, true)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if result.Status != AiSummaryStatusGenerated {
		t.Fatalf("refresh status = %q, want generated", result.Status)
	}
	if *calls != 1 {
		t.Fatalf("llm calls = %d, want 1 (refresh re-evaluates)", *calls)
	}
	if cached := course.GetCourseAiSummary(courseId); cached.Status != course.AiSummaryRowStatusGenerated {
		t.Fatalf("row status after refresh = %q, want generated", cached.Status)
	}
}

// TestAiSummaryCheckMode check 预检只读 DB：cached/insufficient/none/disabled，
// 不生成、不消耗限流名额（多次 check 后首次生成仍成功）。
func TestAiSummaryCheckMode(t *testing.T) {
	courseId := setupAiSummaryTest(t)
	seedAiSummaryReviews(t, courseId, 12)
	enableAiSummaryProvider(t, func() (string, error) { return fakeSummaryJSON, nil })

	// 无缓存行 → none；多次 check 均不消耗名额。
	for i := 0; i < 3; i++ {
		got, err := CheckAiSummary(courseId)
		if err != nil {
			t.Fatalf("check none #%d: %v", i, err)
		}
		if got.Status != AiSummaryStatusNone {
			t.Fatalf("check #%d status = %q, want none", i, got.Status)
		}
	}

	// 首次生成成功 → check 返回 cached（携带 payload）。
	if _, err := GetAiSummary(courseId, false); err != nil {
		t.Fatalf("generate: %v", err)
	}
	got, err := CheckAiSummary(courseId)
	if err != nil {
		t.Fatalf("check cached: %v", err)
	}
	if got.Status != AiSummaryStatusCached || got.Summary == nil {
		t.Fatalf("check status = %q summary=%v, want cached with payload", got.Status, got.Summary)
	}

	// 评价不足场景 check → insufficient_data。
	other := setupAiSummaryTest(t) // 重新初始化（清空表与限流）
	// 0 条有效评价 → insufficient_data。
	if _, err := GetAiSummary(other, false); err != nil {
		t.Fatalf("insufficient generate: %v", err)
	}
	got, err = CheckAiSummary(other)
	if err != nil {
		t.Fatalf("check insufficient: %v", err)
	}
	if got.Status != AiSummaryStatusInsufficientData {
		t.Fatalf("check status = %q, want insufficient_data", got.Status)
	}

	// 功能关闭 check → disabled。
	upsertAiSummaryConfig(t, `{"enabled":false,"globalPerMinute":5}`)
	got, err = CheckAiSummary(other)
	if err != nil {
		t.Fatalf("check disabled: %v", err)
	}
	if got.Status != AiSummaryStatusDisabled {
		t.Fatalf("check status = %q, want disabled", got.Status)
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

// TestAiSummaryLLMFailure LLM 失败 → ErrAiSummaryGenerationFailed，不落库（验收 4），
// 且返还单课名额：失败后立即重试可再次触发生成（不被 10 分钟窗口卡死）。
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

// TestAiSummaryLLMFailureRefundQuota LLM 失败返还单课名额：第二次尝试（无缓存）
// 不被 429 拦截，仍能走到 provider（调用计数=2）。
func TestAiSummaryLLMFailureRefundQuota(t *testing.T) {
	courseId := setupAiSummaryTest(t)
	seedAiSummaryReviews(t, courseId, 10)
	calls := enableAiSummaryProvider(t, func() (string, error) { return "", errors.New("upstream timeout") })

	if _, err := GetAiSummary(courseId, false); err == nil {
		t.Fatal("first call must fail")
	}
	// 名额已返还：第二次调用应再次尝试 LLM（失败），而非 429。
	_, err := GetAiSummary(courseId, false)
	var rateErr *AiSummaryRateLimitError
	if errors.As(err, &rateErr) {
		t.Fatalf("second call after LLM failure must not be rate limited: %v", err)
	}
	if *calls != 2 {
		t.Fatalf("llm calls = %d, want 2 (quota refunded after first failure)", *calls)
	}
}

// TestAiSummaryProviderNotConfiguredRefundQuota provider 未配置（config.toml 与
// pageConfig 均为空）→ 生成失败且返还单课名额；随后配置好 provider 可立即生成。
func TestAiSummaryProviderNotConfiguredRefundQuota(t *testing.T) {
	courseId := setupAiSummaryTest(t)
	seedAiSummaryReviews(t, courseId, 12)
	// 不调用 enableAiSummaryProvider：provider 未配置。
	if _, err := GetAiSummary(courseId, false); !errors.Is(err, ErrAiSummaryGenerationFailed) {
		t.Fatalf("err = %v, want ErrAiSummaryGenerationFailed (provider not configured)", err)
	}
	// 配置 provider 后立即生成：名额已返还，不应 429。
	enableAiSummaryProvider(t, func() (string, error) { return fakeSummaryJSON, nil })
	result, err := GetAiSummary(courseId, false)
	if err != nil {
		t.Fatalf("generate after configuring provider must not be rate limited: %v", err)
	}
	if result.Status != AiSummaryStatusGenerated {
		t.Fatalf("status = %q, want generated", result.Status)
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

// TestResolveAiSummaryConfigAdminWins 管理后台 pageConfig 配置优先于 config.toml。
func TestResolveAiSummaryConfigAdminWins(t *testing.T) {
	// setupAiSummaryTest 负责 AutoMigrate page_config 表（upsert 依赖表存在）。
	setupAiSummaryTest(t)
	conn := dbconnect.Connect()
	// 先写 config.toml 侧的值（兜底），再写 pageConfig 侧的值（优先）。
	preferences.Set("ai_summary.base_url", "https://fallback.example.test/v1")
	preferences.Set("ai_summary.model", "fallback-model")
	preferences.Set("ai_summary.api_key", "fallback-key")
	t.Cleanup(func() {
		preferences.Set("ai_summary.base_url", "")
		preferences.Set("ai_summary.model", "")
		preferences.Set("ai_summary.api_key", "")
		conn.Where("page_type = ?", pageConfig.AiSummarySettings).Delete(&pageConfig.Entity{})
		hotdataserve.ClearAiSummarySettingsConfigCache()
	})
	sealed, err := securestore.EncryptPurpose("admin-key", securestore.AiSummaryAPIKeyPurpose)
	if err != nil {
		t.Fatalf("encrypt admin key: %v", err)
	}
	upsertAiSummaryConfig(t, `{"enabled":true,"globalPerMinute":5,"baseUrl":"https://admin.example.test/v1","model":"admin-model","apiKeyEncrypted":"`+sealed+`"}`)

	cfg := resolveAiSummaryConfig()
	if cfg.BaseURL != "https://admin.example.test/v1" || cfg.Model != "admin-model" {
		t.Fatalf("cfg = %#v, want admin baseURL/model", cfg)
	}
	if cfg.APIKey != "admin-key" {
		t.Fatalf("cfg.APIKey = %q, want decrypted admin-key", cfg.APIKey)
	}
	if cfg.Temperature != 0.3 || cfg.MaxTokens != 1024 {
		t.Fatalf("cfg defaults = %#v, want 0.3/1024", cfg)
	}
}

// TestResolveAiSummaryConfigFallsBackToToml 管理后台未配置时回退 config.toml。
func TestResolveAiSummaryConfigFallsBackToToml(t *testing.T) {
	// setupAiSummaryTest 负责 AutoMigrate page_config 表。
	setupAiSummaryTest(t)
	conn := dbconnect.Connect()
	t.Cleanup(func() {
		preferences.Set("ai_summary.base_url", "")
		preferences.Set("ai_summary.model", "")
		conn.Where("page_type = ?", pageConfig.AiSummarySettings).Delete(&pageConfig.Entity{})
		hotdataserve.ClearAiSummarySettingsConfigCache()
	})
	// 清掉 pageConfig 行，保证走 config.toml 兜底。
	conn.Where("page_type = ?", pageConfig.AiSummarySettings).Delete(&pageConfig.Entity{})
	hotdataserve.ClearAiSummarySettingsConfigCache()
	preferences.Set("ai_summary.base_url", "https://toml.example.test/v1/")
	preferences.Set("ai_summary.model", "toml-model")

	cfg := resolveAiSummaryConfig()
	if cfg.BaseURL != "https://toml.example.test/v1" || cfg.Model != "toml-model" {
		t.Fatalf("cfg = %#v, want toml baseURL/model with trailing slash trimmed", cfg)
	}
}

func TestGetAiSummaryContextCancelsLLM(t *testing.T) {
	courseID := setupAiSummaryTest(t)
	seedAiSummaryReviews(t, courseID, 10)
	preferences.Set("ai_summary.base_url", "http://fake.local/v1")
	preferences.Set("ai_summary.model", "fake-model")
	started := make(chan struct{})
	original := llmChat
	llmChat = func(ctx context.Context, _ llmprovider.Config, _ string) (string, error) {
		close(started)
		<-ctx.Done()
		return "", ctx.Err()
	}
	t.Cleanup(func() { llmChat = original })

	timeoutErr := errors.New("test llm timeout")
	ctx, cancel := context.WithTimeoutCause(context.Background(), 20*time.Millisecond, timeoutErr)
	defer cancel()
	_, err := GetAiSummaryContext(ctx, courseID, false)
	if !errors.Is(err, ErrAiSummaryGenerationFailed) {
		t.Fatalf("GetAiSummaryContext() error = %v, want generation failure", err)
	}
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("LLM was not called")
	}
	if !errors.Is(context.Cause(ctx), timeoutErr) {
		t.Fatalf("context cause = %v, want %v", context.Cause(ctx), timeoutErr)
	}
}
