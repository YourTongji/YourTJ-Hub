package unreadservice

import (
	"encoding/json"
	"testing"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/chat/imUserChatConfigs"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/eventNotification"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
)

func TestUnreadDisplayCountsRetainExistingMembershipAndReadSemantics(t *testing.T) {
	conn := db.Connect()
	if err := conn.AutoMigrate(&users.EntityComplete{}, &eventNotification.Entity{}, &imUserChatConfigs.Entity{}); err != nil {
		t.Fatal(err)
	}
	owner := users.MakeUser("unread-display", "password123", "unread-display@example.test")
	if err := users.Create(owner); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Unscoped().Delete(owner); Invalidate(owner.Id) })
	for i := 0; i < 4; i++ {
		notification := eventNotification.Entity{UserId: owner.Id, EventType: "comment", IsRead: i == 3}
		if err := conn.Create(&notification).Error; err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { conn.Unscoped().Delete(&notification) })
	}
	for _, config := range []imUserChatConfigs.Entity{
		{UserId: owner.Id, PeerId: 11, ConvId: 11, UnreadCount: 4},
		{UserId: owner.Id, PeerId: 12, ConvId: 12, UnreadCount: 7, IsMuted: 1},
		{UserId: owner.Id, PeerId: 13, ConvId: 13, UnreadCount: 100, IsDeleted: 1},
		{UserId: owner.Id + 100, PeerId: 14, ConvId: 14, UnreadCount: 90},
	} {
		if err := conn.Create(&config).Error; err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { conn.Unscoped().Delete(&config) })
	}
	assertWire := func(style string) {
		t.Helper()
		encoded, err := json.Marshal(GetStatus(owner.Id))
		if err != nil {
			t.Fatal(err)
		}
		var wire map[string]any
		if err := json.Unmarshal(encoded, &wire); err != nil {
			t.Fatal(err)
		}
		if wire["notificationCount"] != float64(3) || wire["messageCount"] != float64(11) || wire["badgeStyle"] != style {
			t.Fatalf("unread display projection = %s", encoded)
		}
		if wire["notifications"] != true || wire["messages"] != true {
			t.Fatalf("display style changed unread facts: %s", encoded)
		}
	}
	Invalidate(owner.Id)
	assertWire(users.UnreadBadgeNumber)
	if err := users.SetUnreadBadgeStyle(owner.Id, users.UnreadBadgeDot); err != nil {
		t.Fatal(err)
	}
	Invalidate(owner.Id)
	assertWire(users.UnreadBadgeDot)
}
