package chatservice

import (
	"errors"
	"fmt"
	"testing"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/chat/imConversations"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/chat/imUserChatConfigs"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/chat/messages"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userFollow"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
)

func TestMessagePolicyGatesNewConversations(t *testing.T) {
	setupMarkReadTestDB(t)
	conn := db.Connect()
	if err := conn.AutoMigrate(&users.PrivacySettingsEntity{}, &userFollow.Entity{}); err != nil {
		t.Fatal(err)
	}
	for i, tc := range []struct {
		name, policy                          string
		follows                               int
		empty, history, blocked, bot, allowed bool
	}{
		{name: "default permits", policy: "everyone", allowed: true},
		{name: "no new messages", policy: "nobody"},
		{name: "mutual rejects strangers", policy: "mutual"},
		{name: "one way is insufficient", policy: "mutual", follows: 1},
		{name: "mutual permits", policy: "mutual", follows: 2, allowed: true},
		{name: "empty conversation is insufficient", policy: "nobody", empty: true},
		{name: "real history continues", policy: "nobody", history: true, allowed: true},
		{name: "block overrides history", policy: "nobody", history: true, blocked: true},
		{name: "bot has no exemption", policy: "nobody", bot: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sender, peer := uint64(9843100+i*10), uint64(9843101+i*10)
			for _, id := range []uint64{sender, peer} {
				u := users.MakeUser(fmt.Sprintf("policy%d", id), "password123", fmt.Sprintf("p%d@example.com", id))
				u.Id = id
				if id == sender && tc.bot {
					u.ActorType = users.ActorTypeBot
				}
				if err := conn.Create(u).Error; err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { conn.Unscoped().Delete(u) })
			}
			if _, err := users.UpdatePrivacySettings(peer, users.PrivacySettingsUpdate{ChatMessagePolicy: &tc.policy}); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				conn.Where("user_id = ?", peer).Delete(&users.PrivacySettingsEntity{})
				conn.Where("user_id IN ?", []uint64{sender, peer}).Delete(&userFollow.Entity{})
				conn.Where("owner_id = ?", peer).Delete(&users.BlockEntity{})
				conn.Where("sender_id IN ?", []uint64{sender, peer}).Delete(&messages.Entity{})
				var configs []imUserChatConfigs.Entity
				conn.Where("user_id = ?", sender).Find(&configs)
				for _, cfg := range configs {
					conn.Delete(&imConversations.Entity{}, cfg.ConvId)
				}
				conn.Where("user_id IN ?", []uint64{sender, peer}).Delete(&imUserChatConfigs.Entity{})
			})
			for n := 0; n < tc.follows; n++ {
				a, b := sender, peer
				if n == 1 {
					a, b = peer, sender
				}
				if err := conn.Create(&userFollow.Entity{UserId: a, FollowUserId: b, Status: 1}).Error; err != nil {
					t.Fatal(err)
				}
			}
			if tc.empty || tc.history {
				conv := imConversations.Entity{}
				if err := conn.Create(&conv).Error; err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { conn.Delete(&conv) })
				if err := conn.Create(&[]imUserChatConfigs.Entity{{UserId: sender, PeerId: peer, ConvId: conv.Id}, {UserId: peer, PeerId: sender, ConvId: conv.Id}}).Error; err != nil {
					t.Fatal(err)
				}
				if tc.history {
					if err := conn.Create(&messages.Entity{ConvId: conv.Id, SenderId: peer, Content: "previous", MsgType: 1}).Error; err != nil {
						t.Fatal(err)
					}
				}
			}
			if tc.blocked {
				if err := users.SetBlockedUser(peer, sender, true); err != nil {
					t.Fatal(err)
				}
			}
			_, err := sendMessageWithEffects(conn, sender, peer, "new", 1, fmt.Sprintf("policy-%d", i))
			if !tc.allowed && !errors.Is(err, ErrMessagePermissionDenied) && !errors.Is(err, users.ErrInteractionBlocked) {
				t.Fatalf("wrong error: %v", err)
			}
			if (err == nil) != tc.allowed {
				t.Fatalf("send allowed=%t want=%t err=%v", err == nil, tc.allowed, err)
			}
			if tc.allowed {
				if err := users.SetBlockedUser(peer, sender, true); err != nil {
					t.Fatal(err)
				}
				// A committed retry only acknowledges the earlier write, even after a block.
				if _, err := sendMessageWithEffects(conn, sender, peer, "new", 1, fmt.Sprintf("policy-%d", i)); err != nil {
					t.Fatalf("committed retry: %v", err)
				}
				if _, err := sendMessageWithEffects(conn, sender, peer, "changed", 1, fmt.Sprintf("policy-%d", i)); err == nil {
					t.Fatal("changed retry accepted")
				}
			}
			var count int64
			conn.Model(&messages.Entity{}).Where("sender_id = ?", sender).Count(&count)
			if !tc.allowed && count != 0 {
				t.Fatal("denied send committed a message")
			}
		})
	}
}
