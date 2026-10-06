package feedservice

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topicCategoryIndex"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"gorm.io/gorm"
)

func TestPublicCategoryPoolReusesIDsWithoutMixingCategoryLimitOrHash(t *testing.T) {
	conn := telemetryDB(t)
	if err := conn.AutoMigrate(&topicCategoryIndex.Entity{}); err != nil {
		t.Fatal(err)
	}
	publicPools.clear(0)
	t.Cleanup(func() { publicPools.clear(0) })
	now := time.Now()
	rows := []topics.Entity{
		{Id: 935001, Status: 1, FirstPublicAt: &now, RankScore: 600},
		{Id: 935002, Status: 1, FirstPublicAt: &now, RankScore: 500},
		{Id: 935003, Status: 1, FirstPublicAt: &now, RankScore: 700},
	}
	if err := conn.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}
	indices := []topicCategoryIndex.Entity{{TopicId: 935001, CategoryId: 951, Effective: 1}, {TopicId: 935002, CategoryId: 951, Effective: 1}, {TopicId: 935003, CategoryId: 952, Effective: 1}}
	if err := conn.Create(&indices).Error; err != nil {
		t.Fatal(err)
	}
	queries := 0
	if err := conn.Callback().Query().After("gorm:query").Register("test_public_pool_count", func(tx *gorm.DB) { queries++ }); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := conn.Callback().Query().Remove("test_public_pool_count"); err != nil {
			t.Error(err)
		}
	})
	check := func(q topics.Recall, want []uint64) {
		t.Helper()
		got, err := recallPool(context.Background(), q)
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Fatalf("recall(%+v) = %v, %v; want %v", q, got, err, want)
		}
	}
	q := topics.Recall{Source: "category", Hash: "test-v1", Category: 951, Limit: 1}
	check(q, []uint64{935001})
	before := queries
	check(q, []uint64{935001})
	if queries != before {
		t.Fatal("same public pool performed another SQL recall")
	}
	q.Limit = 2
	check(q, []uint64{935001, 935002})
	q.Category = 952
	check(q, []uint64{935003})
	if err := conn.Model(&topics.Entity{}).Where("id = ?", 935002).UpdateColumn("rank_score", 800).Error; err != nil {
		t.Fatal(err)
	}
	q.Category = 951
	q.Hash = "test-v2"
	check(q, []uint64{935002, 935001})
}
