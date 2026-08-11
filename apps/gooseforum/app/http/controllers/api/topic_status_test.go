package api

import (
	"fmt"
	"testing"
	"time"

	"github.com/leancodebox/GooseForum/app/http/controllers/component"
	"github.com/leancodebox/GooseForum/app/models/forum/pointsRecord"
	"github.com/leancodebox/GooseForum/app/models/forum/posts"
	"github.com/leancodebox/GooseForum/app/models/forum/topics"
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

func TestDeletePostRollsBackWhenRewardReversalFails(t *testing.T) {
	conn := setupLLMSCacheTestDB(t)
	base := uint64(time.Now().UnixNano()%1_000_000_000) + 9_407_000_000
	userID := base + 1
	topicID := base + 2
	post1ID := base + 3
	post2ID := base + 4
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
	if res.Data.Code == component.SUCCESS {
		t.Fatalf("DeletePost() succeeded without user_points row: %+v", res)
	}
	var post posts.Entity
	if err := conn.Where("id = ?", post2ID).Take(&post).Error; err != nil {
		t.Fatalf("post was deleted despite reversal failure: %v", err)
	}
}
