package api

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/eventbus"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/category"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/moderationLog"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/optRecord"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topicCategoryIndex"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/eventhandlers"
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
		&taskQueue.Entity{},
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

// review N1：wiki 分站页面话题禁止经论坛管理端删除，避免软删话题后残留
// wiki_pages/wiki_page_revisions 孤儿页面。
func TestAdminDeleteTopicRejectsWikiTopic(t *testing.T) {
	conn := setupAdminTopicTestDB(t)
	now := time.Date(2026, 7, 7, 15, 0, 0, 0, time.UTC)
	const topicID = uint64(925001)
	if err := conn.Create(&users.EntityComplete{Id: 925001, Username: "wiki-author"}).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := conn.Create(&posts.Entity{Id: 925101, TopicId: topicID, PostNo: 1, UserId: 925001, Content: "wiki body", CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatalf("create wiki first post: %v", err)
	}
	if err := conn.Create(&topics.Entity{
		Id:            topicID,
		Title:         "Wiki page",
		UserId:        925001,
		Status:        1,
		ProcessStatus: topics.ProcessStatusNormal,
		TopicType:     topics.TopicTypeWiki,
		PostCount:     1,
		PostSeq:       1,
		FirstPostId:   925101,
		CreatedAt:     now,
		UpdatedAt:     now,
	}).Error; err != nil {
		t.Fatalf("create wiki topic: %v", err)
	}

	res := DeleteTopic(component.BetterRequest[DeleteTopicReq]{
		UserId: 77,
		Params: DeleteTopicReq{TopicId: topicID, Reason: "policy violation"},
	})
	if res.Data.Code != component.FAIL || res.Data.MessageCode != component.MessageTopicOperationDenied {
		t.Fatalf("DeleteTopic wiki = code=%v msg=%v, want FAIL/MessageTopicOperationDenied", res.Data.Code, res.Data.MessageCode)
	}
	topic := topics.UnscopedGet(topicID)
	if topic.VisibilityStatus != topics.VisibilityActive {
		t.Fatalf("wiki topic visibility = %s, want ACTIVE", topic.VisibilityStatus)
	}
}

// review N1 死区修复：ReviewAction(kind=post) 仅对 wiki 首楼（post_no<=1）拒绝，
// wiki 分站评论（post_no>1）应放行进论坛审核流程。
func TestReviewActionPostGuardNarrowsWikiToFirstPost(t *testing.T) {
	conn := setupAdminTopicTestDB(t)
	now := time.Date(2026, 7, 7, 15, 0, 0, 0, time.UTC)
	const (
		wikiTopicID  = uint64(926001)
		firstPostID  = uint64(926101)
		replyPostID  = uint64(926102)
		forumTopicID = uint64(926002)
		forumPostID  = uint64(926201)
	)
	for _, topic := range []topics.Entity{
		{Id: wikiTopicID, Title: "wiki", TopicType: topics.TopicTypeWiki, CreatedAt: now, UpdatedAt: now},
		{Id: forumTopicID, Title: "forum", TopicType: topics.TopicTypeForum, CreatedAt: now, UpdatedAt: now},
	} {
		if err := conn.Create(&topic).Error; err != nil {
			t.Fatalf("create topic %d: %v", topic.Id, err)
		}
	}
	for _, post := range []posts.Entity{
		{Id: firstPostID, TopicId: wikiTopicID, PostNo: 1, UserId: 1, Content: "wiki first", ProcessStatus: posts.ProcessStatusNormal, CreatedAt: now},
		{Id: replyPostID, TopicId: wikiTopicID, PostNo: 2, UserId: 2, Content: "wiki reply", ProcessStatus: posts.ProcessStatusNormal, CreatedAt: now},
		{Id: forumPostID, TopicId: forumTopicID, PostNo: 1, UserId: 3, Content: "forum first", ProcessStatus: posts.ProcessStatusNormal, CreatedAt: now},
	} {
		if err := conn.Create(&post).Error; err != nil {
			t.Fatalf("create post %d: %v", post.Id, err)
		}
	}

	// wiki 首楼：即使非 pending 也应被 wiki 修订流程拦截。
	first := ReviewAction(component.BetterRequest[ReviewActionReq]{
		Params: ReviewActionReq{Kind: "post", Id: firstPostID, Approve: true},
	})
	if first.Data.Code != component.FAIL || first.Data.MessageCode != component.MessageAdminReviewTargetInvalid {
		t.Fatalf("review wiki first post = code=%v msg=%v, want FAIL/MessageAdminReviewTargetInvalid", first.Data.Code, first.Data.MessageCode)
	}

	// wiki 评论（post_no>1）：越过 wiki 拦截，进入论坛流程；非 pending 时报"已处理"。
	reply := ReviewAction(component.BetterRequest[ReviewActionReq]{
		Params: ReviewActionReq{Kind: "post", Id: replyPostID, Approve: true},
	})
	if reply.Data.MessageCode == component.MessageAdminReviewTargetInvalid {
		t.Fatalf("review wiki reply was wrongly blocked as targetInvalid: %#v", reply)
	}
	if reply.Data.MessageCode != component.MessageAdminReviewProcessed {
		t.Fatalf("review wiki reply = code=%v msg=%v, want MessageAdminReviewProcessed", reply.Data.Code, reply.Data.MessageCode)
	}

	// 论坛首楼不受 wiki 拦截影响。
	forum := ReviewAction(component.BetterRequest[ReviewActionReq]{
		Params: ReviewActionReq{Kind: "post", Id: forumPostID, Approve: true},
	})
	if forum.Data.MessageCode == component.MessageAdminReviewTargetInvalid {
		t.Fatalf("review forum first post was wrongly blocked: %#v", forum)
	}
}

// review #373-1：管理端审核批准待审回复后补发的 CommentCreatedEvent 必须携带 PostNo
// （楼层号）。若缺失，该路径的通知会退回 #post-{id} 锚点，与「楼层号稳定跳转」目标不一致。
func TestReviewActionApprovedPendingPostEventCarriesPostNo(t *testing.T) {
	conn := setupAdminTopicTestDB(t)
	now := time.Date(2026, 7, 7, 15, 0, 0, 0, time.UTC)
	const (
		topicID       = uint64(926401)
		firstPostID   = uint64(926410)
		pendingPostID = uint64(926411)
		userID        = uint64(926420)
		categoryID    = uint64(926430)
	)
	topic := topics.Entity{
		Id:            topicID,
		Title:         "pending reply topic",
		CategoryIds:   []uint64{categoryID},
		UserId:        userID,
		Status:        1,
		ProcessStatus: 0,
		PostCount:     1,
		PostSeq:       5,
		FirstPostId:   firstPostID,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := conn.Create(&topic).Error; err != nil {
		t.Fatalf("create topic: %v", err)
	}
	for _, post := range []posts.Entity{
		{Id: firstPostID, TopicId: topicID, PostNo: 1, UserId: userID, Content: "first", ProcessStatus: posts.ProcessStatusNormal, CreatedAt: now, UpdatedAt: now},
		{Id: pendingPostID, TopicId: topicID, PostNo: 5, UserId: userID, Content: "pending reply", ProcessStatus: posts.ProcessStatusPending, CreatedAt: now, UpdatedAt: now},
	} {
		if err := conn.Create(&post).Error; err != nil {
			t.Fatalf("create post %d: %v", post.Id, err)
		}
	}

	captured := make(chan *eventhandlers.CommentCreatedEvent, 4)
	handler := cqrs.NewEventHandler("TestReviewActionCommentCapture", func(_ context.Context, event *eventhandlers.CommentCreatedEvent) error {
		// 非阻塞投递：路由器是进程级单例，测试结束后仍会收到其他测试发布的事件，直接丢弃。
		select {
		case captured <- event:
		default:
		}
		return nil
	})
	startReviewActionEventBus(t, handler, captured)

	res := ReviewAction(component.BetterRequest[ReviewActionReq]{
		Params: ReviewActionReq{Kind: "post", Id: pendingPostID, Approve: true},
	})
	if res.Data.Code != component.SUCCESS {
		t.Fatalf("ReviewAction approve = code=%v msg=%v, want SUCCESS", res.Data.Code, res.Data.MessageCode)
	}

	select {
	case event := <-captured:
		if event.PostNo != 5 {
			t.Fatalf("CommentCreatedEvent.PostNo = %d, want 5", event.PostNo)
		}
		if event.PostId != pendingPostID || event.TopicId != topicID {
			t.Fatalf("CommentCreatedEvent = %#v, want PostId=%d TopicId=%d", event, pendingPostID, topicID)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for CommentCreatedEvent published by ReviewAction approval")
	}
}

// startReviewActionEventBus 启动事件总线路由器并注册捕获 handler，然后通过哨兵事件
// 回环确认路由器已订阅 CommentCreatedEvent 主题（发布早于订阅的事件会被丢弃）。
func startReviewActionEventBus(t *testing.T, handler cqrs.EventHandler, captured chan *eventhandlers.CommentCreatedEvent) {
	t.Helper()
	eventbus.Start(handler)
	const sentinel = uint64(0xDEADBEEF)
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		eventbus.Publish(context.Background(), &eventhandlers.CommentCreatedEvent{PostNo: sentinel})
		select {
		case event := <-captured:
			if event.PostNo != sentinel {
				t.Fatalf("unexpected event during readiness probe: %#v", event)
			}
			return
		case <-time.After(300 * time.Millisecond):
			// 路由器尚未就绪，重试探针。
		}
	}
	t.Fatal("event bus router did not become ready")
}
