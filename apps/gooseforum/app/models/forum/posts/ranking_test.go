package posts

import (
	"context"
	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"testing"
	"time"
)

func TestLegacyReplyAggregateCountsAnonymousWithoutLoadingBodies(t *testing.T) {
	conn := db.Connect()
	if err := conn.AutoMigrate(&Entity{}, &users.EntityComplete{}); err != nil {
		t.Fatal(err)
	}
	user := users.EntityComplete{Id: 7, Username: "legacy-rank-actor"}
	if err := conn.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Unscoped().Delete(&user) })
	now := time.Now()
	rows := []Entity{{TopicId: 987611, UserId: 7, PostNo: 2, IsAnonymous: true, CreatedAt: now}, {TopicId: 987611, UserId: 7, PostNo: 3, CreatedAt: now}}
	if err := conn.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Unscoped().Where("topic_id = ?", 987611).Delete(&Entity{}) })
	got, err := RankRepliers(context.Background(), 987611)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Replies != 2 || got[0].LastPublicReplyAt == nil {
		t.Fatalf("bad legacy aggregate %+v", got)
	}
}
func TestDailyReplyWindowCapsEachActorAcrossTheWholeWindow(t *testing.T) {
	conn := db.Connect()
	if err := conn.AutoMigrate(&Entity{}, &users.EntityComplete{}); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	user := users.EntityComplete{Id: 987612, Username: "rank-replier"}
	if err := conn.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Unscoped().Where("topic_id = ?", 987612).Delete(&Entity{}); conn.Delete(&user) })
	rows := []Entity{}
	for i, h := range []int{20, 14, 8, 1} {
		at := now.Add(-time.Duration(h) * time.Hour)
		rows = append(rows, Entity{TopicId: 987612, PostNo: uint64(i + 2), UserId: user.Id, FirstPublicAt: &at})
	}
	if err := conn.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}
	got, err := DailyReplyFacts(context.Background(), 987612, now.Add(-24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 || !got[2].FirstPublicAt.Equal(*rows[2].FirstPublicAt) {
		t.Fatalf("per-band cap or newest-three selected %+v", got)
	}
}
