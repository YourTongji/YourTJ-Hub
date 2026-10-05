package forum

import (
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/eventNotification"
)

func TestBuildNotificationPayloadUsesPostNumberURL(t *testing.T) {
	notification := &eventNotification.Entity{
		Payload: eventNotification.NotificationPayload{TopicId: 42, PostId: 99, PostNo: 7},
	}

	item := BuildNotificationPayload(notification)
	if item.Topic == nil || item.Topic.URL != "/p/post/42/7" {
		t.Fatalf("notification topic = %#v, want post number URL", item.Topic)
	}
}

func TestBuildNotificationPayloadFallsBackToPostAnchor(t *testing.T) {
	notification := &eventNotification.Entity{
		Payload: eventNotification.NotificationPayload{TopicId: 42, PostId: 99},
	}
	item := BuildNotificationPayload(notification)
	if item.Topic == nil || item.Topic.URL != "/p/post/42#post-99" {
		t.Fatalf("notification topic = %#v, want legacy post anchor URL", item.Topic)
	}
}

func TestBuildNotificationPayloadWikiUpdatedKeepsProfileURL(t *testing.T) {
	notification := &eventNotification.Entity{
		EventType: eventNotification.EventTypeWikiUpdated,
		Payload: eventNotification.NotificationPayload{
			TopicId: 42,
			PostId:  99,
			Extra:   eventNotification.Extra{ProfileURL: "/docs/project/page"},
		},
	}
	item := BuildNotificationPayload(notification)
	if item.Topic == nil || item.Topic.URL != "/docs/project/page" {
		t.Fatalf("notification topic = %#v, want wiki profile URL", item.Topic)
	}
}

func TestRejectedNotificationRedactsHistoricalOriginalsAndLinksRecovery(t *testing.T) {
	item := BuildNotificationPayload(&eventNotification.Entity{EventType: eventNotification.EventTypeReviewRejected, Payload: eventNotification.NotificationPayload{Title: "旧的敏感标题", TopicTitle: "旧的敏感标题", Content: "被拒原文", TemplateParams: eventNotification.NotificationTemplateParams{Preview: "被拒原文"}, TopicId: 42, PostNo: 1}})
	if item.Title != "" || item.Content != "" || item.Payload.Title != "" || item.Payload.Content != "" || item.Payload.TemplateParams.Preview != "" || item.Topic == nil || item.Topic.Title != "旧******题" || item.Topic.URL != "/settings?tab=content" {
		t.Fatalf("unsafe rejected notification: %+v", item)
	}
}
