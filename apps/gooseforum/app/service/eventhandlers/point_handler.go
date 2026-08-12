package eventhandlers

import (
	"context"
	"fmt"

	"github.com/leancodebox/GooseForum/app/service/pointservice"
)

// handlePointTopicPublished 发帖获得积分
func handlePointTopicPublished(ctx context.Context, event *TopicPublishedEvent) error {
	topicID, userID, _ := event.Subject()
	if userID == 0 || topicID == 0 {
		return nil
	}
	return pointservice.RewardPoints(userID, pointservice.TopicPublishedReward, pointservice.PointsActionTopicPublished, fmt.Sprintf("topic:%d", topicID))
}

// handlePointCommentCreated 评论获得积分
func handlePointCommentCreated(ctx context.Context, event *CommentCreatedEvent) error {
	if event == nil || event.UserId == 0 || event.PostId == 0 {
		return nil
	}
	return pointservice.RewardPoints(event.UserId, pointservice.PostCreatedReward, pointservice.PointsActionPostCreated, fmt.Sprintf("post:%d", event.PostId))
}
