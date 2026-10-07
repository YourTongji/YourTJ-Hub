// Package topicpolicyservice owns immediate topic interaction settings.
// Content revisions do not snapshot or overwrite these settings.
package topicpolicyservice

import (
	"context"
	"errors"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"gorm.io/gorm"
)

var (
	ErrAgentRepliesDisabled = errors.New("topic disables agent replies")
	ErrSettingDenied        = errors.New("topic reply setting cannot be changed")
)

// CheckReplyTx requires the caller to hold the topic write lock until commit.
// The setting applies to bot identities regardless of transport or role.
func CheckReplyTx(tx *gorm.DB, topic topics.Entity, authorID uint64) error {
	if !topic.AgentRepliesDisabled {
		return nil
	}
	author, err := users.GetActorTx(tx, authorID)
	if err != nil {
		return err
	}
	if author.IsBot() {
		return ErrAgentRepliesDisabled
	}
	return nil
}

func SetAgentRepliesDisabled(ctx context.Context, topicID, actorID uint64, disabled bool) (topic topics.Entity, err error) {
	err = db.ConnectContext(ctx).Transaction(func(tx *gorm.DB) error {
		var readErr error
		topic, readErr = topics.GetForUpdateTx(tx, topicID)
		if readErr != nil {
			return readErr
		}
		if topic.UserId != actorID || topic.TopicType != topics.TopicTypeForum || topic.VisibilityStatus != topics.VisibilityActive {
			return ErrSettingDenied
		}
		return topics.UpdateAgentRepliesDisabledTx(tx, topicID, disabled)
	})
	return
}
