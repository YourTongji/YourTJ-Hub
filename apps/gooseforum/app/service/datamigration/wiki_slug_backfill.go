package datamigration

import (
	"log/slog"
	"regexp"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiNamespaces"
	"gorm.io/gorm"
)

// WikiSlugBackfillResult v22 回填结果。
type WikiSlugBackfillResult struct {
	Backfilled int
	Skipped    int
	Failed     int
	LastFailed string
}

// slugBackfillRe 与 wikiservice.ValidateSlug 的 slug 规则一致
// （^[a-z0-9]+(-[a-z0-9]+)*$ ≤64）。回填仅对纯 ASCII 小写 slug 形态的 name 生效。
var slugBackfillRe = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// BackfillWikiNamespaceSlugs v22 回填：为既有 wiki_namespaces 行分配 slug。
//
// 背景：命名空间新增 slug 列（URL 友好标识，与显示名 name 分离）。存量行中，
// name 为纯 ASCII slug 形态（如 "guide" / "deployment"）的命名空间直接用 name
// 回填 slug；中文等非 ASCII 名称无法推导 slug，保持 NULL（由后续 GitHub 同步
// 在 index.md frontmatter 提供 slug 时填充）。空 slug 不参与唯一索引，多个
// 未分配 slug 的行不冲突。
//
// 幂等：slug 已非空的行走过；重复运行零变更。
func BackfillWikiNamespaceSlugs() WikiSlugBackfillResult {
	return BackfillWikiNamespaceSlugsWithDB(dbconnect.Connect())
}

func BackfillWikiNamespaceSlugsWithDB(conn *gorm.DB) WikiSlugBackfillResult {
	result := WikiSlugBackfillResult{}
	if !conn.Migrator().HasTable(&wikiNamespaces.Entity{}) {
		return result
	}

	var entities []wikiNamespaces.Entity
	if err := conn.Model(&wikiNamespaces.Entity{}).Find(&entities).Error; err != nil {
		result.Failed++
		result.LastFailed = "scan:" + err.Error()
		slog.Error("wiki slug backfill: scan namespaces failed", "err", err)
		return result
	}

	for i := range entities {
		ns := &entities[i]
		if ns.Slug != nil && *ns.Slug != "" {
			result.Skipped++
			continue
		}
		// 仅纯 ASCII slug 形态的 name 可回填；中文等非 ASCII 保持 NULL。
		if !slugBackfillRe.MatchString(ns.Name) {
			result.Skipped++
			continue
		}
		// 并发/重复运行保护：目标 slug 已被其他行占用时不回填（保持 NULL）。
		var conflict int64
		if err := conn.Model(&wikiNamespaces.Entity{}).
			Where("slug = ?", ns.Name).
			Where("id <> ?", ns.Id).
			Count(&conflict).Error; err != nil {
			result.Failed++
			result.LastFailed = "conflict_check:" + err.Error()
			slog.Error("wiki slug backfill: conflict check failed", "id", ns.Id, "err", err)
			return result
		}
		if conflict > 0 {
			result.Skipped++
			continue
		}
		slug := ns.Name
		if err := conn.Model(&wikiNamespaces.Entity{}).
			Where("id = ?", ns.Id).
			Update("slug", slug).Error; err != nil {
			result.Failed++
			result.LastFailed = "update:" + err.Error()
			slog.Error("wiki slug backfill: update failed", "id", ns.Id, "err", err)
			return result
		}
		result.Backfilled++
	}
	return result
}
