package feedservice

import (
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/feed"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"gorm.io/gorm"
)

func TestNewTopicCohortUsesActualPublicationAndUniquePeerOutcomes(t *testing.T) {
	conn := telemetryDB(t)
	now := time.Now().Add(-time.Hour).Truncate(time.Second)
	actors := []users.EntityComplete{{Id: 981711, Username: "cohort-author"}, {Id: 981712, Username: "cohort-peer"}}
	if err := conn.Create(&actors).Error; err != nil {
		t.Fatal(err)
	}
	topic := topics.Entity{UserId: actors[0].Id, Status: 1, Title: "new cohort", FirstPublicAt: &now}
	if err := conn.Create(&topic).Error; err != nil {
		t.Fatal(err)
	}
	selfAt, peerAt, pendingAt := now.Add(time.Minute), now.Add(4*time.Minute), now.Add(2*time.Minute)
	rows := []posts.Entity{
		{TopicId: topic.Id, UserId: topic.UserId, PostNo: 1, FirstPublicAt: &now},
		{TopicId: topic.Id, UserId: topic.UserId, PostNo: 2, FirstPublicAt: &selfAt},
		{TopicId: topic.Id, UserId: actors[1].Id, PostNo: 3, FirstPublicAt: &peerAt},
		{TopicId: topic.Id, UserId: actors[1].Id, PostNo: 4, FirstPublicAt: &pendingAt, ProcessStatus: posts.ProcessStatusPending},
	}
	if err := conn.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Model(&topic).UpdateColumn("first_post_id", rows[0].Id).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Unscoped().Delete(&rows); conn.Unscoped().Delete(&topic); conn.Unscoped().Delete(&actors) })
	for i := 0; i < 2; i++ {
		if err := conn.Transaction(func(tx *gorm.DB) error {
			if err := captureNewTopicPublicationTx(tx, feed.Event{TopicID: topic.Id, Kind: "public_reply"}); err != nil {
				return err
			}
			if err := captureNewTopicVisibilityTx(tx, []uint64{topic.Id}, topic.UserId, now.Add(time.Hour)); err != nil {
				return err
			}
			if err := captureNewTopicVisibilityTx(tx, []uint64{topic.Id}, actors[1].Id, now.Add(25*time.Hour)); err != nil {
				return err
			}
			return captureNewTopicVisibilityTx(tx, []uint64{topic.Id}, actors[1].Id, now.Add(time.Hour))
		}); err != nil {
			t.Fatal(err)
		}
	}
	var metrics []feed.MetricsDaily
	if err := conn.Where("feed = ?", "new_topics").Find(&metrics).Error; err != nil {
		t.Fatal(err)
	}
	counts := map[string]int64{}
	for _, row := range metrics {
		counts[row.Metric] += row.Count
		if row.Day != localDay(now) {
			t.Fatal("outcome moved out of publication cohort")
		}
	}
	for metric, want := range map[string]int64{"new_topic_public": 1, "new_topic_visible_24h": 1, "new_topic_first_reply": 1, "first_reply_seconds": 240} {
		if counts[metric] != want {
			t.Fatalf("%s=%d want %d", metric, counts[metric], want)
		}
	}
	if err := conn.Model(&topic).UpdateColumn("first_public_estimated", true).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Where("topic_id = ?", topic.Id).Delete(&feed.NewTopicOutcome{}).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Transaction(func(tx *gorm.DB) error {
		return captureNewTopicPublicationTx(tx, feed.Event{TopicID: topic.Id, Kind: "public_topic"})
	}); err != nil {
		t.Fatal(err)
	}
	var n int64
	conn.Model(&feed.NewTopicOutcome{}).Count(&n)
	if n != 0 {
		t.Fatal("estimated legacy time created another publication cohort")
	}
}
