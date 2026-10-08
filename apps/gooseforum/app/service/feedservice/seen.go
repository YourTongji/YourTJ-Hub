package feedservice

import (
	"context"
	"errors"
	"time"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/feed"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"gorm.io/gorm"
)

var seenKey = []byte(preferences.GetString("app.signingKey") + ":feed-seen-v1:" + processEpoch)

type seenClaim struct {
	ID        string    `json:"i"`
	User      uint64    `json:"u"`
	Topics    []uint64  `json:"t"`
	ContentAt time.Time `json:"c"`
	IssuedAt  int64     `json:"a"`
	ExpiresAt int64     `json:"x"`
}

// SeenProof is independent of sampled analytics and is shared by up to 20 cards.
type SeenProof struct {
	Token     string   `json:"token"`
	TopicIDs  []uint64 `json:"topicIds"`
	IssuedAt  int64    `json:"issuedAt"`
	ExpiresAt int64    `json:"expiresAt"`
}
type SeenPatch struct {
	Proof string        `json:"proof"`
	Seen  map[int]int64 `json:"seen"` // elapsed milliseconds after proof receipt, frozen at qualification
}

func MintSeenProof(uid uint64, ids []uint64, contentAt time.Time) []SeenProof {
	if uid == 0 || len(ids) == 0 || len(ids) > 120 || contentAt.IsZero() {
		return nil
	}
	now := time.Now()
	result := []SeenProof{}
	for start := 0; start < len(ids); start += 20 {
		group := append([]uint64(nil), ids[start:min(start+20, len(ids))]...)
		claim := seenClaim{ID: feed.NewID(), User: uid, Topics: group, ContentAt: contentAt.UTC().Truncate(time.Millisecond), IssuedAt: now.UnixMilli(), ExpiresAt: now.Add(30 * time.Minute).UnixMilli()}
		result = append(result, SeenProof{Token: sign(claim, seenKey), TopicIDs: group, IssuedAt: claim.IssuedAt, ExpiresAt: claim.ExpiresAt})
	}
	return result
}

// CaptureSeen commits before ACK. Client claims affect only this owner's feed;
// they never create opened events or contribute to public rank/points.
func CaptureSeen(ctx context.Context, uid uint64, patches []SeenPatch) error {
	if len(patches) == 0 {
		return nil
	}
	if len(patches) > 50 || uid == 0 {
		return ErrInvalidTrace
	}
	now := time.Now()
	byTopic := map[uint64]feed.SeenState{}
	for _, patch := range patches {
		var claim seenClaim
		if verify(patch.Proof, seenKey, &claim, 2048) != nil || claim.User != uid || len(claim.Topics) == 0 || len(claim.Topics) > 20 || claim.ExpiresAt <= now.UnixMilli() || claim.IssuedAt > now.UnixMilli() || claim.ExpiresAt-claim.IssuedAt != int64(30*time.Minute/time.Millisecond) || claim.ContentAt.IsZero() || claim.ContentAt.After(time.UnixMilli(claim.IssuedAt)) || len(patch.Seen) > 20 {
			return ErrInvalidTrace
		}
		for position, elapsed := range patch.Seen {
			if position < 0 || position >= len(claim.Topics) || elapsed < 1000 || elapsed >= claim.ExpiresAt-claim.IssuedAt || claim.IssuedAt+elapsed > now.UnixMilli() {
				return ErrInvalidTrace
			}
			at := time.UnixMilli(claim.IssuedAt + elapsed).UTC()
			row := feed.SeenState{UserID: uid, TopicID: claim.Topics[position], LastSeenAt: at, SeenContentAt: claim.ContentAt, LastSeenProofID: claim.ID, ExpiresAt: at.Add(30 * 24 * time.Hour)}
			if old, ok := byTopic[row.TopicID]; !ok || old.SeenContentAt.Before(row.SeenContentAt) {
				byTopic[row.TopicID] = row
			}
		}
	}
	if len(byTopic) > 120 {
		return ErrInvalidTrace
	}
	rows := make([]feed.SeenState, 0, len(byTopic))
	for _, row := range byTopic {
		rows = append(rows, row)
	}
	return db.ConnectContext(ctx).Transaction(func(tx *gorm.DB) error {
		eligible, err := users.FeedActorEligibleTx(tx, uid)
		if err != nil {
			return err
		}
		if !eligible {
			return feed.ErrClosed
		}
		if err := feed.LockOwnerTx(tx, uid); err != nil {
			return err
		}
		return feed.UpsertSeenTx(tx, rows)
	})
}

func seenEligibility(ctx context.Context, uid uint64, ids []uint64) (map[uint64]bool, map[uint64]bool, error) {
	seen, err := feed.SeenAmong(db.ConnectContext(ctx), uid, ids, time.Now())
	if err != nil {
		return nil, nil, err
	}
	replies, err := posts.NewPublicRepliesAmong(ctx, uid, seen)
	if err != nil {
		return nil, nil, err
	}
	excluded := map[uint64]bool{}
	for id := range seen {
		excluded[id] = !replies[id]
	}
	return excluded, replies, nil
}

// Reconcile hydrates only requested survivors, in caller order. Seeing a card
// cannot remove it from an already loaded browsing session.
func Reconcile(ctx context.Context, uid uint64, ids []uint64) ([]topics.Entity, []uint64, time.Time, error) {
	contentAt := time.Now()
	if len(ids) > 120 {
		return nil, nil, contentAt, ErrInvalidCursor
	}
	rows, err := topics.PublicTopics(ctx, ids)
	if err != nil {
		return nil, nil, contentAt, err
	}
	authors := []uint64{}
	for _, row := range rows {
		authors = append(authors, row.UserId)
	}
	allowed, err := users.FilterEligibleFeedAuthors(ctx, uid, authors)
	if err != nil {
		return nil, nil, contentAt, err
	}
	byID := map[uint64]topics.Entity{}
	for _, row := range rows {
		if allowed[row.UserId] {
			byID[row.Id] = row
		}
	}
	ordered := []topics.Entity{}
	removed := []uint64{}
	for _, id := range ids {
		if row, ok := byID[id]; ok {
			ordered = append(ordered, row)
		} else {
			removed = append(removed, id)
		}
	}
	return ordered, removed, contentAt, nil
}

// CaptureEvents charges the shared transport once. A full metrics queue cannot
// undo functional state that has already committed.
func CaptureEvents(ctx context.Context, uid uint64, patches []Patch, seen []SeenPatch) (bool, error) {
	if len(patches)+len(seen) > 50 {
		return false, ErrInvalidTrace
	}
	if !allowEvents(uid, time.Now()) {
		return false, ErrRateLimited
	}
	if err := CaptureSeen(ctx, uid, seen); err != nil {
		return false, err
	}
	err := capturePatches(uid, patches, false)
	if len(seen) > 0 && (errors.Is(err, ErrQueueFull) || errors.Is(err, ErrInvalidTrace)) {
		return true, nil
	}
	return len(seen) > 0, err
}
