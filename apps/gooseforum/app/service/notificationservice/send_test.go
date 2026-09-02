package notificationservice

import (
	"testing"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/eventNotification"
)

func TestCommentNotificationsUseTopicPostPayload(t *testing.T) {
	conn := db.Connect()
	if err := conn.AutoMigrate(&eventNotification.Entity{}); err != nil {
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
	if err := conn.AutoMigrate(&eventNotification.Entity{}); err != nil {
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
