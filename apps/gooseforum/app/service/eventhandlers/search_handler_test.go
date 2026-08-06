package eventhandlers

import (
	"context"
	"testing"

	"github.com/leancodebox/GooseForum/app/models/forum/topics"
)

func TestTopicDeletedEventSubject(t *testing.T) {
	if id, userId, title := (*TopicDeletedEvent)(nil).Subject(); id != 0 || userId != 0 || title != "" {
		t.Fatalf("nil event Subject() = (%d, %d, %q), want (0, 0, \"\")", id, userId, title)
	}
	event := &TopicDeletedEvent{Topic: &topics.Entity{
		Id:     123,
		UserId: 456,
		Title:  "deleted topic",
	}}
	if id, userId, title := event.Subject(); id != 123 || userId != 456 || title != "deleted topic" {
		t.Fatalf("event Subject() = (%d, %d, %q), want (123, 456, %q)", id, userId, title, "deleted topic")
	}
}

func TestHandleTopicDeleted(t *testing.T) {
	ctx := context.Background()
	if err := handleTopicDeleted(ctx, nil); err != nil {
		t.Fatalf("handleTopicDeleted(nil) error = %v, want nil", err)
	}
	event := &TopicDeletedEvent{Topic: &topics.Entity{
		Id:            123,
		UserId:        456,
		Title:         "deleted topic",
		ProcessStatus: 1,
	}}
	if err := handleTopicDeleted(ctx, event); err != nil {
		t.Fatalf("handleTopicDeleted(event) error = %v, want nil", err)
	}
}
