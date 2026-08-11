package postservice

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/leancodebox/GooseForum/app/models/forum/pointsRecord"
	"github.com/leancodebox/GooseForum/app/models/forum/posts"
	"github.com/leancodebox/GooseForum/app/models/forum/topics"
	"github.com/leancodebox/GooseForum/app/models/forum/userPoints"
	"github.com/leancodebox/GooseForum/app/models/forum/users"
	"gorm.io/gorm"
)

func TestDeleteTopicPostReversesRewardAtomically(t *testing.T) {
	conn := newDeletePostTestDB(t)
	seedDeletePost(t, conn, true, true)

	deleted, err := deleteTopicPost(conn, 20, 1)
	if err != nil {
		t.Fatalf("delete topic post: %v", err)
	}
	if deleted.Id != 20 {
		t.Fatalf("deleted post id = %d, want 20", deleted.Id)
	}
	assertPostDeleted(t, conn, 20, true)
	assertDeletePoints(t, conn, 100, 0, -2)

	if _, err := deleteTopicPost(conn, 20, 1); !errors.Is(err, ErrPostNotFound) {
		t.Fatalf("second delete error = %v, want ErrPostNotFound", err)
	}
	assertDeletePoints(t, conn, 100, 0, -2)
}

func TestDeleteTopicPostWithoutOriginalRewardDoesNotDeduct(t *testing.T) {
	conn := newDeletePostTestDB(t)
	seedDeletePost(t, conn, false, true)

	if _, err := deleteTopicPost(conn, 20, 1); err != nil {
		t.Fatalf("delete unrewarded post: %v", err)
	}
	assertPostDeleted(t, conn, 20, true)
	assertDeletePoints(t, conn, 100, 0, 0)
}

func TestDeleteTopicPostRollsBackWhenReversalFails(t *testing.T) {
	conn := newDeletePostTestDB(t)
	seedDeletePost(t, conn, true, false)

	if _, err := deleteTopicPost(conn, 20, 1); err == nil {
		t.Fatal("delete topic post succeeded without a user_points row")
	}
	assertPostDeleted(t, conn, 20, false)
	var reversalCount int64
	if err := conn.Model(&pointsRecord.Entity{}).Where("source_key = ?", "post-deleted:20").Count(&reversalCount).Error; err != nil {
		t.Fatalf("count rollback reversal: %v", err)
	}
	if reversalCount != 0 {
		t.Fatalf("reversal count after rollback = %d, want 0", reversalCount)
	}
}

func newDeletePostTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:postservice-delete-%d?mode=memory&cache=shared", time.Now().UnixNano())
	conn, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := conn.AutoMigrate(
		&users.EntityComplete{},
		&userPoints.Entity{},
		&pointsRecord.Entity{},
		&topics.Entity{},
		&posts.Entity{},
	); err != nil {
		t.Fatalf("migrate delete post fixtures: %v", err)
	}
	return conn
}

func seedDeletePost(t *testing.T, conn *gorm.DB, rewarded, withBalance bool) {
	t.Helper()
	prestige := int64(0)
	currentPoints := int64(100)
	if rewarded {
		prestige = 2
		currentPoints = 102
	}
	if err := conn.Create(&users.EntityComplete{Id: 1, Username: "delete-user", Prestige: prestige}).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	if withBalance {
		if err := conn.Create(&userPoints.Entity{UserId: 1, CurrentPoints: currentPoints}).Error; err != nil {
			t.Fatalf("create user points: %v", err)
		}
	}
	if err := conn.Create(&topics.Entity{Id: 10, UserId: 1, Status: 1, ProcessStatus: topics.ProcessStatusNormal}).Error; err != nil {
		t.Fatalf("create topic: %v", err)
	}
	if err := conn.Create(&posts.Entity{Id: 20, TopicId: 10, PostNo: 2, UserId: 1, ProcessStatus: posts.ProcessStatusNormal}).Error; err != nil {
		t.Fatalf("create post: %v", err)
	}
	if rewarded {
		key := "post:20"
		if err := conn.Create(&pointsRecord.Entity{UserId: 1, Action: "post_created", PointsChange: 2, SourceKey: &key}).Error; err != nil {
			t.Fatalf("create original reward: %v", err)
		}
	}
}

func assertPostDeleted(t *testing.T, conn *gorm.DB, postID uint64, wantDeleted bool) {
	t.Helper()
	var post posts.Entity
	if err := conn.Unscoped().Where("id = ?", postID).Take(&post).Error; err != nil {
		t.Fatalf("load post: %v", err)
	}
	if post.DeletedAt.Valid != wantDeleted {
		t.Errorf("post deleted = %t, want %t", post.DeletedAt.Valid, wantDeleted)
	}
}

func assertDeletePoints(t *testing.T, conn *gorm.DB, wantPoints, wantPrestige, wantReversal int64) {
	t.Helper()
	var balance userPoints.Entity
	if err := conn.Where("user_id = ?", 1).Take(&balance).Error; err != nil {
		t.Fatalf("load user points: %v", err)
	}
	if balance.CurrentPoints != wantPoints {
		t.Errorf("current points = %d, want %d", balance.CurrentPoints, wantPoints)
	}
	var user users.EntityComplete
	if err := conn.Where("id = ?", 1).Take(&user).Error; err != nil {
		t.Fatalf("load user: %v", err)
	}
	if user.Prestige != wantPrestige {
		t.Errorf("prestige = %d, want %d", user.Prestige, wantPrestige)
	}
	var reversal pointsRecord.Entity
	if err := conn.Where("source_key = ?", "post-deleted:20").Take(&reversal).Error; err != nil {
		t.Fatalf("load reversal: %v", err)
	}
	if reversal.PointsChange != wantReversal {
		t.Errorf("reversal points = %d, want %d", reversal.PointsChange, wantReversal)
	}
}
