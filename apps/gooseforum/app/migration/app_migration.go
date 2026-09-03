package migration

import (
	"fmt"
	"log/slog"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/datamigration"
)

func runVersionedDataMigrations() error {
	unlock, err := acquireVersionedMigrationLock()
	if err != nil {
		slog.Warn("app migration lock unavailable; data migrations deferred to a later start", "err", err)
		return fmt.Errorf("%w: %w", ErrLockUnavailable, err)
	}
	defer unlock()

	currentVersion := pageConfig.GetMigrationVersion()
	if currentVersion >= pageConfig.AppMigrationVersion {
		return nil
	}

	slog.Info("app migration start", "currentVersion", currentVersion, "targetVersion", pageConfig.AppMigrationVersion)
	if currentVersion < 1 {
		datamigration.EnsureDefaultData()
		result := datamigration.RebuildReplyMarkdown()
		slog.Info("app migration rebuild reply markdown done", "processed", result.Processed, "skipped", result.Skipped, "failed", result.Failed)
		if result.Failed > 0 {
			return dataMigrationError("rebuild reply markdown", 1, result.Failed, "")
		}
		pageConfig.SyncMigrationVersion(1)
		currentVersion = 1
	}
	if currentVersion < 2 {
		result := datamigration.BackfillReplySequence()
		slog.Info("app migration backfill reply sequence done", "articles", result.Articles, "replies", result.Replies, "skipped", result.Skipped, "failed", result.Failed)
		if result.Failed > 0 {
			return dataMigrationError("backfill reply sequence", 2, result.Failed, result.LastFailed)
		}
		pageConfig.SyncMigrationVersion(2)
		currentVersion = 2
	}
	if currentVersion < 3 {
		result := datamigration.BackfillArticleUserAction()
		slog.Info("app migration backfill article user action done", "processed", result.Processed, "skipped", result.Skipped, "failed", result.Failed)
		if result.Failed > 0 {
			return dataMigrationError("backfill article user action", 3, result.Failed, "")
		}
		pageConfig.SyncMigrationVersion(3)
		currentVersion = 3
	}
	if currentVersion < 4 {
		result := datamigration.MigrateSiteChromeContent()
		slog.Info("app migration site chrome content done", "migrated", result.Migrated, "failed", result.Failed)
		if result.Failed > 0 {
			return dataMigrationError("site chrome content", 4, result.Failed, "")
		}
		pageConfig.SyncMigrationVersion(4)
		currentVersion = 4
	}
	if currentVersion < 5 {
		result := datamigration.BackfillTopicPostModel()
		slog.Info("app migration topic post model done", "topics", result.Topics, "posts", result.Posts, "categories", result.Categories, "topicCategoryIndexes", result.TopicCategoryIndexes, "topicUserActions", result.TopicUserActions, "topicUserStats", result.TopicUserStats, "mappings", result.Mappings, "notifications", result.Notifications, "reportsChecked", result.ReportsChecked, "reportsMissing", result.ReportsMissing, "moderationLogs", result.ModerationLogs, "moderationLogsMissing", result.ModerationLogsMissing, "skipped", result.Skipped, "failed", result.Failed, "lastFailed", result.LastFailed)
		if result.Failed > 0 {
			return dataMigrationError("topic post model", 5, result.Failed, result.LastFailed)
		}
		pageConfig.SyncMigrationVersion(5)
		currentVersion = 5
	}
	if currentVersion < 6 {
		result := datamigration.BackfillModerationLogsTopicPost()
		slog.Info("app migration moderation log topic post migration done", "moderationLogs", result.ModerationLogs, "moderationLogsMissing", result.ModerationLogsMissing, "failed", result.Failed, "lastFailed", result.LastFailed)
		if result.Failed > 0 {
			return dataMigrationError("moderation log topic post", 6, result.Failed, result.LastFailed)
		}
		pageConfig.SyncMigrationVersion(6)
		currentVersion = 6
	}
	if currentVersion < 7 {
		result := datamigration.BackfillFileUsagesTopicPost()
		slog.Info("app migration file usage topic post migration done", "fileUsages", result.FileUsages, "fileUsagesMissing", result.FileUsagesMissing, "failed", result.Failed, "lastFailed", result.LastFailed)
		if result.Failed > 0 {
			return dataMigrationError("file usage topic post", 7, result.Failed, result.LastFailed)
		}
		pageConfig.SyncMigrationVersion(7)
		currentVersion = 7
	}
	if currentVersion < 8 {
		result := datamigration.MigrateTopicCountNaming()
		slog.Info("app migration topic count naming done", "userStatisticsMigrated", result.UserStatisticsMigrated, "dailyStatsMigrated", result.DailyStatsMigrated, "failed", result.Failed, "lastFailed", result.LastFailed)
		if result.Failed > 0 {
			return dataMigrationError("topic count naming", 8, result.Failed, result.LastFailed)
		}
		pageConfig.SyncMigrationVersion(8)
		currentVersion = 8
	}
	if currentVersion < 9 {
		result := datamigration.MigrateTopicSearchIndex()
		slog.Info("app migration topic search index done", "skipped", result.Skipped, "rebuilt", result.Rebuilt, "processed", result.ProcessedCount, "failedCount", result.FailedCount, "legacyIndexDeleteTried", result.LegacyIndexDeleteTried, "legacyIndexDeleted", result.LegacyIndexDeleted, "failed", result.Failed, "lastFailed", result.LastFailed)
		if result.Failed > 0 || result.FailedCount > 0 {
			slog.Error("app migration topic search index has failures", "failed", result.Failed, "failedCount", result.FailedCount, "lastFailed", result.LastFailed)
			return fmt.Errorf("app migration v9 topic search index: %d failed (lastFailed: %s)", result.Failed+result.FailedCount, result.LastFailed)
		}
		pageConfig.SyncMigrationVersion(9)
		currentVersion = 9
	}
	if currentVersion < 10 {
		result := datamigration.DropReportLegacyColumns()
		slog.Info("app migration report legacy columns done", "articleIDColumnDropped", result.ArticleIDColumnDropped, "statusArticleIndexDrop", result.StatusArticleIndexDrop, "articleIndexDrop", result.ArticleIndexDrop, "failed", result.Failed, "lastFailed", result.LastFailed)
		if result.Failed > 0 {
			return dataMigrationError("report legacy columns", 10, result.Failed, result.LastFailed)
		}
		pageConfig.SyncMigrationVersion(10)
		currentVersion = 10
	}
	if currentVersion < 11 {
		result := datamigration.MigratePointsRecordAction()
		slog.Info("app migration points record action done", "backfilled", result.Backfilled, "changeReasonColumnDropped", result.ChangeReasonColumnDropped, "failed", result.Failed, "lastFailed", result.LastFailed)
		if result.Failed > 0 {
			return dataMigrationError("points record action", 11, result.Failed, result.LastFailed)
		}
		pageConfig.SyncMigrationVersion(11)
		currentVersion = 11
	}
	if currentVersion < 12 {
		result := datamigration.RebuildPostMarkdown()
		slog.Info("app migration rebuild post markdown done", "processed", result.Processed, "failed", result.Failed, "lastFailed", result.LastFailed)
		if result.Failed > 0 {
			return dataMigrationError("rebuild post markdown", 12, result.Failed, result.LastFailed)
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
			return fmt.Errorf("%w: v13 aggregate search indexes (meilisearch unavailable)", ErrRetryLater)
		}
		if result.Failed > 0 {
			return dataMigrationError("aggregate search indexes", 13, result.Failed, result.LastFailed)
		}
		pageConfig.SyncMigrationVersion(13)
		currentVersion = 13
	}
	if currentVersion < 14 {
		// 积分回填（dev 合并,PR #110 防滥用）
		result := datamigration.BackfillMissingUserPoints()
		slog.Info("app migration user points backfill done", "backfilled", result.Backfilled, "failed", result.Failed, "lastFailed", result.LastFailed)
		if result.Failed > 0 {
			return dataMigrationError("user points backfill", 14, result.Failed, result.LastFailed)
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
			return dataMigrationError("delete lifecycle backfill", 15, deleteResult.Failed, deleteResult.LastFailed)
		}
		pageConfig.SyncMigrationVersion(15)
		currentVersion = 15
	}
	if currentVersion < 16 {
		// 移除 GitHub OAuth 明文 token 持久化并清理历史列（issue #131,PR #150）
		result := datamigration.DropUserOAuthTokenColumns()
		slog.Info("app migration user oauth credentials drop done", "dropped", result.Dropped, "failed", result.Failed, "lastFailed", result.LastFailed)
		if result.Failed > 0 {
			return dataMigrationError("user oauth credentials drop", 16, result.Failed, result.LastFailed)
		}
		pageConfig.SyncMigrationVersion(16)
		currentVersion = 16
	}
	if currentVersion < 17 {
		// 课评删除隔离窗口锚点回填（PR #194, issue #175 B3）：旧删除路径
		// 不写 deleted_at，存量 deleted 行若回退 updated_at 会窗口塌缩、
		// 部署后首轮清理即被清空。回填 now() 给存量 cohort 完整新窗口。
		anchorResult := datamigration.BackfillCourseReviewDeleteAnchors()
		slog.Info("app migration course review delete anchor backfill done", "backfilled", anchorResult.Backfilled, "failed", anchorResult.Failed, "lastFailed", anchorResult.LastFailed)
		if anchorResult.Failed > 0 {
			return dataMigrationError("course review delete anchor backfill", 17, anchorResult.Failed, anchorResult.LastFailed)
		}
		pageConfig.SyncMigrationVersion(17)
		currentVersion = 17
	}
	if currentVersion < 18 {
		// 帖子版本历史 v1 回填（首楼编辑 PR）：部署前已存在的帖子没有
		// post_revisions 行，首次编辑只追加新版本、不回看旧正文，原始正文
		// 会永久丢失。为存量帖子播种 v1（editor = 作者，内容取当前正文）。
		backfillResult := datamigration.BackfillPostRevisionSeeds()
		slog.Info("app migration post revision seed backfill done", "seeded", backfillResult.Seeded, "skipped", backfillResult.Skipped, "failed", backfillResult.Failed, "lastFailed", backfillResult.LastFailed)
		if backfillResult.Failed > 0 {
			return dataMigrationError("post revision seed backfill", 18, backfillResult.Failed, backfillResult.LastFailed)
		}
		pageConfig.SyncMigrationVersion(18)
		currentVersion = 18
	}
	if currentVersion < 19 {
		// 单一事件源架构（wiki 写即发布 + 版本指针 + 物化水印）：
		// 为存量 wiki 页面回填 published_revision_no（= 最新 approved revision_no），
		// 并初始化 topics/posts 的 wiki_synced_revision_no 水印（当前内容即最新版）。
		// 幂等：指针已 >0 的页面跳过。部署前已存在的 wiki 页面（发布即通过）
		// 不先回填指针，CAS 编辑将永远无法匹配 published_revision_no=0 的基线。
		singleSourceResult := datamigration.BackfillWikiSingleSource()
		slog.Info("app migration wiki single source backfill done",
			"pages", singleSourceResult.PagesSeeded,
			"topics", singleSourceResult.TopicsSeeded,
			"posts", singleSourceResult.PostsSeeded,
			"skipped", singleSourceResult.Skipped,
			"failed", singleSourceResult.Failed,
			"lastFailed", singleSourceResult.LastFailed)
		if singleSourceResult.Failed > 0 {
			return dataMigrationError("wiki single source backfill", 19, singleSourceResult.Failed, singleSourceResult.LastFailed)
		}
		pageConfig.SyncMigrationVersion(19)
		currentVersion = 19
	}
	if currentVersion < 20 {
		// 话题搜索索引文档补齐 topicType 字段（review：存量部署的索引在
		// topicType 加入前构建，聚合搜索按 topicType 过滤会失败/漏检；
		// 全量重建写入新字段，filterable 属性由启动 EnsureTopicIndexConfigured 保证）。
		// Meilisearch 不可用时跳过且不推进版本，下次启动重试（与 v13 同模式）。
		topicIndexResult := datamigration.MigrateTopicSearchIndex()
		slog.Info("app migration topic search index topicType rebuild done",
			"skipped", topicIndexResult.Skipped,
			"rebuilt", topicIndexResult.Rebuilt,
			"processed", topicIndexResult.ProcessedCount,
			"failedCount", topicIndexResult.FailedCount,
			"failed", topicIndexResult.Failed,
			"lastFailed", topicIndexResult.LastFailed)
		if topicIndexResult.Skipped {
			slog.Warn("app migration topic search index topicType rebuild skipped (meilisearch unavailable), will retry on next start")
			return fmt.Errorf("%w: v20 topic search index topicType rebuild (meilisearch unavailable)", ErrRetryLater)
		}
		if topicIndexResult.Failed > 0 || topicIndexResult.FailedCount > 0 {
			slog.Error("app migration topic search index topicType rebuild has failures", "failed", topicIndexResult.Failed, "failedCount", topicIndexResult.FailedCount, "lastFailed", topicIndexResult.LastFailed)
			return fmt.Errorf("app migration v20 topic search index topicType rebuild: %d failed (lastFailed: %s)", topicIndexResult.Failed+topicIndexResult.FailedCount, topicIndexResult.LastFailed)
		}
		pageConfig.SyncMigrationVersion(20)
		currentVersion = 20
	}
	if currentVersion < 21 {
		// GitHub SSOT 架构 v21：把存量 wiki 页面最新 approved 修订内容快照
		// 复制到 wiki_pages 投影列（title/content/rendered_html/toc/content_hash）。
		// 升级后公开读直接读投影列；content_hash 与 GitHub 仓库文件不一致时，
		// 首次同步自然触发更新（幂等衔接，无需人工干预）。
		gitSSOTResult := datamigration.BackfillWikiGitSSOT()
		slog.Info("app migration wiki git ssot backfill done",
			"pagesBackfilled", gitSSOTResult.PagesBackfilled,
			"pagesSkipped", gitSSOTResult.PagesSkipped,
			"failed", gitSSOTResult.Failed,
			"lastFailed", gitSSOTResult.LastFailed)
		if gitSSOTResult.Failed > 0 {
			return dataMigrationError("wiki git ssot backfill", 21, gitSSOTResult.Failed, gitSSOTResult.LastFailed)
		}
		pageConfig.SyncMigrationVersion(21)
		currentVersion = 21
	}
	if currentVersion < 22 {
		// 历史版本 v22（命名空间 slug 列回填）已随 slug 机制移除而删除：
		// 旧部署迁移版本号可能已停在 22，直接推进到 23 继续执行后续步骤。
		pageConfig.SyncMigrationVersion(22)
		currentVersion = 22
	}
	if currentVersion < 23 {
		// 页面仓库路径列 v23（review MEDIUM）：为存量 wiki_pages 行回填
		// source_path（= path 首段即仓库目录名的存量语义）。外链必须用
		// source_path，存量行为空会导致 SSR 编辑/历史链接畸形（管理端有回退、
		// SSR 已补回退，但回填仍是根治）。幂等：source_path 非空的行跳过。
		sourcePathResult := datamigration.BackfillWikiPageSourcePaths()
		slog.Info("app migration wiki page source_path backfill done",
			"backfilled", sourcePathResult.Backfilled,
			"failed", sourcePathResult.Failed,
			"lastFailed", sourcePathResult.LastFailed)
		if sourcePathResult.Failed > 0 {
			return dataMigrationError("wiki page source_path backfill", 23, sourcePathResult.Failed, sourcePathResult.LastFailed)
		}
		pageConfig.SyncMigrationVersion(23)
		currentVersion = 23
	}
	if currentVersion < 24 {
		// slug 机制移除 v24：URL 语义回归"仓库顶层目录名即 path 首段"。
		// 对已分配 slug 的存量命名空间，把其全部页面（含软删）的 path 首段
		// 与 namespace 列迁回仓库目录名（显示名），并清空受影响页面的
		// content_hash（下次同步重渲染投影，review P2）。AutoMigrate 不会
		// 删除从模型消失的字段——wiki_namespaces.slug 列与
		// uniq_wiki_namespace_slug 索引由迁移显式 DROP（review P2）。
		// 幂等：仅对 namespace 列 ≠ 目录名的行生效；slug 列已删的库零操作。
		slugRemoval := datamigration.RevertWikiNamespaceSlugs()
		slog.Info("app migration wiki slug removal done",
			"migrated", slugRemoval.Migrated,
			"failed", slugRemoval.Failed,
			"lastFailed", slugRemoval.LastFailed)
		if slugRemoval.Failed > 0 {
			return dataMigrationError("wiki slug removal", 24, slugRemoval.Failed, slugRemoval.LastFailed)
		}
		pageConfig.SyncMigrationVersion(24)
		currentVersion = 24
	}
	if currentVersion < 25 {
		// 管理端设置密钥明文加密 v25（issue #324 S1-S3）：把此前明文落库的
		// 邮件 SMTP 密码、对象存储 accessKey/secretKey、HTTP 通知端点 secret
		// 加密为 securestore 密文（AES-256-GCM）并清空明文。幂等：仅处理
		// 明文非空且密文为空的配置；读取侧在迁移前兼容存量明文。
		secretResult := datamigration.MigrateAdminSecretPlaintext()
		slog.Info("app migration admin secret plaintext encryption done",
			"mailEncrypted", secretResult.MailEncrypted,
			"storageKeys", secretResult.StorageKeys,
			"notifySecrets", secretResult.NotifySecrets,
			"failed", secretResult.Failed,
			"lastFailed", secretResult.LastFailed)
		if secretResult.Failed > 0 {
			return dataMigrationError("admin secret plaintext encryption", 25, secretResult.Failed, secretResult.LastFailed)
		}
		pageConfig.SyncMigrationVersion(25)
		currentVersion = 25
	}
	if currentVersion < 26 {
		// 发帖设置每日新主题上限 v26（issue #369，上游 c47cff94）：为存量
		// postingSettings 配置补齐 textControl.maxDailyTopicsPerUser（默认 10，
		// 0 = 不限额）。幂等：已存在该键的配置跳过；读取侧在迁移前兼容缺键。
		topicLimitResult := datamigration.EnsurePostingSettingsTopicLimit()
		slog.Info("app migration posting settings topic limit done",
			"updated", topicLimitResult.Updated,
			"skipped", topicLimitResult.Skipped,
			"failed", topicLimitResult.Failed,
			"lastFailed", topicLimitResult.LastFailed)
		if topicLimitResult.Failed > 0 {
			slog.Error("app migration posting settings topic limit has failures", "failed", topicLimitResult.Failed, "lastFailed", topicLimitResult.LastFailed)
			return dataMigrationError("posting settings topic limit", 26, topicLimitResult.Failed, topicLimitResult.LastFailed)
		}
		if err := pageConfig.SyncMigrationVersion(26); err != nil {
			slog.Error("app migration sync migration version failed", "version", 26, "err", err)
			return fmt.Errorf("app migration v26 sync migration version: %w", err)
		}
		currentVersion = 26
	}
	if currentVersion < 27 {
		// 安全设置默认词库 v27（Blueprint R4）：新默认 reservedUsernames /
		// sensitiveWords 词库上线。无 securitySettings 存储行时读取侧自然
		// 吃新默认，跳过不写；对已保存过且某数组为空（len==0）的存量实例
		// 各自独立并入对应新默认数组。幂等：数组非空即视为管理员已维护，
		// 绝不覆盖；bannedUsernames 永不被写入（默认 banned 为空，误并入
		// 会把存量账号静默推入冻结）。
		securityResult := datamigration.EnsureSecuritySettingsDefaults()
		slog.Info("app migration security settings defaults done",
			"updated", securityResult.Updated,
			"skipped", securityResult.Skipped,
			"failed", securityResult.Failed,
			"lastFailed", securityResult.LastFailed)
		if securityResult.Failed > 0 {
			slog.Error("app migration security settings defaults has failures", "failed", securityResult.Failed, "lastFailed", securityResult.LastFailed)
			return dataMigrationError("security settings defaults", 27, securityResult.Failed, securityResult.LastFailed)
		}
		if err := pageConfig.SyncMigrationVersion(27); err != nil {
			slog.Error("app migration sync migration version failed", "version", 27, "err", err)
			return fmt.Errorf("app migration v27 sync migration version: %w", err)
		}
		currentVersion = 27
	}
	slog.Info("app migration end", "version", currentVersion)
	return nil
}

func dataMigrationError(name string, version int, failed int, lastFailed any) error {
	slog.Error("app migration "+name+" has failures", "failed", failed, "lastFailed", lastFailed)
	return fmt.Errorf("app migration v%d %s: %d failed (lastFailed: %v)", version, name, failed, lastFailed)
}
