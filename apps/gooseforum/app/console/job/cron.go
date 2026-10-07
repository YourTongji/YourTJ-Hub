package job

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/closer"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/db4fileconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/logging"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/dailyStats"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/networkAccessLog"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/anonymousidentityservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/contentdeleteservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/courseservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/dataservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/fileusageservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/nativepushservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/oidcservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/pkservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/storageservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/totpservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/webpushservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/wikiservice"
	"github.com/robfig/cron/v3"
)

var scheduler = cron.New(
	cron.WithLogger(cron.VerbosePrintfLogger(logging.CronLogging{})),
)
var (
	runMu      sync.Mutex
	running    bool
	registered bool
)

func Run() {
	if !preferences.GetBool("cron.enabled", true) {
		slog.Info("cron disabled", "compensation", "durable workers and explicit operator jobs remain available")
		return
	}

	runMu.Lock()
	defer runMu.Unlock()
	if running {
		slog.Debug("cron already running")
		return
	}
	if !registered {
		closer.RegisterPriorityContext(closer.PriorityProducer, func(ctx context.Context) error {
			return Stop(ctx)
		})
		registerJobs()
		registered = true
	}
	running = true
	scheduler.Start()
}

func registerJobs() {
	slog.Info("start cron")
	_, errAnonymous := scheduler.AddFunc("13 * * * *", upCmd(func() {
		if err := anonymousidentityservice.Default(context.Background()).Cleanup(); err != nil {
			slog.Warn("anonymous candidate cleanup failed", "err", err)
		}
	}))
	if errAnonymous != nil {
		slog.Error("register anonymous cleanup failed", "err", errAnonymous)
	}
	backupSpec := preferences.Get("db.spec", "0 3 * * *")
	entryID, err := scheduler.AddFunc(backupSpec, upCmd(func() {
		dbconnect.BackupSQLiteHandle()
		db4fileconnect.BackupSQLiteHandle()
	}))
	slog.Info("reg cron", "entryID", entryID, "spec", backupSpec, "err", err)
	entryID, err = scheduler.AddFunc("3 3 * * *", upCmd(func() {
		// 实现未来7天的创建。检查除了今天以外6天的是否创建，如果没有创建则进行创建
		now := time.Now()
		keys := []dailyStats.StatType{
			dailyStats.StatTypeRegCount,
			dailyStats.StatTypeTopicCount,
			dailyStats.StatTypeReplyCount,
			dailyStats.StatTypeCourseReviewCount,
		}
		for i := range 7 {
			date := now.AddDate(0, 0, i)
			for _, key := range keys {
				_ = dailyStats.InitStats(date, key)
			}
		}
	}))
	slog.Info("reg cron", "entryID", entryID, "spec", backupSpec, "err", err)
	entryID, err = scheduler.AddFunc("4 3 * * *", upCmd(func() {
		// 清理超过保留期的数据导出文件
		dataservice.CleanupExpiredExports()
	}))
	slog.Info("reg cron", "entryID", entryID, "spec", "4 3 * * *", "err", err)
	entryID, err = scheduler.AddFunc("5 3 * * *", upCmd(func() {
		// 清理过期的 TOTP challenge token 记录
		totpservice.CleanupExpiredChallenges()
	}))
	slog.Info("reg cron", "entryID", entryID, "spec", "5 3 * * *", "err", err)
	entryID, err = scheduler.AddFunc("6 3 * * *", upCmd(func() {
		// 清理过期的 OIDC authorization request 和 access token 记录
		oidcservice.CleanupExpired()
	}))
	slog.Info("reg cron", "entryID", entryID, "spec", "6 3 * * *", "err", err)
	entryID, err = scheduler.AddFunc("7 3 * * *", upCmd(func() {
		// 删除恢复窗口结束的用户内容，并同步清理其附件引用。
		if err := contentdeleteservice.ExpireRecoverableBatch(200); err != nil {
			slog.Error("expire recoverable content failed", "err", err)
		}
		fileusageservice.ExpireRecoveringFiles(200)
	}))
	slog.Info("reg cron", "entryID", entryID, "spec", "7 3 * * *", "err", err)
	entryID, err = scheduler.AddFunc("8 3 * * *", upCmd(func() {
		// 清理超过 180 天保留期的已结案举报证据快照（hold 话题除外）。
		if err := contentdeleteservice.ExpireEvidenceSnapshotsBatch(200); err != nil {
			slog.Error("expire evidence snapshots failed", "err", err)
		}
	}))
	slog.Info("reg cron", "entryID", entryID, "spec", "8 3 * * *", "err", err)
	entryID, err = scheduler.AddFunc("9 3 * * *", upCmd(func() {
		// 清理超过 6 个月（183 天）保留期的网络访问日志。
		if _, err := networkAccessLog.ExpireBefore(time.Now().Add(-networkAccessLog.Retention), 500); err != nil {
			slog.Error("expire network access logs failed", "err", err)
		}
	}))
	slog.Info("reg cron", "entryID", entryID, "spec", "9 3 * * *", "err", err)
	entryID, err = scheduler.AddFunc("10 3 * * *", upCmd(func() {
		// 课评删除隔离窗口清理（issue #175 B3 隐私合规）：入队
		// course-review-cleanup 任务，由 worker 消费；失败按 taskQueue
		// 语义重试至多 3 次后 failed 并有日志（下次 cron 触发重新入队）。
		if err := courseservice.EnqueueCleanupTask(); err != nil {
			slog.Error("enqueue course review cleanup failed", "err", err)
		}
	}))
	slog.Info("reg cron", "entryID", entryID, "spec", "10 3 * * *", "err", err)
	entryID, err = scheduler.AddFunc("11 3 * * *", upCmd(func() {
		// 清理超过 7 天保留期的终态（Success/Failed）推送任务行（review P2）：
		// webpush 通知每条一行 outbox，只置终态不清理会让 task_queue 随通知
		// 流量无界增长。分批删除（500/批），仅清理终态，不动未完成任务。
		if _, cleanupErr := webpushservice.CleanupTerminalTasks(time.Now().Add(-7*24*time.Hour), 500); cleanupErr != nil {
			slog.Error("cleanup terminal webpush tasks failed", "err", cleanupErr)
		}
	}))
	slog.Info("reg cron", "entryID", entryID, "spec", "11 3 * * *", "err", err)
	entryID, err = scheduler.AddFunc("12 3 * * *", upCmd(func() {
		// 清理超过 7 天保留期的终态（Success/Failed）原生推送任务行：
		// nativepush 同样每条通知一行 outbox，与 webpush 共用同一保留策略。
		if _, cleanupErr := nativepushservice.CleanupTerminalTasks(time.Now().Add(-7*24*time.Hour), 500); cleanupErr != nil {
			slog.Error("cleanup terminal nativepush tasks failed", "err", cleanupErr)
		}
	}))
	slog.Info("reg cron", "entryID", entryID, "spec", "12 3 * * *", "err", err)
	entryID, err = scheduler.AddFunc("17 * * * *", upCmd(func() {
		// 清理超过 2 小时未完成的直接上传（S3 presigned，issue #366）：
		// 删除对象与 pending 元数据行，避免中断/过期上传遗留孤儿对象。
		removed, cleanupErr := storageservice.CleanupPending(context.Background(), time.Now().Add(-2*time.Hour), 500)
		if cleanupErr != nil || removed > 0 {
			slog.Info("cleanup pending uploads", "removed", removed, "err", cleanupErr)
		}
	}))
	slog.Info("reg pending upload cleanup", "entryID", entryID, "err", err)
	// wiki GitHub 同步：默认每日 03:00（可配 [wiki.git].schedule 覆盖）；
	// 未配置 [wiki.git].repo 时 Sync 直接报错跳过（幂等，配置后重启即生效）。
	wikiSpec := preferences.GetString("wiki.git.schedule", "0 3 * * *")
	entryID, err = scheduler.AddFunc(wikiSpec, upCmd(func() {
		if _, syncErr := wikiservice.Sync("schedule"); syncErr != nil {
			slog.Warn("wiki scheduled sync failed", "error", syncErr)
		}
	}))
	slog.Info("reg cron", "entryID", entryID, "spec", wikiSpec, "err", err)
	// 排课数据定时同步（issue #569）：默认关闭；管理端「一系统同步」开启并保存
	// cron 表达式后注册本条目，配置热更新走 RefreshPkSyncCron（无需重启）。
	registerPkSyncCron()
}

// pkSyncEntry 排课数据定时同步的 cron entry（0 = 未注册）。配置经管理端保存后
// 通过 RefreshPkSyncCron 增删/改写条目，宿主进程的热修改无需重启。
var pkSyncEntry cron.EntryID

// registerPkSyncCron 按当前配置注册（或注销）排课数据定时同步 cron：
//   - 未启用或表达式为空时不注册；
//   - 已有旧条目先 Remove 再按新表达式 AddFunc（AddFunc 解析失败时保持未注册，
//     管理端保存侧已用同一标准解析器前置校验，理论上不会走到这里）。
//
// 触发时由 pkservice.RunScheduledSync 执行与「立即同步」同一管线；并发/重入由
// fetchlog 1 小时 running 租约窗口兜底（重叠触发会跳过）。
func registerPkSyncCron() {
	if pkSyncEntry != 0 {
		scheduler.Remove(pkSyncEntry)
		pkSyncEntry = 0
	}
	cfg := hotdataserve.GetPkSyncScheduleConfigCache()
	if !cfg.Enabled {
		return
	}
	spec := strings.TrimSpace(cfg.Schedule)
	if spec == "" {
		return
	}
	entryID, err := scheduler.AddFunc(spec, upCmd(func() {
		// 每次触发重读配置：关闭开关或删除条目后即使残留一次调度也立即空转跳过。
		if err := pkservice.RunScheduledSync(context.Background()); err != nil {
			slog.Warn("pk scheduled sync failed", "error", err)
		}
	}))
	slog.Info("reg pk sync cron", "entryID", entryID, "spec", spec, "err", err)
	if err == nil {
		pkSyncEntry = entryID
	}
}

// RefreshPkSyncCron 供管理端保存排课定时同步配置后热刷新 cron 注册（issue #569），
// 避免改动需重启进程才生效。scheduler 未启动（纯 CLI 模式）时保持惰性：下次
// Run() 会读取最新配置注册。
func RefreshPkSyncCron() {
	runMu.Lock()
	defer runMu.Unlock()
	if !registered {
		return
	}
	registerPkSyncCron()
}

func Stop(parentContexts ...context.Context) error {
	shutdownCtx := context.Background()
	if len(parentContexts) > 0 && parentContexts[0] != nil {
		shutdownCtx = parentContexts[0]
	}

	runMu.Lock()
	if !running {
		runMu.Unlock()
		return nil
	}
	running = false
	stopCtx := scheduler.Stop()
	runMu.Unlock()

	select {
	case <-stopCtx.Done():
		return nil
	case <-shutdownCtx.Done():
		slog.Error("timed out waiting for job to stop", "err", shutdownCtx.Err())
		return shutdownCtx.Err()
	}
}

func upCmd(cmd func()) func() {
	return func() {
		defer func() {
			if p := recover(); p != nil {
				slog.Error("cron panic ", "p", p)
			}
		}()
		cmd()
	}
}
