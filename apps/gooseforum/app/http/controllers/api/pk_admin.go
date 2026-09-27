package api

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pk"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/optlogger"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/pkservice"
)

// runPkSync 可注入的排课同步执行函数（测试替换为 stub，避免真实抓取一系统）。
var runPkSync = pkservice.SyncFromClaim
var runPkSyncForAudience = pkservice.SyncFromClaimForAudience

// validatePkCredential 可注入的凭证校验执行函数（测试替换为 stub，避免真实抓取一系统）。
var validatePkCredential = pkservice.ValidateCredential

// maxPkSyncDepth 管理端单次同步可向前回溯的学期数上限（对齐 ListCalendars 默认窗口）。
// depth 过大意味着从学期 1 到目标的破坏性全量重写 + 全量抓取，且后台 goroutine 不可取消。
const maxPkSyncDepth = 8

// SyncPkCalendarReq 排课数据同步请求参数。
type SyncPkCalendarReq struct {
	Term     string `json:"term" validate:"required"`
	Depth    int    `json:"depth"`
	Audience string `json:"audience"`
}

// SyncPkCalendar 管理端触发一系统排课数据同步（issue #248 自愈入口）。
// Cookie 按 ResolveCookie 优先级解析（管理端设置/环境变量），前端无需回传。
// 同步异步后台执行（分页抓取可达数十秒~分钟级），立即返回 started:true；
// 进度与结果经 PkSyncStatus（fetchlog 游标）查询，断点续跑保证幂等。
func SyncPkCalendar(req component.BetterRequest[SyncPkCalendarReq]) component.Response {
	term := strings.TrimSpace(req.Params.Term)
	audience, ok := pk.ParseAudience(req.Params.Audience)
	if !ok {
		return component.FailResponseError(fmt.Errorf("同步数据来源无效：%q，请使用 undergraduate 或 graduate", req.Params.Audience))
	}
	calendarId, _, err := pkservice.ResolveSyncTermForAudience(audience, term)
	if err != nil {
		return component.FailResponseError(fmt.Errorf("同步参数错误：%w", err))
	}
	cookie, err := pkservice.ResolveCredentialForAudience("", audience)
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
	claim, resume, err := pkservice.ClaimSyncCalendarForAudience(audience, calendarId)
	if err != nil {
		return component.FailResponseError(err)
	}

	// 仅真正取得租约的请求记审计并确认开始，避免并发请求被误报为成功。
	optlogger.UserOptCode(req.UserId, optlogger.SyncPk, calendarId, "admin.opt.pk.synced", optlogger.MessageParams{
		"term":     term,
		"depth":    depth,
		"audience": audience,
	})

	jobCtx, cancel := detachedJobContext(req.GinContext)
	go func() {
		defer cancel()
		defer func() {
			if p := recover(); p != nil {
				slog.Error("pk sync panic", "err", p)
				if err := pkservice.FailSyncClaim(claim, fmt.Errorf("排课同步异常：%v", p)); err != nil {
					slog.Error("pk sync panic failure record", "calendarId", calendarId, "err", err)
				}
			}
		}()
		var report *pkservice.SyncReport
		var syncErr error
		if audience == pk.AudienceGraduate {
			report, syncErr = runPkSyncForAudience(jobCtx, cookie, audience, calendarId, depth, true, claim, resume)
		} else {
			report, syncErr = runPkSync(jobCtx, cookie, calendarId, depth, true, claim, resume)
		}
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
		"audience":   audience,
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

// SetRunPkSyncForTest 仅测试用：替换后台同步执行函数（与 pkservice 的 ForTest 钩子同一风格），
// 供路由级契约测试注入 stub，避免测试触发真实抓取一系统。返回恢复函数。
func SetRunPkSyncForTest(fn func(ctx context.Context, cookie string, calendarId uint64, depth int, useMaterialize bool, claim *pk.FetchLogEntity, resume bool) (*pkservice.SyncReport, error)) func() {
	orig := runPkSync
	runPkSync = fn
	return func() { runPkSync = orig }
}

// ValidatePkCredentialReq 一系统凭证校验请求参数。
type ValidatePkCredentialReq struct {
	Audience   string `json:"audience"`
	Credential string `json:"credential"`
}

// validatePkCredentialTimeout 校验探测的整体超时：客户端单请求上限 15s，重试窗口
// （最多 5 次退避）叠加后最长约 40s；留出余量避免在服务器写期限内过早中断。
const validatePkCredentialTimeout = 45 * time.Second

// ValidatePkCredential 管理端一系统凭证校验（保存前探测，issue #856）。
// 以目标受众最新已同步学期为真实目标最小抓取一页（pageSize=1），不写库、不写 fetchlog、
// 不触发配置变更。credential 留空时按 CLI 参数 → 环境变量 → 管理端已保存设置解析。
// 一系统侧失败（凭证失效 HTTP 401/403、业务 code!=0、网络错误等）是业务结果而非 HTTP 错误：
// 返回成功信封 result.valid=false 与脱敏后的失败说明；只有参数/解析类硬错误
// （无效 audience、缺少凭证来源等）返回失败信封。
func ValidatePkCredential(req component.BetterRequest[ValidatePkCredentialReq]) component.Response {
	audience, err := pkservice.ParseAudience(req.Params.Audience)
	if err != nil {
		return component.FailResponseError(err)
	}
	ctx := context.Background()
	if req.GinContext != nil {
		ctx = req.GinContext.Request.Context()
		// 服务器默认 10s 写期限；给重试窗口（最长约 40s）留余量后再响应。
		deadlineErr := http.NewResponseController(req.GinContext.Writer).SetWriteDeadline(time.Now().Add(validatePkCredentialTimeout + 10*time.Second))
		if deadlineErr != nil && !errors.Is(deadlineErr, http.ErrNotSupported) {
			return component.FailResponseError(deadlineErr)
		}
	}
	ctx, cancel := context.WithTimeout(ctx, validatePkCredentialTimeout)
	defer cancel()

	result, err := validatePkCredential(ctx, audience, req.Params.Credential)
	if err != nil {
		return component.FailResponseError(err)
	}
	return component.SuccessResponse(map[string]any{
		"valid":   result.Valid,
		"message": result.Message,
	})
}

// SetValidatePkCredentialForTest 仅测试用：替换凭证校验执行函数（与 SetRunPkSyncForTest
// 同一风格），供路由级契约测试注入 stub，避免探测真实一系统。返回恢复函数。
func SetValidatePkCredentialForTest(fn func(ctx context.Context, audience pkservice.Audience, credential string) (pkservice.CredentialValidation, error)) func() {
	orig := validatePkCredential
	validatePkCredential = fn
	return func() { validatePkCredential = orig }
}
