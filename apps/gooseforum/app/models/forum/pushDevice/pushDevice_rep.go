package pushDevice

import (
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/queryopt"
	"gorm.io/gorm/clause"
)

// Upsert 按 token 唯一键插入或更新设备注册归属。
// 同一设备重新授权/换账号登录后 token 不变：冲突时把注册收敛到当前登录用户
// 并刷新平台与最近注册时间（created_at 保持不变）。
func Upsert(userId uint64, platform string, token string, now time.Time, providers ...string) error {
	provider := ""
	if len(providers) > 0 {
		provider = providers[0]
	}
	return builder().Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "token"}},
		DoUpdates: clause.Assignments(map[string]any{
			"user_id":            userId,
			"platform":           platform,
			"provider":           provider,
			"last_registered_at": now,
		}),
	}).Create(&Entity{
		UserId:           userId,
		Platform:         platform,
		Provider:         provider,
		Token:            token,
		LastRegisteredAt: now,
	}).Error
}

// UpsertCapped 在单用户设备数不超过 maxPerUser 的前提下按 token upsert
// （fan-out 无界防护，与 pushSubscription.UpsertCapped 同口径：RateLimit
// 只能限速不能限总量，恶意客户端可无界堆积使 worker 串行 fan-out 失去上界）。
// token 已存在（本人所有 = 刷新，他人所有 = 换账号归属收敛）时不新增行、
// 不淘汰；仅当 token 全新且用户行数已达上限时按 id 升序淘汰最旧行再插入。
// 返回淘汰行数；并发竞争下可能短暂超限，下次写入收敛。
func UpsertCapped(userId uint64, platform string, token string, maxPerUser int, now time.Time, providers ...string) (int64, error) {
	var tokenRows int64
	builder().Where(queryopt.Eq("token", token)).Count(&tokenRows)
	if tokenRows == 0 && maxPerUser > 0 {
		if over := CountByUser(userId) - int64(maxPerUser) + 1; over > 0 {
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
				return res.RowsAffected, Upsert(userId, platform, token, now, providers...)
			}
		}
	}
	return 0, Upsert(userId, platform, token, now, providers...)
}

// DeleteByUser 删除用户全部设备注册（账号注销 anonymize/delete 两 mode 共用）。
func DeleteByUser(userId uint64) error {
	return builder().Where(queryopt.Eq("user_id", userId)).Delete(&Entity{}).Error
}

// DeleteByToken 删除归属指定用户的单条设备注册（unregister / 推送服务
// 410 BadDeviceToken / 404 UNREGISTERED 失效清理）。删除谓词同时限定
// user_id 与 token：token 可能经 Upsert 在快照与删除之间被其他账号接管，
// 不带 owner 的全局删除会误删新归属者的有效注册。幂等：无匹配行时静默成功。
func DeleteByToken(token string, userId uint64) error {
	return builder().
		Where(queryopt.Eq("token", token)).
		Where(queryopt.Eq("user_id", userId)).
		Delete(&Entity{}).Error
}

// ListByUser 返回用户全部设备注册（原生推送 worker fan-out 用）。
func ListByUser(userId uint64) (entities []*Entity) {
	builder().
		Where(queryopt.Eq("user_id", userId)).
		Order(queryopt.Desc("id")).
		Find(&entities)
	return
}

// CountByUser 返回用户当前设备注册行数。
func CountByUser(userId uint64) int64 {
	var total int64
	builder().Where(queryopt.Eq("user_id", userId)).Count(&total)
	return total
}
