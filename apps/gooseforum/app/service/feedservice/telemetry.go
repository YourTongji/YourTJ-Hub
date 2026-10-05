package feedservice

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/bits"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/feedconfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/feed"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var processEpoch = feed.NewID()
var traceKey = []byte(preferences.GetString("app.signingKey") + ":feed-trace-v1:" + processEpoch)
var cursorKey = []byte(preferences.GetString("app.signingKey") + ":feed-cursor-v1:" + processEpoch)
var ErrInvalidTrace = errors.New("invalid feed trace")
var ErrRateLimited = errors.New("feed event rate limited")
var ErrQueueFull = errors.New("feed telemetry queue full")

type Trace struct {
	ID         string   `json:"i"`
	User       uint64   `json:"u"`
	Topics     []uint64 `json:"t"`
	Feed       string   `json:"f"`
	Hash       string   `json:"h"`
	Experiment string   `json:"e"`
	Variant    string   `json:"v"`
	Created    int64    `json:"c"`
	Expires    int64    `json:"x"`
}

func sign(value any, key []byte) string {
	data, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	mac := hmac.New(sha256.New, key)
	mac.Write(data)
	return base64.RawURLEncoding.EncodeToString(data) + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
func verify(token string, key []byte, out any, maxBytes int) error {
	if len(token) > maxBytes || len(token) < 2 {
		return ErrInvalidTrace
	}
	split := -1
	for i, c := range token {
		if c == '.' {
			split = i
			break
		}
	}
	if split < 0 {
		return ErrInvalidTrace
	}
	data, err := base64.RawURLEncoding.DecodeString(token[:split])
	if err != nil {
		return ErrInvalidTrace
	}
	sig, err := base64.RawURLEncoding.DecodeString(token[split+1:])
	if err != nil {
		return ErrInvalidTrace
	}
	mac := hmac.New(sha256.New, key)
	mac.Write(data)
	if !hmac.Equal(sig, mac.Sum(nil)) {
		return ErrInvalidTrace
	}
	if json.Unmarshal(data, out) != nil {
		return ErrInvalidTrace
	}
	return nil
}
func ParseTrace(token string, uid uint64, now time.Time) (Trace, error) {
	var t Trace
	if err := verify(token, traceKey, &t, 2048); err != nil {
		return t, err
	}
	if t.User != uid || uid == 0 || len(t.Topics) == 0 || len(t.Topics) > 20 || t.Expires <= now.Unix() || t.Created > now.Unix() || t.Expires-t.Created > 6*3600 {
		return t, ErrInvalidTrace
	}
	return t, nil
}
func AttributionContext(ctx context.Context, uid, topicID uint64, token string, position int) context.Context {
	source := "exact"
	if token == "" {
		if e, ok := inferred.get(memoryKey{User: uid, Topic: topicID}, time.Now()); ok {
			token = e.Trace
			position = e.Position
			source = "inferred"
		}
	}
	t, err := ParseTrace(token, uid, time.Now())
	if err != nil || position < 0 || position >= len(t.Topics) || t.Topics[position] != topicID {
		return ctx
	}
	return feed.WithAttribution(ctx, feed.Attribution{UserID: uid, ServeID: t.ID, TopicID: topicID, Position: position, Source: source})
}

type queued struct {
	kind     string
	uid      uint64
	data     []byte
	accepted time.Time
	attempt  int
}

var queueMu sync.Mutex
var queue []queued
var queueBytes int
var dropped atomic.Uint64
var accepted atomic.Uint64

func enqueue(kind string, uid uint64, value any) bool {
	if stopping.Load() || !feedconfig.Current().Metrics {
		return false
	}
	data, err := json.Marshal(value)
	if err != nil || len(data) > 4096 {
		dropped.Add(1)
		return false
	}
	queueMu.Lock()
	defer queueMu.Unlock()
	if queueBytes+len(data) > 8<<20 {
		dropped.Add(1)
		return false
	}
	queue = append(queue, queued{kind: kind, uid: uid, data: data, accepted: time.Now()})
	queueBytes += len(data)
	accepted.Add(1)
	return true
}

func CaptureServe(uid uint64, actualFeed string, items []Candidate, variant, capability string) string {
	return CaptureServeVersion(uid, actualFeed, items, variant, capability, "")
}
func CaptureServeVersion(uid uint64, actualFeed string, items []Candidate, variant, capability, hash string, versions ...feedconfig.Config) string {
	c := feedconfig.Current()
	if !c.Metrics {
		return ""
	}
	if len(versions) > 0 {
		c = versions[0]
	}
	if !c.Metrics || uid == 0 || len(items) == 0 || len(items) > 20 {
		return ""
	}
	if hash != "" {
		c.Hash = hash
	}
	now := time.Now()
	ids := make([]uint64, len(items))
	for i, item := range items {
		ids[i] = item.ID
	}
	t := Trace{ID: feed.NewID(), User: uid, Topics: ids, Feed: actualFeed, Hash: c.Hash, Experiment: c.Experiment, Variant: variant, Created: now.Unix(), Expires: now.Add(6 * time.Hour).Unix()}
	data, err := encodeServed(items)
	if err != nil {
		dropped.Add(1)
		return ""
	}
	row := feed.Serve{ID: t.ID, UserID: uid, Feed: actualFeed, Hash: c.Hash, RankHash: c.RankHash, Experiment: c.Experiment, Variant: variant, Capability: capability, WeightVariant: weightVariant(uid, c), Applied: true, Items: data, CreatedAt: now, ExpiresAt: now.Add(time.Duration(c.RetentionDays) * 24 * time.Hour)}
	if !enqueue("serve", uid, row) {
		return ""
	}
	return sign(t, traceKey)
}

type Patch struct {
	Trace   string      `json:"trace"`
	Visible uint32      `json:"visibleMask"`
	Dwell   map[int]int `json:"dwell,omitempty"`
}
type observationPatch struct {
	Trace   Trace
	Visible uint32
	Opened  uint32
	Dwell   map[int]int
}

var rateMu sync.Mutex

type rateState struct {
	tokens float64
	at     time.Time
}

var eventRates = map[uint64]rateState{}
var globalEvents = rateState{tokens: 400, at: time.Now()}

func allowEvents(uid uint64, now time.Time) bool {
	rateMu.Lock()
	defer rateMu.Unlock()
	if len(eventRates) > 16384 {
		for u, s := range eventRates {
			if now.Sub(s.at) > time.Minute {
				delete(eventRates, u)
			}
		}
		if len(eventRates) > 16384 {
			return false
		}
	}
	globalEvents.tokens = min(400, globalEvents.tokens+now.Sub(globalEvents.at).Seconds()*200)
	globalEvents.at = now
	if globalEvents.tokens < 1 {
		return false
	}
	s, ok := eventRates[uid]
	if !ok {
		s = rateState{24, now}
	}
	s.tokens = min(24, s.tokens+now.Sub(s.at).Seconds()*.2)
	s.at = now
	if s.tokens < 1 {
		eventRates[uid] = s
		return false
	}
	s.tokens--
	globalEvents.tokens--
	eventRates[uid] = s
	return true
}
func CapturePatches(uid uint64, patches []Patch) error {
	if !feedconfig.Current().Metrics {
		return nil
	}
	if len(patches) > 50 {
		return ErrInvalidTrace
	}
	now := time.Now()
	if !allowEvents(uid, now) {
		return ErrRateLimited
	}
	validated := make([]observationPatch, 0, len(patches))
	for _, p := range patches {
		t, err := ParseTrace(p.Trace, uid, now)
		if err != nil {
			return err
		}
		if p.Visible>>len(t.Topics) != 0 || len(p.Dwell) > 20 {
			return ErrInvalidTrace
		}
		for pos, seconds := range p.Dwell {
			if pos < 0 || pos >= len(t.Topics) || seconds < 0 || seconds > 600 || seconds%5 != 0 {
				return ErrInvalidTrace
			}
		}
		validated = append(validated, observationPatch{Trace: t, Visible: p.Visible, Dwell: p.Dwell})
	}
	for _, p := range validated {
		if !enqueue("observation", uid, p) {
			return ErrQueueFull
		}
	}
	return nil
}

// Opened is accepted only after the detail owner has successfully authorized
// and rendered a public topic. Client patches cannot set the opened mask.
func CaptureOpened(uid, topicID uint64, token string, position int) {
	if token == "" {
		if e, ok := inferred.get(memoryKey{User: uid, Topic: topicID}, time.Now()); ok {
			token = e.Trace
			position = e.Position
		}
	}
	if !feedconfig.Current().Metrics {
		return
	}
	t, err := ParseTrace(token, uid, time.Now())
	if err != nil || position < 0 || position >= len(t.Topics) || t.Topics[position] != topicID {
		return
	}
	enqueue("observation", uid, observationPatch{Trace: t, Opened: 1 << position})
}
func CaptureView(uid, topicID, authorID uint64) {
	if uid == 0 || uid == authorID || !feedconfig.Current().Metrics {
		return
	}
	now := time.Now()
	captureViewFact(feed.ViewFact{TopicID: topicID, UserID: uid, ViewedAt: now, ExpiresAt: now.Add(24 * time.Hour)})
}
func CaptureActivity(uid uint64) {
	if uid == 0 {
		return
	}
	c := feedconfig.Current()
	if !c.Metrics {
		return
	}
	now := time.Now()
	if dailySeen.once(memoryKey{User: uid, Name: localDay(now)}, now, 24*time.Hour) {
		enqueue("activity", uid, feed.UserDaily{UserID: uid, Day: localDay(now), Active: true, ExpiresAt: now.Add(time.Duration(c.RetentionDays) * 24 * time.Hour)})
	}
}
func localDay(t time.Time) string { return t.In(time.FixedZone("UTC+8", 8*3600)).Format("2006-01-02") }

func flushTelemetry(ctx context.Context, c feedconfig.Config) error {
	queueMu.Lock()
	if !c.Metrics {
		dropped.Add(uint64(len(queue)))
		queue = nil
		queueBytes = 0
		dropped.Add(uint64(len(sampleQueue)))
		sampleQueue = nil
		sampleBytes = 0
		queueMu.Unlock()
		clearViewerMemory(0)
		return rollupEvents(ctx)
	}
	n := min(50, len(queue))
	batch := append([]queued(nil), queue[:n]...)
	for _, q := range batch {
		queueBytes -= len(q.data)
	}
	copy(queue, queue[n:])
	clear(queue[len(queue)-n:])
	queue = queue[:len(queue)-n]
	queueMu.Unlock()
	for _, q := range batch {
		if time.Since(q.accepted) > 5*time.Second {
			dropped.Add(1)
			continue
		}
		err := db.ConnectContext(ctx).Transaction(func(tx *gorm.DB) error {
			if e := feed.LockOwnerTx(tx, q.uid); e != nil {
				return e
			}
			switch q.kind {
			case "serve":
				var row feed.Serve
				if e := json.Unmarshal(q.data, &row); e != nil {
					return e
				}
				created := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&row)
				if created.Error != nil {
					return created.Error
				}
				if created.RowsAffected == 0 {
					return nil
				}
				items, err := decodeServed(row.Items)
				if err != nil {
					return err
				}
				return addMetric(tx, row, "served", int64(len(items)))
			case "observation":
				var p observationPatch
				if e := json.Unmarshal(q.data, &p); e != nil {
					return e
				}
				return mergeObservation(tx, p)
			case "view":
				var row feed.ViewFact
				if e := json.Unmarshal(q.data, &row); e != nil {
					return e
				}
				result := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "topic_id"}, {Name: "user_id"}}, DoUpdates: clause.AssignmentColumns([]string{"viewed_at", "expires_at"}), Where: clause.Where{Exprs: []clause.Expression{clause.Expr{SQL: "topic_view_fact.viewed_at <= ?", Vars: []any{row.ViewedAt.Add(-5 * time.Minute)}}}}}).Create(&row)
				if result.Error != nil {
					return result.Error
				}
				if result.RowsAffected > 0 {
					return feed.MarkTx(tx, row.TopicID)
				}
			case "activity":
				var row feed.UserDaily
				if e := json.Unmarshal(q.data, &row); e != nil {
					return e
				}
				return tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "user_id"}, {Name: "day"}}, DoUpdates: clause.Assignments(map[string]any{"active": true})}).Create(&row).Error
			}
			return nil
		})
		if err != nil {
			if !errors.Is(err, feed.ErrClosed) && q.attempt < 2 && time.Since(q.accepted) < 5*time.Second {
				q.attempt++
				queueMu.Lock()
				if queueBytes+len(q.data) <= 8<<20 {
					queue = append(queue, q)
					queueBytes += len(q.data)
				} else {
					dropped.Add(1)
				}
				queueMu.Unlock()
			} else {
				dropped.Add(1)
			}
		}
	}
	if err := flushViews(ctx, c); err != nil {
		return err
	}
	if err := flushSample(ctx, c); err != nil {
		return err
	}
	return rollupEvents(ctx)
}

func addMetric(tx *gorm.DB, s feed.Serve, metric string, delta int64) error {
	if delta == 0 {
		return nil
	}
	row := feed.MetricsDaily{WeightVariant: s.WeightVariant, Capability: s.Capability, RankHash: s.RankHash, Day: localDay(s.CreatedAt), Feed: s.Feed, Hash: s.Hash, Experiment: s.Experiment, Variant: s.Variant, Metric: metric, Count: delta}
	return tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "day"}, {Name: "feed"}, {Name: "hash"}, {Name: "experiment"}, {Name: "variant"}, {Name: "metric"}, {Name: "capability"}, {Name: "rank_hash"}, {Name: "weight_variant"}}, DoUpdates: clause.Assignments(map[string]any{"count": gorm.Expr("feed_metrics_daily.count + ?", delta)})}).Create(&row).Error
}
func mergeObservation(tx *gorm.DB, p observationPatch) error {
	var serve feed.Serve
	if err := tx.First(&serve, "id = ? AND user_id = ? AND expires_at > ?", p.Trace.ID, p.Trace.User, time.Now()).Error; err != nil {
		return err
	}
	row := feed.Observation{ID: serve.ID, UserID: serve.UserID, CreatedAt: serve.CreatedAt, ExpiresAt: serve.ExpiresAt, Dwell: "{}", AppliedDwell: "{}"}
	if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&row).Error; err != nil {
		return err
	}
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&row, "id = ?", serve.ID).Error; err != nil {
		return err
	}
	before := row
	row.Visible |= p.Visible
	row.Opened |= p.Opened
	dwell := map[int]int{}
	_ = json.Unmarshal([]byte(row.Dwell), &dwell)
	for pos, v := range p.Dwell {
		if row.Opened&(1<<pos) != 0 && v > dwell[pos] {
			dwell[pos] = v
		}
	}
	data, err := json.Marshal(dwell)
	if err != nil {
		return err
	}
	row.Dwell = string(data)
	for metric, delta := range map[string]int64{"visible": int64(bits.OnesCount32(row.Visible) - bits.OnesCount32(before.Visible)), "served_open": int64(bits.OnesCount32(row.Opened) - bits.OnesCount32(before.Opened)), "visible_open": int64(bits.OnesCount32(row.Visible&row.Opened) - bits.OnesCount32(before.Visible&before.Opened))} {
		if err := addMetric(tx, serve, metric, delta); err != nil {
			return err
		}
	}
	items, err := decodeServed(serve.Items)
	if err != nil {
		return err
	}
	visibleTopics := []uint64{}
	for pos, item := range items {
		if row.Visible & ^before.Visible & (1<<pos) != 0 {
			visibleTopics = append(visibleTopics, item.ID)
		}
	}
	if err := captureNewTopicVisibilityTx(tx, visibleTopics, serve.UserID, time.Now()); err != nil {
		return err
	}
	for pos, item := range items {
		if item.Team != "A" && item.Team != "B" || item.Explore {
			continue
		}
		bit := uint32(1 << pos)
		for name, delta := range map[string]int64{"team_" + item.Team + "_served_open": int64(bits.OnesCount32(row.Opened&bit) - bits.OnesCount32(before.Opened&bit)), "team_" + item.Team + "_visible_open": int64(bits.OnesCount32(row.Opened&row.Visible&bit) - bits.OnesCount32(before.Opened&before.Visible&bit))} {
			if e := addMetric(tx, serve, name, delta); e != nil {
				return e
			}
		}
	}
	old := map[int]int{}
	_ = json.Unmarshal([]byte(before.Dwell), &old)
	for pos, v := range dwell {
		previous := 0
		if before.Visible&(1<<pos) != 0 {
			previous = old[pos]
		}
		if row.Visible&(1<<pos) != 0 && v > previous {
			if e := addMetric(tx, serve, "foreground_read_seconds", int64(v-previous)); e != nil {
				return e
			}
		}
		if v >= 10 && row.Visible&(1<<pos) != 0 && (old[pos] < 10 || before.Visible&(1<<pos) == 0) {
			if err := addMetric(tx, serve, "read_10s", 1); err != nil {
				return err
			}
		}
	}
	if p.Visible != 0 {
		rememberVisible(serve, p.Visible)
	}
	return tx.Save(&row).Error
}

func rollupEvents(ctx context.Context) error {
	var rows []feed.Event
	if err := db.ConnectContext(ctx).Where("applied = ? AND expires_at > ?", false, time.Now()).Order("user_id").Order("source_version").Limit(50).Find(&rows).Error; err != nil {
		return err
	}
	for _, event := range rows {
		if err := db.ConnectContext(ctx).Transaction(func(tx *gorm.DB) error {
			if e := feed.LockOwnerTx(tx, event.UserID); e != nil {
				if errors.Is(e, feed.ErrClosed) {
					return tx.Delete(&event).Error
				}
				return e
			}
			var row feed.Event
			if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&row, "id = ?", event.ID).Error; e != nil {
				return e
			}
			if row.Applied {
				return nil
			}
			if row.Kind == "like" || row.Kind == "bookmark" || row.Kind == "post_like" || row.Kind == "post_bookmark" {
				if e := applyAction(tx, row); e != nil {
					return e
				}
			}
			if row.Kind == "intent" || row.Kind == "like" || row.Kind == "bookmark" || row.Kind == "post_like" || row.Kind == "post_bookmark" {
				if e := markActiveTx(tx, row.UserID, row.CreatedAt, row.ExpiresAt); e != nil {
					return e
				}
			}
			if row.Kind == "public_topic" || row.Kind == "public_reply" {
				if e := captureNewTopicPublicationTx(tx, row); e != nil {
					return e
				}
				daily := feed.UserDaily{UserID: row.UserID, Day: localDay(row.CreatedAt), ExpiresAt: row.ExpiresAt}
				if e := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&daily).Error; e != nil {
					return e
				}
				col := "public_replies"
				if row.Kind == "public_topic" {
					col = "public_topics"
				}
				if e := tx.Model(&feed.UserDaily{}).Where("user_id = ? AND day = ?", row.UserID, daily.Day).UpdateColumn(col, gorm.Expr(col+" + 1")).Error; e != nil {
					return e
				}
			}
			if (row.Kind == "public_topic" || row.Kind == "public_reply") && row.ServeID != "" {
				var serve feed.Serve
				if e := tx.First(&serve, "id = ? AND user_id = ? AND expires_at > ?", row.ServeID, row.UserID, time.Now()).Error; e == nil {
					if e = addMetric(tx, serve, row.Source+"_"+row.Kind, 1); e != nil {
						return e
					}
				} else if !errors.Is(e, gorm.ErrRecordNotFound) {
					return e
				}
			}
			return tx.Model(&row).UpdateColumn("applied", true).Error
		}); err != nil {
			return err
		}
	}
	return nil
}

func TelemetryHealth() map[string]any {
	queueMu.Lock()
	defer queueMu.Unlock()
	lag := int64(0)
	if len(queue) > 0 {
		lag = time.Since(queue[0].accepted).Milliseconds()
	}
	return map[string]any{"accepted": accepted.Load(), "dropped": dropped.Load(), "queueBytes": queueBytes, "queueLength": len(queue), "oldestQueuedMs": lag, "sampleQueueBytes": sampleBytes, "sampleQueueLength": len(sampleQueue), "backgroundFailures": backgroundFailures.Load(), "lastRankAt": lastRankAt.Load(), "previousEpochIncomplete": previousEpochIncomplete.Load(), "epoch": processEpoch}
}

// Cleanup performs one bounded SQL delete per raw table. Composite keys use a
// subquery rather than 500 individual deletes; remaining batches run next tick.
var cleanupBacklog atomic.Bool

func Cleanup(ctx context.Context, now time.Time) error {
	backlog := false
	conn := db.ConnectContext(ctx)
	for _, table := range feed.RawTables {
		keys := rawKeys(table)
		tuple := "(" + strings.Join(keys, ",") + ")"
		sub := conn.Table(table).Select(strings.Join(keys, ",")).Where("expires_at <= ?", now).Limit(500)
		result := conn.Table(table).Where(tuple+" IN (?)", sub).Delete(map[string]any{})
		if result.Error != nil {
			return result.Error
		}
		backlog = backlog || result.RowsAffected == 500
	}
	cleanupBacklog.Store(backlog)
	return purgeClosed(ctx)
}
func rawKeys(table string) []string {
	switch table {
	case "feed_new_topic_outcome":
		return []string{"topic_id"}
	case "feed_actor_work", "feed_owner":
		return []string{"user_id"}
	case "topic_view_fact":
		return []string{"topic_id", "user_id"}
	case "topic_action_credit":
		return []string{"topic_id", "user_id", "kind"}
	case "feed_action_result":
		return []string{"user_id", "object_id", "kind"}
	case "feed_user_daily":
		return []string{"user_id", "day"}
	case "feed_experiment_assignment":
		return []string{"experiment", "user_id"}
	default:
		return []string{"id"}
	}
}
func purgeClosed(ctx context.Context) error {
	conn := db.ConnectContext(ctx)
	var o feed.Owner
	if err := conn.Where("closed = ? AND purged = ?", true, false).First(&o).Error; errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	} else if err != nil {
		return err
	}
	remaining := false
	for _, table := range feed.RawTables {
		if table == "feed_actor_work" || table == "feed_owner" || table == "feed_rank_snapshot" || table == "feed_period_progress" {
			continue
		}
		keys := rawKeys(table)
		sub := conn.Table(table).Select(strings.Join(keys, ",")).Where("user_id = ?", o.UserID).Limit(500)
		r := conn.Table(table).Where("("+strings.Join(keys, ",")+") IN (?)", sub).Delete(map[string]any{})
		if r.Error != nil {
			return r.Error
		}
		remaining = remaining || r.RowsAffected == 500
	}
	invalidateViewer(o.UserID)
	clearViewerMemory(o.UserID)
	if !remaining {
		return conn.Model(&o).UpdateColumn("purged", true).Error
	}
	return nil
}

func DebugTrace(token string, uid uint64) (Trace, error) {
	t, err := ParseTrace(token, uid, time.Now())
	if err != nil {
		return t, fmt.Errorf("trace: %w", err)
	}
	return t, nil
}

func markActiveTx(tx *gorm.DB, uid uint64, at, expiry time.Time) error {
	row := feed.UserDaily{UserID: uid, Day: localDay(at), Active: true, ExpiresAt: expiry}
	return tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "user_id"}, {Name: "day"}}, DoUpdates: clause.Assignments(map[string]any{"active": true})}).Create(&row).Error
}
func applyAction(tx *gorm.DB, event feed.Event) error {
	var state feed.ActionResult
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&state, "user_id = ? AND object_id = ? AND kind = ?", event.UserID, event.ObjectID, event.Kind).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if state.Version >= event.SourceVersion {
		return nil
	}
	if err != nil || !state.ExpiresAt.After(time.Now()) {
		state = feed.ActionResult{UserID: event.UserID, ObjectID: event.ObjectID, Kind: event.Kind, Day: localDay(event.CreatedAt), ServeID: event.ServeID, Position: event.Position, Source: event.Source, ExpiresAt: event.ExpiresAt}
	}
	delta := int64(0)
	if event.Active && !state.Active {
		delta = 1
	}
	if !event.Active && state.Active {
		delta = -1
	}
	if delta != 0 {
		daily := feed.UserDaily{UserID: event.UserID, Day: state.Day, ExpiresAt: state.ExpiresAt}
		if e := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&daily).Error; e != nil {
			return e
		}
		col := "likes"
		if event.Kind == "bookmark" || event.Kind == "post_bookmark" {
			col = "bookmarks"
		}
		if e := tx.Model(&feed.UserDaily{}).Where("user_id = ? AND day = ?", event.UserID, state.Day).UpdateColumn(col, gorm.Expr("CASE WHEN "+col+" + ? < 0 THEN 0 ELSE "+col+" + ? END", delta, delta)).Error; e != nil {
			return e
		}
		if state.ServeID != "" {
			var serve feed.Serve
			if e := tx.First(&serve, "id = ? AND user_id = ? AND expires_at > ?", state.ServeID, state.UserID, time.Now()).Error; e == nil {
				if e = addMetric(tx, serve, state.Source+"_"+event.Kind, delta); e != nil {
					return e
				}
			} else if !errors.Is(e, gorm.ErrRecordNotFound) {
				return e
			}
		}
	}
	state.Active = event.Active
	state.Version = event.SourceVersion
	return tx.Save(&state).Error
}

func CaptureLegacyServe(uid uint64, actualFeed string, items []Candidate) {
	trace := CaptureServe(uid, actualFeed, items, "unassigned", "legacy")
	if trace != "" {
		rememberLegacy(uid, items, trace)
	}
}

func weightVariant(uid uint64, c feedconfig.Config) string {
	if c.Interleaving {
		return "interleaved"
	}
	if c.Comparison {
		if bucket(uid, c.Salt+":weights") >= 50 {
			return "B"
		}
		return "A"
	}
	return "default"
}
