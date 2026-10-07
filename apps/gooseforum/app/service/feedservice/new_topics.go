package feedservice

import (
	"errors"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/feedconfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/feed"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func newTopicMetric(tx *gorm.DB, row feed.NewTopicOutcome, name string, delta int64) error {
	return addMetric(tx, feed.Serve{CreatedAt: row.PublicAt, Feed: "new_topics", Hash: row.Hash, RankHash: row.RankHash, Variant: "all", WeightVariant: "all", Capability: "all"}, name, delta)
}

func newTopicCohortTx(tx *gorm.DB, topic topics.Entity) (*feed.NewTopicOutcome, error) {
	if topic.FirstPublicAt == nil || topic.FirstPublicEstimated || topic.FirstPublicAt.After(time.Now()) || time.Since(*topic.FirstPublicAt) >= 30*24*time.Hour {
		return nil, nil
	}
	if err := feed.LockOwnerTx(tx, topic.UserId); errors.Is(err, feed.ErrClosed) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	cfg := feedconfig.Current()
	row := feed.NewTopicOutcome{TopicID: topic.Id, UserID: topic.UserId, PublicAt: *topic.FirstPublicAt, ExpiresAt: topic.FirstPublicAt.Add(30 * 24 * time.Hour), Hash: cfg.Hash, RankHash: cfg.RankHash}
	r := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&row)
	if r.Error != nil {
		return nil, r.Error
	}
	if r.RowsAffected == 1 {
		if err := newTopicMetric(tx, row, "new_topic_public", 1); err != nil {
			return nil, err
		}
	}
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&row, "topic_id = ? AND expires_at > ?", topic.Id, time.Now()).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

// The denominator is actual new publication, never historical estimated time.
// Unique visibility within 24 hours and first peer response are cohort metrics,
// independent of entry assignment; they do not assert an experimental effect.
func captureNewTopicPublicationTx(tx *gorm.DB, event feed.Event) error {
	rows, err := topics.NewTopicFactsTx(tx, []uint64{event.TopicID})
	if err != nil {
		return err
	}
	for _, topic := range rows {
		row, err := newTopicCohortTx(tx, topic)
		if err != nil {
			return err
		}
		if row == nil || row.Replied || event.Kind != "public_reply" {
			continue
		}
		at, err := posts.FirstPeerReplyTx(tx, topic.Id, topic.UserId)
		if err != nil {
			return err
		}
		if at == nil || at.Before(row.PublicAt) {
			continue
		}
		if err := newTopicMetric(tx, *row, "new_topic_first_reply", 1); err != nil {
			return err
		}
		if err := newTopicMetric(tx, *row, "first_reply_seconds", int64(at.Sub(row.PublicAt).Seconds())); err != nil {
			return err
		}
		if err := tx.Model(row).UpdateColumn("replied", true).Error; err != nil {
			return err
		}
	}
	return nil
}

func captureNewTopicVisibilityTx(tx *gorm.DB, ids []uint64, viewer uint64, visibleAt time.Time) error {
	if len(ids) == 0 {
		return nil
	}
	// Almost all scrolling rows already have a first visibility watermark.
	// Batch those checks before any owner lock or publication metadata read.
	var existing []feed.NewTopicOutcome
	if err := tx.Where("topic_id IN ? AND visible = ? AND expires_at > ?", ids, true, time.Now()).Find(&existing).Error; err != nil {
		return err
	}
	seen := map[uint64]bool{}
	for _, row := range existing {
		seen[row.TopicID] = true
	}
	pending := []uint64{}
	for _, id := range ids {
		if !seen[id] {
			pending = append(pending, id)
		}
	}
	if len(pending) == 0 {
		return nil
	}
	rows, err := topics.NewTopicFactsTx(tx, pending)
	if err != nil {
		return err
	}
	for _, topic := range rows {
		// Old topics need no cohort lookup or owner lock on a scrolling request.
		if topic.UserId == viewer || topic.FirstPublicAt == nil || visibleAt.Before(*topic.FirstPublicAt) || visibleAt.Sub(*topic.FirstPublicAt) >= 24*time.Hour {
			continue
		}
		row, err := newTopicCohortTx(tx, topic)
		if err != nil {
			return err
		}
		if row == nil || row.Visible {
			continue
		}
		if err := newTopicMetric(tx, *row, "new_topic_visible_24h", 1); err != nil {
			return err
		}
		if err := tx.Model(row).UpdateColumn("visible", true).Error; err != nil {
			return err
		}
	}
	return nil
}
