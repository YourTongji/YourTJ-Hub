package api

import (
	"fmt"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/category"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/dailyStats"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/fileUsage"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/moderators"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pointsRecord"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/postRevisions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/postUserAction"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topicCategoryIndex"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topicUserAction"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topicUserStat"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userActivities"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userBadges"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userPoints"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userStatistics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"gorm.io/gorm"
)

func setupTopicWriteTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	conn := dbconnect.Connect()
	err := conn.AutoMigrate(
		&users.EntityComplete{},
		&userStatistics.Entity{},
		&topics.Entity{},
		&postRevisions.Entity{},
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
		&postUserAction.Entity{},
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

// createTopicReply 在指定话题下直接插入一条由 userID 创作的回复（PostNo>1），
// 用于模拟话题变隐藏/封禁前已存在的楼层，供 UpdatePost/DeletePost 可见性守卫测试使用。
func createTopicReply(t *testing.T, conn *gorm.DB, id, topicID, postNo, userID uint64, content string) {
	t.Helper()
	now := time.Now().Add(-time.Hour)
	if err := conn.Create(&posts.Entity{Id: id, TopicId: topicID, PostNo: postNo, UserId: userID, Content: content, ProcessStatus: posts.ProcessStatusNormal, CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatalf("create reply: %v", err)
	}
}

// 用户对曾互动过的话题，话题变隐藏后仍应能取消点赞/收藏/关注，避免
// like_count 与 user_action 行被永久卡住（PR #118 review 修复项）。
func TestUnlikeTopicAfterHiddenAllowed(t *testing.T) {
	conn := setupTopicWriteTestDB(t)
	topicID, _ := visibilityRejectionFixture(t, conn, 1, 1401, 1402, 5100, 6100)
	if res := LikeTopic(component.BetterRequest[LikeTopicReq]{UserId: 1402, Params: LikeTopicReq{TopicId: topicID, Action: 1}}); res.Data.Code != component.SUCCESS {
		t.Fatalf("like visible topic = code=%v", res.Data.Code)
	}
	if got := topics.Get(topicID); got.LikeCount != 1 {
		t.Fatalf("like count = %d, want 1", got.LikeCount)
	}
	if err := conn.Model(&topics.Entity{}).Where("id = ?", topicID).Update("status", 0).Error; err != nil {
		t.Fatalf("hide topic: %v", err)
	}
	res := LikeTopic(component.BetterRequest[LikeTopicReq]{UserId: 1402, Params: LikeTopicReq{TopicId: topicID, Action: 2}})
	if res.Data.Code != component.SUCCESS {
		t.Fatalf("unlike hidden topic = code=%v msg=%v, want SUCCESS", res.Data.Code, res.Data.MessageCode)
	}
	if got := topics.Get(topicID); got.LikeCount != 0 {
		t.Fatalf("like count after unlike = %d, want 0", got.LikeCount)
	}
	if action := topicUserAction.GetByTopicId(1402, topicID); action.LikedAt != nil {
		t.Fatalf("action liked_at not cleared = %#v", action)
	}
}

func TestUnbookmarkTopicAfterHiddenAllowed(t *testing.T) {
	conn := setupTopicWriteTestDB(t)
	topicID, _ := visibilityRejectionFixture(t, conn, 1, 1411, 1412, 5110, 6110)
	if res := BookmarkTopic(component.BetterRequest[BookmarkTopicReq]{UserId: 1412, Params: BookmarkTopicReq{TopicId: topicID, Action: 1}}); res.Data.Code != component.SUCCESS {
		t.Fatalf("bookmark visible topic = code=%v", res.Data.Code)
	}
	if err := conn.Model(&topics.Entity{}).Where("id = ?", topicID).Update("status", 0).Error; err != nil {
		t.Fatalf("hide topic: %v", err)
	}
	res := BookmarkTopic(component.BetterRequest[BookmarkTopicReq]{UserId: 1412, Params: BookmarkTopicReq{TopicId: topicID, Action: 2}})
	if res.Data.Code != component.SUCCESS {
		t.Fatalf("unbookmark hidden topic = code=%v msg=%v, want SUCCESS", res.Data.Code, res.Data.MessageCode)
	}
	if action := topicUserAction.GetByTopicId(1412, topicID); action.BookmarkedAt != nil {
		t.Fatalf("action bookmarked_at not cleared = %#v", action)
	}
}

func TestUnwatchTopicAfterHiddenAllowed(t *testing.T) {
	conn := setupTopicWriteTestDB(t)
	topicID, _ := visibilityRejectionFixture(t, conn, 1, 1421, 1422, 5120, 6120)
	if res := WatchTopic(component.BetterRequest[WatchTopicReq]{UserId: 1422, Params: WatchTopicReq{TopicId: topicID, Action: 1}}); res.Data.Code != component.SUCCESS {
		t.Fatalf("watch visible topic = code=%v", res.Data.Code)
	}
	if err := conn.Model(&topics.Entity{}).Where("id = ?", topicID).Update("status", 0).Error; err != nil {
		t.Fatalf("hide topic: %v", err)
	}
	res := WatchTopic(component.BetterRequest[WatchTopicReq]{UserId: 1422, Params: WatchTopicReq{TopicId: topicID, Action: 2}})
	if res.Data.Code != component.SUCCESS {
		t.Fatalf("unwatch hidden topic = code=%v msg=%v, want SUCCESS", res.Data.Code, res.Data.MessageCode)
	}
	if action := topicUserAction.GetByTopicId(1422, topicID); action.WatchedAt != nil {
		t.Fatalf("action watched_at not cleared = %#v", action)
	}
}

func TestUnlikePostAfterHiddenAllowed(t *testing.T) {
	conn := setupTopicWriteTestDB(t)
	topicID, firstPostID := visibilityRejectionFixture(t, conn, 1, 1431, 1432, 5130, 6130)
	if res := LikePost(component.BetterRequest[LikePostReq]{UserId: 1432, Params: LikePostReq{PostId: firstPostID, Action: 1}}); res.Data.Code != component.SUCCESS {
		t.Fatalf("like post in visible topic = code=%v", res.Data.Code)
	}
	if err := conn.Model(&topics.Entity{}).Where("id = ?", topicID).Update("status", 0).Error; err != nil {
		t.Fatalf("hide topic: %v", err)
	}
	res := LikePost(component.BetterRequest[LikePostReq]{UserId: 1432, Params: LikePostReq{PostId: firstPostID, Action: 2}})
	if res.Data.Code != component.SUCCESS {
		t.Fatalf("unlike post in hidden topic = code=%v msg=%v, want SUCCESS", res.Data.Code, res.Data.MessageCode)
	}
	if action := postUserAction.GetByPostId(1432, firstPostID); action.LikedAt != nil {
		t.Fatalf("post action liked_at not cleared = %#v", action)
	}
}

func TestUnbookmarkPostAfterHiddenAllowed(t *testing.T) {
	conn := setupTopicWriteTestDB(t)
	topicID, firstPostID := visibilityRejectionFixture(t, conn, 1, 1441, 1442, 5140, 6140)
	if res := BookmarkPost(component.BetterRequest[BookmarkPostReq]{UserId: 1442, Params: BookmarkPostReq{PostId: firstPostID, Action: 1}}); res.Data.Code != component.SUCCESS {
		t.Fatalf("bookmark post in visible topic = code=%v", res.Data.Code)
	}
	if err := conn.Model(&topics.Entity{}).Where("id = ?", topicID).Update("status", 0).Error; err != nil {
		t.Fatalf("hide topic: %v", err)
	}
	res := BookmarkPost(component.BetterRequest[BookmarkPostReq]{UserId: 1442, Params: BookmarkPostReq{PostId: firstPostID, Action: 2}})
	if res.Data.Code != component.SUCCESS {
		t.Fatalf("unbookmark post in hidden topic = code=%v msg=%v, want SUCCESS", res.Data.Code, res.Data.MessageCode)
	}
	if action := postUserAction.GetByPostId(1442, firstPostID); action.BookmarkedAt != nil {
		t.Fatalf("post action bookmarked_at not cleared = %#v", action)
	}
}

// 无既有互动状态的用户对隐藏话题执行取消，应同样按不可见拒绝，避免成为话题存在性探针。
func TestUnlikeHiddenTopicWithoutPriorStateRejected(t *testing.T) {
	conn := setupTopicWriteTestDB(t)
	topicID, _ := visibilityRejectionFixture(t, conn, 0, 1451, 1452, 5150, 6150)
	res := LikeTopic(component.BetterRequest[LikeTopicReq]{UserId: 1452, Params: LikeTopicReq{TopicId: topicID, Action: 2}})
	if res.Data.Code != component.FAIL || res.Data.MessageCode != component.MessageTopicNotFound {
		t.Fatalf("unlike hidden topic without state = code=%v msg=%v, want FAIL/MessageTopicNotFound", res.Data.Code, res.Data.MessageCode)
	}
}

// 回复作者对读路径不可见（隐藏/封禁）话题中的楼层，不可编辑/删除（与 LikePost/BookmarkPost 守卫一致）。
func TestUpdatePostRejectsHiddenTopic(t *testing.T) {
	conn := setupTopicWriteTestDB(t)
	topicID, _ := visibilityRejectionFixture(t, conn, 0, 1461, 1462, 5160, 6160)
	createTopicReply(t, conn, 6161, topicID, 2, 1462, "reply before hidden")
	res := UpdatePost(component.BetterRequest[UpdatePostReq]{UserId: 1462, Params: UpdatePostReq{PostId: 6161, Content: "updated content with enough words"}})
	if res.Data.Code != component.FAIL || res.Data.MessageCode != component.MessagePostNotFound {
		t.Fatalf("UpdatePost on hidden topic = code=%v msg=%v, want FAIL/MessagePostNotFound", res.Data.Code, res.Data.MessageCode)
	}
}

func TestDeletePostRejectsHiddenTopic(t *testing.T) {
	conn := setupTopicWriteTestDB(t)
	topicID, _ := visibilityRejectionFixture(t, conn, 0, 1471, 1472, 5170, 6170)
	createTopicReply(t, conn, 6171, topicID, 2, 1472, "reply before hidden")
	res := DeletePost(component.BetterRequest[DeletePostReq]{UserId: 1472, Params: DeletePostReq{PostId: 6171}})
	if res.Data.Code != component.FAIL || res.Data.MessageCode != component.MessagePostNotFound {
		t.Fatalf("DeletePost on hidden topic = code=%v msg=%v, want FAIL/MessagePostNotFound", res.Data.Code, res.Data.MessageCode)
	}
}

func TestUpdatePostRejectsBannedTopic(t *testing.T) {
	conn := setupTopicWriteTestDB(t)
	topicID, _ := visibilityRejectionFixture(t, conn, 1, 1481, 1482, 5180, 6180)
	if err := conn.Model(&topics.Entity{}).Where("id = ?", topicID).Update("process_status", topics.ProcessStatusBlocked).Error; err != nil {
		t.Fatalf("set process_status: %v", err)
	}
	createTopicReply(t, conn, 6181, topicID, 2, 1482, "reply before ban")
	res := UpdatePost(component.BetterRequest[UpdatePostReq]{UserId: 1482, Params: UpdatePostReq{PostId: 6181, Content: "updated content with enough words"}})
	if res.Data.Code != component.FAIL || res.Data.MessageCode != component.MessagePostNotFound {
		t.Fatalf("UpdatePost on banned topic = code=%v msg=%v, want FAIL/MessagePostNotFound", res.Data.Code, res.Data.MessageCode)
	}
}

// 作者对自身隐藏话题中的楼层仍可编辑，与读路径 canViewTopicSimple 的作者放行分支一致。
func TestAuthorCanUpdatePostOnOwnHiddenTopic(t *testing.T) {
	conn := setupTopicWriteTestDB(t)
	topicID, _ := visibilityRejectionFixture(t, conn, 0, 1491, 1492, 5190, 6190)
	createTopicReply(t, conn, 6191, topicID, 2, 1491, "author reply")
	res := UpdatePost(component.BetterRequest[UpdatePostReq]{UserId: 1491, Params: UpdatePostReq{PostId: 6191, Content: "author updated with enough words"}})
	if res.Data.Code != component.SUCCESS {
		t.Fatalf("author UpdatePost own hidden topic = code=%v msg=%v, want SUCCESS", res.Data.Code, res.Data.MessageCode)
	}
	post := posts.Get(6191)
	if post.Content != "author updated with enough words" {
		t.Fatalf("updated post content = %q", post.Content)
	}
}

// review N1：wiki 分站页面话题禁止经论坛编辑端点改写，防止绕过 wiki_page_revisions
// 版本流直接改 topic 行/首楼。即便页面创建者是请求者，也应被拒绝。
func TestWriteTopicRejectsWikiTopicEdit(t *testing.T) {
	conn := setupTopicWriteTestDB(t)
	createTopicWriteUser(t, conn, 1501, "wiki-author")
	now := time.Now().Add(-time.Hour)
	topic := topics.Entity{Id: 5201, Title: "Wiki page", UserId: 1501, Status: 1, TopicType: topics.TopicTypeWiki, PostCount: 1, PostSeq: 1, CreatedAt: now, UpdatedAt: now}
	if err := conn.Create(&topic).Error; err != nil {
		t.Fatalf("create wiki topic: %v", err)
	}
	firstPost := posts.Entity{Id: 6201, TopicId: topic.Id, PostNo: 1, UserId: 1501, Content: "wiki body", ProcessStatus: posts.ProcessStatusNormal, CreatedAt: now, UpdatedAt: now}
	if err := conn.Create(&firstPost).Error; err != nil {
		t.Fatalf("create wiki first post: %v", err)
	}
	if err := conn.Model(&topics.Entity{}).Where("id = ?", topic.Id).Update("first_post_id", firstPost.Id).Error; err != nil {
		t.Fatalf("set first_post_id: %v", err)
	}

	res := WriteTopic(component.BetterRequest[WriteTopicReq]{
		UserId: 1501,
		Params: WriteTopicReq{
			TopicId:     topic.Id,
			Title:       "Hacked title",
			Content:     "hacked body with enough words",
			CategoryId:  []uint64{2001},
			TopicStatus: 1,
		},
	})
	if res.Data.Code != component.FAIL || res.Data.MessageCode != component.MessageTopicOperationDenied {
		t.Fatalf("WriteTopic edit wiki = code=%v msg=%v, want FAIL/MessageTopicOperationDenied", res.Data.Code, res.Data.MessageCode)
	}
	if got := topics.Get(topic.Id); got.Title != "Wiki page" {
		t.Fatalf("wiki topic title mutated = %q", got.Title)
	}
	if got := posts.Get(firstPost.Id); got.Content != "wiki body" {
		t.Fatalf("wiki first post content mutated = %q", got.Content)
	}
}

// review 第三轮（WALKERKILLER）：UpdatePost 的 wiki 拦截必须只挡首楼
// （PostNo<=1），wiki 页面下的评论（post_no>1）是受支持功能，作者仍可编辑
// 自己的回复；同时确认首楼编辑仍被拒绝（不回归第二轮 High 修复）。
func TestUpdatePostWikiReplyEditableFirstPostBlocked(t *testing.T) {
	conn := setupTopicWriteTestDB(t)
	createTopicWriteUser(t, conn, 1601, "wiki-author")
	now := time.Now().Add(-time.Hour)
	topic := topics.Entity{Id: 5301, Title: "Wiki page", UserId: 1601, Status: 1, TopicType: topics.TopicTypeWiki, PostCount: 1, PostSeq: 1, CreatedAt: now, UpdatedAt: now}
	if err := conn.Create(&topic).Error; err != nil {
		t.Fatalf("create wiki topic: %v", err)
	}
	firstPost := posts.Entity{Id: 6301, TopicId: topic.Id, PostNo: 1, UserId: 1601, Content: "wiki body", ProcessStatus: posts.ProcessStatusNormal, CreatedAt: now, UpdatedAt: now}
	if err := conn.Create(&firstPost).Error; err != nil {
		t.Fatalf("create wiki first post: %v", err)
	}
	if err := conn.Model(&topics.Entity{}).Where("id = ?", topic.Id).Update("first_post_id", firstPost.Id).Error; err != nil {
		t.Fatalf("set first_post_id: %v", err)
	}
	createTopicReply(t, conn, 6302, topic.Id, 2, 1601, "wiki comment")

	// wiki 首楼编辑仍被拒绝（版本流独占）。
	res := UpdatePost(component.BetterRequest[UpdatePostReq]{UserId: 1601, Params: UpdatePostReq{PostId: 6301, Content: "hacked first post with enough words"}})
	if res.Data.Code != component.FAIL || res.Data.MessageCode != component.MessageTopicOperationDenied {
		t.Fatalf("UpdatePost wiki first post = code=%v msg=%v, want FAIL/MessageTopicOperationDenied", res.Data.Code, res.Data.MessageCode)
	}
	if got := posts.Get(6301).Content; got != "wiki body" {
		t.Fatalf("wiki first post content mutated = %q", got)
	}

	// wiki 评论（post_no=2）作者仍可编辑自己的回复。
	res = UpdatePost(component.BetterRequest[UpdatePostReq]{UserId: 1601, Params: UpdatePostReq{PostId: 6302, Content: "updated wiki comment with enough words"}})
	if res.Data.Code != component.SUCCESS {
		t.Fatalf("UpdatePost wiki reply = code=%v msg=%v, want SUCCESS", res.Data.Code, res.Data.MessageCode)
	}
	if got := posts.Get(6302).Content; got != "updated wiki comment with enough words" {
		t.Fatalf("wiki reply content = %q", got)
	}
}
