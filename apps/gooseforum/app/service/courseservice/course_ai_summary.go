package courseservice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/llmprovider"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/ratelimit"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/course"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
)

// ---- B7: AI 课程总结（issue #181） ----

// AiSummaryStatus 总结响应状态（与前端 api.ts CourseSummaryStatus 对齐）。
type AiSummaryStatus string

const (
	AiSummaryStatusCached           AiSummaryStatus = "cached"            // DB 缓存命中
	AiSummaryStatusGenerated        AiSummaryStatus = "generated"         // 本次新生成
	AiSummaryStatusInsufficientData AiSummaryStatus = "insufficient_data" // 有效评价不足，未生成
	AiSummaryStatusDisabled         AiSummaryStatus = "disabled"          // 功能未启用
	AiSummaryStatusNone             AiSummaryStatus = "none"              // check 预检：无缓存行，未生成过
)

// AiSummaryConsensus 五档口碑共识（与前端 AISummaryCard 的 ConsensusLevel 对齐）。
type AiSummaryConsensus string

const (
	ConsensusStrongRecommend AiSummaryConsensus = "strong_recommend"
	ConsensusRecommend       AiSummaryConsensus = "recommend"
	ConsensusNeutral         AiSummaryConsensus = "neutral"
	ConsensusCautious        AiSummaryConsensus = "cautious"
	ConsensusNotRecommend    AiSummaryConsensus = "not_recommend"
)

// AiSummarySentiment 代表性评价情绪（与前端 CourseSummarySentiment 对齐）。
type AiSummarySentiment string

const (
	SentimentPositive AiSummarySentiment = "positive"
	SentimentNeutral  AiSummarySentiment = "neutral"
	SentimentNegative AiSummarySentiment = "negative"
)

// AiSummaryRepresentativeReview 代表性评价摘录。
type AiSummaryRepresentativeReview struct {
	Excerpt   string             `json:"excerpt"`
	Sentiment AiSummarySentiment `json:"sentiment"`
}

// AiSummaryPayload 结构化总结（schema 与前端 CourseSummaryPayload 对齐）。
type AiSummaryPayload struct {
	Consensus             AiSummaryConsensus              `json:"consensus"`
	Keywords              []string                        `json:"keywords"`
	Pros                  []string                        `json:"pros"`
	Cons                  []string                        `json:"cons"`
	RepresentativeReviews []AiSummaryRepresentativeReview `json:"representativeReviews"`
}

// AiSummaryResult 总结端点响应（schema 与前端 CourseSummaryResult 对齐）。
type AiSummaryResult struct {
	Status      AiSummaryStatus   `json:"status"`
	Summary     *AiSummaryPayload `json:"summary,omitempty"`
	GeneratedAt string            `json:"generatedAt,omitempty"`
	Model       string            `json:"model,omitempty"`
}

// AI 总结输入裁剪参数（与上游 YourTJCourse-Serverless 对齐）。
const (
	AiSummaryMaxReviews    = 30   // 取最新 N 条可见有效评价
	AiSummaryReviewMaxRune = 2000 // 单条正文截断（rune）
	AiSummaryMinReviews    = 1    // 少于 N 条可见有正文评价视为数据不足（≥1 即可生成，issue #181 需求）
)

// 限流常量（service 内第二/三层限流，先于 LLM 调用消耗）。
const (
	aiSummaryGlobalWindow    = time.Minute
	aiSummaryCourseLimit     = 1 // 单课 10 分钟 1 次生成
	aiSummaryCourseWindow    = 10 * time.Minute
	aiSummaryGlobalKey       = "ai.summary.global"
	aiSummaryCourseKeyPrefix = "ai.summary.course:"
	aiSummaryTimeout         = 30 * time.Second
)

// aiSummaryGlobalLimit 全局每分钟生成上限；0 表示用默认值（成本护栏，
// 实际值可经 pageConfig aiSummarySettings.globalPerMinute 热调整）。
const aiSummaryGlobalLimit = 5

// promptVersion 当前 prompt 版本；变更需同步升级（存库用于追溯生成时使用的版本）。
const promptVersion = "v1"

// llmChatFunc 便于测试注入的 LLM 调用函数（生产实现为 llmprovider.Config.Complete）。
type llmChatFunc func(ctx context.Context, cfg llmprovider.Config, prompt string) (string, error)

var llmChat llmChatFunc = func(ctx context.Context, cfg llmprovider.Config, prompt string) (string, error) {
	return cfg.Complete(ctx, llmprovider.ChatRequest{
		Messages: []llmprovider.Message{
			{Role: "system", Content: aiSummarySystemPrompt},
			{Role: "user", Content: prompt},
		},
		ResponseFormat: &llmprovider.ResponseFormat{Type: "json_object"},
	})
}

// resolveAiSummaryConfig 组装 LLM provider 配置：管理后台 pageConfig 配置优先
// （BaseURL/Model 任一已配置即整体优先，APIKey 已由缓存解密为运行时明文），
// 未配置时回退 config.toml [ai_summary]（向后兼容现有部署）。
func resolveAiSummaryConfig() llmprovider.Config {
	aiCfg := hotdataserve.GetAiSummarySettingsConfigCache()
	if strings.TrimSpace(aiCfg.BaseURL) != "" || strings.TrimSpace(aiCfg.Model) != "" {
		return llmprovider.Config{
			BaseURL:     strings.TrimRight(aiCfg.BaseURL, "/"),
			APIKey:      aiCfg.APIKey,
			Model:       aiCfg.Model,
			Temperature: float64Or(aiCfg.Temperature, 0.3),
			MaxTokens:   intOr(aiCfg.MaxTokens, 1024),
		}
	}
	return llmprovider.LoadConfig()
}

func float64Or(v *float64, def float64) float64 {
	if v == nil {
		return def
	}
	return *v
}

func intOr(v *int, def int) int {
	if v == nil {
		return def
	}
	return *v
}

// aiSummarySystemPrompt 约束 LLM 只输出 JSON 且字段与前端契约一致（英文枚举）。
const aiSummarySystemPrompt = `你是「选课评课 AI 助手」。你的任务是基于多条学生评价，生成一门课程的结构化总结。
只分析用户提供的评价文本，不添加外部事实。如果评价之间结论矛盾，要在总结中体现，不要抹平分歧。
使用简体中文（keywords/pros/cons/excerpt 用中文）。
只返回 JSON 对象，不要返回 Markdown、代码块、解释文字、思考过程或额外字段。

# 输出字段（必须严格符合）
- consensus: 评分共识，只能是 "strong_recommend" / "recommend" / "neutral" / "cautious" / "not_recommend" 之一
- keywords: 高频关键词数组，最多 5 个，每词不超过 6 个字，如 ["给分好","作业多"]
- pros: 学生普遍认可的优点数组，最多 4 条，每条一句话
- cons: 学生普遍反馈的缺点/槽点数组，最多 4 条，每条一句话
- representativeReviews: 代表性评价摘录数组，最多 3 条，每条为 {"excerpt": "原文摘录", "sentiment": "positive"|"neutral"|"negative"}

# 内部工作流（不要输出）
1. 统计评分分布。2. 提取高频主题。3. 分类正面/负面并按频次排序。
4. 选择代表性评价：优先有具体案例/细节的，不要只说"好/不好"。5. 按评分分布选择 consensus。
6. 只输出符合约束的 JSON。`

// buildAiSummaryPrompt 构造用户 prompt（编号评价列表 + 课程信息；无评分的 legacy 评价标注「无评分」）。
func buildAiSummaryPrompt(courseName, courseCode string, reviews []reviewInput) string {
	var b strings.Builder
	fmt.Fprintf(&b, "课程名称：%s\n课程代码：%s\n评价数量：%d 条\n\n各条评价（评分 1-5，无评分标注「无评分」）：\n", courseName, courseCode, len(reviews))
	for i, r := range reviews {
		if r.Rating > 0 {
			fmt.Fprintf(&b, "%d. 评分%d：%s\n", i+1, r.Rating, r.Content)
		} else {
			fmt.Fprintf(&b, "%d. （无评分）%s\n", i+1, r.Content)
		}
	}
	return b.String()
}

// reviewInput 喂给 LLM 的评价输入（仅评分与正文，不含任何作者信息）。
type reviewInput struct {
	Rating  int
	Content string
}

// 错误 sentinel（控制器映射 HTTP 状态）。
var (
	// ErrAiSummaryCourseNotFound 课程不存在或已隐藏。
	ErrAiSummaryCourseNotFound = errors.New("course not found for ai summary")
	// ErrAiSummaryGenerationFailed LLM 生成失败/超时/输出非法。
	ErrAiSummaryGenerationFailed = errors.New("ai summary generation failed")
)

// GetAiSummary 课程 AI 总结主流程（状态机见设计文档）：
// 课程可见 → 开关 → DB 缓存（!refresh）→ 取评价 → 数据不足落库 insufficient
// （不消耗限流）→ 限流 → LLM → sanitize → 落库 generated。
//
// 限流后置：只有真正要调 LLM 才消耗单课/全局名额；数据不足（insufficient）
// 不消耗任何名额；LLM 失败/未配置/落库失败返还单课名额（全局名额保留——
// 确实发生了调用，防刷语义不变）。
func GetAiSummary(courseId uint64, refresh bool) (AiSummaryResult, error) {
	return GetAiSummaryContext(context.Background(), courseId, refresh)
}

// GetAiSummaryContext keeps request cancellation attached to the provider
// call and applies the service-level upper bound for every caller. Background
// jobs should still pass their own worker context so shutdown can cancel it.
func GetAiSummaryContext(ctx context.Context, courseId uint64, refresh bool) (AiSummaryResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeoutCause(ctx, aiSummaryTimeout, context.DeadlineExceeded)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return AiSummaryResult{}, ErrAiSummaryGenerationFailed
	}
	// 1. 课程存在且可见。
	entity := course.GetCourse(courseId)
	if entity.Id == 0 || entity.Status != course.StatusVisible {
		return AiSummaryResult{}, ErrAiSummaryCourseNotFound
	}

	// 2. 功能开关（pageConfig aiSummarySettings 热配置）。
	aiCfg := hotdataserve.GetAiSummarySettingsConfigCache()
	if !aiCfg.Enabled {
		return AiSummaryResult{Status: AiSummaryStatusDisabled}, nil
	}

	// 3. DB 缓存优先（!refresh 时）：
	//    - insufficient 行：已评估过且评价不足。命中时以当前可见有正文评价数为准
	//      重新判定（自愈）：评价已增加（写路径漏失效/阈值下调/offering 改派）时
	//      忽略过期标记继续生成，否则直接返回 insufficient_data。
	//    - generated 行（含存量无 status 列的行，DEFAULT 'generated'）：直接命中。
	if !refresh {
		if cached := course.GetCourseAiSummary(courseId); cached.CourseId > 0 {
			if cached.Status == course.AiSummaryRowStatusInsufficient {
				enough, err := aiSummaryReviewsSufficient(courseId)
				if err != nil {
					return AiSummaryResult{}, err
				}
				if !enough {
					return AiSummaryResult{Status: AiSummaryStatusInsufficientData}, nil
				}
				slog.Info("ai_summary_insufficient_self_healed", "courseId", courseId)
			} else if cached.SummaryJson != "" {
				payload, decodeErr := decodeAiSummaryPayload(cached.SummaryJson)
				if decodeErr != nil {
					slog.Warn("ai_summary_cache_decode_failed", "courseId", courseId, "error", decodeErr)
				} else {
					return AiSummaryResult{
						Status:      AiSummaryStatusCached,
						Summary:     &payload,
						GeneratedAt: cached.GeneratedAt.Format(time.RFC3339),
						Model:       cached.Model,
					}, nil
				}
			}
		}
	}

	// 4. 取最新 N 条可见有效评价（纯读，不消耗任何名额）。
	reviews, err := listRecentVisibleReviews(courseId)
	if err != nil {
		return AiSummaryResult{}, err
	}
	if len(reviews) < AiSummaryMinReviews {
		// 数据不足：落库 insufficient 标记（下次 check/请求直接命中，不再评估），
		// 不消耗任何限流名额。
		if err := upsertAiSummaryInsufficient(courseId); err != nil {
			slog.Error("ai_summary_insufficient_upsert_failed", "courseId", courseId, "error", err)
			return AiSummaryResult{}, err
		}
		return AiSummaryResult{Status: AiSummaryStatusInsufficientData}, nil
	}

	// 5. 限流（成本护栏，仅在确认要调 LLM 时消耗）：
	//    先查单课 10 分钟窗口是否已满（Count 不计数）——被单课拒绝的请求
	//    不消耗全局每分钟配额（review P1：避免个别课的 refresh 刷爆全局生成池）；
	//    再消耗全局配额（Allow 计数），最后记录单课生成（Allow 计数）。
	store := ratelimit.Default()
	courseKey := aiSummaryCourseKeyPrefix + fmt.Sprint(courseId)
	if store.Count(courseKey) >= aiSummaryCourseLimit {
		// 单课窗口已满：回退 DB 缓存（上游语义）；无缓存则 429，不消耗全局配额。
		if cached := course.GetCourseAiSummary(courseId); cached.CourseId > 0 && cached.SummaryJson != "" {
			if payload, err := decodeAiSummaryPayload(cached.SummaryJson); err == nil {
				return AiSummaryResult{
					Status:      AiSummaryStatusCached,
					Summary:     &payload,
					GeneratedAt: cached.GeneratedAt.Format(time.RFC3339),
					Model:       cached.Model,
				}, nil
			}
		}
		return AiSummaryResult{}, &AiSummaryRateLimitError{RetryAfter: aiSummaryCourseWindow}
	}

	globalLimit := aiCfg.GlobalPerMinute
	if globalLimit <= 0 {
		globalLimit = aiSummaryGlobalLimit
	}
	if ok, retryAfter, _ := store.Allow(aiSummaryGlobalKey, globalLimit, aiSummaryGlobalWindow); !ok {
		// 全局配额满：有缓存则回退缓存，无缓存返回错误（控制器映射 429）。
		if cached := course.GetCourseAiSummary(courseId); cached.CourseId > 0 && cached.SummaryJson != "" {
			if payload, err := decodeAiSummaryPayload(cached.SummaryJson); err == nil {
				return AiSummaryResult{
					Status:      AiSummaryStatusCached,
					Summary:     &payload,
					GeneratedAt: cached.GeneratedAt.Format(time.RFC3339),
					Model:       cached.Model,
				}, nil
			}
		}
		return AiSummaryResult{}, &AiSummaryRateLimitError{RetryAfter: retryAfter}
	}

	// 记录单课生成（10 分钟 1 次）。Count 与 Allow 之间的并发窗口内可能被
	// 其它请求抢先消耗单课名额，此时 Allow 拒绝——回退缓存或 429
	// （全局配额已在上面消耗，属罕见的可接受竞争）。
	if ok, retryAfter, _ := store.Allow(courseKey, aiSummaryCourseLimit, aiSummaryCourseWindow); !ok {
		if cached := course.GetCourseAiSummary(courseId); cached.CourseId > 0 && cached.SummaryJson != "" {
			if payload, err := decodeAiSummaryPayload(cached.SummaryJson); err == nil {
				return AiSummaryResult{
					Status:      AiSummaryStatusCached,
					Summary:     &payload,
					GeneratedAt: cached.GeneratedAt.Format(time.RFC3339),
					Model:       cached.Model,
				}, nil
			}
		}
		return AiSummaryResult{}, &AiSummaryRateLimitError{RetryAfter: retryAfter}
	}

	// 6. 构建 prompt 并调用 LLM。任何失败都返还单课名额：
	//    用户可立即重试，不会被已消耗的名额卡 10 分钟（全局名额保留）。
	cfg := resolveAiSummaryConfig()
	if !cfg.Enabled() {
		slog.Warn("ai_summary_provider_not_configured", "courseId", courseId)
		store.Reset(courseKey)
		return AiSummaryResult{}, ErrAiSummaryGenerationFailed
	}
	prompt := buildAiSummaryPrompt(entity.Name, entity.PrimaryCode, reviews)
	raw, err := llmChat(ctx, cfg, prompt)
	if err != nil {
		slog.Error("ai_summary_generation_failed", "courseId", courseId, "error", err)
		store.Reset(courseKey)
		return AiSummaryResult{}, ErrAiSummaryGenerationFailed
	}

	// 7. sanitize 校验输出（枚举/数组截断/非法值按评分分布兜底）。
	payload, err := sanitizeAiSummaryPayload(raw, reviews)
	if err != nil {
		slog.Error("ai_summary_sanitize_failed", "courseId", courseId, "error", err)
		store.Reset(courseKey)
		return AiSummaryResult{}, ErrAiSummaryGenerationFailed
	}

	// 8. 落库（UPSERT 覆盖，status=generated）。
	summaryJSON, err := json.Marshal(payload)
	if err != nil {
		return AiSummaryResult{}, err
	}
	now := time.Now()
	if err := course.UpsertCourseAiSummary(&course.CourseAiSummaryEntity{
		CourseId:      courseId,
		SummaryJson:   string(summaryJSON),
		Model:         cfg.Model,
		PromptVersion: promptVersion,
		GeneratedAt:   now,
		Status:        course.AiSummaryRowStatusGenerated,
	}); err != nil {
		slog.Error("ai_summary_upsert_failed", "courseId", courseId, "error", err)
		store.Reset(courseKey)
		return AiSummaryResult{}, err
	}
	return AiSummaryResult{
		Status:      AiSummaryStatusGenerated,
		Summary:     &payload,
		GeneratedAt: now.Format(time.RFC3339),
		Model:       cfg.Model,
	}, nil
}

// CheckAiSummary check 预检（前端挂载时调用）：只读 DB 返回当前状态，
// 不生成、不消耗任何限流名额。返回：
//   - cached：已有有效总结（携带 payload，前端自动展开）
//   - insufficient_data：已评估过且评价不足（前端保持折叠，展示占位）
//   - none：从未生成过（前端保持折叠，点击展开才触发生成）
//   - disabled：功能未启用
func CheckAiSummary(courseId uint64) (AiSummaryResult, error) {
	entity := course.GetCourse(courseId)
	if entity.Id == 0 || entity.Status != course.StatusVisible {
		return AiSummaryResult{}, ErrAiSummaryCourseNotFound
	}
	aiCfg := hotdataserve.GetAiSummarySettingsConfigCache()
	if !aiCfg.Enabled {
		return AiSummaryResult{Status: AiSummaryStatusDisabled}, nil
	}
	cached := course.GetCourseAiSummary(courseId)
	if cached.CourseId == 0 {
		return AiSummaryResult{Status: AiSummaryStatusNone}, nil
	}
	if cached.Status == course.AiSummaryRowStatusInsufficient {
		// 自愈：insufficient 标记过期（评价已增加/阈值下调）时返回 none，
		// 前端展开即触发生成；仍不足才返回 insufficient_data。
		enough, err := aiSummaryReviewsSufficient(courseId)
		if err != nil {
			return AiSummaryResult{}, err
		}
		if !enough {
			return AiSummaryResult{Status: AiSummaryStatusInsufficientData}, nil
		}
		return AiSummaryResult{Status: AiSummaryStatusNone}, nil
	}
	if cached.SummaryJson != "" {
		if payload, err := decodeAiSummaryPayload(cached.SummaryJson); err == nil {
			return AiSummaryResult{
				Status:      AiSummaryStatusCached,
				Summary:     &payload,
				GeneratedAt: cached.GeneratedAt.Format(time.RFC3339),
				Model:       cached.Model,
			}, nil
		}
	}
	// 行存在但 summary 损坏/为空：按未生成处理，允许重新生成。
	return AiSummaryResult{Status: AiSummaryStatusNone}, nil
}

// upsertAiSummaryInsufficient 落库「评价不足已评估」标记（无 summary_json）。
// 评价变更时写路径 DeleteCourseAiSummaryTx 会删除该行，课程重新可评估。
func upsertAiSummaryInsufficient(courseId uint64) error {
	return course.UpsertCourseAiSummary(&course.CourseAiSummaryEntity{
		CourseId:      courseId,
		PromptVersion: promptVersion,
		GeneratedAt:   time.Now(),
		Status:        course.AiSummaryRowStatusInsufficient,
	})
}

// AiSummaryRateLimitError 全局/单课生成限流错误（携带 Retry-After 供控制器回写 header）。
type AiSummaryRateLimitError struct {
	RetryAfter time.Duration
}

func (e *AiSummaryRateLimitError) Error() string {
	return "ai summary rate limited"
}

// listRecentVisibleReviews 返回课程下最新 N 条可见有效评价（仅评分与正文）。
// 复用 ListReviewsPage 的可见性口径：仅可见 offering + 可见评价 + 未软删；
// 分页续取直到收满 AiSummaryMaxReviews 条有正文的评价（review P2：空正文
// 不能占掉有效配额，避免"最新 30 行含空正文 → 误判 insufficient_data"）。
func listRecentVisibleReviews(courseId uint64) ([]reviewInput, error) {
	reviews := make([]reviewInput, 0, AiSummaryMaxReviews)
	var cursorOfferingId, cursorReviewId uint64
	for len(reviews) < AiSummaryMaxReviews {
		entities, err := course.ListReviewsPage(course.ReviewPageQuery{
			CourseId:         courseId,
			CursorOfferingId: cursorOfferingId,
			CursorReviewId:   cursorReviewId,
			Limit:            AiSummaryMaxReviews,
		})
		if err != nil {
			return nil, err
		}
		if len(entities) == 0 {
			break
		}
		for _, e := range entities {
			content := strings.TrimSpace(e.Content)
			if content == "" {
				continue
			}
			rating := 0
			if e.Rating != nil {
				rating = *e.Rating
			}
			if rating < 1 || rating > 5 {
				rating = 0
			}
			reviews = append(reviews, reviewInput{
				Rating:  rating,
				Content: truncateRunes(content, AiSummaryReviewMaxRune),
			})
			if len(reviews) >= AiSummaryMaxReviews {
				break
			}
		}
		last := entities[len(entities)-1]
		cursorOfferingId = last.OfferingId
		cursorReviewId = last.Id
	}
	return reviews, nil
}

// aiSummaryReviewsSufficient 判断课程当前可见有正文评价数是否达到生成阈值。
// 与生成路径同口径（listRecentVisibleReviews：仅可见 offering + 可见评价 +
// 未软删 + 有正文），供 insufficient 标记自愈校验——评价增加但写路径漏失效
// （catalog 导入改派 offering 等）或阈值下调时，不再永久返回 insufficient_data。
func aiSummaryReviewsSufficient(courseId uint64) (bool, error) {
	reviews, err := listRecentVisibleReviews(courseId)
	if err != nil {
		return false, err
	}
	return len(reviews) >= AiSummaryMinReviews, nil
}

// truncateRunes 按 rune 截断（与 ReviewContentMaxLength 的 rune 计数口径一致）。
func truncateRunes(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max])
}

// decodeAiSummaryPayload 解析缓存 JSON 为 payload。
func decodeAiSummaryPayload(raw string) (AiSummaryPayload, error) {
	var payload AiSummaryPayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return AiSummaryPayload{}, err
	}
	return payload, nil
}

// sanitizeAiSummaryPayload 校验并归一化 LLM 输出：
// 数组截断（keywords≤5/pros≤4/cons≤4/representative≤3）、枚举白名单校验、
// consensus 非法时按评分分布兜底映射（保证 schema 永不破，验收 1）。
func sanitizeAiSummaryPayload(raw string, reviews []reviewInput) (AiSummaryPayload, error) {
	var parsed struct {
		Consensus             string                          `json:"consensus"`
		Keywords              []string                        `json:"keywords"`
		Pros                  []string                        `json:"pros"`
		Cons                  []string                        `json:"cons"`
		RepresentativeReviews []AiSummaryRepresentativeReview `json:"representativeReviews"`
	}
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return AiSummaryPayload{}, err
	}
	payload := AiSummaryPayload{
		Consensus:             consensusFromString(parsed.Consensus),
		Keywords:              sanitizeStrings(parsed.Keywords, 5),
		Pros:                  sanitizeStrings(parsed.Pros, 4),
		Cons:                  sanitizeStrings(parsed.Cons, 4),
		RepresentativeReviews: sanitizeRepresentative(parsed.RepresentativeReviews, reviews),
	}
	if payload.Consensus == "" {
		payload.Consensus = consensusFromRatings(reviews)
	}
	return payload, nil
}

// consensusFromString 白名单映射；非法值返回 ""（由调用方按评分兜底）。
func consensusFromString(s string) AiSummaryConsensus {
	switch AiSummaryConsensus(s) {
	case ConsensusStrongRecommend, ConsensusRecommend, ConsensusNeutral, ConsensusCautious, ConsensusNotRecommend:
		return AiSummaryConsensus(s)
	}
	return ""
}

// consensusFromRatings 按评分分布映射五档（sanitize 兜底，验收 1 schema 合规）。
// 无评分（Rating<=0）的 legacy 评价不参与均分计算（review P2），
// 全部无评分时返回中性。
func consensusFromRatings(reviews []reviewInput) AiSummaryConsensus {
	var sum, count int
	for _, r := range reviews {
		if r.Rating < 1 || r.Rating > 5 {
			continue
		}
		sum += r.Rating
		count++
	}
	if count == 0 {
		return ConsensusNeutral
	}
	avg := float64(sum) / float64(count)
	switch {
	case avg >= 4.5:
		return ConsensusStrongRecommend
	case avg >= 3.5:
		return ConsensusRecommend
	case avg >= 2.5:
		return ConsensusNeutral
	case avg >= 1.5:
		return ConsensusCautious
	default:
		return ConsensusNotRecommend
	}
}

// sanitizeStrings 去空/截断数组。
func sanitizeStrings(items []string, max int) []string {
	out := make([]string, 0, len(items))
	for _, s := range items {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		out = append(out, s)
		if len(out) >= max {
			break
		}
	}
	return out
}

// sanitizeRepresentative 归一化代表性评价（sentiment 白名单，excerpt 截断；
// 且必须能在输入评价中匹配到原文摘录——review P2：防止模型幻觉/注入伪造
// "学生原话"）。过短摘录（<4 字）不做强校验，避免误杀合法摘要。
func sanitizeRepresentative(items []AiSummaryRepresentativeReview, reviews []reviewInput) []AiSummaryRepresentativeReview {
	out := make([]AiSummaryRepresentativeReview, 0, len(items))
	for _, r := range items {
		r.Excerpt = strings.TrimSpace(r.Excerpt)
		if r.Excerpt == "" {
			continue
		}
		if !excerptMatchesInput(r.Excerpt, reviews) {
			continue
		}
		switch r.Sentiment {
		case SentimentPositive, SentimentNeutral, SentimentNegative:
		default:
			r.Sentiment = SentimentNeutral
		}
		r.Excerpt = truncateRunes(r.Excerpt, 500)
		out = append(out, r)
		if len(out) >= 3 {
			break
		}
	}
	return out
}

// excerptMatchesInput 判断摘录是否为某条输入评价的子串（去除首尾引号与
// 省略号等语气符号后）。无评分评价的正文同样参与匹配。
func excerptMatchesInput(excerpt string, reviews []reviewInput) bool {
	needle := strings.TrimSpace(excerpt)
	needle = strings.Trim(needle, `「」『』“”"…`)
	needle = strings.TrimRight(needle, "。！？!?. ")
	if needle == "" {
		return false
	}
	if len([]rune(needle)) < 4 {
		return true // 过短，无法可靠判定，放行
	}
	for _, r := range reviews {
		if strings.Contains(r.Content, needle) {
			return true
		}
	}
	return false
}
