package api

// issue #553：回复删除端点的错误语义细分。
// 重复删除保留错误但返回 post.alreadyDeleted；首楼返回 post.firstPostUndeletable；
// 非属主探测已删行保持 post.notFound（不泄露已删内容存在性）。

import (
	"fmt"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"gorm.io/gorm"
)

// createPostDeleteTestUser 创建带唯一用户名的激活用户（createLLMSCacheUser 的
// 用户名固定，同一测试创建多个用户会撞唯一约束）。
func createPostDeleteTestUser(t *testing.T, conn *gorm.DB, id uint64) {
	t.Helper()
	t.Cleanup(func() {
		conn.Unscoped().Where("id = ?", id).Delete(&users.EntityComplete{})
	})
	now := time.Now().Add(-time.Hour)
	if err := conn.Create(&users.EntityComplete{Id: id, Username: fmt.Sprintf("post-delete-user-%d", id), IsActivated: users.ActivationSuccess, CreatedAt: now}).Error; err != nil {
		t.Fatalf("create post delete test user: %v", err)
	}
}

func TestDeletePostDuplicateReturnsAlreadyDeleted(t *testing.T) {
	conn := setupLLMSCacheTestDB(t)
	base := uint64(time.Now().UnixNano()%1_000_000_000) + 9_501_000_000
	userID := base + 1
	topicID := base + 2
	firstPostID := base + 3
	replyID := base + 4
	createPostDeleteTestUser(t, conn, userID)
	createLLMSCacheTopic(t, conn, topicID, firstPostID, userID, "Duplicate delete topic", "first post body", nil)
	createLLMSCacheReply(t, conn, replyID, topicID, userID, 2, "reply deleted twice")

	first := DeletePost(component.BetterRequest[DeletePostReq]{UserId: userID, Params: DeletePostReq{PostId: replyID}})
	if first.Data.Code != component.SUCCESS {
		t.Fatalf("first delete = code=%v msg=%v, want SUCCESS", first.Data.Code, first.Data.MessageCode)
	}
	second := DeletePost(component.BetterRequest[DeletePostReq]{UserId: userID, Params: DeletePostReq{PostId: replyID}})
	if second.Data.Code != component.FAIL || second.Data.MessageCode != component.MessagePostAlreadyDeleted {
		t.Fatalf("duplicate delete = code=%v msg=%v, want FAIL/MessagePostAlreadyDeleted", second.Data.Code, second.Data.MessageCode)
	}
}

func TestDeletePostRejectsFirstPostWithDedicatedCode(t *testing.T) {
	conn := setupLLMSCacheTestDB(t)
	base := uint64(time.Now().UnixNano()%1_000_000_000) + 9_502_000_000
	userID := base + 1
	topicID := base + 2
	firstPostID := base + 3
	createPostDeleteTestUser(t, conn, userID)
	createLLMSCacheTopic(t, conn, topicID, firstPostID, userID, "First post delete topic", "first post body", nil)

	res := DeletePost(component.BetterRequest[DeletePostReq]{UserId: userID, Params: DeletePostReq{PostId: firstPostID}})
	if res.Data.Code != component.FAIL || res.Data.MessageCode != component.MessagePostFirstPostUndeletable {
		t.Fatalf("first post delete = code=%v msg=%v, want FAIL/MessagePostFirstPostUndeletable", res.Data.Code, res.Data.MessageCode)
	}
}

// 非属主对已删回复的删除尝试保持 post.notFound，不向探测者泄露已删内容存在性。
func TestDeletePostNonOwnerProbeOfDeletedReplyKeepsNotFound(t *testing.T) {
	conn := setupLLMSCacheTestDB(t)
	base := uint64(time.Now().UnixNano()%1_000_000_000) + 9_503_000_000
	authorID := base + 1
	proberID := base + 2
	topicID := base + 3
	firstPostID := base + 4
	replyID := base + 5
	createPostDeleteTestUser(t, conn, authorID)
	createPostDeleteTestUser(t, conn, proberID)
	createLLMSCacheTopic(t, conn, topicID, firstPostID, authorID, "Probe topic", "first post body", nil)
	createLLMSCacheReply(t, conn, replyID, topicID, authorID, 2, "reply hidden from prober")

	deleted := DeletePost(component.BetterRequest[DeletePostReq]{UserId: authorID, Params: DeletePostReq{PostId: replyID}})
	if deleted.Data.Code != component.SUCCESS {
		t.Fatalf("author delete = code=%v msg=%v, want SUCCESS", deleted.Data.Code, deleted.Data.MessageCode)
	}
	probe := DeletePost(component.BetterRequest[DeletePostReq]{UserId: proberID, Params: DeletePostReq{PostId: replyID}})
	if probe.Data.Code != component.FAIL || probe.Data.MessageCode != component.MessagePostNotFound {
		t.Fatalf("non-owner probe = code=%v msg=%v, want FAIL/MessagePostNotFound", probe.Data.Code, probe.Data.MessageCode)
	}
}

// 未删除的他人回复维持 topic.operationDenied（既有语义回归守卫）。
func TestDeletePostNonOwnerActiveReplyKeepsOperationDenied(t *testing.T) {
	conn := setupLLMSCacheTestDB(t)
	base := uint64(time.Now().UnixNano()%1_000_000_000) + 9_504_000_000
	authorID := base + 1
	otherID := base + 2
	topicID := base + 3
	firstPostID := base + 4
	replyID := base + 5
	createPostDeleteTestUser(t, conn, authorID)
	createPostDeleteTestUser(t, conn, otherID)
	createLLMSCacheTopic(t, conn, topicID, firstPostID, authorID, "Denied topic", "first post body", nil)
	createLLMSCacheReply(t, conn, replyID, topicID, authorID, 2, "active reply of another user")

	res := DeletePost(component.BetterRequest[DeletePostReq]{UserId: otherID, Params: DeletePostReq{PostId: replyID}})
	if res.Data.Code != component.FAIL || res.Data.MessageCode != component.MessageTopicOperationDenied {
		t.Fatalf("non-owner active delete = code=%v msg=%v, want FAIL/MessageTopicOperationDenied", res.Data.Code, res.Data.MessageCode)
	}
}
