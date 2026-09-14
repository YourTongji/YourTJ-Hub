package users

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

// ErrPendingEmailSwitchStale 表示条件切换未命中：暂存邮箱不匹配、已被另一条
// 链接完成切换，或已超出占用窗口（视为用户放弃）。调用方按“链接无效”处理。
var ErrPendingEmailSwitchStale = errors.New("pending email switch stale")

// ErrEmailOccupied 注册事务内占用复查命中（issue #678 review：注册与换绑暂存
// 并发竞争同一邮箱的跨列残余窗口）。调用方按“邮箱已占用”失败处理。
var ErrEmailOccupied = errors.New("email occupied")

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

// EmailOrFreshPendingOccupiedTx 是 ExistEmailOrFreshPending 的事务内复查变体
// （issue #678 review P2）：注册在事务内、写入前再查一次，收窄「外层检查通过
// 后、插行前，他账号对同一邮箱完成换绑暂存提交」的跨列竞争窗口。
func EmailOrFreshPendingOccupiedTx(tx *gorm.DB, email string, excludeUserID uint64) bool {
	var id uint64
	cutoff := tx.NowFunc().Add(-PendingEmailWindow)
	return tx.Table(tableName).Select("1").
		Where("email = ? OR (pending_email = ? AND pending_email_at >= ?)", email, email, cutoff).
		Where("id <> ?", excludeUserID).
		Limit(1).Scan(&id).RowsAffected > 0
}

// ExistEmailExcluding 检查邮箱是否已被其他账号的当前 email 占用（不含暂存）。
// 换绑暂存写入后的补偿复查使用：并发注册若在暂存落库后占用同一邮箱，
// 该暂存在切换时必撞 email 唯一索引，立即撤销并把邮箱还给占用者。
func ExistEmailExcluding(email string, excludeUserID uint64) bool {
	var id uint64
	return builder().Select("1").
		Where("email = ?", email).
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

// StagePendingEmail 写入换绑暂存（两阶段第一阶段）。定向列更新而非全行
// Save：一是不与并发的密码变更（ApplyPasswordChange CAS）互相回写，
// 二是天然受部分唯一索引约束——并发对同一邮箱的暂存由数据库拒绝。
// 调用方负责用户缓存失效。
func StagePendingEmail(userID uint64, pendingEmail string, now time.Time) error {
	return builder().Where(pid, userID).Updates(map[string]any{
		"pending_email":    pendingEmail,
		"pending_email_at": now,
	}).Error
}

// ActivateCurrentEmail 以 CAS 条件更新激活当前邮箱，并清除换绑暂存
// （用户点击发往当前邮箱的激活链接 = 放弃换绑，issue #678 review P2）。
// WHERE 同时约束读取时看到的 email 与 pending_email：若并发的新邮箱确认
// 切换已先完成（email 已被替换、暂存已清空），本次更新不命中，调用方按
// 链接无效处理——绝不用全行 Save 把已切换的邮箱回写成旧值。
// 返回 applied=false 表示 CAS 未命中（状态已被并发操作改变）。
func ActivateCurrentEmail(userID uint64, email, pendingEmail string, now time.Time) (applied bool, err error) {
	result := builder().Where(pid, userID).
		Where("email = ?", email).
		Where("pending_email = ?", pendingEmail).
		Updates(map[string]any{
			"is_activated":     ActivationSuccess,
			"activated_at":     now,
			"pending_email":    "",
			"pending_email_at": nil,
			"updated_at":       now,
		})
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected == 1, nil
}

// CancelPendingEmailSwitch 按 CAS 清除指定暂存（补偿路径：并发注册抢先占用
// 目标邮箱后撤销暂存）。WHERE 只约束暂存值，不约束新鲜度——无论窗口内外，
// 指定暂存都应被清掉；未命中（已被切换/清除）视为已完成，不报错。
func CancelPendingEmailSwitch(userID uint64, pendingEmail string) error {
	return builder().Where(pid, userID).
		Where("pending_email = ?", pendingEmail).
		Updates(map[string]any{"pending_email": "", "pending_email_at": nil}).Error
}

// ApplyPasswordChange 用单条 CAS UPDATE 完成密码变更：写入新哈希、
// token_version = expected + 1（吊销全部既有会话与重置令牌），并清除换绑
// 暂存（issue #678 review P1：改密证明账号已收回，已签发的换绑确认链接
// 必须随之失效——激活令牌不绑定 token_version，不能靠自增吊销）。
// WHERE token_version = expected 同时防止两条路径的全行 Save 竞态把并发
// 完成的邮箱切换回写成旧值。未命中返回 gorm.ErrRecordNotFound 语义的
// ErrConcurrentPasswordChange，调用方按“凭据已变化，请重试”处理。
var ErrConcurrentPasswordChange = errors.New("concurrent password change")

func ApplyPasswordChange(userID uint64, passwordHash string, expectedTokenVersion uint64) error {
	result := builder().Where(pid, userID).
		Where("token_version = ?", expectedTokenVersion).
		Updates(map[string]any{
			"password":         passwordHash,
			"token_version":    expectedTokenVersion + 1,
			"pending_email":    "",
			"pending_email_at": nil,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return ErrConcurrentPasswordChange
	}
	return nil
}

// CompletePendingEmailSwitch 以单条条件 UPDATE 原子完成两阶段换绑的第二阶段：
// email ← pending_email、清空暂存、置为已激活并刷新 email_changed_at（24 小时
// 找回冷静期从真正切换时起算）。WHERE 同时约束暂存值与新鲜度，保证与并发
// 的重复点击 / 过期链接 / 抢注注册互斥；未命中返回 ErrPendingEmailSwitchStale。
// 切换前复查目标邮箱的当前占用（issue #678 review P2）：若并发注册已在本
// 暂存之后把目标邮箱注册为自己的当前 email（跨列索引不互斥的残余窗口），
// 本次切换永远不可能成功——撤销暂存并按链接无效返回，把邮箱留给占用者。
func CompletePendingEmailSwitch(userID uint64, pendingEmail string, now time.Time) (PendingEmailSwitchResult, error) {
	if ExistEmailExcluding(pendingEmail, userID) {
		if err := CancelPendingEmailSwitch(userID, pendingEmail); err != nil {
			return PendingEmailSwitchResult{}, err
		}
		return PendingEmailSwitchResult{}, ErrPendingEmailSwitchStale
	}

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
