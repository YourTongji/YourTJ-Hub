package agenteventservice

import (
	"context"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/agentEvents"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
	"gorm.io/gorm"
)

func pendingLifecycleTask(t *testing.T, conn *gorm.DB, postID uint64) (agentEvents.Intent, taskQueue.Entity) {
	t.Helper()
	var intent agentEvents.Intent
	if err := conn.Where("post_id = ?", postID).Take(&intent).Error; err != nil {
		t.Fatal(err)
	}
	task, claimed, err := taskQueue.ClaimTask(intent.TaskID)
	if err != nil || !claimed {
		t.Fatalf("claim %t %v", claimed, err)
	}
	return intent, task
}

func TestEventExpiryDuringAuthorizationDoesNotBecomeWithdrawal(t *testing.T) {
	conn, post, agent := setup(t)
	event := materialize(t, conn, post)
	expired := false
	if err := conn.Callback().Query().Before("gorm:query").Register("test:expiry-during-fence", func(tx *gorm.DB) {
		if tx.Statement.Table != "users" || expired {
			return
		}
		if _, locking := tx.Statement.Clauses["FOR"]; !locking {
			return
		}
		expired = true
		// Model retention elapsing while a request waits for participant fences,
		// without making the test depend on sleeps or execution speed.
		_ = tx.AddError(tx.Session(&gorm.Session{NewDB: true}).Model(&agentEvents.Entity{}).Where("id = ?", event.ID).Update("expires_at", time.Now().UTC().Add(-time.Hour)).Error)
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Callback().Query().Remove("test:expiry-during-fence") })
	got, err := Get(agent.UserId, event.ID)
	if !expired || err != nil || got.State != "expired" || got.Data != nil {
		t.Fatalf("expiry during authorization was treated as withdrawal: %#v, %v", got, err)
	}
}

func TestRetentionDoesNotErasePermanentWithdrawal(t *testing.T) {
	conn, post, agent := setup(t)
	event := materialize(t, conn, post)
	if err := conn.Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&posts.Entity{Id: post.Id}).Error; err != nil {
			return err
		}
		return WithdrawContentTx(tx, post.TopicId, post.Id)
	}); err != nil {
		t.Fatal(err)
	}
	if err := conn.Model(&event).Update("expires_at", time.Now().UTC().Add(-time.Hour)).Error; err != nil {
		t.Fatal(err)
	}
	if err := Cleanup(); err != nil {
		t.Fatal(err)
	}
	got, err := Get(agent.UserId, event.ID)
	if err != nil || got.State != "withdrawn" || got.Data != nil {
		t.Fatalf("retention erased permanent revocation: %#v, %v", got, err)
	}
}

func TestWorkerExpiredIntentRedactsReferencesAndCleanupRepairsLegacyExpiry(t *testing.T) {
	conn, post, _ := setup(t)
	if err := conn.Transaction(func(tx *gorm.DB) error { return CapturePublicTx(tx, &post) }); err != nil {
		t.Fatal(err)
	}
	intent, task := pendingLifecycleTask(t, conn, post.Id)
	if err := conn.Model(&intent).Updates(map[string]any{"expires_at": time.Now().UTC().Add(-time.Hour), "previous_version": 4}).Error; err != nil {
		t.Fatal(err)
	}
	if err := HandleTask(context.Background(), &task); err != nil {
		t.Fatal(err)
	}
	assertRedacted := func() {
		t.Helper()
		if err := conn.Where("id = ?", intent.ID).Take(&intent).Error; err != nil {
			t.Fatal(err)
		}
		if intent.Status != "expired" || intent.LastError != "retention_expired" || intent.ActorID != 0 || intent.PostID != 0 || intent.Version != 0 || intent.PreviousVersion != 0 {
			t.Errorf("expired intent retained source references: %#v", intent)
		}
	}
	assertRedacted()
	// Repair rows expired by older workers, even when their status is already final.
	if err := conn.Model(&intent).Updates(map[string]any{"actor_id": post.UserId, "post_id": post.Id, "version": 1, "previous_version": 4}).Error; err != nil {
		t.Fatal(err)
	}
	if err := Cleanup(); err != nil {
		t.Fatal(err)
	}
	assertRedacted()
}

func TestExpiredEventRemainsExpiredAcrossReadAndCleanup(t *testing.T) {
	for _, cleanupFirst := range []bool{false, true} {
		t.Run(map[bool]string{false: "read", true: "cleanup_then_read"}[cleanupFirst], func(t *testing.T) {
			conn, post, agent := setup(t)
			event := materialize(t, conn, post)
			if err := conn.Model(&event).Update("expires_at", time.Now().UTC().Add(-time.Hour)).Error; err != nil {
				t.Fatal(err)
			}
			if cleanupFirst {
				if err := Cleanup(); err != nil {
					t.Fatal(err)
				}
			}
			for range 2 {
				got, err := Get(agent.UserId, event.ID)
				if err != nil || got.State != "expired" || got.Data != nil {
					t.Errorf("expired event classified as revocation: %#v, %v", got, err)
				}
			}
			if err := conn.Where("id = ?", event.ID).Take(&event).Error; err != nil {
				t.Fatal(err)
			}
			if event.WithdrawnAt != nil || event.ActorID != 0 || event.PostID != 0 || event.TopicID != 0 || len(event.Reasons) != 0 {
				t.Errorf("expiry retained references or recorded a withdrawal: %#v", event)
			}
			if err := Ack(agent.UserId, []string{event.ID}); err == nil {
				t.Fatal("expired event was ACKed")
			}
			if err := conn.Transaction(func(tx *gorm.DB) error {
				allowed, err := AuthorizeEventTx(tx, &event)
				if allowed {
					t.Error("expired event authorized a delivery")
				}
				return err
			}); err != nil {
				t.Fatal(err)
			}
			if err := conn.Transaction(func(tx *gorm.DB) error { return ValidateSourceTx(tx, agent.UserId, event.ID, post.TopicId) }); err == nil {
				t.Fatal("expired event authorized a source-linked write")
			}
			rows, err := agentEvents.ExpiringEventsTx(conn, event.InstanceID, time.Now().UTC(), 500)
			if err != nil || len(rows) != 0 {
				t.Fatalf("redacted expiry keeps occupying the cleanup batch: %d, %v", len(rows), err)
			}
		})
	}
}
