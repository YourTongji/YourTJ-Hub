package wikiservice

import (
	"fmt"
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
