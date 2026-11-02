package forum

import (
	"strings"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/moderationLog"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/reports"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/rolePermissionRs"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topicUserAction"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/permission"
)

const stickerExcerptToken = "[:sticker:Atari哭哭:]"

// 纯文本预览不展开贴纸（MADR 0030 附录）：收藏/点赞列表的摘要是文本落点，
// token 必须去掉，而不是透传给前端。
func TestProfilePreviewListsStripStickerTokensFromExcerpts(t *testing.T) {
	db := dbconnect.Connect()
	if err := db.AutoMigrate(&users.EntityComplete{}, &topics.Entity{}, &posts.Entity{}); err != nil {
		t.Fatal(err)
	}
	author := users.EntityComplete{Id: 978701, Username: "sticker-preview-author"}
	topic := topics.Entity{Id: 978701, UserId: author.Id, Title: "贴纸话题", Excerpt: "开头 [:sticker:Atari哭哭:] 结尾", Status: 1, FirstPostId: 978705}
	first := posts.Entity{Id: 978705, TopicId: topic.Id, PostNo: 1, UserId: author.Id, Content: "正文"}
	reply := posts.Entity{Id: 978706, TopicId: topic.Id, PostNo: 2, UserId: author.Id, Content: "回复 [:sticker:Atari哭哭:] 正文"}
	for _, row := range []any{&author, &topic, &first, &reply} {
		if err := db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { db.Unscoped().Delete(row) })
	}

	likes := buildUserLikes([]topicUserAction.LikedTopicRef{{ID: 1, TopicID: topic.Id, LikedAt: time.Now()}})
	if len(likes) != 1 || likes[0].Excerpt != "开头 结尾" {
		t.Fatalf("liked topic excerpt = %+v", likes)
	}
	bookmarks := buildUserBookmarks([]topicUserAction.BookmarkedTopicRef{{ID: 1, TopicID: topic.Id, BookmarkedAt: time.Now()}})
	if len(bookmarks) != 1 || bookmarks[0].Excerpt != "开头 结尾" {
		t.Fatalf("bookmarked topic excerpt = %+v", bookmarks)
	}
	merged := buildBookmarkPayloads([]mergedBookmarkRef{
		{kind: "topic", refID: 1, topicID: topic.Id},
		{kind: "post", refID: 2, postID: reply.Id},
	})
	if len(merged) != 2 {
		t.Fatalf("merged bookmark payloads = %+v", merged)
	}
	if merged[0].Excerpt != "开头 结尾" {
		t.Fatalf("merged topic excerpt = %q", merged[0].Excerpt)
	}
	if merged[1].Excerpt != "回复 正文" {
		t.Fatalf("merged reply excerpt = %q", merged[1].Excerpt)
	}
}

// 版主工作台与举报快照共用的摘要截断：先剥 token 再截断，既不透传 token，
// 也不会只截出半个 token。
func TestModerationExcerptStripsStickerTokensBeforeTruncation(t *testing.T) {
	if got := moderationExcerpt("前缀 [:sticker:Atari哭哭:] 尾巴"); got != "前缀 尾巴" {
		t.Fatalf("moderation excerpt = %q", got)
	}
	if kept := moderationExcerpt("没有贴纸的摘要"); kept != "没有贴纸的摘要" {
		t.Fatalf("token-free excerpt changed: %q", kept)
	}
	long := strings.Repeat("字", 118) + " [:sticker:Atari哭哭:]"
	if got := moderationExcerpt(long); got != strings.Repeat("字", 118) {
		t.Fatalf("truncation kept a partial token: %q", got)
	}
}

// 存量举报证据快照（写入早于剥离修复）在读回显处剥除：举报列表、举报处理日志、
// 快捷处置预览都是纯文本落点，历史行必须与新建快照同口径。
func TestReportEvidenceSnapshotReadPathsStripStickerTokens(t *testing.T) {
	conn := setupReportSnapshotTestDB(t)
	const moderatorID, topicID = uint64(979001), uint64(979010)
	createLLMSModerator(t, conn, moderatorID)
	removed := topics.Entity{
		Id: topicID, UserId: moderatorID, Title: "已删贴纸话题", Excerpt: "live 摘要", Status: 1,
		FirstPostId: topicID + 1, VisibilityStatus: topics.VisibilityUserDeleted, RetentionStatus: topics.RetentionRecoverable,
	}
	visibleTopic := topics.Entity{Id: topicID + 1, UserId: moderatorID, Title: "贴纸话题", Excerpt: "live 摘要", Status: 1, FirstPostId: topicID + 2}
	removedPost := posts.Entity{
		Id: topicID + 3, TopicId: visibleTopic.Id, PostNo: 2, UserId: moderatorID, Content: "live 回复正文",
		VisibilityStatus: posts.VisibilityUserDeleted, RetentionStatus: posts.RetentionRecoverable,
	}
	report := reports.Entity{
		TargetType: reports.TargetTopic, TargetId: topicID, ReporterId: moderatorID,
		Reason: reports.ReasonSpam, Status: reports.StatusOpen,
		EvidenceSnapshot: reports.EvidenceSnapshotData{
			TargetType: reports.TargetTopic, TargetID: topicID, Title: "已删贴纸话题",
			Excerpt: "旧快照 " + stickerExcerptToken + " 尾巴", CategoryIDs: []uint64{979020},
		},
	}
	postReport := reports.Entity{
		TargetType: reports.TargetPost, TargetId: removedPost.Id, TopicId: visibleTopic.Id, ReporterId: moderatorID,
		Reason: reports.ReasonAbuse, Status: reports.StatusOpen,
		EvidenceSnapshot: reports.EvidenceSnapshotData{
			TargetType: reports.TargetPost, TargetID: removedPost.Id, Title: "贴纸话题",
			Excerpt: "旧回复快照 " + stickerExcerptToken,
		},
	}
	for _, row := range []any{&removed, &visibleTopic, &removedPost, &report, &postReport} {
		if err := conn.Create(row).Error; err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { conn.Unscoped().Delete(row) })
	}

	item, ok := buildModerationReportItem(moderatorID, 0, report, reportBatchMaps([]reports.Entity{report}))
	if !ok || !item.TargetDeleted || item.Excerpt != "旧快照 尾巴" {
		t.Fatalf("report list excerpt = %+v %v", item, ok)
	}
	if snapshot := buildReportLogSnapshot(report, ""); snapshot.Excerpt != "旧快照 尾巴" {
		t.Fatalf("report log excerpt = %q", snapshot.Excerpt)
	}
	// 回复不可见（作者删除）但话题仍在：审计摘要同样回退举报时刻快照并剥除。
	if snapshot := buildReportLogSnapshot(postReport, ""); snapshot.Excerpt != "旧回复快照" {
		t.Fatalf("post report log excerpt = %q", snapshot.Excerpt)
	}
	if preview := reportQuickActionPreview(report); preview.Excerpt != "旧快照 尾巴" {
		t.Fatalf("quick action excerpt = %q", preview.Excerpt)
	}
}

// 私信举报快照只对管理员展示，读回显同样按纯文本投影剥除。
func TestPrivateMessageReportSnapshotStripsStickerTokensOnRead(t *testing.T) {
	conn := setupReportSnapshotTestDB(t)
	const adminID, roleID = uint64(979101), uint64(979102)
	if err := conn.AutoMigrate(&rolePermissionRs.Entity{}); err != nil {
		t.Fatal(err)
	}
	if err := conn.Create(&users.EntityComplete{Id: adminID, Username: "sticker_report_admin", RoleId: roleID, IsActivated: 1}).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Create(&rolePermissionRs.Entity{RoleId: roleID, PermissionId: permission.Admin.Id(), Effective: 1}).Error; err != nil {
		t.Fatal(err)
	}
	permission.InvalidateRole(roleID)
	t.Cleanup(func() {
		conn.Unscoped().Where("id = ?", adminID).Delete(&users.EntityComplete{})
		conn.Unscoped().Where("role_id = ?", roleID).Delete(&rolePermissionRs.Entity{})
		permission.InvalidateRole(roleID)
	})
	report := reports.Entity{
		TargetType: reports.TargetChatMessage, TargetId: 979110, ReporterId: adminID,
		Reason: reports.ReasonAbuse, Status: reports.StatusOpen,
		EvidenceSnapshot: reports.EvidenceSnapshotData{Excerpt: "私信 " + stickerExcerptToken + " 尾巴", AuthorID: adminID},
	}
	item, ok := buildModerationReportItem(adminID, 0, report, reportBatchMaps([]reports.Entity{report}))
	if !ok || item.Excerpt != "私信 尾巴" {
		t.Fatalf("private message report excerpt = %+v %v", item, ok)
	}
}

// 版主操作日志的 subject 摘要：话题日志读 live 摘要，回复与举报日志读 params["excerpt"]
// （写入端未剥的存量行），三处都是纯文本落点。
func TestModerationLogSubjectStripsStickerTokens(t *testing.T) {
	topic := topics.Entity{Id: 979201, Title: "贴纸话题", Excerpt: "摘要 " + stickerExcerptToken}
	topicSubject := moderationLogSubject(
		moderationLog.Entity{SubjectType: moderationLog.SubjectTopic, SubjectId: topic.Id},
		map[string]any{}, map[uint64]*topics.Entity{topic.Id: &topic},
	)
	if topicSubject.Excerpt != "摘要" {
		t.Fatalf("topic log excerpt = %q", topicSubject.Excerpt)
	}
	postSubject := moderationLogSubject(
		moderationLog.Entity{SubjectType: moderationLog.SubjectPost, SubjectId: 979202},
		map[string]any{"title": "贴纸话题", "topicId": topic.Id, "postNo": uint64(2), "excerpt": "回复 " + stickerExcerptToken + " 尾巴"},
		nil,
	)
	if postSubject.Excerpt != "回复 尾巴" {
		t.Fatalf("post log excerpt = %q", postSubject.Excerpt)
	}
	reportSubject := moderationLogSubject(
		moderationLog.Entity{SubjectType: moderationLog.SubjectReport, SubjectId: 979203},
		map[string]any{"title": "贴纸话题", "postNo": uint64(2), "excerpt": "举报快照 " + stickerExcerptToken + " 尾巴"},
		nil,
	)
	if reportSubject.Excerpt != "举报快照 尾巴" {
		t.Fatalf("report log excerpt = %q", reportSubject.Excerpt)
	}
}

// 草稿列表描述是纯文本 line-clamp；编辑器回填走话题详情，不读 DraftPayload。
func TestDraftPayloadsStripStickerTokensFromDescription(t *testing.T) {
	setupReportSnapshotTestDB(t)
	items := buildDraftPayloads([]*topics.Entity{{Id: 979301, Title: "草稿", Excerpt: "草稿 " + stickerExcerptToken}})
	if len(items) != 1 || items[0].Description != "草稿" || items[0].Title != "草稿" {
		t.Fatalf("draft payloads = %+v", items)
	}
}
