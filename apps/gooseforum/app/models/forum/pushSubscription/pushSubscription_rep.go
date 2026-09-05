package pushSubscription

import (
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/queryopt"
	"gorm.io/gorm/clause"
	"time"
)

// Upsert 按 endpoint 唯一键插入或更新订阅归属。
// 同一浏览器重新授权/换账号登录后 endpoint 不变：冲突时把订阅收敛到
// 当前登录用户并刷新密钥与语言。keys 为空时（仅注销旧归属）不覆盖已有
// 密钥，仅迁移归属。
func Upsert(userId uint64, endpoint string, p256dh string, auth string, lang string) error {
	updates := map[string]any{
		"user_id": userId,
		"lang":    lang,
	}
	if p256dh != "" {
		updates["p256dh"] = p256dh
	}
	if auth != "" {
		updates["auth"] = auth
	}
	return builder().Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "endpoint"}},
		DoUpdates: clause.Assignments(updates),
	}).Create(&Entity{
		UserId:   userId,
		Endpoint: endpoint,
		P256dh:   p256dh,
		Auth:     auth,
		Lang:     lang,
	}).Error
}

// DeleteByUser 删除用户全部订阅（账号注销 anonymize/delete 两 mode 共用）。
func DeleteByUser(userId uint64) error {
	return builder().Where(queryopt.Eq("user_id", userId)).Delete(&Entity{}).Error
}

// DeleteByEndpoint 删除单条订阅（unsubscribe / 推送端点 404/410 失效）。
// 幂等：endpoint 不存在时静默成功。
func DeleteByEndpoint(endpoint string) error {
	return builder().Where(queryopt.Eq("endpoint", endpoint)).Delete(&Entity{}).Error
}

// ListByUser 返回用户全部订阅（推送 worker fan-out 用）。
func ListByUser(userId uint64) (entities []*Entity) {
	builder().
		Where(queryopt.Eq("user_id", userId)).
		Order(queryopt.Desc("id")).
		Find(&entities)
	return
}

// TouchActive 更新订阅最近活跃时间（成功发送推送后调用）。
func TouchActive(endpoint string, now time.Time) error {
	return builder().
		Where(queryopt.Eq("endpoint", endpoint)).
		Update("last_active_at", now).Error
}
