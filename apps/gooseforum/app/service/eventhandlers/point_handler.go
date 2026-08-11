package eventhandlers

import (
	"context"
	"fmt"

	"github.com/leancodebox/GooseForum/app/models/forum/posts"
	"github.com/leancodebox/GooseForum/app/models/forum/topics"
	"github.com/leancodebox/GooseForum/app/service/pointservice"
)

// handlePointTopicPublished 发帖获得积分
func handlePointTopicPublished(ctx context.Context, event *TopicPublishedEvent) error {
	topicID, userID, _ := event.Subject()
	if userID == 0 || topicID == 0 {
		return nil
	}
	currentTopic := topics.Get(topicID)
	if currentTopic.Id == 0 || currentTopic.Status != 1 {
		return nil
	}
	return pointservice.RewardPoints(userID, 10, pointservice.PointsActionTopicPublished, fmt.Sprintf("topic:%d", topicID))
}

// handlePointCommentCreated 评论获得积分
func handlePointCommentCreated(ctx context.Context, event *CommentCreatedEvent) error {
	if event == nil || event.PostId == 0 || posts.Get(event.PostId).Id == 0 {
		return nil
	}
	return pointservice.RewardPoints(event.UserId, 2, pointservice.PointsActionPostCreated, fmt.Sprintf("post:%d", event.PostId))
}
