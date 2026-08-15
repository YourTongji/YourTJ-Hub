package job

import (
	"context"
	"errors"
	"log/slog"
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
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/totpservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/wikiservice"
	"github.com/robfig/cron/v3"
)

var scheduler = cron.New(
	cron.WithLogger(cron.VerbosePrintfLogger(logging.CronLogging{})),
)
var running = false
var jobContext, cancelJobs = context.WithCancel(context.Background())

func Run() {
	jobContext, cancelJobs = context.WithCancel(context.Background())
	closer.RegisterPriority(closer.PriorityProducer, Stop)
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
	// Wiki 同步只有显式启用时才注册，避免仅填写 repo 就开始写投影。
	wikiConfig := wikiservice.LoadGitConfig()
	if wikiConfig.Enabled() {
		entryID, err = scheduler.AddFunc(wikiConfig.Schedule, upCmd(runScheduledWikiSync))
		slog.Info("reg cron", "entryID", entryID, "spec", wikiConfig.Schedule, "err", err)
	} else {
		slog.Info("wiki git sync cron disabled")
	}
	running = true
	scheduler.Start()
}

// runScheduledWikiSync 执行定时 wiki 同步：以 jobContext 为父 context（服务停止时
// 取消正在运行的同步），单次 10 分钟超时兜底。命名函数而非闭包内派生 context，
// 避免 fatcontext 报嵌套 context（golangci-lint）。
func runScheduledWikiSync() {
	ctx, cancel := context.WithTimeout(jobContext, 10*time.Minute)
	defer cancel()
	if _, syncErr := wikiservice.SyncContext(ctx, "schedule"); syncErr != nil {
		slog.Warn("wiki scheduled sync failed", "error", syncErr)
	}
}

func Stop() error {
	if !running {
		return nil
	}
	cancelJobs()
	ctx := scheduler.Stop()
	select {
	case <-ctx.Done():
		running = false
		return nil
	case <-time.After(10 * time.Second):
		slog.Error("timed out waiting for job to stop")
		return errors.New("timed out waiting for job to stop")
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
