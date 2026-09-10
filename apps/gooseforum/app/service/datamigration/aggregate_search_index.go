package datamigration

import (
	"log/slog"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/meiliconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/searchservice"
)

// AggregateSearchIndexMigrationResult 汇总 v13 迁移结果（users + categories 索引构建）。
type AggregateSearchIndexMigrationResult struct {
	Skipped             bool   `json:"skipped"`
	UsersRebuilt        bool   `json:"usersRebuilt"`
	UsersProcessed      int    `json:"usersProcessed"`
	UsersFailed         int    `json:"usersFailed"`
	CategoriesRebuilt   bool   `json:"categoriesRebuilt"`
	CategoriesProcessed int    `json:"categoriesProcessed"`
	CategoriesFailed    int    `json:"categoriesFailed"`
	Failed              int    `json:"failed"`
	LastFailed          string `json:"lastFailed"`
}

// MigrateAggregateSearchIndexes 构建 users + categories 搜索索引。
// Meilisearch 不可用 → Skipped（app_migration 不推进版本，下次启动重试）；
// 构建失败（FailedCount>0）→ Failed++（同样不推进版本，下次启动重试）。
func MigrateAggregateSearchIndexes() AggregateSearchIndexMigrationResult {
	return migrateAggregateSearchIndexes(
		meiliconnect.IsAvailable,
		searchservice.BuildUserIndex,
		searchservice.BuildCategoryIndex,
	)
}

func migrateAggregateSearchIndexes(
	isAvailable func() bool,
	buildUserIndex func() (*searchservice.IndexBuildResult, error),
	buildCategoryIndex func() (*searchservice.IndexBuildResult, error),
) AggregateSearchIndexMigrationResult {
	result := AggregateSearchIndexMigrationResult{}
	if !isAvailable() {
		result.Skipped = true
		return result
	}

	userResult, err := buildUserIndex()
	if err != nil {
		result.Failed++
		result.LastFailed = err.Error()
		return result
	}
	result.UsersRebuilt = true
	result.UsersProcessed = userResult.ProcessedCount
	result.UsersFailed = userResult.FailedCount
	if userResult.FailedCount > 0 {
		result.Failed++
		result.LastFailed = "user index build had failures"
		return result
	}

	categoryResult, err := buildCategoryIndex()
	if err != nil {
		result.Failed++
		result.LastFailed = err.Error()
		return result
	}
	result.CategoriesRebuilt = true
	result.CategoriesProcessed = categoryResult.ProcessedCount
	result.CategoriesFailed = categoryResult.FailedCount
	if categoryResult.FailedCount > 0 {
		result.Failed++
		result.LastFailed = "category index build had failures"
		return result
	}
	return result
}

// LogAggregateSearchIndexMigration 输出 v13 迁移结果（供 app_migration 调用）。
func LogAggregateSearchIndexMigration(result AggregateSearchIndexMigrationResult) {
	slog.Info("app migration aggregate search indexes done",
		"skipped", result.Skipped,
		"usersRebuilt", result.UsersRebuilt,
		"usersProcessed", result.UsersProcessed,
		"usersFailed", result.UsersFailed,
		"categoriesRebuilt", result.CategoriesRebuilt,
		"categoriesProcessed", result.CategoriesProcessed,
		"categoriesFailed", result.CategoriesFailed,
		"failed", result.Failed,
		"lastFailed", result.LastFailed,
	)
}
