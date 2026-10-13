package hotdataserve

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/category"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topicCategoryIndex"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
)

type blockedPeersQueryCounter struct {
	count atomic.Int64
}

func (l *blockedPeersQueryCounter) LogMode(gormLogger.LogLevel) gormLogger.Interface { return l }
func (l *blockedPeersQueryCounter) Info(context.Context, string, ...any)             {}
func (l *blockedPeersQueryCounter) Warn(context.Context, string, ...any)             {}
func (l *blockedPeersQueryCounter) Error(context.Context, string, ...any)            {}
func (l *blockedPeersQueryCounter) Trace(context.Context, time.Time, func() (string, int64), error) {
	l.count.Add(1)
}

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

// clearBlockedViewerCaches 清理共享列表缓存与读者级屏蔽检查缓存：后者按进程
// 存活，测试必须显式失效自己用到的读者 id，避免复用 id 的用例被陈旧判定污染。
func clearBlockedViewerCaches(t *testing.T, viewerIDs ...uint64) {
	t.Helper()
	ClearTopicListCache()
	users.InvalidateBlockedPeersCache(viewerIDs...)
	t.Cleanup(func() {
		ClearTopicListCache()
		users.InvalidateBlockedPeersCache(viewerIDs...)
	})
}

// TestBlockedViewerListBypassesSharedCache 覆盖三件事：有屏蔽关系的观看者列表
// 被过滤、过滤结果不进入全站共享缓存、且带屏蔽关系的观看者每次请求都读实时
// 关系（本用例直写库新增的关系立即生效；经生产写路径的立即性由
// TestBlockedPeersCacheServesWithinTTLAndInvalidatesOnWrite 与
// TestHomePayloadAppliesBlockChangesImmediately 覆盖）。
func TestBlockedViewerListBypassesSharedCache(t *testing.T) {
	conn := blockedViewerTestDB(t)
	const viewer, blockedAuthor, neutralAuthor, cleanViewer uint64 = 885001, 885002, 885003, 885004
	seedBlockedViewerAuthor(t, conn, blockedAuthor)
	seedBlockedViewerAuthor(t, conn, neutralAuthor)
	seedBlockedViewerRelation(t, conn, viewer, blockedAuthor)
	at := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	seedBlockedViewerTopic(t, conn, 885010, blockedAuthor, at.Add(time.Hour))
	seedBlockedViewerTopic(t, conn, 885011, neutralAuthor, at)
	clearBlockedViewerCaches(t, viewer, blockedAuthor, neutralAuthor, cleanViewer)

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

	// 观看者已处于绕过共享缓存的路径：直写库新增的关系在下一次请求立即生效。
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
	clearBlockedViewerCaches(t, viewer, blockedAuthor, neutralAuthor)

	if got := topicListIDs(GetTopicsByCategorySimpleVo(viewer, categoryID, "latest", 1)); len(got) != 1 || got[0] != 885031 {
		t.Fatalf("blocked viewer category topics=%v, want only 885031", got)
	}
	if got := topicListIDs(GetTopicsByCategorySimpleVo(0, categoryID, "latest", 1)); len(got) != 2 {
		t.Fatalf("guest category topics=%v, want both topics", got)
	}
}

// TestBlockedViewerCachedRequestIssuesNoQuery 覆盖 issue #1129 的性能验收：
// 登录读者在共享列表缓存窗口内的重复请求必须完全由缓存服务，不得再为屏蔽
// 存在性检查产生每请求 DB 往返；游客路径不触库的既有性质保持不变。
func TestBlockedViewerCachedRequestIssuesNoQuery(t *testing.T) {
	conn := blockedViewerTestDB(t)
	const viewer, author uint64 = 885041, 885042
	seedBlockedViewerAuthor(t, conn, author)
	seedBlockedViewerTopic(t, conn, 885050, author, time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC))
	clearBlockedViewerCaches(t, viewer, author)

	// 第一次请求建立共享列表缓存与读者级屏蔽检查缓存。
	if got := topicListIDs(GetLatestTopicsSimpleVoPaginated(viewer, 1, "latest")); len(got) != 1 || got[0] != 885050 {
		t.Fatalf("warm-up topics=%v, want the author topic", got)
	}

	counter := &blockedPeersQueryCounter{}
	previousLogger := conn.Logger
	conn.Logger = counter
	t.Cleanup(func() { conn.Logger = previousLogger })

	// TTL 窗口内的第二次请求：屏蔽存在性检查与共享列表都必须命中缓存。
	if got := topicListIDs(GetLatestTopicsSimpleVoPaginated(viewer, 1, "latest")); len(got) != 1 || got[0] != 885050 {
		t.Fatalf("cached topics=%v, want the author topic", got)
	}
	if queries := counter.count.Load(); queries != 0 {
		t.Fatalf("cached request issued %d queries, want 0", queries)
	}

	// 游客短路不触库的性质保持不变。
	if got := topicListIDs(GetLatestTopicsSimpleVoPaginated(0, 1, "latest")); len(got) != 1 {
		t.Fatalf("guest topics=%v, want the author topic", got)
	}
	if queries := counter.count.Load(); queries != 0 {
		t.Fatalf("guest request issued %d queries, want 0", queries)
	}
}
