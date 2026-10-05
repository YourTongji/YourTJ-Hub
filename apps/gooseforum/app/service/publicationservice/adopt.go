package publicationservice

import (
	"context"
	"errors"
	"gorm.io/gorm"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
)

// AdoptPending upgrades active legacy review entries once. Already versioned
// rows are skipped, including after a partially completed migration is retried.
func AdoptPending(ctx context.Context) error {
	var cursor uint64
	for {
		var batch []posts.Entity
		if err := db.ConnectContext(ctx).Where("id > ? AND latest_revision_id = 0 AND process_status = ? AND visibility_status = ?", cursor, posts.ProcessStatusPending, posts.VisibilityActive).Order("id").Limit(100).Find(&batch).Error; err != nil {
			return err
		}
		if len(batch) == 0 {
			return nil
		}
		for _, post := range batch {
			cursor = post.Id
			var topic topics.Entity
			if err := db.ConnectContext(ctx).First(&topic, post.TopicId).Error; errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			} else if err != nil {
				return err
			}
			if !available(topic, post) || (post.PostNo > 1 && topic.ProcessStatus != topics.ProcessStatusNormal) {
				continue
			}
			if err := Submit(ctx, &topic, &post); err != nil {
				return err
			}
		}
	}
}
