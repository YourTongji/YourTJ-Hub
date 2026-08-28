package eventhandlers

import (
	"context"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
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
	if err := handleUserSearchIndexUpdated(ctx, nil); err != nil {
		t.Fatalf("handleUserSearchIndexUpdated(nil) error = %v, want nil", err)
	}
	// 建表使 users.Get 对不存在用户返回 RecordNotFound（而非 no such table）
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(&users.EntityComplete{}); err != nil {
		t.Fatalf("migrate users table: %v", err)
	}
	// 用户不存在（RecordNotFound）→ 删除分支 → 无 Meilisearch 返回 nil
	if err := handleUserSearchIndexUpdated(ctx, &UserSearchIndexUpdatedEvent{UserId: 999999}); err != nil {
		t.Fatalf("handleUserSearchIndexUpdated(event) error = %v, want nil", err)
	}
}

func TestHandleUserSignUpSearchIndex(t *testing.T) {
	ctx := context.Background()
	if err := handleUserSignUpSearchIndex(ctx, nil); err != nil {
		t.Fatalf("handleUserSignUpSearchIndex(nil) error = %v, want nil", err)
	}
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(&users.EntityComplete{}); err != nil {
		t.Fatalf("migrate users table: %v", err)
	}
	if err := handleUserSignUpSearchIndex(ctx, &UserSignUpEvent{UserId: 999999}); err != nil {
		t.Fatalf("handleUserSignUpSearchIndex(event) error = %v, want nil", err)
	}
}

func TestHandleUserSignUpSearchIndexReturnsDatabaseError(t *testing.T) {
	ctx := context.Background()
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(&users.EntityComplete{}); err != nil {
		t.Fatalf("migrate users table: %v", err)
	}
	if err := conn.Migrator().DropTable(&users.EntityComplete{}); err != nil {
		t.Fatalf("drop users table: %v", err)
	}
	t.Cleanup(func() {
		if err := conn.AutoMigrate(&users.EntityComplete{}); err != nil {
			t.Errorf("restore users table: %v", err)
		}
	})

	if err := handleUserSignUpSearchIndex(ctx, &UserSignUpEvent{UserId: 999999}); err == nil {
		t.Fatal("handleUserSignUpSearchIndex(event) error = nil, want database error")
	}
}

func TestHandleCategorySearchIndexUpdated(t *testing.T) {
	ctx := context.Background()
	if err := handleCategorySearchIndexUpdated(ctx, nil); err != nil {
		t.Fatalf("handleCategorySearchIndexUpdated(nil) error = %v, want nil", err)
	}
	if err := handleCategorySearchIndexUpdated(ctx, &CategorySearchIndexUpdatedEvent{CategoryId: 999999}); err != nil {
		t.Fatalf("handleCategorySearchIndexUpdated(event) error = %v, want nil", err)
	}
}

func TestHandleCategorySearchIndexDeleted(t *testing.T) {
	ctx := context.Background()
	if err := handleCategorySearchIndexDeleted(ctx, nil); err != nil {
		t.Fatalf("handleCategorySearchIndexDeleted(nil) error = %v, want nil", err)
	}
	if err := handleCategorySearchIndexDeleted(ctx, &CategorySearchIndexDeletedEvent{CategoryId: 999999}); err != nil {
		t.Fatalf("handleCategorySearchIndexDeleted(event) error = %v, want nil", err)
	}
}
