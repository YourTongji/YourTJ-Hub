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
}
