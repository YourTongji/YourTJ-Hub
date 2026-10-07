package feedservice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/closer"
	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/feedconfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/feed"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topicUserAction"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topicUserStat"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var startOnce sync.Once
var workerCancel context.CancelFunc
var workerDone = make(chan struct{})

func Start() {
	startOnce.Do(func() {
		ctx, cancel := context.WithCancel(context.Background())
		workerCancel = cancel
		closer.RegisterPriority(closer.PriorityProducer, func() error { Stop(); return nil })
		go func() { defer close(workerDone); runWorker(ctx) }()
	})
}

var stopping atomic.Bool
var backgroundFailures atomic.Uint64
var lastRankAt atomic.Int64
var previousEpochIncomplete atomic.Bool
var handledRankGeneration atomic.Uint64

func Stop() {
	stopping.Store(true)
	if workerCancel == nil {
		return
	}
	workerCancel()
	select {
	case <-workerDone:
	case <-time.After(2 * time.Second):
	}
}

// One loop owns all new background DB activity. It observes the existing pool
// instead of adding connections; no task spawns parallel SQL goroutines.
func runWorker(ctx context.Context) {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	lastCleanup := time.Time{}
	lastRank := time.Time{}
	lastAnalytics := time.Time{}
	lastSnapshot := time.Time{}
	lastEpoch := time.Time{}
	lastReconcile := time.Time{}
	lastPurge := time.Time{}
	lastMaintenance := time.Time{}
	lastTelemetry := time.Time{}
	lastParamsHash := ""
	recoveryPending := getState(ctx, "epoch_clean") != "true"
	if recoveryPending {
		previousEpochIncomplete.Store(true)
		_ = abortPeriods(ctx, "analytics epoch incomplete or restored")
		feedconfig.SetRankReady(false)
	}
	_ = putState(ctx, "epoch_clean", "false")
	_ = putState(ctx, "epoch", processEpoch)
	budgetStart := time.Now()
	busy := time.Duration(0)
	lastBackfill := time.Time{}
	for {
		select {
		case <-ctx.Done():
			flushCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			for {
				queueMu.Lock()
				n := len(queue) + len(sampleQueue)
				queueMu.Unlock()
				if n == 0 || flushCtx.Err() != nil {
					break
				}
				_ = flushTelemetry(flushCtx, feedconfig.Current())
			}
			_ = flushViews(flushCtx, feedconfig.Current())
			queueMu.Lock()
			dropped.Add(uint64(len(queue) + len(sampleQueue)))
			queue = nil
			sampleQueue = nil
			queueBytes = 0
			sampleBytes = 0
			queueMu.Unlock()
			// A pause can precede the next maintenance tick, or be resumed before
			// its rebuild starts. Preserve that fence across a clean restart.
			if !feedconfig.Current().Ranking || feedconfig.RankGeneration() != handledRankGeneration.Load() {
				if err := putState(flushCtx, "ranking_enabled", "false"); err != nil {
					cancel()
					return
				}
			}
			_ = putState(flushCtx, "epoch_clean", "true")
			cancel()
			return
		case <-ticker.C:
		}
		now := time.Now()
		if now.Sub(budgetStart) >= time.Minute {
			budgetStart = now
			busy = 0
		}
		if busy >= 15*time.Second {
			continue
		}
		conn := db.Connect()
		sqlDB, err := conn.DB()
		if err != nil {
			continue
		}
		stats := sqlDB.Stats()
		if stats.InUse >= max(stats.MaxOpenConnections-1, 1) {
			continue
		}
		cfg := feedconfig.Current()
		if !cfg.Metrics && !cfg.Ranking {
			if now.Sub(lastMaintenance) < 10*time.Second {
				continue
			}
			lastMaintenance = now
		}
		start := time.Now()
		jobCtx, cancel := context.WithTimeout(ctx, time.Duration(cfg.QueryTimeoutMS)*time.Millisecond)
		if !cfg.Ranking && getState(jobCtx, "ranking_enabled") != "false" {
			err = putState(jobCtx, "ranking_enabled", "false")
		}
		cleanupInterval := time.Hour
		if cleanupBacklog.Load() {
			cleanupInterval = time.Second
		}
		if err == nil && now.Sub(lastCleanup) >= cleanupInterval {
			err = Cleanup(jobCtx, now)
			if err == nil {
				lastCleanup = now
			}
		}
		if err == nil && (cfg.Metrics || now.Sub(lastTelemetry) >= time.Second) {
			err = flushTelemetry(jobCtx, cfg)
			lastTelemetry = now
		}
		if err == nil && now.Sub(lastPurge) >= time.Second {
			err = purgeClosed(jobCtx)
			lastPurge = now
		}
		if err == nil && now.Sub(lastEpoch) > time.Second {
			_ = putState(jobCtx, "epoch_watermark", now.UTC().Format(time.RFC3339Nano))
			lastEpoch = now
		}
		if err == nil && now.Sub(lastAnalytics) >= time.Second {
			if !cfg.Enabled || !cfg.Metrics || cfg.Rollout == 0 {
				err = abortPeriods(jobCtx, "capture or default feed disabled")
			} else {
				err = abortMismatchedPeriods(jobCtx, cfg.Hash)
				if err == nil {
					err = analyzePeriod(jobCtx)
				}
			}
			lastAnalytics = now
		}
		if err == nil && (cfg.Ranking || cfg.Metrics) && lastParamsHash != cfg.Hash {
			data, encodeErr := json.Marshal(cfg)
			err = encodeErr
			if err == nil {
				err = db.ConnectContext(jobCtx).Clauses(clause.OnConflict{DoNothing: true}).Create(&feed.ParamsVersion{Hash: cfg.Hash, Params: string(data), CreatedAt: now}).Error
			}
			if err == nil {
				lastParamsHash = cfg.Hash
			}
		}
		if err == nil && cfg.Ranking {
			// Reserve a bounded maintenance slot even when due jobs never drain.
			// Rebuild requests and changed hashes must not wait for queue idleness.
			if now.Sub(lastBackfill) >= time.Second {
				if recoveryPending {
					err = RequestRebuild(jobCtx)
					recoveryPending = err != nil
				}
				if err == nil {
					err = backfillRank(jobCtx)
				}
				lastBackfill = now
				if err == nil && feedconfig.RankReady() && now.Sub(lastReconcile) >= time.Second {
					err = reconcileRank(jobCtx)
					lastReconcile = now
				}
				if err == nil && feedconfig.RankReady() && now.Sub(lastSnapshot) >= 6*time.Hour {
					err = captureRankSnapshot(jobCtx, now)
					if err == nil {
						lastSnapshot = now
					}
				}
			}
			if err == nil && now.Sub(lastRank) >= time.Second/time.Duration(cfg.JobsPerSecond) {
				lastRank = now
				var worked bool
				worked, err = workActor(jobCtx)
				if !worked && err == nil {
					_, err = workRank(jobCtx, now)
				}
			}
		}
		cancel()
		if err != nil {
			backgroundFailures.Add(1)
		}
		busy += time.Since(start)
		if err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
			slog.Warn("feed background work failed", "err", err)
		}
	}
}

func rankInput(ctx context.Context, id uint64, topic topics.Entity) (RankInput, error) {
	in := RankInput{Author: topic.UserId}
	if topic.FirstPublicAt != nil {
		in.FirstPublicAt = *topic.FirstPublicAt
	}
	likers, err := topicUserAction.RankLikers(ctx, id)
	if err != nil {
		return in, err
	}
	if len(likers) > 50000 {
		return in, fmt.Errorf("rank participant bound exceeded")
	}
	byUser := map[uint64]Participant{}
	// Initial backfill and lifecycle invalidation rebuild the bounded aggregate.
	// Avoid reading an old projection that would immediately be discarded.
	if !topic.RankReady {
		rows, e := posts.RankRepliers(ctx, id)
		if e != nil {
			return in, e
		}
		if len(rows) > 50000 {
			return in, fmt.Errorf("rank replier bound exceeded")
		}
		for _, r := range rows {
			byUser[r.UserID] = Participant{UserID: r.UserID, Replies: r.Replies, LastPublicReplyAt: r.LastPublicReplyAt}
		}
	} else {
		participants, e := topicUserStat.RankParticipants(ctx, id)
		if e != nil {
			return in, e
		}
		if len(participants) > 50000 {
			return in, fmt.Errorf("rank participant bound exceeded")
		}
		for _, p := range participants {
			byUser[p.UserId] = Participant{UserID: p.UserId, Replies: p.RankReplyCount, LastPublicReplyAt: p.LastPublicReplyAt}
		}
	}
	for _, uid := range likers {
		p := byUser[uid]
		p.UserID = uid
		p.Liked = true
		byUser[uid] = p
	}
	for _, p := range byUser {
		in.Participants = append(in.Participants, p)
	}
	cut := time.Now().Add(-24 * time.Hour)
	replies, err := posts.DailyReplyFacts(ctx, id, cut)
	if err != nil {
		return in, err
	}
	if len(replies) > 50000 {
		return in, fmt.Errorf("daily reply bound exceeded")
	}
	for _, r := range replies {
		in.Actions = append(in.Actions, TimedAction{r.UserID, "reply", r.FirstPublicAt})
	}
	var views []feed.ViewFact
	if err = db.ConnectContext(ctx).Where("topic_id = ? AND expires_at > ?", id, time.Now()).Where("user_id IN (?)", users.EligibleIDsQuery(ctx)).Limit(50001).Find(&views).Error; err != nil {
		return in, err
	}
	if len(views) > 50000 {
		return in, fmt.Errorf("view bound exceeded")
	}
	for _, v := range views {
		in.Actions = append(in.Actions, TimedAction{v.UserID, "view", v.ViewedAt})
	}
	var credit []feed.ActionCredit
	if err = db.ConnectContext(ctx).Where("topic_id = ? AND active = ? AND expires_at > ?", id, true, time.Now()).Where("user_id IN (?)", users.EligibleIDsQuery(ctx)).Limit(50001).Find(&credit).Error; err != nil {
		return in, err
	}
	if len(credit) > 50000 {
		return in, fmt.Errorf("credit bound exceeded")
	}
	for _, v := range credit {
		in.Actions = append(in.Actions, TimedAction{v.UserID, v.Kind, v.CreditAt})
	}
	return in, nil
}

func workRank(ctx context.Context, now time.Time) (bool, error) {
	cfg := feedconfig.Current()
	var job feed.Schedule
	err := db.ConnectContext(ctx).Where("due_at <= ?", now).Order("dirty DESC").Order("due_at").First(&job).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	topic, err := topics.RankTopic(ctx, job.TopicID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return true, db.ConnectContext(ctx).Transaction(func(tx *gorm.DB) error {
			var source topics.Entity
			if e := tx.Unscoped().Clauses(clause.Locking{Strength: "UPDATE"}).Select("id").First(&source, "id = ?", job.TopicID).Error; e != nil && !errors.Is(e, gorm.ErrRecordNotFound) {
				return e
			}
			var locked feed.Schedule
			if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&locked, "topic_id = ?", job.TopicID).Error; errors.Is(e, gorm.ErrRecordNotFound) {
				return nil
			} else if e != nil {
				return e
			}
			if locked.Generation != job.Generation || locked.Version != job.Version || !locked.DueAt.Equal(job.DueAt) {
				return nil
			}
			if err := topics.ClearRankTx(tx, job.TopicID, cfg.RankHash, now); err != nil {
				return err
			}
			return tx.Where("topic_id = ? AND generation = ? AND version = ?", job.TopicID, job.Generation, job.Version).Delete(&feed.Schedule{}).Error
		})
	}
	if err != nil {
		backoffRank(ctx, job, now, err)
		return true, err
	}
	if job.ProjectionDirty {
		// Public replies and lifecycle changes invalidate the projection in
		// their transaction. Rebuild before acknowledging the source watermark.
		topic.RankReady = false
	}
	in, err := rankInput(ctx, job.TopicID, topic)
	if err != nil {
		backoffRank(ctx, job, now, err)
		return true, err
	}
	r := ScoreRankWithRules(in, now, cfg.Rules)
	err = db.ConnectContext(ctx).Transaction(func(tx *gorm.DB) error {
		var lockedTopic topics.Entity
		if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&lockedTopic, "id = ?", job.TopicID).Error; e != nil {
			return e
		}
		var locked feed.Schedule
		if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&locked, "topic_id = ?", job.TopicID).Error; e != nil {
			return e
		}
		if locked.Generation != job.Generation || locked.Version != job.Version || !locked.DueAt.Equal(job.DueAt) {
			return nil
		}
		var due *time.Time
		if !r.NextDue.IsZero() {
			at := r.NextDue
			due = &at
		}
		if e := tx.Model(&topics.Entity{}).Where("id = ?", job.TopicID).UpdateColumn("rank_due_at", due).Error; e != nil {
			return e
		}
		if !topic.RankReady {
			projections := []topicUserStat.ReplierStat{}
			var lastPublic *time.Time
			for _, p := range in.Participants {
				if p.Replies > 0 {
					projections = append(projections, topicUserStat.ReplierStat{UserID: p.UserID, RankReplyCount: p.Replies, LastPublicReplyAt: p.LastPublicReplyAt})
				}
				if p.UserID != topic.UserId && p.LastPublicReplyAt != nil && (lastPublic == nil || p.LastPublicReplyAt.After(*lastPublic)) {
					at := *p.LastPublicReplyAt
					lastPublic = &at
				}
			}
			if e := topicUserStat.BackfillRankProjectionTx(tx, job.TopicID, projections); e != nil {
				return e
			}
			if e := tx.Model(&topics.Entity{}).Where("id = ?", job.TopicID).UpdateColumn("last_public_reply_at", lastPublic).Error; e != nil {
				return e
			}
			topic.LastPublicReplyAt = lastPublic
		}
		if e := tx.Model(&topics.Entity{}).Where("id = ?", job.TopicID).UpdateColumn("rank_source", topics.RankSource(topic)).Error; e != nil {
			return e
		}
		components, e := json.Marshal(r)
		if e != nil {
			return e
		}
		if e := tx.Model(&topics.Entity{}).Where("id = ?", job.TopicID).UpdateColumns(map[string]any{"rank_engaged": r.HotPeople, "rank_components": string(components)}).Error; e != nil {
			return e
		}
		if e := topics.WriteRankTx(tx, job.TopicID, r.Hot, r.Daily, cfg.RankHash, now); e != nil {
			return e
		}
		if r.NextDue.IsZero() {
			return tx.Where("topic_id = ? AND generation = ? AND version = ?", job.TopicID, job.Generation, job.Version).Delete(&feed.Schedule{}).Error
		}
		return tx.Model(&feed.Schedule{}).Where("topic_id = ? AND generation = ? AND version = ?", job.TopicID, job.Generation, job.Version).UpdateColumns(map[string]any{"dirty": false, "projection_dirty": false, "due_at": r.NextDue, "failures": 0, "last_error": ""}).Error
	})
	if err == nil {
		lastRankAt.Store(now.Unix())
	}
	return true, err
}

func backoffRank(ctx context.Context, job feed.Schedule, now time.Time, failure error) {
	repairCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 50*time.Millisecond)
	defer cancel()
	_ = db.ConnectContext(repairCtx).Model(&feed.Schedule{}).Where("topic_id = ? AND generation = ? AND version = ?", job.TopicID, job.Generation, job.Version).UpdateColumns(map[string]any{"due_at": now.Add(time.Minute), "failures": gorm.Expr("failures + 1"), "last_error": failure.Error()[:min(len(failure.Error()), 256)]}).Error
}

func getState(ctx context.Context, key string) string {
	var row feed.State
	_ = db.ConnectContext(ctx).First(&row, "key = ?", key).Error
	return row.Value
}
func putState(ctx context.Context, key, value string) error {
	return db.ConnectContext(ctx).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "key"}}, DoUpdates: clause.AssignmentColumns([]string{"value"})}).Create(&feed.State{Key: key, Value: value}).Error
}

// Backfills never run in the startup migration gate. Cursors persist after a
// successful bounded batch; ranking remains unavailable until the queue drains.
func backfillRank(ctx context.Context) error {
	cfg := feedconfig.Current()
	generation := feedconfig.RankGeneration()
	if generation != handledRankGeneration.Load() || getState(ctx, "ranking_enabled") != "true" || getState(ctx, "desired_rank_hash") != cfg.RankHash {
		feedconfig.SetRankReady(false)
		if err := RequestRebuild(ctx); err != nil {
			return err
		}
		if err := putState(ctx, "desired_rank_hash", cfg.RankHash); err != nil {
			return err
		}
		if err := putState(ctx, "ranking_enabled", "true"); err != nil {
			return err
		}
		handledRankGeneration.Store(generation)
	}
	if feedconfig.RankReady() {
		return nil
	}
	if getState(ctx, "posts_backfilled") != "true" {
		cursor, _ := strconv.ParseUint(getState(ctx, "posts_cursor"), 10, 64)
		next, done, err := posts.BackfillFirstPublicBatch(ctx, cursor)
		if err != nil {
			return err
		}
		if err = putState(ctx, "posts_cursor", strconv.FormatUint(next, 10)); err != nil {
			return err
		}
		if done {
			return putState(ctx, "posts_backfilled", "true")
		}
		return nil
	}
	if getState(ctx, "topics_backfilled") != "true" {
		cursor, _ := strconv.ParseUint(getState(ctx, "topics_cursor"), 10, 64)
		ids, err := topics.RankBackfillBatch(ctx, cursor, 200)
		if err != nil {
			return err
		}
		if len(ids) == 0 {
			return putState(ctx, "topics_backfilled", "true")
		}
		if err = topics.BackfillFirstPublic(ctx, ids); err != nil {
			return err
		}
		if err = db.ConnectContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := topics.ResetRankReadinessTx(tx, ids); err != nil {
				return err
			}
			return feed.MarkManyProjectionTx(tx, ids)
		}); err != nil {
			return err
		}
		return putState(ctx, "topics_cursor", strconv.FormatUint(ids[len(ids)-1], 10))
	}
	ready, err := topics.AllRankReady(ctx, feedconfig.Current().RankHash)
	if err != nil {
		return err
	}
	feedconfig.SetRankReady(ready)
	if ready {
		if err := putState(ctx, "promoted_rank_hash", feedconfig.Current().RankHash); err != nil {
			return err
		}
		cfg := feedconfig.Current()
		data, err := json.Marshal(cfg)
		if err != nil {
			return err
		}
		return db.ConnectContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&feed.ParamsVersion{Hash: cfg.Hash, Params: string(data), CreatedAt: time.Now()}).Error
	}
	return nil
}

// RequestRebuild only queues work. The serve process remains the only scorer.
func RequestRebuild(ctx context.Context) error {
	feedconfig.SetRankReady(false)
	if err := putState(ctx, "promoted_rank_hash", ""); err != nil {
		return err
	}
	if err := putState(ctx, "topics_cursor", "0"); err != nil {
		return err
	}
	return putState(ctx, "topics_backfilled", "false")
}

func reconcileRank(ctx context.Context) error {
	if getState(ctx, "promoted_rank_hash") != feedconfig.Current().RankHash {
		return RequestRebuild(ctx)
	}
	cursor, _ := strconv.ParseUint(getState(ctx, "reconcile_cursor"), 10, 64)
	rows, err := topics.ReconcileRankBatch(ctx, cursor)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return putState(ctx, "reconcile_cursor", "0")
	}
	ids := []uint64{}
	for _, r := range rows {
		if topics.RankSource(r) != r.RankSource {
			ids = append(ids, r.Id)
		}
	}
	if len(ids) > 0 {
		if err = db.ConnectContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := topics.ResetRankReadinessTx(tx, ids); err != nil {
				return err
			}
			return feed.MarkManyProjectionTx(tx, ids)
		}); err != nil {
			return err
		}
	}
	return putState(ctx, "reconcile_cursor", strconv.FormatUint(rows[len(rows)-1].Id, 10))
}
func captureRankSnapshot(ctx context.Context, now time.Time) error {
	type savedRank struct {
		ID         uint64
		Hot, Daily int64
		Components string
	}
	items := map[string][]savedRank{}
	for _, source := range []string{"hot", "daily"} {
		ids, err := topics.RecallRankIDs(ctx, topics.Recall{Source: source, Hash: feedconfig.Current().RankHash, Limit: 100})
		if err != nil {
			return err
		}
		rows, err := topics.RankTopics(ctx, ids)
		if err != nil {
			return err
		}
		// Compact, anonymous topic statistics, no viewer identity or post content.
		list := []savedRank{}
		for _, r := range rows {
			list = append(list, savedRank{ID: r.Id, Hot: r.RankScore, Daily: r.DailyScore, Components: r.RankComponents})
		}
		items[source] = list
	}
	data, err := json.Marshal(items)
	if err != nil {
		return err
	}
	return db.ConnectContext(ctx).Create(&feed.RankSnapshot{ID: feed.NewID(), Hash: feedconfig.Current().RankHash, Items: string(data), CreatedAt: now, ExpiresAt: now.Add(30 * 24 * time.Hour)}).Error
}

// Identity changes share the same worker, quota and query deadline as scoring.
func workActor(ctx context.Context) (bool, error) {
	var work feed.ActorWork
	conn := db.ConnectContext(ctx)
	if err := conn.Order("user_id").First(&work).Error; errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	var ids []uint64
	var err error
	switch work.Phase {
	case 0:
		ids, err = topics.ActorTopicIDs(ctx, work.UserID, work.Cursor)
	case 1:
		ids, err = posts.ActorTopicIDs(ctx, work.UserID, work.Cursor)
	case 2:
		ids, err = topicUserAction.ActorTopicIDs(ctx, work.UserID, work.Cursor)
	}
	if err != nil {
		return true, err
	}
	err = conn.Transaction(func(tx *gorm.DB) error {
		var locked feed.ActorWork
		if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&locked, "user_id = ?", work.UserID).Error; errors.Is(e, gorm.ErrRecordNotFound) {
			return nil
		} else if e != nil {
			return e
		}
		if locked.Version != work.Version {
			return nil
		}
		if e := feed.MarkManyProjectionTx(tx, ids); e != nil {
			return e
		}
		if len(ids) > 0 {
			return tx.Model(&locked).UpdateColumn("cursor", ids[len(ids)-1]).Error
		}
		if work.Phase >= 2 {
			return tx.Delete(&locked).Error
		}
		return tx.Model(&locked).UpdateColumns(map[string]any{"phase": work.Phase + 1, "cursor": 0}).Error
	})
	return true, err
}
