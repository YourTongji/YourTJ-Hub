package api

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pointsRecord"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userPoints"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"gorm.io/gorm"
)

func TestUpdateTopicStatusRejectsPublishingPendingTopic(t *testing.T) {
	conn := setupLLMSCacheTestDB(t)
	base := uint64(time.Now().UnixNano()%1_000_000_000) + 9_406_000_000
	ownerID := base + 1
	topicID := base + 2
	postID := base + 3
	createLLMSCacheTopic(t, conn, topicID, postID, ownerID, "Pending topic", "pending body", nil)
	if err := conn.Model(&topics.Entity{}).Where("id = ?", topicID).
		Update("process_status", topics.ProcessStatusPending).Error; err != nil {
		t.Fatalf("mark topic pending: %v", err)
	}

	res := UpdateTopicStatus(component.BetterRequest[TopicStatusReq]{
		UserId: ownerID,
		Params: TopicStatusReq{TopicId: topicID, TopicStatus: 1},
	})
	if res.Data.Code == component.SUCCESS {
		t.Fatalf("UpdateTopicStatus() unexpectedly published pending topic: %+v", res)
	}

	var topic topics.Entity
	if err := conn.Where("id = ?", topicID).Take(&topic).Error; err != nil {
		t.Fatalf("reload pending topic: %v", err)
	}
	if topic.Status != 1 {
		t.Fatalf("pending topic status = %d, want 1", topic.Status)
	}
}

func TestDeletePostReversesPrestigeWithoutBalanceRow(t *testing.T) {
	conn := setupLLMSCacheTestDB(t)
	base := uint64(time.Now().UnixNano()%1_000_000_000) + 9_407_000_000
	userID := base + 1
	topicID := base + 2
	post1ID := base + 3
	post2ID := base + 4
	// Seed a rewarded user (prestige matches the +2 reward) but no user_points row,
	// simulating a legacy/imported deployment whose balance row was lost. The backfill
	// v14 is responsible for reconstructing the balance from the ledger.
	if err := conn.Create(&users.EntityComplete{Id: userID, Username: fmt.Sprintf("rollback-user-%d", userID), Prestige: 2}).Error; err != nil {
		t.Fatalf("create rewarded user: %v", err)
	}
	t.Cleanup(func() {
		conn.Unscoped().Where("id = ?", userID).Delete(&users.EntityComplete{})
		conn.Where("source_key LIKE ?", fmt.Sprintf("post:%%")).Delete(&pointsRecord.Entity{})
		conn.Where("user_id = ?", userID).Delete(&userPoints.Entity{})
	})
	createLLMSCacheTopic(t, conn, topicID, post1ID, userID, "Rollback topic", "first post", nil)
	createLLMSCacheReply(t, conn, post2ID, topicID, userID, 2, "reply with reward")
	rewardKey := fmt.Sprintf("post:%d", post2ID)
	if err := conn.Create(&pointsRecord.Entity{UserId: userID, Action: "post_created", PointsChange: 2, SourceKey: &rewardKey}).Error; err != nil {
		t.Fatalf("create reward record: %v", err)
	}

	res := DeletePost(component.BetterRequest[DeletePostReq]{
		UserId: userID,
		Params: DeletePostReq{PostId: post2ID},
	})
	if res.Data.Code != component.SUCCESS {
		t.Fatalf("DeletePost() without user_points row failed: %+v", res)
	}
	var post posts.Entity
	if err := conn.Unscoped().Where("id = ?", post2ID).Take(&post).Error; err != nil {
		t.Fatalf("reload post: %v", err)
	}
	if !post.DeletedAt.Valid {
		t.Errorf("post should be soft-deleted, deleted_at = %+v", post.DeletedAt)
	}
	var user users.EntityComplete
	if err := conn.Where("id = ?", userID).Take(&user).Error; err != nil {
		t.Fatalf("reload user: %v", err)
	}
	if user.Prestige != 0 {
		t.Errorf("prestige = %d, want 0 (reversal applied to users table)", user.Prestige)
	}
	reversalKey := fmt.Sprintf("post-deleted:%d", post2ID)
	var reversal pointsRecord.Entity
	if err := conn.Where("source_key = ?", reversalKey).Take(&reversal).Error; err != nil {
		t.Fatalf("load reversal tombstone %q: %v", reversalKey, err)
	}
	if reversal.PointsChange != -2 {
		t.Errorf("tombstone points = %d, want -2", reversal.PointsChange)
	}
	var balance userPoints.Entity
	if err := conn.Where("user_id = ?", userID).Take(&balance).Error; !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("user_points row should be absent for backfill, got %v / %+v", err, balance)
	}
}
