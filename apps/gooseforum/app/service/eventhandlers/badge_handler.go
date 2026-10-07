package eventhandlers

import (
	"context"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/badgeservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/userservice"
)

func handleBadgePost(ctx context.Context, event *TopicPublishedEvent) error {
	if event.Topic != nil && event.Topic.PersonaUID != "" {
		return nil
	}
	_, userID, _ := event.Subject()
	if userID == 0 {
		return nil
	}
	checkAndInvalidateUserBadges(userID, badgeservice.TriggerPost)
	return nil
}

func checkAndInvalidateUserBadges(userID uint64, trigger badgeservice.Trigger) {
	before := len(badgeservice.GetUserBadges(userID))
	badgeservice.CheckAndGrant(userID, trigger)
	if len(badgeservice.GetUserBadges(userID)) != before {
		userservice.InvalidateUserPublicProfileCache(userID)
	}
}

func handleBadgeComment(ctx context.Context, event *CommentCreatedEvent) error {
	if event.PersonaUID != "" {
		return nil
	}
	checkAndInvalidateUserBadges(event.UserId, badgeservice.TriggerComment)
	return nil
}

func handleBadgeLike(ctx context.Context, event *TopicLikedEvent) error {
	if topics.GetSimple(event.TopicId).PersonaUID != "" {
		return nil
	}
	checkAndInvalidateUserBadges(event.LikerId, badgeservice.TriggerLike)
	checkAndInvalidateUserBadges(event.UserId, badgeservice.TriggerLike)
	return nil
}

func handleBadgeFollow(ctx context.Context, event *UserFollowedEvent) error {
	checkAndInvalidateUserBadges(event.UserId, badgeservice.TriggerFollow)
	return nil
}
