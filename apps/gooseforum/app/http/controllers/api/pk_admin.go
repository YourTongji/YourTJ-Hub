package api

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/optlogger"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/pkservice"
)

// runPkSync 可注入的排课同步执行函数（测试替换为 stub，避免真实抓取一系统）。
var runPkSync = pkservice.SyncFromClaim

// maxPkSyncDepth 管理端单次同步可向前回溯的学期数上限（对齐 ListCalendars 默认窗口）。
// depth 过大意味着从学期 1 到目标的破坏性全量重写 + 全量抓取，且后台 goroutine 不可取消。
const maxPkSyncDepth = 8

// SyncPkCalendarReq 排课数据同步请求参数。
type SyncPkCalendarReq struct {
	Term  string `json:"term" validate:"required"`
	Depth int    `json:"depth"`
}

// SyncPkCalendar 管理端触发一系统排课数据同步（issue #248 自愈入口）。
// Cookie 按 ResolveCookie 优先级解析（管理端设置/环境变量），前端无需回传。
// 同步异步后台执行（分页抓取可达数十秒~分钟级），立即返回 started:true；
// 进度与结果经 PkSyncStatus（fetchlog 游标）查询，断点续跑保证幂等。
func SyncPkCalendar(req component.BetterRequest[SyncPkCalendarReq]) component.Response {
	term := strings.TrimSpace(req.Params.Term)
	calendarId, _, err := pkservice.ResolveSyncTerm(term)
	if err != nil {
		return component.FailResponseError(fmt.Errorf("同步参数错误：%w", err))
	}
	cookie, err := pkservice.ResolveCookie("", "", "")
	if err != nil {
		return component.FailResponseError(err)
	}
	depth := req.Params.Depth
	if depth < 1 {
		depth = 1
	}
	if depth > maxPkSyncDepth {
		depth = maxPkSyncDepth
	}
	claim, resume, err := pkservice.ClaimSyncCalendar(calendarId)
	if err != nil {
		return component.FailResponseError(err)
	}

	// 仅真正取得租约的请求记审计并确认开始，避免并发请求被误报为成功。
	optlogger.UserOptCode(req.UserId, optlogger.SyncPk, calendarId, "admin.opt.pk.synced", optlogger.MessageParams{
		"term":  term,
		"depth": depth,
	})

	go func() {
		defer func() {
			if p := recover(); p != nil {
				slog.Error("pk sync panic", "err", p)
				if err := pkservice.FailSyncClaim(claim, fmt.Errorf("排课同步异常：%v", p)); err != nil {
					slog.Error("pk sync panic failure record", "calendarId", calendarId, "err", err)
				}
			}
		}()
		report, syncErr := runPkSync(context.Background(), cookie, calendarId, depth, false, claim, resume)
		if syncErr != nil {
			slog.Error("pk sync failed", "calendarId", calendarId, "term", term, "err", syncErr)
			return
		}
		slog.Info("pk sync completed", "calendarId", calendarId, "term", term,
			"teachingClassInserted", report.TeachingClassInserted, "calendars", report.CalendarIDs)
	}()

	return component.SuccessResponse(map[string]any{
		"started":    true,
		"calendarId": calendarId,
		"term":       term,
	})
}

// PkSyncStatus 返回各学期排课数据同步状态汇总。
func PkSyncStatus(req component.BetterRequest[component.Null]) component.Response {
	items, err := pkservice.SyncStatusOverview()
	if err != nil {
		return component.FailResponseError(err)
	}
	return component.SuccessResponse(items)
}
