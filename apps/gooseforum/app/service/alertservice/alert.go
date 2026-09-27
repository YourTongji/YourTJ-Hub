// Package alertservice 实现站内运维告警的聚合编排（issue #855）：
// 认领告警接收人（SiteManager 及 Admin 超级集角色）、按来源去重并逐人发送
// 系统站内通知。告警链路是 best-effort：任何失败只记日志，绝不影响调用方
// （如排课同步流程）的错误路径与返回。
package alertservice

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/localcache"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/cacheconfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pk"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/role"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/notificationservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/permission"
)

// alertWindow 是每位接收人同一来源（受众）告警的去重窗口：成功发送后不再重复提醒，
// 避免凭证持续失效时 cron/手动重试高频刷屏。
const alertWindow = 6 * time.Hour

// now 是可注入时钟：生产走 time.Now，测试可覆盖以控制记录的告警时间。
var now = time.Now

// pkAlertCache 记录 "最近一次告警的时间点" 标记；条目存活期 = alertWindow，
// 存活期内同 key 再次告警直接跳过。key 见 pkAlertKey。
var pkAlertCache = localcache.Cache[time.Time]{MaxEntries: cacheconfig.Current().RolePermission}

// pkAlertKey 按受众和接收人隔离；发送失败者下次可重试，已成功者不重复打扰。
func pkAlertKey(audience string, userID uint64) string {
	return fmt.Sprintf("pk_alert:%s:%d", audience, userID)
}

// audienceLabel 把原始受众字符串转成中文标签；空/未知值回落「本科」
// （与 pkservice.audienceLabel 同口径）。
func audienceLabel(audience string) string {
	if pk.DefaultAudience(audience) == pk.AudienceGraduate {
		return "研究生"
	}
	return "本科"
}

// NotifyPkSyncFailed 在排课同步失败（fetchlog 标记 failed）时向拥有
// SiteManager 权限（含 Admin 超级集）的全部活跃用户发送站内系统通知。
//
//   - 去重：每位接收人同一来源（受众）在 alertWindow（6h）窗口内只成功发送一次；
//   - best-effort：不 panic、不向调用方返回错误，单项失败仅 slog.Warn。
//
// 注意：message 来自同步错误文本（err.Error()），上游已脱敏，不包含凭证原文。
func NotifyPkSyncFailed(audience string, message string) {
	// 告警是旁路职责：即使下面某步 panic 也不能让同步流程跟着崩。
	defer func() {
		if r := recover(); r != nil {
			slog.Warn("alertservice: NotifyPkSyncFailed recovered", "audience", audience, "panic", r)
		}
	}()

	audience = string(pk.DefaultAudience(audience))

	// 认领接收人：具备 SiteManager 权限的角色（CheckRole 对 Admin 角色视为超集）。
	roleIds := make([]uint64, 0, 4)
	for _, r := range role.AllEffective() {
		if r == nil {
			continue
		}
		if permission.CheckRole(r.Id, permission.SiteManager) {
			roleIds = append(roleIds, r.Id)
		}
	}
	userIDs := users.GetActiveUserIdsByRoleIds(roleIds)

	title := "一系统排课同步失败"
	content := fmt.Sprintf("【%s】同步失败：%s", audienceLabel(audience), message)

	for _, userID := range userIDs {
		if userID == 0 {
			continue
		}
		// GetOrLoadE 的 singleflight 将查询、发送和成功标记作为同一次操作；
		// loader 失败不缓存，部分成功时只重试未送达者。缓存为进程内，重启后重置。
		_, err := pkAlertCache.GetOrLoadE(pkAlertKey(audience, userID), func() (time.Time, error) {
			if err := notificationservice.SendSystemAlert(userID, title, content); err != nil {
				return time.Time{}, err
			}
			return now(), nil
		}, alertWindow)
		if err != nil {
			slog.Warn("alertservice: send pk sync failure alert failed", "userId", userID, "err", err)
		}
	}
}
