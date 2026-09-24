// Package homefeedservice owns personalized Home feed selection and cursors.
package homefeedservice

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
)

var (
	ErrLoginRequired = errors.New("following feed requires login")
	ErrInvalidCursor = errors.New("invalid following feed cursor")
)

const PageSize = 20

type FollowingPage struct {
	Topics     []topics.Entity
	NextCursor string
}

type followingCursor struct {
	Version   int       `json:"v"`
	ViewerID  uint64    `json:"u"`
	CreatedAt time.Time `json:"t"`
	ID        uint64    `json:"id"`
}

// Following reads the current follow relationship on every page. Refreshing
// without a cursor discovers newly followed authors/newer posts; continuation
// only returns topics below its immutable created_at/ID boundary. A cursor is
// a position, never authority: all visibility/follow filters run again.
func Following(ctx context.Context, viewerID uint64, cursor string) (FollowingPage, error) {
	if viewerID == 0 {
		return FollowingPage{}, ErrLoginRequired
	}
	var before *topics.CreatedCursor
	if cursor != "" {
		if len(cursor) > 512 {
			return FollowingPage{}, ErrInvalidCursor
		}
		raw, err := base64.RawURLEncoding.DecodeString(cursor)
		if err != nil {
			return FollowingPage{}, ErrInvalidCursor
		}
		var value followingCursor
		if json.Unmarshal(raw, &value) != nil || value.Version != 1 || value.ViewerID != viewerID || value.ID == 0 || value.CreatedAt.IsZero() {
			return FollowingPage{}, ErrInvalidCursor
		}
		before = &topics.CreatedCursor{CreatedAt: value.CreatedAt, ID: value.ID}
	}
	rows, hasNext, err := topics.FollowingPage(ctx, viewerID, before, PageSize)
	if err != nil {
		return FollowingPage{}, err
	}
	result := FollowingPage{Topics: rows}
	if hasNext {
		last := rows[len(rows)-1]
		raw, err := json.Marshal(followingCursor{Version: 1, ViewerID: viewerID, CreatedAt: last.CreatedAt, ID: last.Id})
		if err != nil {
			return FollowingPage{}, err
		}
		result.NextCursor = base64.RawURLEncoding.EncodeToString(raw)
	}
	return result, nil
}
