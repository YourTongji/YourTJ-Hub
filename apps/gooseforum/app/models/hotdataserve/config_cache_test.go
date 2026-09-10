package hotdataserve

import (
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
)

func TestMergeDefaultRateLimitActionsAddsRewardAbuseGuards(t *testing.T) {
	cfg := pageConfig.RateLimitConfig{
		Enabled: true,
		Actions: []pageConfig.RateLimitRule{{
			Action:        "topic.write",
			WindowSeconds: 999,
			LimitPerIp:    1,
			LimitPerUser:  1,
		}},
	}
	mergeDefaultRateLimitActions(&cfg)
	cfg.BuildActionIndex()

	found := map[string]pageConfig.RateLimitRule{}
	for _, rule := range cfg.Actions {
		found[rule.Action] = rule
	}
	for _, action := range []string{"topic.status", "post.delete"} {
		if rule, ok := found[action]; !ok || rule.WindowSeconds <= 0 || rule.LimitPerIp <= 0 || rule.LimitPerUser <= 0 {
			t.Errorf("merged rule %q = %+v, want positive default limits", action, rule)
		}
	}
	if got := found["topic.write"].WindowSeconds; got != 999 {
		t.Errorf("existing topic.write window = %d, want 999", got)
	}
	if rule, ok := cfg.RuleForAction("topic.write"); !ok || rule.WindowSeconds != 999 {
		t.Fatalf("indexed topic.write rule = %+v, %v; want first configured rule", rule, ok)
	}
	if _, ok := cfg.RuleForAction("unknown"); ok {
		t.Fatal("unknown action should not be indexed")
	}
}
