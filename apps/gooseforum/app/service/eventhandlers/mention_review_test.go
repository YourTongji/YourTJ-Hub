package eventhandlers

import (
	"context"
	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/eventNotification"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"testing"
)

func TestMentionReviewEligibility(t *testing.T) {
	conn := db.Connect()
	if err := conn.AutoMigrate(&users.EntityComplete{}, &topics.Entity{}, &posts.Entity{}, &eventNotification.Entity{}); err != nil {
		t.Fatal(err)
	}
	target := users.MakeUser("mention_review", "pass1234", "mention_review@example.com")
	if err := users.Create(target); err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"frozen", "bot"} {
		t.Run(kind, func(t *testing.T) {
			updates := map[string]any{"is_frozen": users.StatusNormal, "actor_type": users.ActorTypeHuman}
			if kind == "frozen" {
				updates["is_frozen"] = users.StatusFrozen
			} else {
				updates["actor_type"] = users.ActorTypeBot
			}
			if err := conn.Model(&users.EntityComplete{}).Where("id = ?", target.Id).Updates(updates).Error; err != nil {
				t.Fatal(err)
			}
			if got := resolveMentionUserIDs("@mention_review", 0, 20); len(got) != 0 {
				t.Fatalf("ineligible target notified: %v", got)
			}
		})
	}
	if err := conn.Model(&users.EntityComplete{}).Where("id = ?", target.Id).Updates(map[string]any{"is_frozen": 0, "actor_type": 0}).Error; err != nil {
		t.Fatal(err)
	}
	for _, status := range []int8{0, 1} {
		topic := topics.Entity{UserId: 9999, Status: status, Title: "mention review"}
		if err := conn.Create(&topic).Error; err != nil {
			t.Fatal(err)
		}
		post := posts.Entity{TopicId: topic.Id, PostNo: 1, UserId: 9999, Content: "@mention_review"}
		if err := conn.Create(&post).Error; err != nil {
			t.Fatal(err)
		}
		if err := handlePostUpdated(context.Background(), &PostUpdatedEvent{TopicId: topic.Id, PostId: post.Id, PostNo: 1, UserId: 9999, NewContent: post.Content}); err != nil {
			t.Fatal(err)
		}
		var count int64
		if err := conn.Model(&eventNotification.Entity{}).Where("user_id = ? AND event_type = ?", target.Id, eventNotification.EventTypeMention).Count(&count).Error; err != nil {
			t.Fatal(err)
		}
		if count != int64(status) {
			t.Fatalf("topic status %d: notifications=%d", status, count)
		}
	}
}

func TestPublishedTopicMentionsAndUnchangedEdits(t *testing.T) {
	conn := db.Connect()
	if err := conn.AutoMigrate(&users.EntityComplete{}, &topics.Entity{}, &posts.Entity{}, &eventNotification.Entity{}); err != nil {
		t.Fatal(err)
	}
	target := users.MakeUser("mention_first", "pass1234", "mention_first@example.com")
	if err := users.Create(target); err != nil {
		t.Fatal(err)
	}
	topic := topics.Entity{UserId: 9999, Status: 1, Title: "mention first"}
	if err := conn.Create(&topic).Error; err != nil {
		t.Fatal(err)
	}
	post := posts.Entity{TopicId: topic.Id, PostNo: 1, UserId: 9999, Content: "@mention_first"}
	if err := conn.Create(&post).Error; err != nil {
		t.Fatal(err)
	}
	if err := handleTopicMentionPublished(context.Background(), &TopicPublishedEvent{Topic: &topic, FirstPost: &post}); err != nil {
		t.Fatal(err)
	}
	if err := handlePostUpdated(context.Background(), &PostUpdatedEvent{TopicId: topic.Id, PostId: post.Id, PostNo: 1, UserId: 9999, OldContent: post.Content, NewContent: post.Content + " text"}); err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := conn.Model(&eventNotification.Entity{}).Where("user_id = ? AND event_type = ?", target.Id, eventNotification.EventTypeMention).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected one first-post mention, got %d", count)
	}
	for _, update := range []map[string]any{{"process_status": posts.ProcessStatusPending}, {"process_status": posts.ProcessStatusNormal, "visibility_status": posts.VisibilityUserDeleted}, {"visibility_status": posts.VisibilityActive, "is_anonymous": true}} {
		if err := conn.Model(&posts.Entity{}).Where("id = ?", post.Id).Updates(update).Error; err != nil {
			t.Fatal(err)
		}
		if err := handleTopicMentionPublished(context.Background(), &TopicPublishedEvent{Topic: &topic, FirstPost: &post}); err != nil {
			t.Fatal(err)
		}
		if err := conn.Model(&eventNotification.Entity{}).Where("user_id = ? AND event_type = ?", target.Id, eventNotification.EventTypeMention).Count(&count).Error; err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Fatalf("private post must not notify: %v count=%d", update, count)
		}
	}
}
