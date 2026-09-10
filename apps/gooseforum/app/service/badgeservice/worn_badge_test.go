package badgeservice

import (
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userBadges"
)

func TestWornBadgeFromListReturnsGrantedBadge(t *testing.T) {
	items := []UserBadge{
		{Badge: Badge{Code: "plain", IsEnabled: true, IsWearable: false}},
		{Badge: Badge{Code: "wearable", IsEnabled: true, IsWearable: true}},
	}

	// 已获得（列表中出现）的徽章即可佩戴，不再受 IsWearable 定义拦截
	if got := WornBadgeFromList(items, "plain"); got == nil || got.Code != "plain" {
		t.Fatalf("WornBadgeFromList() = %#v, want granted badge plain", got)
	}
	if got := WornBadgeFromList(items, "wearable"); got == nil || got.Code != "wearable" {
		t.Fatalf("WornBadgeFromList() = %#v, want wearable badge", got)
	}
	if got := WornBadgeFromList(items, "missing"); got != nil {
		t.Fatalf("WornBadgeFromList() = %#v, want nil for non-granted badge", got)
	}
}

func TestWornBadgesFromRecordsMatchesSelectedActiveBadge(t *testing.T) {
	selected := map[uint64]string{1: "wearable", 2: "plain", 3: "wearable"}
	records := []*userBadges.Entity{
		{UserId: 1, BadgeCode: "other"},
		{UserId: 1, BadgeCode: "wearable"},
		{UserId: 2, BadgeCode: "plain"},
	}
	definitions := map[string]Badge{
		"wearable": {Code: "wearable", IsEnabled: true, IsWearable: true},
		"plain":    {Code: "plain", IsEnabled: true, IsWearable: false},
		"other":    {Code: "other", IsEnabled: true, IsWearable: true},
	}

	got := wornBadgesFromRecords(selected, records, definitions)
	if len(got) != 2 {
		t.Fatalf("wornBadgesFromRecords() = %#v, want user 1 and user 2 granted badges", got)
	}
	if got[1] == nil || got[1].Code != "wearable" {
		t.Fatalf("wornBadgesFromRecords() user 1 = %#v, want wearable", got[1])
	}
	if got[2] == nil || got[2].Code != "plain" {
		t.Fatalf("wornBadgesFromRecords() user 2 = %#v, want granted plain", got[2])
	}
}
