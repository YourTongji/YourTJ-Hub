package posts

import (
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
)

func TestPostRepositoryWindows(t *testing.T) {
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(&Entity{}); err != nil {
		t.Fatalf("migrate posts: %v", err)
	}
	conn.Where("1 = 1").Delete(&Entity{})

	now := time.Date(2026, 7, 7, 12, 0, 0, 0, time.UTC)
	conn.Create(&[]Entity{
		{Id: 10, TopicId: 1, PostNo: 1, UserId: 1, Content: "first", CreatedAt: now},
		{Id: 11, TopicId: 1, PostNo: 2, UserId: 2, Content: "second", CreatedAt: now.Add(time.Minute)},
		{Id: 12, TopicId: 1, PostNo: 3, UserId: 3, Content: "third", CreatedAt: now.Add(2 * time.Minute)},
		{Id: 13, TopicId: 1, PostNo: 4, UserId: 4, Content: "pending", ProcessStatus: ProcessStatusPending, CreatedAt: now.Add(3 * time.Minute)},
		{Id: 14, TopicId: 1, PostNo: 5, UserId: 5, Content: "deleted", CreatedAt: now.Add(4 * time.Minute)},
		{Id: 20, TopicId: 2, PostNo: 1, UserId: 4, Content: "other", CreatedAt: now},
	})
	conn.Delete(&Entity{Id: 14})

	first := GetFirstPageByTopicId(1)
	if len(first) != 4 || first[0].PostNo != 1 || first[3].PostNo != 4 {
		t.Fatalf("GetFirstPageByTopicId() = %#v", postNos(first))
	}


	after := GetByTopicPostNoAfter(1, 1, 10)
	if len(after) != 3 || after[0].PostNo != 2 || after[2].PostNo != 4 {
		t.Fatalf("GetByTopicPostNoAfter() = %#v", postNos(after))
	}

	before := GetByTopicPostNoBefore(1, 3, 10)
	if len(before) != 2 || before[0].PostNo != 1 || before[1].PostNo != 2 {
		t.Fatalf("GetByTopicPostNoBefore() = %#v", postNos(before))
	}

	if got := GetMaxPostNoByTopicId(1); got != 4 {
		t.Fatalf("GetMaxPostNoByTopicId()=%d, want 4", got)
	}

	if err := UpdateProcessStatus(11, 1); err != nil {
		t.Fatalf("UpdateProcessStatus() err=%v", err)
	}
	if got := Get(11); got.ProcessStatus != 1 {
		t.Fatalf("post ProcessStatus=%d, want 1", got.ProcessStatus)
	}

	normal, err := GetNormalByTopicPostNoAfter(1, 0, 10)
	if err != nil {
		t.Fatalf("GetNormalByTopicPostNoAfter() err=%v", err)
	}
	if got := postNos(normal); len(got) != 2 || got[0] != 1 || got[1] != 3 {
		t.Fatalf("GetNormalByTopicPostNoAfter() = %#v, want [1 3]", got)
	}
}

func TestHasChildrenCountsOnlyActiveChildren(t *testing.T) {
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(&Entity{}); err != nil {
		t.Fatalf("migrate posts: %v", err)
	}

	topicID := uint64(time.Now().UnixNano())
	parent := Entity{TopicId: topicID, PostNo: 1, UserId: 1, Content: "parent"}
	child := Entity{TopicId: topicID, PostNo: 2, UserId: 2, ReplyToPostId: 0, Content: "child"}
	if err := conn.Create(&parent).Error; err != nil {
		t.Fatalf("create parent: %v", err)
	}
	child.ReplyToPostId = parent.Id
	if err := conn.Create(&child).Error; err != nil {
		t.Fatalf("create child: %v", err)
	}
	t.Cleanup(func() {
		conn.Unscoped().Where("topic_id = ?", topicID).Delete(&Entity{})
	})

	if !HasChildren(parent.Id) {
		t.Fatal("HasChildren() = false for an active child")
	}
	if err := conn.Delete(&child).Error; err != nil {
		t.Fatalf("soft-delete child: %v", err)
	}
	if HasChildren(parent.Id) {
		t.Fatal("HasChildren() = true after the only child was soft-deleted")
	}
}

func postNos(rows []*Entity) []uint64 {
	res := make([]uint64, 0, len(rows))
	for _, row := range rows {
		res = append(res, row.PostNo)
	}
	return res
}

// review N1 死区修复：wiki 首楼由 wiki 修订审核队列管理，不进入论坛审核队列；
// wiki 分站评论（post_no>1）与论坛话题帖子（无论楼层）仍进入论坛审核队列。
func TestPagePendingReviewIncludesWikiReplies(t *testing.T) {
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(&Entity{}, &topics.Entity{}); err != nil {
		t.Fatalf("migrate pending review tables: %v", err)
	}
	now := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	const (
		wikiTopicID  = uint64(200_001)
		forumTopicID = uint64(200_002)
	)
	topicIDs := []uint64{wikiTopicID, forumTopicID}
	postIDs := []uint64{200_101, 200_102, 200_103, 200_104}
	conn.Unscoped().Where("id IN ?", topicIDs).Delete(&topics.Entity{})
	conn.Unscoped().Where("id IN ?", postIDs).Delete(&Entity{})
	t.Cleanup(func() {
		conn.Unscoped().Where("id IN ?", topicIDs).Delete(&topics.Entity{})
		conn.Unscoped().Where("id IN ?", postIDs).Delete(&Entity{})
	})

	// wiki 话题：首楼 pending（应由 wiki 修订队列管理，排除）、评论 pending（应入队）。
	// 论坛话题：首楼 pending（应入队）、评论 normal（应排除）。
	if err := conn.Create(&[]topics.Entity{
		{Id: wikiTopicID, Title: "wiki", TopicType: topics.TopicTypeWiki, CreatedAt: now, UpdatedAt: now},
		{Id: forumTopicID, Title: "forum", TopicType: topics.TopicTypeForum, CreatedAt: now, UpdatedAt: now},
	}).Error; err != nil {
		t.Fatalf("create topics: %v", err)
	}
	if err := conn.Create(&[]Entity{
		{Id: postIDs[0], TopicId: wikiTopicID, PostNo: 1, UserId: 1, Content: "wiki first", ProcessStatus: ProcessStatusPending, CreatedAt: now},
		{Id: postIDs[1], TopicId: wikiTopicID, PostNo: 2, UserId: 2, Content: "wiki reply", ProcessStatus: ProcessStatusPending, CreatedAt: now},
		{Id: postIDs[2], TopicId: forumTopicID, PostNo: 1, UserId: 3, Content: "forum first", ProcessStatus: ProcessStatusPending, CreatedAt: now},
		{Id: postIDs[3], TopicId: forumTopicID, PostNo: 2, UserId: 4, Content: "forum reply", ProcessStatus: ProcessStatusNormal, CreatedAt: now},
	}).Error; err != nil {
		t.Fatalf("create posts: %v", err)
	}

	page := PagePendingReview(1, 50)
	got := map[uint64]bool{}
	for _, post := range page.Data {
		got[post.Id] = true
	}
	if !got[postIDs[1]] || !got[postIDs[2]] {
		t.Fatalf("PagePendingReview ids = %v, want wiki reply %d and forum first %d included", pendingReviewIDs(page.Data), postIDs[1], postIDs[2])
	}
	if got[postIDs[0]] || got[postIDs[3]] {
		t.Fatalf("PagePendingReview leaked wiki first post or normal-status post: %v", pendingReviewIDs(page.Data))
	}
}

func pendingReviewIDs(rows []Entity) []uint64 {
	res := make([]uint64, 0, len(rows))
	for _, row := range rows {
		res = append(res, row.Id)
	}
	return res
}
