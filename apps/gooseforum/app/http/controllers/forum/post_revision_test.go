package forum

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/moderators"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/postRevisions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/moderationservice"
	"gorm.io/gorm"
)

// revisionJSON 是 PostRevisions 响应版本的 JSON 视图（与 handler 的
// revisionPayload json 标签一致），用于断言响应内容。
type revisionJSON struct {
	Version       uint64         `json:"version"`
	Editor        editorJSON     `json:"editor"`
	Content       string         `json:"content"`
	RenderedHTML  string         `json:"renderedHTML"`
	ProcessStatus int8           `json:"processStatus"`
	CreatedAt     string         `json:"createdAt"`
	WornBadge     map[string]any `json:"wornBadge,omitempty"`
}

type editorJSON struct {
	ID        uint64 `json:"id"`
	Username  string `json:"username"`
	Nickname  string `json:"nickname,omitempty"`
	AvatarURL string `json:"avatarUrl"`
}

func setupRevisionTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(
		&topics.Entity{},
		&posts.Entity{},
		&postRevisions.Entity{},
		&users.EntityComplete{},
		&moderators.Entity{},
	); err != nil {
		t.Fatalf("migrate revision tables: %v", err)
	}
	moderationservice.Invalidate()
	t.Cleanup(func() {
		moderationservice.Invalidate()
	})
	return conn
}

func createRevisionUser(t *testing.T, conn *gorm.DB, id uint64, username string) {
	t.Helper()
	now := time.Now().Add(-time.Hour)
	// 用户名带 ID 后缀，避免共享内存库中跨测试的 uniq_users_username 冲突
	if err := conn.Create(&users.EntityComplete{Id: id, Username: fmt.Sprintf("%s-%d", username, id), IsActivated: users.ActivationSuccess, CreatedAt: now}).Error; err != nil {
		t.Fatalf("create user %d: %v", id, err)
	}
}

func createGlobalModerator(t *testing.T, conn *gorm.DB, id uint64, username string) {
	t.Helper()
	createRevisionUser(t, conn, id, username)
	t.Cleanup(func() {
		conn.Unscoped().Where("user_id = ?", id).Delete(&moderators.Entity{})
		moderationservice.Invalidate()
	})
	if err := conn.Create(&moderators.Entity{UserId: id, ScopeType: moderators.ScopeGlobal, Status: moderators.StatusEnabled}).Error; err != nil {
		t.Fatalf("create moderator grant: %v", err)
	}
	moderationservice.Invalidate()
}

// createRevisionFixture 创建话题 + 首楼 + 版本历史，返回 (topicID, postID)。
// 版本按传入的 versions 逐条插入（Version 需从 1 起连续）。
func createRevisionFixture(t *testing.T, conn *gorm.DB, topicID, postID, authorID uint64, topicStatus int8, versions []postRevisions.Entity) {
	t.Helper()
	now := time.Now().Add(-time.Hour)
	topic := topics.Entity{Id: topicID, Title: "Revision topic", UserId: authorID, Status: topicStatus, PostCount: 1, PostSeq: 1, CreatedAt: now, UpdatedAt: now}
	if err := conn.Create(&topic).Error; err != nil {
		t.Fatalf("create topic: %v", err)
	}
	firstPost := posts.Entity{Id: postID, TopicId: topicID, PostNo: 1, UserId: authorID, Content: "first", ProcessStatus: posts.ProcessStatusNormal, CreatedAt: now, UpdatedAt: now}
	if err := conn.Create(&firstPost).Error; err != nil {
		t.Fatalf("create first post: %v", err)
	}
	if err := conn.Model(&topics.Entity{}).Where("id = ?", topicID).Update("first_post_id", postID).Error; err != nil {
		t.Fatalf("set first_post_id: %v", err)
	}
	for i := range versions {
		versions[i].PostId = postID
		if err := conn.Create(&versions[i]).Error; err != nil {
			t.Fatalf("create revision v%d: %v", versions[i].Version, err)
		}
	}
}

// decodeRevisions 把 PostRevisions 成功响应的 Result 解码为版本列表。
func decodeRevisions(t *testing.T, res component.Response) (uint64, []revisionJSON) {
	t.Helper()
	raw, err := json.Marshal(res.Data.Result)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}
	var payload struct {
		PostId   uint64         `json:"postId"`
		Versions []revisionJSON `json:"versions"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	return payload.PostId, payload.Versions
}

// TestPostRevisionsListsVersionsAscendingWithEditor 验证版本列表按 version 升序，
// v1 editor = 作者、content 与首楼一致（行为 3 的读侧）。
func TestPostRevisionsListsVersionsAscendingWithEditor(t *testing.T) {
	conn := setupRevisionTestDB(t)
	authorID := uint64(8101)
	readerID := uint64(8102)
	createRevisionUser(t, conn, authorID, "author")
	createRevisionUser(t, conn, readerID, "reader")
	createRevisionFixture(t, conn, 8201, 8202, authorID, 1, []postRevisions.Entity{
		{Version: 1, EditorId: authorID, Content: "original body", RenderedHTML: "<p>original body</p>", ProcessStatus: posts.ProcessStatusNormal},
		{Version: 2, EditorId: authorID, Content: "edited body", RenderedHTML: "<p>edited body</p>", ProcessStatus: posts.ProcessStatusNormal},
	})

	res := PostRevisions(component.BetterRequest[PostRevisionsReq]{
		UserId: readerID,
		Params: PostRevisionsReq{PostID: 8202},
	})
	if res.Data.Code != component.SUCCESS {
		t.Fatalf("PostRevisions() failed: %+v", res)
	}
	postID, versions := decodeRevisions(t, res)
	if postID != 8202 {
		t.Fatalf("postId = %d, want 8202", postID)
	}
	if len(versions) != 2 {
		t.Fatalf("version count = %d, want 2", len(versions))
	}
	if versions[0].Version != 1 || versions[1].Version != 2 {
		t.Fatalf("versions not ascending: %d, %d", versions[0].Version, versions[1].Version)
	}
	if versions[0].Editor.ID != authorID || versions[0].Editor.Username != fmt.Sprintf("author-%d", authorID) {
		t.Fatalf("v1 editor = %#v, want author %d", versions[0].Editor, authorID)
	}
	if versions[0].Content != "original body" || versions[0].RenderedHTML != "<p>original body</p>" {
		t.Fatalf("v1 content = %q / %q", versions[0].Content, versions[0].RenderedHTML)
	}
	if versions[1].Content != "edited body" || versions[1].ProcessStatus != posts.ProcessStatusNormal {
		t.Fatalf("v2 = %#v", versions[1])
	}
}

// TestPostRevisionsHidesPendingContentFromNonModerator 验证待审版本正文对非版主
// 屏蔽（content/renderedHTML 为空、状态可见），版主可见完整正文（行为 4）。
func TestPostRevisionsHidesPendingContentFromNonModerator(t *testing.T) {
	conn := setupRevisionTestDB(t)
	authorID := uint64(8301)
	readerID := uint64(8302)
	moderatorID := uint64(8303)
	createRevisionUser(t, conn, authorID, "author")
	createRevisionUser(t, conn, readerID, "reader")
	createGlobalModerator(t, conn, moderatorID, "moderator")
	createRevisionFixture(t, conn, 8401, 8402, authorID, 1, []postRevisions.Entity{
		{Version: 1, EditorId: authorID, Content: "normal body", RenderedHTML: "<p>normal body</p>", ProcessStatus: posts.ProcessStatusNormal},
		{Version: 2, EditorId: authorID, Content: "pending secret body", RenderedHTML: "<p>pending secret body</p>", ProcessStatus: posts.ProcessStatusPending},
	})

	// 非版主：pending 版本正文屏蔽
	res := PostRevisions(component.BetterRequest[PostRevisionsReq]{
		UserId: readerID,
		Params: PostRevisionsReq{PostID: 8402},
	})
	if res.Data.Code != component.SUCCESS {
		t.Fatalf("PostRevisions(reader) failed: %+v", res)
	}
	_, readerVersions := decodeRevisions(t, res)
	if len(readerVersions) != 2 {
		t.Fatalf("reader version count = %d, want 2", len(readerVersions))
	}
	if readerVersions[1].ProcessStatus != posts.ProcessStatusPending {
		t.Fatalf("reader v2 processStatus = %d, want pending", readerVersions[1].ProcessStatus)
	}
	if readerVersions[1].Content != "" || readerVersions[1].RenderedHTML != "" {
		t.Fatalf("reader sees pending content = %q / %q, want blanked", readerVersions[1].Content, readerVersions[1].RenderedHTML)
	}
	if readerVersions[0].Content != "normal body" {
		t.Fatalf("reader v1 content = %q, want visible normal body", readerVersions[0].Content)
	}

	// 版主：pending 版本正文可见
	resMod := PostRevisions(component.BetterRequest[PostRevisionsReq]{
		UserId: moderatorID,
		Params: PostRevisionsReq{PostID: 8402},
	})
	if resMod.Data.Code != component.SUCCESS {
		t.Fatalf("PostRevisions(moderator) failed: %+v", resMod)
	}
	_, modVersions := decodeRevisions(t, resMod)
	if len(modVersions) != 2 {
		t.Fatalf("moderator version count = %d, want 2", len(modVersions))
	}
	if modVersions[1].Content != "pending secret body" || modVersions[1].RenderedHTML != "<p>pending secret body</p>" {
		t.Fatalf("moderator pending content = %q / %q, want visible", modVersions[1].Content, modVersions[1].RenderedHTML)
	}
}

// TestPostRevisionsRejectsInvisibleTopic 验证不可见话题的版本历史返回
// MessagePostNotFound（行为 4）。
func TestPostRevisionsRejectsInvisibleTopic(t *testing.T) {
	conn := setupRevisionTestDB(t)
	authorID := uint64(8501)
	readerID := uint64(8502)
	createRevisionUser(t, conn, authorID, "author")
	createRevisionUser(t, conn, readerID, "reader")
	// topic status=0（草稿/隐藏），普通读者不可见
	createRevisionFixture(t, conn, 8601, 8602, authorID, 0, []postRevisions.Entity{
		{Version: 1, EditorId: authorID, Content: "hidden body", RenderedHTML: "<p>hidden body</p>", ProcessStatus: posts.ProcessStatusNormal},
	})

	res := PostRevisions(component.BetterRequest[PostRevisionsReq]{
		UserId: readerID,
		Params: PostRevisionsReq{PostID: 8602},
	})
	if res.Data.Code != component.FAIL || res.Data.MessageCode != component.MessagePostNotFound {
		t.Fatalf("PostRevisions on hidden topic = code=%v msg=%v, want FAIL/%q", res.Data.Code, res.Data.MessageCode, component.MessagePostNotFound)
	}
}
