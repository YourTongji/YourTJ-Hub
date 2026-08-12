package migration

import (
	"log/slog"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/datamigration"
)

func runVersionedDataMigrations() {
	currentVersion := pageConfig.GetMigrationVersion()
	if currentVersion >= pageConfig.AppMigrationVersion {
		return
	}

	slog.Info("app migration start", "currentVersion", currentVersion, "targetVersion", pageConfig.AppMigrationVersion)
	if currentVersion < 1 {
		datamigration.EnsureDefaultData()
		result := datamigration.RebuildReplyMarkdown()
		slog.Info("app migration rebuild reply markdown done", "processed", result.Processed, "skipped", result.Skipped, "failed", result.Failed)
		if result.Failed > 0 {
			slog.Error("app migration rebuild reply markdown has failures", "failed", result.Failed)
			return
		}
		pageConfig.SyncMigrationVersion(1)
		currentVersion = 1
	}
	if currentVersion < 2 {
		result := datamigration.BackfillReplySequence()
		slog.Info("app migration backfill reply sequence done", "articles", result.Articles, "replies", result.Replies, "skipped", result.Skipped, "failed", result.Failed)
		if result.Failed > 0 {
			slog.Error("app migration backfill reply sequence has failures", "failed", result.Failed, "lastFailed", result.LastFailed)
			return
		}
		pageConfig.SyncMigrationVersion(2)
		currentVersion = 2
	}
	if currentVersion < 3 {
		result := datamigration.BackfillArticleUserAction()
		slog.Info("app migration backfill article user action done", "processed", result.Processed, "skipped", result.Skipped, "failed", result.Failed)
		if result.Failed > 0 {
			slog.Error("app migration backfill article user action has failures", "failed", result.Failed)
			return
		}
		pageConfig.SyncMigrationVersion(3)
		currentVersion = 3
	}
	if currentVersion < 4 {
		result := datamigration.MigrateSiteChromeContent()
		slog.Info("app migration site chrome content done", "migrated", result.Migrated, "failed", result.Failed)
		if result.Failed > 0 {
			slog.Error("app migration site chrome content has failures", "failed", result.Failed)
			return
		}
		pageConfig.SyncMigrationVersion(4)
		currentVersion = 4
	}
	if currentVersion < 5 {
		result := datamigration.BackfillTopicPostModel()
		slog.Info("app migration topic post model done", "topics", result.Topics, "posts", result.Posts, "categories", result.Categories, "topicCategoryIndexes", result.TopicCategoryIndexes, "topicUserActions", result.TopicUserActions, "topicUserStats", result.TopicUserStats, "mappings", result.Mappings, "notifications", result.Notifications, "reportsChecked", result.ReportsChecked, "reportsMissing", result.ReportsMissing, "moderationLogs", result.ModerationLogs, "moderationLogsMissing", result.ModerationLogsMissing, "skipped", result.Skipped, "failed", result.Failed, "lastFailed", result.LastFailed)
		if result.Failed > 0 {
			slog.Error("app migration topic post model has failures", "failed", result.Failed, "lastFailed", result.LastFailed)
			return
		}
		pageConfig.SyncMigrationVersion(5)
		currentVersion = 5
	}
	if currentVersion < 6 {
		result := datamigration.BackfillModerationLogsTopicPost()
		slog.Info("app migration moderation log topic post migration done", "moderationLogs", result.ModerationLogs, "moderationLogsMissing", result.ModerationLogsMissing, "failed", result.Failed, "lastFailed", result.LastFailed)
		if result.Failed > 0 {
			slog.Error("app migration moderation log topic post migration has failures", "failed", result.Failed, "lastFailed", result.LastFailed)
			return
		}
		pageConfig.SyncMigrationVersion(6)
		currentVersion = 6
	}
	if currentVersion < 7 {
		result := datamigration.BackfillFileUsagesTopicPost()
		slog.Info("app migration file usage topic post migration done", "fileUsages", result.FileUsages, "fileUsagesMissing", result.FileUsagesMissing, "failed", result.Failed, "lastFailed", result.LastFailed)
		if result.Failed > 0 {
			slog.Error("app migration file usage topic post migration has failures", "failed", result.Failed, "lastFailed", result.LastFailed)
			return
		}
		pageConfig.SyncMigrationVersion(7)
		currentVersion = 7
	}
	if currentVersion < 8 {
		result := datamigration.MigrateTopicCountNaming()
		slog.Info("app migration topic count naming done", "userStatisticsMigrated", result.UserStatisticsMigrated, "dailyStatsMigrated", result.DailyStatsMigrated, "failed", result.Failed, "lastFailed", result.LastFailed)
		if result.Failed > 0 {
			slog.Error("app migration topic count naming has failures", "failed", result.Failed, "lastFailed", result.LastFailed)
			return
		}
		pageConfig.SyncMigrationVersion(8)
		currentVersion = 8
	}
	if currentVersion < 9 {
		result := datamigration.MigrateTopicSearchIndex()
		slog.Info("app migration topic search index done", "skipped", result.Skipped, "rebuilt", result.Rebuilt, "processed", result.ProcessedCount, "failedCount", result.FailedCount, "legacyIndexDeleteTried", result.LegacyIndexDeleteTried, "legacyIndexDeleted", result.LegacyIndexDeleted, "failed", result.Failed, "lastFailed", result.LastFailed)
		if result.Failed > 0 || result.FailedCount > 0 {
			slog.Error("app migration topic search index has failures", "failed", result.Failed, "failedCount", result.FailedCount, "lastFailed", result.LastFailed)
			return
		}
		pageConfig.SyncMigrationVersion(9)
		currentVersion = 9
	}
	if currentVersion < 10 {
		result := datamigration.DropReportLegacyColumns()
		slog.Info("app migration report legacy columns done", "articleIDColumnDropped", result.ArticleIDColumnDropped, "statusArticleIndexDrop", result.StatusArticleIndexDrop, "articleIndexDrop", result.ArticleIndexDrop, "failed", result.Failed, "lastFailed", result.LastFailed)
		if result.Failed > 0 {
			slog.Error("app migration report legacy columns has failures", "failed", result.Failed, "lastFailed", result.LastFailed)
			return
		}
		pageConfig.SyncMigrationVersion(10)
		currentVersion = 10
	}
	if currentVersion < 11 {
		result := datamigration.MigratePointsRecordAction()
		slog.Info("app migration points record action done", "backfilled", result.Backfilled, "changeReasonColumnDropped", result.ChangeReasonColumnDropped, "failed", result.Failed, "lastFailed", result.LastFailed)
		if result.Failed > 0 {
			slog.Error("app migration points record action has failures", "failed", result.Failed, "lastFailed", result.LastFailed)
			return
		}
		pageConfig.SyncMigrationVersion(11)
		currentVersion = 11
	}
	if currentVersion < 12 {
		result := datamigration.RebuildPostMarkdown()
		slog.Info("app migration rebuild post markdown done", "processed", result.Processed, "failed", result.Failed, "lastFailed", result.LastFailed)
		if result.Failed > 0 {
			slog.Error("app migration rebuild post markdown has failures", "failed", result.Failed, "lastFailed", result.LastFailed)
			return
		}
		pageConfig.SyncMigrationVersion(12)
		currentVersion = 12
	}
	if currentVersion < 13 {
		result := datamigration.MigrateAggregateSearchIndexes()
		datamigration.LogAggregateSearchIndexMigration(result)
		if result.Skipped {
			// Meilisearch 不可用：不推进版本，下次启动重试（避免永久跳过索引构建）
			slog.Warn("app migration aggregate search indexes skipped (meilisearch unavailable), will retry on next start")
			return
		}
		if result.Failed > 0 {
			slog.Error("app migration aggregate search indexes has failures", "failed", result.Failed, "lastFailed", result.LastFailed)
			return
		}
		pageConfig.SyncMigrationVersion(13)
		currentVersion = 13
	}
	if currentVersion < 14 {
		// 积分回填（dev 合并,PR #110 防滥用）
		result := datamigration.BackfillMissingUserPoints()
		slog.Info("app migration user points backfill done", "backfilled", result.Backfilled, "failed", result.Failed, "lastFailed", result.LastFailed)
		if result.Failed > 0 {
			slog.Error("app migration user points backfill has failures", "failed", result.Failed, "lastFailed", result.LastFailed)
			return
		}
		pageConfig.SyncMigrationVersion(14)
		currentVersion = 14
	}
	if currentVersion < 15 {
		// 删除生命周期回填（Issue #94）：dev 的 v14 已被积分回填占用，
		// 删除回填必须用独立版本号，否则已跑过 v14 的线上实例会永远跳过。
		deleteResult := datamigration.BackfillDeleteLifecycle()
		slog.Info("app migration delete lifecycle backfill done", "topics", deleteResult.TopicsBackfilled, "posts", deleteResult.PostsBackfilled, "failed", deleteResult.Failed, "lastFailed", deleteResult.LastFailed)
		if deleteResult.Failed > 0 {
			slog.Error("app migration delete lifecycle backfill has failures", "failed", deleteResult.Failed, "lastFailed", deleteResult.LastFailed)
			return
		}
		pageConfig.SyncMigrationVersion(15)
		currentVersion = 15
	}
	if currentVersion < 16 {
		// 移除 GitHub OAuth 明文 token 持久化并清理历史列（issue #131,PR #150）
		result := datamigration.DropUserOAuthTokenColumns()
		slog.Info("app migration user oauth credentials drop done", "dropped", result.Dropped, "failed", result.Failed, "lastFailed", result.LastFailed)
		if result.Failed > 0 {
			slog.Error("app migration user oauth credentials drop has failures", "failed", result.Failed, "lastFailed", result.LastFailed)
			return
		}
		pageConfig.SyncMigrationVersion(16)
		currentVersion = 16
	}
	slog.Info("app migration end", "version", currentVersion)
}
