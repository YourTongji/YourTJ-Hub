package pointservice

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pointsRecord"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userPoints"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"gorm.io/gorm"
)

func TestPointsActionCode(t *testing.T) {
	tests := []struct {
		value PointsAction
		code  string
	}{
		{value: PointsActionInit, code: "init"},
		{value: PointsActionTopicPublished, code: "topic_published"},
		{value: PointsActionPostCreated, code: "post_created"},
		{value: PointsActionPostDeleted, code: "post_deleted"},
		{value: PointsAction(99), code: "unknown"},
	}

	for _, tt := range tests {
		if got := tt.value.Code(); got != tt.code {
			t.Fatalf("Code() = %q, want %q", got, tt.code)
		}
	}
}

func TestApplyPointsIsIdempotentAndValidatesTopicOwner(t *testing.T) {
	conn := newPointsTestDB(t)
	createPointsUser(t, conn, 1, 100)
	createPointsUser(t, conn, 2, 100)
	createPointsTopic(t, conn, 10, 1, 1, topics.ProcessStatusNormal)

	for range 2 {
		if err := conn.Transaction(func(tx *gorm.DB) error {
			return applyPointsTx(tx, 1, TopicPublishedReward, PointsActionTopicPublished, "topic:10", "")
		}); err != nil {
			t.Fatalf("reward published topic: %v", err)
		}
	}
	assertPointsState(t, conn, 1, 110, 10)
	assertSourceRecord(t, conn, "topic:10", 1, TopicPublishedReward)

	createPointsTopic(t, conn, 11, 1, 1, topics.ProcessStatusPending)
	for _, tc := range []struct {
		userID    uint64
		sourceKey string
	}{
		{userID: 2, sourceKey: "topic:11"},
		{userID: 1, sourceKey: "topic:999"},
	} {
		if err := conn.Transaction(func(tx *gorm.DB) error {
			return applyPointsTx(tx, tc.userID, TopicPublishedReward, PointsActionTopicPublished, tc.sourceKey, "")
		}); err != nil {
			t.Fatalf("reject invalid topic reward: %v", err)
		}
	}
	assertPointsState(t, conn, 1, 110, 10)
	assertPointsState(t, conn, 2, 100, 0)
	assertNoSourceRecord(t, conn, "topic:11")
	assertNoSourceRecord(t, conn, "topic:999")
}

func TestApplyPostPointsRejectsDeletedAndWrongOwner(t *testing.T) {
	conn := newPointsTestDB(t)
	createPointsUser(t, conn, 1, 100)
	createPointsUser(t, conn, 2, 100)
	createPointsTopic(t, conn, 20, 1, 1, topics.ProcessStatusNormal)
	createPointsPost(t, conn, 21, 20, 1, posts.ProcessStatusNormal)
	createPointsPost(t, conn, 22, 20, 1, posts.ProcessStatusPending)
	deletedKey := "post-deleted:22"
	if err := conn.Create(&pointsRecord.Entity{UserId: 1, Action: PointsActionPostDeleted.Code(), SourceKey: &deletedKey}).Error; err != nil {
		t.Fatalf("create deletion marker: %v", err)
	}

	if err := conn.Transaction(func(tx *gorm.DB) error {
		return applyPointsTx(tx, 1, PostCreatedReward, PointsActionPostCreated, "post:21", "")
	}); err != nil {
		t.Fatalf("reward normal post: %v", err)
	}
	for _, tc := range []struct {
		userID    uint64
		sourceKey string
	}{
		{userID: 1, sourceKey: "post:22"},
		{userID: 2, sourceKey: "post:21"},
	} {
		if err := conn.Transaction(func(tx *gorm.DB) error {
			return applyPointsTx(tx, tc.userID, PostCreatedReward, PointsActionPostCreated, tc.sourceKey, "")
		}); err != nil {
			t.Fatalf("reject invalid post reward: %v", err)
		}
	}
	assertPointsState(t, conn, 1, 102, 2)
	assertPointsState(t, conn, 2, 100, 0)
	assertNoSourceRecord(t, conn, "post:22")
}

func newPointsTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:pointservice-%d?mode=memory&cache=shared", time.Now().UnixNano())
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
		t.Fatalf("migrate points fixtures: %v", err)
	}
	return conn
}

func createPointsUser(t *testing.T, conn *gorm.DB, userID uint64, currentPoints int64) {
	t.Helper()
	if err := conn.Create(&users.EntityComplete{Id: userID, Username: fmt.Sprintf("user-%d", userID)}).Error; err != nil {
		t.Fatalf("create user %d: %v", userID, err)
	}
	if err := conn.Create(&userPoints.Entity{UserId: userID, CurrentPoints: currentPoints}).Error; err != nil {
		t.Fatalf("create user points %d: %v", userID, err)
	}
}

func createPointsTopic(t *testing.T, conn *gorm.DB, topicID, userID uint64, status, processStatus int8) {
	t.Helper()
	if err := conn.Create(&topics.Entity{Id: topicID, UserId: userID, Status: status, ProcessStatus: processStatus}).Error; err != nil {
		t.Fatalf("create topic %d: %v", topicID, err)
	}
}

func createPointsPost(t *testing.T, conn *gorm.DB, postID, topicID, userID uint64, processStatus int8) {
	t.Helper()
	if err := conn.Create(&posts.Entity{Id: postID, TopicId: topicID, PostNo: postID, UserId: userID, ProcessStatus: processStatus}).Error; err != nil {
		t.Fatalf("create post %d: %v", postID, err)
	}
}

func assertPointsState(t *testing.T, conn *gorm.DB, userID uint64, currentPoints, prestige int64) {
	t.Helper()
	var balance userPoints.Entity
	if err := conn.Where("user_id = ?", userID).Take(&balance).Error; err != nil {
		t.Fatalf("load points for user %d: %v", userID, err)
	}
	if balance.CurrentPoints != currentPoints {
		t.Errorf("user %d points = %d, want %d", userID, balance.CurrentPoints, currentPoints)
	}
	var user users.EntityComplete
	if err := conn.Where("id = ?", userID).Take(&user).Error; err != nil {
		t.Fatalf("load user %d: %v", userID, err)
	}
	if user.Prestige != prestige {
		t.Errorf("user %d prestige = %d, want %d", userID, user.Prestige, prestige)
	}
}

func assertSourceRecord(t *testing.T, conn *gorm.DB, sourceKey string, userID uint64, points int64) {
	t.Helper()
	var record pointsRecord.Entity
	if err := conn.Where("source_key = ?", sourceKey).Take(&record).Error; err != nil {
		t.Fatalf("load source record %q: %v", sourceKey, err)
	}
	if record.UserId != userID || record.PointsChange != points {
		t.Errorf("source record %q = (user %d, points %d), want (user %d, points %d)", sourceKey, record.UserId, record.PointsChange, userID, points)
	}
}

func assertNoSourceRecord(t *testing.T, conn *gorm.DB, sourceKey string) {
	t.Helper()
	var count int64
	if err := conn.Model(&pointsRecord.Entity{}).Where("source_key = ?", sourceKey).Count(&count).Error; err != nil {
		t.Fatalf("count source record %q: %v", sourceKey, err)
	}
	if count != 0 {
		t.Errorf("source record %q count = %d, want 0", sourceKey, count)
	}
}

func TestApplyPointsSkipsBotUsers(t *testing.T) {
	conn := newPointsTestDB(t)
	botID := uint64(42)
	if err := conn.Create(&users.EntityComplete{Id: botID, Username: "bot", ActorType: users.ActorTypeBot}).Error; err != nil {
		t.Fatalf("create bot user: %v", err)
	}
	createPointsTopic(t, conn, 50, botID, 1, topics.ProcessStatusNormal)

	for _, tc := range []struct {
		name      string
		action    PointsAction
		points    int64
		sourceKey string
	}{
		{"topic reward", PointsActionTopicPublished, TopicPublishedReward, "topic:50"},
		{"post reward", PointsActionPostCreated, PostCreatedReward, "post:50"},
		{"post reverse", PointsActionPostDeleted, -PostCreatedReward, "post-deleted:50"},
	} {
		if err := conn.Transaction(func(tx *gorm.DB) error {
			return applyPointsTx(tx, botID, tc.points, tc.action, tc.sourceKey, "")
		}); err != nil {
			t.Fatalf("%s: applyPointsTx for bot returned error: %v", tc.name, err)
		}
	}

	assertNoSourceRecord(t, conn, "topic:50")
	assertNoSourceRecord(t, conn, "post:50")
	assertNoSourceRecord(t, conn, "post-deleted:50")
	var botBalance userPoints.Entity
	err := conn.Where("user_id = ?", botID).Take(&botBalance).Error
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("bot should have no user_points row, got %v / %+v", err, botBalance)
	}
	var botUser users.EntityComplete
	if err := conn.Where("id = ?", botID).Take(&botUser).Error; err != nil {
		t.Fatalf("load bot user: %v", err)
	}
	if botUser.Prestige != 0 {
		t.Errorf("bot prestige = %d, want 0 (no reward, no reversal)", botUser.Prestige)
	}
}

func TestApplyPointsLazyCreatesMissingUserPointsRow(t *testing.T) {
	conn := newPointsTestDB(t)
	// Human user without a user_points row — simulates a legacy deployment whose
	// users table predates the points feature and has not yet run backfill v14.
	if err := conn.Create(&users.EntityComplete{Id: 7, Username: "legacy-human"}).Error; err != nil {
		t.Fatalf("create legacy human: %v", err)
	}
	createPointsTopic(t, conn, 70, 7, 7, topics.ProcessStatusNormal)

	if err := conn.Transaction(func(tx *gorm.DB) error {
		return applyPointsTx(tx, 7, TopicPublishedReward, PointsActionTopicPublished, "topic:70", "")
	}); err != nil {
		t.Fatalf("reward legacy user without user_points row: %v", err)
	}

	assertPointsState(t, conn, 7, TopicPublishedReward, TopicPublishedReward)
	var balance userPoints.Entity
	if err := conn.Where("user_id = ?", 7).Take(&balance).Error; err != nil {
		t.Fatalf("lazy-created user_points row not found: %v", err)
	}
	if balance.CurrentPoints != TopicPublishedReward {
		t.Errorf("legacy user_points current_points = %d, want %d", balance.CurrentPoints, TopicPublishedReward)
	}
}

func TestReversePostRewardIsIdempotent(t *testing.T) {
	conn := newPointsTestDB(t)
	createPointsUser(t, conn, 1, 100)
	createPointsTopic(t, conn, 80, 1, 1, topics.ProcessStatusNormal)
	createPointsPost(t, conn, 81, 80, 1, posts.ProcessStatusNormal)

	if err := conn.Transaction(func(tx *gorm.DB) error {
		return applyPointsTx(tx, 1, PostCreatedReward, PointsActionPostCreated, "post:81", "")
	}); err != nil {
		t.Fatalf("reward post: %v", err)
	}
	assertPointsState(t, conn, 1, 100+PostCreatedReward, PostCreatedReward)

	if err := conn.Transaction(func(tx *gorm.DB) error {
		return applyPointsTx(tx, 1, -PostCreatedReward, PointsActionPostDeleted, "post-deleted:81", "post:81")
	}); err != nil {
		t.Fatalf("reverse post reward: %v", err)
	}
	assertPointsState(t, conn, 1, 100, 0)

	// Second reverse attempt for the same post is a no-op: the unique source_key
	// tombstone forces ErrDuplicatedKey → rollback to savepoint → nil, leaving the
	// balance and prestige from the first reversal untouched.
	if err := conn.Transaction(func(tx *gorm.DB) error {
		return applyPointsTx(tx, 1, -PostCreatedReward, PointsActionPostDeleted, "post-deleted:81", "post:81")
	}); err != nil {
		t.Fatalf("duplicate reverse post reward: %v", err)
	}
	assertPointsState(t, conn, 1, 100, 0)

	var tombstoneCount int64
	if err := conn.Model(&pointsRecord.Entity{}).Where("source_key = ?", "post-deleted:81").Count(&tombstoneCount).Error; err != nil {
		t.Fatalf("count tombstone records: %v", err)
	}
	if tombstoneCount != 1 {
		t.Errorf("post-deleted:81 record count = %d, want 1 (reverse applied exactly once)", tombstoneCount)
	}
}
