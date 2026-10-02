package inboxmail

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"gorm.io/gorm"
)

func testContent(body string) VersionContent {
	return VersionContent{
		Title:         "标题",
		Summary:       "摘要",
		BodyMarkdown:  body,
		Blocks:        []Block{{Type: "badge", Payload: json.RawMessage(`{"badgeCode":"welcome_2026"}`)}},
		SchemaVersion: 1,
	}
}

func mustCreateMessage(t *testing.T, tx *gorm.DB, code string) MessageEntity {
	t.Helper()
	message, err := CreateMessageTx(tx, code, "测试消息", 1)
	if err != nil {
		t.Fatalf("CreateMessageTx(%q): %v", code, err)
	}
	return message
}

func mustCreateDraft(t *testing.T, tx *gorm.DB, messageID uint64, content VersionContent) MessageVersionEntity {
	t.Helper()
	version, err := CreateDraftVersionTx(tx, messageID, content, 1)
	if err != nil {
		t.Fatalf("CreateDraftVersionTx: %v", err)
	}
	return version
}

func mustPublish(t *testing.T, tx *gorm.DB, versionID uint64) MessageVersionEntity {
	t.Helper()
	published, err := PublishVersionTx(tx, versionID, 7, time.Date(2026, 9, 28, 10, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("PublishVersionTx: %v", err)
	}
	return published
}

// 发布后的 Message Version 是投递历史的唯一内容源：任何路径都不得原地修改或删除，
// 修订只能通过新建 version 行完成（epic #769 核心决策 3）。
func TestPublishedVersionRejectsInPlaceMutation(t *testing.T) {
	tx := openTestDB(t)
	message := mustCreateMessage(t, tx, "security.password_missing")
	draft := mustCreateDraft(t, tx, message.Id, testContent("draft body"))
	published := mustPublish(t, tx, draft.Id)
	if published.Status != StatusPublished || published.PublishedAt == nil || published.PublishedBy != 7 {
		t.Fatalf("published row not frozen correctly: %#v", published)
	}
	wantHash, err := HashVersionContent(testContent("draft body"))
	if err != nil {
		t.Fatalf("HashVersionContent: %v", err)
	}
	if published.ContentHash != wantHash {
		t.Fatalf("published content hash = %q, want %q", published.ContentHash, wantHash)
	}

	cases := []struct {
		name string
		run  func() error
	}{
		{
			name: "repository draft update",
			run: func() error {
				return UpdateDraftContentTx(tx, published.Id, testContent("mutated body"))
			},
		},
		{
			name: "model map update",
			run: func() error {
				return tx.Model(&MessageVersionEntity{Id: published.Id}).
					Updates(map[string]any{"body_markdown": "mutated"}).Error
			},
		},
		{
			name: "model single column update",
			run: func() error {
				return tx.Model(&MessageVersionEntity{Id: published.Id}).Update("title", "mutated").Error
			},
		},
		{
			name: "model save",
			run: func() error {
				loaded, err := GetVersionByIDTx(tx, published.Id)
				if err != nil {
					return err
				}
				loaded.BodyMarkdown = "mutated"
				return tx.Save(&loaded).Error
			},
		},
		{
			name: "model delete",
			run: func() error {
				return tx.Where("id = ?", published.Id).Delete(&MessageVersionEntity{Id: published.Id}).Error
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.run(); !errors.Is(err, ErrPublishedVersionImmutable) {
				t.Fatalf("mutation error = %v, want ErrPublishedVersionImmutable", err)
			}
		})
	}

	stored, err := GetVersionByIDTx(tx, published.Id)
	if err != nil {
		t.Fatalf("reload published version: %v", err)
	}
	if stored.BodyMarkdown != "draft body" || stored.ContentHash != wantHash || stored.Status != StatusPublished {
		t.Fatalf("published row was mutated: %#v", stored)
	}

	// 重复发布必须幂等：Worker 重试/事件重放不得改变已发布内容。
	again, err := PublishVersionTx(tx, published.Id, 99, time.Now())
	if err != nil {
		t.Fatalf("idempotent publish: %v", err)
	}
	if again.ContentHash != wantHash || again.PublishedBy != 7 {
		t.Fatalf("re-publish changed published row: %#v", again)
	}
}

// 模型层必须拒绝无法定位目标行的批量更新：hook 无法确认目标是否为草稿时，
// 宁可拒绝也不能冒改写已发布历史的风险。
func TestVersionMutationWithoutPrimaryKeyIsRefused(t *testing.T) {
	tx := openTestDB(t)
	message := mustCreateMessage(t, tx, "security.scoped")
	draft := mustCreateDraft(t, tx, message.Id, testContent("body"))

	err := tx.Model(&MessageVersionEntity{}).Where("id = ?", draft.Id).Update("title", "mutated").Error
	if !errors.Is(err, ErrUnscopedVersionMutation) {
		t.Fatalf("unscoped update error = %v, want ErrUnscopedVersionMutation", err)
	}
}

func TestDraftVersionRemainsEditable(t *testing.T) {
	tx := openTestDB(t)
	message := mustCreateMessage(t, tx, "security.draft_edit")
	draft := mustCreateDraft(t, tx, message.Id, testContent("first body"))

	updated := testContent("second body")
	if err := UpdateDraftContentTx(tx, draft.Id, updated); err != nil {
		t.Fatalf("UpdateDraftContentTx: %v", err)
	}
	stored, err := GetVersionByIDTx(tx, draft.Id)
	if err != nil {
		t.Fatalf("reload draft: %v", err)
	}
	if stored.BodyMarkdown != "second body" || stored.Status != StatusDraft {
		t.Fatalf("draft not updated: %#v", stored)
	}
	wantHash, err := HashVersionContent(updated)
	if err != nil {
		t.Fatalf("HashVersionContent: %v", err)
	}
	if stored.ContentHash != wantHash {
		t.Fatalf("draft content hash = %q, want %q", stored.ContentHash, wantHash)
	}
}

func TestRevisionCreatesNewVersionRow(t *testing.T) {
	tx := openTestDB(t)
	message := mustCreateMessage(t, tx, "activity.welcome")
	first := mustCreateDraft(t, tx, message.Id, testContent("v1 body"))
	publishedFirst := mustPublish(t, tx, first.Id)

	second := mustCreateDraft(t, tx, message.Id, testContent("v2 body"))
	if second.VersionNo != publishedFirst.VersionNo+1 {
		t.Fatalf("revision version_no = %d, want %d", second.VersionNo, publishedFirst.VersionNo+1)
	}

	// 列表/投递默认读取最新已发布版本，草稿修订不得改变它。
	latest, err := GetLatestPublishedVersionTx(tx, message.Id)
	if err != nil {
		t.Fatalf("GetLatestPublishedVersionTx: %v", err)
	}
	if latest.Id != publishedFirst.Id || latest.BodyMarkdown != "v1 body" {
		t.Fatalf("latest published version = %#v, want %#v", latest, publishedFirst)
	}
	if _, err := GetLatestPublishedVersionTx(tx, message.Id+999); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("missing message error = %v, want gorm.ErrRecordNotFound", err)
	}
}

func TestCreateDraftVersionStampsContentHashAndSequence(t *testing.T) {
	tx := openTestDB(t)
	message := mustCreateMessage(t, tx, "system.maintenance")
	first := mustCreateDraft(t, tx, message.Id, testContent("body"))
	wantHash, err := HashVersionContent(testContent("body"))
	if err != nil {
		t.Fatalf("HashVersionContent: %v", err)
	}
	if first.VersionNo != 1 || first.ContentHash != wantHash || first.Status != StatusDraft || first.MessageId != message.Id {
		t.Fatalf("draft row = %#v", first)
	}
	if err := tx.Create(&MessageVersionEntity{MessageId: message.Id, VersionNo: first.VersionNo}).Error; !errors.Is(err, gorm.ErrDuplicatedKey) {
		t.Fatalf("duplicate (message_id, version_no) error = %v, want gorm.ErrDuplicatedKey", err)
	}
}

// 内容哈希必须对语义相同的内容稳定：Block payload 的 JSON key 顺序不同
// （由不同序列化路径产生）不得改变哈希，否则幂等判断会误判为内容变更。
func TestHashVersionContentIsCanonical(t *testing.T) {
	left := VersionContent{
		Title:        "t",
		BodyMarkdown: "b",
		Blocks:       []Block{{Type: "badge", Payload: json.RawMessage(`{"b":1,"a":{"d":2,"c":3}}`)}},
	}
	right := VersionContent{
		Title:        "t",
		BodyMarkdown: "b",
		Blocks:       []Block{{Type: "badge", Payload: json.RawMessage(`{"a":{"c":3,"d":2},"b":1}`)}},
	}
	leftHash, err := HashVersionContent(left)
	if err != nil {
		t.Fatalf("HashVersionContent(left): %v", err)
	}
	rightHash, err := HashVersionContent(right)
	if err != nil {
		t.Fatalf("HashVersionContent(right): %v", err)
	}
	if leftHash != rightHash {
		t.Fatalf("canonical hashes differ: %q vs %q", leftHash, rightHash)
	}

	changed := left
	changed.BodyMarkdown = "b2"
	changedHash, err := HashVersionContent(changed)
	if err != nil {
		t.Fatalf("HashVersionContent(changed): %v", err)
	}
	if changedHash == leftHash {
		t.Fatal("content change did not change the hash")
	}

	// 大整数必须保留精度：float64 解码会让这两个不同 id 产生同一哈希。
	bigInt := VersionContent{Blocks: []Block{{Type: "badge", Payload: json.RawMessage(`{"badgeId":9007199254740993}`)}}}
	smallerInt := VersionContent{Blocks: []Block{{Type: "badge", Payload: json.RawMessage(`{"badgeId":9007199254740992}`)}}}
	bigHash, err := HashVersionContent(bigInt)
	if err != nil {
		t.Fatalf("HashVersionContent(bigInt): %v", err)
	}
	smallerHash, err := HashVersionContent(smallerInt)
	if err != nil {
		t.Fatalf("HashVersionContent(smallerInt): %v", err)
	}
	if bigHash == smallerHash {
		t.Fatal("large integer precision lost in content hash")
	}
}

// Block payload 在 JSON 值之后带尾随数据时必须拒绝：Decode 只读单个值，
// 不校验会让 {"a":1}junk 按 {"a":1} 参与哈希，而库存 RawMessage 仍含 junk，
// 哈希与事实内容不再一致。
func TestHashVersionContentRejectsTrailingGarbage(t *testing.T) {
	for _, payload := range []string{
		`{"a":1}junk`,
		`{"a":1} {"b":2}`,
		`{"a":1}"`,
	} {
		_, err := HashVersionContent(VersionContent{
			Blocks: []Block{{Type: "badge", Payload: json.RawMessage(payload)}},
		})
		if err == nil {
			t.Fatalf("payload %q with trailing data was accepted", payload)
		}
	}
	// 值后的空白属于合法 JSON，必须仍然通过。
	if _, err := HashVersionContent(VersionContent{
		Blocks: []Block{{Type: "badge", Payload: json.RawMessage("{\"a\":1}\n")}},
	}); err != nil {
		t.Fatalf("trailing whitespace rejected: %v", err)
	}
}
