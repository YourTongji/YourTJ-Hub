package posts

import (
	"testing"
	"time"

	"github.com/leancodebox/GooseForum/app/bundles/connect/dbconnect"
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

	desc := GetByTopicPostNoDesc(1, 2)
	if len(desc) != 2 || desc[0].PostNo != 3 || desc[1].PostNo != 4 {
		t.Fatalf("GetByTopicPostNoDesc() = %#v, want ascending returned window [3 4]", postNos(desc))
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
