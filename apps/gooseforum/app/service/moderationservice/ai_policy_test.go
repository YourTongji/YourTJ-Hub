package moderationservice

import (
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/moderationDecision"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
)

func ptr(value float64) *float64 { return &value }

// aiTestOptions 默认四条规则启用、动作 review；adult 改为 block 用于升级用例。
func aiTestOptions() pageConfig.AiModerationOptions {
	opts := pageConfig.AiModerationOptions{}.Normalize()
	for i := range opts.Policies {
		if opts.Policies[i].Key == pageConfig.AiPolicyAdult {
			opts.Policies[i].Action = pageConfig.AiModerationActionBlock
		}
	}
	return opts
}

func cleanSignals(overrides map[string]float64) AISignals {
	probabilities := map[string]float64{
		pageConfig.AiPolicyAdult: 0.01, pageConfig.AiPolicyPoliticalSensitive: 0.01,
		pageConfig.AiPolicyViolence: 0.01, pageConfig.AiPolicyIllegalOrDangerous: 0.01,
	}
	for key, value := range overrides {
		probabilities[key] = value
	}
	return AISignals{RuleProbabilities: probabilities, Severity: ptr(0.1), ReviewNeeded: ptr(0.05), EvidenceComplete: true}
}

func TestResolveAISignals(t *testing.T) {
	opts := aiTestOptions()
	cases := []struct {
		name      string
		signals   AISignals
		action    string
		triggered []string
	}{
		{"clean content allows", cleanSignals(nil), moderationDecision.ActionAllow, nil},
		{"block rule above block threshold blocks", cleanSignals(map[string]float64{pageConfig.AiPolicyAdult: 0.97}), moderationDecision.ActionBlock, []string{pageConfig.AiPolicyAdult}},
		{"block rule between thresholds reviews", cleanSignals(map[string]float64{pageConfig.AiPolicyAdult: 0.6}), moderationDecision.ActionReview, []string{pageConfig.AiPolicyAdult}},
		{"review-capped rule never auto-blocks", cleanSignals(map[string]float64{pageConfig.AiPolicyPoliticalSensitive: 0.999}), moderationDecision.ActionReview, []string{pageConfig.AiPolicyPoliticalSensitive}},
		{"multiple violations keep every triggered rule", cleanSignals(map[string]float64{pageConfig.AiPolicyViolence: 0.8, pageConfig.AiPolicyIllegalOrDangerous: 0.7}), moderationDecision.ActionReview, []string{pageConfig.AiPolicyViolence, pageConfig.AiPolicyIllegalOrDangerous}},
		{"review_needed alone reviews", func() AISignals { s := cleanSignals(nil); s.ReviewNeeded = ptr(0.9); return s }(), moderationDecision.ActionReview, nil},
		{"incomplete evidence never allows", func() AISignals { s := cleanSignals(nil); s.EvidenceComplete = false; return s }(), moderationDecision.ActionReview, nil},
		{"incomplete evidence still allows a strong block", func() AISignals {
			s := cleanSignals(map[string]float64{pageConfig.AiPolicyAdult: 0.95})
			s.EvidenceComplete = false
			return s
		}(), moderationDecision.ActionBlock, []string{pageConfig.AiPolicyAdult}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ResolveAISignals(opts, tc.signals)
			if got.Action != tc.action {
				t.Fatalf("action = %s, want %s", got.Action, tc.action)
			}
			if len(got.Triggered) != len(tc.triggered) {
				t.Fatalf("triggered = %v, want %v", got.Triggered, tc.triggered)
			}
			for i := range tc.triggered {
				if got.Triggered[i] != tc.triggered[i] {
					t.Fatalf("triggered = %v, want %v", got.Triggered, tc.triggered)
				}
			}
		})
	}
}

// severity 只升级已越过 review 阈值的 block 规则，不能单独把普通内容 block。
func TestResolveAISignalsSeverityOnlyEscalatesTriggeredBlockRules(t *testing.T) {
	opts := aiTestOptions()
	quiet := cleanSignals(nil)
	quiet.Severity = ptr(3)
	if got := ResolveAISignals(opts, quiet); got.Action != moderationDecision.ActionAllow {
		t.Fatalf("severity alone produced %s, want allow", got.Action)
	}
	borderline := cleanSignals(map[string]float64{pageConfig.AiPolicyAdult: 0.6})
	borderline.Severity = ptr(2.8)
	if got := ResolveAISignals(opts, borderline); got.Action != moderationDecision.ActionBlock {
		t.Fatalf("severe borderline block rule produced %s, want block", got.Action)
	}
	reviewCapped := cleanSignals(map[string]float64{pageConfig.AiPolicyViolence: 0.6})
	reviewCapped.Severity = ptr(3)
	if got := ResolveAISignals(opts, reviewCapped); got.Action != moderationDecision.ActionReview {
		t.Fatalf("severe review-capped rule produced %s, want review", got.Action)
	}
}

func TestResolveAISignalsHonoursRuleOverridesAndDisabledRules(t *testing.T) {
	opts := aiTestOptions()
	for i := range opts.Policies {
		switch opts.Policies[i].Key {
		case pageConfig.AiPolicyViolence:
			opts.Policies[i].ReviewThreshold = ptr(0.95)
		case pageConfig.AiPolicyIllegalOrDangerous:
			opts.Policies[i].Enabled = false
		}
	}
	signals := cleanSignals(map[string]float64{pageConfig.AiPolicyViolence: 0.9, pageConfig.AiPolicyIllegalOrDangerous: 0.99})
	if got := ResolveAISignals(opts, signals); got.Action != moderationDecision.ActionAllow {
		t.Fatalf("override/disabled rules produced %s %v, want allow", got.Action, got.Triggered)
	}
}

func TestAIModerationOptionsNormalize(t *testing.T) {
	opts := pageConfig.AiModerationOptions{
		Mode: "bogus", DefaultReviewThreshold: 0.8, DefaultBlockThreshold: 0.3, JevRetries: 5,
		ExternalImageAction: "allow", MaxImagesPerDecision: 500,
		Policies: []pageConfig.AiModerationPolicyRule{{Key: "unknown", Enabled: true}, {Key: pageConfig.AiPolicyAdult, Action: "allow", Definition: "  "}},
	}.Normalize()
	if opts.Mode != pageConfig.AiModerationModeShadow || opts.JevRetries != 1 || opts.ExternalImageAction != pageConfig.AiModerationActionReview {
		t.Fatalf("enum/limit normalization failed: %+v", opts)
	}
	if opts.DefaultBlockThreshold < opts.DefaultReviewThreshold {
		t.Fatalf("block threshold %v below review threshold %v", opts.DefaultBlockThreshold, opts.DefaultReviewThreshold)
	}
	if opts.MaxImagesPerDecision != 20 || len(opts.Policies) != len(pageConfig.AiModerationPolicyKeys) {
		t.Fatalf("unexpected normalization: images=%d policies=%d", opts.MaxImagesPerDecision, len(opts.Policies))
	}
	adult := opts.Policies[0]
	if adult.Key != pageConfig.AiPolicyAdult || adult.Action != pageConfig.AiModerationActionReview || adult.Definition == "" {
		t.Fatalf("adult rule not normalized: %+v", adult)
	}
}

// labeledFixtures 人工标注样本（每类政策覆盖正常/边界/明显违规三档），用于
// shadow 校准演示：同一批原始概率在不同阈值下输出不同混淆矩阵。
func labeledFixtures() []moderationDecision.Entity {
	sample := func(human string, final string, probabilities map[string]float64, severity float64) moderationDecision.Entity {
		signals := cleanSignals(probabilities)
		signals.Severity = ptr(severity)
		return moderationDecision.Entity{HumanAction: human, FinalAction: final, EvidenceStatus: moderationDecision.EvidenceComplete,
			Signals: moderationDecision.Signals{RuleProbabilities: signals.RuleProbabilities, Severity: signals.Severity, ReviewNeeded: signals.ReviewNeeded}}
	}
	return []moderationDecision.Entity{
		sample(moderationDecision.HumanApproved, moderationDecision.ActionAllow, nil, 0.1),                                                 // normal
		sample(moderationDecision.HumanApproved, moderationDecision.ActionReview, map[string]float64{pageConfig.AiPolicyAdult: 0.55}, 1.0), // borderline art
		sample(moderationDecision.HumanRejected, moderationDecision.ActionBlock, map[string]float64{pageConfig.AiPolicyAdult: 0.98}, 2.9),  // explicit
		sample(moderationDecision.HumanRejected, moderationDecision.ActionAllow, map[string]float64{pageConfig.AiPolicyViolence: 0.45}, 2.0),
		sample(moderationDecision.HumanApproved, moderationDecision.ActionReview, map[string]float64{pageConfig.AiPolicyPoliticalSensitive: 0.6}, 1.0),
		{HumanAction: moderationDecision.HumanApproved, FinalAction: moderationDecision.ActionReview, EvidenceStatus: moderationDecision.EvidenceUnavailable},
		{HumanAction: "", FinalAction: moderationDecision.ActionAllow, EvidenceStatus: moderationDecision.EvidenceComplete}, // unlabeled → ignored
	}
}

func TestReplayAIDecisionsConfusionMatrix(t *testing.T) {
	report := ReplayAIDecisions(aiTestOptions(), labeledFixtures())
	if report.Samples != 6 {
		t.Fatalf("samples = %d, want 6", report.Samples)
	}
	approved, rejected := report.Matrix[moderationDecision.HumanApproved], report.Matrix[moderationDecision.HumanRejected]
	if approved[moderationDecision.ActionAllow] != 1 || approved[moderationDecision.ActionReview] != 3 || approved[moderationDecision.ActionBlock] != 0 {
		t.Fatalf("approved row = %v", approved)
	}
	if rejected[moderationDecision.ActionBlock] != 1 || rejected[moderationDecision.ActionAllow] != 1 {
		t.Fatalf("rejected row = %v", rejected)
	}
	if report.MissedViolation != 1 || report.FalseBlock != 0 {
		t.Fatalf("missed=%d falseBlock=%d", report.MissedViolation, report.FalseBlock)
	}

	// 候选阈值：把 review 阈值降到 0.4 后漏检样本转入 review，回放无需重新调用模型。
	candidate := aiTestOptions()
	candidate.DefaultReviewThreshold = 0.4
	tighter := ReplayAIDecisions(candidate, labeledFixtures())
	if tighter.MissedViolation != 0 || tighter.Changed == 0 {
		t.Fatalf("candidate replay missed=%d changed=%d", tighter.MissedViolation, tighter.Changed)
	}
}

// 回归（issue #975 预览反馈）：规则设为“拦截”时，分数介于转人工线与拦截线
// 之间仍转人工，原因须明确写出两条分数线，避免管理员误以为配置未生效。
func TestResolveAISignalsExplainsEveryOutcome(t *testing.T) {
	opts := aiTestOptions()
	for i := range opts.Policies {
		opts.Policies[i].Action = pageConfig.AiModerationActionBlock
	}
	opts.Policies[2].Action = pageConfig.AiModerationActionReview // violence: review only
	cases := []struct {
		name    string
		signals AISignals
		action  string
		code    string
	}{
		{"between lines goes to review", cleanSignals(map[string]float64{pageConfig.AiPolicyPoliticalSensitive: 0.56}), moderationDecision.ActionReview, moderationDecision.ReasonBetweenThresholds},
		{"block line blocks", cleanSignals(map[string]float64{pageConfig.AiPolicyAdult: 0.9}), moderationDecision.ActionBlock, moderationDecision.ReasonBlockThreshold},
		{"severity escalates", func() AISignals {
			s := cleanSignals(map[string]float64{pageConfig.AiPolicyAdult: 0.6})
			s.Severity = ptr(2.8)
			return s
		}(), moderationDecision.ActionBlock, moderationDecision.ReasonSeverityEscalation},
		{"review-only rule", cleanSignals(map[string]float64{pageConfig.AiPolicyViolence: 0.99}), moderationDecision.ActionReview, moderationDecision.ReasonReviewOnlyRule},
		{"needs human review", func() AISignals { s := cleanSignals(nil); s.ReviewNeeded = ptr(0.8); return s }(), moderationDecision.ActionReview, moderationDecision.ReasonReviewNeeded},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ResolveAISignals(opts, tc.signals)
			if got.Action != tc.action || len(got.Reasons) != 1 || got.Reasons[0].Code != tc.code {
				t.Fatalf("action=%s reasons=%+v", got.Action, got.Reasons)
			}
		})
	}
	between := ResolveAISignals(opts, cleanSignals(map[string]float64{pageConfig.AiPolicyPoliticalSensitive: 0.56})).Reasons[0]
	if *between.Score != 0.56 || *between.Threshold != 0.5 || *between.Limit != 0.9 || between.Policy != pageConfig.AiPolicyPoliticalSensitive {
		t.Fatalf("between reason = %+v", between)
	}
	if got := ResolveAISignals(opts, cleanSignals(nil)); got.Action != moderationDecision.ActionAllow || len(got.Reasons) != 0 {
		t.Fatalf("clean content = %+v", got)
	}
}
