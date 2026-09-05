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

// DeleteByEndpoint 删除归属指定用户的单条订阅（unsubscribe / 推送端点
// 404/410 失效 / 结构性非法端点自愈）。删除谓词同时限定 user_id 与
// endpoint：endpoint 可能经 Upsert 在快照与删除之间被其他账号接管，
// 不带 owner 的全局删除会误删新归属者的有效订阅（review P2）。
// 幂等：无匹配行时静默成功。
func DeleteByEndpoint(endpoint string, userId uint64) error {
	return builder().
		Where(queryopt.Eq("endpoint", endpoint)).
		Where(queryopt.Eq("user_id", userId)).
		Delete(&Entity{}).Error
}

// countByUser 返回用户当前订阅行数。
func countByUser(userId uint64) int64 {
	var total int64
	builder().Where(queryopt.Eq("user_id", userId)).Count(&total)
	return total
}

// UpsertCapped 在单用户订阅数不超过 maxPerUser 的前提下按 endpoint upsert
// （review P1 fan-out 无界防护：每浏览器一条订阅，RateLimit 只能限速不能
// 限总量，恶意客户端可无界堆积使 worker 串行 fan-out 失去上界）。endpoint
// 已存在（本人所有 = 刷新，他人所有 = 换账号归属收敛）时不新增行、不淘汰；
// 仅当 endpoint 全新且用户行数已达上限时按 id 升序淘汰最旧行再插入。
// 返回淘汰行数；并发竞争下可能短暂超限，下次写入收敛。
func UpsertCapped(userId uint64, endpoint, p256dh, auth, lang string, maxPerUser int) (int64, error) {
	var endpointRows int64
	builder().Where(queryopt.Eq("endpoint", endpoint)).Count(&endpointRows)
	if endpointRows == 0 && maxPerUser > 0 {
		if over := countByUser(userId) - int64(maxPerUser) + 1; over > 0 {
			var oldest []uint64
			builder().
				Where(queryopt.Eq("user_id", userId)).
				Order(queryopt.Asc("id")).
				Limit(int(over)).
				Pluck("id", &oldest)
			if len(oldest) > 0 {
				res := builder().
					Where(queryopt.Eq("user_id", userId)).
					Where("id IN ?", oldest).
					Delete(&Entity{})
				if res.Error != nil {
					return 0, res.Error
				}
				return res.RowsAffected, Upsert(userId, endpoint, p256dh, auth, lang)
			}
		}
	}
	return 0, Upsert(userId, endpoint, p256dh, auth, lang)
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
