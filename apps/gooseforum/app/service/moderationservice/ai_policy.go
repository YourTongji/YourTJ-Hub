package moderationservice

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/moderationDecision"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
)

// AIQuestionSchemaVersion Jev 问题形状版本（每条政策一个并行 Noul + severity
// Score + review_needed Noul）。问题措辞或结构变化时递增，决策记录据此区分样本。
const AIQuestionSchemaVersion = "jev-q1"

// Jev 中除政策规则外的两个固定问题名。
const (
	aiQuestionSeverity     = "severity"
	aiQuestionReviewNeeded = "review_needed"
)

// AISignals Jev 返回并通过校验的概率信号。EvidenceComplete=false 表示有图片
// 证据缺失（拒答/超时/超量等），此时 resolver 不会给出 allow。
type AISignals struct {
	RuleProbabilities map[string]float64
	Severity          *float64
	ReviewNeeded      *float64
	EvidenceComplete  bool
}

// AIResolution resolver 结论：Action ∈ allow/review/block；Triggered 为越过
// review 阈值的全部规则（多值，不收敛成单一类别）。
type AIResolution struct {
	Action    string
	Triggered []string
	Reasons   []moderationDecision.Reason
}

// ResolveAISignals 把 Jev 信号按站点配置转为动作。最终动作只由这里的确定性
// 代码决定，模型不直接给出 allow/review/block：
//  1. 任一启用规则 p ≥ review 阈值 → 至少 review，并计入 Triggered；
//  2. 动作上限为 block 的规则：p ≥ block 阈值，或 p 已越过 review 阈值且
//     severity ≥ severityBlockThreshold → block（severity 不能单独 block）；
//  3. review_needed ≥ 阈值 → 至少 review；
//  4. 证据不完整 → 至少 review；
//  5. 其余 → allow。
func ResolveAISignals(opts pageConfig.AiModerationOptions, signals AISignals) AIResolution {
	resolution := AIResolution{Action: moderationDecision.ActionAllow, Triggered: []string{}, Reasons: []moderationDecision.Reason{}}
	for _, rule := range opts.Policies {
		if !rule.Enabled {
			continue
		}
		probability, ok := signals.RuleProbabilities[rule.Key]
		reviewLine, blockLine := opts.ReviewThresholdFor(rule), opts.BlockThresholdFor(rule)
		if !ok || probability < reviewLine {
			continue
		}
		resolution.Triggered = append(resolution.Triggered, rule.Key)
		reason := moderationDecision.Reason{Policy: rule.Key, Score: ptrFloat(probability), Threshold: ptrFloat(reviewLine)}
		switch {
		case rule.Action != pageConfig.AiModerationActionBlock:
			reason.Code = moderationDecision.ReasonReviewOnlyRule
		case probability >= blockLine:
			reason.Code, reason.Threshold = moderationDecision.ReasonBlockThreshold, ptrFloat(blockLine)
		case signals.Severity != nil && *signals.Severity >= opts.SeverityBlockThreshold:
			reason.Code, reason.Severity, reason.Limit = moderationDecision.ReasonSeverityEscalation, signals.Severity, ptrFloat(opts.SeverityBlockThreshold)
		default:
			reason.Code, reason.Limit = moderationDecision.ReasonBetweenThresholds, ptrFloat(blockLine)
		}
		resolution.Reasons = append(resolution.Reasons, reason)
		if reason.Code == moderationDecision.ReasonBlockThreshold || reason.Code == moderationDecision.ReasonSeverityEscalation {
			resolution.Action = moderationDecision.ActionBlock
			continue
		}
		resolution.Action = strongerAction(resolution.Action, moderationDecision.ActionReview)
	}
	if signals.ReviewNeeded != nil && *signals.ReviewNeeded >= opts.ReviewNeededThreshold {
		resolution.Action = strongerAction(resolution.Action, moderationDecision.ActionReview)
		resolution.Reasons = append(resolution.Reasons, moderationDecision.Reason{
			Code: moderationDecision.ReasonReviewNeeded, Score: signals.ReviewNeeded, Threshold: ptrFloat(opts.ReviewNeededThreshold),
		})
	}
	if !signals.EvidenceComplete {
		resolution.Action = strongerAction(resolution.Action, moderationDecision.ActionReview)
	}
	return resolution
}

func ptrFloat(value float64) *float64 { return &value }

func actionRank(action string) int {
	switch action {
	case moderationDecision.ActionBlock:
		return 2
	case moderationDecision.ActionReview:
		return 1
	default:
		return 0
	}
}

func strongerAction(a, b string) string {
	if actionRank(b) > actionRank(a) {
		return b
	}
	return a
}

// validProbability 校验 Jev 概率落在 [0,1]（NaN/Inf/越界一律视为 schema 无效）。
func validProbability(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= 0 && value <= 1
}

// AIPolicyHash 当前生效政策（规则文案/动作/阈值）的稳定哈希，随决策落库，
// 回放与评估据此分组，政策变化无需人工维护版本号也可区分样本。
func AIPolicyHash(opts pageConfig.AiModerationOptions) string {
	payload, _ := json.Marshal(struct {
		Revision      string
		Policies      []pageConfig.AiModerationPolicyRule
		Review, Block float64
		ReviewNeeded  float64
		Severity      float64
	}{opts.PolicyRevision, opts.Policies, opts.DefaultReviewThreshold, opts.DefaultBlockThreshold, opts.ReviewNeededThreshold, opts.SeverityBlockThreshold})
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

// AIReplaySampleLimit 离线回放最多读取的已标注样本数。
const AIReplaySampleLimit = 2000

// AIReplayReport 阈值回放结果。Matrix[human][predicted] 计数；FalseBlock 为
// 人工通过但预测 block（误杀），MissedViolation 为人工拒绝但预测 allow（漏检）。
type AIReplayReport struct {
	Samples         int                       `json:"samples"`
	Matrix          map[string]map[string]int `json:"matrix"`
	FalseBlock      int                       `json:"falseBlock"`
	MissedViolation int                       `json:"missedViolation"`
	ReviewRate      float64                   `json:"reviewRate"`
	Changed         int                       `json:"changed"` // 预测动作与当时记录动作不同的样本数
}

// ReplayAIDecisions 用给定配置对已标注决策重放 resolver。证据不完整的样本
// 保持当时的动作（其结论不依赖阈值）。
func ReplayAIDecisions(opts pageConfig.AiModerationOptions, labeled []moderationDecision.Entity) AIReplayReport {
	report := AIReplayReport{Matrix: map[string]map[string]int{
		moderationDecision.HumanApproved: {moderationDecision.ActionAllow: 0, moderationDecision.ActionReview: 0, moderationDecision.ActionBlock: 0},
		moderationDecision.HumanRejected: {moderationDecision.ActionAllow: 0, moderationDecision.ActionReview: 0, moderationDecision.ActionBlock: 0},
	}}
	reviews := 0
	for _, entity := range labeled {
		row, ok := report.Matrix[entity.HumanAction]
		if !ok {
			continue
		}
		predicted := entity.FinalAction
		if entity.EvidenceStatus == moderationDecision.EvidenceComplete && entity.Signals.RuleProbabilities != nil {
			predicted = ResolveAISignals(opts, AISignals{
				RuleProbabilities: entity.Signals.RuleProbabilities,
				Severity:          entity.Signals.Severity,
				ReviewNeeded:      entity.Signals.ReviewNeeded,
				EvidenceComplete:  true,
			}).Action
		}
		report.Samples++
		row[predicted]++
		if predicted != entity.FinalAction {
			report.Changed++
		}
		switch {
		case predicted == moderationDecision.ActionReview:
			reviews++
		case predicted == moderationDecision.ActionBlock && entity.HumanAction == moderationDecision.HumanApproved:
			report.FalseBlock++
		case predicted == moderationDecision.ActionAllow && entity.HumanAction == moderationDecision.HumanRejected:
			report.MissedViolation++
		}
	}
	if report.Samples > 0 {
		report.ReviewRate = float64(reviews) / float64(report.Samples)
	}
	return report
}
