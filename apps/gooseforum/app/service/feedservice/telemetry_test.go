package feedservice

import (
	"context"
	"encoding/json"
	"errors"
	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/feedconfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/feed"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"gorm.io/gorm"
	"testing"
	"time"
)

func telemetryDB(t *testing.T) *gorm.DB {
	t.Helper()
	conn := db.Connect()
	if err := conn.AutoMigrate(append(feed.Models(), &topics.Entity{}, &posts.Entity{}, &users.EntityComplete{})...); err != nil {
		t.Fatal(err)
	}
	for _, model := range feed.Models() {
		if err := conn.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(model).Error; err != nil {
			t.Fatal(err)
		}
	}
	preferences.Set("feed.metrics.enabled", true)
	preferences.Set("ranking.enabled", false)
	queueMu.Lock()
	queue = nil
	queueBytes = 0
	sampleQueue = nil
	sampleBytes = 0
	queueMu.Unlock()
	clearViewerMemory(0)
	stopping.Store(false)
	t.Cleanup(func() {
		preferences.Set("feed.metrics.enabled", false)
		preferences.Set("feed.for_you.enabled", false)
		preferences.Set("ranking.enabled", false)
		clearViewerMemory(0)
	})
	return conn
}
func TestTraceAccountPositionExpiryAndCursorBinding(t *testing.T) {
	telemetryDB(t)
	trace := CaptureServe(12, "latest", []Candidate{{ID: 31}, {ID: 32}}, "control", "v2")
	if trace == "" {
		t.Fatal("missing accepted trace")
	}
	if _, err := ParseTrace(trace, 13, time.Now()); err == nil {
		t.Fatal("cross-account accepted")
	}
	if _, err := ParseTrace(trace+"x", 12, time.Now()); err == nil {
		t.Fatal("tampered trace accepted")
	}
	if _, err := ParseTrace(trace, 12, time.Now().Add(7*time.Hour)); err == nil {
		t.Fatal("expired trace accepted")
	}
	if err := CapturePatches(12, []Patch{{Trace: trace, Visible: 4}}); err == nil {
		t.Fatal("foreign bit accepted")
	}
	if err := CapturePatches(12, []Patch{{Trace: trace, Dwell: map[int]int{2: 10}}}); err == nil {
		t.Fatal("foreign dwell accepted")
	}
	if err := CapturePatches(12, []Patch{{Trace: trace, Dwell: map[int]int{0: 601}}}); err == nil {
		t.Fatal("unbounded dwell accepted")
	}
}
func TestObservationReplayAndEarlyOpenKeepPairedDenominators(t *testing.T) {
	conn := telemetryDB(t)
	ctx := context.Background()
	cfg := feedconfig.Current()
	token := CaptureServe(12, "for_you", []Candidate{{ID: 31}}, "treatment", "v2")
	if err := flushTelemetry(ctx, cfg); err != nil {
		t.Fatal(err)
	}
	CaptureOpened(12, 31, token, 0)
	if err := flushTelemetry(ctx, cfg); err != nil {
		t.Fatal(err)
	}
	patch := Patch{Trace: token, Visible: 1, Dwell: map[int]int{0: 10}}
	for i := 0; i < 2; i++ {
		if err := CapturePatches(12, []Patch{patch}); err != nil {
			t.Fatal(err)
		}
		if err := flushTelemetry(ctx, cfg); err != nil {
			t.Fatal(err)
		}
	}
	var rows []feed.MetricsDaily
	if err := conn.Find(&rows).Error; err != nil {
		t.Fatal(err)
	}
	counts := map[string]int64{}
	for _, r := range rows {
		counts[r.Metric] += r.Count
	}
	for _, name := range []string{"served", "visible", "served_open", "visible_open", "read_10s"} {
		if counts[name] != 1 {
			t.Fatalf("%s=%d rows=%+v", name, counts[name], rows)
		}
	}
}
func TestClientCannotCreateOpenedOrReadMetric(t *testing.T) {
	conn := telemetryDB(t)
	token := CaptureServe(12, "latest", []Candidate{{ID: 31}}, "control", "v2")
	if err := CapturePatches(12, []Patch{{Trace: token, Visible: 1, Dwell: map[int]int{0: 600}}}); err != nil {
		t.Fatal(err)
	}
	if err := flushTelemetry(context.Background(), feedconfig.Current()); err != nil {
		t.Fatal(err)
	}
	var n int64
	conn.Model(&feed.MetricsDaily{}).Where("metric IN ?", []string{"served_open", "read_10s"}).Count(&n)
	if n != 0 {
		t.Fatal("client invented a successful detail open")
	}
}
func TestCloseFenceWinsOverQueuedServeAndEraseKeepsFence(t *testing.T) {
	conn := telemetryDB(t)
	CaptureServe(12, "latest", []Candidate{{ID: 31}}, "control", "v2")
	if err := conn.Transaction(func(tx *gorm.DB) error { return feed.CloseTx(tx, 12) }); err != nil {
		t.Fatal(err)
	}
	if err := flushTelemetry(context.Background(), feedconfig.Current()); err != nil {
		t.Fatal(err)
	}
	if err := purgeClosed(context.Background()); err != nil {
		t.Fatal(err)
	}
	var n int64
	conn.Model(&feed.Serve{}).Count(&n)
	if n != 0 {
		t.Fatal("queued raw recreated after close")
	}
	err := conn.Transaction(func(tx *gorm.DB) error { return feed.LockOwnerTx(tx, 12) })
	if !errors.Is(err, feed.ErrClosed) {
		t.Fatalf("close fence removed: %v", err)
	}
}
func TestNativeActionsCompensateOnceAndPublicationIsNotActivity(t *testing.T) {
	conn := telemetryDB(t)
	now := time.Now()
	events := []feed.Event{{ID: feed.NewID(), UserID: 12, ObjectID: 1, TopicID: 1, Kind: "like", SourceVersion: 3, Active: true, CreatedAt: now, ExpiresAt: now.Add(30 * 24 * time.Hour)}, {ID: feed.NewID(), UserID: 12, ObjectID: 1, TopicID: 1, Kind: "like", SourceVersion: 1, Active: true, CreatedAt: now, ExpiresAt: now.Add(30 * 24 * time.Hour)}, {ID: feed.NewID(), UserID: 12, ObjectID: 1, TopicID: 1, Kind: "like", SourceVersion: 2, Active: false, CreatedAt: now, ExpiresAt: now.Add(30 * 24 * time.Hour)}, {ID: feed.NewID(), UserID: 13, ObjectID: 2, TopicID: 1, Kind: "public_reply", SourceVersion: 1, Active: true, CreatedAt: now, ExpiresAt: now.Add(30 * 24 * time.Hour)}}
	if err := conn.Create(&events).Error; err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err := rollupEvents(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	var user feed.UserDaily
	conn.First(&user, "user_id = ?", 12)
	if user.Likes != 1 || !user.Active {
		t.Fatalf("toggle contribution %+v", user)
	}
	conn.First(&user, "user_id = ?", 13) // use fresh variable because GORM retains primary keys
	var author feed.UserDaily
	if err := conn.First(&author, "user_id = ?", 13).Error; err != nil {
		t.Fatal(err)
	}
	if author.PublicReplies != 1 || author.Active {
		t.Fatalf("approval invented author visit %+v", author)
	}
}
func TestRetentionDeletesCompositeRowsAndStopsCapture(t *testing.T) {
	conn := telemetryDB(t)
	old := time.Now().Add(-time.Hour)
	if err := conn.Create(&feed.ViewFact{TopicID: 1, UserID: 12, ViewedAt: old, ExpiresAt: old}).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Create(&feed.ActionResult{UserID: 12, ObjectID: 1, Kind: "like", ExpiresAt: old}).Error; err != nil {
		t.Fatal(err)
	}
	if err := Cleanup(context.Background(), time.Now()); err != nil {
		t.Fatal(err)
	}
	var n int64
	conn.Model(&feed.ViewFact{}).Count(&n)
	if n != 0 {
		t.Fatal("expired composite view remains")
	}
	preferences.Set("feed.metrics.enabled", false)
	CaptureView(12, 1, 13)
	CaptureActivity(12)
	if got := CaptureServe(12, "latest", []Candidate{{ID: 1}}, "control", "v2"); got != "" {
		t.Fatal("capture continued after kill switch")
	}
	if err := flushTelemetry(context.Background(), feedconfig.Current()); err != nil {
		t.Fatal(err)
	}
}
func TestCreditToggleKeepsAgeAndExpiredActivationRenews(t *testing.T) {
	conn := telemetryDB(t)
	preferences.Set("ranking.enabled", true)
	now := time.Now()
	for _, active := range []bool{true, false, true} {
		if err := conn.Transaction(func(tx *gorm.DB) error { return feed.CreditTx(tx, 12, 1, "like", active, now) }); err != nil {
			t.Fatal(err)
		}
	}
	var row feed.ActionCredit
	conn.First(&row)
	if !row.Active || !row.CreditAt.Equal(now) {
		t.Fatalf("credit refreshed %+v", row)
	}
	later := now.Add(25 * time.Hour)
	if err := conn.Transaction(func(tx *gorm.DB) error { return feed.CreditTx(tx, 12, 1, "like", true, later) }); err != nil {
		t.Fatal(err)
	}
	conn.First(&row)
	if !row.CreditAt.Equal(later) {
		t.Fatal("expired credit failed to renew")
	}
}
func TestReplayUsesRecordedRelaxationPhase(t *testing.T) {
	pool := []Candidate{{ID: 1, Author: 1, Features: Features{0, 10000}, SoftFiltered: true}, {ID: 2, Author: 2, Features: Features{0, 1}}, {ID: 3, Author: 3, Fallback: true}}
	live := SelectCandidates(pool, feedconfig.Default().Weights, 5)
	data, err := json.Marshal(pool)
	if err != nil {
		t.Fatal(err)
	}
	var restored []Candidate
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatal(err)
	}
	replay := SelectCandidates(restored, feedconfig.Default().Weights, 5)
	if len(live) != len(replay) || len(live) != 3 {
		t.Fatalf("relaxation lost %+v %+v", live, replay)
	}
	for i := range live {
		if live[i].ID != replay[i].ID {
			t.Fatal("replay differs")
		}
	}
}
func TestQuietOldTopicHasNoRepeatingTimerAndSharedInterleaving(t *testing.T) {
	now := time.Now()
	score := ScoreRank(RankInput{Author: 1, FirstPublicAt: now.Add(-30 * 24 * time.Hour), Participants: []Participant{{UserID: 2, Replies: 50000}}}, now)
	if !score.NextDue.IsZero() || score.Hot <= 0 {
		t.Fatalf("static topic %+v", score)
	}
	pool := []Candidate{{ID: 1, Author: 1, Features: Features{10000}}, {ID: 2, Author: 1, Features: Features{9000}}, {ID: 3, Author: 2, Features: Features{8000}}}
	w := feedconfig.Default().Weights
	for _, c := range InterleaveCandidates(pool, w, w, 0) {
		if c.Team != "shared" {
			t.Fatal("identical lists fabricated a winner")
		}
	}
}

func TestApprovalCannotAttributeToModeratorTrace(t *testing.T) {
	conn := telemetryDB(t)
	ctx := feed.WithAttribution(context.Background(), feed.Attribution{UserID: 99, TopicID: 1, ServeID: "moderator", Position: 0, Source: "exact"})
	if err := conn.WithContext(ctx).Transaction(func(tx *gorm.DB) error { return feed.EventTx(tx, 12, 1, 2, "public_reply", true) }); err != nil {
		t.Fatal(err)
	}
	var row feed.Event
	conn.First(&row)
	if row.UserID != 12 || row.ServeID != "" || row.Source != "unattributed" {
		t.Fatalf("moderator credited %+v", row)
	}
}
func TestMaterializedParameterChangeDisablesOldPromotion(t *testing.T) {
	telemetryDB(t)
	preferences.Set("ranking.enabled", true)
	feedconfig.SetRankReady(true)
	if !feedconfig.RankReady() {
		t.Fatal("version not promoted")
	}
	before := feedconfig.Current()
	preferences.Set("ranking.hot.like_weight", 7.0)
	after := feedconfig.Current()
	t.Cleanup(func() { preferences.Set("ranking.hot.like_weight", 6.0); feedconfig.SetRankReady(false) })
	if before.RankHash == after.RankHash || feedconfig.RankReady() {
		t.Fatal("changed weights continued serving old rank version")
	}
}

func TestFullServedPageFitsBoundAndSharedExplorationHasNoTeamWins(t *testing.T) {
	conn := telemetryDB(t)
	items := []Candidate{}
	for i := 0; i < 20; i++ {
		items = append(items, Candidate{ID: uint64(123456789 + i), Author: uint64(987654321 + i), Features: Features{10000, 9876, 5432, 9876, 1234, 6789, 5432}, Sources: 63, Reason: "following", Team: "A", Base: 34567, Adjusted: 12345})
	}
	items[0].Explore = true
	items[0].Team = "shared"
	trace := CaptureServe(12, "for_you", items, "control", "v2")
	if trace == "" {
		t.Fatal("full 20-item page exceeded queue bound")
	}
	parsed, err := ParseTrace(trace, 12, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err = flushTelemetry(context.Background(), feedconfig.Current()); err != nil {
		t.Fatal(err)
	}
	if err = mergeObservation(conn, observationPatch{Trace: parsed, Opened: 3, Visible: 3, Dwell: map[int]int{0: 10, 1: 10}}); err != nil {
		t.Fatal(err)
	}
	var metrics []feed.MetricsDaily
	if err = conn.Find(&metrics).Error; err != nil {
		t.Fatal(err)
	}
	found := false
	for _, m := range metrics {
		if m.Metric == "team_A_visible_open" {
			found = true
			if m.Count != 1 {
				t.Fatalf("shared exploration supplied team win %+v", m)
			}
		}
	}
	if !found {
		t.Fatal("team comparison missing")
	}
	replay, err := ReplaySample(context.Background(), parsed.ID, feedconfig.Default().Weights)
	if err != nil || replay.CompleteCandidates || len(replay.Items) != 20 {
		t.Fatalf("served subset fallback %+v %v", replay, err)
	}
}

func TestAccountClosureDoesNotBlockModeratorsPublishingRetainedContent(t *testing.T) {
	conn := telemetryDB(t)
	if err := conn.Transaction(func(tx *gorm.DB) error { return feed.CloseTx(tx, 12) }); err != nil {
		t.Fatal(err)
	}
	if err := conn.Transaction(func(tx *gorm.DB) error { return feed.EventTx(tx, 12, 1, 2, "public_reply", true) }); err != nil {
		t.Fatalf("closed author analytics blocked publication: %v", err)
	}
	var n int64
	conn.Model(&feed.Event{}).Where("user_id = ?", 12).Count(&n)
	if n != 0 {
		t.Fatal("closed author personal events recreated")
	}
}

func TestZeroRolloutStopsExistingTreatmentAndAbortsChangedPeriod(t *testing.T) {
	conn := telemetryDB(t)
	preferences.Set("ranking.enabled", true)
	preferences.Set("feed.for_you.enabled", true)
	preferences.Set("feed.for_you.rollout_percent", 100)
	preferences.Set("feed.experiments.period", "rollback-test")
	feedconfig.SetRankReady(true)
	t.Cleanup(func() {
		preferences.Set("feed.for_you.rollout_percent", 20)
		preferences.Set("feed.experiments.period", "default-entry-v1")
		feedconfig.SetRankReady(false)
	})
	treatment, _, err := assignDefault(context.Background(), 12)
	if err != nil || !treatment {
		t.Fatalf("initial treatment %v %v", treatment, err)
	}
	preferences.Set("feed.for_you.rollout_percent", 0)
	if treatment, _, err = assignDefault(context.Background(), 12); err != nil || treatment {
		t.Fatalf("0 rollout retained treatment %v %v", treatment, err)
	}
	if err = abortMismatchedPeriods(context.Background(), feedconfig.Current().Hash); err != nil {
		t.Fatal(err)
	}
	var period feed.ExperimentPeriod
	if err = conn.First(&period, "id = ?", "rollback-test").Error; err != nil || period.Aborted == "" {
		t.Fatalf("changed period still live %+v %v", period, err)
	}
}
