package feedservice

import (
	"context"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/feedconfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/feed"
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
