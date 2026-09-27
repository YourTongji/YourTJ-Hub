package pkservice

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pk"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
)

// maxScheduledSyncDepth 定时同步可向前回溯的学期数上限（与手动同步入口一致，
// 避免破坏性全量重写 + 全量抓取失控）。
const maxScheduledSyncDepth = 8

// scheduledSyncExecutor 定时同步的同步执行体（与手动「立即同步」同一管线）。
// 默认按受众分发到 SyncFromClaim / SyncFromClaimForAudience（含租约校验、分页
// 抓取、时间片重建、课评物化）；测试替换为 stub 以隔离上游抓取。
var scheduledSyncExecutor = func(ctx context.Context, cookie string, audience Audience, calendarId uint64, depth int, materialize bool, claim *pk.FetchLogEntity, resume bool) (*SyncReport, error) {
	if audience == AudienceGraduate {
		return SyncFromClaimForAudience(ctx, cookie, audience, calendarId, depth, materialize, claim, resume)
	}
	return SyncFromClaim(ctx, cookie, calendarId, depth, materialize, claim, resume)
}

// RunScheduledSync 执行管理端配置的排课数据定时同步（issue #569）。
// 由 console/job cron 在进程内触发，复用与手动「立即同步」完全相同的管线：
//   - semester 参数：读配置 Term（数字 calendarId / 学期名 / 空=最近已同步学期）
//   - depth 参数：读配置 Depth，clamp 到 [1, maxScheduledSyncDepth]
//   - PkFetchLog 续跑 + 重入保护：ClaimSyncCalendarForAudience 原子认领租约，
//     1 小时 running 窗口内的重入会被拒绝并跳过本次（不排队、不重复删数据）
//   - 执行体与手动入口一致：SyncFromClaim / SyncFromClaimForAudience（分页抓取、
//     时间片重建、课评物化、断点游标），panic 时把租约标记为 failed
//
// 返回 error 仅表示「本次触发未进行/失败」；cron 触发方各自记录日志。
func RunScheduledSync(ctx context.Context) (err error) {
	cfg := hotdataserve.GetPkSyncScheduleConfigCache()
	if !cfg.Enabled {
		return errors.New("排课定时同步未启用（enabled=false），跳过")
	}
	audience, err := ParseAudience(cfg.Audience)
	if err != nil {
		return fmt.Errorf("排课定时同步配置的数据来源无效：%w", err)
	}
	calendarId, err := resolveScheduledSyncTerm(audience, cfg.Term)
	if err != nil {
		return err
	}
	cookie, err := ResolveCredentialForAudience("", audience)
	if err != nil {
		return err
	}
	depth := cfg.Depth
	if depth < 1 {
		depth = 1
	}
	if depth > maxScheduledSyncDepth {
		depth = maxScheduledSyncDepth
	}

	// 重入保护：只有取得租约的触发才会执行；running 窗口内的并发触发在这里被拒绝。
	claim, resume, err := ClaimSyncCalendarForAudience(audience, calendarId)
	if err != nil {
		return fmt.Errorf("排课定时同步被拒绝（重入保护/并发防护）：%w", err)
	}

	// panic 防护与手动入口对齐：panic 会把租约遗留为 running，续跑窗口内会挡住
	// 后续触发，必须显式标记 failed 并转成 error 返回（不吞 panic）。
	defer func() {
		if p := recover(); p != nil {
			slog.Error("pk scheduled sync panic", "err", p)
			if markErr := FailSyncClaim(claim, fmt.Errorf("排课定时同步异常：%v", p)); markErr != nil {
				slog.Warn("pk scheduled sync panic failure record", "calendarId", calendarId, "err", markErr)
			}
			err = fmt.Errorf("排课定时同步异常：%v", p)
		}
	}()

	var report *SyncReport
	report, err = scheduledSyncExecutor(ctx, cookie, audience, calendarId, depth, true, claim, resume)
	if err != nil {
		return fmt.Errorf("排课定时同步失败（fetchlog 已标记 failed，下轮触发续跑）：%w", err)
	}
	slog.Info("pk scheduled sync completed", "calendarId", calendarId, "audience", audience,
		"depth", depth, "teachingClassInserted", report.TeachingClassInserted, "calendars", report.CalendarIDs)
	return nil
}

// resolveScheduledSyncTerm 解析定时同步的目标学期：
//   - Term 非空：走 ResolveSyncTermForAudience（数字 calendarId / 学期名 / 归一化反查）
//   - Term 留空：取该数据来源最近已同步的学期（pk_calendar.calendar_id 最大者），
//     不会探测上游新学期；新学期须先手动同步或指定 ID。尚无任何学期时报错。
func resolveScheduledSyncTerm(audience Audience, term string) (uint64, error) {
	if strings.TrimSpace(term) != "" {
		id, _, err := ResolveSyncTermForAudience(audience, term)
		return id, err
	}
	calendars, err := pk.ListCalendarsForAudience(audience, 1)
	if err != nil {
		return 0, err
	}
	if len(calendars) == 0 {
		return 0, errors.New("排课定时同步未指定学期且尚无已同步学期（pk_calendar 为空）：请在管理端定时同步配置中填写目标学期")
	}
	return pk.ExternalID(audience, calendars[0].CalendarId), nil
}
