package feedservice

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/feedconfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/feed"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/postRevisions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topicUserAction"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topicUserStat"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/postservice"
	"gorm.io/gorm"
)

func TestPublicReplyRebuildsProjectionBeforeQueuedRankAdvancesWatermark(t *testing.T) {
	for _, anonymous := range []bool{false, true} {
		name := "public"
		if anonymous {
			name = "anonymous"
		}
		t.Run(name, func(t *testing.T) {
			conn := telemetryDB(t)
			if err := conn.AutoMigrate(&topicUserAction.Entity{}, &topicUserStat.Entity{}, &topicUserStat.RankParticipant{}, &postRevisions.Entity{}); err != nil {
				t.Fatal(err)
			}
			preferences.Set("ranking.enabled", true)
			const topicID, author, peer, newcomer uint64 = 998201, 998211, 998212, 998213
			at := time.Now().Add(-8 * 24 * time.Hour)
			people := []users.EntityComplete{{Id: author, Username: "rank-reply-author", Email: "rank-reply-author@example.invalid"}, {Id: peer, Username: "rank-reply-peer", Email: "rank-reply-peer@example.invalid"}, {Id: newcomer, Username: "rank-reply-new", Email: "rank-reply-new@example.invalid"}}
			if err := conn.Create(&people).Error; err != nil {
				t.Fatal(err)
			}
			topic := topics.Entity{Id: topicID, UserId: author, Status: 1, FirstPostId: topicID, FirstPublicAt: &at, PostSeq: 2, PostCount: 2, ReplyCount: 1}
			if err := conn.Create(&topic).Error; err != nil {
				t.Fatal(err)
			}
			rows := []posts.Entity{{Id: topicID, TopicId: topicID, UserId: author, PostNo: 1, FirstPublicAt: &at}, {Id: topicID + 1, TopicId: topicID, UserId: peer, PostNo: 2, FirstPublicAt: &at}}
			if err := conn.Create(&rows).Error; err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				conn.Where("post_id IN (?)", conn.Unscoped().Model(&posts.Entity{}).Select("id").Where("topic_id = ?", topicID)).Delete(&postRevisions.Entity{})
				conn.Unscoped().Where("topic_id = ?", topicID).Delete(&posts.Entity{})
				conn.Unscoped().Delete(&topic)
				conn.Unscoped().Where("id IN ?", []uint64{author, peer, newcomer}).Delete(&users.EntityComplete{})
				conn.Where("topic_id = ?", topicID).Delete(&topicUserStat.Entity{})
				conn.Where("topic_id = ?", topicID).Delete(&topicUserStat.RankParticipant{})
			})
			ctx := context.Background()
			mark := func() {
				t.Helper()
				if err := conn.Transaction(func(tx *gorm.DB) error { return feed.MarkTx(tx, topicID) }); err != nil {
					t.Fatal(err)
				}
			}
			rank := func() topics.Entity {
				t.Helper()
				if worked, err := workRank(ctx, time.Now()); err != nil || !worked {
					t.Fatalf("rank: worked=%v err=%v", worked, err)
				}
				var got topics.Entity
				if err := conn.First(&got, "id = ?", topicID).Error; err != nil {
					t.Fatal(err)
				}
				return got
			}
			mark()
			before := rank()
			if !before.RankReady || before.RankEngaged != 1 {
				t.Fatalf("fixture has no ready peer projection: %+v", before)
			}
			// A view/like job is already due. Run it after the reply commits but
			// before reconcile gets a chance to detect the changed source.
			mark()
			replyAt := time.Now().Add(-time.Minute)
			reply := posts.Entity{TopicId: topicID, UserId: newcomer, Content: "new public reply", FirstPublicAt: &replyAt, IsAnonymous: anonymous, VisibilityStatus: posts.VisibilityActive}
			if err := postservice.CreateTopicPost(&reply, topic); err != nil {
				t.Fatal(err)
			}
			mark() // Later ordinary dirtiness must preserve the reply invalidation.
			got := rank()
			if got.RankEngaged != 2 || got.LastPublicReplyAt == nil || !got.LastPublicReplyAt.Equal(replyAt) {
				t.Fatalf("queued rank lost the new reply: people=%d last=%v want %v", got.RankEngaged, got.LastPublicReplyAt, replyAt)
			}
			if got.RankSource != topics.RankSource(got) {
				t.Fatal("rank watermark does not match the rebuilt source")
			}
			participants, err := topicUserStat.RankParticipants(ctx, topicID)
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, p := range participants {
				if p.UserId == newcomer && p.RankReplyCount == 1 {
					found = true
				}
			}
			if !found {
				t.Fatal("new reply missing from the stored participant projection")
			}
		})
	}
}

func TestReplyLifecycleRebuildsRankingWithoutSynchronousProjection(t *testing.T) {
	conn := telemetryDB(t)
	if err := conn.AutoMigrate(&topicUserAction.Entity{}, &topicUserStat.Entity{}, &topicUserStat.RankParticipant{}); err != nil {
		t.Fatal(err)
	}
	preferences.Set("ranking.enabled", true)
	t.Cleanup(func() { feedconfig.SetRankReady(false) })
	const topicID, author, peer uint64 = 998101, 998111, 998112
	at := time.Now().Add(-8 * 24 * time.Hour)
	people := []users.EntityComplete{{Id: author, Username: "rank-life-author", Email: "rank-life-author@example.invalid"}, {Id: peer, Username: "rank-life-peer", Email: "rank-life-peer@example.invalid"}}
	if err := conn.Create(&people).Error; err != nil {
		t.Fatal(err)
	}
	topic := topics.Entity{Id: topicID, UserId: author, Status: 1, FirstPostId: topicID, FirstPublicAt: &at}
	if err := conn.Create(&topic).Error; err != nil {
		t.Fatal(err)
	}
	rows := []posts.Entity{{Id: topicID, TopicId: topicID, UserId: author, PostNo: 1, FirstPublicAt: &at}, {Id: topicID + 1, TopicId: topicID, UserId: peer, PostNo: 2, FirstPublicAt: &at}}
	if err := conn.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		conn.Unscoped().Where("topic_id = ?", topicID).Delete(&posts.Entity{})
		conn.Unscoped().Delete(&topic)
		conn.Unscoped().Where("id IN ?", []uint64{author, peer}).Delete(&users.EntityComplete{})
		conn.Where("topic_id = ?", topicID).Delete(&topicUserStat.Entity{})
		conn.Where("topic_id = ?", topicID).Delete(&topicUserStat.RankParticipant{})
	})
	ctx := context.Background()
	score := func() int64 {
		t.Helper()
		worked, err := workRank(ctx, time.Now())
		if err != nil || !worked {
			t.Fatalf("lifecycle not durably queued: worked=%v err=%v", worked, err)
		}
		var got topics.Entity
		if err := conn.First(&got, "id = ?", topicID).Error; err != nil {
			t.Fatal(err)
		}
		if got.RankReady && got.RankSource != topics.RankSource(got) {
			t.Fatal("rebuilt projection left a stale watermark and would queue another rebuild")
		}
		return got.RankScore
	}
	if err := conn.Transaction(func(tx *gorm.DB) error { return feed.MarkTx(tx, topicID) }); err != nil {
		t.Fatal(err)
	}
	before := score()
	if before <= 0 {
		t.Fatal("fixture has no peer contribution")
	}
	if err := posts.UpdateProcessStatus(topicID+1, posts.ProcessStatusBlocked); err != nil {
		t.Fatal(err)
	}
	if err := conn.Transaction(func(tx *gorm.DB) error { return feed.MarkTx(tx, topicID) }); err != nil {
		t.Fatal(err)
	}
	if got := score(); got >= before {
		t.Fatalf("blocked reply still contributes: %d >= %d", got, before)
	}
	if err := posts.UpdateProcessStatus(topicID+1, posts.ProcessStatusNormal); err != nil {
		t.Fatal(err)
	}
	if got := score(); got != before {
		t.Fatalf("unblocked score=%d want %d", got, before)
	}
	if err := conn.Transaction(func(tx *gorm.DB) error { return posts.MarkUserDeletedTx(tx, topicID+1, peer, "") }); err != nil {
		t.Fatal(err)
	}
	if got := score(); got >= before {
		t.Fatalf("deleted reply still contributes: %d", got)
	}
	if err := posts.Restore(topicID + 1); err != nil {
		t.Fatal(err)
	}
	if got := score(); got != before {
		t.Fatalf("restored score=%d want %d", got, before)
	}
	// A hidden parent may consume a reply invalidation before becoming public.
	if err := conn.Transaction(func(tx *gorm.DB) error { return topics.UpdateProcessStatusTx(tx, topicID, topics.ProcessStatusBlocked) }); err != nil {
		t.Fatal(err)
	}
	if err := posts.UpdateProcessStatus(topicID+1, posts.ProcessStatusBlocked); err != nil {
		t.Fatal(err)
	}
	if got := score(); got != 0 {
		t.Fatalf("hidden parent score=%d", got)
	}
	if err := conn.Transaction(func(tx *gorm.DB) error { return topics.UpdateProcessStatusTx(tx, topicID, topics.ProcessStatusNormal) }); err != nil {
		t.Fatal(err)
	}
	if got := score(); got != 0 {
		t.Fatalf("parent restoration revived a blocked reply: %d", got)
	}
	if err := posts.UpdateProcessStatus(topicID+1, posts.ProcessStatusNormal); err != nil {
		t.Fatal(err)
	}
	if got := score(); got != before {
		t.Fatalf("peer restoration score=%d want %d", got, before)
	}
	// Frozen actors can disappear from the rebuilt projection; unfreezing must
	// derive their current public replies again, rather than trust that projection.
	for _, status := range []int8{users.StatusFrozen, users.StatusNormal} {
		if err := users.UpdateFields(peer, map[string]any{"is_frozen": status}); err != nil {
			t.Fatal(err)
		}
		if status == users.StatusFrozen {
			for _, state := range []int8{posts.ProcessStatusBlocked, posts.ProcessStatusNormal} {
				if err := posts.UpdateProcessStatus(topicID+1, state); err != nil {
					t.Fatal(err)
				}
			}
		}
		for i := 0; i < 5; i++ {
			worked, err := workActor(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if !worked {
				break
			}
		}
		got := score()
		if status == users.StatusFrozen && got != 0 {
			t.Fatalf("frozen peer score=%d", got)
		}
		if status == users.StatusNormal && got != before {
			t.Fatalf("unfreezing lost public peer history: %d want %d", got, before)
		}
	}
	// Pausing ranking skips invalidation writes. Re-enabling must rebuild those
	// missed public-reply changes before the old scores can become ready again.
	for key, value := range map[string]string{"desired_rank_hash": feedconfig.Current().RankHash, "promoted_rank_hash": feedconfig.Current().RankHash, "posts_backfilled": "true", "topics_backfilled": "true", "ranking_enabled": "true"} {
		if err := putState(ctx, key, value); err != nil {
			t.Fatal(err)
		}
	}
	feedconfig.SetRankReady(true)
	preferences.Set("ranking.enabled", false)
	if err := posts.UpdateProcessStatus(topicID+1, posts.ProcessStatusBlocked); err != nil {
		t.Fatal(err)
	}
	preferences.Set("ranking.enabled", true)
	if feedconfig.RankReady() {
		t.Fatal("pause/resume promoted stale scores before rebuilding missed changes")
	}
	for i := 0; i < 5; i++ {
		if err := backfillRank(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if got := score(); got != 0 {
		t.Fatalf("paused moderation still contributes after rebuilding: %d", got)
	}
	if err := posts.UpdateProcessStatus(topicID+1, posts.ProcessStatusNormal); err != nil {
		t.Fatal(err)
	}
	if got := score(); got != before {
		t.Fatalf("post-resume score=%d want %d", got, before)
	}
	failure := errors.New("roll back lifecycle")
	if err := conn.Transaction(func(tx *gorm.DB) error {
		if err := posts.UpdateProcessStatusTx(tx, topicID+1, posts.ProcessStatusBlocked); err != nil {
			return err
		}
		return failure
	}); !errors.Is(err, failure) {
		t.Fatal(err)
	}
	var scheduled int64
	if err := conn.Model(&feed.Schedule{}).Where("topic_id = ?", topicID).Count(&scheduled).Error; err != nil {
		t.Fatal(err)
	}
	if scheduled != 0 || posts.Get(topicID+1).ProcessStatus != posts.ProcessStatusNormal {
		t.Fatal("rolled-back moderation left ranking work or changed visibility")
	}
	if err := conn.Callback().Create().Before("gorm:create").Register("test_fail_rank_schedule", func(tx *gorm.DB) {
		if tx.Statement.Table == "topic_rank_schedule" {
			_ = tx.AddError(failure)
		}
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := conn.Callback().Create().Remove("test_fail_rank_schedule"); err != nil {
			t.Error(err)
		}
	})
	if err := posts.UpdateProcessStatus(topicID+1, posts.ProcessStatusBlocked); !errors.Is(err, failure) {
		t.Fatalf("enqueue failure not propagated: %v", err)
	}
	if posts.Get(topicID+1).ProcessStatus != posts.ProcessStatusNormal {
		t.Fatal("enqueue failure committed moderation without repair work")
	}

}
