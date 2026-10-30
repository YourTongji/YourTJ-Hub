package publicationservice

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/postRevisions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/fileusageservice"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrReviewReason = errors.New("a rejection explanation of 1 to 512 characters is required")

// NormalizeRejectionReason bounds author-visible plain text. Model evidence and
// private moderation notes are not copied into this field automatically.
func NormalizeRejectionReason(reason string) (string, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" || utf8.RuneCountInString(reason) > 512 {
		return "", ErrReviewReason
	}
	return reason, nil
}

// RecordLegacyRejectionTx retains the decision for the compatibility review path
// using the existing immutable revision and appeal model. The caller updates the
// blocked status in this same transaction; any later failure rolls back both.
func RecordLegacyRejectionTx(tx *gorm.DB, postID, actorID uint64, reason string) (uint64, error) {
	normalized, err := NormalizeRejectionReason(reason)
	if err != nil {
		return 0, err
	}
	var post posts.Entity
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&post, postID).Error; err != nil {
		return 0, err
	}
	var topic topics.Entity
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&topic, post.TopicId).Error; err != nil {
		return 0, err
	}
	if post.LatestRevisionId != 0 || post.ProcessStatus != posts.ProcessStatusPending || !available(topic, post) || (topic.TopicType == topics.TopicTypeWiki && post.PostNo == 1) {
		return 0, ErrUnavailable
	}
	revision := snapshot(&topic, &post)
	revision.Version = postRevisions.NextVersionTx(tx, post.Id)
	revision.ProcessStatus = posts.ProcessStatusBlocked
	revision.ReviewReason = normalized
	revision.ReviewActorId = actorID
	now := time.Now()
	revision.ReviewedAt = &now
	if err := tx.Create(&revision).Error; err != nil {
		return 0, err
	}
	if err := fileusageservice.RetainLegacyRevisionImagesTx(tx, revision.Id, topic.Id, post.Id, post.UserId, post.PostNo == 1); err != nil {
		return 0, err
	}
	if err := tx.Model(&post).Update("latest_revision_id", revision.Id).Error; err != nil {
		return 0, err
	}
	return revision.Id, nil
}
