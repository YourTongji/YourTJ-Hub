package datamigration

import (
	"log/slog"
	"strconv"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiPages"
	"gorm.io/gorm"
)

// WikiSourcePathBackfillResult v23 回填结果。
type WikiSourcePathBackfillResult struct {
	Backfilled int
	Failed     int
	LastFailed string
}

// BackfillWikiPageSourcePaths v23 回填：为存量 wiki_pages 行填充 source_path。
//
// 背景：source_path 列（仓库真实相对路径，GitHub 编辑/历史外链用）随 PR 新增，
// 存量行该列为空。D7 语义下 path 首段 = URL key（slug），与仓库目录名解耦，
// 外链必须用 source_path。存量行创建于 slug 语义之前，其 path 首段即仓库
// 目录名（显示名），因此 source_path = path 即为正确的仓库路径；随后续同步
// 按仓库文件更新为权威值。幂等：source_path 已非空的行跳过。
func BackfillWikiPageSourcePaths() WikiSourcePathBackfillResult {
	return BackfillWikiPageSourcePathsWithDB(dbconnect.Connect())
}

func BackfillWikiPageSourcePathsWithDB(conn *gorm.DB) WikiSourcePathBackfillResult {
	result := WikiSourcePathBackfillResult{}
	if !conn.Migrator().HasTable(&wikiPages.Entity{}) {
		return result
	}

	var entities []wikiPages.Entity
	if err := conn.Unscoped().Model(&wikiPages.Entity{}).Find(&entities).Error; err != nil {
		result.Failed++
		result.LastFailed = "scan:" + err.Error()
		slog.Error("wiki source_path backfill: scan pages failed", "err", err)
		return result
	}

	for i := range entities {
		p := &entities[i]
		if p.SourcePath != "" {
			continue
		}
		if p.Path == "" {
			result.Failed++ // 空 path 的行无仓库路径可推导，计入失败以便排查
			result.LastFailed = "empty path for page " + strconv.FormatUint(p.Id, 10)
			continue
		}
		if err := conn.Unscoped().Model(&wikiPages.Entity{}).
			Where("id = ?", p.Id).
			Update("source_path", p.Path).Error; err != nil {
			result.Failed++
			result.LastFailed = "update:" + err.Error()
			slog.Error("wiki source_path backfill: update failed", "id", p.Id, "err", err)
			return result
		}
		result.Backfilled++
	}
	return result
}
