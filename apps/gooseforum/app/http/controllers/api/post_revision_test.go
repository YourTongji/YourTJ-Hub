package api

import (
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/category"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/postRevisions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"gorm.io/gorm"
)

// createRevisionTopic 直接插入一条已发布话题及其首楼（PostNo=1），
// 并模拟创建时播种的版本 v1（与 postservice.SeedPostRevision 语义一致），
// 使 UpdatePost 编辑后版本历史为 [v1, v2]。
func createRevisionTopic(t *testing.T, conn *gorm.DB, authorID, topicID, postID uint64, content string) {
	t.Helper()
	now := time.Now().Add(-time.Hour)
	topic := topics.Entity{Id: topicID, Title: "Revision topic", UserId: authorID, Status: 1, PostCount: 1, PostSeq: 1, CreatedAt: now, UpdatedAt: now}
	if err := conn.Create(&topic).Error; err != nil {
		t.Fatalf("create topic: %v", err)
	}
	firstPost := posts.Entity{Id: postID, TopicId: topicID, PostNo: 1, UserId: authorID, Content: content, ProcessStatus: posts.ProcessStatusNormal, CreatedAt: now, UpdatedAt: now}
	if err := conn.Create(&firstPost).Error; err != nil {
		t.Fatalf("create first post: %v", err)
	}
	if err := conn.Model(&topics.Entity{}).Where("id = ?", topicID).Update("first_post_id", postID).Error; err != nil {
		t.Fatalf("set first_post_id: %v", err)
	}
	if err := conn.Create(&postRevisions.Entity{
		PostId:        postID,
		Version:       1,
		EditorId:      authorID,
		Content:       content,
		ProcessStatus: posts.ProcessStatusNormal,
	}).Error; err != nil {
		t.Fatalf("seed revision v1: %v", err)
	}
}

// TestWriteTopicSeedsFirstPostRevisionV1 验证创建话题时在同一事务内播种版本 v1：
// editor = 作者，content 与首楼一致（行为 3 的创建侧）。
func TestWriteTopicSeedsFirstPostRevisionV1(t *testing.T) {
	conn := setupTopicWriteTestDB(t)
	authorID := uint64(7001)
	createTopicWriteUser(t, conn, authorID, "author")
	if err := conn.Create(&category.Entity{Id: 8001, Name: "General", Slug: "general"}).Error; err != nil {
		t.Fatalf("create category: %v", err)
	}

	res := WriteTopic(component.BetterRequest[WriteTopicReq]{
		UserId: authorID,
		Params: WriteTopicReq{
			Title:       "Revision seed topic",
			Content:     "Revision seed content with enough words",
			CategoryId:  []uint64{8001},
			TopicStatus: 1,
		},
	})
	topicID, ok := res.Data.Result.(uint64)
	if !ok || topicID == 0 {
		t.Fatalf("result = %#v, want topic id", res.Data.Result)
	}
	firstPost := posts.Get(topics.Get(topicID).FirstPostId)
	if firstPost.Id == 0 {
		t.Fatalf("first post missing for topic %d", topicID)
	}

	versions := postRevisions.ListByPostId(firstPost.Id)
	if len(versions) != 1 {
		t.Fatalf("revision count after create = %d, want 1", len(versions))
	}
	v1 := versions[0]
	if v1.Version != 1 || v1.EditorId != authorID || v1.Content != firstPost.Content || v1.ProcessStatus != posts.ProcessStatusNormal {
		t.Fatalf("seeded v1 = %#v, want version=1 editor=%d content=%q status=normal", v1, authorID, firstPost.Content)
	}
}

// TestUpdateFirstPostAppendsRevisionAndPersistsLastEditor 验证作者编辑首楼成功：
// 响应携带 lastEditorId/lastEditedAt/revisionCount，posts 行持久化最后编辑者/时间，
// 版本历史追加 v2（行为 1）。
func TestUpdateFirstPostAppendsRevisionAndPersistsLastEditor(t *testing.T) {
	conn := setupTopicWriteTestDB(t)
	authorID := uint64(7101)
	createTopicWriteUser(t, conn, authorID, "author")
	createRevisionTopic(t, conn, authorID, 7201, 7202, "original first post content")

	newContent := "updated first post content with enough words"
	res := UpdatePost(component.BetterRequest[UpdatePostReq]{
		UserId: authorID,
		Params: UpdatePostReq{PostId: 7202, Content: newContent},
	})
	if res.Data.Code != component.SUCCESS {
		t.Fatalf("UpdatePost() failed: %+v", res)
	}
	payload, ok := res.Data.Result.(map[string]any)
	if !ok {
		t.Fatalf("result = %#v, want payload map", res.Data.Result)
	}
	if got, ok := payload["id"].(uint64); !ok || got != 7202 {
		t.Fatalf("payload id = %#v, want 7202", payload["id"])
	}
	if got, ok := payload["postNo"].(uint64); !ok || got != 1 {
		t.Fatalf("payload postNo = %#v, want 1", payload["postNo"])
	}
	if got, ok := payload["content"].(string); !ok || got != newContent {
		t.Fatalf("payload content = %#v, want %q", payload["content"], newContent)
	}
	if got, ok := payload["renderedContent"].(string); !ok || got == "" {
		t.Fatalf("payload renderedContent = %#v, want non-empty", payload["renderedContent"])
	}
	if got, ok := payload["updatedAt"].(string); !ok || got == "" {
		t.Fatalf("payload updatedAt = %#v, want non-empty", payload["updatedAt"])
	}
	if got, ok := payload["lastEditorId"].(uint64); !ok || got != authorID {
		t.Fatalf("payload lastEditorId = %#v, want %d", payload["lastEditorId"], authorID)
	}
	if got, ok := payload["lastEditedAt"].(string); !ok || got == "" {
		t.Fatalf("payload lastEditedAt = %#v, want non-empty", payload["lastEditedAt"])
	}
	if got, ok := payload["revisionCount"].(int64); !ok || got != 2 {
		t.Fatalf("payload revisionCount = %#v, want 2", payload["revisionCount"])
	}

	// posts 行持久化 last_editor_id / last_edited_at
	post := posts.Get(7202)
	if post.LastEditorId != authorID || post.LastEditedAt == nil {
		t.Fatalf("post last editor not persisted = %#v", post)
	}

	// 版本历史：v1（播种）+ v2（本次编辑），按版本号升序
	versions := postRevisions.ListByPostId(7202)
	if len(versions) != 2 {
		t.Fatalf("revision count = %d, want 2", len(versions))
	}
	if versions[0].Version != 1 || versions[0].EditorId != authorID || versions[0].Content != "original first post content" {
		t.Fatalf("revision v1 = %#v", versions[0])
	}
	if versions[1].Version != 2 || versions[1].EditorId != authorID || versions[1].Content != newContent {
		t.Fatalf("revision v2 = %#v", versions[1])
	}
}

// TestUpdateFirstPostRejectsNonAuthor 验证非作者编辑首楼被拒（MessageTopicOperationDenied），
// 且不产生任何写入（帖子内容、最后编辑者/时间与版本历史均不变，行为 2）。
func TestUpdateFirstPostRejectsNonAuthor(t *testing.T) {
	conn := setupTopicWriteTestDB(t)
	authorID := uint64(7301)
	otherID := uint64(7302)
	createTopicWriteUser(t, conn, authorID, "author")
	createTopicWriteUser(t, conn, otherID, "other")
	createRevisionTopic(t, conn, authorID, 7401, 7402, "original content")

	res := UpdatePost(component.BetterRequest[UpdatePostReq]{
		UserId: otherID,
		Params: UpdatePostReq{PostId: 7402, Content: "hijack attempt with enough words"},
	})
	if res.Data.Code != component.FAIL || res.Data.MessageCode != component.MessageTopicOperationDenied {
		t.Fatalf("non-author UpdatePost = code=%v msg=%v, want FAIL/%q", res.Data.Code, res.Data.MessageCode, component.MessageTopicOperationDenied)
	}
	post := posts.Get(7402)
	if post.Content != "original content" || post.LastEditorId != 0 || post.LastEditedAt != nil {
		t.Fatalf("post mutated by non-author edit = %#v", post)
	}
	if versions := postRevisions.ListByPostId(7402); len(versions) != 1 || versions[0].Content != "original content" {
		t.Fatalf("non-author edit changed revisions, got %#v, want unchanged [v1]", versions)
	}
}

// TestWriteTopicSeedsRevisionWithRenderedHTML 验证 writeTopic 创建与编辑分支
// 的版本快照都携带渲染后的 HTML（历史查看依赖 renderedHTML 展示正文）。
func TestWriteTopicSeedsRevisionWithRenderedHTML(t *testing.T) {
	conn := setupTopicWriteTestDB(t)
	authorID := uint64(7501)
	createTopicWriteUser(t, conn, authorID, "author")
	if err := conn.Create(&category.Entity{Id: 8101, Name: "Rendered", Slug: "rendered"}).Error; err != nil {
		t.Fatalf("create category: %v", err)
	}

	res := WriteTopic(component.BetterRequest[WriteTopicReq]{
		UserId: authorID,
		Params: WriteTopicReq{
			Title:       "Rendered revision topic",
			Content:     "Rendered revision content with enough words",
			CategoryId:  []uint64{8101},
			TopicStatus: 1,
		},
	})
	topicID, ok := res.Data.Result.(uint64)
	if !ok || topicID == 0 {
		t.Fatalf("result = %#v, want topic id", res.Data.Result)
	}
	firstPost := posts.Get(topics.Get(topicID).FirstPostId)
	versions := postRevisions.ListByPostId(firstPost.Id)
	if len(versions) != 1 || versions[0].RenderedHTML == "" {
		t.Fatalf("seeded v1 renderedHTML = %q, want non-empty (versions=%d)", versions[0].RenderedHTML, len(versions))
	}

	// 编辑分支（PublishPage 经 /publish?id= 走 WriteTopic）追加的版本同样带 HTML。
	newContent := "Edited rendered content with enough words"
	editRes := WriteTopic(component.BetterRequest[WriteTopicReq]{
		UserId: authorID,
		Params: WriteTopicReq{
			TopicId:     topicID,
			Title:       "Rendered revision topic",
			Content:     newContent,
			CategoryId:  []uint64{8101},
			TopicStatus: 1,
		},
	})
	if editRes.Data.Code != component.SUCCESS {
		t.Fatalf("WriteTopic edit failed: %+v", editRes)
	}
	versions = postRevisions.ListByPostId(firstPost.Id)
	if len(versions) != 2 {
		t.Fatalf("revision count after edit = %d, want 2", len(versions))
	}
	if versions[1].Content != newContent || versions[1].RenderedHTML == "" {
		t.Fatalf("edited revision = %#v, want content %q with non-empty renderedHTML", versions[1], newContent)
	}
}

// TestUpdatePostLazilySeedsV1ForLegacyPost 验证存量帖子（部署前存在、无版本快照）
// 首次编辑时惰性播种 v1 = 旧正文，本次编辑成为 v2，原始正文不丢失。
func TestUpdatePostLazilySeedsV1ForLegacyPost(t *testing.T) {
	conn := setupTopicWriteTestDB(t)
	authorID := uint64(7601)
	createTopicWriteUser(t, conn, authorID, "author")
	// 模拟存量帖子：只建话题 + 首楼，不播种任何版本。
	now := time.Now().Add(-time.Hour)
	legacyTopic := topics.Entity{Id: 8301, Title: "Legacy topic", UserId: authorID, Status: 1, PostCount: 1, PostSeq: 1, CreatedAt: now, UpdatedAt: now}
	if err := conn.Create(&legacyTopic).Error; err != nil {
		t.Fatalf("create legacy topic: %v", err)
	}
	legacyPost := posts.Entity{Id: 8302, TopicId: 8301, PostNo: 1, UserId: authorID, Content: "legacy original body content", ProcessStatus: posts.ProcessStatusNormal, CreatedAt: now, UpdatedAt: now}
	if err := conn.Create(&legacyPost).Error; err != nil {
		t.Fatalf("create legacy first post: %v", err)
	}
	if err := conn.Model(&topics.Entity{}).Where("id = ?", 8301).Update("first_post_id", 8302).Error; err != nil {
		t.Fatalf("set first_post_id: %v", err)
	}

	newContent := "legacy edited body with enough words"
	res := UpdatePost(component.BetterRequest[UpdatePostReq]{
		UserId: authorID,
		Params: UpdatePostReq{PostId: 8302, Content: newContent},
	})
	if res.Data.Code != component.SUCCESS {
		t.Fatalf("UpdatePost() failed: %+v", res)
	}
	versions := postRevisions.ListByPostId(8302)
	if len(versions) != 2 {
		t.Fatalf("revision count after first edit = %d, want 2 (v1=legacy, v2=edit)", len(versions))
	}
	if versions[0].Version != 1 || versions[0].EditorId != authorID || versions[0].Content != "legacy original body content" {
		t.Fatalf("lazy v1 = %#v, want original body preserved", versions[0])
	}
	if versions[1].Version != 2 || versions[1].Content != newContent {
		t.Fatalf("v2 = %#v, want edited body", versions[1])
	}
}

// TestUpdateFirstPostKeepsConcurrentReplyStats 并发回归：首楼编辑只更新
// 摘要/首图/待审等派生字段，不得把并发新建回复刚写入的 post_seq/post_count
// 回写为旧值（否则新回复会撞 post_no 唯一约束或统计倒退）。
func TestUpdateFirstPostKeepsConcurrentReplyStats(t *testing.T) {
	conn := setupTopicWriteTestDB(t)
	authorID := uint64(7701)
	replyerID := uint64(7702)
	createTopicWriteUser(t, conn, authorID, "author")
	createTopicWriteUser(t, conn, replyerID, "replyer")
	createRevisionTopic(t, conn, authorID, 8401, 8402, "original first post content")

	// 模拟并发窗口：编辑请求已读取 topicEntity 快照（PostSeq=1），
	// 此时另一请求创建回复，推进 post_seq=2 / post_count=2。
	staleTopic := topics.GetSimple(8401)
	if err := conn.Create(&posts.Entity{
		TopicId: 8401, PostNo: 2, UserId: replyerID,
		Content:       "concurrent reply with enough words",
		ProcessStatus: posts.ProcessStatusNormal,
		CreatedAt:     time.Now(), UpdatedAt: time.Now(),
	}).Error; err != nil {
		t.Fatalf("create concurrent reply: %v", err)
	}
	if err := conn.Model(&topics.Entity{}).Where("id = ?", 8401).Updates(map[string]any{
		"post_seq": 2, "post_count": 2, "reply_count": 1,
	}).Error; err != nil {
		t.Fatalf("advance topic stats: %v", err)
	}
	if staleTopic.PostSeq != 1 {
		t.Fatalf("precondition failed: stale snapshot PostSeq = %d, want 1", staleTopic.PostSeq)
	}

	res := UpdatePost(component.BetterRequest[UpdatePostReq]{
		UserId: authorID,
		Params: UpdatePostReq{PostId: 8402, Content: "edited first post with enough words"},
	})
	if res.Data.Code != component.SUCCESS {
		t.Fatalf("UpdatePost() failed: %+v", res)
	}
	topic := topics.Get(8401)
	if topic.PostSeq != 2 || topic.PostCount != 2 || topic.ReplyCount != 1 {
		t.Fatalf("concurrent reply stats overwritten by first-post edit: %#v", topic)
	}
	if topic.Excerpt == "" || topic.Excerpt == "original first post content" {
		t.Fatalf("excerpt not updated to edited content: %q", topic.Excerpt)
	}
}
