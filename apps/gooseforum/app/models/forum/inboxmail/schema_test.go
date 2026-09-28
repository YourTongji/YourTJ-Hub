package inboxmail

import (
	"errors"
	"testing"

	"gorm.io/gorm"
)

// 新建库必须一次迁移出全部 7 张表；表名是后续 19 个子 issue 的共享契约。
func TestAllModelsMigrateOnSQLite(t *testing.T) {
	tx := openTestDB(t)
	for _, table := range []string{
		"inbox_message",
		"inbox_message_version",
		"inbox_campaign",
		"inbox_campaign_attachment",
		"inbox_campaign_run",
		"inbox_delivery",
		"inbox_claim",
	} {
		if !tx.Migrator().HasTable(table) {
			t.Fatalf("table %s missing after AutoMigrate", table)
		}
	}
}

// 核心列表查询（信箱游标分页 / 未读 / 归档 / 待领取 / popup 候选 / campaign 维度）
// 依赖这些索引；改索引名或列序会静默拖垮线上列表。
func TestCoreListIndexesExist(t *testing.T) {
	tx := openTestDB(t)
	assertIndexColumns(t, tx, &DeliveryEntity{}, "idx_inbox_delivery_user_id", []string{"user_id", "id"})
	assertIndexColumns(t, tx, &DeliveryEntity{}, "idx_inbox_delivery_user_read_id", []string{"user_id", "is_read", "id"})
	assertIndexColumns(t, tx, &DeliveryEntity{}, "idx_inbox_delivery_user_archived_id", []string{"user_id", "is_archived", "id"})
	assertIndexColumns(t, tx, &DeliveryEntity{}, "idx_inbox_delivery_user_claim_id", []string{"user_id", "claim_state", "id"})
	assertIndexColumns(t, tx, &DeliveryEntity{}, "idx_inbox_delivery_user_popup_id", []string{"user_id", "popup_state", "id"})
	assertIndexColumns(t, tx, &DeliveryEntity{}, "idx_inbox_delivery_campaign_status", []string{"campaign_id", "status"})
	assertIndexColumns(t, tx, &ClaimEntity{}, "idx_inbox_claim_user_id", []string{"user_id", "id"})
	assertIndexColumns(t, tx, &ClaimEntity{}, "idx_inbox_claim_campaign", []string{"campaign_id"})
	assertIndexColumns(t, tx, &ClaimEntity{}, "idx_inbox_claim_status", []string{"status"})
	assertIndexColumns(t, tx, &CampaignRunEntity{}, "idx_inbox_campaign_run_status", []string{"status", "id"})
}

func createDelivery(t *testing.T, tx *gorm.DB, userID uint64, dedupeKey string) DeliveryEntity {
	t.Helper()
	delivery := DeliveryEntity{
		UserId:           userID,
		MessageId:        1,
		MessageVersionId: 1,
		CampaignId:       1,
		DedupeKey:        dedupeKey,
		Status:           DeliveryStatusDelivered,
		PopupState:       PopupStatePending,
		ClaimState:       ClaimStateNone,
	}
	if err := tx.Create(&delivery).Error; err != nil {
		t.Fatalf("create delivery: %v", err)
	}
	return delivery
}

// dedupe_key / source_key / (delivery_id, attachment_id) 必须由数据库唯一约束兜底：
// Worker 重试、事件重放、双端并发领取都依赖这三个约束（epic #769 验收标准）。
func TestUniqueConstraintsRejectDuplicates(t *testing.T) {
	tx := openTestDB(t)

	delivery := createDelivery(t, tx, 11, CampaignDedupeKey(1, 1, 11))
	if err := tx.Create(&DeliveryEntity{
		UserId: 11, MessageId: 1, MessageVersionId: 1, CampaignId: 1,
		DedupeKey: CampaignDedupeKey(1, 1, 11), Status: DeliveryStatusDelivered,
	}).Error; !errors.Is(err, gorm.ErrDuplicatedKey) {
		t.Fatalf("duplicate dedupe_key error = %v, want gorm.ErrDuplicatedKey", err)
	}

	claim := ClaimEntity{
		DeliveryId: delivery.Id, AttachmentId: 5, UserId: 11, CampaignId: 1,
		Handler: "badge", SourceKey: ClaimSourceKey("badge", "welcome_2026", delivery.Id),
		Status: ClaimStatusGranted,
	}
	if err := tx.Create(&claim).Error; err != nil {
		t.Fatalf("create claim: %v", err)
	}
	// 同一 (delivery, attachment) 的第二次领取（不同 source key）必须被拦下。
	if err := tx.Create(&ClaimEntity{
		DeliveryId: delivery.Id, AttachmentId: 5, UserId: 11, CampaignId: 1,
		Handler: "badge", SourceKey: ClaimSourceKey("badge", "other_code", delivery.Id),
		Status: ClaimStatusPending,
	}).Error; !errors.Is(err, gorm.ErrDuplicatedKey) {
		t.Fatalf("duplicate (delivery_id, attachment_id) error = %v, want gorm.ErrDuplicatedKey", err)
	}
	// 同一 Reward Handler source key 的第二次领取（不同 delivery）必须被拦下。
	other := createDelivery(t, tx, 11, CampaignDedupeKey(1, 1, 12))
	if err := tx.Create(&ClaimEntity{
		DeliveryId: other.Id, AttachmentId: 5, UserId: 12, CampaignId: 1,
		Handler: "badge", SourceKey: claim.SourceKey, Status: ClaimStatusPending,
	}).Error; !errors.Is(err, gorm.ErrDuplicatedKey) {
		t.Fatalf("duplicate source_key error = %v, want gorm.ErrDuplicatedKey", err)
	}
	// 同 delivery 的不同 attachment 必须允许。
	if err := tx.Create(&ClaimEntity{
		DeliveryId: delivery.Id, AttachmentId: 6, UserId: 11, CampaignId: 1,
		Handler: "badge", SourceKey: ClaimSourceKey("badge", "second", delivery.Id),
		Status: ClaimStatusPending,
	}).Error; err != nil {
		t.Fatalf("distinct attachment claim rejected: %v", err)
	}
}

func TestCampaignAndAttachmentNaturalKeys(t *testing.T) {
	tx := openTestDB(t)
	campaign := CampaignEntity{
		MessageId: 1, MessageVersionId: 1, Name: "开学提醒",
		Status: CampaignStatusDraft, ScheduleType: ScheduleTypeImmediate,
	}
	if err := tx.Create(&campaign).Error; err != nil {
		t.Fatalf("create campaign: %v", err)
	}
	attachment := CampaignAttachmentEntity{CampaignId: campaign.Id, AttachmentKey: "welcome-badge", Handler: "badge", Name: "欢迎徽章"}
	if err := tx.Create(&attachment).Error; err != nil {
		t.Fatalf("create attachment: %v", err)
	}
	if err := tx.Create(&CampaignAttachmentEntity{CampaignId: campaign.Id, AttachmentKey: "welcome-badge", Handler: "badge"}).Error; !errors.Is(err, gorm.ErrDuplicatedKey) {
		t.Fatalf("duplicate attachment key error = %v, want gorm.ErrDuplicatedKey", err)
	}
	run := CampaignRunEntity{CampaignId: campaign.Id, RunNo: 1, Status: RunStatusPending}
	if err := tx.Create(&run).Error; err != nil {
		t.Fatalf("create campaign run: %v", err)
	}
	if err := tx.Create(&CampaignRunEntity{CampaignId: campaign.Id, RunNo: 1, Status: RunStatusPending}).Error; !errors.Is(err, gorm.ErrDuplicatedKey) {
		t.Fatalf("duplicate run_no error = %v, want gorm.ErrDuplicatedKey", err)
	}
}
