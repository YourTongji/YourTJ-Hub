package datamigration

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiPages"
	"gorm.io/gorm"
)

// WikiSlugRevertResult v24 slug 机制移除的数据迁移结果。
type WikiSlugRevertResult struct {
	Migrated          int
	Failed            int
	LastFailed        string
	SlugIndexDropped  bool
	SlugColumnDropped bool
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
// 刷新，即新语义下 path 的目标值）。本迁移不依赖 slug 列。
//
// 唯一键安全（review P1）：两个命名空间可能互为 slug（目录 a 的 slug=b、
// 目录 b 的 slug=a），逐行更新会因目标 path 仍被对方占用而违反
// uniq_wiki_page_path。因此两阶段更新：先全部迁到临时唯一路径释放旧路径，
// 再写入最终路径，全程单事务。
//
// 重渲染衔接（review P2）：迁移只改 path/namespace，不重写
// rendered_html/first-post HTML（仓库 clone 与引用解析不属于迁移职责）。
// 受影响页面清空 content_hash，下次同步必然走更新路径重渲染投影
// （同步器幂等判断以 content_hash 为准），期间旧渲染保留（可展示，旧 slug
// 链接在同步前 404——这正是 slug 移除的预期语义）。
//
// 幂等：path 已等于 source_path 的行跳过；source_path 为空（从未同步）且
// path 首段 = 显示名（无 slug 痕迹）的行同样跳过；无法推导的行留给下次
// 同步自动收养修复（sync 按 source_path 找回复用路径）。
func RevertWikiNamespaceSlugs() WikiSlugRevertResult {
	return RevertWikiNamespaceSlugsWithDB(dbconnect.Connect())
}

func RevertWikiNamespaceSlugsWithDB(conn *gorm.DB) WikiSlugRevertResult {
	result := WikiSlugRevertResult{}
	if conn.Migrator().HasTable(&wikiPages.Entity{}) {
		var pages []wikiPages.Entity
		if err := conn.Unscoped().Model(&wikiPages.Entity{}).Find(&pages).Error; err != nil {
			result.Failed++
			result.LastFailed = "scan_pages:" + err.Error()
			slog.Error("wiki slug revert: scan pages failed", "err", err)
			return result
		}

		type pendingPage struct {
			id      uint64
			newPath string
			dirName string
		}
		pending := make([]pendingPage, 0, len(pages))
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
			pending = append(pending, pendingPage{id: p.Id, newPath: newPath, dirName: dirName})
		}
		if len(pending) > 0 {
			// 两阶段 + 单事务（review P1）：临时路径用不可能与仓库真实路径
			// 冲突的前缀（wiki 页面路径必须合法，仓库目录名不能以该前缀
			// 命名——_assets 保留前缀同理，双下划线开头段被 ValidatePath 拒绝）。
			if err := conn.Transaction(func(tx *gorm.DB) error {
				for _, pp := range pending {
					tmpPath := fmt.Sprintf("__wiki_slug_revert__/%d", pp.id)
					if err := tx.Unscoped().Model(&wikiPages.Entity{}).Where("id = ?", pp.id).
						Update("path", tmpPath).Error; err != nil {
						return fmt.Errorf("stage1 temp path id %d: %w", pp.id, err)
					}
				}
				for _, pp := range pending {
					if err := tx.Unscoped().Model(&wikiPages.Entity{}).Where("id = ?", pp.id).
						Updates(map[string]any{
							"path":         pp.newPath,
							"namespace":    pp.dirName,
							"content_hash": "", // 强制下次同步重渲染投影（review P2）
							"updated_at":   gorm.Expr("updated_at"),
						}).Error; err != nil {
						return fmt.Errorf("stage2 final path id %d: %w", pp.id, err)
					}
				}
				return nil
			}); err != nil {
				result.Failed++
				result.LastFailed = "update_page:" + err.Error()
				slog.Error("wiki slug revert: update page failed", "err", err)
				return result
			}
			result.Migrated = len(pending)
		}
	}

	// AutoMigrate 不删除从模型消失的字段：显式清理遗留 slug 列与唯一索引
	// （review P2）。仅当数据迁移成功（或无需迁移）时执行。
	dropLegacySlugSchema(conn, &result)
	return result
}

// dropLegacySlugSchema 删除 wiki_namespaces.slug 列与 uniq_wiki_namespace_slug
// 索引。GORM AutoMigrate 不会删除从模型消失的字段，存量库升级后列与索引会
// 残留并继续携带陈旧数据（review P2）。先删索引再删列。
func dropLegacySlugSchema(conn *gorm.DB, result *WikiSlugRevertResult) {
	if !conn.Migrator().HasTable("wiki_namespaces") {
		return
	}
	if conn.Migrator().HasIndex("wiki_namespaces", "uniq_wiki_namespace_slug") {
		// SQLite 与 PostgreSQL 均支持 DROP INDEX IF EXISTS（MySQL 语法不同，已不再支持 MySQL）。
		if err := conn.Exec("DROP INDEX IF EXISTS uniq_wiki_namespace_slug").Error; err != nil {
			result.Failed++
			result.LastFailed = "drop_slug_index:" + err.Error()
			slog.Error("wiki slug revert: drop slug index failed", "err", err)
			return
		}
		result.SlugIndexDropped = true
	}
	if conn.Migrator().HasColumn("wiki_namespaces", "slug") {
		if err := conn.Exec("ALTER TABLE wiki_namespaces DROP COLUMN slug").Error; err != nil {
			result.Failed++
			result.LastFailed = "drop_slug_column:" + err.Error()
			slog.Error("wiki slug revert: drop slug column failed", "err", err)
			return
		}
		result.SlugColumnDropped = true
	}
}
