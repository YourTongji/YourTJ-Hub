package topicUserAction

import (
	"context"
	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/feed"
	"gorm.io/gorm"
	"testing"
)

func TestNativeActionCommitsOwnerCreditAndEventToTheirOwnTables(t *testing.T) {
	conn := db.Connect()
	models := append(feed.Models(), &Entity{})
	if err := conn.AutoMigrate(models...); err != nil {
		t.Fatal(err)
	}
	preferences.Set("ranking.enabled", true)
	preferences.Set("feed.metrics.enabled", true)
	const uid, topic uint64 = 976812, 976813
	t.Cleanup(func() {
		preferences.Set("ranking.enabled", false)
		preferences.Set("feed.metrics.enabled", false)
		conn.Where("user_id = ?", uid).Delete(&Entity{})
		conn.Where("topic_id = ?", topic).Delete(&feed.ActionCredit{})
		conn.Where("user_id = ?", uid).Delete(&feed.Event{})
		conn.Where("user_id = ?", uid).Delete(&feed.Owner{})
		conn.Where("topic_id = ?", topic).Delete(&feed.Schedule{})
	})
	changed, err := SetState(context.Background(), uid, topic, "liked_at", true)
	if err != nil || !changed {
		t.Fatalf("native action failed changed=%v: %v", changed, err)
	}
	changed, err = SetState(context.Background(), uid, topic, "liked_at", true)
	if err != nil || changed {
		t.Fatalf("duplicate state emitted: %v %v", changed, err)
	}
	var events int64
	if err = conn.Model(&feed.Event{}).Where("user_id = ?", uid).Count(&events).Error; err != nil || events != 1 {
		t.Fatalf("events=%d %v", events, err)
	}
	if err = conn.Transaction(func(tx *gorm.DB) error { return feed.CloseTx(tx, uid) }); err != nil {
		t.Fatal(err)
	}
	if _, err = SetState(context.Background(), uid, topic, "liked_at", false); err == nil {
		t.Fatal("closed owner wrote event")
	}
	var native Entity
	if err = conn.First(&native, "user_id = ? AND topic_id = ?", uid, topic).Error; err != nil || native.LikedAt == nil {
		t.Fatal("event failure did not roll back native state")
	}
}
