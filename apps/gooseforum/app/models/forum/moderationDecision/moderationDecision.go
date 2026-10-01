// Package moderationDecision 存放 AI 图文审查（issue #975）的结构化决策记录：
// 每次 AI 判定保存当时的政策版本、模型、原始概率信号与最终动作，人工审核结论
// 回写 HumanAction，供离线阈值回放与误杀/漏检评估；阈值调整无需重新调用模型。
package moderationDecision

import "time"

const tableName = "moderation_ai_decisions"

// 审查主体类型（与 moderationLog 主体同名）。
const (
	SubjectTopic = "topic"
	SubjectPost  = "post"
)

// 最终动作。
const (
	ActionAllow  = "allow"
	ActionReview = "review"
	ActionBlock  = "block"
)

// 证据状态：complete 表示所有图片证据与 Jev 决策均有效；其余任一值都意味着
// 判定不完整，resolver 只会给出 review（或外链 block 策略），绝不 allow。
const (
	EvidenceComplete      = "complete"
	EvidenceUnavailable   = "unavailable"    // 视觉模型拒答/超时/非 JSON/不可读等
	EvidenceExternal      = "external"       // 含非站内外链图片（P0 不做服务端抓取）
	EvidenceTooMany       = "too_many"       // 图片数超过 maxImagesPerDecision
	EvidenceJevFailed     = "jev_failed"     // Jev 超时/限流/5xx/概率无效
	EvidenceNotConfigured = "not_configured" // 端点/模型/密钥缺失
	EvidenceRateLimited   = "rate_limited"   // 命中 AI 调用成本护栏
)

// 人工结论（审核队列 approve/reject 或管理端标注）。
const (
	HumanApproved = "approved" // 人工认为可公开（AI 若判 review/block 即误杀样本）
	HumanRejected = "rejected" // 人工认为违规（AI 若判 allow 即漏检样本）
)

// 结论原因码（解释“为什么是这个结果”，管理端本地化展示）。
const (
	ReasonBlockThreshold       = "block_threshold"        // 分数达到拦截线，规则设为拦截
	ReasonSeverityEscalation   = "severity_escalation"    // 分数达到转人工线且严重程度达到提级线
	ReasonBetweenThresholds    = "between_thresholds"     // 规则设为拦截，但分数介于转人工线与拦截线之间
	ReasonReviewOnlyRule       = "review_only_rule"       // 规则设为只转人工
	ReasonReviewNeeded         = "review_needed"          // 模型认为需要人工确认
	ReasonEvidenceIncomplete   = "evidence_incomplete"    // 证据不完整（Detail 为证据状态）
	ReasonExternalImageBlocked = "external_image_blocked" // 站外图片且策略为拦截
)

// Reason 单条结论原因：Score 为规则分数（或 review_needed 概率），Threshold 为
// 达到/比较的分数线，Limit 为上方的拦截线，Severity 为严重程度。
type Reason struct {
	Code      string   `json:"code"`
	Policy    string   `json:"policy,omitempty"`
	Score     *float64 `json:"score,omitempty"`
	Threshold *float64 `json:"threshold,omitempty"`
	Limit     *float64 `json:"limit,omitempty"`
	Severity  *float64 `json:"severity,omitempty"`
	Detail    string   `json:"detail,omitempty"`
}

// Signals Jev 返回的原始概率信号（保存原值，阈值回放只读这里）。
type Signals struct {
	RuleProbabilities map[string]float64 `json:"ruleProbabilities,omitempty"`
	Severity          *float64           `json:"severity,omitempty"`
	ReviewNeeded      *float64           `json:"reviewNeeded,omitempty"`
}

// ImageRecord 单张图片的审计摘要。Evidence 仅在结论为 review 时保存（截断），
// 供审核员理解 AI 触因；allow/block 不长期保存图片转述。
type ImageRecord struct {
	FileName string `json:"fileName,omitempty"`
	URL      string `json:"url,omitempty"`
	SHA256   string `json:"sha256,omitempty"`
	Status   string `json:"status"` // ok | cache | refused | invalid | error | too_large | unsupported | external | skipped
	Evidence string `json:"evidence,omitempty"`
}

type Entity struct {
	Id                uint64        `gorm:"primaryKey;column:id;autoIncrement;not null;" json:"id"`
	SubjectType       string        `gorm:"column:subject_type;type:varchar(32);not null;default:'';index:idx_moderation_ai_decisions_subject,priority:1;" json:"subjectType"`
	SubjectId         uint64        `gorm:"column:subject_id;not null;default:0;index:idx_moderation_ai_decisions_subject,priority:2;" json:"subjectId"`
	AuthorId          uint64        `gorm:"column:author_id;not null;default:0;index:idx_moderation_ai_decisions_author;" json:"authorId"`
	Mode              string        `gorm:"column:mode;type:varchar(16);not null;default:'';" json:"mode"`
	PolicyRevision    string        `gorm:"column:policy_revision;type:varchar(64);not null;default:'';" json:"policyRevision"`
	PolicyHash        string        `gorm:"column:policy_hash;type:varchar(64);not null;default:'';" json:"policyHash"`
	StateHash         string        `gorm:"column:state_hash;type:varchar(64);not null;default:'';" json:"stateHash"`
	QuestionSchemaVer string        `gorm:"column:question_schema_ver;type:varchar(32);not null;default:'';" json:"questionSchemaVer"`
	VisionModel       string        `gorm:"column:vision_model;type:varchar(128);not null;default:'';" json:"visionModel"`
	JevModel          string        `gorm:"column:jev_model;type:varchar(128);not null;default:'';" json:"jevModel"`
	JevProvider       string        `gorm:"column:jev_provider;type:varchar(64);not null;default:'';" json:"jevProvider"`
	Images            []ImageRecord `gorm:"column:images;type:text;serializer:json" json:"images"`
	Signals           Signals       `gorm:"column:signals;type:text;serializer:json" json:"signals"`
	TriggeredPolicies []string      `gorm:"column:triggered_policies;type:text;serializer:json" json:"triggeredPolicies"`
	Reasons           []Reason      `gorm:"column:reasons;type:text;serializer:json" json:"reasons"`
	EvidenceStatus    string        `gorm:"column:evidence_status;type:varchar(32);not null;default:'';" json:"evidenceStatus"`
	FinalAction       string        `gorm:"column:final_action;type:varchar(16);not null;default:'';index:idx_moderation_ai_decisions_final;" json:"finalAction"`
	AppliedAction     string        `gorm:"column:applied_action;type:varchar(16);not null;default:'';" json:"appliedAction"` // shadow 恒为 allow
	ErrorKind         string        `gorm:"column:error_kind;type:varchar(64);not null;default:'';" json:"errorKind"`
	HumanAction       string        `gorm:"column:human_action;type:varchar(16);not null;default:'';index:idx_moderation_ai_decisions_human;" json:"humanAction"`
	HumanActorId      uint64        `gorm:"column:human_actor_id;not null;default:0;" json:"humanActorId"`
	HumanAt           *time.Time    `gorm:"column:human_at;null;" json:"humanAt"`
	LatencyMs         int64         `gorm:"column:latency_ms;not null;default:0;" json:"latencyMs"`
	Cost              float64       `gorm:"column:cost;not null;default:0;" json:"cost"`
	CreatedAt         time.Time     `gorm:"column:created_at;autoCreateTime;<-:create;" json:"createdAt"`
}

func (itself *Entity) TableName() string {
	return tableName
}
