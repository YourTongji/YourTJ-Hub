package pkservice

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pk"
	"gorm.io/gorm"
)

// batchRows 每个事务批次写入的教学班行数（AC：每批 500 行）。
const batchRows = 500

// Sync 同步一系统排课数据到 PK 域。
//   - cookie：一系统会话 Cookie header（ONESYSTEM_COOKIE）
//   - calendarId：目标学期（一系统侧 ID）
//   - depth：以 calendarId 为终点向前同步 N 个连续学期（默认 1）
//   - materialize：同步完成后是否物化课程目录（默认 off）
//
// AC1 幂等：同一学期重复运行先清空再全量重写；AC3 断点续跑：上次 running/failed 的日志
// 从最后已提交页继续（不回滚已成功批次）；AC2 cookie 失效：抓取失败即中止并标记 failed，
// 已提交批次保留。
func Sync(ctx context.Context, cookie string, calendarId uint64, depth int, materialize bool) (*SyncReport, error) {
	return syncWithClaim(ctx, newOnesystemClient(), cookie, calendarId, depth, materialize, nil, false)
}

// ClaimSyncCalendar 原子认领一个学期的同步租约。管理端在确认请求前调用它，只有取得租约的
// 请求才能返回 started=true；后台执行会通过 SyncFromClaim 复用同一租约。
func ClaimSyncCalendar(calendarId uint64) (*pk.FetchLogEntity, bool, error) {
	return beginOrResumeFetchLog(calendarId)
}

// SyncFromClaim 使用已由 ClaimSyncCalendar 认领的目标学期租约执行同步。
func SyncFromClaim(ctx context.Context, cookie string, calendarId uint64, depth int, materialize bool, claim *pk.FetchLogEntity, resume bool) (*SyncReport, error) {
	if claim == nil || claim.CalendarId != calendarId || claim.Status != pk.FetchStatusRunning {
		return nil, errors.New("无效的排课同步租约")
	}
	return syncWithClaim(ctx, newOnesystemClient(), cookie, calendarId, depth, materialize, claim, resume)
}

// FailSyncClaim records a terminal failure for a worker that already claimed a sync lease.
func FailSyncClaim(claim *pk.FetchLogEntity, err error) error {
	return markFailed(claim, err)
}

// syncWith 注入 onesystemClient，便于测试使用本地 httptest 服务。
func syncWith(ctx context.Context, client *onesystemClient, cookie string, calendarId uint64, depth int, materialize bool) (*SyncReport, error) {
	return syncWithClaim(ctx, client, cookie, calendarId, depth, materialize, nil, false)
}

func syncWithClaim(ctx context.Context, client *onesystemClient, cookie string, calendarId uint64, depth int, materialize bool, claimedTarget *pk.FetchLogEntity, claimedTargetResume bool) (*SyncReport, error) {
	if strings.TrimSpace(cookie) == "" {
		return nil, errors.New("缺少 ONESYSTEM_COOKIE（一系统 Cookie header），请通过 --onesystem-cookie / ONESYSTEM_COOKIE 环境变量 / 管理端设置提供")
	}
	if calendarId == 0 {
		return nil, errors.New("calendarId 无效")
	}
	if depth < 1 {
		depth = 1
	}

	report := &SyncReport{}
	start := calendarId - uint64(depth-1)
	if uint64(depth) > calendarId {
		start = 1
	}
	for id := start; id <= calendarId; id++ {
		var claim *pk.FetchLogEntity
		resume := false
		if id == calendarId {
			claim = claimedTarget
			resume = claimedTargetResume
		}
		perCalendar, err := syncOneCalendar(ctx, client, cookie, id, claim, resume)
		if err != nil {
			// 管理端会先认领目标学期；若前置的历史学期失败，必须释放目标租约并留下可见失败，
			// 否则前端会一直把尚未执行的目标显示为 running。
			if claimedTarget != nil && id != calendarId {
				if markErr := markFailed(claimedTarget, fmt.Errorf("同步前置学期 %d 失败：%w", id, err)); markErr != nil {
					slog.Warn("course-pk-sync: mark claimed target failed", "calendarId", calendarId, "err", markErr)
				}
			}
			// 部分失败：已成功学期的 teacher_timeslots 仍须重建（它们在学期开头被清空，
			// 若不重建会留下空时间片）。重建失败只告警，不掩盖原始错误。
			if len(report.CalendarIDs) > 0 {
				if rebuilt, rebuildErr := rebuildTimeslots(ctx, report.CalendarIDs); rebuildErr == nil {
					report.TimeslotsRebuilt = rebuilt
				} else {
					slog.Warn("course-pk-sync: rebuild timeslots for synced calendars", "calendars", report.CalendarIDs, "err", rebuildErr)
				}
			}
			return report, err
		}
		report.CalendarIDs = append(report.CalendarIDs, id)
		report.TeachingClassInserted += perCalendar.inserted
		report.BatchesCommitted += perCalendar.batches
		report.FetchedPages += perCalendar.pages
		report.ResumedFromPage = perCalendar.resumePage
	}

	if len(report.CalendarIDs) > 0 {
		rebuilt, err := rebuildTimeslots(ctx, report.CalendarIDs)
		if err != nil {
			return report, err
		}
		report.TimeslotsRebuilt = rebuilt
	}

	if materialize && len(report.CalendarIDs) > 0 {
		materialized, err := materializeToCatalog(ctx, report.CalendarIDs)
		if err != nil {
			return report, err
		}
		report.MaterializedCourses = materialized
	}

	if len(report.CalendarIDs) > 0 {
		// 历史 fetchlog 清理失败只告警不回滚同步结果（已提交批次保留）。
		if err := pk.CleanupOldFetchLogs(); err != nil {
			slog.Warn("course-pk-sync: cleanup old fetch logs", "err", err)
		}
	}
	return report, nil
}

type calendarSyncResult struct {
	inserted   int
	batches    int
	pages      int
	resumePage int
}

// syncOneCalendar 同步单个学期：断点判定 → 分页抓取 →（cookie 验证后）清空 → 500 行/批事务写入。
// 破坏性删除（DeleteCalendarDataTx）只在该学期"全新同步"或"续跑且上一轮仅删未写"时执行，且必须
// 在首页抓取成功（cookie 有效）之后，避免无效 cookie 摧毁存量数据（AC2）。
func syncOneCalendar(ctx context.Context, client *onesystemClient, cookie string, calendarId uint64, claimedLog *pk.FetchLogEntity, claimedResume bool) (calendarSyncResult, error) {
	var result calendarSyncResult

	log := claimedLog
	resume := claimedResume
	if log == nil {
		var err error
		log, resume, err = beginOrResumeFetchLog(calendarId)
		if err != nil {
			return result, err
		}
	}

	startPage := 1
	if resume {
		startPage = log.LastCommittedPage + 1
		result.resumePage = startPage
		// 崩溃窗口：所有页已提交但上一轮未标记完成 → 直接补标记，不再抓取。
		if log.TotalPages > 0 && startPage > log.TotalPages {
			now := time.Now()
			log.Status = pk.FetchStatusCompleted
			log.FinishedAt = &now
			if err := pk.SaveFetchLog(log); err != nil {
				return result, err
			}
			return result, nil
		}
	}

	// 先抓首页：既验证 cookie，又取得 total_。
	first, err := client.fetchPage(ctx, cookie, int(calendarId), startPage, onesystemPageSize)
	if err != nil {
		return result, markFailed(log, err)
	}

	// cookie 有效后才清空：全新同步，或续跑且上一轮仅删未写（lastCommittedPage==0 且无写入）时补删。
	if !resume || (log.LastCommittedPage == 0 && log.RowsWritten == 0) {
		if err := deleteCalendarData(log, calendarId); err != nil {
			return result, markFailed(log, err)
		}
	}

	result.pages++
	buffer := append([]CourseRaw(nil), first.Data.List...)

	totalPages := 1
	if first.Data.Total_ > 0 {
		totalPages = (first.Data.Total_ + onesystemPageSize - 1) / onesystemPageSize
	}
	if totalPages < startPage {
		totalPages = startPage
	}
	log.TotalPages = totalPages
	if err := pk.SaveFetchLog(log); err != nil {
		return result, err
	}

	for page := startPage + 1; page <= totalPages; page++ {
		p, err := client.fetchPage(ctx, cookie, int(calendarId), page, onesystemPageSize)
		if err != nil {
			return result, markFailed(log, err)
		}
		result.pages++
		buffer = append(buffer, p.Data.List...)
		if len(buffer) >= batchRows || page == totalPages {
			n, err := commitBatch(log, calendarId, buffer, page)
			if err != nil {
				return result, markFailed(log, err)
			}
			result.inserted += n
			result.batches++
			buffer = nil
		}
	}
	// 单页场景（startPage == totalPages）的收尾。
	if len(buffer) > 0 {
		n, err := commitBatch(log, calendarId, buffer, startPage)
		if err != nil {
			return result, markFailed(log, err)
		}
		result.inserted += n
		result.batches++
	}

	now := time.Now()
	log.Status = pk.FetchStatusCompleted
	log.FinishedAt = &now
	if err := pk.SaveFetchLog(log); err != nil {
		return result, err
	}
	return result, nil
}

// fetchLogStaleWindow 视为"陈旧/中断"的 running 日志时间窗：窗内拒绝并发，窗外按中断续跑。
const fetchLogStaleWindow = time.Hour

// beginOrResumeFetchLog 决定本次运行是全新同步还是断点续跑：
//   - 无日志或最近一次 completed → 新建 running 日志，返回全新（需清空）
//   - 最近一次 running 且未陈旧 → 拒绝：另一进程正在同步（并发防护，避免相互删数据）
//   - 最近一次 running 已陈旧 / failed → 原子认领该日志后续跑
//
// 并发防护三层（review HIGH「唯一/租约」）：
//  1. 应用层按 StartedAt 时间窗拒绝未陈旧的并发；
//  2. 续跑（stale running / failed）用 ClaimFetchLog 原子认领（lease_version 精确 CAS，
//     WHERE lease_version=<旧值> SET lease_version=lease_version+1，RowsAffected 判定胜负），
//     两进程同时读到同一行时仅一个能成功，另一进程返回「并发续跑冲突」，避免复用同一行导致
//     double-delete；
//  3. 新建用 running_key 唯一索引兜底（running 行 running_key=calendar_id，其余为 NULL），
//     消除「两进程同时读不到 running → 各自 CreateFetchLog 成功」的 TOCTOU 竞态。三种数据库
//     均允许唯一索引含多个 NULL，故 completed/failed 行永不冲突，但同一 calendar 的两条
//     running 行必然冲突（跨方言，不依赖 PG 专属 partial unique index）。
func beginOrResumeFetchLog(calendarId uint64) (*pk.FetchLogEntity, bool, error) {
	latest, ok := pk.LatestFetchLogByCalendar(calendarId)
	if ok {
		switch latest.Status {
		case pk.FetchStatusRunning:
			if latest.StartedAt != nil && time.Since(*latest.StartedAt) < fetchLogStaleWindow {
				return nil, false, errors.New("该学期同步正在进行中（fetchlog running），请勿并发运行；若确已中断请稍后再试")
			}
			// stale running：原子认领（lease_version CAS，赢家 +1 后旧版本失效），防止并发续跑同一行。
			claimed, err := pk.ClaimFetchLog(latest.Id, pk.FetchStatusRunning, latest.LeaseVersion)
			if err != nil {
				return nil, false, err
			}
			if !claimed {
				return nil, false, errors.New("该学期同步正在进行中（并发续跑冲突），请稍后再试")
			}
			now := time.Now()
			latest.StartedAt = &now
			latest.LeaseVersion++
			return &latest, true, nil
		case pk.FetchStatusFailed:
			// failed：原子改回 running 认领（状态转换 + lease_version CAS 双重串行化），防止并发续跑同一行。
			claimed, err := pk.ClaimFetchLog(latest.Id, pk.FetchStatusFailed, latest.LeaseVersion)
			if err != nil {
				return nil, false, err
			}
			if !claimed {
				return nil, false, errors.New("该学期同步正在进行中（并发续跑冲突），请稍后再试")
			}
			now := time.Now()
			latest.Status = pk.FetchStatusRunning
			latest.StartedAt = &now
			latest.LeaseVersion++
			return &latest, true, nil
		}
	}
	log, err := pk.CreateFetchLog(calendarId)
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, false, errors.New("该学期同步正在进行中（fetchlog running 唯一约束冲突），请勿并发运行；若确已中断请稍后再试")
		}
		return nil, false, err
	}
	return log, false, nil
}

// markFailed 将同步日志标记为 failed 并记录错误（保留已提交批次，AC2/AC3）。
// 返回原始错误（而非 SaveFetchLog 的错误），避免上层把失败的同步误判为成功。
func markFailed(log *pk.FetchLogEntity, err error) error {
	now := time.Now()
	log.Status = pk.FetchStatusFailed
	log.ErrorMsg = err.Error()
	log.FinishedAt = &now
	if saveErr := pk.SaveFetchLog(log); saveErr != nil {
		return fmt.Errorf("%w（另：更新 fetchlog 失败：%v）", err, saveErr)
	}
	return err
}

// commitBatch 在独立事务中写入一批教学班（约 500 行），成功后推进 fetchlog 游标。
// 事务失败不影响已提交批次（AC2）。
func commitBatch(log *pk.FetchLogEntity, calendarId uint64, rows []CourseRaw, committedPage int) (int, error) {
	n, err := writeBatchWithLeaseTx(log, calendarId, rows)
	if err != nil {
		return 0, err
	}
	log.LastCommittedPage = committedPage
	log.RowsWritten += n
	if err := pk.SaveFetchLog(log); err != nil {
		return 0, err
	}
	return n, nil
}

// deleteCalendarData 在单事务内清空某学期排课数据（幂等全量重写前置步骤）。
func deleteCalendarData(log *pk.FetchLogEntity, calendarId uint64) error {
	return db.Connect().Transaction(func(tx *gorm.DB) error {
		if err := pk.RenewFetchLogLeaseTx(tx, log); err != nil {
			return err
		}
		return pk.DeleteCalendarDataTx(tx, calendarId)
	})
}

// writeBatchTx 是写入转换的无租约测试入口；同步流程必须使用 writeBatchWithLeaseTx。
func writeBatchTx(calendarId uint64, rows []CourseRaw) (int, error) {
	return writeBatchWithLeaseTx(nil, calendarId, rows)
}

// writeBatchWithLeaseTx 在单事务内续租并批量 upsert 一批教学班，返回处理行数。
func writeBatchWithLeaseTx(log *pk.FetchLogEntity, calendarId uint64, rows []CourseRaw) (int, error) {
	var n int
	err := db.Connect().Transaction(func(tx *gorm.DB) error {
		if log != nil {
			if err := pk.RenewFetchLogLeaseTx(tx, log); err != nil {
				return err
			}
		}
		written, err := writeBatchTxInner(tx, calendarId, rows)
		if err != nil {
			return err
		}
		n = written
		return nil
	})
	return n, err
}
