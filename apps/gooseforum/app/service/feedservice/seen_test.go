package feedservice

import (
	"context"
	"errors"
	"fmt"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/feed"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/feedconfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topicUserAction"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userFollow"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"gorm.io/gorm"
)

func seenFixture(t *testing.T) (*gorm.DB, uint64, uint64, []Candidate) {
	t.Helper()
	conn := telemetryDB(t)
	if err := conn.AutoMigrate(&userFollow.Entity{}, &users.BlockEntity{}, &topicUserAction.Entity{}); err != nil {
		t.Fatal(err)
	}
	viewer, author := uint64(953101), uint64(953102)
	for _, id := range []uint64{viewer, author} {
		row := &users.EntityComplete{Id: id, Username: fmt.Sprintf("seen-%d", id), Email: fmt.Sprintf("seen-%d@example.invalid", id)}
		if err := conn.Create(row).Error; err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { conn.Unscoped().Delete(row) })
	}
	preferences.Set("ranking.enabled", true)
	preferences.Set("feed.for_you.enabled", true)
	preferences.Set("feed.metrics.enabled", false)
	feedconfig.SetRankReady(true)
	t.Cleanup(func() { feedconfig.SetRankReady(false); invalidateViewer(viewer) })
	items := []Candidate{}
	now := time.Now().Add(-time.Hour)
	for i := 0; i < 25; i++ {
		id := uint64(953200 + i)
		topic := &topics.Entity{Id: id, UserId: author, Status: 1, FirstPostId: id, FirstPublicAt: &now, RankReady: true, RankParamsHash: feedconfig.Current().RankHash, RankScore: 1000}
		post := &posts.Entity{Id: id, TopicId: id, UserId: author, PostNo: 1, FirstPublicAt: &now}
		for _, row := range []any{topic, post} {
			if err := conn.Create(row).Error; err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { conn.Unscoped().Delete(row) })
		}
		items = append(items, Candidate{ID: id, Author: author})
	}
	return conn, viewer, author, items
}

func TestForYouNewFirstPageDoesNotReusePreviousBatch(t *testing.T) {
	_, viewer, _, items := seenFixture(t)
	cfg := feedconfig.Current()
	now := time.Now()
	if !storeSnapshot(&snapshot{ID: "previous-first-page", User: viewer, Hash: cfg.Hash, Config: cfg, Items: items, Created: now, Expires: now.Add(30 * time.Minute)}) {
		t.Fatal("store previous snapshot")
	}
	page, err := ForYou(context.Background(), viewer, "")
	if err != nil {
		t.Fatal(err)
	}
	var next cursor
	if verify(page.NextCursor, cursorKey, &next, 512) != nil {
		t.Fatal("missing continuation")
	}
	if next.ID == "previous-first-page" {
		t.Fatal("new first-page request reused the old batch instead of applying current seen state")
	}
}

func TestSeenIsHardFilteredAcrossAllCandidatePaths(t *testing.T) {
	conn, viewer, _, items := seenFixture(t)
	now := time.Now().UTC()
	rows := []feed.SeenState{}
	for _, item := range items {
		rows = append(rows, feed.SeenState{UserID: viewer, TopicID: item.ID, LastSeenAt: now, SeenContentAt: now, LastSeenProofID: "first", ExpiresAt: now.Add(30 * 24 * time.Hour)})
	}
	if err := conn.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}
	page, err := ForYou(context.Background(), viewer, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Topics) != 0 || page.NextCursor != "" {
		t.Fatalf("seen pool resurrected: %d rows", len(page.Topics))
	}
}

func TestSeenNewReplyUsesPublicFactsNotDelayedRankProjection(t *testing.T) {
	conn, viewer, author, items := seenFixture(t)
	at := time.Now().UTC().Add(-time.Minute)
	id := items[0].ID
	seen := feed.SeenState{UserID: viewer, TopicID: id, LastSeenAt: at.Add(20 * time.Second), SeenContentAt: at, LastSeenProofID: "old", ExpiresAt: at.Add(30 * 24 * time.Hour)}
	if err := conn.Create(&seen).Error; err != nil {
		t.Fatal(err)
	}
	self := posts.Entity{Id: 954001, TopicId: id, UserId: viewer, PostNo: 2, FirstPublicAt: ptrTime(at.Add(time.Second))}
	if err := conn.Create(&self).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Unscoped().Delete(&self) })
	ctx := context.Background()
	excluded, _, err := seenEligibility(ctx, viewer, []uint64{id})
	if err != nil {
		t.Fatal(err)
	}
	if !excluded[id] {
		t.Fatal("own reply reopened recommendation")
	}
	peer := posts.Entity{Id: 954002, TopicId: id, UserId: author, PostNo: 3, FirstPublicAt: ptrTime(at.Add(2 * time.Second))}
	if err := conn.Create(&peer).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Unscoped().Delete(&peer) })
	excluded, replies, err := seenEligibility(ctx, viewer, []uint64{id})
	if err != nil {
		t.Fatal(err)
	}
	if excluded[id] || !replies[id] {
		t.Fatal("author public reply after card cutoff was swallowed by receipt time or stale projection")
	}

	if err := conn.Create(&topicUserAction.Entity{UserId: viewer, TopicId: id, LikedAt: ptrTime(at), BookmarkedAt: ptrTime(at)}).Error; err != nil {
		t.Fatal(err)
	}
	snap, err := buildSnapshot(ctx, viewer, feedconfig.Current())
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, item := range snap.Items {
		if item.ID == id {
			found = true
			if item.SoftFiltered || item.Repeat {
				t.Fatal("valid new reply retained stale read/action suppression")
			}
		}
	}
	if !found {
		t.Fatal("new public reply never entered the selectable pool")
	}
	if err := conn.Model(&peer).Update("visibility_status", posts.VisibilityModeratorRemoved).Error; err != nil {
		t.Fatal(err)
	}
	excluded, _, err = seenEligibility(ctx, viewer, []uint64{id})
	if err != nil {
		t.Fatal(err)
	}
	if !excluded[id] {
		t.Fatal("hidden reply reopened recommendation")
	}
	if err := conn.Model(&peer).Updates(map[string]any{"visibility_status": posts.VisibilityActive, "first_public_at": at.Add(-time.Second), "updated_at": time.Now()}).Error; err != nil {
		t.Fatal(err)
	}
	excluded, _, err = seenEligibility(ctx, viewer, []uint64{id})
	if err != nil {
		t.Fatal(err)
	}
	if !excluded[id] {
		t.Fatal("edit/restoration of old reply reopened recommendation")
	}
	if err := conn.Model(&peer).Update("first_public_at", at.Add(2*time.Second)).Error; err != nil {
		t.Fatal(err)
	}
	block := users.BlockEntity{OwnerID: author, TargetUserID: viewer}
	if err := conn.Create(&block).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Delete(&block) })
	excluded, _, err = seenEligibility(ctx, viewer, []uint64{id})
	if err != nil {
		t.Fatal(err)
	}
	if !excluded[id] {
		t.Fatal("blocked replier reopened recommendation")
	}
}
func ptrTime(at time.Time) *time.Time { return &at }

func TestSeenProofDurabilityReplayOrderingAndClosure(t *testing.T) {
	conn, viewer, _, items := seenFixture(t)
	now := time.Now().UTC()
	cut := now.Add(-time.Minute)
	claim := seenClaim{ID: "test-proof", User: viewer, Topics: []uint64{items[0].ID}, ContentAt: cut, IssuedAt: now.Add(-5 * time.Second).UnixMilli(), ExpiresAt: now.Add(-5 * time.Second).Add(30 * time.Minute).UnixMilli()}
	patch := SeenPatch{Proof: sign(claim, seenKey), Seen: map[int]int64{0: 1500}}
	ctx := context.Background()
	for _, bad := range []SeenPatch{{Proof: patch.Proof + "x", Seen: patch.Seen}, {Proof: patch.Proof, Seen: map[int]int64{1: 1500}}, {Proof: patch.Proof, Seen: map[int]int64{0: 999}}} {
		if CaptureSeen(ctx, viewer, []SeenPatch{bad}) == nil {
			t.Fatal("invalid claim accepted")
		}
	}
	if CaptureSeen(ctx, viewer+1, []SeenPatch{patch}) == nil {
		t.Fatal("foreign owner accepted")
	}
	if err := CaptureSeen(ctx, viewer, []SeenPatch{patch}); err != nil {
		t.Fatal(err)
	}
	var first feed.SeenState
	if err := conn.First(&first, "user_id = ? AND topic_id = ?", viewer, items[0].ID).Error; err != nil {
		t.Fatal(err)
	}
	if first.SeenContentAt.Sub(cut) != 0 || first.ExpiresAt.Sub(first.LastSeenAt) != 30*24*time.Hour {
		t.Fatal("wrong content cutoff/retention")
	}
	patch.Seen[0] = 3000
	if err := CaptureSeen(ctx, viewer, []SeenPatch{patch}); err != nil {
		t.Fatal(err)
	}
	var replay feed.SeenState
	conn.First(&replay, "user_id = ? AND topic_id = ?", viewer, items[0].ID)
	if !replay.ExpiresAt.Equal(first.ExpiresAt) {
		t.Fatal("same proof retry extended retention")
	}
	claim.ID = "older-proof"
	claim.ContentAt = cut.Add(-time.Second)
	patch.Proof = sign(claim, seenKey)
	if err := CaptureSeen(ctx, viewer, []SeenPatch{patch}); err != nil {
		t.Fatal(err)
	}
	conn.First(&replay, "user_id = ? AND topic_id = ?", viewer, items[0].ID)
	if !replay.SeenContentAt.Equal(first.SeenContentAt) {
		t.Fatal("older cutoff regressed")
	}
	if err := feed.CloseTx(conn, viewer); err != nil {
		t.Fatal(err)
	}
	claim.ID = "new-after-close"
	claim.ContentAt = cut.Add(time.Second)
	patch.Proof = sign(claim, seenKey)
	if !errors.Is(CaptureSeen(ctx, viewer, []SeenPatch{patch}), feed.ErrClosed) {
		t.Fatal("closure fence resurrected")
	}
}

func TestReconcilePreservesSeenCardsOrderAndChecksVisibility(t *testing.T) {
	conn, viewer, _, items := seenFixture(t)
	now := time.Now().UTC()
	if err := conn.Create(&feed.SeenState{UserID: viewer, TopicID: items[0].ID, LastSeenAt: now, SeenContentAt: now, LastSeenProofID: "seen", ExpiresAt: now.Add(30 * 24 * time.Hour)}).Error; err != nil {
		t.Fatal(err)
	}
	requested := []uint64{items[2].ID, items[0].ID, items[1].ID}
	if err := conn.Model(&topics.Entity{}).Where("id = ?", items[1].ID).Update("status", 0).Error; err != nil {
		t.Fatal(err)
	}
	rows, removed, _, err := Reconcile(context.Background(), viewer, requested)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || rows[0].Id != requested[0] || rows[1].Id != requested[1] || len(removed) != 1 || removed[0] != requested[2] {
		t.Fatal("reconcile reordered/removed seen card or kept private card")
	}
}

func TestSeenCleanupIsBoundedAcrossTimeZonesAndErasesClosedOwners(t *testing.T) {
	conn := telemetryDB(t)
	now := time.Now().UTC()
	rows := make([]feed.SeenState, 0, 502)
	for id := uint64(1); id <= 501; id++ {
		rows = append(rows, feed.SeenState{UserID: 55, TopicID: id, LastSeenAt: now.Add(-31 * 24 * time.Hour), SeenContentAt: now.Add(-31 * 24 * time.Hour), LastSeenProofID: "expired", ExpiresAt: now})
	}
	rows = append(rows, feed.SeenState{UserID: 55, TopicID: 1000, LastSeenAt: now.Add(-29 * 24 * time.Hour), SeenContentAt: now.Add(-time.Hour), LastSeenProofID: "live", ExpiresAt: now.Add(time.Hour)})
	if err := conn.CreateInBatches(&rows, 120).Error; err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := Cleanup(ctx, now.In(time.FixedZone("UTC+8", 8*60*60))); err != nil {
		t.Fatal(err)
	}
	live, err := feed.SeenAmong(conn, 55, []uint64{1000}, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(live) != 1 {
		t.Fatal("non-UTC cleanup erased a not-yet-expired state")
	}
	var count int64
	if err := conn.Model(&feed.SeenState{}).Where("user_id = ?", 55).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("one cleanup must delete exactly its 500-row bound, remaining=%d", count)
	}
	if err := Cleanup(ctx, now.In(time.FixedZone("UTC+8", 8*60*60))); err != nil {
		t.Fatal(err)
	}
	live, err = feed.SeenAmong(conn, 55, []uint64{1000}, now)
	if err != nil || len(live) != 1 {
		t.Fatal("second cleanup batch erased a live state", err)
	}
	if err := conn.Transaction(func(tx *gorm.DB) error { return feed.CloseTx(tx, 55) }); err != nil {
		t.Fatal(err)
	}
	if err := purgeClosed(ctx); err != nil {
		t.Fatal(err)
	}
	if err := conn.Model(&feed.SeenState{}).Where("user_id = ?", 55).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("closed owner retained seen state")
	}
}
