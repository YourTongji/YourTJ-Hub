package notificationservice

import (
	"testing"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/eventNotification"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
)

func TestNormalizePageSize(t *testing.T) {
	tests := []struct {
		name string
		in   int
		want int
	}{
		{name: "default for zero", in: 0, want: DefaultNotificationPageSize},
		{name: "default for negative", in: -1, want: DefaultNotificationPageSize},
		{name: "keeps valid size", in: 12, want: 12},
		{name: "caps max size", in: MaxNotificationPageSize + 1, want: MaxNotificationPageSize},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizePageSize(tt.in); got != tt.want {
				t.Fatalf("normalizePageSize(%d) = %d, want %d", tt.in, got, tt.want)
			}
		})
	}
}

func TestHydrateReviewKeepsReviewedVersionSubject(t *testing.T) {
	conn := db.Connect()
	if err := conn.AutoMigrate(&topics.Entity{}, &users.EntityComplete{}); err != nil {
		t.Fatal(err)
	}
	for _, liveTitle := range []string{"", "Old public title"} {
		topic := topics.Entity{Title: liveTitle}
		if err := conn.Create(&topic).Error; err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { conn.Delete(&topic) })
		for _, event := range []string{eventNotification.EventTypeReviewPending, eventNotification.EventTypeReviewApproved, eventNotification.EventTypeReviewRejected} {
			subject := "Candidate body excerpt"
			if event == eventNotification.EventTypeReviewRejected {
				subject = "C******t"
			}
			notice := &eventNotification.Entity{EventType: event, Payload: eventNotification.NotificationPayload{TopicId: topic.Id, TopicTitle: subject}}
			if err := hydrateNotifications([]*eventNotification.Entity{notice}); err != nil {
				t.Fatal(err)
			}
			if notice.Payload.TopicTitle != subject {
				t.Errorf("%s: subject=%q, want %q", event, notice.Payload.TopicTitle, subject)
			}
		}
		social := &eventNotification.Entity{EventType: eventNotification.EventTypeLike, Payload: eventNotification.NotificationPayload{TopicId: topic.Id, TopicTitle: "Old snapshot"}}
		if err := hydrateNotifications([]*eventNotification.Entity{social}); err != nil {
			t.Fatal(err)
		}
		if social.Payload.TopicTitle != liveTitle {
			t.Errorf("social title=%q", social.Payload.TopicTitle)
		}
	}
}
