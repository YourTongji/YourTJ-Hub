package migration

import (
	"encoding/json"
	"errors"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/chat/messages"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/eventNotification"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/inboxmail"
	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var inboxTableNames = []string{
	"inbox_message",
	"inbox_message_version",
	"inbox_campaign",
	"inbox_campaign_attachment",
	"inbox_campaign_run",
	"inbox_delivery",
	"inbox_claim",
}

// 生产迁移入口必须注册全部站内信模型：注册丢失时表永远不会创建，
// 而模型单测用的是 AllModels()，不会发现。
func TestSchemaModelsRegistersInboxModels(t *testing.T) {
	registered := make(map[string]bool)
	for _, model := range SchemaModels() {
		if tabler, ok := model.(interface{ TableName() string }); ok {
			registered[tabler.TableName()] = true
		}
	}
	for _, table := range inboxTableNames {
		if !registered[table] {
			t.Errorf("SchemaModels() does not register %s", table)
		}
	}
}

// 全新数据库必须一次 AutoMigrate 建出全部站内信表（acceptance #1），
// 且三个幂等唯一约束在 SQLite 方言下真实生效。
func TestInboxSchemaFreshDatabaseOnSQLite(t *testing.T) {
	conn, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := conn.AutoMigrate(SchemaModels()...); err != nil {
		t.Fatalf("fresh AutoMigrate(SchemaModels) on sqlite: %v", err)
	}
	assertInboxTables(t, conn)
	assertInboxUniqueConstraints(t, conn, "sqlite")
}

// 存量库升级场景：旧库只有通知/私信表，部署新二进制后 AutoMigrate 必须补齐
// 站内信表，且不改变既有通知/私信的行与表结构（acceptance #2）。
func TestInboxSchemaUpgradePreservesExistingDataOnSQLite(t *testing.T) {
	conn, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	// 旧库形态：通知 + 私信各自的现状表。
	if err := conn.AutoMigrate(&eventNotification.Entity{}, &messages.Entity{}); err != nil {
		t.Fatalf("migrate legacy tables: %v", err)
	}
	notification := eventNotification.Entity{
		UserId:    1,
		EventType: eventNotification.EventTypeComment,
		TopicID:   1001,
		Payload: eventNotification.NotificationPayload{
			Title:   "legacy title",
			Content: "legacy content",
			TopicId: 1001,
			PostId:  2001,
		},
	}
	if err := conn.Create(&notification).Error; err != nil {
		t.Fatalf("insert legacy notification: %v", err)
	}
	chatMessage := messages.Entity{Id: 1, ConvId: 1, SenderId: 1, Content: "legacy dm", MsgType: 1}
	if err := conn.Create(&chatMessage).Error; err != nil {
		t.Fatalf("insert legacy chat message: %v", err)
	}
	legacyNotificationDDL := tableDDL(t, conn, "event_notification")
	legacyMessagesDDL := tableDDL(t, conn, "messages")

	if err := conn.AutoMigrate(SchemaModels()...); err != nil {
		t.Fatalf("upgrade AutoMigrate(SchemaModels) on sqlite: %v", err)
	}

	assertInboxTables(t, conn)
	if got := tableDDL(t, conn, "event_notification"); got != legacyNotificationDDL {
		t.Fatalf("upgrade rewrote event_notification schema:\nbefore: %s\nafter:  %s", legacyNotificationDDL, got)
	}
	if got := tableDDL(t, conn, "messages"); got != legacyMessagesDDL {
		t.Fatalf("upgrade rewrote chat messages schema:\nbefore: %s\nafter:  %s", legacyMessagesDDL, got)
	}
	var storedNotification eventNotification.Entity
	if err := conn.Where("id = ?", notification.Id).Take(&storedNotification).Error; err != nil {
		t.Fatalf("reload legacy notification: %v", err)
	}
	if storedNotification.Payload.Title != "legacy title" || storedNotification.Payload.Content != "legacy content" ||
		storedNotification.Payload.TopicId != 1001 || storedNotification.IsRead {
		t.Fatalf("legacy notification changed by upgrade: %#v", storedNotification)
	}
	var storedMessage messages.Entity
	if err := conn.Where("id = ?", chatMessage.Id).Take(&storedMessage).Error; err != nil {
		t.Fatalf("reload legacy chat message: %v", err)
	}
	if storedMessage.Content != "legacy dm" || storedMessage.ConvId != 1 || storedMessage.SenderId != 1 {
		t.Fatalf("legacy chat message changed by upgrade: %#v", storedMessage)
	}
}

// PostgreSQL 行为门禁：除了建表，还要验证 JSON 列写入、发布不可变、双重幂等
// 与匿名化 SQL 在真实 PG 方言下成立（CAST/|| 的文本推断）。
func TestInboxSchemaBehaviorOnPostgreSQL(t *testing.T) {
	dsn := os.Getenv("YOURTJ_TEST_PG_URL")
	if dsn == "" {
		t.Skip("YOURTJ_TEST_PG_URL not set; skipping PostgreSQL migration test")
	}
	conn, err := gorm.Open(postgres.Open(dsn), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	if err := conn.Exec(`DROP SCHEMA public CASCADE; CREATE SCHEMA public;`).Error; err != nil {
		t.Fatalf("reset schema: %v", err)
	}
	if err := conn.AutoMigrate(SchemaModels()...); err != nil {
		t.Fatalf("fresh AutoMigrate(SchemaModels) on postgres: %v", err)
	}
	assertInboxTables(t, conn)
	assertInboxUniqueConstraints(t, conn, "postgres")

	message, err := inboxmail.CreateMessageTx(conn, "security.pg_probe", "PG 探针", 1)
	if err != nil {
		t.Fatalf("create message on postgres: %v", err)
	}
	content := inboxmail.VersionContent{
		Title:         "安全提醒",
		Summary:       "摘要",
		BodyMarkdown:  "**正文**",
		SchemaVersion: 1,
		Blocks: []inboxmail.Block{
			{Type: "badge", Payload: json.RawMessage(`{"badgeCode":"welcome_2026"}`)},
		},
	}
	draft, err := inboxmail.CreateDraftVersionTx(conn, message.Id, content, 1)
	if err != nil {
		t.Fatalf("create draft on postgres: %v", err)
	}
	published, err := inboxmail.PublishVersionTx(conn, draft.Id, 7, time.Now())
	if err != nil {
		t.Fatalf("publish on postgres: %v", err)
	}
	reloaded, err := inboxmail.GetVersionByIDTx(conn, published.Id)
	if err != nil {
		t.Fatalf("reload version on postgres: %v", err)
	}
	if len(reloaded.Blocks) != 1 || reloaded.Blocks[0].Type != "badge" {
		t.Fatalf("blocks JSON did not round-trip on postgres: %#v", reloaded.Blocks)
	}
	wantHash, err := inboxmail.HashVersionContent(reloaded.Content())
	if err != nil {
		t.Fatalf("hash reloaded content: %v", err)
	}
	if reloaded.ContentHash != wantHash || published.ContentHash != wantHash {
		t.Fatalf("content hash mismatch on postgres: published=%q reloaded=%q want=%q", published.ContentHash, reloaded.ContentHash, wantHash)
	}
	if err := inboxmail.UpdateDraftContentTx(conn, published.Id, content); !errors.Is(err, inboxmail.ErrPublishedVersionImmutable) {
		t.Fatalf("published version update on postgres error = %v, want ErrPublishedVersionImmutable", err)
	}

	// 草稿更新走 Updates(map) + 预序列化 JSON（GORM 的 map 更新不应用 serializer:json），
	// 必须在 PG 上真实写入成功且读回一致（#771/#773 的编辑路径都依赖它）。
	draft2, err := inboxmail.CreateDraftVersionTx(conn, message.Id, content, 1)
	if err != nil {
		t.Fatalf("create second draft on postgres: %v", err)
	}
	updatedContent := content
	updatedContent.BodyMarkdown = "**更新正文**"
	updatedContent.Blocks = []inboxmail.Block{{Type: "badge", Payload: json.RawMessage(`{"badgeCode":"second"}`)}}
	if err := inboxmail.UpdateDraftContentTx(conn, draft2.Id, updatedContent); err != nil {
		t.Fatalf("update draft on postgres: %v", err)
	}
	reloadedDraft, err := inboxmail.GetVersionByIDTx(conn, draft2.Id)
	if err != nil {
		t.Fatalf("reload updated draft on postgres: %v", err)
	}
	updatedHash, err := inboxmail.HashVersionContent(updatedContent)
	if err != nil {
		t.Fatalf("hash updated content: %v", err)
	}
	if reloadedDraft.BodyMarkdown != "**更新正文**" || len(reloadedDraft.Blocks) != 1 ||
		reloadedDraft.Blocks[0].Type != "badge" || reloadedDraft.ContentHash != updatedHash {
		t.Fatalf("draft update did not round-trip on postgres: %#v (want hash %q)", reloadedDraft, updatedHash)
	}

	// Campaign 的 Audience/Presentation JSON 信封在 PG 上可写可读。
	campaign := inboxmail.CampaignEntity{
		MessageId:        message.Id,
		MessageVersionId: published.Id,
		Name:             "PG 活动",
		Status:           inboxmail.CampaignStatusDraft,
		ScheduleType:     inboxmail.ScheduleTypeImmediate,
		Audience:         inboxmail.AudienceSpec{Kind: "all_users", Payload: json.RawMessage(`{"activeWithinDays":30}`)},
		Presentation:     inboxmail.PresentationSpec{PopupEnabled: true, PopupDelaySeconds: 5},
	}
	if err := conn.Create(&campaign).Error; err != nil {
		t.Fatalf("create campaign on postgres: %v", err)
	}
	var storedCampaign inboxmail.CampaignEntity
	if err := conn.Where("id = ?", campaign.Id).Take(&storedCampaign).Error; err != nil {
		t.Fatalf("reload campaign on postgres: %v", err)
	}
	if storedCampaign.Audience.Kind != "all_users" || !storedCampaign.Presentation.PopupEnabled ||
		storedCampaign.Presentation.PopupDelaySeconds != 5 {
		t.Fatalf("campaign JSON envelopes did not round-trip on postgres: %#v", storedCampaign)
	}

	// 用户生命周期边界（#787 接入）：匿名化在 PG 方言下可执行且保留事实行。
	deliveries := []inboxmail.DeliveryEntity{
		{UserId: 501, MessageId: message.Id, MessageVersionId: published.Id, CampaignId: campaign.Id,
			DedupeKey: inboxmail.CampaignDedupeKey(campaign.Id, published.VersionNo, 501), Status: inboxmail.DeliveryStatusDelivered},
		{UserId: 502, MessageId: message.Id, MessageVersionId: published.Id, CampaignId: campaign.Id,
			DedupeKey: inboxmail.CampaignDedupeKey(campaign.Id, published.VersionNo, 502), Status: inboxmail.DeliveryStatusDelivered},
	}
	if err := conn.Create(&deliveries).Error; err != nil {
		t.Fatalf("create deliveries on postgres: %v", err)
	}
	if err := inboxmail.AnonymizeUserDataTx(conn, 501); err != nil {
		t.Fatalf("anonymize on postgres: %v", err)
	}
	var anonymized inboxmail.DeliveryEntity
	if err := conn.Where("id = ?", deliveries[0].Id).Take(&anonymized).Error; err != nil {
		t.Fatalf("reload anonymized delivery on postgres: %v", err)
	}
	if anonymized.UserId != inboxmail.AnonymizedUserID || anonymized.DedupeKey != "anonymized:delivery:"+strconv.FormatUint(deliveries[0].Id, 10) {
		t.Fatalf("anonymized delivery on postgres = %#v", anonymized)
	}
	if err := inboxmail.DeleteUserDataTx(conn, 502); err != nil {
		t.Fatalf("delete user data on postgres: %v", err)
	}
	var remaining int64
	if err := conn.Model(&inboxmail.DeliveryEntity{}).Where("id = ?", deliveries[1].Id).Count(&remaining).Error; err != nil {
		t.Fatalf("count deleted delivery on postgres: %v", err)
	}
	if remaining != 0 {
		t.Fatalf("deleted delivery still present on postgres: count=%d", remaining)
	}
}

func assertInboxTables(t *testing.T, conn *gorm.DB) {
	t.Helper()
	for _, table := range inboxTableNames {
		if !conn.Migrator().HasTable(table) {
			t.Errorf("table %q missing after migration", table)
		}
	}
}

// assertInboxUniqueConstraints 用真实插入验证三个幂等约束，而不是只看索引名：
// 约束名/方言差异不会让「重复发信/重复发奖」漏网。
func assertInboxUniqueConstraints(t *testing.T, conn *gorm.DB, dialect string) {
	t.Helper()
	delivery := inboxmail.DeliveryEntity{
		UserId: 601, MessageId: 1, MessageVersionId: 1, CampaignId: 1,
		DedupeKey: inboxmail.CampaignDedupeKey(1, 1, 601), Status: inboxmail.DeliveryStatusDelivered,
	}
	if err := conn.Create(&delivery).Error; err != nil {
		t.Fatalf("[%s] create delivery: %v", dialect, err)
	}
	duplicate := inboxmail.DeliveryEntity{
		UserId: 601, MessageId: 1, MessageVersionId: 1, CampaignId: 1,
		DedupeKey: inboxmail.CampaignDedupeKey(1, 1, 601), Status: inboxmail.DeliveryStatusDelivered,
	}
	if err := conn.Create(&duplicate).Error; !errors.Is(err, gorm.ErrDuplicatedKey) {
		t.Fatalf("[%s] duplicate dedupe_key error = %v, want gorm.ErrDuplicatedKey", dialect, err)
	}
	claim := inboxmail.ClaimEntity{
		DeliveryId: delivery.Id, AttachmentId: 9, UserId: 601, CampaignId: 1,
		Handler: "badge", SourceKey: inboxmail.ClaimSourceKey("badge", "welcome_2026", delivery.Id),
		Status: inboxmail.ClaimStatusGranted,
	}
	if err := conn.Create(&claim).Error; err != nil {
		t.Fatalf("[%s] create claim: %v", dialect, err)
	}
	if err := conn.Create(&inboxmail.ClaimEntity{
		DeliveryId: delivery.Id, AttachmentId: 9, UserId: 601, CampaignId: 1,
		Handler: "badge", SourceKey: inboxmail.ClaimSourceKey("badge", "other", delivery.Id),
		Status: inboxmail.ClaimStatusPending,
	}).Error; !errors.Is(err, gorm.ErrDuplicatedKey) {
		t.Fatalf("[%s] duplicate (delivery_id, attachment_id) error = %v, want gorm.ErrDuplicatedKey", dialect, err)
	}
	if err := conn.Create(&inboxmail.ClaimEntity{
		DeliveryId: delivery.Id, AttachmentId: 10, UserId: 601, CampaignId: 1,
		Handler: "badge", SourceKey: claim.SourceKey, Status: inboxmail.ClaimStatusPending,
	}).Error; !errors.Is(err, gorm.ErrDuplicatedKey) {
		t.Fatalf("[%s] duplicate source_key error = %v, want gorm.ErrDuplicatedKey", dialect, err)
	}

	// 幂等键的 NOT NULL 是唯一约束的前提：显式 NULL 必须被数据库拒绝，
	// 否则「每人一封/每附件一次」会从 NULL 绕过（Go 结构体零值总是 ''，
	// 只有裸 SQL 能构造该场景）。
	if err := conn.Exec(
		`INSERT INTO inbox_delivery (user_id, message_id, message_version_id, dedupe_key) VALUES (?, ?, ?, ?)`,
		602, 1, 1, nil,
	).Error; err == nil {
		t.Fatalf("[%s] NULL dedupe_key accepted by inbox_delivery", dialect)
	}
	if err := conn.Exec(
		`INSERT INTO inbox_claim (delivery_id, attachment_id, user_id, source_key) VALUES (?, ?, ?, ?)`,
		999, 999, 602, nil,
	).Error; err == nil {
		t.Fatalf("[%s] NULL source_key accepted by inbox_claim", dialect)
	}
}

func tableDDL(t *testing.T, conn *gorm.DB, table string) string {
	t.Helper()
	var ddl struct{ SQL string }
	if err := conn.Raw("SELECT sql FROM sqlite_master WHERE type = 'table' AND name = ?", table).Scan(&ddl).Error; err != nil {
		t.Fatalf("read DDL for %s: %v", table, err)
	}
	return ddl.SQL
}
