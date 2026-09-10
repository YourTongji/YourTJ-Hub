package eventhandlers

import (
	"context"
	"fmt"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/pointservice"
)

// handlePointTopicPublished 发帖获得积分
func handlePointTopicPublished(ctx context.Context, event *TopicPublishedEvent) error {
	topicID, userID, _ := event.Subject()
	if userID == 0 || topicID == 0 {
		return nil
	}
	// wiki 分站页面不是论坛话题：不发放发帖积分（review High；wiki 创建/编辑
	// 走独立的 wiki 贡献激励，PointsActionTopicPublished 仅限论坛话题）。
	if event != nil && event.Topic != nil && event.Topic.TopicType == topics.TopicTypeWiki {
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
