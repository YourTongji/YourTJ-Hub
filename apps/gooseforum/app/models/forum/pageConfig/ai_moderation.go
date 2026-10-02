package pageConfig

import (
	"math"
	"strings"
)

// AI 图文审查（issue #975）运行模式：shadow 只记录建议不影响发布（异步执行，
// 用于积累人工标签与阈值校准）；enforce 在发布请求内同步执行并应用结论。
const (
	AiModerationModeShadow  = "shadow"
	AiModerationModeEnforce = "enforce"
)

// AI 审查结论动作：政策规则的动作上限与外链图片策略均只取 review | block。
const (
	AiModerationActionReview = "review"
	AiModerationActionBlock  = "block"
)

// 固定的政策规则键（同时作为 Jev 问题名）。规则文案由管理员维护，模型只执行。
const (
	AiPolicyAdult              = "adult"
	AiPolicyPoliticalSensitive = "political_sensitive"
	AiPolicyViolence           = "violence"
	AiPolicyIllegalOrDangerous = "illegal_or_dangerous"
	AiPolicyOther              = "other"
)

// AiModerationPolicyKeys 规则键的稳定顺序（管理端展示与 Jev 问题顺序一致）。
var AiModerationPolicyKeys = []string{
	AiPolicyAdult, AiPolicyPoliticalSensitive, AiPolicyViolence, AiPolicyIllegalOrDangerous, AiPolicyOther,
}

// AiModerationPolicyRule 单条站点政策规则：Definition 是写进 Jev state 的站点
// 规则原文；Action 为该规则可触发的最高动作（review 规则无论概率多高都不自动
// block）；阈值为空时取全局默认阈值。
type AiModerationPolicyRule struct {
	Key             string   `json:"key"`
	Label           string   `json:"label"`
	Definition      string   `json:"definition"`
	Enabled         bool     `json:"enabled"`
	Action          string   `json:"action"`
	ReviewThreshold *float64 `json:"reviewThreshold,omitempty"`
	BlockThreshold  *float64 `json:"blockThreshold,omitempty"`
}

// AiModerationOptions AI 审查的非敏感配置（四种形状共享，避免字段重复）。
type AiModerationOptions struct {
	Enabled bool   `json:"enabled"`
	Mode    string `json:"mode"` // shadow | enforce
	// TextModeration 无图内容是否也调用 Jev 做文本语义审查；关闭时无图内容
	// 不产生任何 AI 调用（敏感词仍照常生效）。
	TextModeration bool `json:"textModeration"`

	JevEndpoint  string `json:"jevEndpoint"` // 完整 Decisions API URL（TypeSafe /v1/systemone 或 OpenRouter /api/alpha/decisions）
	JevModel     string `json:"jevModel"`
	JevTimeoutMs int    `json:"jevTimeoutMs"`
	JevRetries   int    `json:"jevRetries"` // 仅对 402/429/5xx/超时重试，最多 1 次

	VisionBaseURL   string `json:"visionBaseUrl"` // OpenAI-compatible 多模态端点，如 https://openrouter.ai/api/v1
	VisionModel     string `json:"visionModel"`
	VisionTimeoutMs int    `json:"visionTimeoutMs"`

	PolicyRevision         string                   `json:"policyRevision"`
	Policies               []AiModerationPolicyRule `json:"policies"`
	DefaultReviewThreshold float64                  `json:"defaultReviewThreshold"`
	DefaultBlockThreshold  float64                  `json:"defaultBlockThreshold"`
	ReviewNeededThreshold  float64                  `json:"reviewNeededThreshold"`
	SeverityBlockThreshold float64                  `json:"severityBlockThreshold"` // 0..3，仅对已越过 review 阈值的 block 规则升级

	ExternalImageAction  string `json:"externalImageAction"` // review | block：非站内图片不做服务端抓取
	MaxImagesPerDecision int    `json:"maxImagesPerDecision"`

	GlobalRequestsPerMinute  int `json:"globalRequestsPerMinute"`
	PerUserRequestsPerMinute int `json:"perUserRequestsPerMinute"`
}

// AiModerationConfig 运行时配置：两个 API Key 为 securestore 解密后的明文，
// 标 json:"-"，绝不随 JSON 序列化导出（issue #324 安全模式）。
type AiModerationConfig struct {
	AiModerationOptions
	JevAPIKey    string `json:"-"`
	VisionAPIKey string `json:"-"`
}

// AiModerationSettingsStorage 落库形状：密钥只以密文出现。
type AiModerationSettingsStorage struct {
	AiModerationOptions
	JevAPIKeyEncrypted    string `json:"jevApiKeyEncrypted,omitempty"`
	VisionAPIKeyEncrypted string `json:"visionApiKeyEncrypted,omitempty"`
}

// AiModerationSettingsView 管理端 GET 回显：密钥只回显是否已配置。
type AiModerationSettingsView struct {
	AiModerationOptions
	JevAPIKeyConfigured    bool `json:"jevApiKeyConfigured"`
	VisionAPIKeyConfigured bool `json:"visionApiKeyConfigured"`
}

// AiModerationSettingsInput 管理端保存请求：密钥明文仅在请求瞬间存在；空串
// 保留已存密文，Clear* 为 true 时显式清除（优先于新值）。
type AiModerationSettingsInput struct {
	AiModerationOptions
	JevAPIKey         string `json:"jevApiKey,omitempty"`
	VisionAPIKey      string `json:"visionApiKey,omitempty"`
	ClearJevAPIKey    bool   `json:"clearJevApiKey,omitempty"`
	ClearVisionAPIKey bool   `json:"clearVisionApiKey,omitempty"`
}

// ToView 生成回显视图（选项先归一化，管理端总能看到生效值）。
func (s AiModerationSettingsStorage) ToView() AiModerationSettingsView {
	return AiModerationSettingsView{
		AiModerationOptions:    s.AiModerationOptions.Normalize(),
		JevAPIKeyConfigured:    strings.TrimSpace(s.JevAPIKeyEncrypted) != "",
		VisionAPIKeyConfigured: strings.TrimSpace(s.VisionAPIKeyEncrypted) != "",
	}
}

// DefaultAiModerationPolicies 默认规则草案。文案仅为起点：上线 enforce 前必须由
// 管理员按社区规则确认（MADR-0056）；默认动作一律 review，不自动拦截。
func DefaultAiModerationPolicies() []AiModerationPolicyRule {
	return []AiModerationPolicyRule{
		{Key: AiPolicyAdult, Label: "成人内容", Enabled: true, Action: AiModerationActionReview,
			Definition: "露骨的性行为、性器官特写、以性挑逗为目的的裸露或性暗示姿态，以及任何涉及未成年人的性化内容。普通着装、医学/艺术/教育语境中的非性化人体不属于此类。"},
		{Key: AiPolicyPoliticalSensitive, Label: "政治敏感", Enabled: true, Action: AiModerationActionReview,
			Definition: "违反本站社区规则的政治敏感内容，以管理员书面规则为准。仅依据内容中明确可见的文字、符号与场景判断，不推断作者或人物的政治立场、身份与动机；正常的时事与学术讨论不属于此类。"},
		{Key: AiPolicyViolence, Label: "暴力血腥", Enabled: true, Action: AiModerationActionReview,
			Definition: "真实或逼真的严重伤害、血腥、尸体、虐待、自残画面，或宣扬、美化、威胁实施暴力。克制的新闻报道、体育运动与影视道具不属于此类。"},
		{Key: AiPolicyIllegalOrDangerous, Label: "违法危险", Enabled: true, Action: AiModerationActionReview,
			Definition: "毒品或管制物品的交易与制作、武器制作、赌博、诈骗、代考代写等学术作弊交易、泄露他人身份证号/手机号等隐私、危险行为教程。"},
		{Key: AiPolicyOther, Label: "其他违规", Enabled: false, Action: AiModerationActionReview,
			Definition: "其他违反本站社区规则的内容，例如广告引流、垃圾信息、人身攻击与骚扰。"},
	}
}

// 默认值与边界。阈值仅为 shadow 校准前的起点，不得未经人工样本回放直接用于 enforce。
const (
	aiDefaultReviewThreshold   = 0.5
	aiDefaultBlockThreshold    = 0.9
	aiDefaultReviewNeeded      = 0.7
	aiDefaultSeverityBlock     = 2.5
	aiDefaultJevTimeoutMs      = 10000
	aiDefaultVisionTimeoutMs   = 20000
	aiMinTimeoutMs             = 1000
	aiMaxTimeoutMs             = 60000
	aiDefaultMaxImages         = 6
	aiMaxImagesLimit           = 20
	aiDefaultGlobalPerMinute   = 30
	aiDefaultPerUserPerMinute  = 5
	aiMaxPolicyDefinitionRunes = 1000
	aiMaxPolicyLabelRunes      = 40
)

// Normalize 归一化选项：非法枚举回落默认值、阈值与计数截断到合法区间、规则
// 按固定键合并默认草案（未知键丢弃）。读写两侧都调用，存量配置缺字段也安全。
func (o AiModerationOptions) Normalize() AiModerationOptions {
	if o.Mode != AiModerationModeEnforce {
		o.Mode = AiModerationModeShadow
	}
	o.JevEndpoint = strings.TrimSpace(o.JevEndpoint)
	o.JevModel = strings.TrimSpace(o.JevModel)
	o.VisionBaseURL = strings.TrimRight(strings.TrimSpace(o.VisionBaseURL), "/")
	o.VisionModel = strings.TrimSpace(o.VisionModel)
	o.JevTimeoutMs = clampInt(o.JevTimeoutMs, aiDefaultJevTimeoutMs, aiMinTimeoutMs, aiMaxTimeoutMs)
	o.VisionTimeoutMs = clampInt(o.VisionTimeoutMs, aiDefaultVisionTimeoutMs, aiMinTimeoutMs, aiMaxTimeoutMs)
	if o.JevRetries < 0 {
		o.JevRetries = 0
	}
	if o.JevRetries > 1 {
		o.JevRetries = 1
	}
	o.PolicyRevision = strings.TrimSpace(o.PolicyRevision)
	if o.PolicyRevision == "" {
		o.PolicyRevision = "default-1"
	}
	o.DefaultReviewThreshold = clampProbability(o.DefaultReviewThreshold, aiDefaultReviewThreshold)
	o.DefaultBlockThreshold = clampProbability(o.DefaultBlockThreshold, aiDefaultBlockThreshold)
	if o.DefaultBlockThreshold < o.DefaultReviewThreshold {
		o.DefaultBlockThreshold = o.DefaultReviewThreshold
	}
	o.ReviewNeededThreshold = clampProbability(o.ReviewNeededThreshold, aiDefaultReviewNeeded)
	if o.SeverityBlockThreshold <= 0 || o.SeverityBlockThreshold > 3 || math.IsNaN(o.SeverityBlockThreshold) {
		o.SeverityBlockThreshold = aiDefaultSeverityBlock
	}
	if o.ExternalImageAction != AiModerationActionBlock {
		o.ExternalImageAction = AiModerationActionReview
	}
	o.MaxImagesPerDecision = clampInt(o.MaxImagesPerDecision, aiDefaultMaxImages, 1, aiMaxImagesLimit)
	o.GlobalRequestsPerMinute = clampInt(o.GlobalRequestsPerMinute, aiDefaultGlobalPerMinute, 1, 10000)
	o.PerUserRequestsPerMinute = clampInt(o.PerUserRequestsPerMinute, aiDefaultPerUserPerMinute, 1, 1000)
	o.Policies = normalizePolicies(o.Policies)
	return o
}

func normalizePolicies(input []AiModerationPolicyRule) []AiModerationPolicyRule {
	byKey := make(map[string]AiModerationPolicyRule, len(input))
	for _, rule := range input {
		byKey[rule.Key] = rule
	}
	defaults := DefaultAiModerationPolicies()
	out := make([]AiModerationPolicyRule, 0, len(defaults))
	for _, def := range defaults {
		rule, ok := byKey[def.Key]
		if !ok {
			out = append(out, def)
			continue
		}
		rule.Key = def.Key
		rule.Label = truncateRunes(strings.TrimSpace(rule.Label), aiMaxPolicyLabelRunes)
		if rule.Label == "" {
			rule.Label = def.Label
		}
		rule.Definition = truncateRunes(strings.TrimSpace(rule.Definition), aiMaxPolicyDefinitionRunes)
		if rule.Definition == "" {
			rule.Definition = def.Definition
		}
		if rule.Action != AiModerationActionBlock {
			rule.Action = AiModerationActionReview
		}
		rule.ReviewThreshold = clampOptionalProbability(rule.ReviewThreshold)
		rule.BlockThreshold = clampOptionalProbability(rule.BlockThreshold)
		out = append(out, rule)
	}
	return out
}

func clampInt(value, fallback, lowest, highest int) int {
	if value <= 0 {
		return fallback
	}
	return min(max(value, lowest), highest)
}

// clampProbability 0 或非法值视为未配置（回落默认）；越界截断到 (0,1]。
func clampProbability(value, fallback float64) float64 {
	if value <= 0 || math.IsNaN(value) {
		return fallback
	}
	return math.Min(value, 1)
}

func clampOptionalProbability(value *float64) *float64 {
	if value == nil || *value <= 0 || math.IsNaN(*value) {
		return nil
	}
	v := math.Min(*value, 1)
	return &v
}

func truncateRunes(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit])
}

// ReviewThresholdFor 规则 review 阈值（未覆盖时取全局默认）。
func (o AiModerationOptions) ReviewThresholdFor(rule AiModerationPolicyRule) float64 {
	if rule.ReviewThreshold != nil {
		return *rule.ReviewThreshold
	}
	return o.DefaultReviewThreshold
}

// BlockThresholdFor 规则 block 阈值（未覆盖时取全局默认；不低于 review 阈值）。
func (o AiModerationOptions) BlockThresholdFor(rule AiModerationPolicyRule) float64 {
	threshold := o.DefaultBlockThreshold
	if rule.BlockThreshold != nil {
		threshold = *rule.BlockThreshold
	}
	return math.Max(threshold, o.ReviewThresholdFor(rule))
}
