package notificationservice

import (
	"testing"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/eventNotification"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/realtimeservice"
)

func TestMentionCommitPublishesOnlyToRecipients(t *testing.T) {
	conn := db.Connect()
	if err := conn.AutoMigrate(&eventNotification.Entity{}, &users.BlockEntity{}, &users.EntityComplete{}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = conn.Where("user_id = ?", 7651).Delete(&eventNotification.Entity{}).Error
	})
	owner, err := realtimeservice.DefaultHub.Subscribe(7651)
	if err != nil {
		t.Fatal(err)
	}
	defer owner.Close()
	other, err := realtimeservice.DefaultHub.Subscribe(7652)
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	if err := SendMentionNotifications([]uint64{7651}, 91, 92, 1, "preview", 3); err != nil {
		t.Fatal(err)
	}
	event := <-owner.Events()
	if event.Type != realtimeservice.EventNotificationsChanged || event.Change != "created" {
		t.Fatalf("event=%+v", event)
	}
	select {
	case event := <-other.Events():
		t.Fatalf("leaked event=%+v", event)
	default:
	}
}

func TestCommentNotificationsUseTopicPostPayload(t *testing.T) {
	conn := db.Connect()
	if err := conn.AutoMigrate(&eventNotification.Entity{}, &users.BlockEntity{}, &users.EntityComplete{}); err != nil {
		t.Fatalf("migrate notifications: %v", err)
	}

	if err := SendCommentNotification(1, 10, "hello", 2, 99, 7); err != nil {
		t.Fatalf("SendCommentNotification() err=%v", err)
	}

	var notification eventNotification.Entity
	if err := conn.First(&notification).Error; err != nil {
		t.Fatalf("load notification: %v", err)
	}
	if notification.Payload.TopicId != 10 || notification.Payload.PostId != 99 || notification.Payload.PostNo != 7 {
		t.Fatalf("payload topic/post/no = %d/%d/%d, want 10/99/7", notification.Payload.TopicId, notification.Payload.PostId, notification.Payload.PostNo)
	}
}

func TestLikeNotificationsUseTopicPostPayload(t *testing.T) {
	conn := db.Connect()
	if err := conn.AutoMigrate(&eventNotification.Entity{}, &users.BlockEntity{}, &users.EntityComplete{}); err != nil {
		t.Fatalf("migrate notifications: %v", err)
	}

	if err := SendLikeNotification(1, 10, "话题", 99, 7, 2); err != nil {
		t.Fatalf("SendLikeNotification() err=%v", err)
	}

	var notification eventNotification.Entity
	if err := conn.First(&notification).Error; err != nil {
		t.Fatalf("load notification: %v", err)
	}
	if notification.Payload.TopicId != 10 || notification.Payload.PostId != 99 || notification.Payload.PostNo != 7 {
		t.Fatalf("payload topic/post/no = %d/%d/%d, want 10/99/7", notification.Payload.TopicId, notification.Payload.PostId, notification.Payload.PostNo)
	}
}

func TestMentionNotificationsUseTopicPostPayload(t *testing.T) {
	conn := db.Connect()
	if err := conn.AutoMigrate(&eventNotification.Entity{}, &users.BlockEntity{}, &users.EntityComplete{}); err != nil {
		t.Fatalf("migrate notifications: %v", err)
	}

	if err := SendMentionNotifications([]uint64{1, 0}, 10, 99, 7, "提到你", 2); err != nil {
		t.Fatalf("SendMentionNotifications() err=%v", err)
	}

	var notifications []eventNotification.Entity
	if err := conn.Where("event_type = ?", eventNotification.EventTypeMention).Find(&notifications).Error; err != nil {
		t.Fatalf("load notifications: %v", err)
	}
	if len(notifications) != 1 {
		t.Fatalf("created %d mention notifications, want 1 (zero id skipped)", len(notifications))
	}
	notification := notifications[0]
	if notification.EventType != eventNotification.EventTypeMention {
		t.Fatalf("eventType = %q, want %q", notification.EventType, eventNotification.EventTypeMention)
	}
	if notification.Payload.TemplateKey != eventNotification.TemplateMention {
		t.Fatalf("templateKey = %q, want %q", notification.Payload.TemplateKey, eventNotification.TemplateMention)
	}
	if notification.Payload.TopicId != 10 || notification.Payload.PostId != 99 || notification.Payload.PostNo != 7 {
		t.Fatalf("payload topic/post/no = %d/%d/%d, want 10/99/7", notification.Payload.TopicId, notification.Payload.PostId, notification.Payload.PostNo)
	}
	if notification.Payload.ActorId != 2 {
		t.Fatalf("actorId = %d, want 2", notification.Payload.ActorId)
	}
}

func TestBlockedInteractionsDoNotCreateNotifications(t *testing.T) {
	conn := db.Connect()
	if err := conn.AutoMigrate(&eventNotification.Entity{}, &users.BlockEntity{}, &users.EntityComplete{}); err != nil {
		t.Fatal(err)
	}
	const actor, blocked, allowed = uint64(88211), uint64(88212), uint64(88213)
	if err := conn.Create(&users.BlockEntity{OwnerID: blocked, TargetUserID: actor}).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		conn.Where("owner_id = ?", blocked).Delete(&users.BlockEntity{})
		conn.Where("user_id IN ?", []uint64{blocked, allowed, actor}).Delete(&eventNotification.Entity{})
	})
	for _, send := range []func() error{
		func() error { return SendFollowNotification(blocked, actor, "actor") },
		func() error { return SendLikeNotification(actor, 1, "topic", 2, 1, blocked) },
		func() error { return SendCommentNotification(blocked, 1, "body", actor, 2, 1) },
		func() error { return SendPostReplyNotification(blocked, 2, 1, 1, "body", actor) },
		func() error { return SendMentionNotifications([]uint64{blocked, allowed}, 1, 2, 1, "body", actor) },
		func() error { return SendTopicPostNotifications([]uint64{blocked, allowed}, 1, 2, 1, "body", actor) },
	} {
		if err := send(); err != nil {
			t.Fatal(err)
		}
	}
	var count int64
	if err := conn.Model(&eventNotification.Entity{}).Where("user_id IN ?", []uint64{blocked, actor}).Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("blocked notifications=%d, %v", count, err)
	}
	if err := conn.Model(&eventNotification.Entity{}).Where("user_id = ?", allowed).Count(&count).Error; err != nil || count != 2 {
		t.Fatalf("allowed notifications=%d, %v", count, err)
	}
}
