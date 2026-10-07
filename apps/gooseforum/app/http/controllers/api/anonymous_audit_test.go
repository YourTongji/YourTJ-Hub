package api

import (
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/moderationDecision"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/moderationLog"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/optRecord"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/moderationservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/optlogger"
)

func TestDeletedAnonymousTopicAuditsNeverRestoreOwner(t *testing.T) {
	conn := setupAdminTopicTestDB(t)
	const topicID = uint64(9891001)
	owner, _ := seedAdminTopic(t, conn, topicID)
	if err := conn.Model(&topics.Entity{}).Where("id = ?", topicID).Update("persona_uid", "0123456789abcdef0123456789abcdef").Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Delete(&topics.Entity{}, topicID).Error; err != nil {
		t.Fatal(err)
	}
	item := aiDecisionItem(moderationDecision.Entity{SubjectType: "topic", SubjectId: topicID, AuthorId: owner})
	if item.AuthorId != 0 {
		t.Fatalf("AI audit exposed owner %d", item.AuthorId)
	}
	moderationservice.TopicDeleted(owner, moderationservice.TopicDeletedSnapshot{TopicId: topicID, DeletedBy: owner, DeletedByUser: "private-owner"})
	var log moderationLog.Entity
	if err := conn.Where("subject_id = ?", topicID).First(&log).Error; err != nil {
		t.Fatal(err)
	}
	if log.ActorUserId != 0 || log.Payload.Params["deletedBy"] != nil || log.Payload.Params["deletedByUser"] != nil {
		t.Fatalf("public audit exposed self-delete actor: %+v", log)
	}
	optlogger.UserOpt(owner, optlogger.EditTopic, topicID, "delete")
	var opt optRecord.Entity
	if err := conn.Where("target_id = ?", topicID).First(&opt).Error; err != nil {
		t.Fatal(err)
	}
	if opt.OptUserId != 0 {
		t.Fatalf("operation audit exposed owner %d", opt.OptUserId)
	}
}
