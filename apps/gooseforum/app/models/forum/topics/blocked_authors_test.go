package topics

import (
	"fmt"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userFollow"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"gorm.io/gorm"
)

func blockedAuthorsTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(&Entity{}, &posts.Entity{}, &users.BlockEntity{}, &userFollow.Entity{}); err != nil {
		t.Fatalf("migrate blocked author tables: %v", err)
	}
	return conn
}

func seedBlock(t *testing.T, conn *gorm.DB, owner, target uint64) {
	t.Helper()
	row := users.BlockEntity{OwnerID: owner, TargetUserID: target, CreatedAt: time.Now()}
	if err := conn.Create(&row).Error; err != nil {
		t.Fatalf("create block %d->%d: %v", owner, target, err)
	}
	t.Cleanup(func() { conn.Delete(&users.BlockEntity{}, "owner_id = ? AND target_user_id = ?", owner, target) })
}

func seedBlockedFeedTopic(t *testing.T, conn *gorm.DB, id, author uint64, at time.Time) {
	t.Helper()
	topic := Entity{Id: id, UserId: author, Title: fmt.Sprintf("topic %d", id), Status: 1, TopicType: TopicTypeForum, FirstPostId: id, CreatedAt: at, UpdatedAt: at}
	post := posts.Entity{Id: id, TopicId: id, PostNo: 1, UserId: author, Content: "body", CreatedAt: at, UpdatedAt: at}
	if err := conn.Create(&topic).Error; err != nil {
		t.Fatalf("create topic %d: %v", id, err)
	}
	if err := conn.Create(&post).Error; err != nil {
		t.Fatalf("create first post %d: %v", id, err)
	}
	t.Cleanup(func() {
		conn.Unscoped().Delete(&posts.Entity{}, id)
		conn.Unscoped().Delete(&Entity{}, id)
	})
}

func topicIDs(rows []Entity) []uint64 {
	ids := make([]uint64, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.Id)
	}
	return ids
}

func TestPageExcludesBlockedAuthorsInEitherDirection(t *testing.T) {
	conn := blockedAuthorsTestDB(t)
	const viewer, blocked, blocker, neutral uint64 = 940001, 940002, 940003, 940004
	seedBlock(t, conn, viewer, blocked) // 观看者拉黑作者
	seedBlock(t, conn, blocker, viewer) // 作者拉黑观看者
	at := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	seedBlockedFeedTopic(t, conn, 940010, blocked, at)
	seedBlockedFeedTopic(t, conn, 940011, blocker, at)
	seedBlockedFeedTopic(t, conn, 940012, neutral, at)

	page := Page(PageQuery{Page: 1, PageSize: 10, FilterStatus: true, ExcludeBlockedViewerID: viewer})
	if len(page.Data) != 1 || page.Data[0].Id != 940012 {
		t.Fatalf("filtered page=%v, want only topic 940012", topicIDs(page.Data))
	}

	all := Page(PageQuery{Page: 1, PageSize: 10, FilterStatus: true})
	if len(all.Data) != 3 {
		t.Fatalf("unfiltered page=%v, want all three topics", topicIDs(all.Data))
	}
}

func TestPageBlockFilterKeepsFullPages(t *testing.T) {
	conn := blockedAuthorsTestDB(t)
	const viewer, blocked, neutral uint64 = 940021, 940022, 940023
	seedBlock(t, conn, viewer, blocked)
	at := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	// 被屏蔽作者的话题更新，排在正常话题之前；过滤必须下沉到 SQL，否则第一页会短页。
	seedBlockedFeedTopic(t, conn, 940031, blocked, at.Add(4*time.Hour))
	seedBlockedFeedTopic(t, conn, 940032, blocked, at.Add(3*time.Hour))
	seedBlockedFeedTopic(t, conn, 940033, neutral, at.Add(2*time.Hour))
	seedBlockedFeedTopic(t, conn, 940034, neutral, at.Add(time.Hour))

	first := Page(PageQuery{Page: 1, PageSize: 2, FilterStatus: true, ExcludeBlockedViewerID: viewer})
	if len(first.Data) != 2 || first.HasNext || first.Data[0].Id != 940033 || first.Data[1].Id != 940034 {
		t.Fatalf("first page=%v hasNext=%v, want the two visible topics and a final page", topicIDs(first.Data), first.HasNext)
	}
}

func TestFollowingPageExcludesBlockedAuthorsInEitherDirection(t *testing.T) {
	conn := blockedAuthorsTestDB(t)
	const viewer, blocked, blocker, neutral uint64 = 940041, 940042, 940043, 940044
	for _, author := range []uint64{blocked, blocker, neutral} {
		row := userFollow.Entity{UserId: viewer, FollowUserId: author, Status: 1}
		if err := conn.Create(&row).Error; err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { conn.Delete(&userFollow.Entity{}, "user_id = ? AND follow_user_id = ?", viewer, author) })
	}
	seedBlock(t, conn, viewer, blocked)
	seedBlock(t, conn, blocker, viewer)
	at := time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC)
	seedBlockedFeedTopic(t, conn, 940051, blocked, at)
	seedBlockedFeedTopic(t, conn, 940052, blocker, at)
	seedBlockedFeedTopic(t, conn, 940053, neutral, at)

	rows, hasNext, err := FollowingPage(t.Context(), viewer, nil, 20)
	if err != nil {
		t.Fatalf("FollowingPage: %v", err)
	}
	if hasNext || len(rows) != 1 || rows[0].Id != 940053 {
		t.Fatalf("following rows=%v hasNext=%v, want only topic 940053", topicIDs(rows), hasNext)
	}
}
