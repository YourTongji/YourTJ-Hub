package api

import (
	"fmt"
	"testing"
	"time"

	"github.com/leancodebox/GooseForum/app/bundles/connect/dbconnect"
	"github.com/leancodebox/GooseForum/app/http/controllers/component"
	"github.com/leancodebox/GooseForum/app/models/forum/category"
	"github.com/leancodebox/GooseForum/app/models/forum/dailyStats"
	"github.com/leancodebox/GooseForum/app/models/forum/fileUsage"
	"github.com/leancodebox/GooseForum/app/models/forum/moderators"
	"github.com/leancodebox/GooseForum/app/models/forum/pointsRecord"
	"github.com/leancodebox/GooseForum/app/models/forum/posts"
	"github.com/leancodebox/GooseForum/app/models/forum/topicCategoryIndex"
	"github.com/leancodebox/GooseForum/app/models/forum/topicUserAction"
	"github.com/leancodebox/GooseForum/app/models/forum/topicUserStat"
	"github.com/leancodebox/GooseForum/app/models/forum/topics"
	"github.com/leancodebox/GooseForum/app/models/forum/userActivities"
	"github.com/leancodebox/GooseForum/app/models/forum/userBadges"
	"github.com/leancodebox/GooseForum/app/models/forum/userPoints"
	"github.com/leancodebox/GooseForum/app/models/forum/userStatistics"
	"github.com/leancodebox/GooseForum/app/models/forum/users"
	"gorm.io/gorm"
)

func setupTopicWriteTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	conn := dbconnect.Connect()
	err := conn.AutoMigrate(
		&users.EntityComplete{},
		&userStatistics.Entity{},
		&topics.Entity{},
		&posts.Entity{},
		&category.Entity{},
		&topicCategoryIndex.Entity{},
		&topicUserAction.Entity{},
		&topicUserStat.Entity{},
		&fileUsage.Entity{},
		&dailyStats.Entity{},
		&userActivities.Entity{},
		&userPoints.Entity{},
		&pointsRecord.Entity{},
		&userBadges.Entity{},
		&moderators.Entity{},
	)
	if err != nil {
		t.Fatalf("migrate topic write tables: %v", err)
	}
	return conn
}

func createTopicWriteUser(t *testing.T, conn *gorm.DB, id uint64, username string) {
	t.Helper()
	now := time.Now().Add(-time.Hour)
	if err := conn.Create(&users.EntityComplete{Id: id, Username: fmt.Sprintf("%s-%d", username, id), IsActivated: users.ActivationSuccess, CreatedAt: now}).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := conn.Create(&userStatistics.Entity{UserId: id}).Error; err != nil {
		t.Fatalf("create user statistics: %v", err)
	}
}

func TestWriteTopicCreatesTopicAndFirstPost(t *testing.T) {
	conn := setupTopicWriteTestDB(t)
	createTopicWriteUser(t, conn, 1001, "author")
	if err := conn.Create(&category.Entity{Id: 2001, Name: "General", Slug: "general"}).Error; err != nil {
		t.Fatalf("create category: %v", err)
	}

	res := WriteTopic(component.BetterRequest[WriteTopicReq]{
		UserId: 1001,
		Params: WriteTopicReq{
			Title:       "Topic title",
			Content:     "Topic content with enough words",
			CategoryId:  []uint64{2001},
			TopicStatus: 1,
		},
	})
	topicID, ok := res.Data.Result.(uint64)
	if !ok || topicID == 0 {
		t.Fatalf("result = %#v, want topic id", res.Data.Result)
	}

	topic := topics.Get(topicID)
	if topic.Id == 0 || topic.Title != "Topic title" || topic.FirstPostId == 0 || topic.PostCount != 1 || topic.PostSeq != 1 {
		t.Fatalf("topic = %#v", topic)
	}
	firstPost := posts.Get(topic.FirstPostId)
	if firstPost.Id == 0 || firstPost.TopicId != topicID || firstPost.PostNo != 1 || firstPost.Content != "Topic content with enough words" {
		t.Fatalf("first post = %#v", firstPost)
	}
	indexes := topicCategoryIndex.GetByTopicId(topicID)
	if len(indexes) != 1 || indexes[0].CategoryId != 2001 || indexes[0].Effective != 1 {
		t.Fatalf("category indexes = %#v", indexes)
	}
}

func TestCreatePostWritesPostAndTopicStats(t *testing.T) {
	conn := setupTopicWriteTestDB(t)
	createTopicWriteUser(t, conn, 1101, "author")
	createTopicWriteUser(t, conn, 1102, "replyer")
	now := time.Now().Add(-time.Hour)
	topic := topics.Entity{Id: 3001, Title: "Topic", UserId: 1101, Status: 1, PostCount: 1, PostSeq: 1, CreatedAt: now, UpdatedAt: now}
	if err := conn.Create(&topic).Error; err != nil {
		t.Fatalf("create topic: %v", err)
	}
	firstPost := posts.Entity{Id: 3101, TopicId: topic.Id, PostNo: 1, UserId: 1101, Content: "first", CreatedAt: now, UpdatedAt: now}
	if err := conn.Create(&firstPost).Error; err != nil {
		t.Fatalf("create first post: %v", err)
	}
	if err := conn.Model(&topics.Entity{}).Where("id = ?", topic.Id).Update("first_post_id", firstPost.Id).Error; err != nil {
		t.Fatalf("set first post: %v", err)
	}

	res := CreatePost(component.BetterRequest[CreatePostReq]{
		UserId: 1102,
		Params: CreatePostReq{
			TopicId:       topic.Id,
			Content:       "reply content with enough words",
			ReplyToPostId: firstPost.Id,
		},
	})
	payload, ok := res.Data.Result.(map[string]any)
	if !ok {
		t.Fatalf("result = %#v, want payload", res.Data.Result)
	}
	postID, ok := payload["id"].(uint64)
	if !ok || postID == 0 {
		t.Fatalf("reply payload = %#v", payload)
	}
	if got, ok := payload["postNo"].(uint64); !ok || got != 2 {
		t.Fatalf("reply payload postNo = %#v, want 2", payload)
	}
	post := posts.Get(postID)
	if post.Id == 0 || post.TopicId != topic.Id || post.PostNo != 2 || post.ReplyToPostId != firstPost.Id {
		t.Fatalf("post = %#v", post)
	}
	topic = topics.Get(topic.Id)
	if topic.PostCount != 2 || topic.ReplyCount != 1 || topic.PostSeq != 2 {
		t.Fatalf("topic stats = %#v", topic)
	}
}

func TestTopicActionsUseTopicUserAction(t *testing.T) {
	conn := setupTopicWriteTestDB(t)
	createTopicWriteUser(t, conn, 1201, "author")
	createTopicWriteUser(t, conn, 1202, "reader")
	if err := conn.Create(&topics.Entity{Id: 4001, Title: "Topic", UserId: 1201, Status: 1}).Error; err != nil {
		t.Fatalf("create topic: %v", err)
	}

	LikeTopic(component.BetterRequest[LikeTopicReq]{UserId: 1202, Params: LikeTopicReq{TopicId: 4001, Action: 1}})
	BookmarkTopic(component.BetterRequest[BookmarkTopicReq]{UserId: 1202, Params: BookmarkTopicReq{TopicId: 4001, Action: 1}})
	WatchTopic(component.BetterRequest[WatchTopicReq]{UserId: 1202, Params: WatchTopicReq{TopicId: 4001, Action: 1}})

	action := topicUserAction.GetByTopicId(uint64(1202), uint64(4001))
	if action.Id == 0 || action.LikedAt == nil || action.BookmarkedAt == nil || action.WatchedAt == nil {
		t.Fatalf("topic action = %#v", action)
	}
	topic := topics.Get(4001)
	if topic.LikeCount != 1 {
		t.Fatalf("like count = %d, want 1", topic.LikeCount)
	}
}

// visibilityRejectionFixture sets up two users (author + actor) and a topic in
// the requested visibility state. It returns the topic id, its first post id
// (created manually so posts.Get works in the post-level endpoints), and a
// cleanup is via t.Cleanup.
func visibilityRejectionFixture(t *testing.T, conn *gorm.DB, status int8, authorID, actorID, topicID, firstPostID uint64) (uint64, uint64) {
	t.Helper()
	now := time.Now().Add(-time.Hour)
	createTopicWriteUser(t, conn, authorID, "author")
	createTopicWriteUser(t, conn, actorID, "actor")
	topic := topics.Entity{Id: topicID, Title: "hidden", UserId: authorID, Status: status, PostCount: 1, PostSeq: 1, CreatedAt: now, UpdatedAt: now}
	if err := conn.Create(&topic).Error; err != nil {
		t.Fatalf("create topic (status=%d): %v", status, err)
	}
	firstPost := posts.Entity{Id: firstPostID, TopicId: topic.Id, PostNo: 1, UserId: authorID, Content: "first", ProcessStatus: posts.ProcessStatusNormal, CreatedAt: now, UpdatedAt: now}
	if err := conn.Create(&firstPost).Error; err != nil {
		t.Fatalf("create first post: %v", err)
	}
	if err := conn.Model(&topics.Entity{}).Where("id = ?", topic.Id).Update("first_post_id", firstPost.Id).Error; err != nil {
		t.Fatalf("set first_post_id: %v", err)
	}
	return topic.Id, firstPost.Id
}

func TestCreatePostRejectsHiddenTopic(t *testing.T) {
	conn := setupTopicWriteTestDB(t)
	topicID, firstPostID := visibilityRejectionFixture(t, conn, 0, 1311, 1312, 5010, 6000)
	res := CreatePost(component.BetterRequest[CreatePostReq]{
		UserId: 1312,
		Params: CreatePostReq{TopicId: topicID, Content: "reply with enough words", ReplyToPostId: firstPostID},
	})
	if res.Data.Code != component.FAIL || res.Data.MessageCode != component.MessageTopicNotFound {
		t.Fatalf("CreatePost on hidden topic = code=%v msg=%v, want FAIL/MessageTopicNotFound", res.Data.Code, res.Data.MessageCode)
	}
	if got := topics.Get(topicID); got.PostCount != 1 || got.ReplyCount != 0 {
		t.Fatalf("hidden topic stats mutated = %#v", got)
	}
}

func TestCreatePostRejectsBannedTopic(t *testing.T) {
	conn := setupTopicWriteTestDB(t)
	topicID, firstPostID := visibilityRejectionFixture(t, conn, 1, 1331, 1332, 5030, 6020)
	if err := conn.Model(&topics.Entity{}).Where("id = ?", topicID).Update("process_status", topics.ProcessStatusBlocked).Error; err != nil {
		t.Fatalf("set process_status: %v", err)
	}
	res := CreatePost(component.BetterRequest[CreatePostReq]{
		UserId: 1332,
		Params: CreatePostReq{TopicId: topicID, Content: "reply with enough words", ReplyToPostId: firstPostID},
	})
	if res.Data.Code != component.FAIL || res.Data.MessageCode != component.MessageTopicNotFound {
		t.Fatalf("CreatePost on banned topic = code=%v msg=%v, want FAIL/MessageTopicNotFound", res.Data.Code, res.Data.MessageCode)
	}
}

func TestLikeTopicRejectsHiddenTopic(t *testing.T) {
	conn := setupTopicWriteTestDB(t)
	topicID, _ := visibilityRejectionFixture(t, conn, 0, 1341, 1342, 5040, 6030)
	res := LikeTopic(component.BetterRequest[LikeTopicReq]{UserId: 1342, Params: LikeTopicReq{TopicId: topicID, Action: 1}})
	if res.Data.Code != component.FAIL || res.Data.MessageCode != component.MessageTopicNotFound {
		t.Fatalf("LikeTopic on hidden topic = code=%v msg=%v", res.Data.Code, res.Data.MessageCode)
	}
	if got := topics.Get(topicID); got.LikeCount != 0 {
		t.Fatalf("hidden topic like_count = %d, want 0", got.LikeCount)
	}
}

func TestBookmarkTopicRejectsHiddenTopic(t *testing.T) {
	conn := setupTopicWriteTestDB(t)
	topicID, _ := visibilityRejectionFixture(t, conn, 0, 1351, 1352, 5050, 6040)
	res := BookmarkTopic(component.BetterRequest[BookmarkTopicReq]{UserId: 1352, Params: BookmarkTopicReq{TopicId: topicID, Action: 1}})
	if res.Data.Code != component.FAIL || res.Data.MessageCode != component.MessageTopicNotFound {
		t.Fatalf("BookmarkTopic on hidden topic = code=%v msg=%v", res.Data.Code, res.Data.MessageCode)
	}
	if action := topicUserAction.GetByTopicId(1352, topicID); action.Id != 0 {
		t.Fatalf("hidden topic bookmark action persisted = %#v", action)
	}
}

func TestWatchTopicRejectsHiddenTopic(t *testing.T) {
	conn := setupTopicWriteTestDB(t)
	topicID, _ := visibilityRejectionFixture(t, conn, 0, 1361, 1362, 5060, 6050)
	res := WatchTopic(component.BetterRequest[WatchTopicReq]{UserId: 1362, Params: WatchTopicReq{TopicId: topicID, Action: 1}})
	if res.Data.Code != component.FAIL || res.Data.MessageCode != component.MessageTopicNotFound {
		t.Fatalf("WatchTopic on hidden topic = code=%v msg=%v", res.Data.Code, res.Data.MessageCode)
	}
	if action := topicUserAction.GetByTopicId(1362, topicID); action.Id != 0 {
		t.Fatalf("hidden topic watch action persisted = %#v", action)
	}
}

func TestLikePostRejectsHiddenTopic(t *testing.T) {
	conn := setupTopicWriteTestDB(t)
	_, firstPostID := visibilityRejectionFixture(t, conn, 0, 1371, 1372, 5070, 6060)
	res := LikePost(component.BetterRequest[LikePostReq]{UserId: 1372, Params: LikePostReq{PostId: firstPostID, Action: 1}})
	if res.Data.Code != component.FAIL || res.Data.MessageCode != component.MessagePostNotFound {
		t.Fatalf("LikePost on hidden topic = code=%v msg=%v, want FAIL/MessagePostNotFound", res.Data.Code, res.Data.MessageCode)
	}
}

func TestBookmarkPostRejectsHiddenTopic(t *testing.T) {
	conn := setupTopicWriteTestDB(t)
	_, firstPostID := visibilityRejectionFixture(t, conn, 0, 1381, 1382, 5080, 6070)
	res := BookmarkPost(component.BetterRequest[BookmarkPostReq]{UserId: 1382, Params: BookmarkPostReq{PostId: firstPostID, Action: 1}})
	if res.Data.Code != component.FAIL || res.Data.MessageCode != component.MessagePostNotFound {
		t.Fatalf("BookmarkPost on hidden topic = code=%v msg=%v, want FAIL/MessagePostNotFound", res.Data.Code, res.Data.MessageCode)
	}
}

func TestWriteEndpointsAllowAuthorOnOwnHiddenTopic(t *testing.T) {
	conn := setupTopicWriteTestDB(t)
	topicID, firstPostID := visibilityRejectionFixture(t, conn, 0, 1391, 1392, 5090, 6080)
	res := LikeTopic(component.BetterRequest[LikeTopicReq]{UserId: 1391, Params: LikeTopicReq{TopicId: topicID, Action: 1}})
	if res.Data.Code != component.SUCCESS {
		t.Fatalf("author LikeTopic own hidden topic = code=%v, want SUCCESS", res.Data.Code)
	}
	if got := topics.Get(topicID); got.LikeCount != 1 {
		t.Fatalf("author like count = %d, want 1", got.LikeCount)
	}
	res2 := LikePost(component.BetterRequest[LikePostReq]{UserId: 1391, Params: LikePostReq{PostId: firstPostID, Action: 1}})
	if res2.Data.Code != component.SUCCESS {
		t.Fatalf("author LikePost own hidden topic = code=%v, want SUCCESS", res2.Data.Code)
	}
}
