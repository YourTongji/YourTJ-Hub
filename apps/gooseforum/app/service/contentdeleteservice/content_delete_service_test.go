package contentdeleteservice

import (
	"fmt"
	"testing"
	"time"

	"github.com/leancodebox/GooseForum/app/bundles/connect/dbconnect"
	"github.com/leancodebox/GooseForum/app/http/controllers/component"
	"github.com/leancodebox/GooseForum/app/models/forum/contentDeleteEvent"
	"github.com/leancodebox/GooseForum/app/models/forum/moderationLog"
	"github.com/leancodebox/GooseForum/app/models/forum/optRecord"
	"github.com/leancodebox/GooseForum/app/models/forum/posts"
	"github.com/leancodebox/GooseForum/app/models/forum/topics"
	"github.com/leancodebox/GooseForum/app/models/forum/users"
	"gorm.io/gorm"
)

func setupContentDeleteTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(
		&users.EntityComplete{},
		&topics.Entity{},
		&posts.Entity{},
		&optRecord.Entity{},
		&moderationLog.Entity{},
		&contentDeleteEvent.Entity{},
	); err != nil {
		t.Fatalf("migrate content delete tables: %v", err)
	}
	return conn
}

func seedTopicWithOptionalReply(t *testing.T, conn *gorm.DB, topicID uint64, withReply bool) (authorID uint64, replyAuthorID uint64) {
	t.Helper()
	authorID = topicID + 10
	firstPostID := topicID + 100
	replyAuthorID = topicID + 20
	replyPostID := topicID + 200
	now := time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC)

	if err := conn.Create(&users.EntityComplete{Id: authorID, Username: fmt.Sprintf("author-%d", topicID)}).Error; err != nil {
		t.Fatalf("create author: %v", err)
	}
	topic := topics.Entity{
		Id:               topicID,
		Title:            fmt.Sprintf("Topic %d", topicID),
		Excerpt:          "excerpt",
		UserId:           authorID,
		Status:           1,
		ProcessStatus:    0,
		PostCount:        1,
		PostSeq:          1,
		FirstPostId:      firstPostID,
		VisibilityStatus: topics.VisibilityActive,
		RetentionStatus:  topics.RetentionNormal,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if withReply {
		topic.PostCount = 2
		topic.PostSeq = 2
		topic.ReplyCount = 1
	}
	if err := conn.Create(&topic).Error; err != nil {
		t.Fatalf("create topic: %v", err)
	}
	if err := conn.Create(&posts.Entity{
		Id:               firstPostID,
		TopicId:          topicID,
		PostNo:           1,
		UserId:           authorID,
		Content:          "first post body",
		VisibilityStatus: posts.VisibilityActive,
		RetentionStatus:  posts.RetentionNormal,
		CreatedAt:        now,
		UpdatedAt:        now,
	}).Error; err != nil {
		t.Fatalf("create first post: %v", err)
	}
	if withReply {
		if err := conn.Create(&users.EntityComplete{Id: replyAuthorID, Username: fmt.Sprintf("replier-%d", topicID)}).Error; err != nil {
			t.Fatalf("create reply author: %v", err)
		}
		if err := conn.Create(&posts.Entity{
			Id:               replyPostID,
			TopicId:          topicID,
			PostNo:           2,
			UserId:           replyAuthorID,
			Content:          "reply body",
			VisibilityStatus: posts.VisibilityActive,
			RetentionStatus:  posts.RetentionNormal,
			CreatedAt:        now,
			UpdatedAt:        now,
		}).Error; err != nil {
			t.Fatalf("create reply: %v", err)
		}
	}
	return authorID, replyAuthorID
}

func TestDeleteTopicByUserWithoutRepliesCascades(t *testing.T) {
	conn := setupContentDeleteTestDB(t)
	authorID, _ := seedTopicWithOptionalReply(t, conn, 940001, false)

	if err := DeleteTopicByUser(authorID, 940001); err != nil {
		t.Fatalf("DeleteTopicByUser: %v", err)
	}

	if visible := topics.Get(940001); visible.Id != 0 {
		t.Fatalf("topic still visible via scoped Get: %#v", visible)
	}
	topic := topics.UnscopedGet(940001)
	if topic.Id == 0 {
		t.Fatal("expected unscoped topic row")
	}
	if topic.VisibilityStatus != topics.VisibilityUserDeleted {
		t.Fatalf("visibility = %s, want USER_DELETED", topic.VisibilityStatus)
	}
	if topic.RetentionStatus != topics.RetentionRecoverable {
		t.Fatalf("retention = %s, want RECOVERABLE", topic.RetentionStatus)
	}
	if !topic.DeletedAt.Valid {
		t.Fatal("expected topic deleted_at set")
	}

	firstPost := posts.UnscopedGet(940001 + 100)
	if firstPost.Id == 0 || !firstPost.DeletedAt.Valid {
		t.Fatalf("first post should be cascade soft-deleted: %#v", firstPost)
	}
	if firstPost.VisibilityStatus != posts.VisibilityUserDeleted {
		t.Fatalf("first post visibility = %s", firstPost.VisibilityStatus)
	}
}

func TestDeleteTopicByUserWithRepliesKeepsDiscussionTombstone(t *testing.T) {
	conn := setupContentDeleteTestDB(t)
	authorID, _ := seedTopicWithOptionalReply(t, conn, 940002, true)

	if err := DeleteTopicByUser(authorID, 940002); err != nil {
		t.Fatalf("DeleteTopicByUser: %v", err)
	}

	topic := topics.UnscopedGet(940002)
	if topic.VisibilityStatus != topics.VisibilityUserDeleted || topic.RetentionStatus != topics.RetentionRecoverable {
		t.Fatalf("topic state = %s/%s", topic.VisibilityStatus, topic.RetentionStatus)
	}

	firstPost := posts.Get(940002 + 100)
	if firstPost.Id == 0 {
		t.Fatal("first post should remain visible (tombstone, no deleted_at)")
	}
	if firstPost.DeletedAt.Valid {
		t.Fatal("tombstone first post must not set deleted_at")
	}
	if firstPost.VisibilityStatus != posts.VisibilityUserDeleted {
		t.Fatalf("first post visibility = %s", firstPost.VisibilityStatus)
	}

	reply := posts.Get(940002 + 200)
	if reply.Id == 0 || reply.VisibilityStatus != posts.VisibilityActive {
		t.Fatalf("reply should stay active: %#v", reply)
	}
}

func TestDeleteTopicAsModeratorRequiresNonRestorableState(t *testing.T) {
	conn := setupContentDeleteTestDB(t)
	authorID, _ := seedTopicWithOptionalReply(t, conn, 940003, false)
	moderatorID := uint64(940099)

	topic := topics.Get(940003)
	if err := DeleteTopicAs(topic, moderatorID, topics.VisibilityModeratorRemoved, "spam content"); err != nil {
		t.Fatalf("DeleteTopicAs: %v", err)
	}

	deleted := topics.UnscopedGet(940003)
	if deleted.VisibilityStatus != topics.VisibilityModeratorRemoved {
		t.Fatalf("visibility = %s, want MODERATOR_REMOVED", deleted.VisibilityStatus)
	}
	if deleted.DeleteReason != "spam content" {
		t.Fatalf("reason = %q", deleted.DeleteReason)
	}
	if deleted.DeletedBy != moderatorID {
		t.Fatalf("deletedBy = %d", deleted.DeletedBy)
	}

	// 作者不可自行恢复管理端删除。
	if err := RestoreContent(authorID, ContentTypeTopic, 940003); err == nil {
		t.Fatal("expected restore to fail for moderator-removed topic")
	} else if msg, ok := err.(component.MessageError); !ok || msg.Code != component.MessageContentNotRecoverable {
		t.Fatalf("restore error = %#v", err)
	}
}

func TestDeleteTopicByUserOwnerMismatch(t *testing.T) {
	conn := setupContentDeleteTestDB(t)
	_, _ = seedTopicWithOptionalReply(t, conn, 940004, false)

	err := DeleteTopicByUser(999999, 940004)
	if err == nil {
		t.Fatal("expected owner mismatch error")
	}
	if msg, ok := err.(component.MessageError); !ok || msg.Code != component.MessageTopicOwnerMismatch {
		t.Fatalf("error = %#v", err)
	}
}

func TestPurgeContentRejectsActiveTopic(t *testing.T) {
	conn := setupContentDeleteTestDB(t)
	authorID, _ := seedTopicWithOptionalReply(t, conn, 940005, false)

	if err := PurgeContent(authorID, ContentTypeTopic, 940005, "user_purge"); err == nil {
		t.Fatal("expected active topic purge to be rejected")
	}

	activeTopic := topics.UnscopedGet(940005)
	if activeTopic.RetentionStatus != topics.RetentionNormal || activeTopic.DeletedAt.Valid {
		t.Fatalf("active topic was changed: %#v", activeTopic)
	}
}

func TestPurgeContentRejectsModeratorRemovedTopic(t *testing.T) {
	conn := setupContentDeleteTestDB(t)
	authorID, _ := seedTopicWithOptionalReply(t, conn, 940006, false)
	topic := topics.Get(940006)
	if err := DeleteTopicAs(topic, 940099, topics.VisibilityModeratorRemoved, "policy violation"); err != nil {
		t.Fatalf("DeleteTopicAs: %v", err)
	}

	if err := PurgeContent(authorID, ContentTypeTopic, 940006, "user_purge"); err == nil {
		t.Fatal("expected moderator-removed topic purge to be rejected")
	}

	removedTopic := topics.UnscopedGet(940006)
	if removedTopic.VisibilityStatus != topics.VisibilityModeratorRemoved || removedTopic.RetentionStatus != topics.RetentionRecoverable {
		t.Fatalf("moderator removal state was changed: %#v", removedTopic)
	}
}

func TestPurgeContentAcceptsRecoverableUserTopic(t *testing.T) {
	conn := setupContentDeleteTestDB(t)
	authorID, _ := seedTopicWithOptionalReply(t, conn, 940007, false)
	if err := DeleteTopicByUser(authorID, 940007); err != nil {
		t.Fatalf("DeleteTopicByUser: %v", err)
	}

	if err := PurgeContent(authorID, ContentTypeTopic, 940007, "user_purge"); err != nil {
		t.Fatalf("PurgeContent: %v", err)
	}

	purgedTopic := topics.UnscopedGet(940007)
	if purgedTopic.RetentionStatus != topics.RetentionPurged || !purgedTopic.DeletedAt.Valid {
		t.Fatalf("topic was not purged: %#v", purgedTopic)
	}
	purgedPost := posts.UnscopedGet(940007 + 100)
	if purgedPost.RetentionStatus != posts.RetentionPurged || !purgedPost.DeletedAt.Valid {
		t.Fatalf("first post was not purged: %#v", purgedPost)
	}
}

func TestRestoreTopicDoesNotRestoreIndependentlyDeletedReply(t *testing.T) {
	conn := setupContentDeleteTestDB(t)
	authorID, replyAuthorID := seedTopicWithOptionalReply(t, conn, 940008, true)
	if _, err := DeletePostByUser(replyAuthorID, 940008+200); err != nil {
		t.Fatalf("DeletePostByUser: %v", err)
	}
	if err := DeleteTopicByUser(authorID, 940008); err != nil {
		t.Fatalf("DeleteTopicByUser: %v", err)
	}
	if err := RestoreContent(authorID, ContentTypeTopic, 940008); err != nil {
		t.Fatalf("RestoreContent: %v", err)
	}

	reply := posts.UnscopedGet(940008 + 200)
	if reply.VisibilityStatus != posts.VisibilityUserDeleted || !reply.DeletedAt.Valid {
		t.Fatalf("independently deleted reply was restored: %#v", reply)
	}
	firstPost := posts.UnscopedGet(940008 + 100)
	if firstPost.VisibilityStatus != posts.VisibilityActive || firstPost.DeletedAt.Valid {
		t.Fatalf("topic deletion tombstone was not restored: %#v", firstPost)
	}
}

// R14：删除/恢复/永久删除/隐私删除等生命周期事件写入埋点表。
func TestContentDeleteEventsRecorded(t *testing.T) {
	conn := setupContentDeleteTestDB(t)
	authorID, _ := seedTopicWithOptionalReply(t, conn, 940010, false)

	countEvents := func(eventType string) int64 {
		var count int64
		conn.Model(&contentDeleteEvent.Entity{}).
			Where("event_type = ? AND content_id = ?", eventType, 940010).
			Count(&count)
		return count
	}

	if err := DeleteTopicByUser(authorID, 940010); err != nil {
		t.Fatalf("DeleteTopicByUser: %v", err)
	}
	if got := countEvents(string(contentDeleteEvent.EventDeleted)); got != 1 {
		t.Fatalf("content_deleted events = %d, want 1", got)
	}

	if err := RestoreContent(authorID, ContentTypeTopic, 940010); err != nil {
		t.Fatalf("RestoreContent: %v", err)
	}
	if got := countEvents(string(contentDeleteEvent.EventRestored)); got != 1 {
		t.Fatalf("content_restored events = %d, want 1", got)
	}

	if err := DeleteTopicByUser(authorID, 940010); err != nil {
		t.Fatalf("DeleteTopicByUser (2nd): %v", err)
	}
	if err := PurgeContent(authorID, ContentTypeTopic, 940010, "user_purge"); err != nil {
		t.Fatalf("PurgeContent: %v", err)
	}
	if got := countEvents(string(contentDeleteEvent.EventPermanentDelete)); got != 1 {
		t.Fatalf("content_permanent_delete events = %d, want 1", got)
	}

	// 隐私删除话题应记录 privacy_delete_requested。
	privacyAuthorID, _ := seedTopicWithOptionalReply(t, conn, 940011, false)
	if err := PrivacyEraseContent(privacyAuthorID, ContentTypeTopic, 940011); err != nil {
		t.Fatalf("PrivacyEraseContent: %v", err)
	}
	var privacyCount int64
	conn.Model(&contentDeleteEvent.Entity{}).
		Where("event_type = ? AND content_id = ?", string(contentDeleteEvent.EventPrivacyDelete), 940011).
		Count(&privacyCount)
	if privacyCount != 1 {
		t.Fatalf("privacy_delete_requested events = %d, want 1", privacyCount)
	}
}
