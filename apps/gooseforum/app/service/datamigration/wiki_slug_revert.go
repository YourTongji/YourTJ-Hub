package datamigration

import (
	"log/slog"
	"strings"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiPages"
	"gorm.io/gorm"
)

// WikiSlugRevertResult v24 slug 机制移除的数据迁移结果。
type WikiSlugRevertResult struct {
	Migrated   int
	Failed     int
	LastFailed string
}

// RevertWikiNamespaceSlugs v24 数据迁移：把已分配 slug 的命名空间页面
// path 首段与 namespace 列迁回仓库顶层目录名（显示名）。
//
// 背景：slug 机制整体移除，URL 语义回归"仓库顶层目录名即 path 首段"。
// 存量部署中 slug 已生效（wiki_pages.path 首段 / namespace 列 = slug，
// 与仓库目录名分离）的行需要迁回，否则新语义下直查 path 会 404
// （ResolvePageByURLPath 不再做 name→slug 重建）。
//
// 权威依据：wiki_pages.source_path（v23 起由同步器按仓库真实相对路径回填/
// 刷新，即新语义下 path 的目标值）。slug 列已在 AutoMigrate 阶段随新模型
// 删除，本迁移不依赖该列。
//
// 幂等：path 已等于 source_path 的行跳过；source_path 为空（从未同步）且
// path 首段 = 显示名（无 slug 痕迹）的行同样跳过；无法推导的行留给下次
// 同步自动收养修复（sync 按 source_path 找回复用路径）。
func RevertWikiNamespaceSlugs() WikiSlugRevertResult {
	return RevertWikiNamespaceSlugsWithDB(dbconnect.Connect())
}

func RevertWikiNamespaceSlugsWithDB(conn *gorm.DB) WikiSlugRevertResult {
	result := WikiSlugRevertResult{}
	if !conn.Migrator().HasTable(&wikiPages.Entity{}) {
		return result
	}

	var pages []wikiPages.Entity
	if err := conn.Unscoped().Model(&wikiPages.Entity{}).Find(&pages).Error; err != nil {
		result.Failed++
		result.LastFailed = "scan_pages:" + err.Error()
		slog.Error("wiki slug revert: scan pages failed", "err", err)
		return result
	}

	for i := range pages {
		p := &pages[i]
		if p.Path == "" || p.SourcePath == "" {
			continue // 无仓库路径可推导（从未同步），留给同步器收养修复
		}
		newPath := p.SourcePath
		dirName, _, _ := strings.Cut(newPath, "/")
		if p.Path == newPath && p.Namespace == dirName {
			continue
		}
		if err := conn.Unscoped().Model(&wikiPages.Entity{}).Where("id = ?", p.Id).Updates(map[string]any{
			"path":       newPath,
			"namespace":  dirName,
			"updated_at": gorm.Expr("updated_at"),
		}).Error; err != nil {
			result.Failed++
			result.LastFailed = "update_page:" + err.Error()
			slog.Error("wiki slug revert: update page failed", "id", p.Id, "err", err)
			return result
		}
		result.Migrated++
	}
	return result
}
