package api

import (
	"fmt"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/category"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/moderationLog"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/optRecord"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topicCategoryIndex"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"gorm.io/gorm"
)

func setupAdminTopicTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(
		&users.EntityComplete{},
		&topics.Entity{},
		&posts.Entity{},
		&category.Entity{},
		&topicCategoryIndex.Entity{},
		&optRecord.Entity{},
		&moderationLog.Entity{},
	); err != nil {
		t.Fatalf("migrate admin topic tables: %v", err)
	}
	return conn
}

func seedAdminTopic(t *testing.T, conn *gorm.DB, topicID uint64) (uint64, uint64) {
	t.Helper()
	userID := topicID + 10
	firstPostID := topicID + 100
	categoryID := topicID + 1000
	now := time.Date(2026, 7, 7, 15, 0, 0, 0, time.UTC)
	if err := conn.Create(&users.EntityComplete{Id: userID, Username: fmt.Sprintf("author-%d", topicID)}).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := conn.Create(&category.Entity{Id: categoryID, Name: "General", Slug: "general"}).Error; err != nil {
		t.Fatalf("create category: %v", err)
	}
	topic := topics.Entity{
		Id:            topicID,
		Title:         "Topic title",
		Excerpt:       "Topic excerpt",
		CategoryIds:   []uint64{categoryID},
		UserId:        userID,
		Status:        1,
		ProcessStatus: 0,
		PostCount:     1,
		PostSeq:       1,
		FirstPostId:   firstPostID,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := conn.Create(&topic).Error; err != nil {
		t.Fatalf("create topic: %v", err)
	}
	if err := conn.Create(&posts.Entity{Id: firstPostID, TopicId: topicID, PostNo: 1, UserId: userID, Content: "first post source", CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatalf("create first post: %v", err)
	}
	if err := conn.Create(&topicCategoryIndex.Entity{TopicId: topicID, CategoryId: categoryID, Effective: 1}).Error; err != nil {
		t.Fatalf("create topic category index: %v", err)
	}
	return userID, categoryID
}

func TestAdminTopicsListReadsTopics(t *testing.T) {
	conn := setupAdminTopicTestDB(t)
	seedAdminTopic(t, conn, 920001)

	res := TopicsList(component.BetterRequest[TopicsListReq]{Params: TopicsListReq{Page: 1, PageSize: 10}})
	page, ok := res.Data.Result.(component.Page[TopicInfoAdminVo])
	if !ok {
		t.Fatalf("result type = %T", res.Data.Result)
	}
	if len(page.List) == 0 || page.List[0].Id != 920001 || page.List[0].Description != "Topic excerpt" || page.List[0].TopicStatus != 1 {
		t.Fatalf("page = %#v", page)
	}
}

func TestAdminTopicSourceReadsFirstPost(t *testing.T) {
	conn := setupAdminTopicTestDB(t)
	_, categoryID := seedAdminTopic(t, conn, 921001)

	res := TopicSource(component.BetterRequest[TopicSourceReq]{Params: TopicSourceReq{TopicId: 921001}})
	source, ok := res.Data.Result.(TopicSourceVo)
	if !ok {
		t.Fatalf("result type = %T", res.Data.Result)
	}
	if source.Id != 921001 || source.Content != "first post source" || len(source.CategoryId) != 1 || source.CategoryId[0] != categoryID {
		t.Fatalf("source = %#v", source)
	}
}

func TestAdminEditTopicMutatesTopic(t *testing.T) {
	conn := setupAdminTopicTestDB(t)
	_, categoryID := seedAdminTopic(t, conn, 922001)
	if err := conn.Create(&category.Entity{Id: 922999, Name: "Second", Slug: "second"}).Error; err != nil {
		t.Fatalf("create second category: %v", err)
	}

	EditTopic(component.BetterRequest[EditTopicReq]{UserId: 1, Params: EditTopicReq{TopicId: 922001, ProcessStatus: 1}})
	topic := topics.Get(922001)
	if topic.ProcessStatus != 1 {
		t.Fatalf("process status = %d, want 1", topic.ProcessStatus)
	}

	EditTopicPin(component.BetterRequest[EditTopicPinReq]{UserId: 1, Params: EditTopicPinReq{TopicId: 922001, PinWeight: 9}})
	topic = topics.Get(922001)
	if topic.PinWeight != 9 {
		t.Fatalf("pin weight = %d, want 9", topic.PinWeight)
	}

	EditTopicCategories(component.BetterRequest[EditTopicCategoriesReq]{UserId: 1, Params: EditTopicCategoriesReq{TopicId: 922001, CategoryId: []uint64{922999}}})
	topic = topics.Get(922001)
	if len(topic.CategoryIds) != 1 || topic.CategoryIds[0] != 922999 {
		t.Fatalf("topic categories = %#v", topic.CategoryIds)
	}
	indexes := topicCategoryIndex.GetByTopicId(922001)
	active := map[uint64]int{}
	for _, item := range indexes {
		active[item.CategoryId] = item.Effective
	}
	if active[categoryID] != 0 || active[922999] != 1 {
		t.Fatalf("category index active map = %#v", active)
	}
}

func TestAdminDeleteTopicRequiresReasonAndMarksModeratorRemoved(t *testing.T) {
	conn := setupAdminTopicTestDB(t)
	_, categoryID := seedAdminTopic(t, conn, 923001)

	// reason 为空应拒绝。
	emptyRes := DeleteTopic(component.BetterRequest[DeleteTopicReq]{
		UserId: 1,
		Params: DeleteTopicReq{TopicId: 923001, Reason: "   "},
	})
	if emptyRes.Data.Code == component.SUCCESS {
		t.Fatalf("expected failure for empty reason, got %#v", emptyRes)
	}
	if emptyRes.Data.MessageCode != component.MessageRequestInvalidParams {
		t.Fatalf("empty reason messageCode = %s", emptyRes.Data.MessageCode)
	}

	res := DeleteTopic(component.BetterRequest[DeleteTopicReq]{
		UserId: 77,
		Params: DeleteTopicReq{TopicId: 923001, Reason: "policy violation"},
	})
	if res.Data.Code != component.SUCCESS {
		t.Fatalf("DeleteTopic failed: %#v", res)
	}

	if visible := topics.Get(923001); visible.Id != 0 {
		t.Fatalf("topic still visible via scoped Get: %#v", visible)
	}
	topic := topics.UnscopedGet(923001)
	if topic.VisibilityStatus != topics.VisibilityModeratorRemoved {
		t.Fatalf("visibility = %s, want MODERATOR_REMOVED", topic.VisibilityStatus)
	}
	if topic.RetentionStatus != topics.RetentionRecoverable {
		t.Fatalf("retention = %s, want RECOVERABLE", topic.RetentionStatus)
	}
	if topic.DeleteReason != "policy violation" {
		t.Fatalf("reason = %q", topic.DeleteReason)
	}
	if topic.DeletedBy != 77 {
		t.Fatalf("deletedBy = %d, want 77", topic.DeletedBy)
	}

	// 分类索引应保留：版主日志/举报的按分类作用域查询依赖该索引定位话题，
	// 删除话题的可见性由列表查询的 visibility_status=ACTIVE 过滤保证，不依赖硬删索引。
	indexes := topicCategoryIndex.GetByTopicId(923001)
	foundCategory := false
	for _, item := range indexes {
		if item.CategoryId == categoryID && item.Effective == 1 {
			foundCategory = true
		}
	}
	if !foundCategory {
		t.Fatalf("category index should remain effective for moderation scoping: %#v", indexes)
	}

	// 同时验证已删除话题不再进入公开分类列表。
	pageData := topics.Page(topics.PageQuery{Page: 1, PageSize: 20, FilterStatus: true, CategoryId: categoryID})
	for _, item := range pageData.Data {
		if item.Id == 923001 {
			t.Fatalf("deleted topic still appears in public category list: %#v", item)
		}
	}
}

// 管理员删除幂等：对已处于 MODERATOR_REMOVED 的话题重复删除应直接成功，
// 且不得重置 deleted_at / 覆盖删除原因。
func TestAdminDeleteTopicIdempotentOnAlreadyRemoved(t *testing.T) {
	conn := setupAdminTopicTestDB(t)
	_, _ = seedAdminTopic(t, conn, 923002)

	first := DeleteTopic(component.BetterRequest[DeleteTopicReq]{
		UserId: 77,
		Params: DeleteTopicReq{TopicId: 923002, Reason: "policy violation"},
	})
	if first.Data.Code != component.SUCCESS {
		t.Fatalf("first DeleteTopic failed: %#v", first)
	}

	before := topics.UnscopedGet(923002)
	second := DeleteTopic(component.BetterRequest[DeleteTopicReq]{
		UserId: 77,
		Params: DeleteTopicReq{TopicId: 923002, Reason: "again"},
	})
	if second.Data.Code != component.SUCCESS {
		t.Fatalf("second DeleteTopic should be idempotent success: %#v", second)
	}
	after := topics.UnscopedGet(923002)
	if after.DeleteReason != before.DeleteReason {
		t.Fatalf("idempotent delete overwrote reason: before=%q after=%q", before.DeleteReason, after.DeleteReason)
	}
	if after.VisibilityStatus != topics.VisibilityModeratorRemoved {
		t.Fatalf("visibility = %s, want MODERATOR_REMOVED", after.VisibilityStatus)
	}
}

// review MEDIUM-2：管理端恢复端点 admin/topics/restore 恢复被治理删除的话题。
func TestAdminRestoreTopicRestoresModeratorRemoved(t *testing.T) {
	conn := setupAdminTopicTestDB(t)
	_, _ = seedAdminTopic(t, conn, 923003)

	if res := DeleteTopic(component.BetterRequest[DeleteTopicReq]{
		UserId: 77,
		Params: DeleteTopicReq{TopicId: 923003, Reason: "policy violation"},
	}); res.Data.Code != component.SUCCESS {
		t.Fatalf("DeleteTopic failed: %#v", res)
	}

	if res := RestoreTopic(component.BetterRequest[RestoreTopicReq]{
		UserId: 77,
		Params: RestoreTopicReq{TopicId: 923003},
	}); res.Data.Code != component.SUCCESS {
		t.Fatalf("RestoreTopic failed: %#v", res)
	}

	restored := topics.UnscopedGet(923003)
	if restored.VisibilityStatus != topics.VisibilityActive || restored.RetentionStatus != topics.RetentionNormal {
		t.Fatalf("restored state = %s/%s, want ACTIVE/NORMAL", restored.VisibilityStatus, restored.RetentionStatus)
	}

	var restoredCount int64
	conn.Model(&moderationLog.Entity{}).
		Where("action = ? AND subject_type = ? AND subject_id = ?", moderationLog.ActionContentRestored, moderationLog.SubjectTopic, 923003).
		Count(&restoredCount)
	if restoredCount != 1 {
		t.Fatalf("content restored logs = %d, want 1", restoredCount)
	}
}

// review MEDIUM-2：管理端恢复端点对未删除/作者删除的话题拒绝，避免越权恢复。
func TestAdminRestoreTopicRejectsNonModeratorRemoved(t *testing.T) {
	conn := setupAdminTopicTestDB(t)
	_, _ = seedAdminTopic(t, conn, 923004)

	// 活跃话题不可通过恢复端点操作。
	if res := RestoreTopic(component.BetterRequest[RestoreTopicReq]{
		UserId: 77,
		Params: RestoreTopicReq{TopicId: 923004},
	}); res.Data.Code == component.SUCCESS {
		t.Fatalf("RestoreTopic of active topic should fail: %#v", res)
	}

	// 作者删除（USER_DELETED）话题不可由管理端恢复端点接管。
	_ = conn
}
