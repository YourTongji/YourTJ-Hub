package eventhandlers

import (
	"context"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"gorm.io/gorm"
)

func setupSearchOutboxDB(t *testing.T) *gorm.DB {
	t.Helper()
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(&taskQueue.Entity{}); err != nil {
		t.Fatalf("migrate task queue: %v", err)
	}
	if err := conn.Unscoped().Where("1 = 1").Delete(&taskQueue.Entity{}).Error; err != nil {
		t.Fatalf("clear task queue: %v", err)
	}
	return conn
}

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
	conn := setupSearchOutboxDB(t)
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
	var count int64
	if err := conn.Model(&taskQueue.Entity{}).Where("type = ?", "topic-search.sync").Count(&count).Error; err != nil {
		t.Fatalf("count topic outbox tasks: %v", err)
	}
	if count != 1 {
		t.Fatalf("topic outbox task count = %d, want 1", count)
	}
}

func TestHandleTopicPublishedAndUpdated(t *testing.T) {
	for _, test := range []struct {
		name    string
		handler func(context.Context, *TopicPublishedEvent) error
	}{
		{name: "published", handler: handleTopicPublished},
		{name: "updated", handler: func(ctx context.Context, event *TopicPublishedEvent) error {
			return handleTopicUpdated(ctx, (*TopicUpdatedEvent)(event))
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()
			conn := setupSearchOutboxDB(t)
			event := &TopicPublishedEvent{Topic: &topics.Entity{Id: 321, UserId: 654, Title: "published topic"}}
			if err := test.handler(ctx, event); err != nil {
				t.Fatalf("handle topic %s error = %v", test.name, err)
			}
			var count int64
			if err := conn.Model(&taskQueue.Entity{}).Where("type = ?", "topic-search.sync").Count(&count).Error; err != nil {
				t.Fatalf("count topic outbox tasks: %v", err)
			}
			if count != 1 {
				t.Fatalf("topic outbox task count = %d, want 1", count)
			}
		})
	}
}

func TestUserSearchIndexUpdatedEventSubject(t *testing.T) {
	if id, userId, title := (*UserSearchIndexUpdatedEvent)(nil).Subject(); id != 0 || userId != 0 || title != "" {
		t.Fatalf("nil event Subject() = (%d, %d, %q), want (0, 0, \"\")", id, userId, title)
	}
	event := &UserSearchIndexUpdatedEvent{UserId: 123}
	if id, userId, title := event.Subject(); id != 123 || userId != 0 || title != "" {
		t.Fatalf("event Subject() = (%d, %d, %q), want (123, 0, \"\")", id, userId, title)
	}
}

func TestCategorySearchIndexUpdatedEventSubject(t *testing.T) {
	if id, userId, title := (*CategorySearchIndexUpdatedEvent)(nil).Subject(); id != 0 || userId != 0 || title != "" {
		t.Fatalf("nil event Subject() = (%d, %d, %q), want (0, 0, \"\")", id, userId, title)
	}
	event := &CategorySearchIndexUpdatedEvent{CategoryId: 456}
	if id, userId, title := event.Subject(); id != 456 || userId != 0 || title != "" {
		t.Fatalf("event Subject() = (%d, %d, %q), want (456, 0, \"\")", id, userId, title)
	}
}

func TestCategorySearchIndexDeletedEventSubject(t *testing.T) {
	if id, userId, title := (*CategorySearchIndexDeletedEvent)(nil).Subject(); id != 0 || userId != 0 || title != "" {
		t.Fatalf("nil event Subject() = (%d, %d, %q), want (0, 0, \"\")", id, userId, title)
	}
	event := &CategorySearchIndexDeletedEvent{CategoryId: 789}
	if id, userId, title := event.Subject(); id != 789 || userId != 0 || title != "" {
		t.Fatalf("event Subject() = (%d, %d, %q), want (789, 0, \"\")", id, userId, title)
	}
}

func TestHandleUserSearchIndexUpdated(t *testing.T) {
	ctx := context.Background()
	conn := setupSearchOutboxDB(t)
	if err := handleUserSearchIndexUpdated(ctx, nil); err != nil {
		t.Fatalf("handleUserSearchIndexUpdated(nil) error = %v, want nil", err)
	}
	if err := handleUserSearchIndexUpdated(ctx, &UserSearchIndexUpdatedEvent{UserId: 999999}); err != nil {
		t.Fatalf("handleUserSearchIndexUpdated(event) error = %v, want nil", err)
	}
	var count int64
	if err := conn.Model(&taskQueue.Entity{}).Where("type = ?", "user-search.sync").Count(&count).Error; err != nil {
		t.Fatalf("count user outbox tasks: %v", err)
	}
	if count != 1 {
		t.Fatalf("user outbox task count = %d, want 1", count)
	}
}

func TestHandleUserSignUpSearchIndex(t *testing.T) {
	ctx := context.Background()
	conn := setupSearchOutboxDB(t)
	if err := handleUserSignUpSearchIndex(ctx, nil); err != nil {
		t.Fatalf("handleUserSignUpSearchIndex(nil) error = %v, want nil", err)
	}
	if err := handleUserSignUpSearchIndex(ctx, &UserSignUpEvent{UserId: 999999}); err != nil {
		t.Fatalf("handleUserSignUpSearchIndex(event) error = %v, want nil", err)
	}
	var count int64
	if err := conn.Model(&taskQueue.Entity{}).Where("type = ?", "user-search.sync").Count(&count).Error; err != nil {
		t.Fatalf("count signup outbox tasks: %v", err)
	}
	if count != 1 {
		t.Fatalf("signup outbox task count = %d, want 1", count)
	}
}

func TestHandleUserSignUpSearchIndexDoesNotReadDatabase(t *testing.T) {
	ctx := context.Background()
	setupSearchOutboxDB(t)

	if err := handleUserSignUpSearchIndex(ctx, &UserSignUpEvent{UserId: 999999}); err != nil {
		t.Fatalf("handleUserSignUpSearchIndex(event) error = %v, want nil", err)
	}
}

func TestHandleCategorySearchIndexUpdated(t *testing.T) {
	ctx := context.Background()
	conn := setupSearchOutboxDB(t)
	if err := handleCategorySearchIndexUpdated(ctx, nil); err != nil {
		t.Fatalf("handleCategorySearchIndexUpdated(nil) error = %v, want nil", err)
	}
	if err := handleCategorySearchIndexUpdated(ctx, &CategorySearchIndexUpdatedEvent{CategoryId: 999999}); err != nil {
		t.Fatalf("handleCategorySearchIndexUpdated(event) error = %v, want nil", err)
	}
	var count int64
	if err := conn.Model(&taskQueue.Entity{}).Where("type = ?", "category-search.sync").Count(&count).Error; err != nil {
		t.Fatalf("count category outbox tasks: %v", err)
	}
	if count != 1 {
		t.Fatalf("category outbox task count = %d, want 1", count)
	}
}

func TestHandleCategorySearchIndexDeleted(t *testing.T) {
	ctx := context.Background()
	conn := setupSearchOutboxDB(t)
	if err := handleCategorySearchIndexDeleted(ctx, nil); err != nil {
		t.Fatalf("handleCategorySearchIndexDeleted(nil) error = %v, want nil", err)
	}
	if err := handleCategorySearchIndexDeleted(ctx, &CategorySearchIndexDeletedEvent{CategoryId: 999999}); err != nil {
		t.Fatalf("handleCategorySearchIndexDeleted(event) error = %v, want nil", err)
	}
	var count int64
	if err := conn.Model(&taskQueue.Entity{}).Where("type = ?", "category-search.sync").Count(&count).Error; err != nil {
		t.Fatalf("count category delete outbox tasks: %v", err)
	}
	if count != 1 {
		t.Fatalf("category delete outbox task count = %d, want 1", count)
	}
}
