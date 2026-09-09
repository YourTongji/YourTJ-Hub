package topicUserStat

import (
	"reflect"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
)

func TestTopicUserStatRepositoryParity(t *testing.T) {
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(&Entity{}); err != nil {
		t.Fatalf("migrate topic user stat: %v", err)
	}
	conn.Where("1 = 1").Delete(&Entity{})

	if err := IncrementUserPost(10, 1); err != nil {
		t.Fatalf("IncrementUserPost first err=%v", err)
	}
	if err := IncrementUserPost(10, 1); err != nil {
		t.Fatalf("IncrementUserPost second err=%v", err)
	}
	if err := IncrementUserPost(10, 2); err != nil {
		t.Fatalf("IncrementUserPost user 2 err=%v", err)
	}

	var row Entity
	conn.Where("topic_id = ? AND user_id = ?", 10, 1).First(&row)
	if row.ReplyCount != 2 {
		t.Fatalf("ReplyCount=%d, want 2", row.ReplyCount)
	}
	if got := SyncTopicPosters(10, 0); !reflect.DeepEqual(got, []uint64{1, 2}) {
		t.Fatalf("SyncTopicPosters()=%#v, want [1 2]", got)
	}
	if got := SyncTopicPosters(10, 1); !reflect.DeepEqual(got, []uint64{2}) {
		t.Fatalf("SyncTopicPosters(exclude 1)=%#v, want [2]", got)
	}
	if err := DecrementUserPost(10, 1); err != nil {
		t.Fatalf("DecrementUserPost err=%v", err)
	}
	conn.Where("topic_id = ? AND user_id = ?", 10, 1).First(&row)
	if row.ReplyCount != 1 {
		t.Fatalf("ReplyCount after decrement=%d, want 1", row.ReplyCount)
	}
}

func TestBulkUpsertRepliersLargeTopic(t *testing.T) {
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(&Entity{}); err != nil {
		t.Fatal(err)
	}
	const topicID = 991234
	t.Cleanup(func() { conn.Where("topic_id = ?", topicID).Delete(&Entity{}) })
	rows := make([]ReplierStat, 12000)
	for i := range rows {
		rows[i] = ReplierStat{UserID: uint64(i + 1), ReplyCount: 2, LastReplyAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)}
	}
	if err := BulkUpsertRepliersTx(conn, topicID, rows); err != nil {
		t.Fatal(err)
	}
	var count int64
	conn.Model(&Entity{}).Where("topic_id = ?", topicID).Count(&count)
	if count != int64(len(rows)) {
		t.Fatalf("got %d rows, want %d", count, len(rows))
	}
}
