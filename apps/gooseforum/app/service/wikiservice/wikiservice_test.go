package wikiservice

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/rolePermissionRs"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topicUserAction"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiNamespaceEditors"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiNamespaces"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiPageRevisions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiPages"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/permission"
)

// setupWikiTestDB 迁移 wiki 相关表并清空。
func setupWikiTestDB(t *testing.T) {
	t.Helper()
	conn := dbconnect.Connect()
	models := []any{
		&topics.Entity{},
		&posts.Entity{},
		&users.EntityComplete{},
		&rolePermissionRs.Entity{},
		&topicUserAction.Entity{},
		&wikiNamespaces.Entity{},
		&wikiNamespaceEditors.Entity{},
		&wikiPages.Entity{},
		&wikiPageRevisions.Entity{},
	}
	if err := conn.AutoMigrate(models...); err != nil {
		t.Fatalf("migrate wiki schema: %v", err)
	}
	for _, model := range models {
		if err := conn.Unscoped().Where("1 = 1").Delete(model).Error; err != nil {
			t.Fatalf("clean wiki table: %v", err)
		}
	}
}

// wikiTestUserSeq 每次测试递增，保证用户 ID 唯一，避免 userservice 缓存串扰。
var wikiTestUserSeq uint64 = 1000

// seedWikiUser 创建测试用户；manager=true 时授予 PageManager（角色 ID 每次唯一）。
func seedWikiUser(t *testing.T, manager bool) uint64 {
	t.Helper()
	wikiTestUserSeq++
	id := wikiTestUserSeq
	user := &users.EntityComplete{
		Id:          id,
		Username:    fmt.Sprintf("wikiuser%d", id),
		Email:       fmt.Sprintf("wikiuser%d@example.test", id),
		IsActivated: users.ActivationSuccess,
	}
	if err := dbconnect.Connect().Create(user).Error; err != nil {
		t.Fatalf("create wiki test user: %v", err)
	}
	if !manager {
		return id
	}
	roleID := uint64(time.Now().UnixNano())
	if err := dbconnect.Connect().Model(&users.EntityComplete{}).Where("id = ?", id).Update("role_id", roleID).Error; err != nil {
		t.Fatalf("set role for wiki test user: %v", err)
	}
	if err := dbconnect.Connect().Create(&rolePermissionRs.Entity{RoleId: roleID, PermissionId: permission.PageManager.Id()}).Error; err != nil {
		t.Fatalf("grant PageManager to wiki test user: %v", err)
	}
	return id
}

func TestValidatePath(t *testing.T) {
	cases := []struct {
		input string
		ok    bool
	}{
		{"guide/getting-started", true},
		{"deployment/waline", true},
		{"guide/sub/page-name", true},
		{"guide", false},                // 至少 namespace + 一个 slug 段
		{"Guide/Getting-Started", true}, // 规范化小写
		{"guide/..", false},             // 禁止 ..
		{"guide/a b", false},            // 空格非法
		{"guide/UPPER", true},           // 小写规范化后合法
		{"", false},
	}
	for _, tc := range cases {
		got, ok := ValidatePath(tc.input)
		if ok != tc.ok {
			t.Fatalf("ValidatePath(%q) ok=%v, want %v", tc.input, ok, tc.ok)
		}
		if ok && got != tc.input && got != lower(tc.input) {
			t.Fatalf("ValidatePath(%q) normalized=%q", tc.input, got)
		}
	}
}

func lower(s string) string {
	b := []byte(s)
	for i := range b {
		if b[i] >= 'A' && b[i] <= 'Z' {
			b[i] += 'a' - 'A'
		}
	}
	return string(b)
}

func TestValidateNamespace(t *testing.T) {
	cases := []struct {
		input string
		ok    bool
	}{
		{"guide", true},
		{"deployment", true},
		{"my-namespace", true},
		{"Guide", true}, // 小写
		{"has space", false},
		{"", false},
		{"UPPER", true},
	}
	for _, tc := range cases {
		if got := ValidateNamespace(tc.input); got != tc.ok {
			t.Fatalf("ValidateNamespace(%q) = %v, want %v", tc.input, got, tc.ok)
		}
	}
}

// TestCreateEditApproveFlow 覆盖创建（直发）→ 编辑（pending）→ 审核（approve）全链路。
func TestCreateEditApproveFlow(t *testing.T) {
	setupWikiTestDB(t)
	editor := seedWikiUser(t, true) // 创建者 + 审核者（PageManager）

	// 创建 namespace，并把 userId=1 设为贡献者。
	if err := wikiNamespaces.Create(&wikiNamespaces.Entity{Name: "guide", Description: "指南"}); err != nil {
		t.Fatalf("create namespace: %v", err)
	}
	if err := wikiNamespaceEditors.SetEditorsTx(dbconnect.Connect(), "guide", []uint64{editor}, editor); err != nil {
		t.Fatalf("set editors: %v", err)
	}

	// 创建页面（直接发布，approved revision#1）。
	created, err := Create(CreateParams{
		Namespace: "guide",
		Path:      "guide/getting-started",
		Title:     "快速开始",
		Content:   "# 标题一\n\n正文内容",
		UserId:    editor,
	})
	if err != nil {
		t.Fatalf("create wiki page: %v", err)
	}
	if created.PageId == 0 || created.Path != "guide/getting-started" {
		t.Fatalf("create result=%+v", created)
	}

	page := wikiPages.Get(created.PageId)
	if page.TopicId == 0 {
		t.Fatal("page topic_id not set")
	}
	topic := topics.Get(page.TopicId)
	if topic.TopicType != topics.TopicTypeWiki {
		t.Fatalf("topic_type=%d, want wiki", topic.TopicType)
	}
	rev := wikiPageRevisions.GetLatestApproved(page.Id)
	if rev.Id == 0 || rev.Status != wikiPageRevisions.StatusApproved {
		t.Fatalf("revision#1 not approved: %+v", rev)
	}

	// 编辑 → pending。
	edited, err := Edit(EditParams{PageID: page.Id, Title: "快速开始 v2", Content: "# 新标题\n\n新内容", UserId: editor})
	if err != nil {
		t.Fatalf("edit wiki page: %v", err)
	}
	if edited.Status != StatusStringPending {
		t.Fatalf("edit status=%q, want pending", edited.Status)
	}
	// 公开视图仍显示旧内容（latest approved）。
	detail, err := LoadPageDetail(&page, &topic)
	if err != nil {
		t.Fatalf("load detail: %v", err)
	}
	if detail.Title != "快速开始" {
		t.Fatalf("public title=%q, want old title", detail.Title)
	}
	pending := LoadPending(page.Id)
	if pending == nil || pending.Title != "快速开始 v2" {
		t.Fatalf("pending not visible to editors: %+v", pending)
	}

	// approve → 公开可见新内容。
	if _, err := Review(ReviewParams{RevisionID: edited.RevisionId, Action: "approve", UserId: editor}); err != nil {
		t.Fatalf("approve: %v", err)
	}
	topic = topics.Get(page.TopicId)
	if topic.Title != "快速开始 v2" {
		t.Fatalf("topic title after approve=%q", topic.Title)
	}
	firstPost := posts.Get(topic.FirstPostId)
	if firstPost.ProcessStatus != posts.ProcessStatusNormal {
		t.Fatalf("first post process_status=%d, want normal", firstPost.ProcessStatus)
	}
	detail, _ = LoadPageDetail(&page, &topic)
	if detail.Title != "快速开始 v2" {
		t.Fatalf("public title after approve=%q", detail.Title)
	}
}

// TestEditSupersedeInvariant 每页至多一条 pending；新编辑 supersede 旧 pending。
func TestEditSupersedeInvariant(t *testing.T) {
	setupWikiTestDB(t)
	editor := seedWikiUser(t, false)

	if err := wikiNamespaces.Create(&wikiNamespaces.Entity{Name: "deploy", Description: "部署"}); err != nil {
		t.Fatalf("create namespace: %v", err)
	}
	if err := wikiNamespaceEditors.SetEditorsTx(dbconnect.Connect(), "deploy", []uint64{editor}, editor); err != nil {
		t.Fatalf("set editors: %v", err)
	}
	created, err := Create(CreateParams{Namespace: "deploy", Path: "deploy/waline", Title: "Waline", Content: "内容", UserId: editor})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	page := wikiPages.Get(created.PageId)

	if _, err := Edit(EditParams{PageID: page.Id, Title: "v1", Content: "c1", UserId: editor}); err != nil {
		t.Fatalf("edit1: %v", err)
	}
	if _, err := Edit(EditParams{PageID: page.Id, Title: "v2", Content: "c2", UserId: editor}); err != nil {
		t.Fatalf("edit2: %v", err)
	}

	revisions := wikiPageRevisions.ListByPage(page.Id)
	pendingCount := 0
	supersededCount := 0
	for _, r := range revisions {
		switch r.Status {
		case wikiPageRevisions.StatusPending:
			pendingCount++
		case wikiPageRevisions.StatusSuperseded:
			supersededCount++
		}
	}
	if pendingCount != 1 {
		t.Fatalf("pending count=%d, want 1", pendingCount)
	}
	if supersededCount != 1 {
		t.Fatalf("superseded count=%d, want 1", supersededCount)
	}
	// revision_no 单调递增（approved#1 + superseded#2 + pending#3）。
	if len(revisions) != 3 {
		t.Fatalf("revision count=%d, want 3", len(revisions))
	}
	if revisions[0].RevisionNo != 3 || revisions[1].RevisionNo != 2 || revisions[2].RevisionNo != 1 {
		t.Fatalf("revision_no order=%d,%d,%d", revisions[0].RevisionNo, revisions[1].RevisionNo, revisions[2].RevisionNo)
	}
}

// TestReviewRejectRollback reject 回滚为上一 approved 内容。
func TestReviewRejectRollback(t *testing.T) {
	setupWikiTestDB(t)
	reviewer := seedWikiUser(t, true) // 审核者（PageManager）

	if err := wikiNamespaces.Create(&wikiNamespaces.Entity{Name: "guide", Description: "指南"}); err != nil {
		t.Fatalf("create namespace: %v", err)
	}
	if err := wikiNamespaceEditors.SetEditorsTx(dbconnect.Connect(), "guide", []uint64{reviewer}, reviewer); err != nil {
		t.Fatalf("set editors: %v", err)
	}
	created, err := Create(CreateParams{Namespace: "guide", Path: "guide/rollback", Title: "原版", Content: "原内容", UserId: reviewer})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	page := wikiPages.Get(created.PageId)
	edited, err := Edit(EditParams{PageID: page.Id, Title: "被拒版", Content: "新内容", UserId: reviewer})
	if err != nil {
		t.Fatalf("edit: %v", err)
	}
	if _, err := Review(ReviewParams{RevisionID: edited.RevisionId, Action: "reject", UserId: reviewer}); err != nil {
		t.Fatalf("reject: %v", err)
	}

	topic := topics.Get(page.TopicId)
	firstPost := posts.Get(topic.FirstPostId)
	if firstPost.Content != "原内容" {
		t.Fatalf("first post content after reject=%q, want original", firstPost.Content)
	}
	if topic.Title != "原版" {
		t.Fatalf("topic title after reject=%q, want original", topic.Title)
	}
	if LoadPending(page.Id) != nil {
		t.Fatal("pending should be cleared after reject")
	}
	// 状态流转经 UpdateStatusTx 记录审核人（review：reject 不再内联表名更新）。
	rejected := wikiPageRevisions.Get(edited.RevisionId)
	if rejected.Status != wikiPageRevisions.StatusRejected || rejected.ReviewedBy != reviewer {
		t.Fatalf("rejected revision=%+v, want status=rejected reviewed_by=%d", rejected, reviewer)
	}
}

// TestPermissionMatrix 权限边界：非贡献者创建被拒；非 PageManager 审核被拒。
func TestPermissionMatrix(t *testing.T) {
	setupWikiTestDB(t)
	editor := seedWikiUser(t, false)   // 贡献者，无 PageManager
	outsider := seedWikiUser(t, false) // 非贡献者，无 PageManager

	if err := wikiNamespaces.Create(&wikiNamespaces.Entity{Name: "guide", Description: "指南"}); err != nil {
		t.Fatalf("create namespace: %v", err)
	}
	// userId=2 非贡献者 → 创建被拒。
	if _, err := Create(CreateParams{Namespace: "guide", Path: "guide/x", Title: "X", Content: "x", UserId: outsider}); err != ErrForbidden {
		t.Fatalf("non-editor create err=%v, want ErrForbidden", err)
	}
	// 贡献者 userId=1 创建成功。
	if err := wikiNamespaceEditors.SetEditorsTx(dbconnect.Connect(), "guide", []uint64{editor}, editor); err != nil {
		t.Fatalf("set editors: %v", err)
	}
	created, err := Create(CreateParams{Namespace: "guide", Path: "guide/x", Title: "X", Content: "x", UserId: editor})
	if err != nil {
		t.Fatalf("editor create: %v", err)
	}
	page := wikiPages.Get(created.PageId)
	// 非 PageManager（userId=2）审核被拒。
	edited, err := Edit(EditParams{PageID: page.Id, Title: "X2", Content: "x2", UserId: editor})
	if err != nil {
		t.Fatalf("edit: %v", err)
	}
	if _, err := Review(ReviewParams{RevisionID: edited.RevisionId, Action: "approve", UserId: outsider}); err != ErrForbidden {
		t.Fatalf("non-manager review err=%v, want ErrForbidden", err)
	}
}

// wikiSeedTestPages 创建 namespace "docs" 并返回页面（manager 需为 PageManager）。
func wikiSeedTestPages(t *testing.T, manager uint64, paths []string) map[string]uint64 {
	t.Helper()
	if err := wikiNamespaces.Create(&wikiNamespaces.Entity{Name: "docs", Description: "文档"}); err != nil {
		t.Fatalf("create namespace: %v", err)
	}
	if err := wikiNamespaceEditors.SetEditorsTx(dbconnect.Connect(), "docs", []uint64{manager}, manager); err != nil {
		t.Fatalf("set editors: %v", err)
	}
	result := make(map[string]uint64, len(paths))
	for _, p := range paths {
		created, err := Create(CreateParams{Namespace: "docs", Path: p, Title: p, Content: "内容", UserId: manager})
		if err != nil {
			t.Fatalf("create page %q: %v", p, err)
		}
		result[p] = created.PageId
	}
	return result
}

// TestApplyTreeOpsAtomicity 批次内任一操作失败整体回滚（契约：any failure aborts the batch）。
func TestApplyTreeOpsAtomicity(t *testing.T) {
	setupWikiTestDB(t)
	manager := seedWikiUser(t, true)
	pages := wikiSeedTestPages(t, manager, []string{"docs/a", "docs/b"})
	aID, bID := pages["docs/a"], pages["docs/b"]

	before := wikiPages.Get(aID).SortOrder
	// op1 合法 sort + op2 非法 move：op2 失败应回滚 op1 的写入。
	err := ApplyTreeOps([]TreeOp{
		{Op: "sort", PageId: aID, SortOrder: 9},
		{Op: "move", PageId: bID, ParentPath: "nonexistent/path"},
	}, manager)
	if err != ErrPageNotFound {
		t.Fatalf("ApplyTreeOps err=%v, want ErrPageNotFound", err)
	}
	if got := wikiPages.Get(aID).SortOrder; got != before {
		t.Fatalf("sort order after rollback=%d, want %d", got, before)
	}
}

// TestDeletePageTreeRollback 删除生命周期原子性：批次内删除被后续失败回滚后，
// topic 不应停留在 MODERATOR_REMOVED，wiki_pages 行与修订应完整保留。
func TestDeletePageTreeRollback(t *testing.T) {
	setupWikiTestDB(t)
	manager := seedWikiUser(t, true)
	pages := wikiSeedTestPages(t, manager, []string{"docs/a", "docs/b"})
	aID, bID := pages["docs/a"], pages["docs/b"]
	bTopicID := wikiPages.Get(bID).TopicId

	// op1 合法 delete B + op2 非法 move A：回滚应撤销 B 的整个删除生命周期。
	err := ApplyTreeOps([]TreeOp{
		{Op: "delete", PageId: bID},
		{Op: "move", PageId: aID, ParentPath: "nonexistent/path"},
	}, manager)
	if err != ErrPageNotFound {
		t.Fatalf("ApplyTreeOps err=%v, want ErrPageNotFound", err)
	}
	if got := wikiPages.Get(bID); got.Id == 0 {
		t.Fatal("B wiki_pages row should survive rollback")
	}
	topic := topics.Get(bTopicID)
	if topic.Id == 0 || topic.DeletedAt.Valid || topic.VisibilityStatus != topics.VisibilityActive {
		t.Fatalf("B topic should not be marked removed after rollback: id=%d deletedAt=%v visibility=%q",
			topic.Id, topic.DeletedAt.Valid, topic.VisibilityStatus)
	}
	if len(wikiPageRevisions.ListByPage(bID)) == 0 {
		t.Fatal("B revisions should survive rollback")
	}
}

// TestMoveValidation 拒绝自引用/子孙循环/跨 namespace 移动，防 parent_id 环死锁。
func TestMoveValidation(t *testing.T) {
	setupWikiTestDB(t)
	manager := seedWikiUser(t, true)
	pages := wikiSeedTestPages(t, manager, []string{"docs/guide", "docs/guide/tips"})
	aID, bID := pages["docs/guide"], pages["docs/guide/tips"]
	// 第二个 namespace 的父页面，用于跨 namespace 校验。
	if err := wikiNamespaces.Create(&wikiNamespaces.Entity{Name: "other", Description: "其他"}); err != nil {
		t.Fatalf("create namespace: %v", err)
	}
	if err := wikiNamespaceEditors.SetEditorsTx(dbconnect.Connect(), "other", []uint64{manager}, manager); err != nil {
		t.Fatalf("set editors: %v", err)
	}
	if _, err := Create(CreateParams{Namespace: "other", Path: "other/root", Title: "Root", Content: "r", UserId: manager}); err != nil {
		t.Fatalf("create other/root: %v", err)
	}

	// 自引用：把 A 移到自身路径下。
	if err := ApplyTreeOps([]TreeOp{{Op: "move", PageId: aID, ParentPath: "docs/guide"}}, manager); err != ErrPathInvalid {
		t.Fatalf("self-parent move err=%v, want ErrPathInvalid", err)
	}
	if got := wikiPages.Get(aID).ParentId; got != 0 {
		t.Fatalf("A.ParentId after self-move=%d, want 0", got)
	}
	// 子孙循环：把 A 移到其子孙 B 下。
	if err := ApplyTreeOps([]TreeOp{{Op: "move", PageId: aID, ParentPath: "docs/guide/tips"}}, manager); err != ErrPathInvalid {
		t.Fatalf("descendant move err=%v, want ErrPathInvalid", err)
	}
	if got := wikiPages.Get(aID).ParentId; got != 0 {
		t.Fatalf("A.ParentId after descendant move=%d, want 0", got)
	}
	// 跨 namespace：把 A 移到 other/root 下。
	if err := ApplyTreeOps([]TreeOp{{Op: "move", PageId: aID, ParentPath: "other/root"}}, manager); err != ErrPathInvalid {
		t.Fatalf("cross-namespace move err=%v, want ErrPathInvalid", err)
	}
	if got := wikiPages.Get(aID).ParentId; got != 0 {
		t.Fatalf("A.ParentId after cross-namespace move=%d, want 0", got)
	}

	// 未持久化任何环：A 有 1 个子（B），B 无子；B 仍可删除。
	if got := wikiPages.CountChildren(aID); got != 1 {
		t.Fatalf("CountChildren(A)=%d, want 1", got)
	}
	if got := wikiPages.CountChildren(bID); got != 0 {
		t.Fatalf("CountChildren(B)=%d, want 0", got)
	}
	if err := ApplyTreeOps([]TreeOp{{Op: "delete", PageId: bID}}, manager); err != nil {
		t.Fatalf("delete B after rejected moves: %v", err)
	}
}

// TestRenameSyncsTopicTitleAndSearchDoc rename 同步 topics.title 与 approved 修订标题
// （review B2：此前 rename 只改修订行，topics.title 停留在旧值）。
func TestRenameSyncsTopicTitleAndSearchDoc(t *testing.T) {
	setupWikiTestDB(t)
	manager := seedWikiUser(t, true)
	pages := wikiSeedTestPages(t, manager, []string{"docs/rename-me"})
	pageID := pages["docs/rename-me"]
	page := wikiPages.Get(pageID)

	if err := ApplyTreeOps([]TreeOp{{Op: "rename", PageId: page.Id, NewTitle: "New Title"}}, manager); err != nil {
		t.Fatalf("rename: %v", err)
	}
	if got := topics.Get(page.TopicId).Title; got != "New Title" {
		t.Fatalf("topic title after rename=%q, want New Title", got)
	}
	if got := wikiPageRevisions.GetLatestApproved(page.Id).Title; got != "New Title" {
		t.Fatalf("approved revision title after rename=%q, want New Title", got)
	}
}

// TestRenameCascadesChildPaths rename 重命名路径时级联更新后代页面的 path 前缀。
func TestRenameCascadesChildPaths(t *testing.T) {
	setupWikiTestDB(t)
	manager := seedWikiUser(t, true)
	pages := wikiSeedTestPages(t, manager, []string{"docs/guide", "docs/guide/tips"})
	aID, bID := pages["docs/guide"], pages["docs/guide/tips"]

	if err := ApplyTreeOps([]TreeOp{{Op: "rename", PageId: aID, NewPath: "docs/tutorial"}}, manager); err != nil {
		t.Fatalf("rename A path: %v", err)
	}
	if got := wikiPages.Get(aID).Path; got != "docs/tutorial" {
		t.Fatalf("A path after rename=%q, want docs/tutorial", got)
	}
	if got := wikiPages.Get(bID).Path; got != "docs/tutorial/tips" {
		t.Fatalf("B path after rename=%q, want docs/tutorial/tips", got)
	}
}

// TestBuildAdminTreeReturnsFullPaths 管理树 path 为完整路径（含 namespace 段），
// 保证前端 href="/wiki/${page.path}" 可直接解析，不再产生相对 slug 的 404（review #219）。
func TestBuildAdminTreeReturnsFullPaths(t *testing.T) {
	setupWikiTestDB(t)
	editor := seedWikiUser(t, true)

	if err := wikiNamespaces.Create(&wikiNamespaces.Entity{Name: "guide", Description: "指南"}); err != nil {
		t.Fatalf("create namespace: %v", err)
	}
	if err := wikiNamespaceEditors.SetEditorsTx(dbconnect.Connect(), "guide", []uint64{editor}, editor); err != nil {
		t.Fatalf("set editors: %v", err)
	}
	for _, path := range []string{"guide/getting-started", "guide/content"} {
		if _, err := Create(CreateParams{Namespace: "guide", Path: path, Title: path, Content: "正文", UserId: editor}); err != nil {
			t.Fatalf("create %s: %v", path, err)
		}
	}

	tree := BuildAdminTree()
	var guide *AdminTreeNamespace
	for i := range tree {
		if tree[i].Name == "guide" {
			guide = &tree[i]
			break
		}
	}
	if guide == nil {
		t.Fatalf("admin tree missing guide namespace: %+v", tree)
	}
	if len(guide.Pages) != 2 {
		t.Fatalf("admin tree pages=%d, want 2: %+v", len(guide.Pages), guide.Pages)
	}
	paths := map[string]bool{}
	for _, p := range guide.Pages {
		paths[p.Path] = true
	}
	for _, want := range []string{"guide/getting-started", "guide/content"} {
		if !paths[want] {
			t.Fatalf("admin tree missing full path %q (paths=%v)", want, paths)
		}
	}
	// URL 往返：/wiki/<full path> 应可被 GetByPath 解析（路由按完整路径查找）。
	if got := wikiPages.GetByPath("guide/getting-started"); got.Id == 0 {
		t.Fatal("GetByPath(guide/getting-started) unresolved")
	}
}

// TestListRevisionsApprovedOnly 公开修订历史仅返回 approved，pending/rejected
// 内容不得泄漏给公众（review #219）。
func TestListRevisionsApprovedOnly(t *testing.T) {
	setupWikiTestDB(t)
	editor := seedWikiUser(t, false)

	if err := wikiNamespaces.Create(&wikiNamespaces.Entity{Name: "guide", Description: "指南"}); err != nil {
		t.Fatalf("create namespace: %v", err)
	}
	if err := wikiNamespaceEditors.SetEditorsTx(dbconnect.Connect(), "guide", []uint64{editor}, editor); err != nil {
		t.Fatalf("set editors: %v", err)
	}
	created, err := Create(CreateParams{Namespace: "guide", Path: "guide/revisions", Title: "原始", Content: "PUBLISHED-CONTENT", UserId: editor})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	page := wikiPages.Get(created.PageId)

	// 追加 pending 修订（编辑生成），再直接插入一条 rejected 修订。
	if _, err := Edit(EditParams{PageID: page.Id, Title: "待审", Content: "PENDING-SECRET", UserId: editor}); err != nil {
		t.Fatalf("edit: %v", err)
	}
	if err := wikiPageRevisions.Create(&wikiPageRevisions.Entity{
		PageId: page.Id, RevisionNo: 3, Title: "被拒", Content: "REJECTED-SECRET",
		Status: wikiPageRevisions.StatusRejected, EditorId: editor,
	}); err != nil {
		t.Fatalf("insert rejected revision: %v", err)
	}

	items := ListRevisions(page.Id)
	if len(items) != 1 {
		t.Fatalf("ListRevisions count=%d, want 1 (pending/rejected excluded): %+v", len(items), items)
	}
	if items[0].RevisionNo != 1 || items[0].Status != StatusStringApproved || items[0].Content != "PUBLISHED-CONTENT" {
		t.Fatalf("ListRevisions[0]=%+v, want approved#1 published content", items[0])
	}
}

// TestCreateUppercaseNamespace 大写 namespace 入参应归一为小写（review #219）。
func TestCreateUppercaseNamespace(t *testing.T) {
	setupWikiTestDB(t)
	editor := seedWikiUser(t, true)

	if err := wikiNamespaces.Create(&wikiNamespaces.Entity{Name: "guide", Description: "指南"}); err != nil {
		t.Fatalf("create namespace: %v", err)
	}
	if err := wikiNamespaceEditors.SetEditorsTx(dbconnect.Connect(), "guide", []uint64{editor}, editor); err != nil {
		t.Fatalf("set editors: %v", err)
	}
	created, err := Create(CreateParams{
		Namespace: "Guide", // 大写 → 应归一为 guide
		Path:      "Guide/Getting-Started",
		Title:     "快速开始",
		Content:   "正文",
		UserId:    editor,
	})
	if err != nil {
		t.Fatalf("create with uppercase namespace: %v", err)
	}
	if created.Path != "guide/getting-started" {
		t.Fatalf("created path=%q, want guide/getting-started", created.Path)
	}
	page := wikiPages.Get(created.PageId)
	if page.Namespace != "guide" {
		t.Fatalf("page namespace=%q, want guide", page.Namespace)
	}
}

// TestReviewApproveSyncsTopicMeta approve 应同步 topic 的 Excerpt/FirstImageURL
// 为修订内容（review #219：此前列表摘要停留在创建时内容）。
func TestReviewApproveSyncsTopicMeta(t *testing.T) {
	setupWikiTestDB(t)
	reviewer := seedWikiUser(t, true)

	if err := wikiNamespaces.Create(&wikiNamespaces.Entity{Name: "guide", Description: "指南"}); err != nil {
		t.Fatalf("create namespace: %v", err)
	}
	if err := wikiNamespaceEditors.SetEditorsTx(dbconnect.Connect(), "guide", []uint64{reviewer}, reviewer); err != nil {
		t.Fatalf("set editors: %v", err)
	}
	created, err := Create(CreateParams{
		Namespace: "guide",
		Path:      "guide/meta",
		Title:     "Meta",
		Content:   "![图](/uploads/a.png)\n\n旧摘要文本",
		UserId:    reviewer,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	page := wikiPages.Get(created.PageId)
	topic := topics.Get(page.TopicId)
	if topic.Excerpt == "" || topic.FirstImageURL != "/uploads/a.png" {
		t.Fatalf("create should set excerpt=%q first_image=%q", topic.Excerpt, topic.FirstImageURL)
	}

	edited, err := Edit(EditParams{PageID: page.Id, Title: "Meta v2", Content: "![图](/uploads/b.png)\n\n新摘要文本", UserId: reviewer})
	if err != nil {
		t.Fatalf("edit: %v", err)
	}
	if _, err := Review(ReviewParams{RevisionID: edited.RevisionId, Action: "approve", UserId: reviewer}); err != nil {
		t.Fatalf("approve: %v", err)
	}

	topic = topics.Get(page.TopicId)
	if topic.Title != "Meta v2" {
		t.Fatalf("topic title after approve=%q, want Meta v2", topic.Title)
	}
	if topic.FirstImageURL != "/uploads/b.png" {
		t.Fatalf("first image after approve=%q, want /uploads/b.png", topic.FirstImageURL)
	}
	if !strings.Contains(topic.Excerpt, "新摘要文本") {
		t.Fatalf("excerpt after approve=%q, want new content", topic.Excerpt)
	}
}
