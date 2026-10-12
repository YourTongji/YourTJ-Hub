package hotdataserve

import (
	"fmt"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/category"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topicCategoryIndex"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"gorm.io/gorm"
)

func blockedViewerTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(&topics.Entity{}, &posts.Entity{}, &users.EntityComplete{}, &users.BlockEntity{}, &category.Entity{}, &topicCategoryIndex.Entity{}); err != nil {
		t.Fatalf("migrate blocked viewer tables: %v", err)
	}
	return conn
}

func seedBlockedViewerTopic(t *testing.T, conn *gorm.DB, id, author uint64, at time.Time) {
	t.Helper()
	topic := topics.Entity{Id: id, UserId: author, Title: fmt.Sprintf("topic %d", id), Status: 1, TopicType: topics.TopicTypeForum, FirstPostId: id, CreatedAt: at, UpdatedAt: at}
	post := posts.Entity{Id: id, TopicId: id, PostNo: 1, UserId: author, Content: "body", CreatedAt: at, UpdatedAt: at}
	if err := conn.Create(&topic).Error; err != nil {
		t.Fatalf("create topic %d: %v", id, err)
	}
	if err := conn.Create(&post).Error; err != nil {
		t.Fatalf("create first post %d: %v", id, err)
	}
	t.Cleanup(func() {
		conn.Unscoped().Delete(&posts.Entity{}, id)
		conn.Unscoped().Delete(&topics.Entity{}, id)
	})
}

func seedBlockedViewerAuthor(t *testing.T, conn *gorm.DB, id uint64) {
	t.Helper()
	if err := conn.Create(&users.EntityComplete{Id: id, Username: fmt.Sprintf("u%d", id)}).Error; err != nil {
		t.Fatalf("create author %d: %v", id, err)
	}
	t.Cleanup(func() { conn.Unscoped().Delete(&users.EntityComplete{}, id) })
}

func seedBlockedViewerRelation(t *testing.T, conn *gorm.DB, owner, target uint64) {
	t.Helper()
	row := users.BlockEntity{OwnerID: owner, TargetUserID: target, CreatedAt: time.Now()}
	if err := conn.Create(&row).Error; err != nil {
		t.Fatalf("create block %d->%d: %v", owner, target, err)
	}
	t.Cleanup(func() { conn.Delete(&users.BlockEntity{}, "owner_id = ? AND target_user_id = ?", owner, target) })
}

func topicListIDs(page TopicSimpleVoPage) []uint64 {
	ids := make([]uint64, 0, len(page.Topics))
	for _, item := range page.Topics {
		ids = append(ids, item.Id)
	}
	return ids
}

// TestBlockedViewerListBypassesSharedCache 覆盖三件事：有屏蔽关系的观看者列表
// 被过滤、过滤结果不进入全站共享缓存、且屏蔽变更在共享缓存窗口内立即生效。
func TestBlockedViewerListBypassesSharedCache(t *testing.T) {
	conn := blockedViewerTestDB(t)
	const viewer, blockedAuthor, neutralAuthor, cleanViewer uint64 = 885001, 885002, 885003, 885004
	seedBlockedViewerAuthor(t, conn, blockedAuthor)
	seedBlockedViewerAuthor(t, conn, neutralAuthor)
	seedBlockedViewerRelation(t, conn, viewer, blockedAuthor)
	at := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	seedBlockedViewerTopic(t, conn, 885010, blockedAuthor, at.Add(time.Hour))
	seedBlockedViewerTopic(t, conn, 885011, neutralAuthor, at)
	ClearTopicListCache()
	t.Cleanup(ClearTopicListCache)

	if got := topicListIDs(GetLatestTopicsSimpleVoPaginated(viewer, 1, "latest")); len(got) != 1 || got[0] != 885011 {
		t.Fatalf("blocked viewer topics=%v, want only 885011", got)
	}

	// 游客与无屏蔽登录用户必须仍拿到完整公开列表：过滤结果若被写入
	// 共享缓存，这里会只剩 885011。
	if got := topicListIDs(GetLatestTopicsSimpleVoPaginated(0, 1, "latest")); len(got) != 2 || got[0] != 885010 || got[1] != 885011 {
		t.Fatalf("guest topics=%v, want both topics", got)
	}
	if got := topicListIDs(GetLatestTopicsSimpleVoPaginated(cleanViewer, 1, "latest")); len(got) != 2 {
		t.Fatalf("unblocked viewer topics=%v, want both topics", got)
	}

	// 共享缓存 TTL 内新增屏蔽关系必须立即对回归的观看者生效。
	seedBlockedViewerRelation(t, conn, viewer, neutralAuthor)
	if got := topicListIDs(GetLatestTopicsSimpleVoPaginated(viewer, 1, "latest")); len(got) != 0 {
		t.Fatalf("topics after new block=%v, want empty", got)
	}
}

func TestBlockedViewerCategoryListBypassesSharedCache(t *testing.T) {
	conn := blockedViewerTestDB(t)
	const viewer, blockedAuthor, neutralAuthor uint64 = 885021, 885022, 885023
	const categoryID uint64 = 885090
	seedBlockedViewerAuthor(t, conn, blockedAuthor)
	seedBlockedViewerAuthor(t, conn, neutralAuthor)
	seedBlockedViewerRelation(t, conn, blockedAuthor, viewer) // 反向：作者拉黑观看者
	if err := conn.Create(&category.Entity{Id: categoryID, Name: "Blocked", Slug: "blocked"}).Error; err != nil {
		t.Fatalf("create category: %v", err)
	}
	t.Cleanup(func() { conn.Unscoped().Delete(&category.Entity{}, categoryID) })
	at := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	seedBlockedViewerTopic(t, conn, 885030, blockedAuthor, at.Add(time.Hour))
	seedBlockedViewerTopic(t, conn, 885031, neutralAuthor, at)
	for _, topicID := range []uint64{885030, 885031} {
		index := topicCategoryIndex.Entity{TopicId: topicID, CategoryId: categoryID, Effective: 1}
		if err := conn.Create(&index).Error; err != nil {
			t.Fatalf("index topic %d: %v", topicID, err)
		}
		t.Cleanup(func() { conn.Delete(&topicCategoryIndex.Entity{}, "topic_id = ?", topicID) })
	}
	ClearTopicListCache()
	t.Cleanup(ClearTopicListCache)

	if got := topicListIDs(GetTopicsByCategorySimpleVo(viewer, categoryID, "latest", 1)); len(got) != 1 || got[0] != 885031 {
		t.Fatalf("blocked viewer category topics=%v, want only 885031", got)
	}
	if got := topicListIDs(GetTopicsByCategorySimpleVo(0, categoryID, "latest", 1)); len(got) != 2 {
		t.Fatalf("guest category topics=%v, want both topics", got)
	}
}
