package inboxmail

import (
	"fmt"

	"gorm.io/gorm"
)

// 用户生命周期 / 数据保留的清理边界（issue #770 预留，#787 落地策略并接入
// userservice.CloseAccount）：
//
//   - DeleteUserDataTx：硬删除该用户的投递与领取事实（隐私优先，默认清理策略）；
//   - AnonymizeUserDataTx：保留投递/领取行（Campaign 统计分母不缩水），只剥离
//     用户身份并把唯一 key 轮换为匿名值。
//
// 两个函数都是幂等的，必须由调用方开启事务（与账号关闭的其它域写入同事务提交）。
// 内容侧（inbox_message / inbox_message_version / inbox_campaign*）不属于单个
// 用户，不在此边界内；其保留期限由 #787 的平台级清理任务负责。

const (
	anonymizedDeliveryKeyPrefix = "anonymized:delivery:"
	anonymizedClaimKeyPrefix    = "anonymized:claim:"
)

// DeleteUserDataTx 物理删除 userID 名下的全部投递与领取行。
// userID 为 0（匿名化哨兵）时不执行任何操作，避免误删已匿名化的聚合行。
func DeleteUserDataTx(tx *gorm.DB, userID uint64) error {
	if userID == 0 {
		return nil
	}
	if err := tx.Exec("DELETE FROM "+claimTableName+" WHERE user_id = ?", userID).Error; err != nil {
		return fmt.Errorf("inboxmail: delete user claims: %w", err)
	}
	if err := tx.Exec("DELETE FROM "+deliveryTableName+" WHERE user_id = ?", userID).Error; err != nil {
		return fmt.Errorf("inboxmail: delete user deliveries: %w", err)
	}
	return nil
}

// AnonymizeUserDataTx 保留 userID 的投递与领取行，把 user_id 置为 AnonymizedUserID
// 并轮换唯一 key（dedupe_key / source_key 含用户维度，必须同时改写才不会与新
// 用户的 key 冲突）。重复调用不会命中任何行，幂等。
//
// CAST(id AS TEXT) 与 || 在 PostgreSQL 与 SQLite 上语义一致；右侧显式转 text
// 保证绑定参数的文本类型推断无歧义。
func AnonymizeUserDataTx(tx *gorm.DB, userID uint64) error {
	if userID == 0 {
		return nil
	}
	if err := tx.Exec(
		"UPDATE "+deliveryTableName+" SET user_id = ?, dedupe_key = ? || CAST(id AS TEXT), updated_at = CURRENT_TIMESTAMP WHERE user_id = ?",
		AnonymizedUserID, anonymizedDeliveryKeyPrefix, userID,
	).Error; err != nil {
		return fmt.Errorf("inboxmail: anonymize user deliveries: %w", err)
	}
	if err := tx.Exec(
		"UPDATE "+claimTableName+" SET user_id = ?, source_key = ? || CAST(id AS TEXT), updated_at = CURRENT_TIMESTAMP WHERE user_id = ?",
		AnonymizedUserID, anonymizedClaimKeyPrefix, userID,
	).Error; err != nil {
		return fmt.Errorf("inboxmail: anonymize user claims: %w", err)
	}
	return nil
}
