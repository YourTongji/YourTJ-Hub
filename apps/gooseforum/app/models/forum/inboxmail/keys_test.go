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
	if got := TriggerDedupeKey(12, "user.registered", "evt-9", 42); got != "trigger:12:user.registered:evt-9:user:42" {
		t.Fatalf("TriggerDedupeKey() = %q", got)
	}
}

// 同一事件 occurrence 被两个 Campaign 触发时必须得到不同 key（#775）：dedupe_key
// 是全局唯一索引，缺 campaign 段会让第二个 Campaign 的信被当作重放静默丢弃。
func TestTriggerDedupeKeyIsCampaignScoped(t *testing.T) {
	t.Parallel()
	first := TriggerDedupeKey(7, "user.registered", "evt-9", 42)
	second := TriggerDedupeKey(8, "user.registered", "evt-9", 42)
	if first == second {
		t.Fatalf("two campaigns produced identical trigger keys: %q", first)
	}
	// 重放同一 occurrence 必须稳定复用同一 key（这正是幂等的来源）。
	if replay := TriggerDedupeKey(7, "user.registered", "evt-9", 42); replay != first {
		t.Fatalf("replayed occurrence key drift: %q != %q", replay, first)
	}
	// 同一 Campaign 的不同 occurrence 各自成信。
	if other := TriggerDedupeKey(7, "user.registered", "evt-10", 42); other == first {
		t.Fatalf("distinct occurrences share a trigger key: %q", other)
	}
	// 同一 Campaign 的不同收件人各自成信。
	if other := TriggerDedupeKey(7, "user.registered", "evt-9", 43); other == first {
		t.Fatalf("distinct users share a trigger key: %q", other)
	}
}

func TestClaimSourceKeyFormat(t *testing.T) {
	t.Parallel()
	// 粒度与 #777 的统一规则一致：每个 (delivery, attachment) 一行领取事实。
	if got := ClaimSourceKey(7, 3); got != "inbox:7:3" {
		t.Fatalf("ClaimSourceKey() = %q", got)
	}
}

// 同一 delivery 的两个附件必须得到不同 source_key：附件唯一性只在
// (campaign_id, attachment_key) 粒度上成立（#777 发布校验只查徽章是否启用），同一
// handler key 出现在两个附件行是合法的，不能因为 key 更粗而互相顶掉领取行。
func TestClaimSourceKeyIsPerAttachment(t *testing.T) {
	t.Parallel()
	first := ClaimSourceKey(7, 1)
	second := ClaimSourceKey(7, 2)
	if first == second {
		t.Fatalf("two attachments of one delivery produced identical source keys: %q", first)
	}
	if other := ClaimSourceKey(8, 1); other == first {
		t.Fatalf("two deliveries produced identical source keys: %q", other)
	}
}
