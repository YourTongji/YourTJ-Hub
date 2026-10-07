package feedservice

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/feedconfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/feed"
	"gorm.io/gorm"
)

func TestRankBackfillProgressesWhileDueQueueStaysNonempty(t *testing.T) {
	conn := telemetryDB(t)
	preferences.Set("feed.metrics.enabled", false)
	preferences.Set("ranking.enabled", true)
	feedconfig.SetRankReady(false)
	t.Cleanup(func() { feedconfig.SetRankReady(false) })
	ctx := context.Background()
	if err := putState(ctx, "desired_rank_hash", feedconfig.Current().RankHash); err != nil {
		t.Fatal(err)
	}
	if err := putState(ctx, "epoch_clean", "true"); err != nil {
		t.Fatal(err)
	}
	// Missing topics are cheap, valid deletion jobs. More due work than the
	// run's 20/s quota can consume keeps the queue busy throughout the test.
	for i := uint64(1); i <= 20; i++ {
		if err := conn.Create(&feed.Schedule{TopicID: 998000 + i, Version: 1, Generation: feed.NewID(), DueAt: time.Now().Add(-time.Hour), Dirty: true}).Error; err != nil {
			t.Fatal(err)
		}
	}
	runCtx, cancel := context.WithTimeout(ctx, 350*time.Millisecond)
	defer cancel()
	runWorker(runCtx)
	var remaining int64
	if err := conn.Model(&feed.Schedule{}).Count(&remaining).Error; err != nil {
		t.Fatal(err)
	}
	if remaining == 0 {
		t.Fatal("fixture failed to keep the due queue busy")
	}
	if getState(ctx, "posts_cursor") == "" && getState(ctx, "posts_backfilled") != "true" {
		t.Fatal("publication backfill starved behind due ranking work")
	}
}

func TestRankSourceReadFailureBacksOffTheScheduledJob(t *testing.T) {
	conn := telemetryDB(t)
	preferences.Set("ranking.enabled", true)
	ctx := context.Background()
	const topicID uint64 = 998901
	if err := conn.Transaction(func(tx *gorm.DB) error { return feed.MarkTx(tx, topicID) }); err != nil {
		t.Fatal(err)
	}
	failure := errors.New("source read failed")
	if err := conn.Callback().Query().Before("gorm:query").Register("test_fail_rank_source", func(tx *gorm.DB) {
		if tx.Statement.Table == "topics" {
			_ = tx.AddError(failure)
		}
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := conn.Callback().Query().Remove("test_fail_rank_source"); err != nil {
			t.Error(err)
		}
	})
	now := time.Now()
	worked, err := workRank(ctx, now)
	if !worked || !errors.Is(err, failure) {
		t.Fatalf("worked=%v error=%v", worked, err)
	}
	var job feed.Schedule
	if err := conn.First(&job, "topic_id = ?", topicID).Error; err != nil {
		t.Fatal(err)
	}
	if job.Failures != 1 || !job.DueAt.After(now) || job.LastError != failure.Error() {
		t.Fatal("failed source read remained due without retry backoff")
	}
}

func TestCleanStopPersistsUnprocessedRankingPause(t *testing.T) {
	telemetryDB(t)
	preferences.Set("ranking.enabled", false)
	ctx := context.Background()
	for key, value := range map[string]string{"epoch_clean": "true", "ranking_enabled": "true"} {
		if err := putState(ctx, key, value); err != nil {
			t.Fatal(err)
		}
	}
	// Stop before a maintenance tick can observe the pause. A clean restart
	// must still rebuild content changes made with invalidation disabled.
	runCtx, cancel := context.WithCancel(ctx)
	cancel()
	runWorker(runCtx)
	if getState(ctx, "epoch_clean") != "true" || getState(ctx, "ranking_enabled") != "false" {
		t.Fatal("clean shutdown lost the ranking pause before the next worker tick")
	}
}
