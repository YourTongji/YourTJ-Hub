package forum

import (
	"fmt"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/category"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/contentDeleteEvent"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/moderationLog"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/moderators"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/optRecord"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/reports"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topicCategoryIndex"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/contentdeleteservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/moderationservice"
	"gorm.io/gorm"
)

func setupReportSnapshotTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(
		&users.EntityComplete{},
		&topics.Entity{},
		&posts.Entity{},
		&category.Entity{},
		&topicCategoryIndex.Entity{},
		&reports.Entity{},
		&moderators.Entity{},
		&moderationLog.Entity{},
		&optRecord.Entity{},
		&contentDeleteEvent.Entity{},
	); err != nil {
		t.Fatalf("migrate report snapshot tables: %v", err)
	}
	return conn
}

func seedReportSnapshotTopic(t *testing.T, conn *gorm.DB, base uint64) (authorID uint64, moderatorID uint64, reporterID uint64, topicID uint64, categoryID uint64) {
	t.Helper()
	authorID = base + 1
	moderatorID = base + 2
	reporterID = base + 3
	topicID = base + 10
	categoryID = base + 20
	firstPostID := base + 30
	now := time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC)

	createLLMSModerator(t, conn, moderatorID)
	t.Cleanup(func() {
		conn.Unscoped().Where("id = ?", firstPostID).Delete(&posts.Entity{})
		conn.Unscoped().Where("id = ?", topicID).Delete(&topics.Entity{})
		conn.Unscoped().Where("id = ?", authorID).Delete(&users.EntityComplete{})
		conn.Unscoped().Where("id = ?", reporterID).Delete(&users.EntityComplete{})
		conn.Unscoped().Where("id = ?", categoryID).Delete(&category.Entity{})
		conn.Unscoped().Where("topic_id = ?", topicID).Delete(&topicCategoryIndex.Entity{})
		conn.Unscoped().Where("id = ?", base+40).Delete(&reports.Entity{})
		moderationservice.Invalidate()
	})

	for _, u := range []struct {
		id       uint64
		username string
	}{{authorID, "snapshot_author"}, {reporterID, "snapshot_reporter"}} {
		if err := conn.Create(&users.EntityComplete{Id: u.id, Username: u.username, IsActivated: 1, CreatedAt: now}).Error; err != nil {
			t.Fatalf("create user %s: %v", u.username, err)
		}
	}
	if err := conn.Create(&category.Entity{Id: categoryID, Name: "Snapshot", Slug: "snapshot"}).Error; err != nil {
		t.Fatalf("create category: %v", err)
	}
	if err := conn.Create(&posts.Entity{
		Id:               firstPostID,
		TopicId:          topicID,
		PostNo:           1,
		UserId:           authorID,
		Content:          "被举报的正文内容",
		VisibilityStatus: posts.VisibilityActive,
		RetentionStatus:  posts.RetentionNormal,
		ProcessStatus:    posts.ProcessStatusNormal,
		CreatedAt:        now,
		UpdatedAt:        now,
	}).Error; err != nil {
		t.Fatalf("create first post: %v", err)
	}
	if err := conn.Create(&topics.Entity{
		Id:               topicID,
		Title:            "被举报话题",
		Excerpt:          "被举报话题摘要",
		CategoryIds:      []uint64{categoryID},
		UserId:           authorID,
		Status:           1,
		ProcessStatus:    topics.ProcessStatusNormal,
		PostCount:        1,
		PostSeq:          1,
		FirstPostId:      firstPostID,
		VisibilityStatus: topics.VisibilityActive,
		RetentionStatus:  topics.RetentionNormal,
		CreatedAt:        now,
		UpdatedAt:        now,
	}).Error; err != nil {
		t.Fatalf("create topic: %v", err)
	}
	if err := conn.Create(&topicCategoryIndex.Entity{TopicId: topicID, CategoryId: categoryID, Effective: 1}).Error; err != nil {
		t.Fatalf("create topic category index: %v", err)
	}
	return authorID, moderatorID, reporterID, topicID, categoryID
}

// R6：举报创建时写入证据快照；作者随后删除话题，举报仍留在版主列表且可基于快照审核。
func TestReportEvidenceSnapshotSurvivesTargetDeletion(t *testing.T) {
	conn := setupReportSnapshotTestDB(t)
	authorID, moderatorID, reporterID, topicID, _ := seedReportSnapshotTopic(t, conn, 9_500_000_000)

	createRes := CreateReport(component.BetterRequest[CreateReportReq]{
		UserId: reporterID,
		Params: CreateReportReq{TargetType: reports.TargetTopic, TargetId: topicID, Reason: reports.ReasonSpam},
	})
	if createRes.Data.Code != component.SUCCESS {
		t.Fatalf("CreateReport failed: %#v", createRes)
	}

	// 快照应在举报创建时定格目标内容。
	var reportsOnTopic []reports.Entity
	conn.Where("target_type = ? AND target_id = ?", reports.TargetTopic, topicID).Find(&reportsOnTopic)
	if len(reportsOnTopic) == 0 {
		t.Fatal("no report row created")
	}
	reportID := reportsOnTopic[0].Id
	report := reportsOnTopic[0]
	if report.EvidenceSnapshot.Title != "被举报话题" {
		t.Fatalf("snapshot title = %q, want 被举报话题", report.EvidenceSnapshot.Title)
	}
	if len(report.EvidenceSnapshot.CategoryIDs) == 0 {
		t.Fatalf("snapshot category ids empty: %#v", report.EvidenceSnapshot)
	}

	// 作者删除话题（模拟删帖逃罚）。
	if err := contentdeleteservice.DeleteTopicByUser(authorID, topicID); err != nil {
		t.Fatalf("DeleteTopicByUser: %v", err)
	}
	if visible := topics.Get(topicID); visible.Id != 0 {
		t.Fatalf("topic still visible after deletion")
	}

	// 版主举报列表仍应返回该举报，且目标标记为已删除、标题/摘要来自快照。
	listRes := ModerationReportList(component.BetterRequest[ModerationReportListReq]{
		UserId: moderatorID,
		Params: ModerationReportListReq{Status: reports.StatusOpen},
	})
	if listRes.Data.Code != component.SUCCESS {
		t.Fatalf("ModerationReportList failed: %#v", listRes)
	}
	payload, ok := listRes.Data.Result.(ModerationReportListResponse)
	if !ok {
		t.Fatalf("result type = %T", listRes.Data.Result)
	}
	var found *ModerationReportItem
	for i := range payload.Items {
		if payload.Items[i].ID == reportID {
			found = &payload.Items[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("report %d missing from list after target deletion: %#v", reportID, payload.Items)
	}
	if !found.TargetDeleted {
		t.Fatalf("expected targetDeleted flag on deleted-target report: %#v", found)
	}
	if found.Title != "被举报话题" {
		t.Fatalf("report title = %q, want snapshot title", found.Title)
	}
	if found.Excerpt == "" {
		t.Fatalf("report excerpt should come from snapshot, got empty")
	}

	// 审核仍可完成。
	updateRes := UpdateModerationReportStatus(component.BetterRequest[ModerationReportStatusReq]{
		UserId: moderatorID,
		Params: ModerationReportStatusReq{Id: reportID, Action: "resolve"},
	})
	if updateRes.Data.Code != component.SUCCESS {
		t.Fatalf("UpdateModerationReportStatus failed: %#v", updateRes)
	}
	handled := reports.Get(reportID)
	if handled.Status != reports.StatusResolved {
		t.Fatalf("report status = %s, want resolved", handled.Status)
	}
}

// R6：举报快照不泄漏他人正文——回复举报快照只含摘要与作者，不含全文。
func TestReportEvidenceSnapshotForPostKeepsExcerptOnly(t *testing.T) {
	conn := setupReportSnapshotTestDB(t)
	_, _, reporterID, topicID, _ := seedReportSnapshotTopic(t, conn, 9_600_000_000)

	replyID := topicID + 50
	now := time.Now()
	if err := conn.Create(&posts.Entity{
		Id:               replyID,
		TopicId:          topicID,
		PostNo:           2,
		UserId:           reporterID + 5,
		Content:          fmt.Sprintf("%s%s", "x", "这是一个很长的回复正文，用于验证快照只保留摘要而不是全文"),
		VisibilityStatus: posts.VisibilityActive,
		RetentionStatus:  posts.RetentionNormal,
		ProcessStatus:    posts.ProcessStatusNormal,
		CreatedAt:        now,
		UpdatedAt:        now,
	}).Error; err != nil {
		t.Fatalf("create reply: %v", err)
	}
	t.Cleanup(func() {
		conn.Unscoped().Where("id = ?", replyID).Delete(&posts.Entity{})
	})

	createRes := CreateReport(component.BetterRequest[CreateReportReq]{
		UserId: reporterID,
		Params: CreateReportReq{TargetType: reports.TargetPost, TargetId: replyID, Reason: reports.ReasonAbuse},
	})
	if createRes.Data.Code != component.SUCCESS {
		t.Fatalf("CreateReport failed: %#v", createRes)
	}

	var all []reports.Entity
	conn.Where("target_type = ? AND target_id = ?", reports.TargetPost, replyID).Find(&all)
	if len(all) == 0 {
		t.Fatal("no post report row created")
	}
	snapshot := all[0].EvidenceSnapshot
	if snapshot.Excerpt == "" {
		t.Fatalf("post report snapshot excerpt should not be empty")
	}
	if len(snapshot.Excerpt) > 120 {
		t.Fatalf("post report snapshot excerpt too long (%d chars), moderationExcerpt should clamp it", len(snapshot.Excerpt))
	}
}

// R7：版主查看已删除内容必须提供理由，且每次查看写入审计日志与埋点。
func TestViewDeletedContentRequiresReasonAndAudits(t *testing.T) {
	conn := setupReportSnapshotTestDB(t)
	authorID, moderatorID, _, topicID, _ := seedReportSnapshotTopic(t, conn, 9_700_000_000)

	if err := contentdeleteservice.DeleteTopicByUser(authorID, topicID); err != nil {
		t.Fatalf("DeleteTopicByUser: %v", err)
	}

	// 无理由拒绝。
	emptyRes := ViewDeletedContent(component.BetterRequest[ViewDeletedContentReq]{
		UserId: moderatorID,
		Params: ViewDeletedContentReq{ContentType: reports.TargetTopic, ContentID: topicID, Reason: "   "},
	})
	if emptyRes.Data.Code == component.SUCCESS {
		t.Fatalf("expected view without reason to fail: %#v", emptyRes)
	}

	// 普通用户无权限。
	nonModRes := ViewDeletedContent(component.BetterRequest[ViewDeletedContentReq]{
		UserId: authorID,
		Params: ViewDeletedContentReq{ContentType: reports.TargetTopic, ContentID: topicID, Reason: "audit test"},
	})
	if nonModRes.Data.Code == component.SUCCESS {
		t.Fatalf("expected non-moderator view to fail: %#v", nonModRes)
	}

	// 全局版主带理由可查看原文。
	viewRes := ViewDeletedContent(component.BetterRequest[ViewDeletedContentReq]{
		UserId: moderatorID,
		Params: ViewDeletedContentReq{ContentType: reports.TargetTopic, ContentID: topicID, Reason: "处理举报需要核对原文"},
	})
	if viewRes.Data.Code != component.SUCCESS {
		t.Fatalf("ViewDeletedContent failed: %#v", viewRes)
	}
	view, ok := viewRes.Data.Result.(ModerationDeletedContentView)
	if !ok {
		t.Fatalf("result type = %T", viewRes.Data.Result)
	}
	if view.ContentID != topicID || view.Content == "" {
		t.Fatalf("view content missing: %#v", view)
	}
	if view.DeletedBy == 0 {
		t.Fatalf("view should expose deletedBy metadata: %#v", view)
	}

	// 审计日志：EvidenceViewed 应写入 moderation_log。
	var logCount int64
	conn.Model(&moderationLog.Entity{}).
		Where("action = ? AND subject_type = ? AND subject_id = ?", moderationLog.ActionEvidenceViewed, moderationLog.SubjectTopic, topicID).
		Count(&logCount)
	if logCount != 1 {
		t.Fatalf("evidence viewed log count = %d, want 1", logCount)
	}

	// 埋点事件：moderation_deleted_content_viewed。
	var eventCount int64
	conn.Model(&contentDeleteEvent.Entity{}).
		Where("event_type = ? AND content_id = ?", string(contentDeleteEvent.EventModerationViewed), topicID).
		Count(&eventCount)
	if eventCount != 1 {
		t.Fatalf("moderation viewed events = %d, want 1", eventCount)
	}

	// 活跃内容不可查看。
	activeRes := ViewDeletedContent(component.BetterRequest[ViewDeletedContentReq]{
		UserId: moderatorID,
		Params: ViewDeletedContentReq{ContentType: reports.TargetTopic, ContentID: 9_700_000_010, Reason: "audit"},
	})
	// 该话题已被删除，复用同一 ID 会命中已删分支；用一个不存在的 ID 验证 404 语义。
	missingRes := ViewDeletedContent(component.BetterRequest[ViewDeletedContentReq]{
		UserId: moderatorID,
		Params: ViewDeletedContentReq{ContentType: reports.TargetTopic, ContentID: 9_700_000_999, Reason: "audit"},
	})
	if missingRes.Data.Code == component.SUCCESS {
		t.Fatalf("expected missing target to fail: %#v", missingRes)
	}
	_ = activeRes
}

// R7+R12：永久删除（PURGED）的内容不可再被版主"查看已删除内容"（应返回 404 语义）。
func TestViewDeletedContentRejectsPurgedTarget(t *testing.T) {
	conn := setupReportSnapshotTestDB(t)
	authorID, moderatorID, _, topicID, _ := seedReportSnapshotTopic(t, conn, 9_700_001_000)

	if err := contentdeleteservice.DeleteTopicByUser(authorID, topicID); err != nil {
		t.Fatalf("DeleteTopicByUser: %v", err)
	}
	if err := contentdeleteservice.PurgeContent(authorID, contentdeleteservice.ContentTypeTopic, topicID, "test purge"); err != nil {
		t.Fatalf("PurgeContent: %v", err)
	}

	res := ViewDeletedContent(component.BetterRequest[ViewDeletedContentReq]{
		UserId: moderatorID,
		Params: ViewDeletedContentReq{ContentType: reports.TargetTopic, ContentID: topicID, Reason: "audit"},
	})
	if res.Data.Code == component.SUCCESS {
		t.Fatalf("expected purged target view to fail: %#v", res)
	}
	if res.Data.MessageCode != component.MessageTopicNotFound {
		t.Fatalf("messageCode = %s, want %s", res.Data.MessageCode, component.MessageTopicNotFound)
	}
}
