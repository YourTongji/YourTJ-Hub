package postservice

import (
	"log/slog"
	"sync"
	"time"

	"github.com/leancodebox/GooseForum/app/models/forum/posts"
	"github.com/leancodebox/GooseForum/app/models/forum/topicUserStat"
	"github.com/leancodebox/GooseForum/app/models/forum/topics"
	"github.com/samber/lo"
)

const topicSequenceLockShards = 256

var topicSequenceLocks [topicSequenceLockShards]sync.Mutex

func CreateTopicPost(entity *posts.Entity, topicEntity topics.Entity) error {
	lock := &topicSequenceLocks[entity.TopicId%topicSequenceLockShards]
	lock.Lock()
	defer lock.Unlock()

	postNo, err := topics.ReservePostSequence(entity.TopicId)
	if err != nil {
		return err
	}

	entity.PostNo = postNo
	if err := posts.Create(entity); err != nil {
		return err
	}

	SyncTopicPostStats(topicEntity, *entity, false)
	return nil
}

func SyncTopicPostStats(topicEntity topics.Entity, postEntity posts.Entity, isDelete bool) {
	userId := postEntity.UserId
	if isDelete {
		if err := topicUserStat.DecrementUserPost(topicEntity.Id, userId); err != nil {
			slog.Error("failed to decrement topic user post stat", "topicId", topicEntity.Id, "userId", userId, "err", err)
		}
	} else {
		if err := topicUserStat.IncrementUserPost(topicEntity.Id, userId); err != nil {
			slog.Error("failed to increment topic user post stat", "topicId", topicEntity.Id, "userId", userId, "err", err)
		}
	}

	list := topicUserStat.SyncTopicPosters(topicEntity.Id)
	filteredList := lo.Filter(list, func(userID uint64, _ int) bool {
		return userID != topicEntity.UserId
	})
	finalList := append([]uint64{topicEntity.UserId}, filteredList...)

	pList := lo.Map(finalList, func(userID uint64, _ int) topics.Poster {
		return topics.Poster{
			UserID: userID,
		}
	})

	if isDelete {
		lastPost, _ := posts.GetLastByTopicID(topicEntity.Id)
		if err := topics.DecrementPostFast(topicEntity.Id, pList, lastPost.Id, lastPost.CreatedAt); err != nil {
			slog.Error("failed to decrement topic post count", "topicId", topicEntity.Id, "err", err)
		}
	} else {
		if err := topics.IncrementPostFast(topicEntity.Id, pList, postEntity.Id, postEntity.CreatedAt); err != nil {
			slog.Error("failed to increment topic post count", "topicId", topicEntity.Id, "err", err)
		}
	}
}

// RebuildTopicPostStats recalculates derived topic and participant counters
// from the current active post set. It is intentionally absolute rather than
// incremental so restoring a topic cannot double-count existing replies.
func RebuildTopicPostStats(topicEntity topics.Entity, activePosts []*posts.Entity) error {
	if err := topicUserStat.DeleteByTopicID(topicEntity.Id); err != nil {
		return err
	}

	var lastPost *posts.Entity
	postCount := uint64(0)
	replyCount := uint64(0)
	for _, post := range activePosts {
		if post == nil || post.VisibilityStatus != posts.VisibilityActive {
			continue
		}
		postCount++
		if post.PostNo > 1 {
			replyCount++
			if err := topicUserStat.IncrementUserPost(topicEntity.Id, post.UserId); err != nil {
				return err
			}
		}
		if lastPost == nil || lastPost.CreatedAt.Before(post.CreatedAt) ||
			(lastPost.CreatedAt.Equal(post.CreatedAt) && lastPost.Id < post.Id) {
			lastPost = post
		}
	}

	lastPostID := uint64(0)
	lastPostedAt := time.Time{}
	if lastPost != nil {
		lastPostID = lastPost.Id
		lastPostedAt = lastPost.CreatedAt
	}

	activePosterIDs := topicUserStat.SyncTopicPosters(topicEntity.Id)
	posterIDs := make([]topics.Poster, 0, len(activePosterIDs)+1)
	posterIDs = append(posterIDs, topics.Poster{UserID: topicEntity.UserId})
	for _, userID := range activePosterIDs {
		if userID != topicEntity.UserId {
			posterIDs = append(posterIDs, topics.Poster{UserID: userID})
		}
	}
	return topics.ReplacePostStats(topicEntity.Id, postCount, replyCount, posterIDs, lastPostID, lastPostedAt)
}
