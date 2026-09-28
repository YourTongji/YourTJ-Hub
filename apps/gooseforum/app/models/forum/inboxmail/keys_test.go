package inboxmail

import "testing"

// 幂等 key 的字符串格式是所有投递/领取写入方的公共契约（#774 投递去重、
// #775 trigger 触发、#778 claim 双幂等），格式漂移会让历史行与新行去重失败。
func TestCampaignDedupeKeyFormat(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name       string
		campaignID uint64
		versionNo  int
		userID     uint64
		want       string
	}{
		{name: "typical", campaignID: 12, versionNo: 3, userID: 456, want: "campaign:12:3:user:456"},
		{name: "single digit", campaignID: 1, versionNo: 1, userID: 1, want: "campaign:1:1:user:1"},
		{name: "large ids stay decimal", campaignID: 18446744073709551615, versionNo: 2, userID: 18446744073709551615, want: "campaign:18446744073709551615:2:user:18446744073709551615"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := CampaignDedupeKey(tc.campaignID, tc.versionNo, tc.userID); got != tc.want {
				t.Fatalf("CampaignDedupeKey() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestTriggerDedupeKeyFormat(t *testing.T) {
	t.Parallel()
	if got := TriggerDedupeKey("user.registered", "evt-9", 42); got != "trigger:user.registered:evt-9:user:42" {
		t.Fatalf("TriggerDedupeKey() = %q", got)
	}
}

func TestClaimSourceKeyFormat(t *testing.T) {
	t.Parallel()
	// 与 epic #769 给出的示例一致：inbox_claim:badge:<badge_code>:delivery:<delivery_id>。
	if got := ClaimSourceKey("badge", "welcome_2026", 7); got != "inbox_claim:badge:welcome_2026:delivery:7" {
		t.Fatalf("ClaimSourceKey() = %q", got)
	}
}
