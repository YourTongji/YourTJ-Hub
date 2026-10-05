package api

import (
	"fmt"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/agentEvents"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/agents"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/postRevisions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userActivities"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"gorm.io/gorm"
	"testing"
	"time"
)

func TestPendingEditApprovalUsesLastPublicMentionBaseline(t *testing.T) {
	t.Setenv("YOURTJ_AGENT_INSTANCE_ID", "review-test")
	t.Setenv("YOURTJ_AGENT_STREAM_EPOCH", "review-epoch")
	t.Setenv("YOURTJ_AGENT_PRODUCER_ENABLED", "true")
	conn := setupAdminTopicTestDB(t)
	if err := conn.AutoMigrate(&postRevisions.Entity{}, &agentEvents.Publication{}, &agentEvents.Intent{}, &agentEvents.Entity{}, &agents.Entity{}, &users.BlockEntity{}, &userActivities.Entity{}); err != nil {
		t.Fatal(err)
	}
	const source = uint64(98330001)
	topic := topics.Entity{Id: source, UserId: source, Status: 1, FirstPostId: source}
	human := users.EntityComplete{Id: source, Username: "review_source", Email: "review_source@example.invalid", ActorType: users.ActorTypeHuman}
	first := posts.Entity{Id: source, TopicId: source, PostNo: 1, UserId: source, Content: "topic"}
	reply := posts.Entity{Id: source + 1, TopicId: source, PostNo: 2, UserId: source, Content: "old @oldtarget plus @newtarget", ProcessStatus: posts.ProcessStatusPending}
	for _, entity := range []any{&human, &topic, &first, &reply, &postRevisions.Entity{PostId: reply.Id, Version: 1, Content: "old @oldtarget"}, &postRevisions.Entity{PostId: reply.Id, Version: 2, Content: reply.Content, ProcessStatus: posts.ProcessStatusPending}, &agentEvents.Publication{InstanceID: "review-test", PostID: reply.Id, Version: 1}, &userActivities.Entity{UserId: source, Action: int(userActivities.ActionComment), SubjectType: userActivities.SubjectPost, SubjectId: reply.Id}} {
		if err := conn.Create(entity).Error; err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() {
		conn.Unscoped().Where("id IN ?", []uint64{source, source + 1}).Delete(&posts.Entity{})
		conn.Unscoped().Delete(&topic)
		conn.Unscoped().Delete(&human)
		conn.Where("post_id = ?", reply.Id).Delete(&postRevisions.Entity{})
		conn.Where("instance_id = ?", "review-test").Delete(&agentEvents.Publication{})
	})
	startReviewActionEventBus(t)
	captured := reviewUpdatedEvents
	for len(captured) > 0 {
		<-captured
	}
	response := ReviewAction(component.BetterRequest[ReviewActionReq]{Params: ReviewActionReq{Kind: "post", Id: reply.Id, Approve: true}})
	if response.Data.Code != component.SUCCESS {
		t.Fatalf("approval failed: %#v", response)
	}
	select {
	case e := <-captured:
		if e.OldContent != "old @oldtarget" || e.NewContent != reply.Content {
			t.Fatalf("wrong public diff %#v", e)
		}
	case <-time.After(time.Second):
		t.Fatal("approval of an existing pending edit omitted its new human mentions")
	}
}

func TestAdminTopicStatusLocksPostsBeforeTopic(t *testing.T) {
	t.Setenv("YOURTJ_AGENT_INSTANCE_ID", "")
	t.Setenv("YOURTJ_AGENT_STREAM_EPOCH", "")
	conn := setupAdminTopicTestDB(t)
	_, firstID := seedAdminTopic(t, conn, 98331000)
	lockedPost := false
	callback := "test1042_post_before_topic"
	if err := conn.Callback().Query().After("gorm:query").Register(callback, func(tx *gorm.DB) {
		if tx.Statement.Table == "posts" {
			if _, ok := tx.Statement.Clauses["FOR"]; ok {
				lockedPost = true
			}
		}
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Callback().Query().Remove(callback); _ = conn.Callback().Update().Remove(callback) })
	if err := conn.Callback().Update().Before("gorm:update").Register(callback, func(tx *gorm.DB) {
		if tx.Statement.Table == "topics" && !lockedPost {
			_ = tx.AddError(fmt.Errorf("topic update acquired before source post locks"))
		}
	}); err != nil {
		t.Fatal(err)
	}
	result := EditTopic(component.BetterRequest[EditTopicReq]{Params: EditTopicReq{TopicId: 98331000, ProcessStatus: 1}})
	if result.Data.Code != component.SUCCESS {
		t.Fatalf("moderation did not lock source posts before topic %d: %#v", firstID, result)
	}
}
