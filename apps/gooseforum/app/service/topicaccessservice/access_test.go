package topicaccessservice

import (
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
)

func TestCanViewPublicAndOwnerDraftTopics(t *testing.T) {
	public := topics.Entity{
		Id:               1,
		Status:           1,
		ProcessStatus:    topics.ProcessStatusNormal,
		VisibilityStatus: topics.VisibilityActive,
	}
	if !CanView(&public, 0) {
		t.Fatal("anonymous viewer should see a public topic")
	}

	draft := public
	draft.Id = 2
	draft.Status = 0
	draft.UserId = 42
	if CanView(&draft, 0) || CanView(&draft, 41) {
		t.Fatal("draft leaked to a non-owner")
	}
	if !CanView(&draft, 42) {
		t.Fatal("draft owner should retain normal read access")
	}
}

func TestCanViewRejectsPurgedTopics(t *testing.T) {
	entity := topics.Entity{
		Id:               3,
		UserId:           42,
		VisibilityStatus: topics.VisibilityUserDeleted,
		RetentionStatus:  topics.RetentionPurged,
	}
	if CanView(&entity, 42) {
		t.Fatal("purged topic must remain unavailable to its former owner")
	}
}
