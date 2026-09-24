package homefeedservice

import (
	"context"
	"errors"
	"fmt"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userFollow"
	"gorm.io/gorm"
	"testing"
	"time"
)

func followingDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := dbconnect.Connect()
	if err := db.AutoMigrate(&topics.Entity{}, &posts.Entity{}, &userFollow.Entity{}); err != nil {
		t.Fatal(err)
	}
	return db
}
func follow(t *testing.T, db *gorm.DB, viewer, author uint64) {
	t.Helper()
	row := userFollow.Entity{UserId: viewer, FollowUserId: author, Status: 1}
	if err := db.Create(&row).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Delete(&row) })
}
func seedTopic(t *testing.T, db *gorm.DB, id, author uint64, created time.Time) topics.Entity {
	t.Helper()
	row := topics.Entity{Id: id, UserId: author, Title: fmt.Sprintf("topic %d", id), Status: 1, FirstPostId: id, CreatedAt: created, UpdatedAt: created}
	post := posts.Entity{Id: id, TopicId: id, PostNo: 1, UserId: author, Content: "body"}
	if err := db.Create(&row).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&post).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Unscoped().Delete(&post); db.Unscoped().Delete(&row) })
	return row
}
func TestFollowingOnlyPublicFollowedTopics(t *testing.T) {
	db := followingDB(t)
	const viewer, author, stranger uint64 = 970100, 970101, 970102
	follow(t, db, viewer, author)
	follow(t, db, viewer+10, stranger)
	now := time.Date(2026, 9, 25, 0, 0, 0, 123000, time.UTC)
	visible := seedTopic(t, db, 970110, author, now)
	seedTopic(t, db, 970111, stranger, now.Add(time.Hour))
	cases := []struct {
		name          string
		table, column string
		value         any
	}{
		{"draft", "topics", "status", 0}, {"blocked", "topics", "process_status", 1}, {"pending", "topics", "process_status", 2},
		{"deleted", "topics", "visibility_status", topics.VisibilityUserDeleted}, {"removed", "topics", "visibility_status", topics.VisibilityModeratorRemoved},
		{"anonymized", "topics", "visibility_status", topics.VisibilityAccountAnonymized}, {"soft_deleted", "topics", "deleted_at", now},
		{"wiki", "topics", "topic_type", topics.TopicTypeWiki}, {"first_blocked", "posts", "process_status", 1},
		{"first_deleted", "posts", "deleted_at", now}, {"first_missing", "topics", "first_post_id", 0},
	}
	for i, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			row := seedTopic(t, db, uint64(970120+i), author, now.Add(time.Hour))
			if err := db.Table(test.table).Where("id = ?", row.Id).UpdateColumn(test.column, test.value).Error; err != nil {
				t.Fatal(err)
			}
			page, err := Following(t.Context(), viewer, "")
			if err != nil || len(page.Topics) != 1 || page.Topics[0].Id != visible.Id {
				t.Fatalf("only public followed topic wanted: %+v %v", page, err)
			}
		})
	}
	page, err := Following(t.Context(), viewer+20, "")
	if err != nil || len(page.Topics) != 0 || page.NextCursor != "" {
		t.Fatalf("no follows must be empty: %+v %v", page, err)
	}
}
func TestFollowingCursorStableAcrossInsertAndBoundaryDeletion(t *testing.T) {
	db := followingDB(t)
	const viewer, author uint64 = 971100, 971101
	follow(t, db, viewer, author)
	now := time.Date(2026, 9, 25, 0, 0, 0, 123000, time.UTC)
	for i := 1; i <= 25; i++ {
		seedTopic(t, db, uint64(971110+i), author, now)
	}
	first, err := Following(t.Context(), viewer, "")
	if err != nil || len(first.Topics) != 20 || first.NextCursor == "" {
		t.Fatalf("first: %+v %v", first, err)
	}
	if first.Topics[0].Id != 971135 || first.Topics[19].Id != 971116 {
		t.Fatalf("timestamp ties must use descending ID: %+v", first.Topics)
	}
	// Inserts above the boundary, edits/pinning and boundary deletion cannot shift the next page.
	newest := seedTopic(t, db, 971109, author, now.Add(time.Hour))
	db.Delete(&topics.Entity{}, first.Topics[19].Id)
	db.Model(&topics.Entity{}).Where("id = ?", 971111).Updates(map[string]any{"pin_weight": 999, "updated_at": now.Add(2 * time.Hour)})
	second, err := Following(t.Context(), viewer, first.NextCursor)
	if err != nil || len(second.Topics) != 5 || second.NextCursor != "" {
		t.Fatalf("second: %+v %v", second, err)
	}
	for i, row := range second.Topics {
		if row.Id != uint64(971115-i) {
			t.Fatalf("second[%d]=%d: duplicated/skipped/reordered", i, row.Id)
		}
	}
	fresh, err := Following(t.Context(), viewer, "")
	if err != nil || fresh.Topics[0].Id != newest.Id {
		t.Fatalf("newer created_at must sort first: %+v %v", fresh, err)
	}
	if _, err := Following(t.Context(), viewer+1, first.NextCursor); !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("cross-viewer cursor accepted: %v", err)
	}
}
func TestFollowingRelationshipChangesApplyWithoutCache(t *testing.T) {
	db := followingDB(t)
	const viewer, author, newAuthor uint64 = 972100, 972101, 972102
	follow(t, db, viewer, author)
	now := time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC)
	for i := 1; i <= 21; i++ {
		seedTopic(t, db, uint64(972110+i), author, now)
	}
	newest := seedTopic(t, db, 972140, newAuthor, now.Add(time.Hour))
	first, err := Following(t.Context(), viewer, "")
	if err != nil || first.NextCursor == "" {
		t.Fatalf("first: %+v %v", first, err)
	}
	db.Model(&userFollow.Entity{}).Where("user_id = ? AND follow_user_id = ?", viewer, author).UpdateColumn("status", 0)
	follow(t, db, viewer, newAuthor)
	next, err := Following(t.Context(), viewer, first.NextCursor)
	if err != nil || len(next.Topics) != 0 {
		t.Fatalf("unfollow must filter next page immediately: %+v %v", next, err)
	}
	fresh, err := Following(t.Context(), viewer, "")
	if err != nil || len(fresh.Topics) != 1 || fresh.Topics[0].Id != newest.Id {
		t.Fatalf("refresh must use current follows: %+v %v", fresh, err)
	}
}
func TestFollowingRejectsGuestMalformedCursorAndReadFailure(t *testing.T) {
	followingDB(t)
	if _, err := Following(t.Context(), 0, ""); !errors.Is(err, ErrLoginRequired) {
		t.Fatalf("guest: %v", err)
	}
	for _, cursor := range []string{"not-a-cursor!", "e30", "bnVsbA"} {
		if _, err := Following(t.Context(), 973100, cursor); !errors.Is(err, ErrInvalidCursor) {
			t.Fatalf("cursor %q: %v", cursor, err)
		}
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := Following(ctx, 973100, ""); !errors.Is(err, context.Canceled) {
		t.Fatalf("read failure must not masquerade as empty: %v", err)
	}
}
