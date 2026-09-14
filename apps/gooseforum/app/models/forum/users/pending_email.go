package users

import (
	"errors"
	"time"
)

// ErrPendingEmailSwitchStale 表示条件切换未命中：暂存邮箱不匹配、已被另一条
// 链接完成切换，或已超出占用窗口（视为用户放弃）。调用方按“链接无效”处理。
var ErrPendingEmailSwitchStale = errors.New("pending email switch stale")

// PendingEmailSwitchResult 携带切换完成后的用户实体与被替换的旧邮箱，
// 供缓存刷新与旧邮箱最终变更通知使用。
type PendingEmailSwitchResult struct {
	User     EntityComplete
	OldEmail string
}

// ExistEmailOrFreshPending 检查邮箱是否被占用：当前 email 命中，或命中任一
// 账号仍在占用窗口内的换绑暂存（issue #678：暂存期间新邮箱同样不可注册，
// 防止验证前被抢注导致切换时唯一索引冲突）。窗口外的过期暂存不占用。
// excludeUserID 用于换绑自查重时排除本人（重新暂存同一邮箱 = 再次发起换绑）。
func ExistEmailOrFreshPending(email string, excludeUserID uint64) bool {
	var id uint64
	cutoff := time.Now().Add(-PendingEmailWindow)
	return builder().Select("1").
		Where("email = ? OR (pending_email = ? AND pending_email_at >= ?)", email, email, cutoff).
		Where("id <> ?", excludeUserID).
		Limit(1).Scan(&id).RowsAffected > 0
}

// ClearExpiredPendingEmails 清掉与 email 相同且已超出占用窗口的换绑暂存
// （过期 = 用户放弃）。部分唯一索引 uniq_users_pending_email_nonempty 不区分
// 新旧暂存，若不先清掉过期行，新账号对同一邮箱的合法暂存会被索引拒绝。
// 只按暂存值定向清理，影响行数为 0 是常态。
func ClearExpiredPendingEmails(email string) error {
	cutoff := time.Now().Add(-PendingEmailWindow)
	result := builder().
		Where("pending_email = ? AND pending_email_at < ?", email, cutoff).
		Updates(map[string]any{"pending_email": "", "pending_email_at": nil})
	return result.Error
}

// CompletePendingEmailSwitch 以单条条件 UPDATE 原子完成两阶段换绑的第二阶段：
// email ← pending_email、清空暂存、置为已激活并刷新 email_changed_at（24 小时
// 找回冷静期从真正切换时起算）。WHERE 同时约束暂存值与新鲜度，保证与并发
// 的重复点击 / 过期链接 / 抢注注册互斥；未命中返回 ErrPendingEmailSwitchStale。
func CompletePendingEmailSwitch(userID uint64, pendingEmail string, now time.Time) (PendingEmailSwitchResult, error) {
	var before EntityComplete
	if err := builder().Where(pid, userID).First(&before).Error; err != nil {
		return PendingEmailSwitchResult{}, err
	}

	result := builder().Where(pid, userID).
		Where("pending_email = ?", pendingEmail).
		Where("pending_email_at >= ?", now.Add(-PendingEmailWindow)).
		Updates(map[string]any{
			"email":            pendingEmail,
			"pending_email":    "",
			"pending_email_at": nil,
			"is_activated":     ActivationSuccess,
			"activated_at":     now,
			"email_changed_at": now,
			"updated_at":       now,
		})
	if result.Error != nil {
		return PendingEmailSwitchResult{}, result.Error
	}
	if result.RowsAffected != 1 {
		return PendingEmailSwitchResult{}, ErrPendingEmailSwitchStale
	}

	var after EntityComplete
	if err := builder().Where(pid, userID).First(&after).Error; err != nil {
		return PendingEmailSwitchResult{}, err
	}
	return PendingEmailSwitchResult{User: after, OldEmail: before.Email}, nil
}
