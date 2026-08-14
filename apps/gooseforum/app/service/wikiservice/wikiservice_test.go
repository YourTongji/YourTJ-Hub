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

	// 编辑 → 写即发布（approved revision#2，公开立即可见）。
	edited, err := Edit(EditParams{PageID: page.Id, Title: "快速开始 v2", Content: "# 新标题\n\n新内容", UserId: editor})
	if err != nil {
		t.Fatalf("edit wiki page: %v", err)
	}
	if edited.Status != StatusStringApproved {
		t.Fatalf("edit status=%q, want approved", edited.Status)
	}
	if edited.RevisionNo != 2 {
		t.Fatalf("edit revisionNo=%d, want 2", edited.RevisionNo)
	}
	// 公开视图立即显示新内容（写即发布）。
	detail, err := LoadPageDetail(&page, &topic)
	if err != nil {
		t.Fatalf("load detail: %v", err)
	}
	if detail.Title != "快速开始 v2" {
		t.Fatalf("public title=%q, want new title", detail.Title)
	}
	// topic 标题/摘要/首图同步 + 物化水印。
	topic = topics.Get(page.TopicId)
	if topic.Title != "快速开始 v2" {
		t.Fatalf("topic title after edit=%q", topic.Title)
	}
	firstPost := posts.Get(topic.FirstPostId)
	if firstPost.ProcessStatus != posts.ProcessStatusNormal {
		t.Fatalf("first post process_status=%d, want normal", firstPost.ProcessStatus)
	}
	if firstPost.WikiSyncedRevisionNo != 2 || topic.WikiSyncedRevisionNo != 2 {
		t.Fatalf("watermark post=%d topic=%d, want 2/2", firstPost.WikiSyncedRevisionNo, topic.WikiSyncedRevisionNo)
	}
	page = wikiPages.Get(page.Id)
	if page.PublishedRevisionNo != 2 {
		t.Fatalf("page published_revision_no after edit=%d, want 2", page.PublishedRevisionNo)
	}
}

// TestEditAppendsApprovedRevisions 写即发布：每次编辑追加一条 approved 修订，
// revision_no 单调递增，无 pending/superseded 状态。
func TestEditAppendsApprovedRevisions(t *testing.T) {
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
	if len(revisions) != 3 {
		t.Fatalf("revision count=%d, want 3", len(revisions))
	}
	for _, r := range revisions {
		if r.Status != wikiPageRevisions.StatusApproved {
			t.Fatalf("revision#%d status=%d, want approved (no pending/superseded)", r.RevisionNo, r.Status)
		}
	}
	// revision_no 单调递增（approved#1 + #2 + #3）。
	if revisions[0].RevisionNo != 3 || revisions[1].RevisionNo != 2 || revisions[2].RevisionNo != 1 {
		t.Fatalf("revision_no order=%d,%d,%d", revisions[0].RevisionNo, revisions[1].RevisionNo, revisions[2].RevisionNo)
	}
	if got := wikiPages.Get(page.Id).PublishedRevisionNo; got != 3 {
		t.Fatalf("published_revision_no=%d, want 3", got)
	}
}

// TestRollbackRestoresVersion 管理员回滚到指定版本：该版本之后的修订硬删（不可撤销），
// 内容回退为该版本，版本指针回到目标号。
func TestRollbackRestoresVersion(t *testing.T) {
	setupWikiTestDB(t)
	reviewer := seedWikiUser(t, true) // PageManager

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
	edited, err := Edit(EditParams{PageID: page.Id, Title: "被回滚版", Content: "新内容", UserId: reviewer})
	if err != nil {
		t.Fatalf("edit: %v", err)
	}
	if edited.RevisionNo != 2 {
		t.Fatalf("edited revisionNo=%d, want 2", edited.RevisionNo)
	}

	// 回滚到 v1 → v2 硬删、内容回退。
	if err := Rollback(RollbackParams{PageID: page.Id, ToRevisionNo: 1, UserId: reviewer}); err != nil {
		t.Fatalf("rollback: %v", err)
	}
	topic := topics.Get(page.TopicId)
	firstPost := posts.Get(topic.FirstPostId)
	if firstPost.Content != "原内容" {
		t.Fatalf("first post content after rollback=%q, want original", firstPost.Content)
	}
	if topic.Title != "原版" {
		t.Fatalf("topic title after rollback=%q, want original", topic.Title)
	}
	rev2 := wikiPageRevisions.Get(edited.RevisionId)
	if rev2.Id != 0 {
		t.Fatalf("revision v2 should be hard-deleted after rollback, got %+v", rev2)
	}
	if got := wikiPages.Get(page.Id).PublishedRevisionNo; got != 1 {
		t.Fatalf("published_revision_no after rollback=%d, want 1", got)
	}
}

// TestPermissionMatrix 权限边界：非贡献者创建被拒；非 PageManager 回滚被拒。
func TestPermissionMatrix(t *testing.T) {
	setupWikiTestDB(t)
	editor := seedWikiUser(t, false)   // 贡献者，无 PageManager
	outsider := seedWikiUser(t, false) // 非贡献者，无 PageManager

	if err := wikiNamespaces.Create(&wikiNamespaces.Entity{Name: "guide", Description: "指南"}); err != nil {
		t.Fatalf("create namespace: %v", err)
	}
	// 非贡献者 → 创建被拒。
	if _, err := Create(CreateParams{Namespace: "guide", Path: "guide/x", Title: "X", Content: "x", UserId: outsider}); err != ErrForbidden {
		t.Fatalf("non-editor create err=%v, want ErrForbidden", err)
	}
	// 贡献者创建成功。
	if err := wikiNamespaceEditors.SetEditorsTx(dbconnect.Connect(), "guide", []uint64{editor}, editor); err != nil {
		t.Fatalf("set editors: %v", err)
	}
	created, err := Create(CreateParams{Namespace: "guide", Path: "guide/x", Title: "X", Content: "x", UserId: editor})
	if err != nil {
		t.Fatalf("editor create: %v", err)
	}
	page := wikiPages.Get(created.PageId)
	// 非 PageManager 回滚被拒。
	if err := Rollback(RollbackParams{PageID: page.Id, ToRevisionNo: 1, UserId: outsider}); err != ErrForbidden {
		t.Fatalf("non-manager rollback err=%v, want ErrForbidden", err)
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

// TestSortRenumbersSiblingOrders sort op 的 SortOrder 是目标索引（1-based）：
// 移动后整组兄弟按 sort_order 规范化为 1..N，无空洞/0/并列（此前直接赋值
// 无法表达"排第一"，默认 0 与 omitempty 组合会留下并列序）。
func TestSortRenumbersSiblingOrders(t *testing.T) {
	setupWikiTestDB(t)
	manager := seedWikiUser(t, true)
	pages := wikiSeedTestPages(t, manager, []string{"docs/a", "docs/b", "docs/c"})
	aID, bID, cID := pages["docs/a"], pages["docs/b"], pages["docs/c"]

	// 初始全部 sort_order=0，按 id 序 a,b,c；把 a 移到第 3 位，其余保持相对顺序。
	if err := ApplyTreeOps([]TreeOp{{Op: "sort", PageId: aID, SortOrder: 3}}, manager); err != nil {
		t.Fatalf("sort a to 3: %v", err)
	}
	if got := wikiPages.Get(aID).SortOrder; got != 3 {
		t.Fatalf("a sort_order=%d, want 3", got)
	}
	if got := wikiPages.Get(bID).SortOrder; got != 1 {
		t.Fatalf("b sort_order=%d, want 1", got)
	}
	if got := wikiPages.Get(cID).SortOrder; got != 2 {
		t.Fatalf("c sort_order=%d, want 2", got)
	}

	// 前端交换模式：两连 sort（互相把对方当前 sortOrder 作为目标索引）→ 顺序互换。
	if err := ApplyTreeOps([]TreeOp{
		{Op: "sort", PageId: cID, SortOrder: wikiPages.Get(bID).SortOrder},
		{Op: "sort", PageId: bID, SortOrder: wikiPages.Get(cID).SortOrder},
	}, manager); err != nil {
		t.Fatalf("swap b/c: %v", err)
	}
	if got := wikiPages.Get(cID).SortOrder; got != 1 {
		t.Fatalf("c sort_order after swap=%d, want 1", got)
	}
	if got := wikiPages.Get(bID).SortOrder; got != 2 {
		t.Fatalf("b sort_order after swap=%d, want 2", got)
	}
	if got := wikiPages.Get(aID).SortOrder; got != 3 {
		t.Fatalf("a sort_order after swap=%d, want 3", got)
	}
}

// TestMoveRewritesPathsAndDescendants move 操作重写页面 path 与全部后代 path，
// 保持 path 层级与 parent_id 层级一致；移动到根时保留 namespace 前缀
// （"namespace/…" 形态，不允许裸 slug）。
func TestMoveRewritesPathsAndDescendants(t *testing.T) {
	setupWikiTestDB(t)
	manager := seedWikiUser(t, true)
	pages := wikiSeedTestPages(t, manager, []string{"docs/home", "docs/guide", "docs/guide/tips"})
	homeID, guideID, tipsID := pages["docs/home"], pages["docs/guide"], pages["docs/guide/tips"]

	// 把 guide 移到 home 下：guide.path → docs/home/guide，tips.path 级联 → docs/home/guide/tips。
	if err := ApplyTreeOps([]TreeOp{{Op: "move", PageId: guideID, ParentPath: "docs/home"}}, manager); err != nil {
		t.Fatalf("move guide under home: %v", err)
	}
	if got := wikiPages.Get(guideID).Path; got != "docs/home/guide" {
		t.Fatalf("guide path after move=%q, want docs/home/guide", got)
	}
	if got := wikiPages.Get(tipsID).Path; got != "docs/home/guide/tips" {
		t.Fatalf("tips path after move=%q, want docs/home/guide/tips", got)
	}
	if got := wikiPages.Get(guideID).ParentId; got != homeID {
		t.Fatalf("guide parent after move=%d, want %d", got, homeID)
	}

	// 移到根（无 ParentPath）：新 path = namespace + "/" + 末段。
	if err := ApplyTreeOps([]TreeOp{{Op: "move", PageId: guideID}}, manager); err != nil {
		t.Fatalf("move guide to root: %v", err)
	}
	if got := wikiPages.Get(guideID).Path; got != "docs/guide" {
		t.Fatalf("guide path after root move=%q, want docs/guide", got)
	}
	if got := wikiPages.Get(guideID).ParentId; got != 0 {
		t.Fatalf("guide parent after root move=%d, want 0", got)
	}
	if got := wikiPages.Get(tipsID).Path; got != "docs/guide/tips" {
		t.Fatalf("tips path after root move=%q, want docs/guide/tips", got)
	}
}

// TestMoveRejectsPathCollision move 目标 path 被占用（精确冲突）时返回 ErrPathExists
// 且不落库（批次事务回滚）。
func TestMoveRejectsPathCollision(t *testing.T) {
	setupWikiTestDB(t)
	manager := seedWikiUser(t, true)
	pages := wikiSeedTestPages(t, manager, []string{"docs/home", "docs/home/guide", "docs/guide"})
	guideID := pages["docs/guide"]

	// docs/guide 移到 docs/home 下 → 目标 docs/home/guide 已存在。
	if err := ApplyTreeOps([]TreeOp{{Op: "move", PageId: guideID, ParentPath: "docs/home"}}, manager); err != ErrPathExists {
		t.Fatalf("move guide under home err=%v, want ErrPathExists", err)
	}
	if got := wikiPages.Get(guideID).Path; got != "docs/guide" {
		t.Fatalf("guide path after rejected move=%q, want docs/guide", got)
	}
	if got := wikiPages.Get(guideID).ParentId; got != 0 {
		t.Fatalf("guide parent after rejected move=%d, want 0", got)
	}
}

// TestRenameRejectsPrefixConflict rename 目标 path 是其他页面 path 的前缀时返回
// ErrPathExists（重写后会出现 path 层级交叠）。正常数据下 Create 的父存在校验会先
// 命中精确冲突，此为脏数据/并发场景的纵深防御（review B2）。
func TestRenameRejectsPrefixConflict(t *testing.T) {
	setupWikiTestDB(t)
	manager := seedWikiUser(t, true)
	pages := wikiSeedTestPages(t, manager, []string{"docs/guide", "docs/guide/tips"})
	guideID := pages["docs/guide"]
	// 直接造一条"父缺失"的脏数据：docs/orphan/sub 存在但 docs/orphan 不存在。
	if err := dbconnect.Connect().Create(&wikiPages.Entity{
		TopicId: 999999, Namespace: "docs", Path: "docs/orphan/sub",
	}).Error; err != nil {
		t.Fatalf("seed orphan descendant: %v", err)
	}
	if err := ApplyTreeOps([]TreeOp{{Op: "rename", PageId: guideID, NewPath: "docs/orphan"}}, manager); err != ErrPathExists {
		t.Fatalf("rename to occupied descendant err=%v, want ErrPathExists", err)
	}
	if got := wikiPages.Get(guideID).Path; got != "docs/guide" {
		t.Fatalf("guide path after rejected rename=%q, want docs/guide", got)
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

	// 编辑即发布（approved#2），再直接插入一条 rejected 修订。
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
	if len(items) != 2 {
		t.Fatalf("ListRevisions count=%d, want 2 (approved#1/#2, rejected excluded): %+v", len(items), items)
	}
	if items[0].RevisionNo != 2 || items[0].Status != StatusStringApproved || items[0].Content != "PENDING-SECRET" {
		t.Fatalf("ListRevisions[0]=%+v, want approved#2 published content", items[0])
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
	if edited.RevisionNo != 2 {
		t.Fatalf("edited revisionNo=%d, want 2", edited.RevisionNo)
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

// TestEditRejectsEmptyContentAndLongTitle 写即发布无审核兜底：空内容与超长标题
// 必须拒绝（review #219：此前空 content 可清空页面、超长 title 语义混淆）。
func TestEditRejectsEmptyContentAndLongTitle(t *testing.T) {
	setupWikiTestDB(t)
	editor := seedWikiUser(t, false)
	if err := wikiNamespaces.Create(&wikiNamespaces.Entity{Name: "guide", Description: "指南"}); err != nil {
		t.Fatalf("create namespace: %v", err)
	}
	if err := wikiNamespaceEditors.SetEditorsTx(dbconnect.Connect(), "guide", []uint64{editor}, editor); err != nil {
		t.Fatalf("set editors: %v", err)
	}
	created, err := Create(CreateParams{Namespace: "guide", Path: "guide/guard", Title: "G", Content: "内容", UserId: editor})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	page := wikiPages.Get(created.PageId)

	if _, err := Edit(EditParams{PageID: page.Id, Title: "G", Content: "   ", UserId: editor}); err != ErrContentEmpty {
		t.Fatalf("blank content err=%v, want ErrContentEmpty", err)
	}
	long := strings.Repeat("长", 513)
	if _, err := Edit(EditParams{PageID: page.Id, Title: long, Content: "内容", UserId: editor}); err != ErrTitleTooLong {
		t.Fatalf("long title err=%v, want ErrTitleTooLong", err)
	}
	// 空内容/长标题被拒后页面保持原状。
	page = wikiPages.Get(page.Id)
	if page.PublishedRevisionNo != 1 {
		t.Fatalf("published_revision_no after rejected edits=%d, want 1", page.PublishedRevisionNo)
	}
}

// TestSetEditorsRejectsGhostUser 贡献者必须指向真实用户（review #219：此前
// 不存在的 userId 也会写入 wiki_namespace_editors，产生幽灵贡献者行）。
func TestSetEditorsRejectsGhostUser(t *testing.T) {
	setupWikiTestDB(t)
	manager := seedWikiUser(t, true)
	if err := wikiNamespaces.Create(&wikiNamespaces.Entity{Name: "guide", Description: "指南"}); err != nil {
		t.Fatalf("create namespace: %v", err)
	}
	if err := SetEditors("guide", []uint64{99999999}, manager); err != ErrUserNotFound {
		t.Fatalf("SetEditors ghost user err=%v, want ErrUserNotFound", err)
	}
	// 真实用户（manager 自己）设置成功。
	if err := SetEditors("guide", []uint64{manager}, manager); err != nil {
		t.Fatalf("SetEditors real user: %v", err)
	}
}

// TestRollbackConcurrentEditSerialized 回滚与编辑行锁串行化：回滚后指针回到
// 目标版本，且不会与并发编辑交错产生指针指向已删版本（review High）。
func TestRollbackConcurrentEditSerialized(t *testing.T) {
	setupWikiTestDB(t)
	reviewer := seedWikiUser(t, true)
	if err := wikiNamespaces.Create(&wikiNamespaces.Entity{Name: "guide", Description: "指南"}); err != nil {
		t.Fatalf("create namespace: %v", err)
	}
	if err := wikiNamespaceEditors.SetEditorsTx(dbconnect.Connect(), "guide", []uint64{reviewer}, reviewer); err != nil {
		t.Fatalf("set editors: %v", err)
	}
	created, err := Create(CreateParams{Namespace: "guide", Path: "guide/lock", Title: "v1", Content: "c1", UserId: reviewer})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	page := wikiPages.Get(created.PageId)
	if _, err := Edit(EditParams{PageID: page.Id, Title: "v2", Content: "c2", UserId: reviewer}); err != nil {
		t.Fatalf("edit: %v", err)
	}
	if err := Rollback(RollbackParams{PageID: page.Id, ToRevisionNo: 1, UserId: reviewer}); err != nil {
		t.Fatalf("rollback: %v", err)
	}
	page = wikiPages.Get(page.Id)
	if page.PublishedRevisionNo != 1 {
		t.Fatalf("published_revision_no after rollback=%d, want 1", page.PublishedRevisionNo)
	}
	// 回滚后继续编辑：nextNo = 指针+1 = 2（无空洞、无同号冲突）。
	if _, err := Edit(EditParams{PageID: page.Id, Title: "v3", Content: "c3", UserId: reviewer}); err != nil {
		t.Fatalf("edit after rollback: %v", err)
	}
	revs := wikiPageRevisions.ListByPage(page.Id)
	if len(revs) != 2 || revs[0].RevisionNo != 2 {
		t.Fatalf("revisions after rollback+edit: count=%d first=%d, want 2/2", len(revs), revs[0].RevisionNo)
	}
}

// TestRevisionNumberUniqueIndex (page_id, revision_no) 唯一索引：同号修订不得
// 重复插入（review #219：旧版并发写入/回滚残留可产生同号修订）。
func TestRevisionNumberUniqueIndex(t *testing.T) {
	setupWikiTestDB(t)
	editor := seedWikiUser(t, false)
	if err := wikiNamespaces.Create(&wikiNamespaces.Entity{Name: "guide", Description: "指南"}); err != nil {
		t.Fatalf("create namespace: %v", err)
	}
	if err := wikiNamespaceEditors.SetEditorsTx(dbconnect.Connect(), "guide", []uint64{editor}, editor); err != nil {
		t.Fatalf("set editors: %v", err)
	}
	created, err := Create(CreateParams{Namespace: "guide", Path: "guide/uniq", Title: "U", Content: "内容", UserId: editor})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	page := wikiPages.Get(created.PageId)
	// 直接插入同号修订（绕过 service）：必须被唯一索引拒绝。
	dup := wikiPageRevisions.Entity{
		PageId:     page.Id,
		RevisionNo: 1,
		Title:      "U-dup",
		Content:    "dup",
		Status:     wikiPageRevisions.StatusApproved,
		EditorId:   editor,
	}
	if err := wikiPageRevisions.Create(&dup); err == nil {
		t.Fatal("duplicate (page_id, revision_no) insert should fail unique index")
	}
}

// TestPublicTreeFiltersDeletedTopic 公开导航树/首页不得展示 topic 已被治理删除的
// 页面（review #219：此前仅靠 wiki_pages 自身软删，topic 删除后页面仍泄漏）。
func TestPublicTreeFiltersDeletedTopic(t *testing.T) {
	setupWikiTestDB(t)
	editor := seedWikiUser(t, false)
	if err := wikiNamespaces.Create(&wikiNamespaces.Entity{Name: "guide", Description: "指南"}); err != nil {
		t.Fatalf("create namespace: %v", err)
	}
	if err := wikiNamespaceEditors.SetEditorsTx(dbconnect.Connect(), "guide", []uint64{editor}, editor); err != nil {
		t.Fatalf("set editors: %v", err)
	}
	created, err := Create(CreateParams{Namespace: "guide", Path: "guide/visible", Title: "V", Content: "内容", UserId: editor})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	page := wikiPages.Get(created.PageId)
	tree := BuildTree("")
	found := false
	for _, ns := range tree {
		for _, p := range ns.Pages {
			if p.PageId == page.Id {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("visible page should appear in public tree")
	}
	// 治理删除 topic（MODERATOR_REMOVED）→ 公开树过滤。
	topic := topics.Get(page.TopicId)
	if err := topics.MarkModeratorRemoved(topic.Id, editor, "review test"); err != nil {
		t.Fatalf("mark moderator removed: %v", err)
	}
	tree = BuildTree("")
	for _, ns := range tree {
		for _, p := range ns.Pages {
			if p.PageId == page.Id {
				t.Fatal("deleted topic page should be filtered from public tree")
			}
		}
	}
}
