package pointservice

import (
	"errors"
	"fmt"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"gorm.io/gorm"
)

// RewardPublicationTx commits a local reward with the content becoming public.
// The caller owns the publication transaction and its content locks. Reading
// through tx observes approval/status changes made earlier in that transaction.
// Existing source keys make a repeated approval or an old event harmless.
func RewardPublicationTx(tx *gorm.DB, topicID, postID uint64) error {
	var post posts.Entity
	if err := tx.Where("id = ? AND topic_id = ?", postID, topicID).Take(&post).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	var topic topics.Entity
	if err := tx.Take(&topic, topicID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	if topic.Status != 1 || topic.ProcessStatus != topics.ProcessStatusNormal ||
		topic.VisibilityStatus != topics.VisibilityActive || post.ProcessStatus != posts.ProcessStatusNormal ||
		post.VisibilityStatus != posts.VisibilityActive {
		return nil
	}
	if post.PostNo == 1 {
		if topic.TopicType == topics.TopicTypeWiki {
			return nil
		}
		return applyPointsTx(tx, topic.UserId, TopicPublishedReward, PointsActionTopicPublished,
			fmt.Sprintf("topic:%d", topicID), "")
	}
	return applyPointsTx(tx, post.UserId, PostCreatedReward, PointsActionPostCreated,
		fmt.Sprintf("post:%d", postID), "")
}
