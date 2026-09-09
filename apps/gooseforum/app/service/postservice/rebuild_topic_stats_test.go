package postservice

import (
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topicUserStat"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// PR #575 review 修复的回归锚点：重建必须（1）保留历史 last_reply_at
// （MAX(created_at)，而非增量路径的部署时间）；（2）作者在 top-3 LIMIT
// 前排除（作者高频回复不得挤占其他参与者名额）；（3）不扰动
// topics.updated_at 首页排序键；（4）匿名/治理删除楼层不进入任何身份
// 表面但计入楼层计数与最后回复指针。
func TestRebuildTopicPostStatsTxPreservesHistoryAndExcludesAuthor(t *testing.T) {
	conn := newRebuildStatsTestDB(t)

	const (
		topicID  = uint64(9_100_000_001)
		authorID = uint64(9_100_000_090)
		u2       = uint64(9_100_000_092)
		u3       = uint64(9_100_000_093)
		u4       = uint64(9_100_000_094)
		u5       = uint64(9_100_000_095)
		u6       = uint64(9_100_000_096)
	)
	base := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
	staleUpdatedAt := base.Add(48 * time.Hour)

	if err := conn.Create(&topics.Entity{Id: topicID, Title: "rebuild", UserId: authorID, Status: 1}).Error; err != nil {
		t.Fatalf("create topic: %v", err)
	}
	if err := conn.Model(&topics.Entity{}).Where("id = ?", topicID).
		UpdateColumn("updated_at", staleUpdatedAt).Error; err != nil {
		t.Fatalf("set stale updated_at: %v", err)
	}

	newPost := func(id uint64, postNo uint64, userID uint64, createdAt time.Time, anonymous bool, visibility string) {
		t.Helper()
		entity := posts.Entity{
			Id:               id,
			TopicId:          topicID,
			PostNo:           postNo,
			UserId:           userID,
			Content:          "c",
			IsAnonymous:      anonymous,
			VisibilityStatus: visibility,
			RetentionStatus:  posts.RetentionNormal,
			CreatedAt:        createdAt,
		}
		if err := conn.Create(&entity).Error; err != nil {
			t.Fatalf("create post %d: %v", id, err)
		}
	}
	// 作者自己回复最多（占榜场景：count=2 压过其他 count=1 的回复者）。
	newPost(11, 1, authorID, base, false, posts.VisibilityActive)
	newPost(12, 2, authorID, base.Add(1*time.Hour), false, posts.VisibilityActive)
	newPost(13, 3, authorID, base.Add(2*time.Hour), false, posts.VisibilityActive)
	newPost(14, 4, u2, base.Add(3*time.Hour), false, posts.VisibilityActive)
	newPost(15, 5, u3, base.Add(4*time.Hour), false, posts.VisibilityActive)
	newPost(16, 6, u4, base.Add(5*time.Hour), false, posts.VisibilityActive)
	newPost(17, 7, u5, base.Add(6*time.Hour), true, posts.VisibilityActive)
	newPost(18, 8, u6, base.Add(7*time.Hour), false, posts.VisibilityUserDeleted)

	err := conn.Transaction(func(tx *gorm.DB) error {
		return RebuildTopicPostStatsTx(tx, topics.Entity{Id: topicID})
	})
	if err != nil {
		t.Fatalf("rebuild in tx: %v", err)
	}

	// topic_user_stat：作者自己的回复同样保留参与行（与增量路径口径一致，
	// 「参与过的话题」列表依赖它），匿名/治理删除楼层排除；
	// last_reply_at = 历史回复的 MAX(created_at)，绝不被重建动作的时间覆盖。
	var stats []topicUserStat.Entity
	if err := conn.Where("topic_id = ?", topicID).Find(&stats).Error; err != nil {
		t.Fatalf("read stats: %v", err)
	}
	statByUser := map[uint64]topicUserStat.Entity{}
	for _, s := range stats {
		statByUser[s.UserId] = s
	}
	if len(stats) != 4 {
		t.Fatalf("stat rows=%d (%v), want 4 (author replies + 3 repliers)", len(stats), stats)
	}
	for _, excluded := range []uint64{u5, u6} {
		if _, ok := statByUser[excluded]; ok {
			t.Fatalf("stats must exclude user %d (anonymous/removed)", excluded)
		}
	}
	if authorRow, ok := statByUser[authorID]; !ok {
		t.Fatalf("author's own replies must keep a participation row")
	} else if authorRow.ReplyCount != 2 || !authorRow.LastReplyAt.Equal(base.Add(2*time.Hour)) {
		t.Fatalf("author stat row count=%d last=%v, want 2 @ %v", authorRow.ReplyCount, authorRow.LastReplyAt, base.Add(2*time.Hour))
	}
	wantLastReplyAt := map[uint64]time.Time{
		u2: base.Add(3 * time.Hour),
		u3: base.Add(4 * time.Hour),
		u4: base.Add(5 * time.Hour),
	}
	for userID, want := range wantLastReplyAt {
		row := statByUser[userID]
		if row.ReplyCount != 1 {
			t.Errorf("user %d reply_count=%d, want 1", userID, row.ReplyCount)
		}
		if !row.LastReplyAt.Equal(want) {
			t.Errorf("user %d last_reply_at=%v, want historical %v", userID, row.LastReplyAt, want)
		}
	}

	var after topics.Entity
	if err := conn.Unscoped().Where("id = ?", topicID).First(&after).Error; err != nil {
		t.Fatalf("reload topic: %v", err)
	}

	// posters：作者置前 + 非作者回复者 top-3（作者占榜不得压缩名额）。
	if len(after.Posters) != 4 {
		t.Fatalf("posters=%v, want 4 (author + 3 repliers)", after.Posters)
	}
	if after.Posters[0].UserID != authorID {
		t.Fatalf("posters[0]=%d, want author %d first", after.Posters[0].UserID, authorID)
	}
	seen := map[uint64]bool{}
	for _, poster := range after.Posters[1:] {
		seen[poster.UserID] = true
	}
	for _, want := range []uint64{u2, u3, u4} {
		if !seen[want] {
			t.Fatalf("posters missing replier %d: %v", want, after.Posters)
		}
	}

	// 派生计数与指针：匿名楼层计入计数与最后回复，治理删除楼层不计。
	if after.PostCount != 7 || after.ReplyCount != 6 {
		t.Fatalf("counts post=%d reply=%d, want 7/6", after.PostCount, after.ReplyCount)
	}
	if after.LastPostId != 17 {
		t.Fatalf("lastPostId=%d, want 17 (latest active anonymous reply)", after.LastPostId)
	}
	if after.LastPostedAt == nil || !after.LastPostedAt.Equal(base.Add(6*time.Hour)) {
		t.Fatalf("lastPostedAt=%v, want %v", after.LastPostedAt, base.Add(6*time.Hour))
	}

	// 首页排序键不被派生重建触碰。
	if !after.UpdatedAt.Equal(staleUpdatedAt) {
		t.Fatalf("updated_at perturbed: %v, want %v", after.UpdatedAt, staleUpdatedAt)
	}
}

func newRebuildStatsTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:postservice-rebuild-stats?mode=memory&cache=shared"
	conn, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := conn.AutoMigrate(&topics.Entity{}, &posts.Entity{}, &topicUserStat.Entity{}); err != nil {
		t.Fatalf("migrate rebuild stats fixtures: %v", err)
	}
	t.Cleanup(func() {
		conn.Where("1 = 1").Delete(&topics.Entity{})
		conn.Where("1 = 1").Delete(&posts.Entity{})
		conn.Where("1 = 1").Delete(&topicUserStat.Entity{})
	})
	return conn
}
