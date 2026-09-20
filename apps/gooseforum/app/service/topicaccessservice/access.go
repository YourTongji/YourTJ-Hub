// Package topicaccessservice owns the shared topic read-visibility policy.
package topicaccessservice

import (
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/moderationservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/permission"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/userservice"
)

// CanView reports whether viewerID may read entity through the normal topic
// detail path. Keeping this policy outside HTTP controllers lets auxiliary read
// paths, such as link previews, reuse the same authorization decision.
func CanView(entity *topics.Entity, viewerID uint64) bool {
	if entity == nil || entity.Id == 0 {
		return false
	}
	if entity.VisibilityStatus != topics.VisibilityActive {
		return canViewDeleted(entity, viewerID)
	}
	if entity.Status != 1 {
		return viewerID != 0 && viewerID == entity.UserId
	}
	if entity.ProcessStatus != topics.ProcessStatusNormal &&
		!canViewProcessed(viewerID) &&
		!moderationservice.CanModerateAnyCategory(viewerID, entity.CategoryIds) {
		return false
	}
	return true
}

func canViewDeleted(entity *topics.Entity, viewerID uint64) bool {
	if entity.RetentionStatus == topics.RetentionPurged {
		return false
	}
	if entity.VisibilityStatus == topics.VisibilityAccountAnonymized ||
		entity.VisibilityStatus == topics.VisibilityModeratorRemoved {
		return moderationservice.CanModerateAnyCategory(viewerID, entity.CategoryIds)
	}
	if viewerID > 0 && viewerID == entity.UserId {
		return true
	}
	if moderationservice.CanModerateAnyCategory(viewerID, entity.CategoryIds) {
		return true
	}
	// Preserve the existing topic-detail behavior: when an author removes the
	// first post, normal replies may keep the discussion context readable.
	return len(posts.GetByTopicPostNoAfter(entity.Id, 1, 1)) > 0
}

func canViewProcessed(viewerID uint64) bool {
	if viewerID == 0 {
		return false
	}
	roleID, ok := userservice.GetUserRoleId(viewerID)
	return ok && permission.CheckRole(roleID, permission.TopicsManager)
}
