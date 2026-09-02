package job

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/closer"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/db4fileconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/logging"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/dailyStats"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/networkAccessLog"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/contentdeleteservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/courseservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/dataservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/fileusageservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/oidcservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/storageservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/totpservice"
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
