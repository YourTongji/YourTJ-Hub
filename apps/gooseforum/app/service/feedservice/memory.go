package feedservice

import (
	"context"
	"sync"
	"time"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/feedconfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/feed"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type memoryKey struct {
	User, Topic uint64
	Name        string
}
type memoryEntry struct {
	At, Expires time.Time
	Trace       string
	Position    int
}

// Each entry is conservatively charged 256 bytes plus string storage, including
// map overhead. These caches are lossy hints, never authoritative access state.
type boundedMemory struct {
	sync.Mutex
	entries      map[memoryKey]memoryEntry
	bytes, limit int
}

func (m *boundedMemory) get(k memoryKey, now time.Time) (memoryEntry, bool) {
	m.Lock()
	defer m.Unlock()
	e, ok := m.entries[k]
	return e, ok && e.Expires.After(now)
}
func (m *boundedMemory) put(k memoryKey, e memoryEntry) bool {
	m.Lock()
	defer m.Unlock()
	if m.entries == nil {
		m.entries = map[memoryKey]memoryEntry{}
	}
	size := 256 + len(k.Name) + len(e.Trace)
	if m.bytes+size > m.limit {
		n := 0
		for key, old := range m.entries {
			if !old.Expires.After(time.Now()) {
				m.bytes -= 256 + len(key.Name) + len(old.Trace)
				delete(m.entries, key)
			}
			n++
			if n >= 64 {
				break
			}
		}
	}
	if old, ok := m.entries[k]; ok {
		m.bytes -= 256 + len(k.Name) + len(old.Trace)
		delete(m.entries, k)
	}
	if m.bytes+size > m.limit {
		return false
	}
	m.entries[k] = e
	m.bytes += size
	return true
}
func (m *boundedMemory) once(k memoryKey, now time.Time, ttl time.Duration) bool {
	if _, ok := m.get(k, now); ok {
		return false
	}
	return m.put(k, memoryEntry{At: now, Expires: now.Add(ttl)})
}
func (m *boundedMemory) clear(uid uint64) {
	m.Lock()
	defer m.Unlock()
	for k, e := range m.entries {
		if uid == 0 || k.User == uid {
			m.bytes -= 256 + len(k.Name) + len(e.Trace)
			delete(m.entries, k)
		}
	}
}

var dailySeen = boundedMemory{limit: 2 << 20}
var repeats = boundedMemory{limit: 4 << 20}
var inferred = boundedMemory{limit: 2 << 20}

func rememberVisible(s feed.Serve, mask uint32) {
	items, err := decodeServed(s.Items)
	if err != nil {
		return
	}
	now := time.Now()
	for pos, c := range items {
		if mask&(1<<pos) != 0 {
			repeats.put(memoryKey{User: s.UserID, Topic: c.ID}, memoryEntry{At: now, Expires: now.Add(24 * time.Hour)})
		}
	}
}
func wasRepeated(uid, topic uint64, lastReply *time.Time, now time.Time) bool {
	e, ok := repeats.get(memoryKey{User: uid, Topic: topic}, now)
	return ok && (lastReply == nil || !lastReply.After(e.At))
}
func rememberLegacy(uid uint64, items []Candidate, trace string) {
	now := time.Now()
	for pos, c := range items {
		inferred.put(memoryKey{User: uid, Topic: c.ID}, memoryEntry{At: now, Expires: now.Add(6 * time.Hour), Trace: trace, Position: pos})
	}
}
func clearViewerMemory(uid uint64) {
	dailySeen.clear(uid)
	repeats.clear(uid)
	inferred.clear(uid)
	assignments.clear(uid)
	viewWritten.clear(uid)
	viewsMu.Lock()
	for k := range pendingViews {
		if uid == 0 || k.User == uid {
			delete(pendingViews, k)
		}
	}
	viewsMu.Unlock()
}
func InvalidateAccount(uid uint64) { invalidateViewer(uid); clearViewerMemory(uid) }

var viewsMu sync.Mutex
var pendingViews = map[memoryKey]feed.ViewFact{}
var viewWritten = boundedMemory{limit: 1 << 20}

func captureViewFact(row feed.ViewFact) {
	k := memoryKey{User: row.UserID, Topic: row.TopicID}
	if _, ok := viewWritten.get(k, row.ViewedAt); ok {
		return
	}
	viewsMu.Lock()
	defer viewsMu.Unlock()
	if len(pendingViews) >= 4096 {
		dropped.Add(1)
		return
	}
	pendingViews[k] = row
}
func flushViews(ctx context.Context, c feedconfig.Config) error {
	viewsMu.Lock()
	if !c.Metrics {
		pendingViews = map[memoryKey]feed.ViewFact{}
		viewsMu.Unlock()
		viewWritten.clear(0)
		return nil
	}
	rows := make([]feed.ViewFact, 0, 50)
	for k, row := range pendingViews {
		rows = append(rows, row)
		delete(pendingViews, k)
		if len(rows) == 50 {
			break
		}
	}
	viewsMu.Unlock()
	for _, row := range rows {
		err := db.ConnectContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := feed.LockOwnerTx(tx, row.UserID); err != nil {
				return err
			}
			r := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "topic_id"}, {Name: "user_id"}}, DoUpdates: clause.AssignmentColumns([]string{"viewed_at", "expires_at"}), Where: clause.Where{Exprs: []clause.Expression{clause.Expr{SQL: "topic_view_fact.viewed_at <= ?", Vars: []any{row.ViewedAt.Add(-5 * time.Minute)}}}}}).Create(&row)
			if r.Error != nil {
				return r.Error
			}
			if r.RowsAffected > 0 {
				return feed.MarkTx(tx, row.TopicID)
			}
			return nil
		})
		if err != nil {
			dropped.Add(1)
		} else {
			viewWritten.put(memoryKey{User: row.UserID, Topic: row.TopicID}, memoryEntry{At: row.ViewedAt, Expires: row.ViewedAt.Add(5 * time.Minute)})
		}
	}
	return nil
}
