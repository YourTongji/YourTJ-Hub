package api

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/markdown2html"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/securestore"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/moderationDecision"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/moderationservice"
)

// 管理端 AI 图文审查（issue #975）：配置读写、决策记录、人工标注与离线阈值回放。
// 全部挂在 SiteManager 权限组（与审核队列同权）。

// GetAiModerationSettings 回显配置：两个 API key 只回显是否已配置。
func GetAiModerationSettings(req component.BetterRequest[component.Null]) component.Response {
	return component.SuccessResponse(hotdataserve.GetAiModerationSettingsView())
}

type SaveAiModerationSettingsReq struct {
	Settings pageConfig.AiModerationSettingsInput `json:"settings" validate:"required"`
}

// SaveAiModerationSettings 保存配置：key 明文只在请求瞬间存在，非空时加密落库，
// 空串保留旧密文，clear*ApiKey=true 显式清除。保存后清缓存，热生效。
func SaveAiModerationSettings(req component.BetterRequest[SaveAiModerationSettingsReq]) component.Response {
	input := req.Params.Settings
	options := input.AiModerationOptions.Normalize()
	if options.JevEndpoint != "" && !isValidHTTPURL(options.JevEndpoint) {
		return component.FailResponseCode(component.MessageAdminAiModerationSaveFailed,
			component.MessageParams{"error": "请填写以 http:// 或 https:// 开头的 Jev 地址"})
	}
	if options.VisionBaseURL != "" && !isValidHTTPURL(options.VisionBaseURL) {
		return component.FailResponseCode(component.MessageAdminAiModerationSaveFailed,
			component.MessageParams{"error": "请填写以 http:// 或 https:// 开头的视觉模型地址"})
	}
	storage := hotdataserve.GetAiModerationSettingsStorage()
	storage.AiModerationOptions = options
	jevKey, err := mergeSecret(storage.JevAPIKeyEncrypted, input.JevAPIKey, input.ClearJevAPIKey, securestore.ModerationJevAPIKeyPurpose)
	if err != nil {
		return component.FailResponseError(err)
	}
	visionKey, err := mergeSecret(storage.VisionAPIKeyEncrypted, input.VisionAPIKey, input.ClearVisionAPIKey, securestore.ModerationVisionAPIKeyPurpose)
	if err != nil {
		return component.FailResponseError(err)
	}
	storage.JevAPIKeyEncrypted, storage.VisionAPIKeyEncrypted = jevKey, visionKey
	return savePageConfig(pageConfig.AiModerationPage, storage, hotdataserve.ClearAiModerationConfigCache)
}

// mergeSecret 计算保存后的密文：clear 优先；新明文非空则加密；否则保留旧密文。
func mergeSecret(existing, plain string, clear bool, purpose string) (string, error) {
	if clear {
		return "", nil
	}
	plain = strings.TrimSpace(plain)
	if plain == "" {
		return existing, nil
	}
	sealed, err := securestore.EncryptPurpose(plain, purpose)
	if err != nil {
		return "", fmt.Errorf("无法加密 API 密钥。请确认已配置 app.signingKey：%w", err)
	}
	return sealed, nil
}

type AiModerationTestReq struct {
	// Target jev | vision；Settings 为当前表单值（未保存也可测试），key 留空时用已存 key。
	Target   string                               `json:"target" validate:"required,oneof=jev vision"`
	Settings pageConfig.AiModerationSettingsInput `json:"settings"`
}

// TestAiModerationConnection 管理端「测试连接」：用表单值真实调用一次 provider，
// 返回分类/状态码/耗时（不回带响应原文与 key）。不保存任何配置。
func TestAiModerationConnection(req component.BetterRequest[AiModerationTestReq]) component.Response {
	input := req.Params.Settings
	options := input.AiModerationOptions.Normalize()
	if (options.JevEndpoint != "" && !isValidHTTPURL(options.JevEndpoint)) ||
		(options.VisionBaseURL != "" && !isValidHTTPURL(options.VisionBaseURL)) {
		return component.FailResponseCode(component.MessageAdminAiModerationSaveFailed,
			component.MessageParams{"error": "请填写以 http:// 或 https:// 开头的地址"})
	}
	stored := hotdataserve.GetAiModerationConfigCache()
	cfg := pageConfig.AiModerationConfig{
		AiModerationOptions: options,
		JevAPIKey:           pickTestKey(input.JevAPIKey, input.ClearJevAPIKey, stored.JevAPIKey),
		VisionAPIKey:        pickTestKey(input.VisionAPIKey, input.ClearVisionAPIKey, stored.VisionAPIKey),
	}
	extendWriteDeadline(req.GinContext, 45*time.Second)
	ctx, cancel := context.WithTimeout(requestContext(req.GinContext), 40*time.Second)
	defer cancel()
	if req.Params.Target == "vision" {
		return component.SuccessResponse(moderationservice.TestAIVisionConnection(ctx, cfg))
	}
	return component.SuccessResponse(moderationservice.TestAIJevConnection(ctx, cfg))
}

func pickTestKey(plain string, clear bool, stored string) string {
	if clear {
		return ""
	}
	if plain = strings.TrimSpace(plain); plain != "" {
		return plain
	}
	return stored
}

type AiModerationDecisionListReq struct {
	Page        int    `json:"page"`
	PageSize    int    `json:"pageSize"`
	FinalAction string `json:"finalAction" validate:"omitempty,oneof=allow review block"`
	HumanAction string `json:"humanAction" validate:"omitempty,oneof=none approved rejected"`
	Mode        string `json:"mode" validate:"omitempty,oneof=shadow enforce"`
}

// AiModerationDecisionItem 决策记录的管理端视图（不含任何密钥/原始 prompt）。
type AiModerationDecisionItem struct {
	Id                uint64                           `json:"id"`
	SubjectType       string                           `json:"subjectType"`
	SubjectId         uint64                           `json:"subjectId"`
	AuthorId          uint64                           `json:"authorId"`
	Mode              string                           `json:"mode"`
	PolicyRevision    string                           `json:"policyRevision"`
	VisionModel       string                           `json:"visionModel"`
	JevModel          string                           `json:"jevModel"`
	Images            []moderationDecision.ImageRecord `json:"images"`
	Signals           moderationDecision.Signals       `json:"signals"`
	TriggeredPolicies []string                         `json:"triggeredPolicies"`
	Reasons           []moderationDecision.Reason      `json:"reasons"`
	EvidenceStatus    string                           `json:"evidenceStatus"`
	FinalAction       string                           `json:"finalAction"`
	AppliedAction     string                           `json:"appliedAction"`
	ErrorKind         string                           `json:"errorKind"`
	HumanAction       string                           `json:"humanAction"`
	LatencyMs         int64                            `json:"latencyMs"`
	Cost              float64                          `json:"cost"`
	CreatedAt         string                           `json:"createdAt"`
}

func aiDecisionItem(entity moderationDecision.Entity) AiModerationDecisionItem {
	images := entity.Images
	if images == nil {
		images = []moderationDecision.ImageRecord{}
	}
	triggered := entity.TriggeredPolicies
	if triggered == nil {
		triggered = []string{}
	}
	reasons := entity.Reasons
	if reasons == nil {
		reasons = []moderationDecision.Reason{}
	}
	return AiModerationDecisionItem{
		Id: entity.Id, SubjectType: entity.SubjectType, SubjectId: entity.SubjectId, AuthorId: entity.AuthorId,
		Mode: entity.Mode, PolicyRevision: entity.PolicyRevision, VisionModel: entity.VisionModel, JevModel: entity.JevModel,
		Images: images, Signals: entity.Signals, TriggeredPolicies: triggered, Reasons: reasons, EvidenceStatus: entity.EvidenceStatus,
		FinalAction: entity.FinalAction, AppliedAction: entity.AppliedAction, ErrorKind: entity.ErrorKind,
		HumanAction: entity.HumanAction, LatencyMs: entity.LatencyMs, Cost: entity.Cost,
		CreatedAt: entity.CreatedAt.Format(time.RFC3339),
	}
}

// ListAiModerationDecisions 分页列出 AI 决策（最新在前）。
func ListAiModerationDecisions(req component.BetterRequest[AiModerationDecisionListReq]) component.Response {
	page := max(req.Params.Page, 1)
	pageSize := req.Params.PageSize
	if pageSize < 1 || pageSize > 50 {
		pageSize = 20
	}
	list, total := moderationDecision.Page(moderationDecision.ListQuery{
		Page: page, PageSize: pageSize, FinalAction: req.Params.FinalAction,
		HumanAction: req.Params.HumanAction, Mode: req.Params.Mode,
	})
	items := make([]AiModerationDecisionItem, 0, len(list))
	for _, entity := range list {
		items = append(items, aiDecisionItem(entity))
	}
	return component.SuccessResponse(map[string]any{"items": items, "total": total, "page": page, "pageSize": pageSize})
}

type AiModerationLabelReq struct {
	Id    uint64 `json:"id" validate:"required"`
	Label string `json:"label" validate:"required,oneof=approved rejected"`
}

// LabelAiModerationDecision 人工标注一条决策（shadow 样本主要靠这里积累标签）。
func LabelAiModerationDecision(req component.BetterRequest[AiModerationLabelReq]) component.Response {
	entity := moderationDecision.Get(req.Params.Id)
	if entity.Id == 0 {
		return component.FailResponseCode(component.MessageAdminAiModerationDecisionNotFound, nil)
	}
	if err := moderationDecision.SetHumanAction(entity.Id, req.Params.Label, req.UserId, time.Now()); err != nil {
		return component.FailResponseCode(component.MessageOperationFailed, nil)
	}
	return component.SuccessResponseCode("success", component.MessageOperationSuccess, nil)
}

type AiModerationReplayReq struct {
	// Options 为空时用当前已保存配置回放；非空时用候选阈值/规则回放（不保存）。
	Options *pageConfig.AiModerationOptions `json:"options,omitempty"`
}

// ReplayAiModerationDecisions 用当前或候选阈值对已标注样本离线重放，输出混淆
// 矩阵；只读保存的原始概率，不重新调用任何模型。
func ReplayAiModerationDecisions(req component.BetterRequest[AiModerationReplayReq]) component.Response {
	options := hotdataserve.GetAiModerationConfigCache().AiModerationOptions
	if req.Params.Options != nil {
		options = req.Params.Options.Normalize()
	}
	return component.SuccessResponse(moderationservice.ReplayAIDecisions(options, moderationDecision.ListLabeled(moderationservice.AIReplaySampleLimit)))
}

// reviewQueueImages 审核队列条目的图片预览地址（经 /file/img 授权预览读取）。
func reviewQueueImages(urls []string) []string {
	const limit = 9
	out := make([]string, 0, min(len(urls), limit))
	seen := make(map[string]bool, len(urls))
	for _, value := range urls {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
		if len(out) == limit {
			break
		}
	}
	return out
}

func reviewQueuePostImages(content string) []string {
	return reviewQueueImages(markdown2html.ExtractImageURLs(content))
}
