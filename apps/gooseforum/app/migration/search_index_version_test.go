package migration

import (
	"errors"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/meiliconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/badges"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userBadges"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	xpmodel "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/xp"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/datamigration"
	"gorm.io/gorm"
)

// issue #1284 A2：v13/v20 依赖 Meilisearch（可选依赖），而它们之后的本地版本块
// （待审帖接管、机器人徽章回填、被拒内容申诉基线、空摘要回填、XP 初始规则）
// 只改数据库。修复前索引步骤在无 Meilisearch 的实例上直接返回 ErrRetryLater
// 且不推进主水位，导致这些块永不执行；新审核动作
// 按 post.LatestRevisionId == revision.Id 判定，缺回填会让接管逻辑静默失效。
//
// restoreMainMigrationWatermark 在测试改写主水位之前登记还原：测试共用同一内存库，
// 把主水位推进到 AppMigrationVersion 后不还原，后续用例会读到高水位而跳过
// 迁移块（issue #1284 review）。
func restoreMainMigrationWatermark(t *testing.T, conn *gorm.DB) {
	t.Helper()
	var previous pageConfig.Entity
	hadPrevious := conn.Where("page_type = ?", pageConfig.Migration).Take(&previous).Error == nil
	t.Cleanup(func() {
		if hadPrevious {
			if err := conn.Save(&previous).Error; err != nil {
				t.Errorf("restore main migration watermark: %v", err)
			}
			return
		}
		if err := conn.Where("page_type = ?", pageConfig.Migration).Delete(&pageConfig.Entity{}).Error; err != nil {
			t.Errorf("delete main migration watermark: %v", err)
		}
	})
}

// 测试环境无 Meilisearch（client == nil ⇒ IsAvailable() == false），因此天然覆盖
// 「索引步骤可跳过」的路径。主水位从 v30 起跑，避免重跑 v1-v29 的存量数据重建。
func TestVersionedMigrationAdvancesLocalVersionsWhenSearchIndexUnavailable(t *testing.T) {
	if meiliconnect.IsAvailable() {
		t.Skip("asserts the absent-meilisearch path; an ambient meilisearch would run real index builds")
	}
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(&pageConfig.Entity{}, &posts.Entity{}, &users.EntityComplete{}, &badges.Entity{}, &userBadges.Entity{}, &topics.Entity{},
		&xpmodel.RuleSet{}, &xpmodel.RulePublicationLock{}, &xpmodel.RewardRule{}); err != nil {
		t.Fatal(err)
	}
	restoreMainMigrationWatermark(t, conn)
	if err := pageConfig.SyncMigrationVersion(30); err != nil {
		t.Fatal(err)
	}
	// 索引水位落后：显式落 0，模拟「索引步骤一直没能跑完」的实例。缺键时
	// GetSearchIndexMigrationVersion 会按主水位推断为已完成，故这里必须显式写。
	if err := pageConfig.SyncSearchIndexMigrationVersion(0); err != nil {
		t.Fatal(err)
	}
	// 测试共用同一内存库：清掉本用例写入的 0，否则后续用例按主水位推断
	// 已完成时仍读到 0，会被误判为索引步骤待重试。
	t.Cleanup(func() {
		if err := conn.Where("page_type = ?", pageConfig.MigrationSearchIndex).Delete(&pageConfig.Entity{}).Error; err != nil {
			t.Errorf("cleanup search index watermark: %v", err)
		}
	})

	err := runVersionedDataMigrations()
	if !errors.Is(err, ErrRetryLater) {
		t.Fatalf("err = %v; search index steps must stay deferred (non-fatal) while local versions advance", err)
	}
	if got := pageConfig.GetMigrationVersion(); got != pageConfig.AppMigrationVersion {
		t.Fatalf("main migration version = %d; local versions must not be blocked by an unavailable meilisearch", got)
	}
	if got := pageConfig.GetSearchIndexMigrationVersion(pageConfig.AppMigrationVersion); got != 0 {
		t.Fatalf("search index version = %d; index steps must not advance while meilisearch is unavailable", got)
	}
}

// 索引水位键是本次修复才引入的。存量实例（已跑完整迁移）不能因此重跑一次全量
// 索引重建；卡在 v12 的实例必须从 0 重试。推断只做一次并落库，避免修复后主水位
// 推进到 33 时把待重试的索引步骤误标为已完成。
func TestSearchIndexMigrationVersionSeedsOnceFromMainVersion(t *testing.T) {
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(&pageConfig.Entity{}); err != nil {
		t.Fatal(err)
	}
	// 测试共用同一内存库：先清掉索引水位键，模拟「本次升级前该键不存在」。
	if err := conn.Where("page_type = ?", pageConfig.MigrationSearchIndex).Delete(&pageConfig.Entity{}).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := conn.Where("page_type = ?", pageConfig.MigrationSearchIndex).Delete(&pageConfig.Entity{}).Error; err != nil {
			t.Errorf("cleanup search index watermark: %v", err)
		}
	})

	if got := pageConfig.GetSearchIndexMigrationVersion(33); got != pageConfig.SearchIndexMigrationVersion {
		t.Fatalf("existing instance (main=33) seeded %d; must be treated as already indexed", got)
	}
	// 落库后再按较低主水位读取不得回退（推断已固化）。
	if got := pageConfig.GetSearchIndexMigrationVersion(12); got != pageConfig.SearchIndexMigrationVersion {
		t.Fatalf("persisted search index version = %d; seeding must happen only once", got)
	}

	if err := pageConfig.SyncSearchIndexMigrationVersion(0); err != nil {
		t.Fatal(err)
	}
	if got := pageConfig.GetSearchIndexMigrationVersion(33); got != 0 {
		t.Fatalf("search index version = %d; a persisted pending watermark must not be re-seeded as done", got)
	}
}

// 索引步骤成功时只能推进索引水位：主水位已到 33 的实例重试索引步骤，不得被
// 改写成 13/20（会让 v14-v33 的本地步骤重跑）。CI/本地无 Meilisearch，只有
// 替换可注入入口才走得到「索引步骤成功」分支（issue #1284 A2）。
func TestSearchIndexStepsDoNotRewindMainWatermark(t *testing.T) {
	if meiliconnect.IsAvailable() {
		t.Skip("stubs the index migration entry points; an ambient meilisearch would run real index builds")
	}
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(&pageConfig.Entity{}, &posts.Entity{}, &users.EntityComplete{}, &badges.Entity{}, &userBadges.Entity{}, &topics.Entity{}); err != nil {
		t.Fatal(err)
	}
	// 主水位已跑完、索引水位仍是 0：模拟「本次升级前索引步骤一直没跑成」的实例。
	restoreMainMigrationWatermark(t, conn)
	if err := pageConfig.SyncMigrationVersion(pageConfig.AppMigrationVersion); err != nil {
		t.Fatal(err)
	}
	if err := pageConfig.SyncSearchIndexMigrationVersion(0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := conn.Where("page_type = ?", pageConfig.MigrationSearchIndex).Delete(&pageConfig.Entity{}).Error; err != nil {
			t.Errorf("cleanup search index watermark: %v", err)
		}
	})
	previousAggregate, previousTopic := migrateAggregateSearchIndexesFn, migrateTopicSearchIndexFn
	t.Cleanup(func() {
		migrateAggregateSearchIndexesFn, migrateTopicSearchIndexFn = previousAggregate, previousTopic
	})
	// 未跳过、零失败的成功结果：让两个版本块都走进「推进索引水位」分支。
	migrateAggregateSearchIndexesFn = func() datamigration.AggregateSearchIndexMigrationResult {
		return datamigration.AggregateSearchIndexMigrationResult{UsersRebuilt: true, CategoriesRebuilt: true}
	}
	migrateTopicSearchIndexFn = func() datamigration.TopicSearchIndexMigrationResult {
		return datamigration.TopicSearchIndexMigrationResult{Rebuilt: true}
	}

	err := runVersionedDataMigrations()
	// 先断言水位（本用例的不变量），再断言返回：守卫缺失时主水位被改写成 13/20，
	// 后续本地代码块在测试库里缺表而报错只是连带后果（issue #1284）。
	if got := pageConfig.GetMigrationVersion(); got != pageConfig.AppMigrationVersion {
		t.Fatalf("main version = %d; index steps must not rewind the main watermark", got)
	}
	if got := pageConfig.GetSearchIndexMigrationVersion(pageConfig.AppMigrationVersion); got != pageConfig.SearchIndexMigrationVersion {
		t.Fatalf("search index version = %d; index steps must advance to %d", got, pageConfig.SearchIndexMigrationVersion)
	}
	if err != nil {
		t.Fatalf("migrate = %v; both watermarks reach their targets so the chain must return nil", err)
	}
}
