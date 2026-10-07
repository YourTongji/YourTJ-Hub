// Package publicationservice owns versioned forum submissions and their review.
// Public rows remain the approved projection; post_revisions owns candidate content.
package publicationservice

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/markdown2html"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/feed"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/postRevisions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topicCategoryIndex"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/agenteventservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/fileusageservice"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const TaskType = "content-review"

var ErrUnavailable = errors.New("content is no longer available for review")

type Task struct {
	RevisionId uint64 `json:"revisionId"`
}

// Submit commits the immutable submission, private attachments and durable job
// together. An approved public version is never overwritten by a candidate.
// topic contains proposed metadata for a first post, existing metadata for replies.
func Submit(ctx context.Context, topic *topics.Entity, post *posts.Entity) error {
	return SubmitWithHooks(ctx, topic, post, nil, nil)
}

// SubmitWithHooks keeps Agent credential/policy authorization, idempotency and
// causal metadata in the same transaction as the pending revision and review
// task. The before hook runs before any insertion; either hook can roll back all
// submission effects. Model evaluation remains exclusively in the worker.
func SubmitWithHooks(ctx context.Context, topic *topics.Entity, post *posts.Entity, before, after func(*gorm.DB) error) error {
	return db.ConnectContext(ctx).Transaction(func(tx *gorm.DB) error {
		if before != nil {
			if err := before(tx); err != nil {
				return err
			}
		}
		if topic.Id == 0 {
			topic.ProcessStatus = topics.ProcessStatusPending
			topic.PostSeq, topic.PostCount = 1, 0
			if err := topics.CreateTx(tx, topic); err != nil {
				return err
			}
			post.TopicId, post.PostNo = topic.Id, 1
		}
		var live posts.Entity
		if post.Id != 0 {
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&live, post.Id).Error; err != nil {
				return err
			}
			if live.UserId != post.UserId || live.VisibilityStatus != posts.VisibilityActive {
				return ErrUnavailable
			}
		}
		var liveTopic topics.Entity
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&liveTopic, topic.Id).Error; err != nil {
			return err
		}
		if liveTopic.VisibilityStatus != topics.VisibilityActive {
			return ErrUnavailable
		}
		if post.PostNo != 1 && (liveTopic.Status != 1 || liveTopic.ProcessStatus != topics.ProcessStatusNormal) {
			return ErrUnavailable
		}
		if post.Id == 0 {
			if post.PostNo != 1 {
				no, err := topics.ReservePostSequenceTx(tx, topic.Id)
				if err != nil {
					return err
				}
				post.PostNo = no
			}
			post.ProcessStatus = posts.ProcessStatusPending
			if err := posts.CreateTx(tx, post); err != nil {
				return err
			}
			live = *post
			if post.PostNo == 1 {
				topic.FirstPostId = post.Id
				if err := topics.SaveTx(tx, topic); err != nil {
					return err
				}
			}
		} else if live.LatestRevisionId == 0 {
			// Lazy baseline uses the current row, never MAX(version): legacy history may
			// contain rejected bodies and does not have complete topic metadata.
			baseline := snapshot(&liveTopic, &live)
			baseline.Version = postRevisions.NextVersionTx(tx, post.Id)
			if err := postRevisions.CreateTx(tx, &baseline); err != nil {
				return err
			}
			live.LatestRevisionId = baseline.Id
			if live.ProcessStatus == posts.ProcessStatusNormal && liveTopic.Status == 1 && liveTopic.ProcessStatus == topics.ProcessStatusNormal {
				live.PublishedRevisionId = baseline.Id
			}
		}
		// Establish a legacy event baseline while the previous projection is still
		// provably public. A draft can have normal revisions but has never produced
		// a public occurrence. Do not infer a baseline later from revision status.
		if live.PublishedRevisionId != 0 {
			if _, _, err := agenteventservice.PublicSourceTx(tx, live.Id); err == nil {
				var published postRevisions.Entity
				if err := tx.Where("id = ? AND post_id = ?", live.PublishedRevisionId, live.Id).Take(&published).Error; err != nil {
					return err
				}
				if err := agenteventservice.BaselineTx(tx, live.Id, published.Version); err != nil {
					return err
				}
			} else if !errors.Is(err, agenteventservice.ErrInaccessible) && !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
		}
		preserveLive := live.ProcessStatus == posts.ProcessStatusNormal && liveTopic.Status == 1 && liveTopic.ProcessStatus == topics.ProcessStatusNormal

		revision := snapshot(topic, post)
		revision.Version = postRevisions.NextVersionTx(tx, post.Id)
		revision.ProcessStatus = posts.ProcessStatusPending
		if err := postRevisions.CreateTx(tx, &revision); err != nil {
			return err
		}
		now := time.Now()
		changes := map[string]any{"latest_revision_id": revision.Id, "published_revision_id": live.PublishedRevisionId, "last_editor_id": post.UserId, "last_edited_at": now}
		if !preserveLive {
			if err := fileusageservice.PrivatizeUnpublishedImagesTx(tx, topic.Id, post.Id, post.PostNo == 1); err != nil {
				return err
			}
			changes["content"], changes["rendered_html"], changes["rendered_version"], changes["content_type"], changes["process_status"] = post.Content, post.RenderedHTML, post.RenderedVersion, post.ContentType, posts.ProcessStatusPending
			if post.PostNo == 1 {
				topic.ProcessStatus = topics.ProcessStatusPending
				if err := topics.UpdateTopicEditableTx(tx, topic); err != nil {
					return err
				}
				if err := topicCategoryIndex.ReplaceTopicCategoriesTx(tx, topic.Id, topic.CategoryIds); err != nil {
					return err
				}
			}
		}
		if err := tx.Model(&posts.Entity{}).Where("id = ?", post.Id).Updates(changes).Error; err != nil {
			return err
		}
		if err := fileusageservice.RegisterRevisionImagesTx(tx, revision.Id, post.UserId, post.Content, revision.ImageUrls); err != nil {
			return err
		}
		payload, err := json.Marshal(Task{RevisionId: revision.Id})
		if err != nil {
			return err
		}
		if err := taskQueue.CreateTx(tx, &taskQueue.Entity{Type: TaskType, TaskJson: string(payload)}); err != nil {
			return err
		}
		post.LatestRevisionId, post.PublishedRevisionId, post.ProcessStatus = revision.Id, live.PublishedRevisionId, posts.ProcessStatusPending
		post.LastEditorId, post.LastEditedAt = post.UserId, &now
		if live.PublishedRevisionId == 0 {
			if err := feed.EventTx(tx, post.UserId, topic.Id, post.Id, "intent", true); err != nil {
				return err
			}
		}
		if after != nil {
			return after(tx)
		}
		return nil
	})
}

func snapshot(topic *topics.Entity, post *posts.Entity) postRevisions.Entity {
	r := postRevisions.Entity{PostId: post.Id, Version: 1, EditorId: post.UserId, Content: post.Content, RenderedHTML: post.RenderedHTML, ProcessStatus: post.ProcessStatus, ContentType: post.ContentType}
	if post.PostNo == 1 {
		r.Title, r.CategoryIds, r.ImageUrls = topic.Title, topic.CategoryIds, topic.ImageUrls
	}
	return r
}

// ApplySnapshot only mutates caller-owned copies (including cached list entries).
func ApplySnapshot(topic *topics.Entity, post *posts.Entity, revision postRevisions.Entity) {
	post.Content, post.RenderedHTML, post.RenderedVersion, post.ContentType, post.ProcessStatus = revision.Content, revision.RenderedHTML, markdown2html.GetPostVersion(), revision.ContentType, revision.ProcessStatus
	if post.PostNo == 1 {
		topic.Title, topic.CategoryIds, topic.ImageUrls, topic.ProcessStatus = revision.Title, revision.CategoryIds, revision.ImageUrls, revision.ProcessStatus
		topic.Excerpt = markdown2html.ExtractDescription(revision.Content, 200)
		topic.FirstImageURL = markdown2html.ExtractFirstImageURL(revision.Content)
		if topic.FirstImageURL == "" && len(topic.ImageUrls) > 0 {
			topic.FirstImageURL = topic.ImageUrls[0]
		}
	}
}
