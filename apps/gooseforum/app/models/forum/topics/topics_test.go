package topics

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topicCategoryIndex"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestGetWithContextHonorsCancellation(t *testing.T) {
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(&Entity{}); err != nil {
		t.Fatalf("migrate topics table: %v", err)
	}

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := GetWithContext(ctx, 1)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("GetWithContext() err=%v, want context.Canceled", err)
	}
}

func TestTopicAndPostSchemaMigrates(t *testing.T) {
	conn, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	if err := conn.AutoMigrate(&Entity{}, &posts.Entity{}); err != nil {
		t.Fatalf("migrate topic/post schema: %v", err)
	}

	if !conn.Migrator().HasTable("topics") {
		t.Fatal("topics table was not created")
	}
	if !conn.Migrator().HasTable("posts") {
		t.Fatal("posts table was not created")
	}

	for _, column := range []string{"title", "category_id", "first_post_id", "last_post_id", "post_count", "reply_count", "excerpt"} {
		if !conn.Migrator().HasColumn(&Entity{}, column) {
			t.Fatalf("topics.%s column was not created", column)
		}
	}

	for _, column := range []string{"topic_id", "post_no", "user_id", "content", "reply_to_post_id", "process_status"} {
		if !conn.Migrator().HasColumn(&posts.Entity{}, column) {
			t.Fatalf("posts.%s column was not created", column)
		}
	}

	if !conn.Migrator().HasIndex(&posts.Entity{}, "idx_posts_topic_no") {
		t.Fatal("posts idx_posts_topic_no index was not created")
	}
	if !conn.Migrator().HasIndex(&posts.Entity{}, "idx_posts_topic_created") {
		t.Fatal("posts idx_posts_topic_created index was not created")
	}
}

func TestPublishedQueriesExcludeNonPublicTopics(t *testing.T) {
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(&Entity{}, &posts.Entity{}); err != nil {
		t.Fatalf("migrate published topic tables: %v", err)
	}

	const (
		firstTopicID            = uint64(110)
		secondTopicID           = uint64(120)
		draftTopicID            = uint64(130)
		blockedTopicID          = uint64(140)
		blockedFirstPostTopicID = uint64(150)
	)
	topicIDs := []uint64{firstTopicID, secondTopicID, draftTopicID, blockedTopicID, blockedFirstPostTopicID}
	postIDs := []uint64{1010, 1020, 1030, 1040, 1050}
	conn.Unscoped().Where("id IN ?", topicIDs).Delete(&Entity{})
	conn.Unscoped().Where("id IN ?", postIDs).Delete(&posts.Entity{})
	t.Cleanup(func() {
		conn.Unscoped().Where("id IN ?", topicIDs).Delete(&Entity{})
		conn.Unscoped().Where("id IN ?", postIDs).Delete(&posts.Entity{})
	})

	now := time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)
	if err := conn.Create(&[]posts.Entity{
		{Id: postIDs[0], TopicId: firstTopicID, PostNo: 1, Content: "first", ProcessStatus: posts.ProcessStatusNormal, CreatedAt: now},
		{Id: postIDs[1], TopicId: secondTopicID, PostNo: 1, Content: "second", ProcessStatus: posts.ProcessStatusNormal, CreatedAt: now},
		{Id: postIDs[2], TopicId: draftTopicID, PostNo: 1, Content: "draft", ProcessStatus: posts.ProcessStatusNormal, CreatedAt: now},
		{Id: postIDs[3], TopicId: blockedTopicID, PostNo: 1, Content: "blocked topic", ProcessStatus: posts.ProcessStatusNormal, CreatedAt: now},
		{Id: postIDs[4], TopicId: blockedFirstPostTopicID, PostNo: 1, Content: "blocked first post", ProcessStatus: posts.ProcessStatusBlocked, CreatedAt: now},
	}).Error; err != nil {
		t.Fatalf("create published topic posts: %v", err)
	}
	if err := conn.Create(&[]Entity{
		{Id: firstTopicID, Title: "first", FirstPostId: postIDs[0], Status: 1, ProcessStatus: ProcessStatusNormal, CreatedAt: now, UpdatedAt: now},
		{Id: secondTopicID, Title: "second", FirstPostId: postIDs[1], Status: 1, ProcessStatus: ProcessStatusNormal, CreatedAt: now, UpdatedAt: now},
		{Id: draftTopicID, Title: "draft", FirstPostId: postIDs[2], Status: 0, ProcessStatus: ProcessStatusNormal, CreatedAt: now, UpdatedAt: now},
		{Id: blockedTopicID, Title: "blocked", FirstPostId: postIDs[3], Status: 1, ProcessStatus: ProcessStatusBlocked, CreatedAt: now, UpdatedAt: now},
		{Id: blockedFirstPostTopicID, Title: "blocked first", FirstPostId: postIDs[4], Status: 1, ProcessStatus: ProcessStatusNormal, CreatedAt: now, UpdatedAt: now},
	}).Error; err != nil {
		t.Fatalf("create published topics: %v", err)
	}

	// GetPublishedBeforeID 按 id 倒序（最新优先）分页，且同样过滤草稿/封禁/首帖异常。
	beforeBatch, err := GetPublishedBeforeID(200, 1)
	if err != nil {
		t.Fatalf("GetPublishedBeforeID first batch: %v", err)
	}
	if len(beforeBatch) != 1 || beforeBatch[0].Id != secondTopicID {
		t.Fatalf("before published batch = %#v, want topic %d", beforeBatch, secondTopicID)
	}
	beforeBatch2, err := GetPublishedBeforeID(secondTopicID, 10)
	if err != nil {
		t.Fatalf("GetPublishedBeforeID second batch: %v", err)
	}
	if len(beforeBatch2) != 1 || beforeBatch2[0].Id != firstTopicID {
		t.Fatalf("second before batch = %#v, want only topic %d", beforeBatch2, firstTopicID)
	}

	if topic, err := GetPublished(firstTopicID); err != nil || topic.Id != firstTopicID {
		t.Fatalf("GetPublished(%d) = %#v, err=%v", firstTopicID, topic, err)
	}
	for _, id := range []uint64{draftTopicID, blockedTopicID, blockedFirstPostTopicID} {
		if _, err := GetPublished(id); !errors.Is(err, gorm.ErrRecordNotFound) {
			t.Fatalf("GetPublished(%d) err=%v, want record not found", id, err)
		}
	}
}

func TestTopicRepositoryParity(t *testing.T) {
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(&Entity{}, &posts.Entity{}, &topicCategoryIndex.Entity{}); err != nil {
		t.Fatalf("migrate topic repository tables: %v", err)
	}
	conn.Where("1 = 1").Delete(&Entity{})
	conn.Where("1 = 1").Delete(&posts.Entity{})
	conn.Where("1 = 1").Delete(&topicCategoryIndex.Entity{})

	now := time.Date(2026, 7, 7, 12, 0, 0, 0, time.UTC)
	conn.Create(&[]Entity{
		{Id: 10, Title: "zeta topic", CategoryIds: []uint64{3}, UserId: 1, Status: 1, ProcessStatus: 0, ReplyCount: 1, ViewCount: 9, PinWeight: 0, CreatedAt: now, UpdatedAt: now},
		{Id: 20, Title: "alpha topic", CategoryIds: []uint64{4}, UserId: 2, Status: 1, ProcessStatus: 0, ReplyCount: 4, ViewCount: 3, PinWeight: 20, CreatedAt: now.Add(time.Minute), UpdatedAt: now.Add(time.Minute)},
		{Id: 30, Title: "draft topic", CategoryIds: []uint64{3}, UserId: 1, Status: 0, ProcessStatus: 0, CreatedAt: now.Add(2 * time.Minute), UpdatedAt: now.Add(2 * time.Minute)},
		{Id: 40, Title: "blocked topic", CategoryIds: []uint64{3}, UserId: 1, Status: 1, ProcessStatus: 1, CreatedAt: now.Add(3 * time.Minute), UpdatedAt: now.Add(3 * time.Minute)},
	})
	conn.Create(&[]topicCategoryIndex.Entity{
		{TopicId: 10, CategoryId: 3, Effective: 1},
		{TopicId: 20, CategoryId: 4, Effective: 1},
		{TopicId: 40, CategoryId: 3, Effective: 1},
	})

	page := Page(PageQuery{Page: 1, PageSize: 10, FilterStatus: true, CategoryId: 3, Sort: "new"})
	if len(page.Data) != 1 || page.Data[0].Id != 10 {
		t.Fatalf("Page() filtered ids = %#v, want only topic 10", page.Data)
	}

	moderationPage := PageForModeration(ModerationPageQuery{
		Page:                1,
		PageSize:            10,
		FilterProcessStatus: true,
		ProcessStatus:       1,
		CategoryIDs:         []uint64{3},
	})
	if len(moderationPage.Data) != 1 || moderationPage.Data[0].Id != 40 {
		t.Fatalf("PageForModeration() = %#v, want blocked topic 40", moderationPage.Data)
	}

	if err := UpdatePinWeight(10, 99); err != nil {
		t.Fatalf("UpdatePinWeight() err=%v", err)
	}
	adminPage := PageForAdmin(AdminPageQuery{Page: 1, PageSize: 10, UserId: 1})
	if len(adminPage.Data) != 3 || adminPage.Data[0].Id != 10 || adminPage.Data[1].Id != 40 || adminPage.Data[2].Id != 30 {
		t.Fatalf("PageForAdmin() ids = %#v, want pinned topic 10 before topics 40, 30", adminPage.Data)
	}

	if err := UpdateProcessStatus(10, 1); err != nil {
		t.Fatalf("UpdateProcessStatus() err=%v", err)
	}
	if got := GetSimple(10); got.ProcessStatus != 1 {
		t.Fatalf("GetSimple(10).ProcessStatus=%d, want 1", got.ProcessStatus)
	}
	if got := Get(10); got.PinWeight != 99 {
		t.Fatalf("Get(10).PinWeight=%d, want 99", got.PinWeight)
	}

	if IncrementLike(Entity{Id: 10}) != 1 {
		t.Fatal("IncrementLike() rows != 1")
	}
	if DecrementLike(Entity{Id: 10}) != 1 {
		t.Fatal("DecrementLike() rows != 1")
	}
	if got := Get(10); got.LikeCount != 0 {
		t.Fatalf("LikeCount=%d, want back to 0", got.LikeCount)
	}

	nextNo, err := ReservePostSequence(10)
	if err != nil {
		t.Fatalf("ReservePostSequence() err=%v", err)
	}
	if nextNo != 1 {
		t.Fatalf("ReservePostSequence()=%d, want 1", nextNo)
	}
	lastPostedAt := now.Add(5 * time.Minute)
	if err := IncrementPostFast(10, []Poster{{UserID: 2}}, 101, lastPostedAt); err != nil {
		t.Fatalf("IncrementPostFast() err=%v", err)
	}
	if got := Get(10); got.PostCount != 1 || got.ReplyCount != 2 || len(got.Posters) != 1 || got.LastPostId != 101 || got.LastPostedAt == nil || !got.LastPostedAt.Equal(lastPostedAt) {
		t.Fatalf("after IncrementPostFast() topic=%#v", got)
	}
	if err := IncrementPostFast(10, []Poster{{UserID: 3}}, 99, lastPostedAt.Add(-time.Minute)); err != nil {
		t.Fatalf("older IncrementPostFast() err=%v", err)
	}
	if got := Get(10); got.LastPostId != 101 || got.LastPostedAt == nil || !got.LastPostedAt.Equal(lastPostedAt) {
		t.Fatalf("older increment moved last post backward: topic=%#v", got)
	}
	previousPostedAt := now.Add(time.Minute)
	if err := DecrementPostFast(10, []Poster{}, 100, previousPostedAt); err != nil {
		t.Fatalf("DecrementPostFast() err=%v", err)
	}
	if got := Get(10); got.ReplyCount != 2 || got.LastPostId != 100 || got.LastPostedAt == nil || !got.LastPostedAt.Equal(previousPostedAt) {
		t.Fatalf("after DecrementPostFast() topic=%#v", got)
	}
}

// PagePendingReview 必须显式排除软删话题：Count 绕过 GORM 软删 scope，
// 只靠 scope 会导致 total 计入已删话题而 Data 不含（review：subquery passes
// on soft-deleted topics）。
func TestPagePendingReviewExcludesSoftDeletedTopics(t *testing.T) {
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(&Entity{}); err != nil {
		t.Fatalf("migrate pending review tables: %v", err)
	}
	conn.Unscoped().Where("id IN ?", []uint64{210, 220, 230}).Delete(&Entity{})
	t.Cleanup(func() {
		conn.Unscoped().Where("id IN ?", []uint64{210, 220, 230}).Delete(&Entity{})
	})

	now := time.Now()
	conn.Create(&[]Entity{
		{Id: 210, Title: "pending live", Status: 1, ProcessStatus: ProcessStatusPending, TopicType: TopicTypeForum, CreatedAt: now, UpdatedAt: now},
		{Id: 220, Title: "pending soft-deleted", Status: 1, ProcessStatus: ProcessStatusPending, TopicType: TopicTypeForum, CreatedAt: now, UpdatedAt: now},
		{Id: 230, Title: "pending wiki", Status: 1, ProcessStatus: ProcessStatusPending, TopicType: TopicTypeWiki, CreatedAt: now, UpdatedAt: now},
	})
	// 软删 220：直接置 deleted_at（保留 process_status=pending 模拟历史脏数据）。
	conn.Unscoped().Table("topics").Where("id = ?", 220).
		Update("deleted_at", now)

	page := PagePendingReview(1, 50)
	if page.Total != 1 {
		t.Fatalf("PagePendingReview total=%d, want 1 (soft-deleted excluded from count)", page.Total)
	}
	if len(page.Data) != 1 || page.Data[0].Id != 210 {
		t.Fatalf("PagePendingReview data=%+v, want only live pending topic 210", page.Data)
	}
}
