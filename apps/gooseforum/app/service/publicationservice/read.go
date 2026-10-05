package publicationservice

import (
	"encoding/json"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/postRevisions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
)

// OwnerSnapshot is private projection only. Public caches and persisted rows
// must never receive these copies. Rejected candidates appear only in management.
func OwnerSnapshot(topic *topics.Entity, post *posts.Entity, userID uint64, includeBlocked bool) postRevisions.Entity {
	if userID == 0 || post.UserId != userID || post.VisibilityStatus != posts.VisibilityActive || topic.VisibilityStatus != topics.VisibilityActive || post.LatestRevisionId == 0 || post.LatestRevisionId == post.PublishedRevisionId {
		return postRevisions.Entity{}
	}
	revision := postRevisions.Get(post.LatestRevisionId)
	visible := revision.ProcessStatus == posts.ProcessStatusPending || (includeBlocked && revision.ProcessStatus == posts.ProcessStatusBlocked)
	if revision.PostId != post.Id || !visible {
		return postRevisions.Entity{}
	}
	ApplySnapshot(topic, post, revision)
	return revision
}

// Checking includes queued and leased work and remains valid after restart.
func Checking(revisionIDs []uint64) map[uint64]bool {
	payloads := make([]string, 0, len(revisionIDs))
	for _, id := range revisionIDs {
		raw, err := json.Marshal(Task{RevisionId: id})
		if err != nil {
			continue
		}
		payloads = append(payloads, string(raw))
	}
	result := map[uint64]bool{}
	for _, raw := range taskQueue.ActivePayloads(TaskType, payloads) {
		var task Task
		if json.Unmarshal([]byte(raw), &task) == nil {
			result[task.RevisionId] = true
		}
	}
	return result
}
