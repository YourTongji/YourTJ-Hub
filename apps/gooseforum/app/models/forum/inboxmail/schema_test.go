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
		Handler: "badge", SourceKey: ClaimSourceKey(delivery.Id, 5),
		Status: ClaimStatusGranted,
	}
	if err := tx.Create(&claim).Error; err != nil {
		t.Fatalf("create claim: %v", err)
	}
	// 同一 (delivery, attachment) 的第二次领取必须被拦下；用不同的字面 source_key
	// 证明拦截者是 (delivery_id, attachment_id) 约束本身，而不是 source_key。
	if err := tx.Create(&ClaimEntity{
		DeliveryId: delivery.Id, AttachmentId: 5, UserId: 11, CampaignId: 1,
		Handler: "badge", SourceKey: "probe:pair-constraint",
		Status: ClaimStatusPending,
	}).Error; !errors.Is(err, gorm.ErrDuplicatedKey) {
		t.Fatalf("duplicate (delivery_id, attachment_id) error = %v, want gorm.ErrDuplicatedKey", err)
	}
	// 同一 source key 出现在不同 (delivery, attachment) 上必须被拦下（handler 层
	// 幂等的最后防线，正常路径不会构造出该组合）。
	if err := tx.Create(&ClaimEntity{
		DeliveryId: delivery.Id, AttachmentId: 6, UserId: 11, CampaignId: 1,
		Handler: "badge", SourceKey: claim.SourceKey, Status: ClaimStatusPending,
	}).Error; !errors.Is(err, gorm.ErrDuplicatedKey) {
		t.Fatalf("duplicate source_key error = %v, want gorm.ErrDuplicatedKey", err)
	}
}

// #775 要求同一事件 occurrence 对不同 Campaign 各自成信：dedupe_key 是全局唯一索引，
// 旧格式（缺 campaign 段）会让第二个 Campaign 的投递撞唯一索引，再被「重复=已投递」
// 的幂等语义静默吞掉；同一 Campaign 内的事件重放仍必须被拦下。
func TestTriggerDeliveriesForTwoCampaignsBothPersist(t *testing.T) {
	tx := openTestDB(t)
	userID := uint64(77)
	for _, campaignID := range []uint64{1, 2} {
		delivery := DeliveryEntity{
			UserId: userID, MessageId: 1, MessageVersionId: 1, CampaignId: campaignID,
			SourceType: DeliverySourceTrigger, TriggerEvent: "user.registered", TriggerEventId: "evt-9",
			DedupeKey: TriggerDedupeKey(campaignID, "user.registered", "evt-9", userID),
			Status:    DeliveryStatusDelivered,
		}
		if err := tx.Create(&delivery).Error; err != nil {
			t.Fatalf("campaign %d trigger delivery rejected: %v", campaignID, err)
		}
		replay := delivery
		replay.Id = 0
		if err := tx.Create(&replay).Error; !errors.Is(err, gorm.ErrDuplicatedKey) {
			t.Fatalf("campaign %d replay error = %v, want gorm.ErrDuplicatedKey", campaignID, err)
		}
	}
	var count int64
	if err := tx.Model(&DeliveryEntity{}).
		Where("user_id = ? AND source_type = ?", userID, DeliverySourceTrigger).
		Count(&count).Error; err != nil {
		t.Fatalf("count trigger deliveries: %v", err)
	}
	if count != 2 {
		t.Fatalf("trigger deliveries = %d, want 2 (one per campaign)", count)
	}
}

// #777 的发布校验只在 (campaign_id, attachment_key) 粒度上保证附件唯一，同一 badge
// code 出现在两个附件行是合法的；同一 delivery 的两个附件必须能各自领取一行，
// 否则第二个附件会被 source_key 唯一约束顶掉（#770 评审定位的领取粒度缺陷）。
func TestSameDeliveryTwoAttachmentsAreBothClaimable(t *testing.T) {
	tx := openTestDB(t)
	delivery := createDelivery(t, tx, 11, CampaignDedupeKey(1, 1, 11))
	for _, attachmentID := range []uint64{5, 6} {
		if err := tx.Create(&ClaimEntity{
			DeliveryId: delivery.Id, AttachmentId: attachmentID, UserId: 11, CampaignId: 1,
			Handler: "badge", SourceKey: ClaimSourceKey(delivery.Id, attachmentID),
			Status: ClaimStatusPending,
		}).Error; err != nil {
			t.Fatalf("claim of attachment %d rejected: %v", attachmentID, err)
		}
	}
	var count int64
	if err := tx.Model(&ClaimEntity{}).Where("delivery_id = ?", delivery.Id).Count(&count).Error; err != nil {
		t.Fatalf("count claims: %v", err)
	}
	if count != 2 {
		t.Fatalf("claims of one delivery = %d, want 2", count)
	}
}

// 四个业务键都必须拒绝空串：GORM 会把 Go 字符串零值写进 INSERT，若只靠唯一索引，
// 第一次漏传会静默写入，第二次才报 gorm.ErrDuplicatedKey——而本仓库把重复键当作
// 「已创建」的幂等成功，收件人 B..N 会被静默跳过、运行却报告成功。CHECK 约束必须让
// 空串立刻以非重复错误失败。
func TestEmptyBusinessKeysRejectedOnSQLite(t *testing.T) {
	tx := openTestDB(t)
	cases := []struct {
		name  string
		model any
	}{
		{"inbox_delivery.dedupe_key", &DeliveryEntity{
			UserId: 1, MessageId: 1, MessageVersionId: 1, CampaignId: 1, Status: DeliveryStatusDelivered}},
		{"inbox_claim.source_key", &ClaimEntity{
			DeliveryId: 1, AttachmentId: 1, UserId: 1, CampaignId: 1, Handler: "badge", Status: ClaimStatusPending}},
		{"inbox_message.code", &MessageEntity{Name: "probe"}},
		{"inbox_campaign_attachment.attachment_key", &CampaignAttachmentEntity{CampaignId: 1, Handler: "badge"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tx.Create(tc.model).Error
			if err == nil {
				t.Fatalf("empty %s accepted by database", tc.name)
			}
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				t.Fatalf("empty %s reported as duplicate; callers would swallow it as idempotent success: %v", tc.name, err)
			}
		})
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
