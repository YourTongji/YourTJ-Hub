package publicationservice

import (
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/feedconfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/feed"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"gorm.io/gorm"
)

// Account closure remains authoritative after the 30-day raw fence is erased.
// Moderation can publish retained content without recreating author telemetry.
func capturePublicContributionTx(tx *gorm.DB, uid, topicID, postID uint64, kind string) error {
	if !feedconfig.Current().Metrics {
		return nil
	}
	eligible, err := users.FeedActorEligibleTx(tx, uid)
	if err != nil {
		return err
	}
	if !eligible {
		return nil
	}
	return feed.EventTx(tx, uid, topicID, postID, kind, true)
}
