package contentdeleteservice

import (
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
)

// 回归 M2：DeleteAllUserContent 应终止并删除该用户全部话题与回复
// （含他人话题下的本人回复），不会因游标恒为 0 而无限循环。
func TestDeleteAllUserContentDeletesOwnContent(t *testing.T) {
	conn := setupContentDeleteTestDB(t)
	const authorID = uint64(947099)
	if err := conn.Create(&users.EntityComplete{Id: authorID, Username: "closing-user"}).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	now := time.Now()

	seed := func(topicID, firstPostID uint64) {
		if err := conn.Create(&topics.Entity{
			Id: topicID, Title: "t", UserId: authorID, Status: 1, PostSeq: 1, FirstPostId: firstPostID,
			VisibilityStatus: topics.VisibilityActive, RetentionStatus: topics.RetentionNormal,
			CreatedAt: now, UpdatedAt: now,
		}).Error; err != nil {
			t.Fatalf("create topic %d: %v", topicID, err)
		}
		if err := conn.Create(&posts.Entity{
			Id: firstPostID, TopicId: topicID, PostNo: 1, UserId: authorID, Content: "first",
			VisibilityStatus: posts.VisibilityActive, RetentionStatus: posts.RetentionNormal,
			CreatedAt: now, UpdatedAt: now,
		}).Error; err != nil {
			t.Fatalf("create first post %d: %v", firstPostID, err)
		}
	}
	// 作者 2 个话题
	seed(947100, 947101)
	seed(947200, 947201)
	// 他人话题（作者在该话题下有一条回复）
	if err := conn.Create(&topics.Entity{
		Id: 947300, Title: "other", UserId: 947999, Status: 1, PostSeq: 2, FirstPostId: 947301,
		VisibilityStatus: topics.VisibilityActive, RetentionStatus: topics.RetentionNormal,
		CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatalf("create other topic: %v", err)
	}
	if err := conn.Create(&posts.Entity{
		Id: 947301, TopicId: 947300, PostNo: 1, UserId: 947999, Content: "of",
		VisibilityStatus: posts.VisibilityActive, RetentionStatus: posts.RetentionNormal,
		CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatalf("create other first: %v", err)
	}
	if err := conn.Create(&posts.Entity{
		Id: 947302, TopicId: 947300, PostNo: 2, UserId: authorID, Content: "author reply in other topic",
		VisibilityStatus: posts.VisibilityActive, RetentionStatus: posts.RetentionNormal,
		CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatalf("create author reply: %v", err)
	}

	if err := DeleteAllUserContent(authorID); err != nil {
		t.Fatalf("DeleteAllUserContent: %v", err)
	}

	// 作者话题全部进入 USER_DELETED
	for _, topicID := range []uint64{947100, 947200} {
		if tpc := topics.UnscopedGet(topicID); tpc.VisibilityStatus != topics.VisibilityUserDeleted {
			t.Fatalf("topic %d vis = %s, want USER_DELETED", topicID, tpc.VisibilityStatus)
		}
	}
	// 作者在他话题下的回复进入 USER_DELETED
	reply := posts.UnscopedGet(947302)
	if reply.VisibilityStatus != posts.VisibilityUserDeleted {
		t.Fatalf("author reply vis = %s, want USER_DELETED", reply.VisibilityStatus)
	}
	// 他人内容不受影响
	otherTopic := topics.UnscopedGet(947300)
	if otherTopic.VisibilityStatus != topics.VisibilityActive {
		t.Fatalf("other topic changed: %s", otherTopic.VisibilityStatus)
	}
	if otherFirst := posts.UnscopedGet(947301); otherFirst.VisibilityStatus != posts.VisibilityActive {
		t.Fatalf("other first post changed: %s", otherFirst.VisibilityStatus)
	}
}
