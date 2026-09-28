package inboxmail

import (
	"strings"
	"testing"

	"gorm.io/gorm"
)

func seedUserData(t *testing.T, tx *gorm.DB, userID uint64) (DeliveryEntity, ClaimEntity) {
	t.Helper()
	delivery := createDelivery(t, tx, userID, CampaignDedupeKey(userID, 1, userID))
	claim := ClaimEntity{
		DeliveryId: delivery.Id, AttachmentId: 5, UserId: userID, CampaignId: 1,
		Handler: "badge", SourceKey: ClaimSourceKey("badge", "welcome_2026", delivery.Id),
		Status: ClaimStatusGranted,
	}
	if err := tx.Create(&claim).Error; err != nil {
		t.Fatalf("create claim: %v", err)
	}
	return delivery, claim
}

func TestDeleteUserDataRemovesOnlyTargetUser(t *testing.T) {
	tx := openTestDB(t)
	targetDelivery, targetClaim := seedUserData(t, tx, 21)
	otherDelivery, otherClaim := seedUserData(t, tx, 22)

	if err := DeleteUserDataTx(tx, 21); err != nil {
		t.Fatalf("DeleteUserDataTx: %v", err)
	}
	var deliveryCount, claimCount int64
	if err := tx.Model(&DeliveryEntity{}).Where("id = ?", targetDelivery.Id).Count(&deliveryCount).Error; err != nil {
		t.Fatalf("count target delivery: %v", err)
	}
	if err := tx.Model(&ClaimEntity{}).Where("id = ?", targetClaim.Id).Count(&claimCount).Error; err != nil {
		t.Fatalf("count target claim: %v", err)
	}
	if deliveryCount != 0 || claimCount != 0 {
		t.Fatalf("target user rows survive delete: deliveries=%d claims=%d", deliveryCount, claimCount)
	}
	if err := tx.Model(&DeliveryEntity{}).Where("id = ?", otherDelivery.Id).Count(&deliveryCount).Error; err != nil {
		t.Fatalf("count other delivery: %v", err)
	}
	if err := tx.Model(&ClaimEntity{}).Where("id = ?", otherClaim.Id).Count(&claimCount).Error; err != nil {
		t.Fatalf("count other claim: %v", err)
	}
	if deliveryCount != 1 || claimCount != 1 {
		t.Fatalf("other user rows affected by delete: deliveries=%d claims=%d", deliveryCount, claimCount)
	}
	// 幂等：重复清理不留错误。
	if err := DeleteUserDataTx(tx, 21); err != nil {
		t.Fatalf("repeat DeleteUserDataTx: %v", err)
	}
}

// 匿名化保留投递/领取事实行（Campaign 统计分母不变），只剥离用户身份并重建唯一键，
// 供 user lifecycle 选择「保留聚合、移除个人关联」的清理策略。
func TestAnonymizeUserDataKeepsAggregateRows(t *testing.T) {
	tx := openTestDB(t)
	delivery, claim := seedUserData(t, tx, 31)

	if err := AnonymizeUserDataTx(tx, 31); err != nil {
		t.Fatalf("AnonymizeUserDataTx: %v", err)
	}
	var storedDelivery DeliveryEntity
	if err := tx.Where("id = ?", delivery.Id).Take(&storedDelivery).Error; err != nil {
		t.Fatalf("anonymized delivery missing: %v", err)
	}
	if storedDelivery.UserId != AnonymizedUserID {
		t.Fatalf("delivery user_id = %d, want %d", storedDelivery.UserId, AnonymizedUserID)
	}
	if !strings.HasPrefix(storedDelivery.DedupeKey, "anonymized:delivery:") {
		t.Fatalf("delivery dedupe_key not rotated: %q", storedDelivery.DedupeKey)
	}
	var storedClaim ClaimEntity
	if err := tx.Where("id = ?", claim.Id).Take(&storedClaim).Error; err != nil {
		t.Fatalf("anonymized claim missing: %v", err)
	}
	if storedClaim.UserId != AnonymizedUserID {
		t.Fatalf("claim user_id = %d, want %d", storedClaim.UserId, AnonymizedUserID)
	}
	if !strings.HasPrefix(storedClaim.SourceKey, "anonymized:claim:") {
		t.Fatalf("claim source_key not rotated: %q", storedClaim.SourceKey)
	}
	// 幂等：二次匿名化不报错也不改变行数。
	if err := AnonymizeUserDataTx(tx, 31); err != nil {
		t.Fatalf("repeat AnonymizeUserDataTx: %v", err)
	}
	var deliveryCount int64
	if err := tx.Model(&DeliveryEntity{}).Where("id = ?", delivery.Id).Count(&deliveryCount).Error; err != nil {
		t.Fatalf("count anonymized delivery: %v", err)
	}
	if deliveryCount != 1 {
		t.Fatalf("anonymized delivery count = %d, want 1", deliveryCount)
	}
}
