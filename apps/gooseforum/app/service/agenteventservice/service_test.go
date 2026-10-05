package agenteventservice

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/agentEvents"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/agents"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/postRevisions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"gorm.io/gorm"
)

func setup(t *testing.T) (*gorm.DB, posts.Entity, agents.Entity) {
	t.Helper()
	t.Setenv("YOURTJ_AGENT_INSTANCE_ID", "test-instance")
	t.Setenv("YOURTJ_AGENT_STREAM_EPOCH", "epoch-1")
	t.Setenv("YOURTJ_AGENT_PRODUCER_ENABLED", "true")
	t.Setenv("YOURTJ_AGENT_API_ENABLED", "true")
	conn := db.Connect()
	if err := conn.AutoMigrate(&users.EntityComplete{}, &users.BlockEntity{}, &agents.Entity{}, &topics.Entity{}, &posts.Entity{}, &postRevisions.Entity{}, &taskQueue.Entity{}, &agentEvents.ReplayState{}, &agentEvents.Publication{}, &agentEvents.Intent{}, &agentEvents.Entity{}); err != nil {
		t.Fatal(err)
	}
	base := uint64(time.Now().UnixNano()%100000000) + 900000000
	human := users.EntityComplete{Id: base, Username: fmt.Sprintf("human%d", base), Email: fmt.Sprintf("h%d@invalid.example", base), ActorType: users.ActorTypeHuman}
	bot := users.EntityComplete{Id: base + 1, Username: fmt.Sprintf("bot%d", base), Email: fmt.Sprintf("b%d@invalid.example", base), ActorType: users.ActorTypeBot}
	now := time.Now().Add(-time.Hour)
	a := agents.Entity{UserId: bot.Id, TokenPrefix: fmt.Sprintf("agt%d", base), Enabled: 1, EventsEnabled: true, EventsEnabledAt: &now, SubscriptionGeneration: 1, EventTypes: `["agent.mentioned","agent.post_replied","agent.topic_commented"]`}
	topic := topics.Entity{Id: base, UserId: bot.Id, Status: 1, FirstPostId: base, PostSeq: 2}
	first := posts.Entity{Id: base, TopicId: topic.Id, PostNo: 1, UserId: bot.Id, Content: "bot first"}
	p := posts.Entity{Id: base + 1, TopicId: topic.Id, PostNo: 2, UserId: human.Id, ReplyToPostId: first.Id, Content: "hello @" + bot.Username}
	for _, value := range []any{&human, &bot, &a, &topic, &first, &p, &postRevisions.Entity{PostId: p.Id, Version: 1, EditorId: human.Id, Content: p.Content}} {
		if err := conn.Create(value).Error; err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() {
		conn.Where("instance_id = ?", "test-instance").Delete(&agentEvents.Entity{})
		conn.Where("instance_id = ?", "test-instance").Delete(&agentEvents.Intent{})
		conn.Where("instance_id = ?", "test-instance").Delete(&agentEvents.ReplayState{}, &agentEvents.Publication{})
		conn.Where("type = ?", TaskType).Delete(&taskQueue.Entity{})
		conn.Where("post_id = ?", p.Id).Delete(&postRevisions.Entity{})
		conn.Unscoped().Where("topic_id = ?", topic.Id).Delete(&posts.Entity{})
		conn.Unscoped().Delete(&topic)
		conn.Where("user_id = ?", a.UserId).Delete(&agents.Entity{})
		conn.Unscoped().Where("id IN ?", []uint64{human.Id, bot.Id}).Delete(&users.EntityComplete{})
	})
	return conn, p, a
}
func materialize(t *testing.T, conn *gorm.DB, p posts.Entity) agentEvents.Entity {
	t.Helper()
	if err := conn.Transaction(func(tx *gorm.DB) error { return CapturePublicTx(tx, &p) }); err != nil {
		t.Fatal(err)
	}
	var i agentEvents.Intent
	if err := conn.Where("post_id = ?", p.Id).Take(&i).Error; err != nil {
		t.Fatal(err)
	}
	task, claimed, err := taskQueue.ClaimTask(i.TaskID)
	if err != nil || !claimed {
		t.Fatalf("claim %t %v", claimed, err)
	}
	if err := HandleTask(context.Background(), &task); err != nil {
		t.Fatal(err)
	}
	if err := HandleTask(context.Background(), &task); err != nil {
		t.Fatal(err)
	}
	var e agentEvents.Entity
	if err := conn.Where("source_intent_id = ?", i.ID).Take(&e).Error; err != nil {
		t.Fatal(err)
	}
	return e
}
func TestCaptureRollbackAndStablePriorityReplay(t *testing.T) {
	conn, p, a := setup(t)
	sentinel := errors.New("rollback")
	err := conn.Transaction(func(tx *gorm.DB) error {
		if err := CapturePublicTx(tx, &p); err != nil {
			return err
		}
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatal(err)
	}
	var count int64
	conn.Model(&agentEvents.Intent{}).Where("post_id = ?", p.Id).Count(&count)
	if count != 0 {
		t.Fatal("intent escaped rollback")
	}
	e := materialize(t, conn, p)
	if e.Type != "agent.post_replied" || len(e.Reasons) != 3 || e.Seq != 1 {
		t.Fatalf("event %#v", e)
	}
	if err := conn.Transaction(func(tx *gorm.DB) error { return CapturePublicTx(tx, &p) }); err != nil {
		t.Fatal(err)
	}
	conn.Model(&agentEvents.Entity{}).Where("agent_id = ?", a.UserId).Count(&count)
	if count != 1 {
		t.Fatalf("duplicate %d", count)
	}
	page, err := List(a.UserId, "", 10)
	if err != nil || len(page.Events) != 1 || page.Events[0].ID != e.ID {
		t.Fatalf("page %#v %v", page, err)
	}
	if page.Events[0].AckedAt != nil {
		t.Fatal("read automatically ACKed")
	}
	if err := Ack(a.UserId, []string{e.ID}); err != nil {
		t.Fatal(err)
	}
	if err := Ack(a.UserId, []string{e.ID}); err != nil {
		t.Fatal(err)
	}
	if err := Ack(a.UserId+1, []string{e.ID}); err == nil {
		t.Fatal("cross-Agent ACK accepted")
	}
	t.Setenv("YOURTJ_AGENT_STREAM_EPOCH", "epoch-2")
	_, err = List(a.UserId, page.NextCursor, 10)
	var cursorError *Error
	if !errors.As(err, &cursorError) || cursorError.Code != "cursor_reset" {
		t.Fatalf("old epoch %v", err)
	}
}
func TestFrozenGenerationAndVisibilityTombstone(t *testing.T) {
	conn, p, a := setup(t)
	e := materialize(t, conn, p)
	if err := conn.Model(&posts.Entity{}).Where("id = ?", p.Id).Update("process_status", posts.ProcessStatusPending).Error; err != nil {
		t.Fatal(err)
	}
	got, err := Get(a.UserId, e.ID)
	if err != nil || got.State != "withdrawn" || got.Data != nil {
		t.Fatalf("leak %#v %v", got, err)
	}
	if err := Ack(a.UserId, []string{e.ID}); err == nil {
		t.Fatal("withdrawn ACK accepted")
	}
	var stored agentEvents.Entity
	conn.First(&stored, "id = ?", e.ID)
	if stored.ActorID != 0 || stored.PostID != 0 {
		t.Fatalf("retained private references %#v", stored)
	}
}
func TestLateIntentDoesNotAdoptNewSubscription(t *testing.T) {
	conn, p, a := setup(t)
	if err := conn.Transaction(func(tx *gorm.DB) error { return CapturePublicTx(tx, &p) }); err != nil {
		t.Fatal(err)
	}
	if err := agents.UpdateColumns(conn, a.UserId, map[string]any{"subscription_generation": 2}); err != nil {
		t.Fatal(err)
	}
	var i agentEvents.Intent
	conn.Where("post_id = ?", p.Id).Take(&i)
	task, claimed, err := taskQueue.ClaimTask(i.TaskID)
	if err != nil || !claimed {
		t.Fatal(err)
	}
	if err := HandleTask(context.Background(), &task); err != nil {
		t.Fatal(err)
	}
	var count int64
	conn.Model(&agentEvents.Entity{}).Where("source_intent_id = ?", i.ID).Count(&count)
	if count != 0 {
		t.Fatal("late intent adopted changed subscription")
	}
}
func TestSharedMentionBudgetAndMarkdownExclusion(t *testing.T) {
	conn, p, a := setup(t)
	ids, err := users.ResolveMentionIDsTx(conn, "`@"+fmt.Sprintf("bot%d", p.UserId)+"` @"+fmt.Sprintf("bot%d", p.UserId), p.UserId, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 1 || ids[0] != a.UserId {
		t.Fatalf("mention IDs %v", ids)
	}
}

func TestReplayFloorKeepsEarlierCommittedRetainedEvents(t *testing.T) {
	conn, p, a := setup(t)
	e := materialize(t, conn, p)
	older := e
	older.ID = "evt_older_occurrence"
	older.Seq = 2
	older.SourceIntentID = "src_older_occurrence"
	older.ExpiresAt = time.Now().UTC().Add(-time.Hour)
	if err := conn.Create(&older).Error; err != nil {
		t.Fatal(err)
	}
	page, err := List(a.UserId, encodeCursor("test-instance", "epoch-1", a.UserId, 0), 20)
	if err != nil || len(page.Events) != 2 || page.Events[0].ID != e.ID {
		t.Fatalf("late older occurrence skipped a retained event: %#v %v", page, err)
	}
	if err := conn.Model(&agents.Entity{}).Where("user_id = ?", a.UserId).Update("token_hash", "current").Error; err != nil {
		t.Fatal(err)
	}
	if _, err := ListWithCredential(a.UserId, "", 20, "old"); err == nil {
		t.Fatal("rotated token could read")
	}
	if err := AckWithCredential(a.UserId, []string{e.ID}, "old"); err == nil {
		t.Fatal("rotated token could ACK")
	}
	if err := AckWithCredential(a.UserId, []string{e.ID}, "current"); err != nil {
		t.Fatal(err)
	}
}
func TestPublicVersionEditApprovalAndRepublish(t *testing.T) {
	conn, p, a := setup(t)
	// First public v1 is materialized. The pending v2 must not advance its
	// publication watermark, and approval should emit only newly added mentions.
	first := materialize(t, conn, p)
	p.Content = "no mentions"
	p.ProcessStatus = posts.ProcessStatusPending
	if err := conn.Save(&p).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Create(&postRevisions.Entity{PostId: p.Id, Version: 2, Content: p.Content, ProcessStatus: posts.ProcessStatusPending}).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Transaction(func(tx *gorm.DB) error { return CapturePublicTx(tx, &p) }); err != nil {
		t.Fatal(err)
	}
	state, err := agentEvents.PublicationTx(conn, "test-instance", p.Id)
	if err != nil || state.Version != 1 {
		t.Fatalf("pending advanced public watermark %#v %v", state, err)
	}
	p.ProcessStatus = posts.ProcessStatusNormal
	if err := conn.Save(&p).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Transaction(func(tx *gorm.DB) error { return CapturePublicTx(tx, &p) }); err != nil {
		t.Fatal(err)
	}
	p.Content = fmt.Sprintf("@bot%d", p.UserId)
	if err := conn.Save(&p).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Create(&postRevisions.Entity{PostId: p.Id, Version: 3, Content: p.Content}).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Transaction(func(tx *gorm.DB) error { return CapturePublicTx(tx, &p) }); err != nil {
		t.Fatal(err)
	}
	var i agentEvents.Intent
	if err := conn.Where("post_id = ? AND version = 3", p.Id).Take(&i).Error; err != nil {
		t.Fatal(err)
	}
	task, claimed, err := taskQueue.ClaimTask(i.TaskID)
	if err != nil || !claimed {
		t.Fatal(err)
	}
	if err := HandleTask(context.Background(), &task); err != nil {
		t.Fatal(err)
	}
	page, err := List(a.UserId, "", 20)
	if err != nil || len(page.Events) != 2 {
		t.Fatalf("events %#v %v", page, err)
	}
	if page.Events[1].Type != "agent.mentioned" || len(page.Events[1].Data.Reasons) != 1 || page.Events[1].ID == first.ID {
		t.Fatalf("readd duplicated creation reasons %#v", page.Events[1])
	}
	// Reapproving/republishing the same immutable revision creates no occurrence.
	if err := conn.Transaction(func(tx *gorm.DB) error { return CapturePublicTx(tx, &p) }); err != nil {
		t.Fatal(err)
	}
	var count int64
	conn.Model(&agentEvents.Intent{}).Where("post_id = ?", p.Id).Count(&count)
	if count != 2 {
		t.Fatalf("same revision repeated %d", count)
	}
}

func TestWithdrawCopiesAtomicAndExpiry(t *testing.T) {
	conn, p, a := setup(t)
	e := materialize(t, conn, p)
	called := false
	RegisterWithdrawalHook(func(tx *gorm.DB, instance string, ids []string) error {
		called = true
		if instance != "test-instance" || len(ids) != 1 || ids[0] != e.ID {
			t.Fatalf("wrong copied event ids %s %v", instance, ids)
		}
		return nil
	})
	t.Cleanup(func() { RegisterWithdrawalHook(nil) })
	if err := conn.Transaction(func(tx *gorm.DB) error { return WithdrawContentTx(tx, p.TopicId, p.Id) }); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("retained delivery body was not revoked")
	}
	got, err := Get(a.UserId, e.ID)
	if err != nil || got.Data != nil || got.State != "withdrawn" {
		t.Fatalf("withdrawn %#v %v", got, err)
	}
	var i agentEvents.Intent
	if err := conn.Where("id = ?", e.SourceIntentID).Take(&i).Error; err != nil {
		t.Fatal(err)
	}
	if i.Status != "cancelled" || i.ActorID != 0 {
		t.Fatalf("source refs %#v", i)
	}
	if err := conn.Model(&agentEvents.Intent{}).Where("id = ?", i.ID).Update("expires_at", time.Now().UTC().Add(-time.Hour)).Error; err != nil {
		t.Fatal(err)
	}
	if err := Cleanup(); err != nil {
		t.Fatal(err)
	}
	conn.Where("id = ?", i.ID).Take(&i)
	if i.Status != "expired" || i.PostID != 0 || i.Version != 0 {
		t.Fatalf("expired intent retained refs %#v", i)
	}
}

func TestProducerPauseMaintainsPublicBaseline(t *testing.T) {
	conn, p, a := setup(t)
	p.Content = "no mentions"
	if err := conn.Model(&posts.Entity{}).Where("id = ?", p.Id).Update("content", p.Content).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Model(&postRevisions.Entity{}).Where("post_id = ? AND version = 1", p.Id).Update("content", p.Content).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Transaction(func(tx *gorm.DB) error { return CapturePublicTx(tx, &p) }); err != nil {
		t.Fatal(err)
	}
	t.Setenv("YOURTJ_AGENT_PRODUCER_ENABLED", "false")
	p.Content = "added while paused @" + fmt.Sprintf("bot%d", a.UserId-1)
	if err := conn.Model(&posts.Entity{}).Where("id = ?", p.Id).Update("content", p.Content).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Create(&postRevisions.Entity{PostId: p.Id, Version: 2, Content: p.Content}).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Transaction(func(tx *gorm.DB) error { return CapturePublicTx(tx, &p) }); err != nil {
		t.Fatal(err)
	}
	t.Setenv("YOURTJ_AGENT_PRODUCER_ENABLED", "true")
	p.Content += " punctuation"
	if err := conn.Model(&posts.Entity{}).Where("id = ?", p.Id).Update("content", p.Content).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Create(&postRevisions.Entity{PostId: p.Id, Version: 3, Content: p.Content}).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Transaction(func(tx *gorm.DB) error { return CapturePublicTx(tx, &p) }); err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := conn.Model(&agentEvents.Intent{}).Where("post_id = ? AND version = 3", p.Id).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("unchanged mention added while paused generated a new intent on resume: %d", count)
	}
}

func TestCachedActiveEventCannotAuthorizeAfterPermanentWithdrawal(t *testing.T) {
	conn, p, _ := setup(t)
	stale := materialize(t, conn, p)
	// A read can precede lifecycle withdrawal; restored public content must not
	// resurrect the withdrawn occurrence held in the reader's earlier snapshot.
	if err := conn.Transaction(func(tx *gorm.DB) error { return WithdrawContentTx(tx, p.TopicId, p.Id) }); err != nil {
		t.Fatal(err)
	}
	err := conn.Transaction(func(tx *gorm.DB) error { return ValidateEventTx(tx, &stale) })
	if !errors.Is(err, ErrInaccessible) {
		t.Fatalf("cached revoked occurrence authorized: %v", err)
	}
	if stale.WithdrawnAt == nil || Envelope(stale).Data != nil {
		t.Fatalf("cached event was not refreshed to tombstone: %#v", stale)
	}
}

func TestActorFreezeBeforeParticipantFenceRejectsCachedState(t *testing.T) {
	conn, p, _ := setup(t)
	event := materialize(t, conn, p)
	callback := "test1042_freeze_before_participant_fence"
	frozen := false
	if err := conn.Callback().Query().Before("gorm:query").Register(callback, func(tx *gorm.DB) {
		if tx.Statement.Table != "users" || frozen {
			return
		}
		if _, locking := tx.Statement.Clauses["FOR"]; !locking {
			return
		}
		frozen = true
		// Model a freeze committed before authorization acquires the user fence.
		if err := tx.Session(&gorm.Session{NewDB: true}).Model(&users.EntityComplete{}).Where("id = ?", p.UserId).Update("is_frozen", users.StatusFrozen).Error; err != nil {
			tx.AddError(err)
		}
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Callback().Query().Remove(callback) })
	err := conn.Transaction(func(tx *gorm.DB) error { return ValidateEventTx(tx, &event) })
	if !frozen {
		t.Fatal("authorization did not fence its participants")
	}
	if !errors.Is(err, ErrInaccessible) {
		t.Fatalf("actor state read before participant fence was accepted: %v", err)
	}
}

func TestSourceIntentRecoversAfterDefaultRetryBudgetAndManualReplay(t *testing.T) {
	conn, p, a := setup(t)
	if err := conn.Transaction(func(tx *gorm.DB) error { return CapturePublicTx(tx, &p) }); err != nil {
		t.Fatal(err)
	}
	var intent agentEvents.Intent
	if err := conn.Where("post_id = ?", p.Id).Take(&intent).Error; err != nil {
		t.Fatal(err)
	}
	var revision postRevisions.Entity
	if err := conn.Where("post_id = ? AND version = 1", p.Id).Take(&revision).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Where("post_id = ?", p.Id).Delete(&postRevisions.Entity{}).Error; err != nil {
		t.Fatal(err)
	}
	for attempt := 0; attempt < 4; attempt++ {
		task, claimed, err := taskQueue.ClaimTask(intent.TaskID)
		if err != nil || !claimed {
			t.Fatalf("claim %t %v", claimed, err)
		}
		if err := HandleTask(context.Background(), &task); err == nil {
			t.Fatal("missing revision unexpectedly materialized")
		}
		if err := taskQueue.RetryOwned(task.Id, task.LeaseToken, time.Now().Add(-time.Second), "dependency unavailable", 9); err != nil {
			t.Fatal(err)
		}
	}
	retained, err := taskQueue.GetByID(intent.TaskID)
	if err != nil || retained.Status != taskQueue.StatusRetrying || retained.RetryCount != 4 {
		t.Fatalf("intent did not survive default three retries: %#v %v", retained, err)
	}
	if err := conn.Create(&revision).Error; err != nil {
		t.Fatal(err)
	}
	task, claimed, err := taskQueue.ClaimTask(intent.TaskID)
	if err != nil || !claimed {
		t.Fatalf("recovery claim %t %v", claimed, err)
	}
	if err := HandleTask(context.Background(), &task); err != nil {
		t.Fatal(err)
	}
	var event agentEvents.Entity
	if err := conn.Where("source_intent_id = ?", intent.ID).Take(&event).Error; err != nil {
		t.Fatal(err)
	}
	if event.ID != stableID(intent.ID, a.UserId) {
		t.Fatalf("unstable recovered id %s", event.ID)
	}
	// Explicit replay reuses the frozen source identity and recipient.
	if err := ReplayIntent(intent.ID); err != nil {
		t.Fatal(err)
	}
	if err := conn.Where("id = ?", intent.ID).Take(&intent).Error; err != nil {
		t.Fatal(err)
	}
	task, claimed, err = taskQueue.ClaimTask(intent.TaskID)
	if err != nil || !claimed {
		t.Fatalf("manual claim %t %v", claimed, err)
	}
	if err := HandleTask(context.Background(), &task); err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := conn.Model(&agentEvents.Entity{}).Where("source_intent_id = ?", intent.ID).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("replay duplicated recovered occurrence: %d %v", count, err)
	}
}
