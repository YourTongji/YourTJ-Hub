package contentdeleteservice

import (
	"fmt"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/contentDeleteEvent"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/moderationLog"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/optRecord"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pointsRecord"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/postRevisions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userPoints"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiPageRevisions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiPages"
	"gorm.io/gorm"
)

func setupContentDeleteTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(
		&users.EntityComplete{},
		&topics.Entity{},
		&posts.Entity{},
		&postRevisions.Entity{},
		&optRecord.Entity{},
		&moderationLog.Entity{},
		&contentDeleteEvent.Entity{},
		&pointsRecord.Entity{},
		&userPoints.Entity{},
		&wikiPages.Entity{},
		&wikiPageRevisions.Entity{},
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

// seedWikiTopic 造一个 wiki 分站页面话题（topic_type=wiki）+ 首帖 + wiki 页面 + 1 条
// approved 修订，返回 pageID。供 wiki 话题删除级联测试复用。
func seedWikiTopic(t *testing.T, conn *gorm.DB, topicID, authorID uint64) uint64 {
	t.Helper()
	pageID := topicID + 20
	now := time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC)
	if err := conn.Create(&users.EntityComplete{Id: authorID, Username: fmt.Sprintf("wiki-author-%d", topicID)}).Error; err != nil {
		t.Fatalf("create wiki author: %v", err)
	}
	if err := conn.Create(&topics.Entity{
		Id: topicID, Title: fmt.Sprintf("Wiki Page %d", topicID), UserId: authorID, Status: 1,
		ProcessStatus: topics.ProcessStatusNormal, TopicType: topics.TopicTypeWiki,
		PostCount: 1, PostSeq: 1, FirstPostId: topicID + 10,
		VisibilityStatus: topics.VisibilityActive, RetentionStatus: topics.RetentionNormal,
		CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatalf("create wiki topic: %v", err)
	}
	if err := conn.Create(&posts.Entity{
		Id: topicID + 10, TopicId: topicID, PostNo: 1, UserId: authorID, Content: "wiki body",
		VisibilityStatus: posts.VisibilityActive, RetentionStatus: posts.RetentionNormal,
		CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatalf("create wiki first post: %v", err)
	}
	if err := conn.Create(&wikiPages.Entity{
		Id: pageID, TopicId: topicID, Namespace: "guide", Path: fmt.Sprintf("page-%d", topicID),
		CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatalf("create wiki page: %v", err)
	}
	if err := conn.Create(&wikiPageRevisions.Entity{
		PageId: pageID, RevisionNo: 1, Title: fmt.Sprintf("Wiki Page %d", topicID), Content: "wiki body",
		RenderedHTML: "<p>wiki body</p>", Status: wikiPageRevisions.StatusApproved, EditorId: authorID,
	}).Error; err != nil {
		t.Fatalf("create wiki revision: %v", err)
	}
	return pageID
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

// R1：待审（process_status=2）话题被作者删除后，process_status 应复位为正常，
// 不再停留在管理审核队列（避免"已删除+待审"幽灵项）。
func TestDeleteTopicByUserVoidsPendingReview(t *testing.T) {
	conn := setupContentDeleteTestDB(t)
	authorID, _ := seedTopicWithOptionalReply(t, conn, 941200, false)
	if err := topics.UpdateProcessStatus(941200, topics.ProcessStatusPending); err != nil {
		t.Fatalf("set pending review: %v", err)
	}

	if err := DeleteTopicByUser(authorID, 941200); err != nil {
		t.Fatalf("DeleteTopicByUser: %v", err)
	}

	topic := topics.UnscopedGet(941200)
	if topic.ProcessStatus != topics.ProcessStatusNormal {
		t.Fatalf("process_status = %d after delete, want %d (normal)", topic.ProcessStatus, topics.ProcessStatusNormal)
	}
	if topic.VisibilityStatus != topics.VisibilityUserDeleted {
		t.Fatalf("visibility = %s, want USER_DELETED", topic.VisibilityStatus)
	}
	// 待审队列不应再包含被删话题。
	pending := topics.PagePendingReview(1, 50)
	for _, item := range pending.Data {
		if item.Id == 941200 {
			t.Fatal("deleted topic still appears in pending review queue")
		}
	}
}

// review MEDIUM-1：作者永久删除话题时，只清理已进入删除生命周期的回复
// （首楼墓碑/级联软删行），其他用户仍 ACTIVE 的回复正文与状态必须保留。
func TestPurgeTopicKeepsOtherUsersActiveReplies(t *testing.T) {
	conn := setupContentDeleteTestDB(t)
	authorID, replyAuthorID := seedTopicWithOptionalReply(t, conn, 941300, true)

	if err := DeleteTopicByUser(authorID, 941300); err != nil {
		t.Fatalf("DeleteTopicByUser: %v", err)
	}
	if err := PurgeContent(authorID, ContentTypeTopic, 941300, "user_purge"); err != nil {
		t.Fatalf("PurgeContent: %v", err)
	}

	// 话题本身已永久删除。
	topic := topics.UnscopedGet(941300)
	if topic.RetentionStatus != topics.RetentionPurged {
		t.Fatalf("topic retention = %s, want PURGED", topic.RetentionStatus)
	}

	// 作者首楼（墓碑）被清空并置 PURGED。
	firstPost := posts.UnscopedGet(941300 + 100)
	if firstPost.RetentionStatus != posts.RetentionPurged || firstPost.Content != "" {
		t.Fatalf("author first post should be purged and blanked: %#v", firstPost)
	}

	// 其他用户的 ACTIVE 回复保留正文、不被置 PURGED。
	reply := posts.UnscopedGet(941300 + 200)
	if reply.VisibilityStatus != posts.VisibilityActive || reply.RetentionStatus != posts.RetentionNormal {
		t.Fatalf("other user reply state = %s/%s, want ACTIVE/NORMAL", reply.VisibilityStatus, reply.RetentionStatus)
	}
	if reply.Content != "reply body" {
		t.Fatalf("other user reply content was wiped: %q", reply.Content)
	}
	_ = replyAuthorID
}

// review MEDIUM-2：管理端恢复被治理删除的话题，级联回复与审计一并恢复。
func TestRestoreTopicAsModeratorRestoresTopicAndCascade(t *testing.T) {
	conn := setupContentDeleteTestDB(t)
	authorID, _ := seedTopicWithOptionalReply(t, conn, 941600, true)
	moderatorID := uint64(941699)

	topic := topics.Get(941600)
	if err := DeleteTopicAs(topic, moderatorID, topics.VisibilityModeratorRemoved, "policy violation"); err != nil {
		t.Fatalf("DeleteTopicAs: %v", err)
	}
	// 有回复场景：首楼为墓碑态（MODERATOR_REMOVED，无 deleted_at），他人回复保持 ACTIVE。
	firstPost := posts.UnscopedGet(941600 + 100)
	if firstPost.VisibilityStatus != posts.VisibilityModeratorRemoved {
		t.Fatalf("first post visibility = %s, want MODERATOR_REMOVED", firstPost.VisibilityStatus)
	}

	if err := RestoreTopicAsModerator(moderatorID, 941600); err != nil {
		t.Fatalf("RestoreTopicAsModerator: %v", err)
	}

	restored := topics.UnscopedGet(941600)
	if restored.VisibilityStatus != topics.VisibilityActive || restored.RetentionStatus != topics.RetentionNormal {
		t.Fatalf("restored topic state = %s/%s, want ACTIVE/NORMAL", restored.VisibilityStatus, restored.RetentionStatus)
	}
	restoredFirstPost := posts.UnscopedGet(941600 + 100)
	if restoredFirstPost.VisibilityStatus != posts.VisibilityActive || restoredFirstPost.DeletedAt.Valid {
		t.Fatalf("restored first post state = %s deleted=%v, want ACTIVE/no deleted_at", restoredFirstPost.VisibilityStatus, restoredFirstPost.DeletedAt.Valid)
	}
	restoredReply := posts.UnscopedGet(941600 + 200)
	if restoredReply.VisibilityStatus != posts.VisibilityActive {
		t.Fatalf("restored reply visibility = %s, want ACTIVE", restoredReply.VisibilityStatus)
	}

	// 恢复审计日志应写入。
	var restoredCount int64
	conn.Model(&moderationLog.Entity{}).
		Where("action = ? AND subject_type = ? AND subject_id = ?", moderationLog.ActionContentRestored, moderationLog.SubjectTopic, 941600).
		Count(&restoredCount)
	if restoredCount != 1 {
		t.Fatalf("content restored logs = %d, want 1", restoredCount)
	}
	_ = authorID
}

// review MEDIUM-2：作者不能借管理端恢复通道恢复自己被删的话题（管理端删除
// 状态话题作者不可自行恢复；恢复端点对非 MODERATOR_REMOVED 话题拒绝）。
func TestRestoreTopicAsModeratorRejectsUserDeletedTopic(t *testing.T) {
	conn := setupContentDeleteTestDB(t)
	authorID, _ := seedTopicWithOptionalReply(t, conn, 941601, false)
	moderatorID := uint64(941699)

	if err := DeleteTopicByUser(authorID, 941601); err != nil {
		t.Fatalf("DeleteTopicByUser: %v", err)
	}

	if err := RestoreTopicAsModerator(moderatorID, 941601); err == nil {
		t.Fatal("expected restore of USER_DELETED topic by moderator to fail")
	}
	if visible := topics.UnscopedGet(941601); visible.VisibilityStatus != topics.VisibilityUserDeleted {
		t.Fatalf("user-deleted topic state changed: %#v", visible)
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

// review LOW-7：postDeletionSnapshot 应填充 PostAuthor（审计日志中的作者名），
// 不能恒为空。
func TestPostDeletionSnapshotIncludesPostAuthor(t *testing.T) {
	conn := setupContentDeleteTestDB(t)
	authorID, _ := seedTopicWithOptionalReply(t, conn, 941800, false)

	post := posts.UnscopedGet(941800 + 100)
	snapshot := postDeletionSnapshot(post)
	if snapshot.PostAuthor == "" {
		t.Fatalf("postDeletionSnapshot PostAuthor is empty, want username of %d", authorID)
	}
	if snapshot.PostAuthorId != authorID {
		t.Fatalf("PostAuthorId = %d, want %d", snapshot.PostAuthorId, authorID)
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

// wiki 分站页面话题：作者删除 wiki 话题时级联物理删除 wiki_pages 与全部修订，
// 避免删除话题后残留孤儿页面继续出现在公开导航树/首页（DeleteTopicAs 级联，
// review wiki-topic-delete-cascade）。
func TestDeleteWikiTopicByUserCascadesPage(t *testing.T) {
	conn := setupContentDeleteTestDB(t)
	const authorID = uint64(949101)
	const topicID = uint64(949100)
	pageID := seedWikiTopic(t, conn, topicID, authorID)

	if err := DeleteTopicByUser(authorID, topicID); err != nil {
		t.Fatalf("DeleteTopicByUser: %v", err)
	}

	if topic := topics.UnscopedGet(topicID); topic.VisibilityStatus != topics.VisibilityUserDeleted {
		t.Fatalf("wiki topic visibility = %s, want USER_DELETED", topic.VisibilityStatus)
	}
	if page := wikiPages.GetByTopicId(topicID); page.Id != 0 {
		t.Fatalf("wiki page still present after topic delete: %#v", page)
	}
	if revs := wikiPageRevisions.ListByPage(pageID); len(revs) != 0 {
		t.Fatalf("wiki revisions still present after topic delete: %d", len(revs))
	}
	allPages, err := wikiPages.ListAll()
	if err != nil {
		t.Fatalf("list wiki pages: %v", err)
	}
	for _, p := range allPages {
		if p != nil && p.Id == pageID {
			t.Fatal("deleted wiki page still returned by wikiPages.ListAll()")
		}
	}
}

// 管理端治理删除 wiki 话题同样级联清理 wiki_pages/修订（DeleteTopicAs 级联；
// adminController 入口另有 wiki 话题禁止走论坛管理端删除的守卫，此为纵深防御验证）。
func TestDeleteWikiTopicByModeratorCascadesPage(t *testing.T) {
	conn := setupContentDeleteTestDB(t)
	const authorID = uint64(949201)
	const moderatorID = uint64(949202)
	const topicID = uint64(949200)
	pageID := seedWikiTopic(t, conn, topicID, authorID)

	topic := topics.Get(topicID)
	if err := DeleteTopicAs(topic, moderatorID, topics.VisibilityModeratorRemoved, "policy violation"); err != nil {
		t.Fatalf("DeleteTopicAs: %v", err)
	}

	if tpc := topics.UnscopedGet(topicID); tpc.VisibilityStatus != topics.VisibilityModeratorRemoved {
		t.Fatalf("wiki topic visibility = %s, want MODERATOR_REMOVED", tpc.VisibilityStatus)
	}
	if page := wikiPages.GetByTopicId(topicID); page.Id != 0 {
		t.Fatalf("wiki page still present after moderator delete: %#v", page)
	}
	if revs := wikiPageRevisions.ListByPage(pageID); len(revs) != 0 {
		t.Fatalf("wiki revisions still present after moderator delete: %d", len(revs))
	}
}

// wiki 软删对齐：恢复 wiki 话题时必须一并恢复 wiki_pages 与修订（清除
// deleted_at），否则出现"话题可见、页面消失"的幽灵状态。作者恢复路径验证。
func TestRestoreWikiTopicRestoresPageAndRevisions(t *testing.T) {
	conn := setupContentDeleteTestDB(t)
	const authorID = uint64(949501)
	const topicID = uint64(949500)
	pageID := seedWikiTopic(t, conn, topicID, authorID)

	if err := DeleteTopicByUser(authorID, topicID); err != nil {
		t.Fatalf("DeleteTopicByUser: %v", err)
	}
	if page := wikiPages.GetByTopicId(topicID); page.Id != 0 {
		t.Fatalf("wiki page still visible after delete: %#v", page)
	}
	var deletedPage wikiPages.Entity
	conn.Unscoped().Table("wiki_pages").Where("id = ?", pageID).First(&deletedPage)
	if !deletedPage.DeletedAt.Valid {
		t.Fatal("wiki page should be soft-deleted (deleted_at set) after topic delete")
	}

	if err := RestoreContent(authorID, ContentTypeTopic, topicID); err != nil {
		t.Fatalf("RestoreContent: %v", err)
	}
	topic := topics.UnscopedGet(topicID)
	if topic.VisibilityStatus != topics.VisibilityActive || topic.DeletedAt.Valid {
		t.Fatalf("restored topic state = %s deleted=%v, want ACTIVE/no deleted_at", topic.VisibilityStatus, topic.DeletedAt.Valid)
	}
	if page := wikiPages.GetByTopicId(topicID); page.Id == 0 {
		t.Fatal("wiki page not restored after topic restore")
	}
	if revs := wikiPageRevisions.ListByPage(pageID); len(revs) != 1 {
		t.Fatalf("wiki revisions not restored after topic restore: %d", len(revs))
	}
}

// 管理端恢复治理删除的 wiki 话题同样级联恢复 wiki_pages 与修订（同一 helper）。
func TestRestoreWikiTopicAsModeratorRestoresPageAndRevisions(t *testing.T) {
	conn := setupContentDeleteTestDB(t)
	const authorID = uint64(949601)
	const moderatorID = uint64(949602)
	const topicID = uint64(949600)
	pageID := seedWikiTopic(t, conn, topicID, authorID)

	topic := topics.Get(topicID)
	if err := DeleteTopicAs(topic, moderatorID, topics.VisibilityModeratorRemoved, "policy violation"); err != nil {
		t.Fatalf("DeleteTopicAs: %v", err)
	}
	if err := RestoreTopicAsModerator(moderatorID, topicID); err != nil {
		t.Fatalf("RestoreTopicAsModerator: %v", err)
	}
	if page := wikiPages.GetByTopicId(topicID); page.Id == 0 {
		t.Fatal("wiki page not restored after moderator topic restore")
	}
	if revs := wikiPageRevisions.ListByPage(pageID); len(revs) != 1 {
		t.Fatalf("wiki revisions not restored after moderator topic restore: %d", len(revs))
	}
}

// review P1：注销删除全部内容时，他人 wiki 页面上 editor_id=注销者 的修订也必须
// 清理（DeleteAllUserContent 现只按本人页面 DeleteByPage，未覆盖他人页面上的本人
// 修订），否则修订正文与 editorId 仍从公开 API 输出，与"删除全部本人内容"语义不符。
func TestDeleteAllUserContentRemovesWikiRevisionsByEditorOnOthersPages(t *testing.T) {
	conn := setupContentDeleteTestDB(t)
	const (
		userA = uint64(949301)
		userB = uint64(949302)
		userC = uint64(949303)
	)
	now := time.Now()
	mustCreate := func(v any) {
		t.Helper()
		if err := conn.Create(v).Error; err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	for id, name := range map[uint64]string{userA: "userA", userB: "userB", userC: "userC"} {
		mustCreate(&users.EntityComplete{Id: id, Username: name})
	}

	// 他人 userA 的 wiki 页面 + 3 条修订（rev1 userA approved / rev2 userB approved / rev3 userC pending）。
	const (
		aTopic = uint64(949310)
		aPage  = uint64(949312)
		rev1   = uint64(949320)
		rev2   = uint64(949321)
		rev3   = uint64(949322)
	)
	mustCreate(&topics.Entity{
		Id: aTopic, Title: "A Wiki", UserId: userA, Status: 1, ProcessStatus: topics.ProcessStatusNormal,
		TopicType: topics.TopicTypeWiki, PostCount: 1, PostSeq: 1, FirstPostId: aTopic + 1,
		VisibilityStatus: topics.VisibilityActive, RetentionStatus: topics.RetentionNormal,
		CreatedAt: now, UpdatedAt: now,
	})
	mustCreate(&posts.Entity{
		Id: aTopic + 1, TopicId: aTopic, PostNo: 1, UserId: userA, Content: "a body",
		VisibilityStatus: posts.VisibilityActive, RetentionStatus: posts.RetentionNormal,
		CreatedAt: now, UpdatedAt: now,
	})
	mustCreate(&wikiPages.Entity{
		Id: aPage, TopicId: aTopic, Namespace: "guide", Path: "a-wiki-page", CreatedAt: now, UpdatedAt: now,
	})
	mustCreate(&wikiPageRevisions.Entity{Id: rev1, PageId: aPage, RevisionNo: 1, Title: "A", Content: "a1", Status: wikiPageRevisions.StatusApproved, EditorId: userA})
	mustCreate(&wikiPageRevisions.Entity{Id: rev2, PageId: aPage, RevisionNo: 2, Title: "A", Content: "a2", Status: wikiPageRevisions.StatusApproved, EditorId: userB})
	mustCreate(&wikiPageRevisions.Entity{Id: rev3, PageId: aPage, RevisionNo: 3, Title: "A", Content: "a3", Status: wikiPageRevisions.StatusPending, EditorId: userC})

	// 注销者 userB 自己的 wiki 页面 + 1 条本人修订。
	const (
		bTopic = uint64(949330)
		bPage  = uint64(949332)
		bRev   = uint64(949340)
	)
	mustCreate(&topics.Entity{
		Id: bTopic, Title: "B Wiki", UserId: userB, Status: 1, ProcessStatus: topics.ProcessStatusNormal,
		TopicType: topics.TopicTypeWiki, PostCount: 1, PostSeq: 1, FirstPostId: bTopic + 1,
		VisibilityStatus: topics.VisibilityActive, RetentionStatus: topics.RetentionNormal,
		CreatedAt: now, UpdatedAt: now,
	})
	mustCreate(&posts.Entity{
		Id: bTopic + 1, TopicId: bTopic, PostNo: 1, UserId: userB, Content: "b body",
		VisibilityStatus: posts.VisibilityActive, RetentionStatus: posts.RetentionNormal,
		CreatedAt: now, UpdatedAt: now,
	})
	mustCreate(&wikiPages.Entity{
		Id: bPage, TopicId: bTopic, Namespace: "guide", Path: "b-wiki-page", CreatedAt: now, UpdatedAt: now,
	})
	mustCreate(&wikiPageRevisions.Entity{Id: bRev, PageId: bPage, RevisionNo: 1, Title: "B", Content: "b1", Status: wikiPageRevisions.StatusApproved, EditorId: userB})

	// 他人 forum 话题下 userB 的一条回复（posts 分支仍按既有语义删除）。
	const (
		fTopic = uint64(949350)
		fReply = uint64(949351)
	)
	mustCreate(&topics.Entity{
		Id: fTopic, Title: "Forum", UserId: userA, Status: 1, ProcessStatus: topics.ProcessStatusNormal,
		TopicType: topics.TopicTypeForum, PostCount: 2, PostSeq: 2, ReplyCount: 1, FirstPostId: fTopic + 10,
		VisibilityStatus: topics.VisibilityActive, RetentionStatus: topics.RetentionNormal,
		CreatedAt: now, UpdatedAt: now,
	})
	mustCreate(&posts.Entity{
		Id: fTopic + 10, TopicId: fTopic, PostNo: 1, UserId: userA, Content: "f body",
		VisibilityStatus: posts.VisibilityActive, RetentionStatus: posts.RetentionNormal,
		CreatedAt: now, UpdatedAt: now,
	})
	mustCreate(&posts.Entity{
		Id: fReply, TopicId: fTopic, PostNo: 2, UserId: userB, Content: "userB reply in forum",
		VisibilityStatus: posts.VisibilityActive, RetentionStatus: posts.RetentionNormal,
		CreatedAt: now, UpdatedAt: now,
	})

	if err := DeleteAllUserContent(userB); err != nil {
		t.Fatalf("DeleteAllUserContent: %v", err)
	}

	// 他人页面上 editor_id=userB 的 approved 修订被清理，userA/userC 的修订保留。
	if got := wikiPageRevisions.Get(rev2); got.Id != 0 {
		t.Fatalf("userB revision on others page still present: %#v", got)
	}
	if got := wikiPageRevisions.Get(rev1); got.Id == 0 {
		t.Fatal("userA revision was deleted")
	}
	if got := wikiPageRevisions.Get(rev3); got.Id == 0 {
		t.Fatal("userC pending revision was deleted")
	}
	// 他人 wiki 页面本身不受影响。
	if page := wikiPages.Get(aPage); page.Id == 0 {
		t.Fatal("userA wiki page was deleted")
	}
	// 注销者自己页面的修订被清理。
	if got := wikiPageRevisions.Get(bRev); got.Id != 0 {
		t.Fatalf("userB own page revision still present: %#v", got)
	}
	// userB 在他人 forum 话题下的回复仍按既有 posts 分支删除。
	if reply := posts.UnscopedGet(fReply); reply.VisibilityStatus != posts.VisibilityUserDeleted {
		t.Fatalf("userB forum reply visibility = %s, want USER_DELETED", reply.VisibilityStatus)
	}
}
