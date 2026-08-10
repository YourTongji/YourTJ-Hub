package api

import (
	"fmt"
	"testing"
	"time"

	"github.com/leancodebox/GooseForum/app/bundles/connect/dbconnect"
	"github.com/leancodebox/GooseForum/app/http/controllers/component"
	"github.com/leancodebox/GooseForum/app/models/forum/contentDeleteEvent"
	"github.com/leancodebox/GooseForum/app/models/forum/moderationLog"
	"github.com/leancodebox/GooseForum/app/models/forum/optRecord"
	"github.com/leancodebox/GooseForum/app/models/forum/posts"
	"github.com/leancodebox/GooseForum/app/models/forum/topics"
	"github.com/leancodebox/GooseForum/app/models/forum/users"
	"gorm.io/gorm"
)

func setupBatchDeleteTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(
		&users.EntityComplete{},
		&topics.Entity{},
		&posts.Entity{},
		&optRecord.Entity{},
		&moderationLog.Entity{},
		&contentDeleteEvent.Entity{},
	); err != nil {
		t.Fatalf("migrate batch delete tables: %v", err)
	}
	return conn
}

func seedBatchTopics(t *testing.T, conn *gorm.DB, userID uint64, startID uint64, count int) []uint64 {
	t.Helper()
	ids := make([]uint64, 0, count)
	now := time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC)
	for i := 0; i < count; i++ {
		topicID := startID + uint64(i)
		postID := startID + uint64(i) + 10_000
		if err := conn.Create(&posts.Entity{Id: postID, TopicId: topicID, PostNo: 1, UserId: userID, Content: "body", CreatedAt: now, UpdatedAt: now}).Error; err != nil {
			t.Fatalf("create first post: %v", err)
		}
		if err := conn.Create(&topics.Entity{
			Id:               topicID,
			Title:            fmt.Sprintf("Batch topic %d", topicID),
			UserId:           userID,
			Status:           1,
			ProcessStatus:    topics.ProcessStatusNormal,
			PostCount:        1,
			PostSeq:          1,
			FirstPostId:      postID,
			VisibilityStatus: topics.VisibilityActive,
			RetentionStatus:  topics.RetentionNormal,
			CreatedAt:        now,
			UpdatedAt:        now,
		}).Error; err != nil {
			t.Fatalf("create topic: %v", err)
		}
		ids = append(ids, topicID)
	}
	t.Cleanup(func() {
		conn.Unscoped().Where("user_id = ?", userID).Delete(&topics.Entity{})
		conn.Unscoped().Where("user_id = ?", userID).Delete(&posts.Entity{})
		conn.Unscoped().Where("actor_id = ?", userID).Delete(&contentDeleteEvent.Entity{})
	})
	return ids
}

// R9：批量删除逐条走删除流程并汇总结果。
func TestBatchDeleteContentDeletesAll(t *testing.T) {
	conn := setupBatchDeleteTestDB(t)
	userID := uint64(9_900_000_001)
	if err := conn.Create(&users.EntityComplete{Id: userID, Username: "batch_owner", IsActivated: 1}).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	ids := seedBatchTopics(t, conn, userID, 9_900_000_100, 3)

	res := BatchDeleteContent(component.BetterRequest[BatchDeleteContentReq]{
		UserId: userID,
		Params: BatchDeleteContentReq{ContentType: "topic", ContentIDs: ids},
	})
	if res.Data.Code != component.SUCCESS {
		t.Fatalf("BatchDeleteContent failed: %#v", res)
	}
	data, ok := res.Data.Result.(map[string]any)
	if !ok {
		t.Fatalf("result type = %T", res.Data.Result)
	}
	if data["failed"] != 0 || data["succeeded"] != len(ids) {
		t.Fatalf("batch result = %#v", data)
	}
	for _, id := range ids {
		if visible := topics.Get(id); visible.Id != 0 {
			t.Fatalf("topic %d still visible after batch delete", id)
		}
	}
}

// R9：删除频率超限时要求二次确认，force=true 后放行。
func TestBatchDeleteContentRequiresConfirmOverThreshold(t *testing.T) {
	conn := setupBatchDeleteTestDB(t)
	userID := uint64(9_900_000_002)
	if err := conn.Create(&users.EntityComplete{Id: userID, Username: "batch_confirm", IsActivated: 1}).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	// 模拟历史已删除 20 条：写入 20 条 content_deleted 事件。
	for i := 0; i < 20; i++ {
		if err := contentDeleteEvent.Record(contentDeleteEvent.Entity{
			EventType:   string(contentDeleteEvent.EventDeleted),
			ContentType: "topic",
			ContentID:   uint64(9_900_001_000 + i),
			ActorID:     userID,
		}); err != nil {
			t.Fatalf("seed delete event: %v", err)
		}
	}
	ids := seedBatchTopics(t, conn, userID, 9_900_000_300, 2)

	// 未 force：应要求二次确认。
	noForce := BatchDeleteContent(component.BetterRequest[BatchDeleteContentReq]{
		UserId: userID,
		Params: BatchDeleteContentReq{ContentType: "topic", ContentIDs: ids},
	})
	if noForce.Data.Code == component.SUCCESS {
		t.Fatalf("expected confirm-required failure, got %#v", noForce)
	}
	if noForce.Data.MessageCode != component.MessageContentBatchConfirmRequired {
		t.Fatalf("messageCode = %s, want %s", noForce.Data.MessageCode, component.MessageContentBatchConfirmRequired)
	}
	for _, id := range ids {
		if visible := topics.Get(id); visible.Id == 0 {
			t.Fatalf("topic %d should not be deleted without force confirm", id)
		}
	}

	// force=true：放行。
	forced := BatchDeleteContent(component.BetterRequest[BatchDeleteContentReq]{
		UserId: userID,
		Params: BatchDeleteContentReq{ContentType: "topic", ContentIDs: ids, Force: true},
	})
	if forced.Data.Code != component.SUCCESS {
		t.Fatalf("forced batch delete failed: %#v", forced)
	}
	for _, id := range ids {
		if visible := topics.Get(id); visible.Id != 0 {
			t.Fatalf("topic %d still visible after forced batch delete", id)
		}
	}
}
