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
func decodeRevisions(t *testing.T, res component.Response) (uint64, []revisionJSON, bool, uint64) {
	t.Helper()
	raw, err := json.Marshal(res.Data.Result)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}
	var payload struct {
		PostId        uint64         `json:"postId"`
		Versions      []revisionJSON `json:"versions"`
		HasMore       bool           `json:"hasMore"`
		BeforeVersion uint64         `json:"beforeVersion"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	return payload.PostId, payload.Versions, payload.HasMore, payload.BeforeVersion
}

// TestPostRevisionsPaginatesByVersionCursor 验证版本历史游标分页：
// 单页大小受 BoundPageSize 钳制（默认 10），beforeVersion 返回更早一页
// （升序），hasMore 标记是否还有更早版本，遍历不漏不重
// （P2：响应不再随版本数无界增长）。
func TestPostRevisionsPaginatesByVersionCursor(t *testing.T) {
	conn := setupRevisionTestDB(t)
	authorID := uint64(9001)
	readerID := uint64(9002)
	createRevisionUser(t, conn, authorID, "author")
	createRevisionUser(t, conn, readerID, "reader")
	versions := make([]postRevisions.Entity, 0, 25)
	for v := 1; v <= 25; v++ {
		versions = append(versions, postRevisions.Entity{
			Version:       uint64(v),
			EditorId:      authorID,
			Content:       fmt.Sprintf("body v%d", v),
			RenderedHTML:  fmt.Sprintf("<p>body v%d</p>", v),
			ProcessStatus: posts.ProcessStatusNormal,
		})
	}
	createRevisionFixture(t, conn, 9101, 9102, authorID, 1, versions)

	// 第一页（默认 page size=10）：最新 10 个版本（升序 v16..v25），
	// hasMore=true，cursor 指向 v15。
	res := PostRevisions(component.BetterRequest[PostRevisionsReq]{
		UserId: readerID,
		Params: PostRevisionsReq{PostID: 9102},
	})
	if res.Data.Code != component.SUCCESS {
		t.Fatalf("PostRevisions(page1) failed: %+v", res)
	}
	_, page1, hasMore, cursor := decodeRevisions(t, res)
	if !hasMore || len(page1) != 10 || page1[0].Version != 16 || page1[9].Version != 25 {
		t.Fatalf("page1 = %d versions (%d..%d) hasMore=%v cursor=%d, want 10 versions v16..v25 hasMore=true cursor=15",
			len(page1), page1[0].Version, page1[len(page1)-1].Version, hasMore, cursor)
	}

	// 第二页：v6..v15。
	res2 := PostRevisions(component.BetterRequest[PostRevisionsReq]{
		UserId: readerID,
		Params: PostRevisionsReq{PostID: 9102, BeforeVersion: cursor},
	})
	if res2.Data.Code != component.SUCCESS {
		t.Fatalf("PostRevisions(page2) failed: %+v", res2)
	}
	_, page2, hasMore2, cursor2 := decodeRevisions(t, res2)
	if !hasMore2 || len(page2) != 10 || page2[0].Version != 6 || page2[9].Version != 15 {
		t.Fatalf("page2 = %d versions (%d..%d) hasMore=%v cursor=%d, want 10 versions v6..v15 hasMore=true cursor=5",
			len(page2), page2[0].Version, page2[len(page2)-1].Version, hasMore2, cursor2)
	}

	// 第三页：v1..v5，hasMore=false。
	res3 := PostRevisions(component.BetterRequest[PostRevisionsReq]{
		UserId: readerID,
		Params: PostRevisionsReq{PostID: 9102, BeforeVersion: cursor2},
	})
	if res3.Data.Code != component.SUCCESS {
		t.Fatalf("PostRevisions(page3) failed: %+v", res3)
	}
	_, page3, hasMore3, _ := decodeRevisions(t, res3)
	if hasMore3 || len(page3) != 5 || page3[0].Version != 1 || page3[4].Version != 5 {
		t.Fatalf("page3 = %d versions (%d..%d) hasMore=%v, want 5 versions v1..v5 hasMore=false",
			len(page3), page3[0].Version, page3[len(page3)-1].Version, hasMore3)
	}
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
	postID, versions, _, _ := decodeRevisions(t, res)
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
	_, readerVersions, _, _ := decodeRevisions(t, res)
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
	_, modVersions, _, _ := decodeRevisions(t, resMod)
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

// TestPostRevisionsBlanksDeletedPostContent 验证软删帖的版本历史正文清空：
// 话题仍可见时，任何读者都不能经版本历史读回已删除回复的原文（review blocker）。
func TestPostRevisionsBlanksDeletedPostContent(t *testing.T) {
	conn := setupRevisionTestDB(t)
	authorID := uint64(8701)
	readerID := uint64(8702)
	createRevisionUser(t, conn, authorID, "author")
	createRevisionUser(t, conn, readerID, "reader")
	createRevisionFixture(t, conn, 8801, 8802, authorID, 1, []postRevisions.Entity{
		{Version: 1, EditorId: authorID, Content: "original body", RenderedHTML: "<p>original body</p>", ProcessStatus: posts.ProcessStatusNormal},
		{Version: 2, EditorId: authorID, Content: "edited body", RenderedHTML: "<p>edited body</p>", ProcessStatus: posts.ProcessStatusNormal},
	})
	if err := conn.Unscoped().Model(&posts.Entity{}).Where("id = ?", 8802).Updates(map[string]any{
		"visibility_status": posts.VisibilityUserDeleted,
		"retention_status":  posts.RetentionRecoverable,
		"deleted_at":        time.Now(),
	}).Error; err != nil {
		t.Fatalf("soft delete post: %v", err)
	}

	res := PostRevisions(component.BetterRequest[PostRevisionsReq]{
		UserId: readerID,
		Params: PostRevisionsReq{PostID: 8802},
	})
	if res.Data.Code != component.SUCCESS {
		t.Fatalf("PostRevisions() failed: %+v", res)
	}
	_, versions, _, _ := decodeRevisions(t, res)
	if len(versions) != 2 {
		t.Fatalf("version count = %d, want 2", len(versions))
	}
	for _, v := range versions {
		if v.Content != "" || v.RenderedHTML != "" {
			t.Fatalf("deleted post revision v%d leaks content = %q / %q", v.Version, v.Content, v.RenderedHTML)
		}
	}
}

// TestPostRevisionsHidesBlockedPostContentFromNonModerator 验证封禁帖的版本正文
// 对非版主屏蔽、版主可见（与楼层窗口的 ProcessStatus 过滤同语义）。
func TestPostRevisionsHidesBlockedPostContentFromNonModerator(t *testing.T) {
	conn := setupRevisionTestDB(t)
	authorID := uint64(8901)
	readerID := uint64(8902)
	moderatorID := uint64(8903)
	createRevisionUser(t, conn, authorID, "author")
	createRevisionUser(t, conn, readerID, "reader")
	createGlobalModerator(t, conn, moderatorID, "moderator")
	createRevisionFixture(t, conn, 9001, 9002, authorID, 1, []postRevisions.Entity{
		{Version: 1, EditorId: authorID, Content: "blocked body", RenderedHTML: "<p>blocked body</p>", ProcessStatus: posts.ProcessStatusNormal},
	})
	if err := conn.Model(&posts.Entity{}).Where("id = ?", 9002).Update("process_status", posts.ProcessStatusBlocked).Error; err != nil {
		t.Fatalf("block post: %v", err)
	}

	res := PostRevisions(component.BetterRequest[PostRevisionsReq]{
		UserId: readerID,
		Params: PostRevisionsReq{PostID: 9002},
	})
	if res.Data.Code != component.SUCCESS {
		t.Fatalf("PostRevisions(reader) failed: %+v", res)
	}
	_, readerVersions, _, _ := decodeRevisions(t, res)
	if len(readerVersions) != 1 || readerVersions[0].Content != "" || readerVersions[0].RenderedHTML != "" {
		t.Fatalf("non-moderator sees blocked content = %#v, want blanked", readerVersions)
	}

	resMod := PostRevisions(component.BetterRequest[PostRevisionsReq]{
		UserId: moderatorID,
		Params: PostRevisionsReq{PostID: 9002},
	})
	if resMod.Data.Code != component.SUCCESS {
		t.Fatalf("PostRevisions(moderator) failed: %+v", resMod)
	}
	_, modVersions, _, _ := decodeRevisions(t, resMod)
	if len(modVersions) != 1 || modVersions[0].Content != "blocked body" {
		t.Fatalf("moderator blocked content = %#v, want visible", modVersions)
	}
}

// TestPostRevisionsHidesPendingPostContentFromNonModerator 验证帖子自身处于
// 待审（pending）状态时，其全部版本正文对非版主屏蔽、版主可见——即使某个
// 版本快照本身是 normal 状态。楼层窗口对 pending 帖整体过滤（不进普通用户流），
// 历史接口必须同语义，否则敏感正文经历史泄露（review P1）。
func TestPostRevisionsHidesPendingPostContentFromNonModerator(t *testing.T) {
	conn := setupRevisionTestDB(t)
	authorID := uint64(9101)
	readerID := uint64(9102)
	moderatorID := uint64(9103)
	createRevisionUser(t, conn, authorID, "author")
	createRevisionUser(t, conn, readerID, "reader")
	createGlobalModerator(t, conn, moderatorID, "moderator")
	createRevisionFixture(t, conn, 9201, 9202, authorID, 1, []postRevisions.Entity{
		{Version: 1, EditorId: authorID, Content: "pending post body", RenderedHTML: "<p>pending post body</p>", ProcessStatus: posts.ProcessStatusNormal},
	})
	if err := conn.Model(&posts.Entity{}).Where("id = ?", 9202).Update("process_status", posts.ProcessStatusPending).Error; err != nil {
		t.Fatalf("mark post pending: %v", err)
	}

	res := PostRevisions(component.BetterRequest[PostRevisionsReq]{
		UserId: readerID,
		Params: PostRevisionsReq{PostID: 9202},
	})
	if res.Data.Code != component.SUCCESS {
		t.Fatalf("PostRevisions(reader) failed: %+v", res)
	}
	_, readerVersions, _, _ := decodeRevisions(t, res)
	if len(readerVersions) != 1 || readerVersions[0].Content != "" || readerVersions[0].RenderedHTML != "" {
		t.Fatalf("non-moderator sees pending-post content = %#v, want blanked", readerVersions)
	}

	resMod := PostRevisions(component.BetterRequest[PostRevisionsReq]{
		UserId: moderatorID,
		Params: PostRevisionsReq{PostID: 9202},
	})
	if resMod.Data.Code != component.SUCCESS {
		t.Fatalf("PostRevisions(moderator) failed: %+v", resMod)
	}
	_, modVersions, _, _ := decodeRevisions(t, resMod)
	if len(modVersions) != 1 || modVersions[0].Content != "pending post body" {
		t.Fatalf("moderator pending-post content = %#v, want visible", modVersions)
	}
}
