package datamigration

import (
	"testing"
	"time"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topicUserStat"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
)

// issue #554：存量话题的 topics.posters 由 legacy articles.posters 原样拷入
// （只含楼主），增量路径只在有新回复时修复，老话题首页参与人列长期只显示
// 楼主。回填必须从 active posts 绝对重建 topic_user_stat + posters，口径与
// postservice.RebuildTopicPostStats 一致（楼主置前 + 非匿名回复者取 top-3，
// 匿名/治理删除楼层排除），且不得扰动首页默认排序键 topics.updated_at。
func TestBackfillTopicPostStatsRepairsLegacyPosters(t *testing.T) {
	conn := db.Connect()
	if err := conn.AutoMigrate(&topics.Entity{}, &posts.Entity{}, &topicUserStat.Entity{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	const (
		topicID     = uint64(9_800_002_001)
		authorID    = uint64(9_800_002_099)
		replier1    = uint64(9_800_002_101)
		replier2    = uint64(9_800_002_102)
		replier3    = uint64(9_800_002_103)
		anonymousID = uint64(9_800_002_104)
		removedID   = uint64(9_800_002_105)

		firstPostID = uint64(9_800_002_011)
		reply1ID    = uint64(9_800_002_012)
		reply2ID    = uint64(9_800_002_013)
		reply3ID    = uint64(9_800_002_014)
		anonPostID  = uint64(9_800_002_015)
		removedPID  = uint64(9_800_002_016)
	)
	base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	t.Cleanup(func() {
		conn.Unscoped().Where("id = ?", topicID).Delete(&topics.Entity{})
		conn.Unscoped().Where("topic_id = ?", topicID).Delete(&posts.Entity{})
		conn.Where("topic_id = ?", topicID).Delete(&topicUserStat.Entity{})
	})

	// 存量坏话题：posters 只含楼主（legacy 拷贝形态）；updated_at 固定为历史
	// 值，用于断言回填不扰动排序键。
	staleUpdatedAt := base.Add(48 * time.Hour)
	topic := topics.Entity{
		Id:      topicID,
		Title:   "legacy posters topic",
		UserId:  authorID,
		Status:  1,
		Posters: []topics.Poster{{UserID: authorID}},
	}
	if err := conn.Create(&topic).Error; err != nil {
		t.Fatalf("create topic: %v", err)
	}
	if err := conn.Model(&topics.Entity{}).Where("id = ?", topicID).
		UpdateColumn("updated_at", staleUpdatedAt).Error; err != nil {
		t.Fatalf("set stale updated_at: %v", err)
	}

	newPost := func(id uint64, postNo uint64, userID uint64, createdAt time.Time, anonymous bool, visibility string) {
		entity := posts.Entity{
			Id:               id,
			TopicId:          topicID,
			PostNo:           postNo,
			UserId:           userID,
			Content:          "post content",
			IsAnonymous:      anonymous,
			VisibilityStatus: visibility,
			RetentionStatus:  posts.RetentionNormal,
			CreatedAt:        createdAt,
		}
		if err := conn.Create(&entity).Error; err != nil {
			t.Fatalf("create post %d: %v", id, err)
		}
	}
	newPost(firstPostID, 1, authorID, base, false, posts.VisibilityActive)
	newPost(reply1ID, 2, replier1, base.Add(time.Hour), false, posts.VisibilityActive)
	newPost(reply2ID, 3, replier2, base.Add(2*time.Hour), false, posts.VisibilityActive)
	newPost(reply3ID, 4, replier3, base.Add(3*time.Hour), false, posts.VisibilityActive)
	// 匿名楼层（issue #524）：计入楼层/最后回复指针，但绝不进入参与人表面。
	newPost(anonPostID, 5, anonymousID, base.Add(4*time.Hour), true, posts.VisibilityActive)
	// 治理删除的回复（USER_DELETED，恢复窗口内）：不进入任何派生统计。
	newPost(removedPID, 6, removedID, base.Add(5*time.Hour), false, posts.VisibilityUserDeleted)

	result := BackfillTopicPostStatsWithDB(conn)
	if result.Failed != 0 {
		t.Fatalf("backfill failed=%d last=%s", result.Failed, result.LastFailed)
	}

	after := topics.UnscopedGet(topicID)
	posters := after.Posters
	if len(posters) != 4 {
		t.Fatalf("posters len=%d (%v), want 4 (author + top-3 repliers)", len(posters), posters)
	}
	if posters[0].UserID != authorID {
		t.Fatalf("posters[0]=%d, want author %d first", posters[0].UserID, authorID)
	}
	seen := map[uint64]bool{}
	for _, poster := range posters[1:] {
		seen[poster.UserID] = true
	}
	for _, want := range []uint64{replier1, replier2, replier3} {
		if !seen[want] {
			t.Fatalf("posters missing replier %d: %v", want, posters)
		}
	}
	for _, banned := range []uint64{anonymousID, removedID} {
		if seen[banned] {
			t.Fatalf("posters must exclude user %d: %v", banned, posters)
		}
	}

	if after.PostCount != 5 || after.ReplyCount != 4 {
		t.Fatalf("counts post=%d reply=%d, want 5/4 (removed reply excluded)", after.PostCount, after.ReplyCount)
	}
	if after.LastPostId != anonPostID {
		t.Fatalf("lastPostId=%d, want %d (anonymous reply still tracks last activity)", after.LastPostId, anonPostID)
	}
	if after.LastPostedAt == nil || !after.LastPostedAt.Equal(base.Add(4*time.Hour)) {
		t.Fatalf("lastPostedAt=%v, want %v", after.LastPostedAt, base.Add(4*time.Hour))
	}
	if !after.UpdatedAt.Equal(staleUpdatedAt) {
		t.Fatalf("updated_at perturbed by backfill: %v, want %v", after.UpdatedAt, staleUpdatedAt)
	}

	var statRows []topicUserStat.Entity
	if err := conn.Where("topic_id = ?", topicID).Order("user_id asc").Find(&statRows).Error; err != nil {
		t.Fatalf("read topic_user_stat: %v", err)
	}
	if len(statRows) != 3 {
		t.Fatalf("topic_user_stat rows=%d (%v), want 3 non-anonymous repliers", len(statRows), statRows)
	}
	for _, row := range statRows {
		if row.UserId == anonymousID || row.UserId == removedID || row.UserId == authorID {
			t.Fatalf("topic_user_stat must exclude anonymous/removed/first-post author, got %d", row.UserId)
		}
	}

	// 幂等：第二次执行不再产生修复。
	again := BackfillTopicPostStatsWithDB(conn)
	if again.Failed != 0 {
		t.Fatalf("second backfill failed=%d last=%s", again.Failed, again.LastFailed)
	}
	if again.PostersRepaired != 0 {
		t.Fatalf("second backfill repaired=%d, want 0 (idempotent)", again.PostersRepaired)
	}
}

// 已按新模型口径正确的话题：回填不得计数为修复，也不得改写派生状态。
func TestBackfillTopicPostStatsKeepsCorrectTopics(t *testing.T) {
	conn := db.Connect()
	if err := conn.AutoMigrate(&topics.Entity{}, &posts.Entity{}, &topicUserStat.Entity{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	const (
		topicID   = uint64(9_800_002_201)
		authorID  = uint64(9_800_002_299)
		replierID = uint64(9_800_002_298)
	)
	base := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)

	t.Cleanup(func() {
		conn.Unscoped().Where("id = ?", topicID).Delete(&topics.Entity{})
		conn.Unscoped().Where("topic_id = ?", topicID).Delete(&posts.Entity{})
		conn.Where("topic_id = ?", topicID).Delete(&topicUserStat.Entity{})
	})

	topic := topics.Entity{
		Id:      topicID,
		Title:   "already correct topic",
		UserId:  authorID,
		Status:  1,
		Posters: []topics.Poster{{UserID: authorID}, {UserID: replierID}},
	}
	if err := conn.Create(&topic).Error; err != nil {
		t.Fatalf("create topic: %v", err)
	}
	for _, p := range []posts.Entity{
		{Id: 9_800_002_211, TopicId: topicID, PostNo: 1, UserId: authorID, Content: "first", VisibilityStatus: posts.VisibilityActive, RetentionStatus: posts.RetentionNormal, CreatedAt: base},
		{Id: 9_800_002_212, TopicId: topicID, PostNo: 2, UserId: replierID, Content: "reply", VisibilityStatus: posts.VisibilityActive, RetentionStatus: posts.RetentionNormal, CreatedAt: base.Add(time.Hour)},
	} {
		if err := conn.Create(&p).Error; err != nil {
			t.Fatalf("create post: %v", err)
		}
	}
	before := topics.UnscopedGet(topicID)

	result := BackfillTopicPostStatsWithDB(conn)
	if result.Failed != 0 {
		t.Fatalf("backfill failed=%d last=%s", result.Failed, result.LastFailed)
	}
	if result.PostersRepaired != 0 {
		t.Fatalf("postersRepaired=%d, want 0 for already-correct topics", result.PostersRepaired)
	}
	after := topics.UnscopedGet(topicID)
	if len(after.Posters) != 2 || after.Posters[0].UserID != authorID || after.Posters[1].UserID != replierID {
		t.Fatalf("posters changed for correct topic: %v", after.Posters)
	}
	if !after.UpdatedAt.Equal(before.UpdatedAt) {
		t.Fatalf("updated_at perturbed for correct topic: %v -> %v", before.UpdatedAt, after.UpdatedAt)
	}
	if after.PostCount != 2 || after.ReplyCount != 1 {
		t.Fatalf("counts post=%d reply=%d, want 2/1", after.PostCount, after.ReplyCount)
	}
}
